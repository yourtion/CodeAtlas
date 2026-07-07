package quality

import (
	"context"
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
