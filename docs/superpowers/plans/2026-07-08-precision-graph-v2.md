# 精准依赖图 v2 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 C++ 重载导致 recall>1.0 的评估器缺陷，展平 Java/Kotlin 方法符号到符号表，基于新基线启用结构断言硬门禁。

**Architecture:** 三个 Task 串联——Task 1 把 `computeEdgeMatch` 匹配 key 从 name 升级为 symbol_id；Task 2 让 `CollectSymbols` 递归展平 `ParsedSymbol.Children` 使方法符号进入候选集；Task 3 跑新基线并按 edge_type 分桶启用硬门禁。改动覆盖评估器（quality）+ 解析器映射（schema/mapper）+ fixture 真值 + 门禁阈值常量，不碰 DB schema/API/前端。

**Tech Stack:** Go 1.25+, Tree-sitter, PostgreSQL, 现有 quality 评估框架

**Spec:** `docs/superpowers/specs/2026-07-08-precision-graph-v2-design.md`

---

## 文件结构

| 文件 | 操作 | 职责 |
|---|---|---|
| `internal/quality/graph_evaluator.go` | 修改 | ExtractedEdge/ExpectedEdge 加 SourceID/TargetID；computeEdgeMatch 改 symbol_id 匹配；启用分桶硬门禁 |
| `pkg/models/graph_metrics.go` | 修改 | ExtractedEdge 加 SourceID/TargetID；ListExtractedEdges SQL 增查 source_id/target_id |
| `internal/quality/graph_data_fetcher.go` | 修改 | ListExtractedEdges 透传新字段 |
| `internal/quality/metrics.go` | 修改 | 新增分桶阈值常量 |
| `internal/schema/mapper.go` | 修改 | CollectSymbols 递归展平 Children |
| `internal/quality/fixtures/graph_ground_truth.go` | 修改 | 新增 Java/Kotlin 方法级真值 |
| `internal/quality/fixtures/graph_ground_truth_test.go` | 修改 | 新增真值校验 |
| `internal/quality/graph_evaluator_test.go` | 修改 | symbol_id 匹配测试 |
| `internal/schema/mapper_test.go` | 修改 | 展平 Children 测试 |
| `tests/integration/quality_gate_test.go` | 修改 | indexRealFixtures 加 Java/Kotlin 多文件；真值 ID 回填；新阈值门禁 |
| `docs/superpowers/baselines/2026-07-08-precision-graph-v2-baseline.md` | 新建 | 新基线文档 |

---

## Task 1：评估器 symbol_id 匹配

**Files:**
- Modify: `pkg/models/graph_metrics.go:155-195`
- Modify: `internal/quality/graph_evaluator.go:8-37, 214-256`
- Modify: `internal/quality/graph_data_fetcher.go:64-78`
- Test: `internal/quality/graph_evaluator_test.go`

### Step 1: 写失败测试 — ExtractedEdge 结构有 SourceID/TargetID

- [ ] **在 `internal/quality/graph_evaluator_test.go` 末尾追加测试**

```go
func TestExtractedEdge_HasIDFields(t *testing.T) {
	e := ExtractedEdge{
		SourceID:   "src-uuid-1",
		SourceName: "MyClass",
		EdgeType:   "call",
		TargetID:   "tgt-uuid-1",
		TargetName: "doSomething",
	}
	if e.SourceID == "" || e.TargetID == "" {
		t.Fatalf("ExtractedEdge 应有 SourceID/TargetID 字段，got SourceID=%q TargetID=%q", e.SourceID, e.TargetID)
	}
}
```

- [ ] **运行测试验证失败**

Run: `go test ./internal/quality/ -run TestExtractedEdge_HasIDFields -v`
Expected: FAIL — `ExtractedEdge` 没有 `SourceID`/`TargetID` 字段，编译错误

### Step 2: 给 ExtractedEdge 加 SourceID/TargetID 字段

- [ ] **修改 `internal/quality/graph_evaluator.go` 的 ExtractedEdge 结构（第 32-37 行）**

把：
```go
// ExtractedEdge 从 DB 查出的提取边（用于真值匹配）。
type ExtractedEdge struct {
	SourceName string
	EdgeType   string
	TargetName string // 悬空时为空
}
```
改为：
```go
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
```

- [ ] **同时修改 `pkg/models/graph_metrics.go` 的 ExtractedEdge 结构（第 156-163 行）**

把：
```go
type ExtractedEdge struct {
	SourceName string
	EdgeType   string
	TargetName string
}
```
改为：
```go
type ExtractedEdge struct {
	SourceID   string
	SourceName string
	EdgeType   string
	TargetID   string
	TargetName string
}
```

- [ ] **运行测试验证通过**

Run: `go test ./internal/quality/ -run TestExtractedEdge_HasIDFields -v`
Expected: PASS

### Step 3: 修改 ListExtractedEdges SQL 返回 source_id/target_id

- [ ] **修改 `pkg/models/graph_metrics.go` 的 ListExtractedEdges（第 171-195 行）**

