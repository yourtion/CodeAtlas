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
}

func (s *stubGraphFetcher) CountEdgesByType(ctx context.Context, repoID string) (map[string]int, error) {
	return s.byType, s.err
}
func (s *stubGraphFetcher) CountDanglingEdges(ctx context.Context, repoID string) (map[string]int, error) {
	return s.dangling, s.err
}
func (s *stubGraphFetcher) CountOrphanSymbols(ctx context.Context, repoID string) (int, error) {
	return s.orphans, s.err
}
func (s *stubGraphFetcher) CountCrossFileEdges(ctx context.Context, repoID string) (int, error) {
	return s.crossFile, s.err
}
func (s *stubGraphFetcher) CountTotalSymbols(ctx context.Context, repoID string) (int, error) {
	return s.totalSymbols, s.err
}
func (s *stubGraphFetcher) CheckCallChainConnectivity(ctx context.Context, repoID string, chains []ExpectedChain) (int, int, error) {
	return s.chainOK, s.chainTotal, s.err
}
func (s *stubGraphFetcher) ListExtractedEdges(ctx context.Context, repoID string) ([]ExtractedEdge, error) {
	return s.extracted, s.err
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
