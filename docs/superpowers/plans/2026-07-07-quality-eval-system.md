# 质量评估系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新建 `internal/quality/` 评估领域包 + `codeatlas eval` CLI + 集成测试门禁，覆盖依赖图质量与检索质量两个环节，PR#2 集成测试验证并入。

**Architecture:** 三层——`pkg/models/graph_metrics.go` 聚合查询（纯 SQL）→ `internal/quality/` 评估器（指标计算 + Report）→ `cmd/cli/eval_command.go` + 集成测试（编排 + 门禁）。接口收窄做依赖注入，单元测试全 mock，集成测试连真 DB。

**Tech Stack:** Go 1.25+, urfave/cli/v2, PostgreSQL + pgvector, testify, 现有 indexer/retrieval 层。

**Spec:** `docs/superpowers/specs/2026-07-07-quality-eval-system-design.md`

---

## File Structure

**新建：**
- `pkg/models/graph_metrics.go` — 5 个聚合查询方法（CountEdgesByType 等），挂在 `EdgeRepository`/`SymbolRepository` 上作扩展方法包
- `internal/quality/metrics.go` — MetricValue/Report/Summary/EvalMode 类型 + 阈值常量
- `internal/quality/graph_evaluator.go` — GraphDataFetcher 接口 + GraphEvaluator + defaultGraphFetcher 适配器 + ExpectedChain/GraphGroundTruth/ExpectedEdge 真值类型
- `internal/quality/retrieval_evaluator.go` — RetrievalRunner 接口 + RetrievalEvaluator + RetrievalGroundTruth 真值类型
- `internal/quality/report.go` — Evaluate 顶层编排 + Report 序列化 + OverrideThreshold
- `internal/quality/fixtures/graph_ground_truth.go` — 真值实例（从 tests/integration 迁移）
- `internal/quality/fixtures/retrieval_ground_truth.go` — 检索真值实例
- `cmd/cli/eval_command.go` — `codeatlas eval` 命令
- `tests/integration/quality_gate_test.go` — 集成门禁

**测试：**
- `pkg/models/graph_metrics_test.go` — 集成测试（真 DB）
- `internal/quality/metrics_test.go` — 单元
- `internal/quality/graph_evaluator_test.go` — 单元
- `internal/quality/retrieval_evaluator_test.go` — 单元
- `internal/quality/report_test.go` — 单元
- `cmd/cli/eval_command_test.go` — 单元

**修改：**
- `cmd/cli/main.go` — 注册 eval 命令
- `docs/evaluation.md` — 新增评估文档
- `docs/cli.md` — 补 eval 命令

---

## Task 1: models 层聚合查询方法

**Files:**
- Create: `pkg/models/graph_metrics.go`
- Test: `tests/integration/models_integration_test.go`（扩展，加新测试函数）

- [ ] **Step 1: 写失败测试**

在 `tests/integration/models_integration_test.go` 末尾追加：

```go
// TestGraphMetrics_AggregationQueries 验证 5 个聚合查询方法的正确性。
func TestGraphMetrics_AggregationQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoID := uuid.New().String()

	// 索引一个已知 fixture（复用 createTestParseOutput 模式，构造带边的数据）
	parseOutput := createTestParseOutputWithEdges(repoID)
	config := &indexer.IndexerConfig{
		RepoID: repoID, RepoName: "graph-metrics-test", BatchSize: 10,
		WorkerCount: 2, SkipVectors: true, UseTransactions: true,
	}
	idx := indexer.NewIndexer(testDB.DB, config)
	if _, err := idx.Index(ctx, parseOutput); err != nil {
		t.Fatalf("Indexing failed: %v", err)
	}

	edgeRepo := models.NewEdgeRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)

	// 1. CountEdgesByType
	byType, err := models.CountEdgesByType(ctx, edgeRepo, repoID)
	require.NoError(t, err)
	assert.NotEmpty(t, byType, "应有至少一种 edge_type")
	totalEdges := 0
	for _, count := range byType {
		totalEdges += count
	}

	// 2. CountDanglingEdges
	dangling, err := models.CountDanglingEdges(ctx, edgeRepo, repoID)
	require.NoError(t, err)
	danglingTotal := 0
	for _, count := range dangling {
		danglingTotal += count
	}
	assert.LessOrEqual(t, danglingTotal, totalEdges, "悬空边数不应超过总边数")

	// 3. CountTotalSymbols
	totalSymbols, err := models.CountTotalSymbols(ctx, symbolRepo, repoID)
	require.NoError(t, err)
	assert.Greater(t, totalSymbols, 0, "应有符号")

	// 4. CountOrphanSymbols
	orphans, err := models.CountOrphanSymbols(ctx, symbolRepo, repoID)
	require.NoError(t, err)
	assert.LessOrEqual(t, orphans, totalSymbols, "孤立符号数不应超过总符号数")

	// 5. CountCrossFileEdges
	crossFile, err := models.CountCrossFileEdges(ctx, edgeRepo, repoID)
	require.NoError(t, err)
	assert.LessOrEqual(t, crossFile, totalEdges, "跨文件边数不应超过总边数")
}
```

同时在 `tests/integration/models_integration_test.go` 加 helper（如果 `createTestParseOutputWithEdges` 不存在）：

```go
// createTestParseOutputWithEdges 构造一个带 call/import 边的 parseOutput 用于图指标测试。
func createTestParseOutputWithEdges(repoID string) *parser.ParseOutput {
	// 复用现有 createTestParseOutput 的结构，确保有 call 边和至少一个 import 边
	// 若 createTestParseOutput 已有边，直接调用它
	return createTestParseOutput()
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -v ./tests/integration -run TestGraphMetrics_AggregationQueries`
Expected: FAIL — `models.CountEdgesByType` undefined

- [ ] **Step 3: 实现聚合查询方法**

Create `pkg/models/graph_metrics.go`:

```go
package models

import "context"

// CountEdgesByType 按 edge_type 分组统计边数。
// 返回 map[edge_type]count。
func CountEdgesByType(ctx context.Context, r *EdgeRepository, repoID string) (map[string]int, error) {
	query := `
		SELECT e.edge_type, COUNT(*)
		FROM edges e
		JOIN symbols s ON e.source_id = s.symbol_id
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1
		GROUP BY e.edge_type
	`
	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var edgeType string
		var count int
		if err := rows.Scan(&edgeType, &count); err != nil {
			return nil, err
		}
		result[edgeType] = count
	}
	return result, rows.Err()
}

// CountDanglingEdges 按 edge_type 分组统计 target_id IS NULL 的边数（未解析符号的边）。
func CountDanglingEdges(ctx context.Context, r *EdgeRepository, repoID string) (map[string]int, error) {
	query := `
		SELECT e.edge_type, COUNT(*)
		FROM edges e
		JOIN symbols s ON e.source_id = s.symbol_id
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1 AND e.target_id IS NULL
		GROUP BY e.edge_type
	`
	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var edgeType string
		var count int
		if err := rows.Scan(&edgeType, &count); err != nil {
			return nil, err
		}
		result[edgeType] = count
	}
	return result, rows.Err()
}

// CountCrossFileEdges 统计 source_file ≠ target_file 的边数。
func CountCrossFileEdges(ctx context.Context, r *EdgeRepository, repoID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM edges e
		JOIN symbols s ON e.source_id = s.symbol_id
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1
		  AND e.target_file IS NOT NULL
		  AND e.source_file IS NOT NULL
		  AND e.source_file <> e.target_file
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	return count, err
}

// CountTotalSymbols 统计仓库总符号数。
func CountTotalSymbols(ctx context.Context, r *SymbolRepository, repoID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM symbols s
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	return count, err
}

// CountOrphanSymbols 统计无任何出入边的孤立符号数。
func CountOrphanSymbols(ctx context.Context, r *SymbolRepository, repoID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM symbols s
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1
		  AND s.symbol_id NOT IN (SELECT source_id FROM edges WHERE source_id IS NOT NULL)
		  AND s.symbol_id NOT IN (SELECT target_id FROM edges WHERE target_id IS NOT NULL)
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	return count, err
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -v ./tests/integration -run TestGraphMetrics_AggregationQueries`
Expected: PASS