把查询和扫描改为：
```go
func ListExtractedEdges(ctx context.Context, r *EdgeRepository, repoID string) ([]ExtractedEdge, error) {
	query := `
		SELECT e.source_id, s_source.name, e.edge_type,
		       COALESCE(e.target_id, ''), COALESCE(s_target.name, COALESCE(e.target_module, ''))
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
		if err := rows.Scan(&e.SourceID, &e.SourceName, &e.EdgeType, &e.TargetID, &e.TargetName); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
```

- [ ] **修改 `internal/quality/graph_data_fetcher.go` 的 ListExtractedEdges（第 64-78 行）**

把：
```go
	for i, e := range rawEdges {
		result[i] = ExtractedEdge{
			SourceName: e.SourceName,
			EdgeType:   e.EdgeType,
			TargetName: e.TargetName,
		}
	}
```
改为：
```go
	for i, e := range rawEdges {
		result[i] = ExtractedEdge{
			SourceID:   e.SourceID,
			SourceName: e.SourceName,
			EdgeType:   e.EdgeType,
			TargetID:   e.TargetID,
			TargetName: e.TargetName,
		}
	}
```

- [ ] **编译验证**

Run: `go build ./internal/quality/... ./pkg/models/...`
Expected: 编译通过

### Step 4: 写失败测试 — ExpectedEdge 有 SourceID/TargetID

- [ ] **在 `internal/quality/graph_evaluator_test.go` 追加测试**

```go
func TestExpectedEdge_HasIDFields(t *testing.T) {
	e := ExpectedEdge{
		SourceID:   "src-uuid-1",
		SourceName: "MyClass",
		EdgeType:   "call",
		TargetID:   "tgt-uuid-1",
		TargetName: "doSomething",
	}
	if e.SourceID == "" || e.TargetID == "" {
		t.Fatalf("ExpectedEdge 应有 SourceID/TargetID 字段")
	}
}
```

- [ ] **运行测试验证失败**

Run: `go test ./internal/quality/ -run TestExpectedEdge_HasIDFields -v`
Expected: FAIL — 编译错误，ExpectedEdge 没有 SourceID/TargetID

### Step 5: 给 ExpectedEdge 加 SourceID/TargetID 字段

- [ ] **修改 `internal/quality/graph_evaluator.go` 的 ExpectedEdge 结构（第 8-15 行）**

把：
```go
// ExpectedEdge 真值里的一条期望边。
// 不依赖 symbol_id（入库后才有），用符号名匹配。
type ExpectedEdge struct {
	SourceName string // "CppClass::CppMethod"
	EdgeType   string // "call"
	TargetName string // "c_init"
	Optional   bool   // true = 提到了不算漏（如标准库 strlen）
}
```
改为：
```go
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
```

- [ ] **运行测试验证通过**

Run: `go test ./internal/quality/ -run TestExpectedEdge_HasIDFields -v`
Expected: PASS

### Step 6: 写失败测试 — computeEdgeMatch 用 symbol_id 匹配

- [ ] **在 `internal/quality/graph_evaluator_test.go` 追加测试**

```go
func TestComputeEdgeMatch_SymbolID_Dedup(t *testing.T) {
	// 模拟 C++ 重载：两个同名 MyClass 构造函数，symbol_id 不同。
	// extracted 有两条 (SourceID 不同, TargetID 不同) 的边。
	// 真值只有一条。按 symbol_id 匹配，recall 不应超过 1.0。
	truth := []ExpectedEdge{
		{SourceID: "src-1", EdgeType: "call", TargetID: "tgt-1"},
	}
	extracted := []ExtractedEdge{
		{SourceID: "src-1", EdgeType: "call", TargetID: "tgt-1"},
		{SourceID: "src-2", EdgeType: "call", TargetID: "tgt-2"}, // 另一个重载
	}
	recall, precision := computeEdgeMatch(truth, extracted)
	if recall > 1.0 {
		t.Fatalf("recall 不应超过 1.0，got %f", recall)
	}
	if recall != 1.0 {
		t.Fatalf("真值边命中，recall 应为 1.0，got %f", recall)
	}
	// precision：2 条 extracted 里 1 条匹配真值
	if precision != 0.5 {
		t.Fatalf("precision 应为 0.5（2条里1条匹配），got %f", precision)
	}
}

func TestComputeEdgeMatch_DanglingExcluded(t *testing.T) {
	// 悬空边（TargetID 空）不参与匹配
	truth := []ExpectedEdge{
		{SourceID: "src-1", EdgeType: "call", TargetID: "tgt-1"},
	}
	extracted := []ExtractedEdge{
		{SourceID: "src-1", EdgeType: "call", TargetID: "tgt-1"},
		{SourceID: "src-2", EdgeType: "call", TargetID: ""}, // 悬空
	}
	recall, precision := computeEdgeMatch(truth, extracted)
	if recall != 1.0 {
		t.Fatalf("recall 应为 1.0，got %f", recall)
	}
	// precision：悬空边不匹配真值，2 条里 1 条匹配
	if precision != 0.5 {
		t.Fatalf("precision 应为 0.5（悬空边不算匹配），got %f", precision)
	}
}

