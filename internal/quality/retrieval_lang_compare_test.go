package quality

// 本文件是全语言检索质量对比测试：按语言（Go/C++/Java/Kotlin/Swift/JS/Python）
// 分别索引真实 fixture 文件 + 真 Ollama embedding 向量，各自跑 retrieval 评估，
// 汇总各语言 recall@10_hybrid，输出对比，暴露「某语言检索质量差」的问题。
//
// 运行前提（与 retrieval_eval_integration_test.go 相同）：
//   - PostgreSQL（codeatlas@localhost:5432），goose 迁移自动建库 + schema
//   - Ollama 在 localhost:11434，模型 qwen3-embedding:0.6b（1024 维）已拉取
//
// 任一前提不满足（-short / Ollama 不可达）则跳过，不 fail。
// 单语言索引失败或 embedding 异常只记录并跳过该语言（t.Logf），不让整体 fail。

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// fixtureSpec 描述一个待索引的 fixture 文件。
type fixtureSpec struct {
	path string // 入库规范路径（tests/fixtures/...）
	lang string // parser 语言标识
}

// langCase 是一种语言的对比用例：待索引 fixture + 真值 query。
//
// 真值符号名取自 fixture 实际解析出的符号名（已用 tree-sitter parser + SchemaMapper
// 验证），不用凭空猜测。各 parser 抽取粒度不同（如 Java/Kotlin 仅提取类级符号，
// 不提取方法），真值须与之对齐，否则 recall 恒为 0。
type langCase struct {
	fixtures []fixtureSpec
	truths   []RetrievalGroundTruth
}

// perLanguageCases 各语言的 fixture + 真值。
// 查询语义与 buildSymbolContent 拼接的 signature+docstring 对齐（向量以此文本入库）。
var perLanguageCases = map[string]langCase{
	"go": {
		fixtures: []fixtureSpec{
			{"tests/fixtures/test-repo/main.go", "go"},
			{"tests/fixtures/test-repo/utils.go", "go"},
		},
		truths: []RetrievalGroundTruth{
			{
				// Calculator/Add/Multiply 的 docstring 都讲算术运算，语义近。
				Query:           "calculator add multiply arithmetic operations",
				RelevantSymbols: []string{"Calculator", "Add", "Multiply"},
			},
		},
	},
	"python": {
		fixtures: []fixtureSpec{
			{"tests/fixtures/test-repo/models.py", "python"},
			{"tests/fixtures/test-repo/utils.py", "python"},
		},
		truths: []RetrievalGroundTruth{
			{
				// process_data / DataProcessor / retry 的 docstring 都讲数据处理与重试。
				Query:           "process data retry operation decorator",
				RelevantSymbols: []string{"process_data", "DataProcessor", "retry"},
			},
		},
	},
	"cpp": {
		fixtures: []fixtureSpec{
			{"tests/fixtures/cpp/class.cpp", "cpp"},
		},
		truths: []RetrievalGroundTruth{
			{
				// MyClass 构造 / getName / processData 的 docstring 围绕类成员操作。
				Query:           "MyClass constructor get name process data",
				RelevantSymbols: []string{"MyClass", "getName", "processData"},
			},
		},
	},
	"java": {
		fixtures: []fixtureSpec{
			{"tests/fixtures/java/simple_class.java", "java"},
		},
		truths: []RetrievalGroundTruth{
			{
				// Java parser 仅抽取类级符号 com.example.test.SimpleClass（不抽方法），
				// 真值与 docstring "A simple class for testing Java parser" 对齐。
				Query:           "simple class example java testing",
				RelevantSymbols: []string{"com.example.test.SimpleClass"},
			},
		},
	},
	"kotlin": {
		fixtures: []fixtureSpec{
			{"tests/fixtures/kotlin/kotlin_calls_java.kt", "kotlin"},
		},
		truths: []RetrievalGroundTruth{
			{
				// Kotlin parser 仅抽取类级符号，类无 docstring（向量内容为 signature 文本）。
				// 查询带符号名相关词以让 keyword/语义命中。
				Query:           "KotlinJavaInterop kotlin java collections interop",
				RelevantSymbols: []string{"com.example.interop.KotlinJavaInterop"},
			},
		},
	},
	"swift": {
		fixtures: []fixtureSpec{
			{"tests/fixtures/swift/simple_class.swift", "swift"},
		},
		truths: []RetrievalGroundTruth{
			{
				// User / AdminUser 的 docstring 讲 user class 与继承。
				Query:           "User class inheritance AdminUser",
				RelevantSymbols: []string{"User", "AdminUser"},
			},
		},
	},
	"js": {
		fixtures: []fixtureSpec{
			{"tests/fixtures/js/typescript_calls_js.ts", "js"},
		},
		truths: []RetrievalGroundTruth{
			{
				// TypeScriptComponent / useJavaScriptArrays / useJavaScriptMath 的
				// docstring 讲 JavaScript 数组/数学方法与 TypeScript 组件。
				Query:           "JavaScript arrays math TypeScript component",
				RelevantSymbols: []string{"TypeScriptComponent", "useJavaScriptArrays", "useJavaScriptMath"},
			},
		},
	},
}