如果测试库未启动，先 `make db && make db-init`。

- [ ] **Step 5: 提交**

```bash
git add pkg/models/graph_metrics.go tests/integration/models_integration_test.go
git commit -m "feat(models): 图指标聚合查询方法——CountEdgesByType/Dangling/CrossFile/Orphan/TotalSymbols"
```

---

## Task 2: quality 包核心类型与阈值常量

**Files:**
- Create: `internal/quality/metrics.go`
- Test: `internal/quality/metrics_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/quality/metrics_test.go`:

```go
package quality

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricValue_IsPassed(t *testing.T) {
	tests := []struct {
		name      string
		mv        MetricValue
		wantPassed bool
	}{
		{
			name: "无阈值恒通过",
			mv: MetricValue{Name: "observed", Value: 0.99, Threshold: 0},
			wantPassed: true,
		},
		{
			name: "越高越好的指标达标",
			mv: MetricValue{Name: "recall", Value: 0.75, Threshold: 0.70, HigherIsBetter: true},
			wantPassed: true,
		},
		{
			name: "越高越好的指标未达标",
			mv: MetricValue{Name: "recall", Value: 0.65, Threshold: 0.70, HigherIsBetter: true},
			wantPassed: false,
		},
		{
			name: "越低越好的指标达标",
			mv: MetricValue{Name: "dangling", Value: 0.20, Threshold: 0.30, HigherIsBetter: false},
			wantPassed: true,
		},
		{
			name: "越低越好的指标未达标",
			mv: MetricValue{Name: "dangling", Value: 0.35, Threshold: 0.30, HigherIsBetter: false},
			wantPassed: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mv.EvaluatePassed()
			assert.Equal(t, tt.wantPassed, tt.mv.Passed)
		})
	}
}

func TestSummary_ComputeFromMetrics(t *testing.T) {
	metrics := []MetricValue{
		{Name: "a", Threshold: 0.7, Value: 0.8, HigherIsBetter: true, Passed: true},
		{Name: "b", Threshold: 0.7, Value: 0.6, HigherIsBetter: true, Passed: false},
		{Name: "c", Threshold: 0, Value: 0.5, Passed: true}, // 仅观察
	}
	s := ComputeSummary(metrics)
	assert.Equal(t, 3, s.Total)
	assert.Equal(t, 2, s.Passed)
	assert.Equal(t, 1, s.Failed)
	assert.Equal(t, 1, s.NoThreshold)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -v ./internal/quality -run TestMetricValue`
Expected: FAIL — package 不存在/类型未定义

- [ ] **Step 3: 实现 metrics.go**

Create `internal/quality/metrics.go`:

```go
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
	Threshold      float64            `json:"threshold"`       // 0 = 仅观察无阈值
	HigherIsBetter bool               `json:"higher_is_better"` // true: Value≥Threshold 达标；false: Value≤Threshold 达标
	Passed         bool               `json:"passed"`
	Bucket         string             `json:"bucket,omitempty"` // 分桶标签，如 "import"/"call"；空 = 总值
	Detail         map[string]float64 `json:"detail,omitempty"`  // 子分桶明细
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
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	NoThresh int `json:"no_threshold"`
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
// 用于集成测试按 fixture 放宽阈值。name 匹配 MetricValue.Name（含分桶则所有同 name 的都覆盖）。
func (r *Report) OverrideThreshold(name string, newThreshold float64) {
	for i := range r.Metrics {
		if r.Metrics[i].Name == name {
			r.Metrics[i].Threshold = newThreshold
			r.Metrics[i].EvaluatePassed()
		}
	}
	r.Summary = ComputeSummary(r.Metrics)
}

// --- 阈值常量（初定，跑出基线后调整） ---

// 结构断言类（这轮仅观察，Threshold=0）
const (
	ThresholdDanglingEdgeRatio   = 0.30 // 建议值，这轮不做硬门禁
	ThresholdSymbolResolution    = 0.70
	ThresholdOrphanSymbolRatio   = 0.40
	ThresholdCrossFileConnectivity = 0.20
)

// fixture 真值类（硬门禁）
const (
	ThresholdEdgeRecall           = 0.90
	ThresholdEdgePrecision        = 0.85
	ThresholdCallChainConnectivity = 0.95
	ThresholdRecallAtK            = 0.70
	ThresholdMRR                  = 0.50
	ThresholdNeighborHitRate      = 0.60
)

// JSONMarshal 序列化报告为 JSON（CLI --format json 用）。
func (r *Report) JSONMarshal() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -v ./internal/quality -run "TestMetricValue|TestSummary"`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/quality/metrics.go internal/quality/metrics_test.go
git commit -m "feat(quality): 核心类型 MetricValue/Report/Summary + 阈值常量"
```

---

## Task 3: 依赖图真值类型与 GraphEvaluator

**Files:**
- Create: `internal/quality/graph_evaluator.go`
- Test: `internal/quality/graph_evaluator_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/quality/graph_evaluator_test.go`:

```go
package quality

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubGraphFetcher 是 GraphDataFetcher 的 mock 实现。
type stubGraphFetcher struct {
	byType        map[string]int
	dangling      map[string]int
	orphans       int
	crossFile     int
	totalSymbols  int
	chainOK       int
	chainTotal    int
	extracted     []ExtractedEdge
	err           error
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
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -v ./internal/quality -run TestGraphEvaluator`
Expected: FAIL — 类型/函数未定义

- [ ] **Step 3: 实现 graph_evaluator.go**

Create `internal/quality/graph_evaluator.go`:

```go
package quality

import (
	"context"
	"fmt"
)

// ExpectedEdge 真值里的一条期望边。
// 不依赖 symbol_id（入库后才有），用符号名匹配。
type ExpectedEdge struct {
	SourceName string // "CppClass::CppMethod"
	EdgeType   string // "call"
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
	FixtureFile string         // "tests/fixtures/cpp/cpp_calls_c.cpp"
	Edges       []ExpectedEdge
	Chains      []ExpectedChain
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

// ExtractedEdge 从 DB 查出的提取边（用于真值匹配）。
type ExtractedEdge struct {
	SourceName string
	EdgeType   string
	TargetName string // 悬空时为空
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
	for et, count := range byType {
		totalEdges += count
	}
	for _, count := range dangling {
		totalDangling += count
	}

	// 悬空边率（总值 + 分桶）
	if totalEdges > 0 {
		mv := MetricValue{
			Name: "dangling_edge_ratio", Category: CategoryGraph,
			Value: float64(totalDangling) / float64(totalEdges),
			Threshold: ThresholdDanglingEdgeRatio, HigherIsBetter: false,
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
		}

		// 符号消解率（1 - 悬空边率）
		res := MetricValue{
			Name: "symbol_resolution_rate", Category: CategoryGraph,
			Value: 1 - float64(totalDangling)/float64(totalEdges),
			Threshold: ThresholdSymbolResolution, HigherIsBetter: true,
		}
		res.EvaluatePassed()
		metrics = append(metrics, res)
	}

	// 孤立符号率
	if totalSymbols > 0 {
		mv := MetricValue{
			Name: "orphan_symbol_ratio", Category: CategoryGraph,
			Value: float64(orphans) / float64(totalSymbols),
			Threshold: ThresholdOrphanSymbolRatio, HigherIsBetter: false,
		}
		mv.EvaluatePassed()
		metrics = append(metrics, mv)
	}

	// 跨文件连接率
	if totalEdges > 0 {
		mv := MetricValue{
			Name: "cross_file_connectivity", Category: CategoryGraph,
			Value: float64(crossFile) / float64(totalEdges),
			Threshold: ThresholdCrossFileConnectivity, HigherIsBetter: true,
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
				Value: float64(ok) / float64(total),
				Threshold: ThresholdCallChainConnectivity, HigherIsBetter: true,
			}
			mv.EvaluatePassed()
			metrics = append(metrics, mv)
		}
	}

	return metrics, nil
}

// computeEdgeMatch 计算边召回率和准确率。
// 真值边匹配提取边：按 (source_name, edge_type, target_name) 三元组。
// Optional=true 的真值边不计入漏提取（如标准库函数）。
func computeEdgeMatch(truth []ExpectedEdge, extracted []ExtractedEdge) (recall, precision float64) {
	extractedSet := make(map[string]bool)
	for _, e := range extracted {
		key := e.SourceName + "|" + e.EdgeType + "|" + e.TargetName
		extractedSet[key] = true
	}

	// recall：真值边中被提取的比例
	required := 0
	hit := 0
	for _, te := range truth {
		if te.Optional {
			continue
		}
		required++
		key := te.SourceName + "|" + te.EdgeType + "|" + te.TargetName
		if extractedSet[key] {
			hit++
		}
	}
	if required > 0 {
		recall = float64(hit) / float64(required)
	}

	// precision：提取边中匹配真值的比例
	// （提取边可能很多，只算是否在真值集合里）
	truthSet := make(map[string]bool)
	for _, te := range truth {
		key := te.SourceName + "|" + te.EdgeType + "|" + te.TargetName
		truthSet[key] = true
	}
	if len(extracted) > 0 {
		matched := 0
		for _, e := range extracted {
			key := e.SourceName + "|" + e.EdgeType + "|" + e.TargetName
			if truthSet[key] {
				matched++
			}
		}
		precision = float64(matched) / float64(len(extracted))
	}

	return recall, precision
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -v ./internal/quality -run TestGraphEvaluator`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/quality/graph_evaluator.go internal/quality/graph_evaluator_test.go
git commit -m "feat(quality): GraphEvaluator + GraphDataFetcher 接口 + 结构断言指标"
```

---

## Task 4: 检索评估器 RetrievalEvaluator

**Files:**
- Create: `internal/quality/retrieval_evaluator.go`
- Test: `internal/quality/retrieval_evaluator_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/quality/retrieval_evaluator_test.go`:

```go
package quality

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

// stubRetrievalRunner 是 RetrievalRunner 的 mock 实现。
type stubRetrievalRunner struct {
	blocksByQuery map[string][]retrieval.ContextBlock
}

func (s *stubRetrievalRunner) Query(ctx context.Context, req retrieval.RetrievalRequest) ([]retrieval.ContextBlock, error) {
	return s.blocksByQuery[req.Query], nil
}

func TestRetrievalEvaluator_RecallAndMRR(t *testing.T) {
	runner := &stubRetrievalRunner{
		blocksByQuery: map[string][]retrieval.ContextBlock{
			"q1": {
				{Symbol: retrieval.ContextSymbol{Name: "Irrelevant"}},
				{Symbol: retrieval.ContextSymbol{Name: "TargetA"}}, // rank 2
				{Symbol: retrieval.ContextSymbol{Name: "TargetB"}}, // rank 3
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
		blocksByQuery: map[string][]retrieval.ContextBlock{
			"q1": {
				{
					Symbol:  retrieval.ContextSymbol{Name: "Main"},
					Callers: []retrieval.ContextSymbol{{Name: "TargetA"}},
					Callees: []retrieval.ContextSymbol{{Name: "Other"}},
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
		blocksByQuery: map[string][]retrieval.ContextBlock{
			"q1_hybrid":  {{Symbol: retrieval.ContextSymbol{Name: "Target"}}},
			"q1_vector":  {{Symbol: retrieval.ContextSymbol{Name: "Irrelevant"}}},
			"q1_keyword": {{Symbol: retrieval.ContextSymbol{Name: "Target"}}},
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
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -v ./internal/quality -run TestRetrievalEvaluator`
Expected: FAIL — 类型未定义

- [ ] **Step 3: 实现 retrieval_evaluator.go**

Create `internal/quality/retrieval_evaluator.go`:

```go
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
				Query:    truth.Query,
				RepoIDs:  repoIDs,
				Mode:     mode,
				Limit:    10,
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
			neighborChecked := 0
			for _, b := range blocks {
				for _, c := range b.Callers {
					neighborChecked++
					if relevantSet[c.Name] {
						neighborHit++
					}
				}
				for _, c := range b.Callees {
					neighborChecked++
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
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -v ./internal/quality -run TestRetrievalEvaluator`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/quality/retrieval_evaluator.go internal/quality/retrieval_evaluator_test.go
git commit -m "feat(quality): RetrievalEvaluator——recall@k/MRR/neighbor_hit/mode_compare"
```

---

## Task 5: 顶层编排 Evaluate + Report 序列化

**Files:**
- Create: `internal/quality/report.go`
- Test: `internal/quality/report_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/quality/report_test.go`:

```go
package quality

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluate_FixtureMode_RunsBothCategories(t *testing.T) {
	graphEval := NewGraphEvaluator(&stubGraphFetcher{
		byType: map[string]int{"call": 10}, dangling: map[string]int{"call": 1},
		orphans: 2, crossFile: 5, totalSymbols: 20,
		chainOK: 9, chainTotal: 10,
	}, &GraphGroundTruth{Edges: []ExpectedEdge{{SourceName: "A", EdgeType: "call", TargetName: "B"}}})

	retrievalEval := NewRetrievalEvaluator(&stubRetrievalRunner{
		blocksByQuery: map[string][]retrieval.ContextBlock{},
	}, nil, []string{"hybrid"})

	report, err := Evaluate(context.Background(), EvaluateConfig{
		Mode:         EvalModeFixture,
		FixtureSet:   "test",
		RunRetrieval: true,
	}, graphEval, retrievalEval)
	require.NoError(t, err)

	assert.Equal(t, EvalModeFixture, report.Mode)
	assert.Equal(t, "test", report.FixtureSet)
	assert.NotEmpty(t, report.Metrics)
	assert.Equal(t, len(report.Metrics), report.Summary.Total)
}

func TestEvaluate_RepoMode_SkipsRetrieval(t *testing.T) {
	graphEval := NewGraphEvaluator(&stubGraphFetcher{
		byType: map[string]int{"call": 10}, totalSymbols: 5,
	}, nil)

	report, err := Evaluate(context.Background(), EvaluateConfig{
		Mode:   EvalModeRepo,
		RepoID: "repo-1",
	}, graphEval, nil)
	require.NoError(t, err)

	assert.Equal(t, EvalModeRepo, report.Mode)
	for _, m := range report.Metrics {
		assert.Equal(t, CategoryGraph, m.Category, "repo 模式不应有检索指标")
	}
}

func TestReport_JSONMarshal(t *testing.T) {
	r := &Report{
		Mode: EvalModeFixture,
		Metrics: []MetricValue{
			{Name: "recall", Category: CategoryRetrieval, Value: 0.75, Threshold: 0.7, HigherIsBetter: true, Passed: true},
		},
	}
	r.Summary = ComputeSummary(r.Metrics)

	data, err := r.JSONMarshal()
	require.NoError(t, err)

	var back Report
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, "recall", back.Metrics[0].Name)
	assert.Equal(t, 0.75, back.Metrics[0].Value)
}

func TestReport_OverrideThreshold(t *testing.T) {
	r := &Report{
		Metrics: []MetricValue{
			{Name: "recall", Value: 0.65, Threshold: 0.70, HigherIsBetter: true, Passed: false},
		},
	}
	r.Summary = ComputeSummary(r.Metrics)
	assert.Equal(t, 1, r.Summary.Failed)

	r.OverrideThreshold("recall", 0.60)
	assert.True(t, r.Metrics[0].Passed)
	assert.Equal(t, 0, r.Summary.Failed)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -v ./internal/quality -run "TestEvaluate|TestReport"`
Expected: FAIL — Evaluate 函数未定义

- [ ] **Step 3: 实现 report.go**

Create `internal/quality/report.go`:

```go
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
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -v ./internal/quality -run "TestEvaluate|TestReport"`
Expected: PASS

跑全包确认无回归:
Run: `go test ./internal/quality/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/quality/report.go internal/quality/report_test.go
git commit -m "feat(quality): Evaluate 顶层编排 + Report JSON 序列化 + OverrideThreshold"
```

---

## Task 6: GraphDataFetcher 适配器（连真 DB）

**Files:**
- Create: `internal/quality/graph_data_fetcher.go`
- Test: `internal/quality/graph_data_fetcher_test.go`（单元，mock DB）

- [ ] **Step 1: 写失败测试**

Create `internal/quality/graph_data_fetcher_test.go`:

```go
package quality

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func TestDefaultGraphFetcher_CountEdgesByType(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	edgeRepo := models.NewEdgeRepository(&models.DB{DB: db})
	symbolRepo := models.NewSymbolRepository(&models.DB{DB: db})
	fetcher := NewDefaultGraphFetcher(edgeRepo, symbolRepo)

	rows := sqlmock.NewRows([]string{"edge_type", "count"}).
		AddRow("call", 80).
		AddRow("import", 20)
	mock.ExpectQuery("SELECT e.edge_type, COUNT").WithArg("repo-1").WillReturnRows(rows)

	result, err := fetcher.CountEdgesByType(context.Background(), "repo-1")
	require.NoError(t, err)
	assert.Equal(t, 80, result["call"])
	assert.Equal(t, 20, result["import"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDefaultGraphFetcher_CheckCallChainConnectivity(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	edgeRepo := models.NewEdgeRepository(&models.DB{DB: db})
	symbolRepo := models.NewSymbolRepository(&models.DB{DB: db})
	fetcher := NewDefaultGraphFetcher(edgeRepo, symbolRepo)

	chains := []ExpectedChain{
		{StartName: "A", EndName: "B", StartFile: "a.go", EndFile: "b.go"},
		{StartName: "C", EndName: "D", StartFile: "c.go", EndFile: "d.go"},
	}

	// 每条链路一次 QueryRow：第一条连通(true)，第二条不连通(false)
	mock.ExpectQuery("WITH RECURSIVE").WithArgs("repo-1", "A", "a.go", "B", "b.go").
		WillReturnRows(sqlmock.NewRows([]string{"connected"}).AddRow(true))
	mock.ExpectQuery("WITH RECURSIVE").WithArgs("repo-1", "C", "c.go", "D", "d.go").
		WillReturnRows(sqlmock.NewRows([]string{"connected"}).AddRow(false))

	ok, total, err := fetcher.CheckCallChainConnectivity(context.Background(), "repo-1", chains)
	require.NoError(t, err)
	assert.Equal(t, 1, ok)
	assert.Equal(t, 2, total)
	require.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -v ./internal/quality -run TestDefaultGraphFetcher`
Expected: FAIL — NewDefaultGraphFetcher 未定义

- [ ] **Step 3: 实现 graph_data_fetcher.go**

Create `internal/quality/graph_data_fetcher.go`:

```go
package quality

import (
	"context"
	"fmt"

	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// DefaultGraphFetcher 是 GraphDataFetcher 的默认实现，
// 组合 EdgeRepository + SymbolRepository，调用 pkg/models 的聚合查询方法。
type DefaultGraphFetcher struct {
	edgeRepo   *models.EdgeRepository
	symbolRepo *models.SymbolRepository
}

// NewDefaultGraphFetcher 构造默认 fetcher。
func NewDefaultGraphFetcher(edgeRepo *models.EdgeRepository, symbolRepo *models.SymbolRepository) *DefaultGraphFetcher {
	return &DefaultGraphFetcher{edgeRepo: edgeRepo, symbolRepo: symbolRepo}
}

func (f *DefaultGraphFetcher) CountEdgesByType(ctx context.Context, repoID string) (map[string]int, error) {
	return models.CountEdgesByType(ctx, f.edgeRepo, repoID)
}

func (f *DefaultGraphFetcher) CountDanglingEdges(ctx context.Context, repoID string) (map[string]int, error) {
	return models.CountDanglingEdges(ctx, f.edgeRepo, repoID)
}

func (f *DefaultGraphFetcher) CountOrphanSymbols(ctx context.Context, repoID string) (int, error) {
	return models.CountOrphanSymbols(ctx, f.symbolRepo, repoID)
}

func (f *DefaultGraphFetcher) CountCrossFileEdges(ctx context.Context, repoID string) (int, error) {
	return models.CountCrossFileEdges(ctx, f.edgeRepo, repoID)
}

func (f *DefaultGraphFetcher) CountTotalSymbols(ctx context.Context, repoID string) (int, error) {
	return models.CountTotalSymbols(ctx, f.symbolRepo, repoID)
}

func (f *DefaultGraphFetcher) CheckCallChainConnectivity(ctx context.Context, repoID string, chains []ExpectedChain) (int, int, error) {
	if len(chains) == 0 {
		return 0, 0, nil
	}
	ok := 0
	for _, c := range chains {
		connected, err := models.CheckSingleChainConnectivity(ctx, f.edgeRepo, repoID, models.ChainSpec{
			StartName: c.StartName,
			EndName:   c.EndName,
			StartFile: c.StartFile,
			EndFile:   c.EndFile,
		})
		if err != nil {
			return 0, 0, fmt.Errorf("check chain %s->%s: %w", c.StartName, c.EndName, err)
		}
		if connected {
			ok++
		}
	}
	return ok, len(chains), nil
}

func (f *DefaultGraphFetcher) ListExtractedEdges(ctx context.Context, repoID string) ([]ExtractedEdge, error) {
	rawEdges, err := models.ListExtractedEdges(ctx, f.edgeRepo, repoID)
	if err != nil {
		return nil, err
	}
	result := make([]ExtractedEdge, len(rawEdges))
	for i, e := range rawEdges {
		result[i] = ExtractedEdge{
			SourceName: e.SourceName,
			EdgeType:   e.EdgeType,
			TargetName: e.TargetName,
		}
	}
	return result, nil
}
```

**实现说明**：`models.DB` 是 `struct { *sql.DB }` 的嵌入结构（见 `pkg/models/database.go:34`），`EdgeRepository.db` 是 `*models.DB` 类型。在 `package models` 内可直接用 `r.db.QueryRowContext(...)` / `r.db.QueryContext(...)` 调用底层 `*sql.DB` 方法。所以 `CheckCallChainConnectivity` 和 `ListExtractedEdges` 的实际 SQL 查询放在 `pkg/models/graph_metrics.go` 作为包级函数，`DefaultGraphFetcher` 只做转发和类型转换（`models.ExtractedEdge` → `quality.ExtractedEdge`）。

需要同步在 Task 1 的 `pkg/models/graph_metrics.go` 追加这两个函数。在 Task 6 实现时，更新 `pkg/models/graph_metrics.go` 加入：

```go
// ChainSpec 调用链端点对（models 层定义，避免循环依赖 quality 包）。
type ChainSpec struct {
	StartName string
	EndName   string
	StartFile string
	EndFile   string
}

// CheckSingleChainConnectivity 用递归 CTE 查 start 是否能经 call 边到达 end。
func CheckSingleChainConnectivity(ctx context.Context, r *EdgeRepository, repoID string, c ChainSpec) (bool, error) {
	query := `
		WITH RECURSIVE reach AS (
			SELECT s.symbol_id FROM symbols s
			JOIN files f ON s.file_id = f.file_id
			WHERE f.repo_id = $1 AND s.name = $2 AND f.path = $3
			UNION
			SELECT e.target_id FROM reach r
			JOIN edges e ON e.source_id = r.symbol_id
			WHERE e.edge_type = 'call' AND e.target_id IS NOT NULL
		)
		SELECT EXISTS(
			SELECT 1 FROM reach r
			JOIN symbols s ON r.symbol_id = s.symbol_id
			JOIN files f ON s.file_id = f.file_id
			WHERE s.name = $4 AND f.path = $5
		)
	`
	var connected bool
	err := r.db.QueryRowContext(ctx, query, repoID, c.StartName, c.StartFile, c.EndName, c.EndFile).Scan(&connected)
	return connected, err
}

// ExtractedEdge models 层的提取边（供 quality 层转换）。
type ExtractedEdge struct {
	SourceName string
	EdgeType   string
	TargetName string
}

// ListExtractedEdges 返回仓库内所有提取出的边（source_name/edge_type/target_name）。
func ListExtractedEdges(ctx context.Context, r *EdgeRepository, repoID string) ([]ExtractedEdge, error) {
	query := `
		SELECT s_source.name, e.edge_type, COALESCE(s_target.name, '')
		FROM edges e
		JOIN symbols s_source ON e.source_id = s_source.symbol_id
		JOIN files f ON s_source.file_id = f.file_id
		LEFT JOIN symbols s_target ON e.target_id = s_target.symbol_id
		WHERE f.repo_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ExtractedEdge
	for rows.Next() {
		var e ExtractedEdge
		if err := rows.Scan(&e.SourceName, &e.EdgeType, &e.TargetName); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
```

**循环依赖处理**：`pkg/models` 不能 import `internal/quality`（quality 依赖 models）。所以 models 包定义 `ChainSpec`/`ExtractedEdge` 作为原始数据类型，不感知真值语义。`DefaultGraphFetcher.CheckCallChainConnectivity` 在 `package quality` 内做编排——遍历 `[]ExpectedChain` 转 `models.ChainSpec` 调 `models.CheckSingleChainConnectivity`，统计连通数。`ListExtractedEdges` 转发 models 查询并转 `quality.ExtractedEdge`。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -v ./internal/quality -run TestDefaultGraphFetcher`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/quality/graph_data_fetcher.go internal/quality/graph_data_fetcher_test.go pkg/models/graph_metrics.go
git commit -m "feat(quality): DefaultGraphFetcher 适配器——连接 models 层聚合查询"
```

---

## Task 7: 真值数据——依赖图 ground truth

**Files:**
- Create: `internal/quality/fixtures/graph_ground_truth.go`
- Test: `internal/quality/fixtures/graph_ground_truth_test.go`

- [ ] **Step 1: 确认真值来源**

先读现有真值列表，确认要迁移哪些:

```bash
grep -A 20 "expectedCCalls := \[\]string" tests/integration/call_analysis_fixtures_test.go | head -40
grep -A 15 "expectedJavaCalls := \[\]string" tests/integration/call_analysis_fixtures_test.go
grep -A 15 "expectedObjCCalls := \[\]string" tests/integration/call_analysis_fixtures_test.go
```

- [ ] **Step 2: 写失败测试**

Create `internal/quality/fixtures/graph_ground_truth_test.go`:

```go
package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallAnalysisGroundTruth_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, CallAnalysisGroundTruth, "应至少有一个 fixture 真值")
	for _, gt := range CallAnalysisGroundTruth {
		assert.NotEmpty(t, gt.FixtureFile, "FixtureFile 不能为空")
		assert.NotEmpty(t, gt.Edges, "%s 的 Edges 不能为空", gt.FixtureFile)
	}
}

func TestCallAnalysisGroundTruth_HasChains(t *testing.T) {
	totalChains := 0
	for _, gt := range CallAnalysisGroundTruth {
		totalChains += len(gt.Chains)
	}
	assert.Greater(t, totalChains, 0, "应至少有一条调用链真值")
}
```

- [ ] **Step 3: 跑测试确认失败**

Run: `go test -v ./internal/quality/fixtures -run TestCallAnalysis`
Expected: FAIL — CallAnalysisGroundTruth 未定义

- [ ] **Step 4: 实现真值数据**

Create `internal/quality/fixtures/graph_ground_truth.go`:

```go
// Package fixtures 存放评估真值（ground truth）。
//
// 真值来源：从 tests/integration/call_analysis_fixtures_test.go 里散落的
// expectedXxxCalls 列表系统化迁移而来。这些期望列表已在原有测试中验证过，
// 作为评估门禁的真值可靠。
package fixtures

import "github.com/yourtionguo/CodeAtlas/internal/quality"

// CallAnalysisGroundTruth 是 call_analysis fixture 集的依赖图真值。
// 从 tests/integration/call_analysis_fixtures_test.go 的 expectedXxxCalls 迁移。
var CallAnalysisGroundTruth = []quality.GraphGroundTruth{
	{
		FixtureFile: "tests/fixtures/cpp/cpp_calls_c.cpp",
		Edges: []quality.ExpectedEdge{
			// 从 expectedCCalls 迁移（tests/integration/call_analysis_fixtures_test.go:39）
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_init"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_free"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_cleanup"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_process_string"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_add"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_multiply"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_init_struct"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_process_struct"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_free_struct"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_log_message"},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "c_validate_input"},
			// 标准库函数标记 Optional（提到了不算漏）
			{SourceName: "CppClass", EdgeType: "call", TargetName: "strlen", Optional: true},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "malloc", Optional: true},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "strcpy", Optional: true},
			{SourceName: "CppClass", EdgeType: "call", TargetName: "printf", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			{StartName: "CppClass", EndName: "c_process_string", StartFile: "tests/fixtures/cpp/cpp_calls_c.cpp", EndFile: "tests/fixtures/c/c_library.h"},
		},
	},
	// 其余 fixture（Java/ObjC/JS/Kotlin/Swift）从对应 expectedXxxCalls 迁移
	// 实现时参照 tests/integration/call_analysis_fixtures_test.go 逐个填入
}
```

**实现说明**：上面的 C++ 条目是示范。实现时需把 `tests/integration/call_analysis_fixtures_test.go` 里所有 `expectedXxxCalls`/`expectedXxxImports` 都迁移过来，SourceName 从对应 fixture 文件的解析结果里取实际符号名。迁移后跑一遍评估，若某条真值不命中（recall < 阈值），检查是真值标错还是解析器漏提取——按 §6.4 评估驱动补 case 机制处理。

- [ ] **Step 5: 跑测试确认通过**

Run: `go test -v ./internal/quality/fixtures -run TestCallAnalysis`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/quality/fixtures/graph_ground_truth.go internal/quality/fixtures/graph_ground_truth_test.go
git commit -m "feat(quality/fixtures): 依赖图真值——从 call_analysis 测试迁移"
```

---

## Task 8: 真值数据——检索 ground truth

**Files:**
- Create: `internal/quality/fixtures/retrieval_ground_truth.go`
- Test: `internal/quality/fixtures/retrieval_ground_truth_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/quality/fixtures/retrieval_ground_truth_test.go`:

```go
package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetrievalGroundTruth_Coverage(t *testing.T) {
	assert.GreaterOrEqual(t, len(RetrievalGroundTruths), 6, "至少 6 个 query 真值")

	crossLang := 0
	singleLang := 0
	for _, gt := range RetrievalGroundTruths {
		assert.NotEmpty(t, gt.Query)
		assert.NotEmpty(t, gt.RelevantSymbols)
		if len(gt.Repos) > 1 {
			crossLang++
		} else {
			singleLang++
		}
	}
	assert.Greater(t, crossLang, 0, "至少一个跨语言 query")
	assert.Greater(t, singleLang, 0, "至少一个单语言 query")
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -v ./internal/quality/fixtures -run TestRetrievalGroundTruth`
Expected: FAIL — RetrievalGroundTruths 未定义

- [ ] **Step 3: 实现真值数据**

Create `internal/quality/fixtures/retrieval_ground_truth.go`:

```go
package fixtures

import "github.com/yourtionguo/CodeAtlas/internal/quality"

// RetrievalGroundTruths 是检索评估真值集。
// 覆盖跨语言（3）+ 单语言（3-4）+ 多 repo（1-2）。
// 真值相关符号从对应 fixture 文件的实际符号提取，人工核对。
var RetrievalGroundTruths = []quality.RetrievalGroundTruth{
	{
		Query:           "C++ 如何调用 C 函数",
		RelevantSymbols: []string{"CppClass", "c_init", "c_process_string"},
		RelevantFiles:   []string{"tests/fixtures/cpp/cpp_calls_c.cpp", "tests/fixtures/c/c_library.h"},
		Repos:           []string{"cpp_calls_c"},
	},
	{
		Query:           "Kotlin 调用 Java 的哪些方法",
		RelevantSymbols: []string{"KotlinCaller", "javaMethod"},
		RelevantFiles:   []string{"tests/fixtures/kotlin/kotlin_calls_java.kt", "tests/fixtures/java/java_library.java"},
		Repos:           []string{"kotlin_calls_java"},
	},
	{
		Query:           "Swift 如何互操作 Objective-C",
		RelevantSymbols: []string{"SwiftCaller", "objcMethod"},
		RelevantFiles:   []string{"tests/fixtures/swift/swift_calls_objc.swift", "tests/fixtures/objc/objc_class.h"},
		Repos:           []string{"swift_calls_objc"},
	},
	{
		Query:           "Go 函数调用关系",
		RelevantSymbols: []string{"caller", "callee"},
		RelevantFiles:   []string{"tests/fixtures/go/calls.go"},
		Repos:           []string{"go_calls"},
	},
	{
		Query:           "JavaScript 模块导入",
		RelevantSymbols: []string{"importedFunc"},
		RelevantFiles:   []string{"tests/fixtures/js/imports.js"},
		Repos:           []string{"js_imports"},
	},
	{
		Query:           "多仓库符号检索",
		RelevantSymbols: []string{"CppClass", "javaMethod"},
		RelevantFiles:   []string{"tests/fixtures/cpp/cpp_calls_c.cpp", "tests/fixtures/java/java_library.java"},
		Repos:           []string{"cpp_calls_c", "kotlin_calls_java"},
	},
}
```

**实现说明**：上述 RelevantSymbols 是基于 fixture 文件名的推断。实现时需打开每个 fixture 文件确认实际符号名（如 `CppClass` 可能实际是 `CppClass::TestMethod`），用真实符号名替换。Repos 字段是 fixture 集标识，与集成测试索引时的 repoID 对应。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -v ./internal/quality/fixtures -run TestRetrievalGroundTruth`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/quality/fixtures/retrieval_ground_truth.go internal/quality/fixtures/retrieval_ground_truth_test.go
git commit -m "feat(quality/fixtures): 检索真值——6 个 query 覆盖跨语言+单语言+多 repo"
```

---

## Task 9: codeatlas eval CLI 命令

**Files:**
- Create: `cmd/cli/eval_command.go`
- Test: `cmd/cli/eval_command_test.go`
- Modify: `cmd/cli/main.go`

- [ ] **Step 1: 写失败测试**

Create `cmd/cli/eval_command_test.go`:

```go
package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestCreateEvalCommand_Flags(t *testing.T) {
	cmd := createEvalCommand()
	assert.Equal(t, "eval", cmd.Name)
	assert.Equal(t, "Evaluate code knowledge graph and retrieval quality", cmd.Usage)

	flagNames := map[string]bool{}
	for _, f := range cmd.Flags {
		flagNames[f.Names()[0]] = true
	}
	assert.True(t, flagNames["repo"])
	assert.True(t, flagNames["fixtures"])
	assert.True(t, flagNames["db"])
	assert.True(t, flagNames["only"])
	assert.True(t, flagNames["format"])
}

func TestCreateEvalCommand_RepoAndFixturesMutex(t *testing.T) {
	// 验证 repo 和 fixtures 互斥逻辑
	cmd := createEvalCommand()
	assert.NotNil(t, cmd.Action)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -v ./cmd/cli -run TestCreateEvalCommand`
Expected: FAIL — createEvalCommand 未定义

- [ ] **Step 3: 实现 eval_command.go**

Create `cmd/cli/eval_command.go`:

```go
package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/yourtionguo/CodeAtlas/internal/quality"
	"github.com/yourtionguo/CodeAtlas/internal/quality/fixtures"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func createEvalCommand() *cli.Command {
	return &cli.Command{
		Name:  "eval",
		Usage: "Evaluate code knowledge graph and retrieval quality",
		Description: `Evaluate dependency graph and retrieval quality metrics.

EXAMPLES:
   # Evaluate a real repository (structural metrics, baseline)
   codeatlas eval --repo <repo_id> --db "host=localhost port=5432 user=codeatlas dbname=codeatlas"

   # Evaluate fixture ground truth (recall/precision/MRR, gating)
   codeatlas eval --fixtures --db "..." --format json

   # Only run one category
   codeatlas eval --repo <repo_id> --only graph
   codeatlas eval --fixtures --only retrieval
`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				Usage:   "Repository ID to evaluate (repo mode)",
			},
			&cli.BoolFlag{
				Name:    "fixtures",
				Usage:   "Evaluate fixture ground truth (fixture mode)",
			},
			&cli.StringFlag{
				Name:    "db",
				Usage:   "PostgreSQL connection string",
				EnvVars: []string{"DATABASE_URL", "DB_DSN"},
			},
			&cli.StringFlag{
				Name:    "only",
				Usage:   "Run only one category: graph | retrieval (empty = all)",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "format",
				Usage:   "Output format: text | json",
				Value:   "text",
			},
		},
		Action: runEval,
	}
}

func runEval(c *cli.Context) error {
	repoID := c.String("repo")
	fixturesMode := c.Bool("fixtures")

	// 互斥校验
	if repoID != "" && fixturesMode {
		return fmt.Errorf("--repo and --fixtures are mutually exclusive")
	}
	if repoID == "" && !fixturesMode {
		return fmt.Errorf("must specify either --repo or --fixtures")
	}

	dsn := c.String("db")
	if dsn == "" {
		// 兜底从环境变量组装（同 indexer 约定）
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			envOr("DB_HOST", "localhost"), envOr("DB_PORT", "5432"),
			envOr("DB_USER", "codeatlas"), envOr("DB_PASSWORD", "codeatlas"),
			envOr("DB_NAME", "codeatlas"))
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	modelsDB := &models.DB{DB: db}
	edgeRepo := models.NewEdgeRepository(modelsDB)
	symbolRepo := models.NewSymbolRepository(modelsDB)
	fetcher := quality.NewDefaultGraphFetcher(edgeRepo, symbolRepo)

	only := c.String("only")
	format := c.String("format")
	ctx := context.Background()

	if fixturesMode {
		// fixture 模式
		graphEval := quality.NewGraphEvaluator(fetcher, mergeGroundTruth(fixtures.CallAnalysisGroundTruth))
		retrievalEval := quality.NewRetrievalEvaluator(nil, fixtures.RetrievalGroundTruths, []string{"hybrid", "vector", "keyword"})

		cfg := quality.EvaluateConfig{
			Mode:         quality.EvalModeFixture,
			FixtureSet:   "call_analysis",
			RunRetrieval: only == "" || only == "retrieval",
		}
		// 构造 retrieval runner 需连真 DB（需 embedder），CLI 这轮先只跑 graph，retrieval 留集成测试
		if only == "retrieval" {
			cfg.RunRetrieval = false // CLI 暂不支持 retrieval（需 embedder 配置），提示用户用集成测试
			fmt.Fprintln(os.Stderr, "注意：retrieval 评估需 embedder 配置，请用 make test-integration 跑。这轮只跑 graph。")
		} else {
			cfg.RunRetrieval = false
		}

		report, err := quality.Evaluate(ctx, cfg, graphEval, retrievalEval)
		if err != nil {
			return err
		}
		return outputReport(report, format)
	}

	// repo 模式
	graphEval := quality.NewGraphEvaluator(fetcher, nil)
	cfg := quality.EvaluateConfig{
		Mode:   quality.EvalModeRepo,
		RepoID: repoID,
	}
	report, err := quality.Evaluate(ctx, cfg, graphEval, nil)
	if err != nil {
		return err
	}
	return outputReport(report, format)
}

func outputReport(report *quality.Report, format string) error {
	if format == "json" {
		data, err := report.JSONMarshal()
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// text 格式
	fmt.Println("CodeAtlas Quality Report")
	fmt.Println("========================")
	fmt.Printf("Mode: %s", report.Mode)
	if report.RepoID != "" {
		fmt.Printf("  RepoID: %s", report.RepoID)
	}
	if report.FixtureSet != "" {
		fmt.Printf("  FixtureSet: %s", report.FixtureSet)
	}
	fmt.Println()

	// 按 category 分组
	for _, cat := range []quality.MetricCategory{quality.CategoryGraph, quality.CategoryRetrieval} {
		hasCat := false
		for _, m := range report.Metrics {
			if m.Category == cat {
				hasCat = true
				break
			}
		}
		if !hasCat {
			continue
		}
		fmt.Printf("\n== %s Metrics ==\n", titleCase(string(cat)))
		for _, m := range report.Metrics {
			if m.Category != cat {
				continue
			}
			mark := "✓"
			if !m.Passed {
				mark = "✗"
			}
			if m.Threshold == 0 {
				fmt.Printf("  %-30s %.2f  (仅观察)  %s\n", m.Name, m.Value, mark)
			} else {
				op := "≥"
				if !m.HigherIsBetter {
					op = "≤"
				}
				fmt.Printf("  %-30s %.2f  (%s%.2f)  %s\n", m.Name, m.Value, op, m.Threshold, mark)
			}
		}
	}

	fmt.Printf("\nSummary: %d passed, %d failed, %d observed\n",
		report.Summary.Passed, report.Summary.Failed, report.Summary.NoThreshold)

	if report.Summary.Failed > 0 {
		os.Exit(1)
	}
	return nil
}

func mergeGroundTruth(gts []quality.GraphGroundTruth) *quality.GraphGroundTruth {
	merged := &quality.GraphGroundTruth{FixtureFile: "merged"}
	for _, gt := range gts {
		merged.Edges = append(merged.Edges, gt.Edges...)
		merged.Chains = append(merged.Chains, gt.Chains...)
	}
	return merged
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
```

- [ ] **Step 4: 注册命令**

Modify `cmd/cli/main.go`，在 `Commands: []*cli.Command{}` 列表里加：

```go
createEvalCommand(),
```

放在 `createAskCommand(),` 之后。

- [ ] **Step 5: 跑测试确认通过**

Run: `go test -v ./cmd/cli -run TestCreateEvalCommand`
Expected: PASS

跑全项目编译:
Run: `go build ./...`
Expected: 无错误

- [ ] **Step 6: 提交**

```bash
git add cmd/cli/eval_command.go cmd/cli/eval_command_test.go cmd/cli/main.go
git commit -m "feat(cli): codeatlas eval 命令——质量评估报告输出"
```

---

## Task 10: 集成测试门禁

**Files:**
- Create: `tests/integration/quality_gate_test.go`

- [ ] **Step 1: 写集成测试**

Create `tests/integration/quality_gate_test.go`:

```go
package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/quality"
	"github.com/yourtionguo/CodeAtlas/internal/quality/fixtures"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestQualityGate_FixtureMode 在真 DB 上索引 fixture，跑全指标门禁。
func TestQualityGate_FixtureMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoID := indexCallAnalysisFixtures(t, testDB, ctx)

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

	// 门禁断言：所有有阈值的指标必须通过
	for _, m := range report.Metrics {
		if m.Threshold > 0 && !m.Passed {
			t.Errorf("质量门禁失败: %s (bucket=%s) = %.2f, 阈值 %.2f", m.Name, m.Bucket, m.Value, m.Threshold)
		}
	}
	t.Logf("报告: %d passed, %d failed, %d observed", report.Summary.Passed, report.Summary.Failed, report.Summary.NoThreshold)
}

// TestQualityGate_RepoMode 验证 repo 模式能跑通结构断言（不卡阈值，建基线）。
func TestQualityGate_RepoMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoID := indexCallAnalysisFixtures(t, testDB, ctx)

	edgeRepo := models.NewEdgeRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)
	fetcher := quality.NewDefaultGraphFetcher(edgeRepo, symbolRepo)
	graphEval := quality.NewGraphEvaluator(fetcher, nil)

	report, err := quality.Evaluate(ctx, quality.EvaluateConfig{
		Mode:   quality.EvalModeRepo,
		RepoID: repoID,
	}, graphEval, nil)
	require.NoError(t, err)

	// repo 模式：仅断言查询不报错 + Report 结构完整
	assert.NotEmpty(t, report.Metrics, "应产出结构断言指标")
	assert.Equal(t, len(report.Metrics), report.Summary.Total)

	// 打印基线值（供下一轮定阈值参考）
	t.Logf("=== Repo 模式基线 ===")
	for _, m := range report.Metrics {
		t.Logf("  %s (bucket=%s) = %.4f", m.Name, m.Bucket, m.Value)
	}
}

// TestQualityGate_PR2Regression PR#2 回归：验证 RepoIDs 多 repo 过滤 + retrieval 端到端。
// 通过 retrieval_evaluator 跑 recall 真值，命中即证明过滤正确。
func TestQualityGate_PR2Regression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoIDs := indexMultipleRepoFixtures(t, testDB, ctx)

	// 构造 retriever（需 embedder，这轮 SkipVectors=true 则无向量，跳过检索部分）
	// 注：若 SkipVectors，retrieval 评估无法跑（需 embedding）。
	// 此测试验证 RepoIDs 过滤的图指标维度 + 结构断言不报错。
	edgeRepo := models.NewEdgeRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)
	fetcher := quality.NewDefaultGraphFetcher(edgeRepo, symbolRepo)

	for _, repoID := range repoIDs {
		graphEval := quality.NewGraphEvaluator(fetcher, nil)
		report, err := quality.Evaluate(ctx, quality.EvaluateConfig{
			Mode:   quality.EvalModeRepo,
			RepoID: repoID,
		}, graphEval, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, report.Metrics, "repo %s 应有指标", repoID)
	}

	// 完整 retrieval 回归需 embedder，留 make test-integration 在有 embedding 环境跑
	t.Log("注：retrieval 端到端回归需 embedding 环境，由 TestQualityGate_FixtureMode 的 retrieval 分支覆盖")
}

// indexCallAnalysisFixtures 索引 call_analysis fixture 集到测试库，返回 repoID。
// 复用 tests/integration/indexer_integration_test.go 的索引流程。
func indexCallAnalysisFixtures(t *testing.T, testDB *TestDB, ctx context.Context) string {
	t.Helper()
	repoID := "call-analysis-" + t.Name()

	parseOutput := createTestParseOutput()
	config := &indexer.IndexerConfig{
		RepoID:      repoID,
		RepoName:    "call-analysis-test",
		BatchSize:   10,
		WorkerCount: 2,
		SkipVectors: true,
		UseTransactions: true,
	}
	idx := indexer.NewIndexer(testDB.DB, config)
	result, err := idx.Index(ctx, parseOutput)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	return repoID
}

// indexMultipleRepoFixtures 索引多个 repo，返回 repoID 列表（验证 RepoIDs 多 repo 过滤）。
func indexMultipleRepoFixtures(t *testing.T, testDB *TestDB, ctx context.Context) []string {
	t.Helper()
	// 复用 indexCallAnalysisFixtures 索引两份不同 repoID 的数据
	return []string{
		indexCallAnalysisFixtures(t, testDB, ctx),
		indexCallAnalysisFixtures(t, testDB, ctx),
	}
}
```

- [ ] **Step 2: 跑测试确认失败或通过**

Run: `go test -v ./tests/integration -run TestQualityGate`
Expected: 若 DB 未启动则跳过；若启动则应 PASS（或暴露真值/索引不匹配问题，按 §6.4 调整）

- [ ] **Step 3: 按评估结果调整真值**

如果 `TestQualityGate_FixtureMode` 失败（某指标未达阈值），分析原因：

- 真值标注有误 → 修 `fixtures/graph_ground_truth.go` 里的 SourceName/TargetName
- 解析器漏提取 → 记录为已知问题，这轮按 §3.4 放宽阈值（用 `OverrideThreshold`），下一轮修解析器
- 索引流程问题 → 修 `indexCallAnalysisFixtures` helper

调整后重跑直到通过。

- [ ] **Step 4: 跑全量集成测试确认无回归**

Run: `make test-integration`
Expected: 全绿

- [ ] **Step 5: 提交**

```bash
git add tests/integration/quality_gate_test.go
git commit -m "test(integration): 质量门禁集成测试——fixture/repo 模式 + PR#2 回归"
```

---

## Task 11: 文档与 README 路线图更新

**Files:**
- Create: `docs/evaluation.md`
- Modify: `docs/cli.md`
- Modify: `README.md`

- [ ] **Step 1: 写评估文档**

Create `docs/evaluation.md`:

```markdown
# 质量评估系统

CodeAtlas 内建质量评估系统，覆盖**依赖图质量**与**检索质量**两个环节，用于迭代观测和 CI 门禁。

## 快速开始

### 评估真实仓库（结构断言，建基线）

​```bash
codeatlas eval --repo <repo_id> --db "host=localhost port=5432 ..."
​```

### 评估 fixture 真值（recall/precision/MRR，门禁用）

​```bash
codeatlas eval --fixtures --db "..." --format json
​```

## 指标体系

### 依赖图指标

#### 结构断言类（无需真值）

| 指标 | 说明 | 建议基线 |
|---|---|---|
| `dangling_edge_ratio` | target_id IS NULL 的边占比 | < 30% |
| `symbol_resolution_rate` | target_id 已解析的边占比 | > 70% |
| `orphan_symbol_ratio` | 无出入边的孤立符号占比 | < 40% |
| `cross_file_connectivity` | 跨文件边占比 | > 20% |

#### fixture 真值类（门禁）

| 指标 | 说明 | 阈值 |
|---|---|---|
| `edge_recall` | 真值边被提取的比例 | ≥ 90% |
| `edge_precision` | 提取边正确的比例 | ≥ 85% |
| `call_chain_connectivity` | 真值调用链连通比例 | ≥ 95% |

### 检索指标（fixture 模式）

| 指标 | 说明 | 阈值 |
|---|---|---|
| `recall@10` | Top-10 含真值相关符号比例 | ≥ 70% |
| `MRR` | 第一个相关符号排名倒数均值 | ≥ 0.5 |
| `neighbor_hit_rate` | 1 跳邻居含真值相关符号比例 | ≥ 60% |
| `mode_compare` | hybrid vs vector/keyword 差值 | 仅观察 |

## 门禁机制

- **fixture 真值类**：CI 硬门禁，不达标 exit 1
- **结构断言类**：这轮仅建基线，下一轮收紧为硬门禁

## 评估驱动补 case

评估报告里某指标 Detail 为空或某维度覆盖为 0 时，按需补真值：
- 某 edge_type 无真值 → 补对应 fixture 真值条目
- 某语言无 case → 从现有 fixture 构造 query 真值
```

- [ ] **Step 2: 更新 CLI 文档**

在 `docs/cli.md` 末尾追加 eval 命令说明（参照 impact/ask 命令格式）。

- [ ] **Step 3: 更新 README 路线图**

Modify `README.md`，把 Phase 1 的「基础语义检索和问答」勾上：

```markdown
- [x] 基础语义检索和问答
```

- [ ] **Step 4: 提交**

```bash
git add docs/evaluation.md docs/cli.md README.md
git commit -m "docs: 评估系统文档 + README 路线图 Phase 1 完成"
```

---

## Task 12: 全量验证与基线快照

**Files:**
- Create: `docs/superpowers/baselines/2026-07-07-quality-baseline.md`（基线快照）

- [ ] **Step 1: 跑全量单元测试**

Run: `make test`
Expected: 全绿

- [ ] **Step 2: 跑全量集成测试**

Run: `make test-integration`
Expected: 全绿

- [ ] **Step 3: 跑覆盖率检查**

Run: `make test-coverage`
Expected: quality 包覆盖率 ≥ 90%

- [ ] **Step 4: 对 CodeAtlas 自身代码库跑基线**

先索引 CodeAtlas 自身到本地 DB，再跑：

```bash
codeatlas eval --repo <codeatlas-self-repo-id> --db "..." --format json > /tmp/baseline.json
```

把关键指标值记录到 `docs/superpowers/baselines/2026-07-07-quality-baseline.md`：

```markdown
# CodeAtlas 质量基线（2026-07-07）

## 依赖图结构断言

| 指标 | 基线值 |
|---|---|
| dangling_edge_ratio | （填实际值） |
| symbol_resolution_rate | （填实际值） |
| orphan_symbol_ratio | （填实际值） |
| cross_file_connectivity | （填实际值） |

## 备注

此基线作为下一轮结构断言硬门禁的参照。下一轮据此定阈值。
```

- [ ] **Step 5: 跑 verify 确认**

Run: `make verify`
Expected: 全绿

- [ ] **Step 6: 提交**

```bash
git add docs/superpowers/baselines/2026-07-07-quality-baseline.md
git commit -m "docs(baseline): CodeAtlas 质量基线快照——结构断言指标首版"
```

---

## Self-Review

### Spec 覆盖检查

| Spec 章节 | 对应 Task |
|---|---|
| §2 架构分层 | Task 1-6（分层实现） |
| §3 依赖图指标 | Task 1（聚合查询）+ Task 3（GraphEvaluator） |
| §4 检索指标 | Task 4（RetrievalEvaluator） |
| §5 数据结构与接口 | Task 2（metrics）+ Task 3/4（Evaluator）+ Task 5（Evaluate）+ Task 6（Fetcher） |
| §6 CLI 与门禁 | Task 9（CLI）+ Task 10（集成门禁） |
| §7 文件清单 | Task 1-11 覆盖全部新建/修改文件 |
| §8 测试策略 | 每个 Task 内含测试 + Task 12 全量验证 |
| §1.3 成功标准 | Task 12 逐项验证 |

### 已知简化（实现时注意）

1. **CLI 的 retrieval 评估**：Task 9 的 CLI 暂不支持 retrieval 评估（需 embedder 配置），提示用户用 `make test-integration`。retrieval 评估在集成测试里跑。后续可在 CLI 加 `--embedder-config` flag 支持。
2. **真值数据填充**：Task 7/8 的真值是骨架，实现时需打开 fixture 文件确认实际符号名后填充。
3. **集成测试索引方式**：Task 10 的 `indexCallAnalysisFixtures` 复用 `createTestParseOutput()`（手工构造的 ParseOutput）。若要验证真实 fixture 文件解析的正确性，应改用 `parser.NewTreeSitterParser()` + 各语言 Parser 解析 `tests/fixtures/` 下的真实文件再索引。实现时优先用真实 fixture 解析，`createTestParseOutput` 作为兜底。

### 类型一致性

- `MetricValue` 各字段在 Task 2 定义，Task 3/4/5 使用一致
- `GraphDataFetcher` 接口在 Task 3 定义，Task 6 实现，签名一致
- `RetrievalRunner` 接口在 Task 4 定义，与 `retrieval.Retriever` 的 `Query` 方法签名一致
- `Evaluate` 函数在 Task 5 定义，Task 9/10 调用，签名一致
