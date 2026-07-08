package quality

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

// stubRetrievalRunner 是 RetrievalRunner 的 mock 实现。
// 按 query -> mode -> blocks 三级映射返回，便于 ModeCompare 测试按 mode 区分结果。
type stubRetrievalRunner struct {
	blocksByQueryMode map[string]map[string][]retrieval.ContextBlock // query -> mode -> blocks
}

func (s *stubRetrievalRunner) Query(ctx context.Context, req retrieval.RetrievalRequest) ([]retrieval.ContextBlock, error) {
	if modes, ok := s.blocksByQueryMode[req.Query]; ok {
		return modes[req.Mode], nil
	}
	return nil, nil
}

func TestRetrievalEvaluator_RecallAndMRR(t *testing.T) {
	runner := &stubRetrievalRunner{
		blocksByQueryMode: map[string]map[string][]retrieval.ContextBlock{
			"q1": {
				"hybrid": {
					{Symbol: retrieval.ContextSymbol{Name: "Irrelevant"}},
					{Symbol: retrieval.ContextSymbol{Name: "TargetA"}}, // rank 2
					{Symbol: retrieval.ContextSymbol{Name: "TargetB"}}, // rank 3
				},
			},
		},
	}
	truths := []RetrievalGroundTruth{
		{Query: "q1", RelevantSymbols: []string{"TargetA", "TargetB"}},
	}
	eval := NewRetrievalEvaluator(runner, truths, []string{"hybrid"})

	metrics, err := eval.Evaluate(context.Background(), []string{"repo-1"})
	require.NoError(t, err)

	found := map[string]float64{}
	for _, m := range metrics {
		found[m.Name] = m.Value
	}
	// recall@10 = 2/2 = 1.0
	assert.InDelta(t, 1.0, found["recall@10_hybrid"], 0.001)
	// MRR = 1/2 = 0.5（第一个相关符号在 rank 2）
	assert.InDelta(t, 0.5, found["MRR_hybrid"], 0.001)
}

func TestRetrievalEvaluator_NeighborHitRate(t *testing.T) {
	runner := &stubRetrievalRunner{
		blocksByQueryMode: map[string]map[string][]retrieval.ContextBlock{
			"q1": {
				"hybrid": {
					{
						Symbol:  retrieval.ContextSymbol{Name: "Main"},
						Callers: []retrieval.ContextSymbol{{Name: "TargetA"}},
						Callees: []retrieval.ContextSymbol{{Name: "Other"}},
					},
				},
			},
		},
	}
	truths := []RetrievalGroundTruth{
		{Query: "q1", RelevantSymbols: []string{"Main", "TargetA"}},
	}
	eval := NewRetrievalEvaluator(runner, truths, []string{"hybrid"})

	metrics, err := eval.Evaluate(context.Background(), []string{"repo-1"})
	require.NoError(t, err)

	for _, m := range metrics {
		if m.Name == "neighbor_hit_rate_hybrid" {
			// 邻居里有 TargetA，真值相关 2 个，命中 1/2=0.5
			assert.InDelta(t, 0.5, m.Value, 0.001)
			return
		}
	}
	t.Fatal("neighbor_hit_rate_hybrid 指标未找到")
}

func TestRetrievalEvaluator_ModeCompare(t *testing.T) {
	runner := &stubRetrievalRunner{
		blocksByQueryMode: map[string]map[string][]retrieval.ContextBlock{
			"q1": {
				"hybrid":  {{Symbol: retrieval.ContextSymbol{Name: "Target"}}},
				"vector":  {{Symbol: retrieval.ContextSymbol{Name: "Irrelevant"}}},
				"keyword": {{Symbol: retrieval.ContextSymbol{Name: "Target"}}},
			},
		},
	}
	truths := []RetrievalGroundTruth{
		{Query: "q1", RelevantSymbols: []string{"Target"}},
	}
	eval := NewRetrievalEvaluator(runner, truths, []string{"hybrid", "vector", "keyword"})

	metrics, err := eval.Evaluate(context.Background(), []string{"repo-1"})
	require.NoError(t, err)

	// 应有 mode_compare 指标
	hasCompare := false
	for _, m := range metrics {
		if m.Name == "mode_compare_hybrid_vs_vector" {
			hasCompare = true
			// hybrid recall=1.0, vector recall=0.0, 差值=+1.0
			assert.InDelta(t, 1.0, m.Value, 0.001)
		}
	}
	assert.True(t, hasCompare, "应有 mode_compare 指标")
}