func TestComputeEdgeMatch_OptionalSkipped(t *testing.T) {
	// Optional=true 的真值边不计入 recall 分母
	truth := []ExpectedEdge{
		{SourceID: "src-1", EdgeType: "call", TargetID: "tgt-1"},
		{SourceID: "src-2", EdgeType: "call", TargetID: "", Optional: true},
	}
	extracted := []ExtractedEdge{
		{SourceID: "src-1", EdgeType: "call", TargetID: "tgt-1"},
	}
	recall, _ := computeEdgeMatch(truth, extracted)
	if recall != 1.0 {
		t.Fatalf("Optional 边不计入分母，recall 应为 1.0，got %f", recall)
	}
}
```

- [ ] **运行测试验证失败**

Run: `go test ./internal/quality/ -run "TestComputeEdgeMatch_" -v`
Expected: FAIL — computeEdgeMatch 仍用 name 匹配，这些测试用 ID 字段会失败

### Step 7: 改 computeEdgeMatch 为 symbol_id 匹配

- [ ] **修改 `internal/quality/graph_evaluator.go` 的 computeEdgeMatch（第 211-256 行）**

把整个函数替换为：
```go
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
```

- [ ] **运行测试验证通过**

Run: `go test ./internal/quality/ -run "TestComputeEdgeMatch_" -v`
Expected: PASS（3 个子测试全过）

- [ ] **运行现有评估器测试确保不退化**

Run: `go test ./internal/quality/ -run "TestComputeEdgeMatch|TestGraphEvaluator" -v`
Expected: PASS

### Step 8: 写 ResolveTruthIDs 辅助函数 — 索引后回填真值 symbol_id

- [ ] **在 `internal/quality/fixtures/graph_ground_truth.go` 末尾追加辅助函数**

先在文件 import 区加 `"context"` 和 `"github.com/yourtionguo/CodeAtlas/pkg/models"`（如果还没有）。然后在文件末尾追加：

```go
// ResolveTruthIDs 索引 fixture 后回填真值边的 SourceID/TargetID。
//
// symbol_id 是 GenerateDeterministicUUID 基于 (file_id, name, start_line, start_byte) 产出的，
// 虽然确定性，但硬编码脆弱。改为索引后从 DB 查出回填。
//
// 匹配策略：按 (name, file_path) 查找符号。file_path 为空时取该 name 的首个候选。
// 查不到的 ID 留空——computeEdgeMatch 会跳过 TargetID 空的边。
func ResolveTruthIDs(ctx context.Context, symbolRepo *models.SymbolRepository, truth []quality.GraphGroundTruth) error {
	for gi := range truth {
		gt := &truth[gi]
		for i := range gt.Edges {
			edge := &gt.Edges[i]
			if edge.SourceName != "" && edge.SourceID == "" {
				if sid, err := lookupSymbolID(ctx, symbolRepo, edge.SourceName, ""); err == nil && sid != "" {
					edge.SourceID = sid
				}
			}
			if edge.TargetName != "" && edge.TargetID == "" {
				if sid, err := lookupSymbolID(ctx, symbolRepo, edge.TargetName, ""); err == nil && sid != "" {
					edge.TargetID = sid
				}
			}
		}
		// Chains 用 name+file 查询连通性，不需回填 ID
	}
	return nil
}

