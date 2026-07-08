package quality

// 本文件是 RetrievalEvaluator 的端到端集成测试：用真 Ollama embedding 跑
// hybrid/vector/keyword 三模式检索，验证评估器能产出 recall@k/MRR/
// neighbor_hit_rate/mode_compare 指标。
//
// 运行前提：
//   - PostgreSQL（codeatlas@localhost:5432），goose 迁移自动建库 + schema
//   - Ollama 在 localhost:11434，模型 qwen3-embedding:0.6b（1024 维）已拉取
//
// 任一前提不满足（-short / Ollama 不可达）则跳过，不 fail。
//
// DB setup 复刻 internal/retrieval/hybrid_retriever_integration_test.go 的
// 逻辑（连 postgres 库 → CREATE DATABASE → 跑 goose 迁移 → 返回 *models.DB），
// 因 retrieval 包的 helper 未导出，本包内自建一份等价实现。

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/config"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// --- DB setup helpers（复刻 retrieval 包集成测试逻辑）---

// integrationTestDB 包裹一个 *models.DB 及其库名，供测试结束后回收。
type integrationTestDB struct {
	*models.DB
	dbName string
}

// setupIntegrationTestDB 创建唯一测试库并跑 goose 迁移，返回 *models.DB。
// 失败时直接 t.Fatalf（与集成测试惯例一致）。
func setupIntegrationTestDB(t *testing.T) *integrationTestDB {
	t.Helper()

	models.SetDBLogger(nil)

	dbName := fmt.Sprintf("codeatlas_test_%s", uuid.New().String()[:8])

	cfg := &config.DatabaseConfig{
		Host:            testEnv("DB_HOST", "localhost"),
		Port:            testEnvInt("DB_PORT", 5432),
		User:            testEnv("DB_USER", "codeatlas"),
		Password:        testEnv("DB_PASSWORD", "codeatlas"),
		Database:        "postgres",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
	adminDB, err := models.NewDBWithConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to postgres database: %v", err)
	}
	defer adminDB.Close()

	ctx := context.Background()
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cfg.Database = dbName
	testDB, err := models.NewDBWithConfig(cfg)
	if err != nil {
		adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	sm := models.NewSchemaManager(testDB)
	if err := sm.InitializeSchema(ctx); err != nil {
		testDB.Close()
		adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return &integrationTestDB{DB: testDB, dbName: dbName}
}

// teardown 回收测试库：先关连接，再回 postgres 库 DROP。
func (tdb *integrationTestDB) teardown(t *testing.T) {
	t.Helper()
	dbName := tdb.dbName
	tdb.Close()

	cfg := &config.DatabaseConfig{
		Host:     testEnv("DB_HOST", "localhost"),
		Port:     testEnvInt("DB_PORT", 5432),
		User:     testEnv("DB_USER", "codeatlas"),
		Password: testEnv("DB_PASSWORD", "codeatlas"),
		Database: "postgres",
		SSLMode:  "disable",
	}
	adminDB, err := models.NewDBWithConfig(cfg)
	if err != nil {
		t.Logf("Warning: failed to connect for cleanup: %v", err)
		return
	}
	defer adminDB.Close()

	if _, err := adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)); err != nil {
		t.Logf("Warning: failed to drop test database: %v", err)
	}
}

func testEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func testEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	return def
}

