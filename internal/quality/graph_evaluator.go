package quality

import (
	"context"
	"fmt"
)

// ExpectedEdge 真值里的一条期望边。
// SourceID/TargetID 用于 symbol_id 精确匹配（测试里索引后回填）；
// SourceName/TargetName 保留供人类可读调试。
type ExpectedEdge struct {
	SourceID   string // 索引后回填（GenerateDeterministicUUID 产出）
	SourceName string // "CppClass::CppMethod"
	EdgeType   string // "call"
	TargetID   string // 索引后回填
	TargetName string // "c_init"
	Optional   bool   // true = 提到了不算漏（如标准库 strlen）
}

// ExpectedChain 真值里的一条调用链（用于 call_chain_connectivity 指标）。
type ExpectedChain struct {
	StartName string // 链路起点符号名
	EndName   string // 链路终点符号名
	StartFile string // 起点所在文件（消歧）
	EndFile   string // 终点所在文件（消歧）
}

// GraphGroundTruth 一个 fixture 的依赖图真值。
type GraphGroundTruth struct {
	FixtureFile string // "tests/fixtures/cpp/cpp_calls_c.cpp"
	Edges       []ExpectedEdge
	Chains      []ExpectedChain
}

// ExtractedEdge 从 DB 查出的提取边（用于真值匹配）。
// SourceID/TargetID 用于 symbol_id 精确匹配（解决 C++ 重载同名问题）；
// SourceName/TargetName 保留供调试日志。
type ExtractedEdge struct {
	SourceID   string
	SourceName string
	EdgeType   string
	TargetID   string // 悬空时为空
	TargetName string
}

// GraphDataFetcher 收窄 models 依赖，便于 mock。
type GraphDataFetcher interface {
	CountEdgesByType(ctx context.Context, repoID string) (map[string]int, error)
	CountDanglingEdges(ctx context.Context, repoID string) (map[string]int, error)
	CountOrphanSymbols(ctx context.Context, repoID string) (int, error)
	CountCrossFileEdges(ctx context.Context, repoID string) (int, error)
	CountTotalSymbols(ctx context.Context, repoID string) (int, error)
	CheckCallChainConnectivity(ctx context.Context, repoID string, chains []ExpectedChain) (int, int, error)
	// ListExtractedEdges 返回仓库内所有提取出的边（用于 edge_recall/precision 对真值）。
	// 只返回 source_name/edge_type/target_name 三元组（target_name 可能为空=悬空）。
	ListExtractedEdges(ctx context.Context, repoID string) ([]ExtractedEdge, error)
}

// GraphEvaluator 依赖图评估器。
type GraphEvaluator struct {
	fetcher GraphDataFetcher
	truth   *GraphGroundTruth // fixture 模式非空；repo 模式 nil
}

// NewGraphEvaluator 构造依赖图评估器。
func NewGraphEvaluator(fetcher GraphDataFetcher, truth *GraphGroundTruth) *GraphEvaluator {
	return &GraphEvaluator{fetcher: fetcher, truth: truth}
}