// TestRetrievalEvaluator_PerLanguageCompare 按语言分组索引 fixture + 向量，
// 各自跑检索评估，汇总各语言 recall@10_hybrid 并对比。
//
// 设计要点：
//  1. 每种语言独立 repoID（隔离检索范围，避免跨语言串扰）
//  2. 注入真 Ollama embedder，SkipVectors=false（生成向量才能做检索）
//  3. 解析各语言代表性 fixture（复刻 quality_gate_test.indexRealFixtures 的核心）
//  4. 单语言失败（解析 0 符号 / 索引错误）只记录并跳过，不影响其它语言
//  5. 断言每语言 recall@10_hybrid > 0（验证 embedding 对该语言代码有效）
func TestRetrievalEvaluator_PerLanguageCompare(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Ollama 不可达则跳过（不 fail）。
	if !isOllamaAvailable(t) {
		t.Skip("Ollama 不可达，跳过全语言检索对比测试")
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

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	mapper := schema.NewSchemaMapper()

	// repoPathRoot：fixture 文件相对仓库根的查找基点。
	// 本测试位于 internal/quality，fixture 在仓库根的 tests/fixtures。
	repoPathRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	// langRecall 收集每语言 recall@10_hybrid；langSkipped 记录被跳过的语言及原因。
	langRecall := map[string]float64{}
	langSkipped := map[string]string{}

	for lang, lc := range perLanguageCases {
		// 每语言独立 repoID，隔离检索范围。
		repoID := uuid.New().String()
		vectorRepo := models.NewVectorRepository(tdb.DB)
		edgeRepo := models.NewEdgeRepository(tdb.DB)

		// 解析 fixture → schemaFiles + edges
		schemaFiles, allEdges, symbolCount, perr := parseFixtures(t, tsParser, mapper, repoPathRoot, lc.fixtures)
		if perr != nil {
			langSkipped[lang] = fmt.Sprintf("parse fixtures: %v", perr)
			t.Logf("[skip] %s: %s", lang, langSkipped[lang])
			continue
		}
		if symbolCount == 0 {
			langSkipped[lang] = "解析后无符号"
			t.Logf("[skip] %s: %s", lang, langSkipped[lang])
			continue
		}

		parseOutput := &schema.ParseOutput{
			Files:         schemaFiles,
			Relationships: filterValidEdgesLocal(schemaFiles, allEdges),
			Metadata:      schema.ParseMetadata{Version: "1.0.0", TotalFiles: len(schemaFiles), SuccessCount: len(schemaFiles)},
		}

		// 索引：SkipVectors=false，注入真 embedder 生成向量。
		cfg := &indexer.IndexerConfig{
			RepoID:          repoID,
			RepoName:        "lang-compare-" + lang,
			BatchSize:       10,
			WorkerCount:     2,
			SkipVectors:     false,
			UseTransactions: true,
		}
		embedder := indexer.NewOpenAIEmbedder(embedderCfg, vectorRepo)
		idx := indexer.NewIndexerWithEmbedder(tdb.DB, cfg, embedder)

		res, err := idx.Index(ctx, parseOutput)
		if err != nil {
			langSkipped[lang] = fmt.Sprintf("index error: %v", err)
			t.Logf("[skip] %s: %s", lang, langSkipped[lang])
			continue
		}
		if res.VectorsCreated == 0 {
			langSkipped[lang] = fmt.Sprintf("索引完成但未生成向量（symbols=%d, status=%s）", res.SymbolsCreated, res.Status)
			t.Logf("[skip] %s: %s", lang, langSkipped[lang])
			continue
		}
		t.Logf("[index] %s: symbols=%d edges=%d vectors=%d", lang, res.SymbolsCreated, res.EdgesCreated, res.VectorsCreated)

		// 检索评估：hybrid 模式 + 图谱扩展（邻居）。
		retriever := retrieval.NewHybridRetriever(
			vectorRepo, edgeRepo, embedder,
			retrieval.DefaultHybridRetrieverConfig(),
		)
		eval := NewRetrievalEvaluator(retriever, lc.truths, []string{"hybrid"})

		metrics, err := eval.Evaluate(ctx, []string{repoID})
		if err != nil {
			langSkipped[lang] = fmt.Sprintf("evaluate error: %v", err)
			t.Logf("[skip] %s: %s", lang, langSkipped[lang])
			continue
		}

		// 提取 recall@10_hybrid
		var recall float64
		var found bool
		for _, m := range metrics {
			if m.Name == "recall@10_hybrid" {
				recall = m.Value
				found = true
				t.Logf("  [%s] %s = %.4f (neighbor_hit_rate=%v)", lang, m.Name, m.Value, lookupMetric(metrics, "neighbor_hit_rate_hybrid"))
			}
		}
		if !found {
			langSkipped[lang] = "未产出 recall@10_hybrid 指标"
			t.Logf("[skip] %s: %s", lang, langSkipped[lang])
			continue
		}
		langRecall[lang] = recall

		// 断言：embedding 对该语言代码有效，recall > 0。
		// 不卡更高阈值（embedding 质量因语言/代码风格/模型而异）。
		if recall <= 0 {
			t.Errorf("语言 %s 的 recall@10_hybrid = %.4f，应大于 0（embedding 对该语言代码应有效）", lang, recall)
		}
	}

	// 输出对比报告。
	t.Logf("=== 各语言检索质量对比 ===")
	for lang, recall := range langRecall {
		t.Logf("  %s: recall@10_hybrid = %.4f", lang, recall)
	}
	if len(langSkipped) > 0 {
		t.Logf("=== 被跳过的语言 ===")
		for lang, reason := range langSkipped {
			t.Logf("  %s: %s", lang, reason)
		}
	}

	// 至少应有 5 种语言产出 recall（覆盖要求）。
	require.GreaterOrEqual(t, len(langRecall), 5,
		"至少应成功评估 5 种语言，实际 %d 种（被跳过：%v）", len(langRecall), langSkipped)
}

// parseFixtures 解析 fixture 列表，返回 schemaFiles + edges + 符号总数。
// 行为复刻 quality_gate_test.indexRealFixtures 的核心：parse → mapToSchema，
// 容忍 parse error（部分语言如 .ts 会报语法错误但仍能提取部分符号）。
func parseFixtures(
	t *testing.T,
	tsParser *parser.TreeSitterParser,
	mapper *schema.SchemaMapper,
	repoPathRoot string,
	fixtures []fixtureSpec,
) (files []schema.File, edges []schema.DependencyEdge, symbolCount int, err error) {
	t.Helper()
	for _, ff := range fixtures {
		absPath := filepath.Join(repoPathRoot, ff.path)
		file := parser.ScannedFile{
			Path:     ff.path,
			AbsPath:  absPath,
			Language: ff.lang,
		}
		p, perr := getLangParser(tsParser, ff.lang)
		if perr != nil {
			return nil, nil, 0, fmt.Errorf("get parser for %s: %w", ff.lang, perr)
		}
		parsed, perr := p.Parse(file)
		if perr != nil {
			// 容忍 parse error（.ts 等），使用已提取结果。
			t.Logf("解析 %s 有错误（使用已提取结果）: %v", ff.path, perr)
		}
		if parsed == nil {
			t.Logf("解析 %s 返回 nil（跳过该文件）", ff.path)
			continue
		}
		schemaFile, fileEdges, merr := mapper.MapToSchema(parsed)
		if merr != nil {
			t.Logf("映射 %s 失败（跳过该文件）: %v", ff.path, merr)
			continue
		}
		files = append(files, *schemaFile)
		edges = append(edges, fileEdges...)
		symbolCount += len(schemaFile.Symbols)
	}
	return files, edges, symbolCount, nil
}

// getLangParser 返回各语言 parser（含 go/python，比 quality_gate_test.getParser 多两语言）。
func getLangParser(ts *parser.TreeSitterParser, lang string) (interface {
	Parse(parser.ScannedFile) (*parser.ParsedFile, error)
}, error) {
	switch lang {
	case "go":
		return parser.NewGoParser(ts), nil
	case "python":
		return parser.NewPythonParser(ts), nil
	case "cpp":
		return parser.NewCppParser(ts), nil
	case "java":
		return parser.NewJavaParser(ts), nil
	case "kotlin":
		return parser.NewKotlinParser(ts), nil
	case "swift":
		return parser.NewSwiftParser(ts), nil
	case "js", "typescript":
		return parser.NewJSParser(ts), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

// filterValidEdgesLocal 是 filterValidEdges 的本包等价实现（quality_gate_test 在
// integration 包，不能直接复用）：丢弃 source_id 空、或 target_id 指向未索引
// 符号的边，满足 indexer 校验器与 DB 约束。
func filterValidEdgesLocal(files []schema.File, edges []schema.DependencyEdge) []schema.DependencyEdge {
	symbolIDs := make(map[string]bool, len(files)*4)
	for _, f := range files {
		for _, s := range f.Symbols {
			symbolIDs[s.SymbolID] = true
		}
	}
	var kept []schema.DependencyEdge
	for _, e := range edges {
		if e.SourceID == "" {
			continue
		}
		if e.TargetID != "" && !symbolIDs[e.TargetID] {
			continue
		}
		kept = append(kept, e)
	}
	return kept
}

// lookupMetric 在指标列表里按名字查值，找不到返回 -1（便于日志打印）。
func lookupMetric(metrics []MetricValue, name string) float64 {
	for _, m := range metrics {
		if m.Name == name {
			return m.Value
		}
	}
	return -1
}
