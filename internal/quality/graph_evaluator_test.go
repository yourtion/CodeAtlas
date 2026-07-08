package quality

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubGraphFetcher 是 GraphDataFetcher 的 mock 实现。
type stubGraphFetcher struct {
	byType       map[string]int
	dangling     map[string]int
	orphans      int
	crossFile    int
	totalSymbols int
	chainOK      int
	chainTotal   int
	extracted    []ExtractedEdge
	err          error
	// errMethod 设置后，仅当被调用的方法名匹配时返回 errMethodErr，
	// 用于精细覆盖各方法在 GraphEvaluator.Evaluate 中的错误包装路径。
	errMethod    string
	errMethodErr error
}

func (s *stubGraphFetcher) methodErr(method string) error {
	if s.errMethod != "" && s.errMethod == method {
		return s.errMethodErr
	}
	return s.err
}

func (s *stubGraphFetcher) CountEdgesByType(ctx context.Context, repoID string) (map[string]int, error) {
	return s.byType, s.methodErr("CountEdgesByType")
}
func (s *stubGraphFetcher) CountDanglingEdges(ctx context.Context, repoID string) (map[string]int, error) {
	return s.dangling, s.methodErr("CountDanglingEdges")
}
func (s *stubGraphFetcher) CountOrphanSymbols(ctx context.Context, repoID string) (int, error) {
	return s.orphans, s.methodErr("CountOrphanSymbols")
}
func (s *stubGraphFetcher) CountCrossFileEdges(ctx context.Context, repoID string) (int, error) {
	return s.crossFile, s.methodErr("CountCrossFileEdges")
}
func (s *stubGraphFetcher) CountTotalSymbols(ctx context.Context, repoID string) (int, error) {
	return s.totalSymbols, s.methodErr("CountTotalSymbols")
}
func (s *stubGraphFetcher) CheckCallChainConnectivity(ctx context.Context, repoID string, chains []ExpectedChain) (int, int, error) {
	return s.chainOK, s.chainTotal, s.methodErr("CheckCallChainConnectivity")
}
func (s *stubGraphFetcher) ListExtractedEdges(ctx context.Context, repoID string) ([]ExtractedEdge, error) {
	return s.extracted, s.methodErr("ListExtractedEdges")
}

func TestGraphEvaluator_RepoMode_StructuralMetrics(t *testing.T) {
	fetcher := &stubGraphFetcher{
		byType:       map[string]int{"call": 80, "import": 20},
		dangling:     map[string]int{"call": 4, "import": 10},
		orphans:      5,
		crossFile:    30,
		totalSymbols: 50,
	}
	eval := NewGraphEvaluator(fetcher, nil) // repo 模式 truth=nil

	metrics, err := eval.Evaluate(context.Background(), "repo-1", EvalModeRepo)
	require.NoError(t, err)

	// 应有结构断言类指标，无真值类指标
	found := map[string]bool{}
	for _, m := range metrics {
		found[m.Name] = true
		assert.Equal(t, CategoryGraph, m.Category)
	}
	assert.True(t, found["dangling_edge_ratio"])
	assert.True(t, found["symbol_resolution_rate"])
	assert.True(t, found["orphan_symbol_ratio"])
	assert.True(t, found["cross_file_connectivity"])
	assert.False(t, found["edge_recall"], "repo 模式不应有真值类指标")

	// 验证悬空边率：call 4/80=0.05, import 10/20=0.5, 总 14/100=0.14
	var totalDangling MetricValue
	for _, m := range metrics {
		if m.Name == "dangling_edge_ratio" && m.Bucket == "" {
			totalDangling = m
		}
	}
	assert.InDelta(t, 0.14, totalDangling.Value, 0.001)
}