// isOllamaAvailable 探测 Ollama 是否可达（轻量 /api/tags 接口，2s 超时）。
func isOllamaAvailable(t *testing.T) bool {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(testEnv("OLLAMA_HOST_URL", "http://localhost:11434") + "/api/tags")
	if err != nil {
		t.Logf("Ollama probe failed: %v", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// --- 集成测试主体 ---

// TestRetrievalEvaluator_Integration 用真 Ollama embedding 跑 hybrid/vector/keyword 三模式。
//
// fixture：1 repo + 1 file + 3 个 auth 相关符号：
//   - LoginUser（语义近 "user login"，主命中）调用 VerifyPassword 与 LogAccess
//   - VerifyPassword（语义近 "password"，主命中）
//   - LogAccess（语义远 "access log"，不在 Top-K 主命中）
//
// 三符号各写一条真 embedding 向量。真值 query "how does user login work" 的相关
// 符号为 {LoginUser, VerifyPassword, LogAccess}：
//   - LoginUser/VerifyPassword 经向量召回为主命中（语义近）
//   - LogAccess 语义远、不进 Top-K 主命中，但作为 LoginUser 的 callee 出现在
//     邻居扩展里 → neighbor_hit_rate > 0，验证了图谱扩展的价值。
//
// 断言：
//   - 三种 mode 均产出 recall@10 指标
//   - 产出 mode_compare 指标
//   - hybrid recall > 0（query 与 docstring 语义相近，真 embedding 应能命中）
//   - neighbor_hit_rate > 0（LogAccess 只能通过邻居发现，验证图谱扩展）
func TestRetrievalEvaluator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Ollama 不可达则跳过（不 fail）。
	if !isOllamaAvailable(t) {
		t.Skip("Ollama 不可达，跳过 retrieval 评估集成测试")
	}

	tdb := setupIntegrationTestDB(t)
	defer tdb.teardown(t)

	ctx := context.Background()

	// 构造 embedder（连 Ollama 的 OpenAI 兼容端点）。
	embedderCfg := &indexer.EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          testEnv("OLLAMA_HOST_URL", "http://localhost:11434") + "/v1/embeddings",
		Model:                testEnv("OLLAMA_EMBED_MODEL", "qwen3-embedding:0.6b"),
		Dimensions:           testEnvInt("EMBEDDING_DIMENSIONS", 1024),
		BatchSize:            10,
		MaxRequestsPerSecond: 10,
		MaxRetries:           2,
		BaseRetryDelay:       100 * time.Millisecond,
		MaxRetryDelay:        5 * time.Second,
		Timeout:              30 * time.Second,
	}

	vectorRepo := models.NewVectorRepository(tdb.DB)
	edgeRepo := models.NewEdgeRepository(tdb.DB)
	symbolRepo := models.NewSymbolRepository(tdb.DB)
	fileRepo := models.NewFileRepository(tdb.DB)
	repoRepo := models.NewRepositoryRepository(tdb.DB)

	embedder := indexer.NewOpenAIEmbedder(embedderCfg, vectorRepo)

	// 1. 构造测试数据：repo + file + 两个符号 + call 边
	repo := &models.Repository{RepoID: uuid.New().String(), Name: "retrieval-eval-test"}
	require.NoError(t, repoRepo.Create(ctx, repo))

	file := &models.File{
		FileID:   uuid.New().String(),
		RepoID:   repo.RepoID,
		Path:     "auth.go",
		Language: "go",
		Size:     100,
		Checksum: "abc",
	}
	require.NoError(t, fileRepo.Create(ctx, file))

	loginFunc := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "LoginUser",
		Kind:      "function",
		Signature: "func LoginUser(username, password string) error",
		StartLine: 1,
		EndLine:   10,
		Docstring: "LoginUser authenticates a user with username and password",
	}
	require.NoError(t, symbolRepo.Create(ctx, loginFunc))

	verifyFunc := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "VerifyPassword",
		Kind:      "function",
		Signature: "func VerifyPassword(hash, input string) bool",
		StartLine: 12,
		EndLine:   20,
		Docstring: "VerifyPassword checks password hash",
	}
	require.NoError(t, symbolRepo.Create(ctx, verifyFunc))

	// LogAccess：访问日志记录。与 login/password 语义远，向量召回里不会进 Top-K
	// 主命中；但作为 LoginUser 的 callee，会通过图谱邻居扩展被发现，使 neighbor_hit_rate > 0。
	logAccessFunc := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "LogAccess",
		Kind:      "function",
		Signature: "func LogAccess(user, action string)",
		StartLine: 22,
		EndLine:   30,
		Docstring: "LogAccess records an access log entry for auditing",
	}
	require.NoError(t, symbolRepo.Create(ctx, logAccessFunc))

	// call 边: LoginUser -> VerifyPassword
	verifyTargetID := verifyFunc.SymbolID
	edge := &models.Edge{
		EdgeID:     uuid.New().String(),
		SourceID:   loginFunc.SymbolID,
		TargetID:   &verifyTargetID,
		EdgeType:   "call",
		SourceFile: file.Path,
		TargetFile: &file.Path,
	}
	require.NoError(t, edgeRepo.Create(ctx, edge))

	// call 边: LoginUser -> LogAccess（LoginUser 登录成功后记访问日志）
	logAccessTargetID := logAccessFunc.SymbolID
	logEdge := &models.Edge{
		EdgeID:     uuid.New().String(),
		SourceID:   loginFunc.SymbolID,
		TargetID:   &logAccessTargetID,
		EdgeType:   "call",
		SourceFile: file.Path,
		TargetFile: &file.Path,
	}
	require.NoError(t, edgeRepo.Create(ctx, logEdge))

	// 2. 用真 embedder 生成向量并写入。
	const loginContent = "LoginUser authenticates a user with username and password"
	const verifyContent = "VerifyPassword checks password hash"
	// LogAccess 内容刻意与 login/password 语义拉开距离（讲访问审计日志），
	// 使其向量召回分数低于 LoginUser/VerifyPassword，不进 Top-K 主命中。
	const logAccessContent = "LogAccess records an access log entry for auditing purposes"
	vectors, err := embedder.BatchEmbed(ctx, []string{loginContent, verifyContent, logAccessContent})
	require.NoError(t, err)
	require.Len(t, vectors, 3, "BatchEmbed 应返回与输入等长的向量")
	require.Len(t, vectors[0], 1024, "向量维度应为 1024")

	vec1 := &models.Vector{
		VectorID:   uuid.New().String(),
		EntityID:   loginFunc.SymbolID,
		EntityType: "symbol",
		Embedding:  vectors[0],
		Content:    loginContent,
		Model:      embedderCfg.Model,
	}
	require.NoError(t, vectorRepo.Create(ctx, vec1))

	vec2 := &models.Vector{
		VectorID:   uuid.New().String(),
		EntityID:   verifyFunc.SymbolID,
		EntityType: "symbol",
		Embedding:  vectors[1],
		Content:    verifyContent,
		Model:      embedderCfg.Model,
	}
	require.NoError(t, vectorRepo.Create(ctx, vec2))

	vec3 := &models.Vector{
		VectorID:   uuid.New().String(),
		EntityID:   logAccessFunc.SymbolID,
		EntityType: "symbol",
		Embedding:  vectors[2],
		Content:    logAccessContent,
		Model:      embedderCfg.Model,
	}
	require.NoError(t, vectorRepo.Create(ctx, vec3))

	// 3. 构造 retriever（带真 embedder，支持 hybrid/vector 模式生成 query 向量）。
	retriever := retrieval.NewHybridRetriever(
		vectorRepo, edgeRepo, embedder,
		retrieval.DefaultHybridRetrieverConfig(),
	)

	// 4. 构造 retrieval evaluator。
	// 真值相关 = {LoginUser, VerifyPassword, LogAccess}：
	//   - LoginUser/VerifyPassword 经语义召回为主命中
	//   - LogAccess 不进 Top-K 主命中，但作为 LoginUser callee 进邻居 → neighbor_hit_rate > 0
	truths := []RetrievalGroundTruth{
		{
			Query:           "how does user login work",
			RelevantSymbols: []string{"LoginUser", "VerifyPassword", "LogAccess"},
		},
	}
	eval := NewRetrievalEvaluator(retriever, truths, []string{"hybrid", "vector", "keyword"})

	// 5. 跑评估。
	metrics, err := eval.Evaluate(ctx, []string{repo.RepoID})
	require.NoError(t, err)

	// 6. 验证产出了指标。
	found := map[string]float64{}
	for _, m := range metrics {
		found[m.Name] = m.Value
		t.Logf("  %s = %.4f (threshold=%.2f, passed=%v)", m.Name, m.Value, m.Threshold, m.Passed)
	}

	// 三种 mode 的 recall@10 均应产出。
	assert.Contains(t, found, "recall@10_hybrid")
	assert.Contains(t, found, "recall@10_vector")
	assert.Contains(t, found, "recall@10_keyword")

	// 三种 mode 的 MRR / neighbor_hit_rate 也应产出。
	assert.Contains(t, found, "MRR_hybrid")
	assert.Contains(t, found, "neighbor_hit_rate_hybrid")

	// 应有 mode_compare（三模式两两组合：hybrid/vector、hybrid/keyword、vector/keyword）。
	assert.Contains(t, found, "mode_compare_hybrid_vs_vector")
	assert.Contains(t, found, "mode_compare_hybrid_vs_keyword")

	// hybrid 的 recall 应大于 0（query 与 docstring 语义相近，有真 embedding 命中）。
	// 不卡硬阈值（embedding 质量因模型而异），只验证能跑通 + 命中。
	assert.Greater(t, found["recall@10_hybrid"], 0.0, "hybrid recall 应大于 0（有语义命中）")

	// neighbor_hit_rate 应大于 0：LogAccess 不在 Top-K 主命中里，但作为 LoginUser
	// 的 callee 经图谱邻居扩展被发现。这是 neighbor_hit_rate 设计要验证的「图谱扩展
	// 能召回语义远但调用相关」的价值——若 ExpandHops 未开或数据里无邻居专属符号，
	// 此指标恒为 0。
	assert.Greater(t, found["neighbor_hit_rate_hybrid"], 0.0,
		"neighbor_hit_rate 应大于 0（LogAccess 仅通过邻居发现）")
}