// failingRetrievalRunner 始终返回错误，用于覆盖 Evaluate 的查询错误分支。
type failingRetrievalRunner struct{}

func (failingRetrievalRunner) Query(ctx context.Context, req retrieval.RetrievalRequest) ([]retrieval.ContextBlock, error) {
	return nil, fmt.Errorf("retrieval unavailable")
}

// TestNewRetrievalEvaluator_DefaultModes 覆盖空 modes 时回退到默认 ["hybrid"] 的分支。
func TestNewRetrievalEvaluator_DefaultModes(t *testing.T) {
	runner := &stubRetrievalRunner{
		blocksByQueryMode: map[string]map[string][]retrieval.ContextBlock{
			"q1": {
				"hybrid": {{Symbol: retrieval.ContextSymbol{Name: "Target"}}},
			},
		},
	}
	truths := []RetrievalGroundTruth{
		{Query: "q1", RelevantSymbols: []string{"Target"}},
	}
	// 传入空 modes，应回退为 ["hybrid"]
	eval := NewRetrievalEvaluator(runner, truths, nil)

	metrics, err := eval.Evaluate(context.Background(), []string{"repo-1"})
	require.NoError(t, err)
	// 回退后只跑了 hybrid，不应有 mode_compare（mode 数 <=1）
	for _, m := range metrics {
		assert.NotContains(t, m.Name, "mode_compare", "单 mode 不应产生 mode_compare")
	}
	// 应有 hybrid 指标
	hasHybrid := false
	for _, m := range metrics {
		if m.Name == "recall@10_hybrid" {
			hasHybrid = true
		}
	}
	assert.True(t, hasHybrid, "回退 hybrid 后应产生 hybrid 指标")
}

// TestRetrievalEvaluator_QueryError 覆盖 runner.Query 出错时的错误包装路径。
func TestRetrievalEvaluator_QueryError(t *testing.T) {
	eval := NewRetrievalEvaluator(&failingRetrievalRunner{}, []RetrievalGroundTruth{
		{Query: "q1", RelevantSymbols: []string{"Target"}},
	}, []string{"hybrid"})

	_, err := eval.Evaluate(context.Background(), []string{"repo-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieval unavailable")
}

// TestRetrievalEvaluator_NeighborHit_CalleesOnly 覆盖 neighbors 计数中的 callees 分支
// （已有测试只命中 callers 分支）。
func TestRetrievalEvaluator_NeighborHit_CalleesOnly(t *testing.T) {
	runner := &stubRetrievalRunner{
		blocksByQueryMode: map[string]map[string][]retrieval.ContextBlock{
			"q1": {
				"hybrid": {
					{
						Symbol:  retrieval.ContextSymbol{Name: "Main"},
						Callees: []retrieval.ContextSymbol{{Name: "CalleeTarget"}},
					},
				},
			},
		},
	}
	truths := []RetrievalGroundTruth{
		{Query: "q1", RelevantSymbols: []string{"Main", "CalleeTarget"}},
	}
	eval := NewRetrievalEvaluator(runner, truths, []string{"hybrid"})

	metrics, err := eval.Evaluate(context.Background(), []string{"repo-1"})
	require.NoError(t, err)

	for _, m := range metrics {
		if m.Name == "neighbor_hit_rate_hybrid" {
			// callees 里有 CalleeTarget，真值相关 2 个，命中 1/2=0.5
			assert.InDelta(t, 0.5, m.Value, 0.001)
			return
		}
	}
	t.Fatal("neighbor_hit_rate_hybrid 指标未找到")
}

// TestAvgFloat64 边界：空切片返回 0（覆盖 len==0 分支）。
func TestAvgFloat64_Empty(t *testing.T) {
	assert.Equal(t, 0.0, avgFloat64(nil))
	assert.Equal(t, 0.0, avgFloat64([]float64{}))
}

func TestAvgFloat64_Average(t *testing.T) {
	// (1.0 + 0.0) / 2 = 0.5
	assert.InDelta(t, 0.5, avgFloat64([]float64{1.0, 0.0}), 0.001)
}
