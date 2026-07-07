package quality

import (
	"context"
	"fmt"
)

// EvaluateConfig 一次评估的配置。
type EvaluateConfig struct {
	Mode           EvalMode
	RepoID         string   // repo 模式必填
	RepoIDs        []string // fixture 模式下检索评估的多 repo
	FixtureSet     string   // fixture 模式标识
	RunRetrieval   bool     // 是否跑检索评估（repo 模式默认 false）
	RetrievalModes []string // 默认 ["hybrid","vector","keyword"]
}

// Evaluate 顶层入口，CLI 和集成测试共用。
// graphEval 必填；retrievalEval 在 RunRetrieval=true 时必填。
func Evaluate(ctx context.Context, cfg EvaluateConfig, graphEval *GraphEvaluator, retrievalEval *RetrievalEvaluator) (*Report, error) {
	report := &Report{
		Mode:       cfg.Mode,
		RepoID:     cfg.RepoID,
		FixtureSet: cfg.FixtureSet,
	}

	// 1. 依赖图评估
	if graphEval != nil {
		graphMetrics, err := graphEval.Evaluate(ctx, cfg.RepoID, cfg.Mode)
		if err != nil {
			return nil, fmt.Errorf("graph evaluation: %w", err)
		}
		report.Metrics = append(report.Metrics, graphMetrics...)
	}

	// 2. 检索评估（repo 模式默认跳过）
	if cfg.RunRetrieval && retrievalEval != nil {
		repoIDs := cfg.RepoIDs
		if cfg.RepoID != "" && len(repoIDs) == 0 {
			repoIDs = []string{cfg.RepoID}
		}
		retrievalMetrics, err := retrievalEval.Evaluate(ctx, repoIDs)
		if err != nil {
			return nil, fmt.Errorf("retrieval evaluation: %w", err)
		}
		report.Metrics = append(report.Metrics, retrievalMetrics...)
	}

	report.Summary = ComputeSummary(report.Metrics)
	return report, nil
}
