// Package quality 提供代码知识图谱与检索的质量评估能力。
//
// 评估分两类指标：
//   - 结构断言类（无需真值）：悬空边率、符号消解率、孤立符号率等
//   - fixture 真值类（需标注）：边召回率/准确率、检索 recall@k/MRR 等
//
// 评估器通过接口收窄 models/retrieval 依赖，便于单测 mock。
// CLI（codeatlas eval）和集成测试共用 Evaluate 顶层入口。
package quality

import "encoding/json"

// EvalMode 区分评估模式
type EvalMode string

const (
	EvalModeFixture EvalMode = "fixture" // 跑真值类指标（recall/precision/MRR）
	EvalModeRepo    EvalMode = "repo"    // 只跑结构断言类指标（真实仓库无真值）
)

// MetricCategory 指标大类
type MetricCategory string

const (
	CategoryGraph     MetricCategory = "graph"     // 依赖图指标
	CategoryRetrieval MetricCategory = "retrieval" // 检索指标
)

// MetricValue 单个指标的值（支持分桶）。
type MetricValue struct {
	Name           string             `json:"name"`
	Category       MetricCategory     `json:"category"`
	Value          float64            `json:"value"`
	Threshold      float64            `json:"threshold"`        // 0 = 仅观察无阈值
	HigherIsBetter bool               `json:"higher_is_better"` // true: Value≥Threshold 达标；false: Value≤Threshold 达标
	Passed         bool               `json:"passed"`
	Bucket         string             `json:"bucket,omitempty"` // 分桶标签，如 "import"/"call"；空 = 总值
	Detail         map[string]float64 `json:"detail,omitempty"` // 子分桶明细
}

// EvaluatePassed 根据 Threshold/HigherIsBetter/Value 计算 Passed。
// 无阈值（Threshold==0）时恒 true。
func (m *MetricValue) EvaluatePassed() {
	if m.Threshold == 0 {
		m.Passed = true
		return
	}
	if m.HigherIsBetter {
		m.Passed = m.Value >= m.Threshold
	} else {
		m.Passed = m.Value <= m.Threshold
	}
}

// Report 评估报告（eval CLI 和集成测试共用）。
type Report struct {
	Mode       EvalMode      `json:"mode"`
	RepoID     string        `json:"repo_id,omitempty"`
	FixtureSet string        `json:"fixture_set,omitempty"`
	Metrics    []MetricValue `json:"metrics"`
	Summary    Summary       `json:"summary"`
}

// Summary 指标通过情况汇总。
type Summary struct {
	Total       int `json:"total"`
	Passed      int `json:"passed"`
	Failed      int `json:"failed"`
	NoThreshold int `json:"no_threshold"`
}

// ComputeSummary 从 metrics 列表计算 Summary。
func ComputeSummary(metrics []MetricValue) Summary {
	s := Summary{Total: len(metrics)}
	for _, m := range metrics {
		if m.Threshold == 0 {
			s.NoThreshold++
		}
		if m.Passed {
			s.Passed++
		} else {
			s.Failed++
		}
	}
	return s
}

// OverrideThreshold 覆盖指定指标的阈值并重新计算 Passed。
// 用于集成测试按 fixture 放宽阈值。只匹配 Bucket=="" 的总值；
// 分桶阈值（如 dangling_edge_ratio 的 call/reference 桶）不受影响。
func (r *Report) OverrideThreshold(name string, newThreshold float64) {
	for i := range r.Metrics {
		if r.Metrics[i].Name == name && r.Metrics[i].Bucket == "" {
			r.Metrics[i].Threshold = newThreshold
			r.Metrics[i].EvaluatePassed()
		}
	}
	r.Summary = ComputeSummary(r.Metrics)
}

// --- 阈值常量（初定，跑出基线后调整） ---

// 结构断言类硬门禁阈值。
// 总值（不分桶）保持 Threshold=0（仅观察），因 import 边 100% 悬空会拖低总值。
// 分桶阈值只对 call/reference 类设硬门禁；import/extends 类悬空符合预期，保持观察。
//
// 阈值依据：2026-07-08 基线值 ± 安全边际（见 baselines/2026-07-08-precision-graph-v2-baseline.md）。
const (
	ThresholdDanglingEdgeRatioCall      = 0.90 // 基线 0.84，留 6% 安全边际
	ThresholdDanglingEdgeRatioReference = 0.80 // 基线 0.67，留 13% 安全边际
	ThresholdSymbolResolutionCall       = 0.10 // 基线 0.16（1-0.84），留 6% 安全边际
	ThresholdOrphanSymbolRatio          = 0.40 // 基线 0.33，留 7% 安全边际
	ThresholdCrossFileConnectivity      = 0.10 // 基线 0.12，留 2% 安全边际
)

// fixture 真值类（硬门禁）
const (
	ThresholdEdgeRecall            = 0.90
	ThresholdEdgePrecision         = 0.85
	ThresholdCallChainConnectivity = 0.95
	ThresholdRecallAtK             = 0.70
	ThresholdMRR                   = 0.50
	ThresholdNeighborHitRate       = 0.60
)

// JSONMarshal 序列化报告为 JSON（CLI --format json 用）。
func (r *Report) JSONMarshal() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