func TestGraphEvaluator_FixtureMode_IncludesTruthMetrics(t *testing.T) {
	fetcher := &stubGraphFetcher{
		byType:     map[string]int{"call": 10},
		dangling:   map[string]int{"call": 1},
		chainOK:    9,
		chainTotal: 10,
		extracted: []ExtractedEdge{
			{SourceName: "A", EdgeType: "call", TargetName: "B"}, // 命中真值
			{SourceName: "A", EdgeType: "call", TargetName: "X"}, // 不在真值里（降低 precision）
		},
	}
	truth := &GraphGroundTruth{
		FixtureFile: "test.go",
		Edges: []ExpectedEdge{
			{SourceName: "A", EdgeType: "call", TargetName: "B"},
			{SourceName: "A", EdgeType: "call", TargetName: "C", Optional: true}, // optional，不算漏
		},
		Chains: []ExpectedChain{
			{StartName: "A", EndName: "B", StartFile: "a.go", EndFile: "b.go"},
		},
	}
	eval := NewGraphEvaluator(fetcher, truth)

	metrics, err := eval.Evaluate(context.Background(), "repo-1", EvalModeFixture)
	require.NoError(t, err)

	found := map[string]float64{}
	for _, m := range metrics {
		found[m.Name] = m.Value
	}
	assert.Contains(t, found, "call_chain_connectivity", "应有调用链连通性")
	assert.Contains(t, found, "edge_recall", "应有边召回率")
	assert.Contains(t, found, "edge_precision", "应有边准确率")

	// edge_recall: 真值非 optional 边 1 条（A->B），提取命中 1，recall=1.0
	assert.InDelta(t, 1.0, found["edge_recall"], 0.001)
	// edge_precision: 提取 2 条，匹配真值 1 条，precision=0.5
	assert.InDelta(t, 0.5, found["edge_precision"], 0.001)
	// 连通性：9/10=0.9
	assert.InDelta(t, 0.9, found["call_chain_connectivity"], 0.001)
}

// TestGraphEvaluator_FetcherError 验证 fetcher 错误被正确包装返回。
func TestGraphEvaluator_FetcherError(t *testing.T) {
	fetcher := &stubGraphFetcher{
		err: fmt.Errorf("db connection lost"),
	}
	eval := NewGraphEvaluator(fetcher, nil)
	_, err := eval.Evaluate(context.Background(), "repo-1", EvalModeRepo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CountEdgesByType")
}

// TestGraphEvaluator_StructuralFetchErrors 逐一覆盖结构断言类各方法的错误包装路径
// （CountDanglingEdges / CountOrphanSymbols / CountCrossFileEdges / CountTotalSymbols）。
// 用 errMethod 精细注入，前序方法成功、目标方法失败，确保每个错误分支被走到。
func TestGraphEvaluator_StructuralFetchErrors(t *testing.T) {
	base := stubGraphFetcher{
		byType:       map[string]int{"call": 10},
		dangling:     map[string]int{"call": 1},
		orphans:      2,
		crossFile:    3,
		totalSymbols: 5,
	}
	for _, method := range []string{
		"CountDanglingEdges", "CountOrphanSymbols", "CountCrossFileEdges", "CountTotalSymbols",
	} {
		t.Run(method, func(t *testing.T) {
			f := base
			f.errMethod = method
			f.errMethodErr = fmt.Errorf("%s boom", method)
			eval := NewGraphEvaluator(&f, nil)
			_, err := eval.Evaluate(context.Background(), "repo-1", EvalModeRepo)
			require.Error(t, err)
			assert.Contains(t, err.Error(), method)
		})
	}
}

// TestGraphEvaluator_FixtureModeFetchErrors 覆盖 fixture 模式下
// ListExtractedEdges 与 CheckCallChainConnectivity 的错误包装路径。
func TestGraphEvaluator_FixtureModeFetchErrors(t *testing.T) {
	truth := &GraphGroundTruth{
		FixtureFile: "test.go",
		Edges:       []ExpectedEdge{{SourceName: "A", EdgeType: "call", TargetName: "B"}},
		Chains:      []ExpectedChain{{StartName: "A", EndName: "B"}},
	}

	t.Run("ListExtractedEdges", func(t *testing.T) {
		f := &stubGraphFetcher{
			byType:       map[string]int{"call": 10},
			totalSymbols: 5,
			errMethod:    "ListExtractedEdges",
			errMethodErr: fmt.Errorf("list boom"),
		}
		eval := NewGraphEvaluator(f, truth)
		_, err := eval.Evaluate(context.Background(), "repo-1", EvalModeFixture)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ListExtractedEdges")
	})

	t.Run("CheckCallChainConnectivity", func(t *testing.T) {
		f := &stubGraphFetcher{
			byType:       map[string]int{"call": 10},
			totalSymbols: 5,
			extracted:    []ExtractedEdge{{SourceName: "A", EdgeType: "call", TargetName: "B"}},
			errMethod:    "CheckCallChainConnectivity",
			errMethodErr: fmt.Errorf("chain boom"),
		}
		eval := NewGraphEvaluator(f, truth)
		_, err := eval.Evaluate(context.Background(), "repo-1", EvalModeFixture)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CheckCallChainConnectivity")
	})
}
