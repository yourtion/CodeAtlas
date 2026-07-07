package quality

import (
	"context"
	"fmt"

	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

// RetrievalGroundTruth 一个检索 query 的真值。
type RetrievalGroundTruth struct {
	Query           string   // "C++ 如何调用 C 函数"
	RelevantSymbols []string // 真值相关符号名（按 name 匹配）
	RelevantFiles   []string // 真值相关文件（符号名歧义时消歧）
	Repos           []string // 涉及的 repo 标识
}

// RetrievalRunner 收窄 retrieval.Retriever 依赖。
type RetrievalRunner interface {
	Query(ctx context.Context, req retrieval.RetrievalRequest) ([]retrieval.ContextBlock, error)
}

// RetrievalEvaluator 检索评估器。
type RetrievalEvaluator struct {
	runner RetrievalRunner
	truths []RetrievalGroundTruth
	modes  []string // ["hybrid","vector","keyword"]
}

// NewRetrievalEvaluator 构造检索评估器。
func NewRetrievalEvaluator(runner RetrievalRunner, truths []RetrievalGroundTruth, modes []string) *RetrievalEvaluator {
	if len(modes) == 0 {
		modes = []string{"hybrid"}
	}
	return &RetrievalEvaluator{runner: runner, truths: truths, modes: modes}
}

// Evaluate 计算检索指标。对每个 query × mode 跑检索，对真值算 recall/MRR/neighbor_hit。
func (e *RetrievalEvaluator) Evaluate(ctx context.Context, repoIDs []string) ([]MetricValue, error) {
	var metrics []MetricValue

	// 按 mode 累积 recall，用于 mode_compare
	recallByMode := map[string][]float64{} // mode -> 每个 query 的 recall

	for _, truth := range e.truths {
		relevantSet := make(map[string]bool)
		for _, s := range truth.RelevantSymbols {
			relevantSet[s] = true
		}

		for _, mode := range e.modes {
			req := retrieval.RetrievalRequest{
				Query:         truth.Query,
				RepoIDs:       repoIDs,
				Mode:          mode,
				Limit:         10,
				ExpandCallers: true,
				ExpandCallees: true,
			}
			blocks, err := e.runner.Query(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("query %q mode %s: %w", truth.Query, mode, err)
			}

			// recall@k
			hit := 0
			for _, b := range blocks {
				if relevantSet[b.Symbol.Name] {
					hit++
				}
			}
			recall := 0.0
			if len(relevantSet) > 0 {
				recall = float64(hit) / float64(len(relevantSet))
			}
			recallByMode[mode] = append(recallByMode[mode], recall)

			// MRR：第一个相关符号的排名倒数
			mrr := 0.0
			for i, b := range blocks {
				if relevantSet[b.Symbol.Name] {
					mrr = 1.0 / float64(i+1)
					break
				}
			}

			// neighbor_hit_rate：邻居里含真值相关符号的比例
			neighborHit := 0
			for _, b := range blocks {
				for _, c := range b.Callers {
					if relevantSet[c.Name] {
						neighborHit++
					}
				}
				for _, c := range b.Callees {
					if relevantSet[c.Name] {
						neighborHit++
					}
				}
			}
			neighborRate := 0.0
			if len(relevantSet) > 0 {
				neighborRate = float64(neighborHit) / float64(len(relevantSet))
			}

			metrics = append(metrics, metricForMode("recall@10", mode, recall, ThresholdRecallAtK, true))
			metrics = append(metrics, metricForMode("MRR", mode, mrr, ThresholdMRR, true))
			metrics = append(metrics, metricForMode("neighbor_hit_rate", mode, neighborRate, ThresholdNeighborHitRate, true))
		}
	}

	// mode_compare（仅当有多个 mode 时）
	if len(e.modes) > 1 {
		for i := 0; i < len(e.modes); i++ {
			for j := i + 1; j < len(e.modes); j++ {
				m1, m2 := e.modes[i], e.modes[j]
				r1 := avgFloat64(recallByMode[m1])
				r2 := avgFloat64(recallByMode[m2])
				mv := MetricValue{
					Name:           fmt.Sprintf("mode_compare_%s_vs_%s", m1, m2),
					Category:       CategoryRetrieval,
					Value:          r1 - r2,
					Threshold:      0, // 仅观察
					HigherIsBetter: true,
				}
				mv.EvaluatePassed()
				metrics = append(metrics, mv)
			}
		}
	}

	return metrics, nil
}

// metricForMode 构造带 mode 后缀的指标值。
func metricForMode(name, mode string, value, threshold float64, higher bool) MetricValue {
	mv := MetricValue{
		Name:           name + "_" + mode,
		Category:       CategoryRetrieval,
		Value:          value,
		Threshold:      threshold,
		HigherIsBetter: higher,
	}
	mv.EvaluatePassed()
	return mv
}

func avgFloat64(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}
