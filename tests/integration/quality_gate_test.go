package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
	"github.com/yourtionguo/CodeAtlas/internal/quality"
	"github.com/yourtionguo/CodeAtlas/internal/quality/fixtures"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// fixtureParser 是 parser 包各语言解析器共同实现的 Parse 方法签名。
type fixtureParser interface {
	Parse(file parser.ScannedFile) (*parser.ParsedFile, error)
}

// TestQualityGate_FixtureMode 在真 DB 上索引 fixture，跑依赖图指标门禁。
func TestQualityGate_FixtureMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoID := indexRealFixtures(t, testDB, ctx)

	edgeRepo := models.NewEdgeRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)
	fetcher := quality.NewDefaultGraphFetcher(edgeRepo, symbolRepo)

	// 合并真值
	truth := &quality.GraphGroundTruth{FixtureFile: "merged"}
	for _, gt := range fixtures.CallAnalysisGroundTruth {
		truth.Edges = append(truth.Edges, gt.Edges...)
		truth.Chains = append(truth.Chains, gt.Chains...)
	}
	graphEval := quality.NewGraphEvaluator(fetcher, truth)

	report, err := quality.Evaluate(ctx, quality.EvaluateConfig{
		Mode:       quality.EvalModeFixture,
		FixtureSet: "call_analysis",
		RepoID:     repoID,
	}, graphEval, nil)
	require.NoError(t, err)

	// 打印报告供调试
	t.Logf("=== Fixture 模式报告（合并真值）===")
	for _, m := range report.Metrics {
		t.Logf("  %s (bucket=%s) = %.4f threshold=%.2f passed=%v", m.Name, m.Bucket, m.Value, m.Threshold, m.Passed)
	}

	// 门禁断言：所有有阈值的指标必须通过
	for _, m := range report.Metrics {
		if m.Threshold > 0 && !m.Passed {
			t.Errorf("质量门禁失败: %s (bucket=%s) = %.4f, 阈值 %.2f", m.Name, m.Bucket, m.Value, m.Threshold)
		}
	}
}

// TestQualityGate_RepoMode 验证 repo 模式能跑通结构断言（不卡阈值，建基线）。
func TestQualityGate_RepoMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoID := indexRealFixtures(t, testDB, ctx)

	edgeRepo := models.NewEdgeRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)
	fetcher := quality.NewDefaultGraphFetcher(edgeRepo, symbolRepo)
	graphEval := quality.NewGraphEvaluator(fetcher, nil)

	report, err := quality.Evaluate(ctx, quality.EvaluateConfig{
		Mode:   quality.EvalModeRepo,
		RepoID: repoID,
	}, graphEval, nil)
	require.NoError(t, err)

	assert.NotEmpty(t, report.Metrics, "应产出结构断言指标")
	assert.Equal(t, len(report.Metrics), report.Summary.Total)

	t.Logf("=== Repo 模式基线 ===")
	for _, m := range report.Metrics {
		t.Logf("  %s (bucket=%s) = %.4f", m.Name, m.Bucket, m.Value)
	}
}