// lookupSymbolID 按 name（和可选 file path）查符号 ID。
func lookupSymbolID(ctx context.Context, repo *models.SymbolRepository, name string, filePath string) (string, error) {
	syms, err := repo.GetByName(ctx, name)
	if err != nil || len(syms) == 0 {
		return "", err
	}
	if filePath == "" {
		return syms[0].SymbolID, nil
	}
	// 优先匹配 file path
	for _, s := range syms {
		if s.FilePath == filePath {
			return s.SymbolID, nil
		}
	}
	return syms[0].SymbolID, nil
}
```

- [ ] **检查 `SymbolRepository` 是否有 `GetByName` 方法**

Run: `grep -n "func.*SymbolRepository.*GetByName\|func.*GetByName" pkg/models/symbol.go`
如果没有，需新增。如果有，确认签名。

- [ ] **如果 `GetByName` 不存在，在 `pkg/models/symbol.go` 新增**

```go
// GetByName 按符号名查询所有匹配的符号（可能多个，如重载）。
func (r *SymbolRepository) GetByName(ctx context.Context, name string) ([]*Symbol, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT symbol_id, file_id, name, kind, signature, start_line, end_line, start_byte, end_byte, docstring
		 FROM symbols WHERE name = $1`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*Symbol
	for rows.Next() {
		var s Symbol
		if err := rows.Scan(&s.SymbolID, &s.FileID, &s.Name, &s.Kind, &s.Signature,
			&s.Span.StartLine, &s.Span.EndLine, &s.Span.StartByte, &s.Span.EndByte, &s.Docstring); err != nil {
			return nil, err
		}
		result = append(result, &s)
	}
	return result, rows.Err()
}
```

注：需确认 `Symbol` 结构字段名与 `symbols` 表列名一致。实现时用 `grep -n "type Symbol struct" pkg/models/symbol.go -A 20` 核对。

- [ ] **编译验证**

Run: `go build ./internal/quality/fixtures/... ./pkg/models/...`
Expected: 编译通过

### Step 9: 在 quality_gate_test.go 里索引后调用 ResolveTruthIDs

- [ ] **修改 `tests/integration/quality_gate_test.go` 的 TestQualityGate_FixtureMode（第 39-62 行）**

在 `graphEval := quality.NewGraphEvaluator(fetcher, truth)` 之前，加回填逻辑。把：
```go
	graphEval := quality.NewGraphEvaluator(fetcher, truth)
```
改为：
```go
	// 回填真值边的 symbol_id（索引后才能查到）
	truthSlice := []quality.GraphGroundTruth{*truth}
	if err := fixtures.ResolveTruthIDs(ctx, symbolRepo, truthSlice); err != nil {
		t.Logf("ResolveTruthIDs 出错（非致命，symbol_id 为空的边会被跳过）: %v", err)
	}
	truth = &truthSlice[0] // 回填后的真值

	graphEval := quality.NewGraphEvaluator(fetcher, truth)
```

注：`truth` 是 `*quality.GraphGroundTruth`，ResolveTruthIDs 接受 `[]quality.GraphGroundTruth`（值切片），所以解引用后传入，再取回填后的地址。

- [ ] **编译验证**

Run: `go build ./tests/integration/...`
Expected: 编译通过

### Step 10: 运行单元测试全量 + 提交

- [ ] **运行 quality 包全量单元测试**

Run: `go test ./internal/quality/... -v`
Expected: PASS（所有现有测试 + 新增测试）

- [ ] **提交**

```bash
git add internal/quality/graph_evaluator.go internal/quality/graph_evaluator_test.go \
  internal/quality/graph_data_fetcher.go internal/quality/fixtures/graph_ground_truth.go \
  pkg/models/graph_metrics.go pkg/models/symbol.go tests/integration/quality_gate_test.go
git commit -m "fix(quality): computeEdgeMatch 改用 symbol_id 匹配——修复 C++ 重载 recall>1.0"
```

---

## Task 2：Java/Kotlin 方法符号展平

**Files:**
- Modify: `internal/schema/mapper.go:76-84`
- Modify: `tests/integration/quality_gate_test.go:115-126`（fixtureFiles 列表）
- Modify: `internal/quality/fixtures/graph_ground_truth.go`（新增方法级真值）
- Test: `internal/schema/mapper_test.go`
- Test: `internal/quality/fixtures/graph_ground_truth_test.go`

### Step 1: 写失败测试 — CollectSymbols 展平 Children

- [ ] **在 `internal/schema/mapper_test.go` 追加测试**

```go
func TestCollectSymbols_FlattenChildren(t *testing.T) {
	mapper := NewSchemaMapper()

	// 构造一个含 Children 的 ParsedFile（模拟 Java/Kotlin 类+方法）
	parsed := &parser.ParsedFile{
		Path:     "Test.java",
		Language: "java",
		Content:  []byte("class Foo { void bar() {} }"),
		Symbols: []parser.ParsedSymbol{
			{
				Name: "Foo",
				Kind: "class",
				Span: parser.Span{StartLine: 1, StartByte: 0, EndLine: 1, EndByte: 25},
				Children: []parser.ParsedSymbol{
					{
						Name: "bar",
						Kind: "method",
						Span: parser.Span{StartLine: 1, StartByte: 12, EndLine: 1, EndByte: 23},
					},
				},
			},
		},
	}

	file, err := mapper.CollectSymbols(parsed)
	if err != nil {
		t.Fatalf("CollectSymbols 失败: %v", err)
	}

	// 顶层类符号 + 展平出的方法符号 = 2
	if len(file.Symbols) != 2 {
		t.Fatalf("应展平出 2 个符号（1 class + 1 method），got %d: %+v", len(file.Symbols), file.Symbols)
	}

	// 方法符号应在 symbolCandidates 里
	candidates := mapper.symbolCandidates["bar"]
	if len(candidates) == 0 {
		t.Fatal("方法符号 bar 应进入 symbolCandidates，但未找到")
	}
}
```

- [ ] **运行测试验证失败**

Run: `go test ./internal/schema/ -run TestCollectSymbols_FlattenChildren -v`
Expected: FAIL — `len(file.Symbols)` 为 1（只收集了顶层 class），Children 没展平

### Step 2: 实现 collectSymbolRecursive 递归展平

- [ ] **修改 `internal/schema/mapper.go` 的 CollectSymbols（第 76-84 行）**

把：
```go
	// 收集符号到候选集（累积，不覆盖）
	for _, parsedSymbol := range parsed.Symbols {
		symbol := m.mapSymbol(parsedSymbol, fileID)
		file.Symbols = append(file.Symbols, symbol)
		m.symbolIDMap[parsedSymbol.Name] = symbol.SymbolID // 向后兼容
		m.symbolCandidates[parsedSymbol.Name] = append(
			m.symbolCandidates[parsedSymbol.Name],
			symbolCandidate{SymbolID: symbol.SymbolID, FileID: fileID, FilePath: parsed.Path},
		)
	}
```
改为：
```go
	// 收集符号到候选集（累积，不覆盖）。
	// 递归展平 Children：Java/Kotlin 解析器把方法/构造器放在类的 Children 字段里，
	// 需要把它们也作为独立符号加入候选集，否则跨文件方法调用边无法消解。
	for _, parsedSymbol := range parsed.Symbols {
		m.collectSymbolRecursive(parsedSymbol, fileID, parsed.Path, file)
	}
```

- [ ] **在 mapper.go 新增 collectSymbolRecursive 方法（紧接 CollectSymbols 之后）**

```go
// collectSymbolRecursive 递归收集符号到候选集和 file.Symbols。
// 方法/构造器/字段等 Children 符号也作为独立符号加入，使跨文件方法调用边可消解。
func (m *SchemaMapper) collectSymbolRecursive(parsedSymbol parser.ParsedSymbol, fileID, filePath string, file *File) {
	symbol := m.mapSymbol(parsedSymbol, fileID)
	file.Symbols = append(file.Symbols, symbol)
	m.symbolIDMap[parsedSymbol.Name] = symbol.SymbolID // 向后兼容
	m.symbolCandidates[parsedSymbol.Name] = append(
		m.symbolCandidates[parsedSymbol.Name],
		symbolCandidate{SymbolID: symbol.SymbolID, FileID: fileID, FilePath: filePath},
	)

	// 递归展平子符号
	for _, child := range parsedSymbol.Children {
		m.collectSymbolRecursive(child, fileID, filePath, file)
	}
}
```

- [ ] **运行测试验证通过**

Run: `go test ./internal/schema/ -run TestCollectSymbols_FlattenChildren -v`
Expected: PASS

### Step 3: 运行 schema 包全量单元测试确保不退化

- [ ] **运行 schema 全量测试**

Run: `go test ./internal/schema/... -v`
Expected: PASS（含现有 mapper_test、mapper_cross_file_test 等）

### Step 4: indexRealFixtures 加 Java/Kotlin 多文件 fixture

- [ ] **修改 `tests/integration/quality_gate_test.go` 的 fixtureFiles 列表（第 115-126 行）**

在列表末尾（`typescript_calls_js.ts` 之后）追加：
```go
		{"tests/fixtures/kotlin/kotlin_calls_java.kt", "kotlin"},
		{"tests/fixtures/swift/swift_calls_objc.swift", "swift"},
		{"tests/fixtures/js/typescript_calls_js.ts", "js"},
		// 多文件 fixture：验证 Java/Kotlin 方法符号展平 + 跨文件方法调用边消解
		{"tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/model/User.kt", "kotlin"},
		{"tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/repository/UserRepository.kt", "kotlin"},
		{"tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt", "kotlin"},
		{"tests/fixtures/java/src/main/java/com/example/myapp/model/User.java", "java"},
		{"tests/fixtures/java/src/main/java/com/example/myapp/repository/UserRepository.java", "java"},
	}
```

注意：先检查 `tests/fixtures/java/src/main/java/com/example/myapp/model/User.java` 是否存在：
Run: `ls tests/fixtures/java/src/main/java/com/example/myapp/model/`
如不存在，检查实际路径并调整。

- [ ] **编译验证**

Run: `go build ./tests/integration/...`
Expected: 编译通过

### Step 5: 新增 Java/Kotlin 方法级真值

- [ ] **在 `internal/quality/fixtures/graph_ground_truth.go` 的 CallAnalysisGroundTruth 列表末尾追加**

```go
	// ──────────────────────────────────────────────────────────────
	// 6. UserService.kt —— Kotlin 跨文件方法调用。
	//    UserService 的方法调用 UserRepository 的方法（跨文件）。
	//    方法符号展平后，这些调用边的 target_id 应能消解到 UserRepository.kt 的方法符号。
	//    Optional 的边为 Kotlin 标准库调用（toList/find/removeIf 等，源文件无定义）。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt",
		Edges: []quality.ExpectedEdge{
			{SourceName: "getAllUsers", EdgeType: "call", TargetName: "findAll"},
			{SourceName: "getUserById", EdgeType: "call", TargetName: "findById"},
			{SourceName: "createUser", EdgeType: "call", TargetName: "save"},
			{SourceName: "deleteUser", EdgeType: "call", TargetName: "delete"},
			// Kotlin 标准库调用（悬空）
			{SourceName: "getAllUsers", EdgeType: "call", TargetName: "", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			{StartName: "getAllUsers", EndName: "findAll",
				StartFile: "tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt",
				EndFile:   "tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/repository/UserRepository.kt"},
		},
	},
	// ──────────────────────────────────────────────────────────────
	// 7. UserRepository.java —— Java 方法间调用（同文件内）。
	//    findById 调用 getId（User 的 getter），但 User 类在另一文件。
	//    方法符号展平后可验证 Java 方法级符号入库。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/java/src/main/java/com/example/myapp/repository/UserRepository.java",
		Edges: []quality.ExpectedEdge{
			// findById 内部调用 user.getId()（跨文件到 User.java）
			{SourceName: "findById", EdgeType: "call", TargetName: "getId"},
			// delete 内部调用 user.getId()
			{SourceName: "delete", EdgeType: "call", TargetName: "getId"},
			// Java 标准库调用（悬空）
			{SourceName: "findAll", EdgeType: "call", TargetName: "", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			{StartName: "findById", EndName: "getId",
				StartFile: "tests/fixtures/java/src/main/java/com/example/myapp/repository/UserRepository.java",
				EndFile:   "tests/fixtures/java/src/main/java/com/example/myapp/model/User.java"},
		},
	},
```

注意：实现时需用 `ls tests/fixtures/java/src/main/java/com/example/myapp/model/` 确认 `User.java` 存在且含 `getId` 方法。如路径或方法名不符，按实际 fixture 代码调整真值。

### Step 6: 运行集成测试验证方法符号展平 + 边消解

- [ ] **运行集成测试（需 DB）**

Run: `make test-integration 2>&1 | grep -E "TestQualityGate|FAIL|PASS|recall|precision"`
Expected: TestQualityGate_FixtureMode 和 TestQualityGate_RepoMode 通过

如果失败，检查：
- `t.Logf` 输出里 Java/Kotlin 方法符号是否出现
- 跨文件方法调用边的 target_id 是否非空
- 真值匹配是否因 symbol_id 回填失败而 recall=0

- [ ] **如真值 symbol_id 回填失败（recall=0），调试 ResolveTruthIDs**

在 `ResolveTruthIDs` 里加临时日志：
```go
fmt.Printf("DEBUG: edge %s|%s|%s -> SourceID=%s TargetID=%s\n",
    edge.SourceName, edge.EdgeType, edge.TargetName, edge.SourceID, edge.TargetID)
```
检查哪些边没查到 symbol_id。常见原因：方法符号 name 与真值 name 不一致（如类前缀差异）。

- [ ] **提交**

```bash
git add internal/schema/mapper.go internal/schema/mapper_test.go \
  tests/integration/quality_gate_test.go \
  internal/quality/fixtures/graph_ground_truth.go \
  internal/quality/fixtures/graph_ground_truth_test.go
git commit -m "feat(schema): CollectSymbols 递归展平 Children——Java/Kotlin 方法符号入候选集"
```

---

## Task 2.5（可选）：Java/Kotlin 检索真值补充

**前提**：需要 embedding 环境（Ollama / OpenAI）。如当前无 embedding 环境，可跳过此 Task，不影响依赖图门禁。方法符号展平后符号已入库，检索真值补充只是扩大评估覆盖面。

**Files:**
- Modify: `internal/quality/fixtures/retrieval_ground_truth.go`
- Test: `internal/quality/fixtures/retrieval_ground_truth_test.go`

### Step 1: 新增 Java/Kotlin 方法级检索真值

- [ ] **在 `internal/quality/fixtures/retrieval_ground_truth.go` 的 RetrievalGroundTruth 列表末尾追加**

```go
	// ──────────────────────────────────────────────────────────────
	// 单语言 3（Java）：UserRepository 方法级检索
	// ──────────────────────────────────────────────────────────────
	{
		Query: "如何根据 ID 查找用户",
		RelevantSymbols: []string{
			"findById",
			"UserRepository",
		},
		RelevantFiles: []string{
			"tests/fixtures/java/src/main/java/com/example/myapp/repository/UserRepository.java",
		},
		Repos: []string{"UserRepository.java"},
	},
	// ──────────────────────────────────────────────────────────────
	// 单语言 4（Kotlin）：UserService 方法级检索
	// ──────────────────────────────────────────────────────────────
	{
		Query: "如何创建和删除用户",
		RelevantSymbols: []string{
			"createUser",
			"deleteUser",
			"UserService",
		},
		RelevantFiles: []string{
			"tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt",
		},
		Repos: []string{"UserService.kt"},
	},
```

- [ ] **如果有 embedding 环境，运行检索评估**

Run: `make test-integration 2>&1 | grep -E "retrieval|recall|Java|Kotlin"`
Expected: Java/Kotlin 方法级符号在 recall 结果中出现

- [ ] **如无 embedding 环境，跳过运行，仅提交真值数据**

```bash
git add internal/quality/fixtures/retrieval_ground_truth.go \
  internal/quality/fixtures/retrieval_ground_truth_test.go
git commit -m "test(quality): 补 Java/Kotlin 方法级检索真值"
```

---

## Task 3：重建基线 + 启用硬门禁

**Files:**
- Modify: `internal/quality/metrics.go:103-110`
- Modify: `internal/quality/graph_evaluator.go:98-166`
- Modify: `tests/integration/quality_gate_test.go`
- Create: `docs/superpowers/baselines/2026-07-08-precision-graph-v2-baseline.md`

### Step 1: 跑新基线，记录 Task1/Task2 前后变化

- [ ] **运行集成测试，打印完整指标**

Run: `make test-integration 2>&1 | grep -E "Fixture 模式报告|RepoMode|dangling|orphan|cross_file|symbol_resolution|recall|precision|connectivity"`
Expected: 输出所有指标值

- [ ] **记录指标到临时文件**

把输出里每个指标的 `name (bucket=X) = Y threshold=Z passed=P` 记录下来，供下一步定阈值用。

### Step 2: 新增分桶阈值常量

- [ ] **修改 `internal/quality/metrics.go` 的阈值常量区（第 103-110 行）**

把：
```go
// 结构断言类建议基线：这轮仅观察、Threshold=0（不做硬门禁）。
// Evaluate 当前不使用这些常量；下一轮跑出基线后启用为硬门禁阈值。
const (
	ThresholdDanglingEdgeRatio     = 0.30 // 建议值，下一轮启用
	ThresholdSymbolResolution      = 0.70 // 建议值，下一轮启用
	ThresholdOrphanSymbolRatio     = 0.40 // 建议值，下一轮启用
	ThresholdCrossFileConnectivity = 0.20 // 建议值，下一轮启用
)
```
改为：
```go
// 结构断言类硬门禁阈值。
// 总值（不分桶）保持 Threshold=0（仅观察），因 import 边 100% 悬空会拖低总值。
// 分桶阈值只对 call/reference 类设硬门禁；import 类悬空符合预期，保持观察。
//
// 阈值依据：2026-07-08 基线值 ± 安全边际（见 baselines/2026-07-08-precision-graph-v2-baseline.md）。
const (
	ThresholdDanglingEdgeRatioCall      = 0.60 // 基线 0.55，留 5% 安全边际
	ThresholdDanglingEdgeRatioReference = 0.10 // 基线 0.00
	ThresholdSymbolResolutionCall       = 0.40 // 基线 0.45，留 5% 安全边际
	ThresholdOrphanSymbolRatio          = 0.40 // 基线 0.31（Task2 后方法符号降低孤立率）
	ThresholdCrossFileConnectivity      = 0.20 // 基线 0.22
)
```

注：具体阈值按 Step 1 跑出的真实基线值调整，保持"基线值 ± 5% 安全边际"。

### Step 3: graph_evaluator 启用分桶硬门禁

- [ ] **修改 `internal/quality/graph_evaluator.go` 的分桶逻辑（第 112-134 行）**

在分桶循环里，把 `Threshold: 0` 改为按 edge_type 设阈值。把：
```go
			bv := MetricValue{
				Name: "dangling_edge_ratio", Category: CategoryGraph,
				Value: float64(d) / float64(t), Bucket: et,
				Threshold: 0, HigherIsBetter: false, // 分桶仅观察
			}
```
改为：
```go
			// 分桶硬门禁：call/reference 类设阈值，import 类保持观察
			bucketThreshold := 0.0
			switch et {
			case "call":
				bucketThreshold = ThresholdDanglingEdgeRatioCall
			case "reference":
				bucketThreshold = ThresholdDanglingEdgeRatioReference
			}
			bv := MetricValue{
				Name: "dangling_edge_ratio", Category: CategoryGraph,
				Value: float64(d) / float64(t), Bucket: et,
				Threshold: bucketThreshold, HigherIsBetter: false,
			}
```

- [ ] **新增 symbol_resolution_rate 的 call 分桶硬门禁**

在 symbol_resolution_rate 总值之后（第 137-143 行），新增 call 分桶。把：
```go
		// 符号消解率（1 - 悬空边率）
		res := MetricValue{
			Name: "symbol_resolution_rate", Category: CategoryGraph,
			Value:     1 - float64(totalDangling)/float64(totalEdges),
			Threshold: 0, HigherIsBetter: true, // 仅观察（下一轮启用 ThresholdSymbolResolution）
		}
		res.EvaluatePassed()
		metrics = append(metrics, res)
```
改为：
```go
		// 符号消解率（总值仅观察，因 import 边 100% 悬空拖低总值）
		res := MetricValue{
			Name: "symbol_resolution_rate", Category: CategoryGraph,
			Value:     1 - float64(totalDangling)/float64(totalEdges),
			Threshold: 0, HigherIsBetter: true,
		}
		res.EvaluatePassed()
		metrics = append(metrics, res)

		// call 分桶硬门禁：call 类消解率应达标
		if callTotal := byType["call"]; callTotal > 0 {
			callDangling := dangling["call"]
			callRes := MetricValue{
				Name: "symbol_resolution_rate", Category: CategoryGraph,
				Value:     1 - float64(callDangling)/float64(callTotal),
				Bucket:    "call",
				Threshold: ThresholdSymbolResolutionCall, HigherIsBetter: true,
			}
			callRes.EvaluatePassed()
			metrics = append(metrics, callRes)
		}
```

- [ ] **启用 orphan_symbol_ratio 和 cross_file_connectivity 硬门禁**

把 orphan（第 147-155 行）的：
```go
			Threshold: 0, HigherIsBetter: false, // 仅观察（下一轮启用 ThresholdOrphanSymbolRatio）
```
改为：
```go
			Threshold: ThresholdOrphanSymbolRatio, HigherIsBetter: false,
```

把 cross_file（第 158-166 行）的：
```go
			Threshold: 0, HigherIsBetter: true, // 仅观察（下一轮启用 ThresholdCrossFileConnectivity）
```
改为：
```go
			Threshold: ThresholdCrossFileConnectivity, HigherIsBetter: true,
```

- [ ] **编译验证**

Run: `go build ./internal/quality/...`
Expected: 编译通过

### Step 4: 运行单元测试确保阈值常量正确

- [ ] **运行 quality 单元测试**

Run: `go test ./internal/quality/... -v`
Expected: PASS

### Step 5: 运行集成测试，门禁应全绿

- [ ] **运行集成测试**

Run: `make test-integration 2>&1 | grep -E "TestQualityGate|质量门禁失败|FAIL"`
Expected: TestQualityGate_FixtureMode 和 TestQualityGate_RepoMode 通过，无"质量门禁失败"

如果有指标不达标：
- 检查 Step 1 记录的基线值，调整阈值（放宽 5%）
- 检查是否 Task 2 的方法符号补充导致数据变化（重新跑 Step 1 记录新基线）

### Step 6: 写新基线文档

- [ ] **创建 `docs/superpowers/baselines/2026-07-08-precision-graph-v2-baseline.md`**

```markdown
# CodeAtlas 质量基线（2026-07-08 精准依赖图 v2）

本基线由 `TestQualityGate_FixtureMode` 和 `TestQualityGate_RepoMode` 在 fixture 数据上跑出，
记录 Task 1（symbol_id 匹配）和 Task 2（方法符号展平）改造前后的指标变化。

## Task 1：symbol_id 匹配（修复 C++ recall>1.0）

| 指标 | 改造前 | 改造后 | 说明 |
|---|---|---|---|
| `edge_recall` | 1.00 | （填入实际值） | symbol_id 匹配后 |
| `edge_precision` | 1.00 | （填入实际值） | |
| C++ recall | 1.33 | ≤1.0 | 修复重载虚高 |

## Task 2：Java/Kotlin 方法符号展平

| 指标 | 改造前 | 改造后 | 说明 |
|---|---|---|---|
| Java 符号数 | （仅类级） | （填入实际值） | 方法符号入库 |
| Kotlin 符号数 | （仅类级） | （填入实际值） | 方法符号入库 |
| `cross_file_connectivity` | 0.22 | （填入实际值） | 方法调用边消解 |
| `orphan_symbol_ratio` | 0.31 | （填入实际值） | 方法符号降低孤立率 |

## Task 3：结构断言硬门禁

| 指标 | 分桶 | 基线值 | 阈值 | 通过 |
|---|---|---|---|---|
| `dangling_edge_ratio` | call | （填入） | ≤0.60 | ✓/✗ |
| `dangling_edge_ratio` | import | （填入） | 观察 | — |
| `dangling_edge_ratio` | reference | （填入） | ≤0.10 | ✓/✗ |
| `symbol_resolution_rate` | call | （填入） | ≥0.40 | ✓/✗ |
| `symbol_resolution_rate` | import | （填入） | 观察 | — |
| `orphan_symbol_ratio` | 总体 | （填入） | ≤0.40 | ✓/✗ |
| `cross_file_connectivity` | 总体 | （填入） | ≥0.20 | ✓/✗ |

## fixture 真值类（门禁）

| 指标 | 值 | 阈值 | 通过 |
|---|---|---|---|
| `edge_recall` | （填入） | ≥0.90 | ✓ |
| `edge_precision` | （填入） | ≥0.85 | ✓ |
| `call_chain_connectivity` | （填入） | ≥0.95 | ✓ |

注：（填入）处用 Step 1 跑出的真实数据替换。
```

- [ ] **用真实数据填充基线文档**

从 Step 1 和 Step 5 的测试输出里取真实值，替换所有（填入）。

### Step 7: 全量回归 + 提交

- [ ] **运行全量测试**

Run: `make test && make test-integration`
Expected: 全绿

- [ ] **提交**

```bash
git add internal/quality/metrics.go internal/quality/graph_evaluator.go \
  tests/integration/quality_gate_test.go \
  docs/superpowers/baselines/2026-07-08-precision-graph-v2-baseline.md
git commit -m "feat(quality): 启用结构断言硬门禁——分桶阈值 + 新基线"
```

### Step 8: 更新 evaluation.md 文档

- [ ] **修改 `docs/evaluation.md` 的门禁机制章节（第 64-67 行）**

把：
```markdown
- **结构断言类**：当前版本仅建基线，不做硬门禁。下一轮有基线数据后收紧为硬门禁
```
改为：
```markdown
- **结构断言类**：分桶硬门禁已启用。dangling_edge_ratio/symbol_resolution_rate 按 edge_type 分桶，call/reference 类设阈值，import 类保持观察（悬空符合预期）。orphan_symbol_ratio/cross_file_connectivity 总值设阈值。阈值依据见 `docs/superpowers/baselines/2026-07-08-precision-graph-v2-baseline.md`
```

- [ ] **提交**

```bash
git add docs/evaluation.md
git commit -m "docs(quality): 更新门禁机制说明——结构断言硬门禁已启用"
```

---

## 验收清单

- [ ] C++ `recall@10` ≤ 1.0（Task 1）
- [ ] Java/Kotlin 方法符号出现在 symbols 表（Task 2）
- [ ] 跨文件方法调用边 target_id 非空（Task 2）
- [ ] 结构断言硬门禁对 call 类 dangling/resolution 生效（Task 3）
- [ ] `make test && make test-integration` 全绿
- [ ] 新基线文档完整记录前后对比（Task 3）
- [ ] `docs/evaluation.md` 门禁说明已更新（Task 3）