// Evaluate 计算依赖图指标。repo 模式只跑结构断言类；fixture 模式额外跑真值类。
func (e *GraphEvaluator) Evaluate(ctx context.Context, repoID string, mode EvalMode) ([]MetricValue, error) {
	var metrics []MetricValue

	// --- 结构断言类（两种模式都跑） ---
	byType, err := e.fetcher.CountEdgesByType(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("CountEdgesByType: %w", err)
	}
	dangling, err := e.fetcher.CountDanglingEdges(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("CountDanglingEdges: %w", err)
	}
	orphans, err := e.fetcher.CountOrphanSymbols(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("CountOrphanSymbols: %w", err)
	}
	crossFile, err := e.fetcher.CountCrossFileEdges(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("CountCrossFileEdges: %w", err)
	}
	totalSymbols, err := e.fetcher.CountTotalSymbols(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("CountTotalSymbols: %w", err)
	}

	totalEdges := 0
	totalDangling := 0
	for _, count := range byType {
		totalEdges += count
	}
	for _, count := range dangling {
		totalDangling += count
	}

	// 结构断言类指标：这轮仅观察、建基线，不做硬门禁（spec §3.4）。
	// Threshold=0 表示无阈值；具体建议值见 metrics.go 的 ThresholdXxx 常量（下一轮启用）。
	//
	// 悬空边率（总值 + 分桶）
	if totalEdges > 0 {
		mv := MetricValue{
			Name: "dangling_edge_ratio", Category: CategoryGraph,
			Value:     float64(totalDangling) / float64(totalEdges),
			Threshold: 0, HigherIsBetter: false, // 仅观察（下一轮启用 ThresholdDanglingEdgeRatio）
		}
		mv.EvaluatePassed()
		metrics = append(metrics, mv)

		// 分桶
		for et := range byType {
			d := dangling[et]
			t := byType[et]
			if t == 0 {
				continue
			}
			bv := MetricValue{
				Name: "dangling_edge_ratio", Category: CategoryGraph,
				Value: float64(d) / float64(t), Bucket: et,
				Threshold: 0, HigherIsBetter: false, // 分桶仅观察
			}
			bv.EvaluatePassed()
			metrics = append(metrics, bv)

			// edge_type_distribution：各 edge_type 占比（仅观察）
			distMV := MetricValue{
				Name: "edge_type_distribution", Category: CategoryGraph,
				Value: float64(t) / float64(totalEdges), Bucket: et,
				Threshold: 0, HigherIsBetter: true,
			}
			distMV.EvaluatePassed()
			metrics = append(metrics, distMV)
		}

		// 符号消解率（1 - 悬空边率）
		res := MetricValue{
			Name: "symbol_resolution_rate", Category: CategoryGraph,
			Value:     1 - float64(totalDangling)/float64(totalEdges),
			Threshold: 0, HigherIsBetter: true, // 仅观察（下一轮启用 ThresholdSymbolResolution）
		}
		res.EvaluatePassed()
		metrics = append(metrics, res)
	}

	// 孤立符号率
	if totalSymbols > 0 {
		mv := MetricValue{
			Name: "orphan_symbol_ratio", Category: CategoryGraph,
			Value:     float64(orphans) / float64(totalSymbols),
			Threshold: 0, HigherIsBetter: false, // 仅观察（下一轮启用 ThresholdOrphanSymbolRatio）
		}
		mv.EvaluatePassed()
		metrics = append(metrics, mv)
	}

	// 跨文件连接率
	if totalEdges > 0 {
		mv := MetricValue{
			Name: "cross_file_connectivity", Category: CategoryGraph,
			Value:     float64(crossFile) / float64(totalEdges),
			Threshold: 0, HigherIsBetter: true, // 仅观察（下一轮启用 ThresholdCrossFileConnectivity）
		}
		mv.EvaluatePassed()
		metrics = append(metrics, mv)
	}

	// --- fixture 真值类（仅 fixture 模式） ---
	if mode == EvalModeFixture && e.truth != nil {
		// edge_recall / edge_precision：对比提取边与真值边
		if len(e.truth.Edges) > 0 {
			extracted, err := e.fetcher.ListExtractedEdges(ctx, repoID)
			if err != nil {
				return nil, fmt.Errorf("ListExtractedEdges: %w", err)
			}
			recall, precision := computeEdgeMatch(e.truth.Edges, extracted)
			recallMV := MetricValue{
				Name: "edge_recall", Category: CategoryGraph,
				Value: recall, Threshold: ThresholdEdgeRecall, HigherIsBetter: true,
			}
			recallMV.EvaluatePassed()
			metrics = append(metrics, recallMV)

			precMV := MetricValue{
				Name: "edge_precision", Category: CategoryGraph,
				Value: precision, Threshold: ThresholdEdgePrecision, HigherIsBetter: true,
			}
			precMV.EvaluatePassed()
			metrics = append(metrics, precMV)
		}

		// 调用链连通性
		if len(e.truth.Chains) > 0 {
			ok, total, err := e.fetcher.CheckCallChainConnectivity(ctx, repoID, e.truth.Chains)
			if err != nil {
				return nil, fmt.Errorf("CheckCallChainConnectivity: %w", err)
			}
			mv := MetricValue{
				Name: "call_chain_connectivity", Category: CategoryGraph,
				Value:     float64(ok) / float64(total),
				Threshold: ThresholdCallChainConnectivity, HigherIsBetter: true,
			}
			mv.EvaluatePassed()
			metrics = append(metrics, mv)
		}
	}

	return metrics, nil
}

// computeEdgeMatch 计算边召回率和准确率。
// 真值边匹配提取边：按 (source_id, edge_type, target_id) 三元组。
// symbol_id 匹配从根本上解决 C++ 重载同名符号导致 recall>1.0 的问题。
// Optional=true 的真值边不计入漏提取（如标准库函数）。
// 悬空边（TargetID 空）不参与匹配——它们无法对齐到具体符号。
func computeEdgeMatch(truth []ExpectedEdge, extracted []ExtractedEdge) (recall, precision float64) {
	extractedSet := make(map[string]bool)
	for _, e := range extracted {
		if e.TargetID == "" {
			continue // 悬空边不入匹配集
		}
		key := e.SourceID + "|" + e.EdgeType + "|" + e.TargetID
		extractedSet[key] = true
	}

	// recall：真值边中被提取的比例
	required := 0
	hit := 0
	for _, te := range truth {
		if te.Optional {
			continue
		}
		if te.TargetID == "" {
			continue // 真值边无 target_id 也不计入（不应发生，但防御）
		}
		required++
		key := te.SourceID + "|" + te.EdgeType + "|" + te.TargetID
		if extractedSet[key] {
			hit++
		}
	}
	if required > 0 {
		recall = float64(hit) / float64(required)
	}

	// precision：提取边中匹配真值的比例（悬空边不计入分母）
	truthSet := make(map[string]bool)
	for _, te := range truth {
		if te.TargetID == "" {
			continue
		}
		key := te.SourceID + "|" + te.EdgeType + "|" + te.TargetID
		truthSet[key] = true
	}
	matchedNonDangling := 0
	totalNonDangling := 0
	for _, e := range extracted {
		if e.TargetID == "" {
			continue // 悬空边不计入 precision 分母
		}
		totalNonDangling++
		key := e.SourceID + "|" + e.EdgeType + "|" + e.TargetID
		if truthSet[key] {
			matchedNonDangling++
		}
	}
	if totalNonDangling > 0 {
		precision = float64(matchedNonDangling) / float64(totalNonDangling)
	}

	return recall, precision
}