// indexRealFixtures 解析真实 fixture 文件并索引到测试库，返回 repoID。
// 使用 parser 解析 tests/fixtures/ 下的文件，经 SchemaMapper 转为 schema.ParseOutput。
func indexRealFixtures(t *testing.T, testDB *TestDB, ctx context.Context) string {
	t.Helper()

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	// 收集所有真值引用的 fixture 文件。
	// 列表必须覆盖 fixtures.CallAnalysisGroundTruth 里引用的全部源文件，
	// 否则符号/边缺失会导致门禁失败。
	// path 用仓库相对规范路径（tests/fixtures/...），与真值/调用链查询里的 file path 对齐；
	// 读盘用从测试包（tests/integration）出发的 ../../tests/fixtures/... 绝对路径。
	fixtureFiles := []struct {
		path string // 入库的规范路径
		lang string
	}{
		{"tests/fixtures/cpp/cpp_calls_c.cpp", "cpp"},
		{"tests/fixtures/cpp/c_library.h", "cpp"},
		{"tests/fixtures/objc/simple_c_calls.m", "objc"},
		{"tests/fixtures/objc/simple_cpp_calls.mm", "objcpp"},
		{"tests/fixtures/kotlin/kotlin_calls_java.kt", "kotlin"},
		{"tests/fixtures/swift/swift_calls_objc.swift", "swift"},
		{"tests/fixtures/js/typescript_calls_js.ts", "js"},
	}

	// 使用两遍扫描：第一遍 CollectSymbols 累积全仓库符号候选 + import 关系，
	// 第二遍 ResolveEdges 用候选集解析所有边（含跨文件调用，如
	// cpp_calls_c.cpp 的 processData -> c_library.h 的 c_process_string）。
	// 这与 cmd/cli parse/index 命令的实际管线一致。
	mapper := schema.NewSchemaMapper()
	var schemaFiles []schema.File
	for _, ff := range fixtureFiles {
		// 读盘路径：测试包在 tests/integration，fixture 在 tests/fixtures。
		absPath, err := filepath.Abs(filepath.Join("..", "..", ff.path))
		require.NoError(t, err)

		file := parser.ScannedFile{
			Path:     ff.path, // 入库路径用规范形式，与真值对齐
			AbsPath:  absPath,
			Language: ff.lang,
		}

		p, err := getParser(tsParser, ff.lang)
		require.NoError(t, err)

		parsed, err := p.Parse(file)
		if err != nil {
			// 部分 fixture（如 .mm / .ts）tree-sitter 会报语法错误但仍能提取部分符号，
			// 这与 CLI 行为一致：容忍 parse error，使用已提取的结果。
			t.Logf("解析 %s 有错误（使用已提取结果）: %v", ff.path, err)
		}
		if parsed == nil {
			t.Logf("解析 %s 返回 nil（跳过）", ff.path)
			continue
		}

		schemaFile, err := mapper.CollectSymbols(parsed)
		if err != nil {
			t.Logf("映射 %s 失败（跳过）: %v", ff.path, err)
			continue
		}
		schemaFiles = append(schemaFiles, *schemaFile)
	}
	require.NotEmpty(t, schemaFiles, "至少应成功索引一个 fixture 文件")

	// 第二遍：用全仓库候选集解析所有边（含跨文件调用）。
	allEdges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	// 注：不显式收集外部模块符号（mapper.GetExternalSymbols）。
	// 索引器 ensureExternalFile 会自行创建外部虚拟文件；而多数 import 边的 target
	// 为外部依赖（无对应符号），target_id 为空，与真值（target_name 空）一致即可。
	// 显式注入外部符号会让 import 边 target_name 变为模块名，反而偏离真值。

	parseOutput := &schema.ParseOutput{
		Files:         schemaFiles,
		Relationships: allEdges,
		Metadata:      schema.ParseMetadata{Version: "1.0.0", TotalFiles: len(schemaFiles), SuccessCount: len(schemaFiles)},
	}

	// 调试：统计 ResolveEdges 产出的边
	t.Logf("ResolveEdges 产出 %d 条边（filterValidEdges 前）", len(allEdges))

	// 过滤掉违反 DB 约束的边，使其满足 indexer 校验器：
	//   - source_id 必须非空（edges.source_id 为 NOT NULL uuid）。
	//   - target_id 若非空，必须指向已索引符号（edges.target_id 为外键）。
	// target_id 空（悬空）的边保留——validator 已允许非 import 边悬空，
	// 这是合法状态，供 symbol_resolution_rate 指标观测。
	// 跨文件边若消解成功，target_id 指向另一文件的符号，此处的 symbolIDs 校验
	// 会通过（所有文件的符号都已收录在 parseOutput.Files 里），不会被误丢。
	parseOutput.Relationships = filterValidEdges(parseOutput)
	t.Logf("filterValidEdges 后保留 %d 条边", len(parseOutput.Relationships))

	repoID := uuid.New().String()
	config := &indexer.IndexerConfig{
		RepoID:          repoID,
		RepoName:        "quality-gate-test",
		BatchSize:       10,
		WorkerCount:     2,
		SkipVectors:     true,
		UseTransactions: true,
	}
	idx := indexer.NewIndexer(testDB.DB, config)
	_, err = idx.Index(ctx, parseOutput)
	require.NoError(t, err)
	return repoID
}

// getParser 根据语言返回对应解析器
func getParser(ts *parser.TreeSitterParser, lang string) (fixtureParser, error) {
	switch lang {
	case "cpp":
		return parser.NewCppParser(ts), nil
	case "c":
		return parser.NewCParser(ts), nil
	case "java":
		return parser.NewJavaParser(ts), nil
	case "kotlin":
		return parser.NewKotlinParser(ts), nil
	case "swift":
		return parser.NewSwiftParser(ts), nil
	case "objc":
		return parser.NewObjCParser(ts), nil
	case "objcpp":
		return parser.NewObjCppParser(ts), nil
	case "js", "typescript":
		return parser.NewJSParser(ts), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

// filterValidEdges 丢弃违反 DB 约束的边，使其满足 indexer 校验器：
//   - source_id 必须非空（edges.source_id 为 NOT NULL uuid；import 边在
//     parser 层 Source="" 时会得到空 source_id，需丢弃）。
//   - target_id 若非空，必须指向已索引符号（edges.target_id 为外键）。
//
// 注意：target_id 空（悬空）的边保留——validator 已允许非 import 边悬空，
// 这是合法状态。跨文件调用经 SchemaMapper 两遍扫描后：
//   - 若 target 消解到本仓库内某符号，target_id 非空且通过 symbolIDs 校验，保留；
//   - 若 target 解析不到（如标准库 strlen、未索引的外部类），target_id 空，保留为悬空。
func filterValidEdges(out *schema.ParseOutput) []schema.DependencyEdge {
	symbolIDs := make(map[string]bool, len(out.Files)*4)
	for _, f := range out.Files {
		for _, s := range f.Symbols {
			symbolIDs[s.SymbolID] = true
		}
	}
	var kept []schema.DependencyEdge
	for _, e := range out.Relationships {
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
