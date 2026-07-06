package retrieval

// 本文件是 HybridRetriever 的真 DB 集成测试（端到端验证检索 + 1 跳图谱扩展）。
//
// 运行前提：需要可连接的 PostgreSQL（由 DB_HOST/DB_PORT/... 环境变量或默认值
// codeatlas@localhost:5432 指定）。goose 迁移会自动建库 + 初始化 schema。
// 单元测试（make test / go test -short）会因 testing.Short() 跳过本文件，
// 实际执行留 Task 17 全量集成验证。
//
// DB setup 方案：在 retrieval 包内自建轻量 helper，复刻 tests/integration 的
// SetupTestDB 模式（连 postgres 库 → CREATE DATABASE → 跑 goose 迁移 → 返回 *models.DB）。
// 之所以不复用 tests/integration.SetupTestDB，是因为该函数未导出、跨包无法调用。

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/config"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// --- DB setup helpers（复刻 tests/integration/test_utils.go 的逻辑）---

// integrationTestDB 包裹一个 *models.DB 及其库名，供测试结束后回收。
type integrationTestDB struct {
	*models.DB
	dbName string
}

// setupIntegrationTestDB 创建唯一测试库并跑 goose 迁移，返回 *models.DB。
//
// 与 tests/integration.SetupTestDB 等价，但定义在本包内以规避跨包不可见。
// 失败时直接 t.Fatalf（与集成测试惯例一致）。
func setupIntegrationTestDB(t *testing.T) *integrationTestDB {
	t.Helper()

	// 测试期关闭 DB 日志降噪。
	models.SetDBLogger(nil)

	dbName := fmt.Sprintf("codeatlas_test_%s", uuid.New().String()[:8])

	// 先连 postgres 默认库用于 CREATE DATABASE。
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

	// 切换到新建库连接。
	cfg.Database = dbName
	testDB, err := models.NewDBWithConfig(cfg)
	if err != nil {
		adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 走真源 goose 迁移初始化 schema（与生产一致，含 content_tsv 等 BM25 列）。
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

// --- 集成测试主体 ---

// TestHybridRetriever_Integration 在真 DB 上端到端验证 HybridRetriever：
// keyword 检索召回 FuncA，并经 1 跳图谱扩展拿到其 Callees（含 FuncB）。
//
// fixture：
//   - 1 repo + 1 file
//   - 符号 FuncA、FuncB（function），FuncA 调用 FuncB（call 边）
//   - FuncA 的向量（content 含 "FuncA" 关键词，使 keyword 检索能命中）
//
// 断言：
//   - blocks 非空
//   - 命中符号为 FuncA
//   - FuncA 的 Callees 包含 FuncB
func TestHybridRetriever_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tdb := setupIntegrationTestDB(t)
	defer tdb.teardown(t)

	ctx := context.Background()

	// 向量维度与迁移硬编码（1024）一致。
	vectorDim := testEnvInt("EMBEDDING_DIMENSIONS", 1024)

	repoRepo := models.NewRepositoryRepository(tdb.DB)
	fileRepo := models.NewFileRepository(tdb.DB)
	symbolRepo := models.NewSymbolRepository(tdb.DB)
	edgeRepo := models.NewEdgeRepository(tdb.DB)
	vectorRepo := models.NewVectorRepository(tdb.DB)

	// 1. repo + file
	repo := &models.Repository{RepoID: uuid.New().String(), Name: "demo-repo"}
	if err := repoRepo.Create(ctx, repo); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	file := &models.File{
		FileID:   uuid.New().String(),
		RepoID:   repo.RepoID,
		Path:     "demo.go",
		Language: "go",
		Size:     100,
		Checksum: "abc",
	}
	if err := fileRepo.Create(ctx, file); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// 2. 两个符号：FuncA 调用 FuncB
	funcA := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "FuncA",
		Kind:      "function",
		Signature: "func FuncA()",
		StartLine: 1,
		EndLine:   10,
		Docstring: "FuncA is the entrypoint that delegates to FuncB",
	}
	if err := symbolRepo.Create(ctx, funcA); err != nil {
		t.Fatalf("Failed to create symbol FuncA: %v", err)
	}
	funcB := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "FuncB",
		Kind:      "function",
		Signature: "func FuncB()",
		StartLine: 12,
		EndLine:   20,
		Docstring: "FuncB does the real work",
	}
	if err := symbolRepo.Create(ctx, funcB); err != nil {
		t.Fatalf("Failed to create symbol FuncB: %v", err)
	}

	// 3. call 边：FuncA -> FuncB
	edge := &models.Edge{
		EdgeID:     uuid.New().String(),
		SourceID:   funcA.SymbolID,
		TargetID:   &funcB.SymbolID,
		EdgeType:   "call",
		SourceFile: file.Path,
		TargetFile: &file.Path,
	}
	if err := edgeRepo.Create(ctx, edge); err != nil {
		t.Fatalf("Failed to create call edge: %v", err)
	}

	// 4. FuncA 的向量（keyword 模式走 content_tsv，content 需含可召回关键词）。
	zeroVec := make([]float32, vectorDim)
	vecA := &models.Vector{
		VectorID:   uuid.New().String(),
		EntityID:   funcA.SymbolID,
		EntityType: "symbol",
		Embedding:  zeroVec,
		Content:    "FuncA entrypoint delegates to FuncB",
		Model:      "test-model",
	}
	if err := vectorRepo.Create(ctx, vecA); err != nil {
		t.Fatalf("Failed to create vector for FuncA: %v", err)
	}

	// 构造被测对象：keyword 模式不调用 embedder，传 nil 即可。
	r := NewHybridRetriever(vectorRepo, edgeRepo, nil, DefaultHybridRetrieverConfig())

	t.Run("keyword search returns block with callees", func(t *testing.T) {
		blocks, err := r.Query(ctx, RetrievalRequest{
			Query:         "FuncA",
			Mode:          "keyword",
			RepoIDs:       []string{repo.RepoID},
			ExpandHops:    1,
			ExpandCallers: true,
			ExpandCallees: true,
		})
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(blocks) == 0 {
			t.Fatal("expected non-empty blocks, got none")
		}

		// 主命中应为 FuncA（keyword 召回的正是 FuncA 的向量）。
		var hit *ContextBlock
		for i := range blocks {
			if blocks[i].Symbol.SymbolID == funcA.SymbolID {
				hit = &blocks[i]
				break
			}
		}
		if hit == nil {
			t.Fatalf("FuncA (symbol_id=%s) not found in %d blocks", funcA.SymbolID, len(blocks))
		}
		if hit.Symbol.Name != "FuncA" {
			t.Errorf("hit symbol name = %q, want FuncA", hit.Symbol.Name)
		}

		// FuncA 的 Callees 应包含 FuncB（call 边 FuncA -> FuncB）。
		var foundB bool
		for _, c := range hit.Callees {
			if c.SymbolID == funcB.SymbolID {
				foundB = true
				if c.Name != "FuncB" {
					t.Errorf("callee name = %q, want FuncB", c.Name)
				}
				break
			}
		}
		if !foundB {
			t.Errorf("FuncB (symbol_id=%s) not found in FuncA callees: %+v", funcB.SymbolID, hit.Callees)
		}
	})
}
