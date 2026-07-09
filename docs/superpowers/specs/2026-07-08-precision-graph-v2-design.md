# 精准依赖图 v2 设计（Precision Dependency Graph v2）

- **日期**: 2026-07-08
- **状态**: 已批准（设计阶段），待写实现计划
- **范围**: 修复 C++ 重载导致 recall>1.0 的评估器缺陷；展平 Java/Kotlin 方法符号到符号表；基于新基线启用结构断言硬门禁
- **不做**: 数据库 schema 变更、API 改动、前端改动、Swift/ObjC/C++ 的 Children 展平、signature 匹配

---

## 1. 背景与目标

上一轮迭代（PR#4 跨文件符号消解）完成了 SchemaMapper 两遍扫描，`cross_file_connectivity` 从 0.00 提升到 0.22，门禁全绿。基线文档（`docs/superpowers/baselines/2026-07-07-quality-baseline.md`）明确列出了三个"下一轮建议"：

1. 结构断言类指标（dangling/orphan/cross_file/symbol_resolution）从"仅观察"升级为硬门禁
2. edge_recall/precision 匹配策略从纯 name 升级为 symbol_id，解决 C++ 重载 recall>1.0
3. Java/Kotlin 解析器补方法符号提取，当前仅类级符号限制检索评估覆盖面

本轮迭代把这三件事按依赖顺序串联完成，让评估器准确、符号集完整、门禁可信。

### 1.1 成功标准

1. C++ `recall@10` ≤ 1.0（修复前 1.33）
2. Java/Kotlin 方法符号出现在 `symbols` 表，跨文件方法调用边可消解
3. 结构断言硬门禁对 call 类 dangling/resolution 生效，CI 全绿
4. 现有语言（Go/Python/Swift 等）recall/precision 不退化
5. 新基线文档记录完整前后对比

---

## 2. 整体架构与 Task 依赖

三个 Task 有真实的数据依赖关系，必须串联：

```
Task 1: 评估器 symbol_id 匹配
  └─ computeEdgeMatch 从 name 三元组 → symbol_id 三元组
  └─ ListExtractedEdges 扩展返回 SourceID/TargetID
  └─ ExpectedEdge 真值增加 SourceID/TargetID 字段
  └─ 验证：C++ recall ≤ 1.0，现有 recall/precision 不退化
        ↓ （评估器变准确，基线可信）
Task 2: Java/Kotlin 方法符号展平
  └─ CollectSymbols 递归展平 ParsedSymbol.Children
  └─ 方法/构造器作为独立符号进入 symbolCandidates
  └─ 补 Java/Kotlin fixture 方法级真值
  └─ 验证：跨文件方法调用边可消解，检索评估覆盖方法级
        ↓ （符号集扩大，基线数据变化）
Task 3: 重建基线 + 启用硬门禁
  └─ 跑新基线，记录 Task1/Task2 前后变化
  └─ dangling/symbol_resolution 按 edge_type 分桶，只对 call 类设阈值
  └─ orphan/cross_file 按新基线值设阈值（留安全边际）
  └─ import 类 dangling 保持 Threshold=0（悬空符合预期）
  └─ 验证：门禁全绿，CI 硬门禁生效
```

**为什么必须串联**：
- Task 3 依赖准确的基线数据 → 如果评估器有 bug（Task 1 未修），基线不可信
- Task 2 会改变符号数量和边数据 → 如果先定门禁阈值再补方法符号，基线会漂移，阈值要重定
- Task 1 和 Task 2 看似独立，但都影响评估输入（一个改匹配 key，一个改 extracted 边集），并行时回归定位困难

**改动范围**：评估器（`internal/quality/`）+ 解析器映射（`internal/schema/mapper.go`）+ fixture 真值（`internal/quality/fixtures/`）+ 门禁阈值常量（`internal/quality/metrics.go`）。不碰数据库 schema、不碰 API、不碰前端。

---

## 3. Task 1：评估器 symbol_id 匹配

### 3.1 问题根因

`computeEdgeMatch`（`internal/quality/graph_evaluator.go:214`）用 `SourceName|EdgeType|TargetName` 做匹配 key。C++ 重载（如 `MyClass::MyClass` 构造函数）在 `symbols` 表里有多行同名记录，`ListExtractedEdges` 的 JOIN 产生多条相同 (name, type, name) 边 → recall 分母按真值条数、分子按命中条数，但 extractedSet 去重后只剩一条 → recall 被虚高到 1.33。

### 3.2 改动点

**3.2.1 `ExtractedEdge` 结构扩展**（`internal/quality/graph_evaluator.go`）：

```go
type ExtractedEdge struct {
    SourceID   string  // 新增
    SourceName string  // 保留：调试用
    EdgeType   string
    TargetID   string  // 新增（悬空时为空）
    TargetName string  // 保留：调试用
}
```

**3.2.2 `ListExtractedEdges` 查询扩展**（`pkg/models/graph_metrics.go`）：

SQL 增加 `e.source_id, COALESCE(e.target_id, '')` 两列，`models.ExtractedEdge` 结构同步加 `SourceID/TargetID` 字段。`DefaultGraphFetcher.ListExtractedEdges` 透传新字段。

**3.2.3 `ExpectedEdge` 真值扩展**（`internal/quality/graph_evaluator.go`）：

```go
type ExpectedEdge struct {
    SourceID   string  // 新增：真值边的源 symbol_id
    SourceName string  // 保留：人类可读，调试用
    EdgeType   string
    TargetID   string  // 新增
    TargetName string  // 保留
    Optional   bool
}
```

真值数据里 `SourceID/TargetID` 不硬编码——因为 symbol_id 虽然是确定性的（`GenerateDeterministicUUID`），但硬编码会脆弱。改为在 `quality_gate_test.go` 里索引 fixture 后、跑评估前，通过 `SymbolRepository` 按名字+文件查出 symbol_id 回填到 `ExpectedEdge`。封装一个 `fixtures.ResolveTruthIDs(symbolRepo, truth)` 辅助函数。

**3.2.4 `computeEdgeMatch` 改为 symbol_id 匹配**：

- key 从 `SourceName|EdgeType|TargetName` → `SourceID|EdgeType|TargetID`
- 悬空边（TargetID 为空）不参与 recall/precision 计算（保持现有行为：悬空边无法匹配真值）

### 3.3 验证标准

- C++ `recall@10` ≤ 1.0（修复前 1.33）
- 现有 Go/Python/Swift 等语言的 recall/precision 不退化
- `edge_recall` 仍 ≥ 0.90，`edge_precision` 仍 ≥ 0.85

### 3.4 不做的事（YAGNI）

- ❌ 不改 signature 匹配（symbol_id 已根本解决重名问题，signature 信息不必要）
- ❌ 不改 `CheckCallChainConnectivity`（它已经用 name+file 匹配，重载场景下链路连通性不受 recall>1.0 影响）

---

## 4. Task 2：Java/Kotlin 方法符号展平

### 4.1 问题根因

`CollectSymbols`（`internal/schema/mapper.go:76-84`）只遍历 `parsed.Symbols` 顶层，方法符号嵌在 `ParsedSymbol.Children` 字段里，**没有递归收集进 `symbolCandidates`**。导致：

- 跨文件调用 Java/Kotlin 方法时，目标方法不在候选集 → 调用边消解失败
- 检索评估只能验证类级召回，方法级符号根本不在索引里

尽管 Java/Kotlin 解析器已有 `extractClassMethods`/`extractConstructors`/`extractInterfaceMethods` 方法（能正确提取方法符号），但提取结果只赋给 `ParsedSymbol.Children`，从未进入 `parsedFile.Symbols` 顶层。

### 4.2 改动点

**4.2.1 `CollectSymbols` 递归展平 Children**（`internal/schema/mapper.go`）：

```go
for _, parsedSymbol := range parsed.Symbols {
    m.collectSymbolRecursive(parsedSymbol, fileID, parsed.Path)
}
```

新增 `collectSymbolRecursive` 方法：把符号加入 `file.Symbols` 和 `symbolCandidates`，然后对 `parsedSymbol.Children` 递归调用。方法/构造器/字段都作为独立符号进入候选集。

**4.2.2 方法符号的命名对齐**：

需检查 `extractClassMethods` 产出的 name 格式是否与调用边 target 的 `Source`/`Target` 字段对齐。如果方法符号 name 是 `methodName`，而调用边 target 是 `ClassName.methodName`，消解会失败。实现时验证并调整，确保一致。

**4.2.3 Java/Kotlin fixture 方法级真值补充**（`internal/quality/fixtures/graph_ground_truth.go`）：

- 从现有 fixture 代码（如 `UserRepository.java`、`UserService.kt`）提取方法级调用关系
- 补充 `ExpectedEdge`：如 `UserRepository.findById` → `User`（reference 边）
- 补充 `ExpectedChain`：跨文件方法调用链

**4.2.4 检索真值补充**（`internal/quality/fixtures/retrieval_ground_truth.go`）：

给 Java/Kotlin 补方法级 query 真值，验证方法级 recall。

### 4.3 影响面排查

| 组件 | 影响 |
|---|---|
| `writer.go` | 展平后符号数增加，写入量增大，批量写入逻辑不变 |
| `graph_builder.go` | 节点数增加，构建逻辑不变 |
| `validator.go` | 需确认方法符号通过验证（预期无新约束） |
| `orphan_symbol_ratio` | 方法符号有出入边后不再孤立，值会下降——Task 3 重建基线 |

### 4.4 验证标准

- Java/Kotlin 方法符号出现在 `symbols` 表（通过 `ListExtractedEdges` 验证）
- 跨文件方法调用边 `target_id` 非空（消解成功）
- 现有语言 recall/precision 不退化
- 新增方法级真值门禁通过

### 4.5 不做的事（YAGNI）

- ❌ 不改 Swift/ObjC/C++ 的 Children 展平（基线未显示它们有此问题；实现时 grep 确认，但本轮不主动扩大范围）
- ❌ 不改解析器本身的 `extractClassMethods` 逻辑（它们已经能提取方法，只是没被展平进符号表）

---

## 5. Task 3：重建基线 + 启用硬门禁

### 5.1 前提

Task 1（评估器准确）和 Task 2（符号集扩大）完成后，基线数据会变化，需重新跑一次建立新基线。

### 5.2 改动点

**5.2.1 跑新基线**（`docs/superpowers/baselines/2026-07-08-precision-graph-v2-baseline.md`）：

记录 Task 1/Task 2 前后对比表，重点观察：
- C++ recall 回归 ≤1.0
- Java/Kotlin 符号数增加
- 方法级边消解后 `cross_file_connectivity` 和 `orphan_symbol_ratio` 变化

**5.2.2 门禁阈值策略**（`internal/quality/metrics.go` + `graph_evaluator.go`）：

dangling/symbol_resolution 按 edge_type 分桶，只对 call 类设硬门禁；import 类悬空符合预期，保持观察。orphan/cross_file 按新基线值设阈值，留 5-10% 安全边际。

| 指标 | 分桶 | 阈值（预估） | 依据 |
|---|---|---|---|
| `dangling_edge_ratio` | call | ≤ 0.60 | 基线 0.55（外部库函数悬空），留 5% 安全边际 |
| `dangling_edge_ratio` | import | **Threshold=0**（观察） | 外部依赖悬空符合预期 |
| `dangling_edge_ratio` | reference | ≤ 0.10 | 基线 0.00 |
| `symbol_resolution_rate` | call | ≥ 0.40 | 基线 0.45，留 5% 安全边际 |
| `symbol_resolution_rate` | import | **Threshold=0**（观察） | 被 100% 悬空拖累，不设门禁 |
| `orphan_symbol_ratio` | 总体 | ≤ 0.40 | 基线 0.31，Task2 后方法符号会进一步降低 |
| `cross_file_connectivity` | 总体 | ≥ 0.20 | 基线 0.22，已达标 |

注：具体阈值在新基线跑出后微调，上表是预估值。

**5.2.3 `graph_evaluator.go` 启用阈值**：

- 把 call 类分桶的 `Threshold: 0` 改为对应常量值
- 新增 `ThresholdDanglingEdgeRatioCall`、`ThresholdSymbolResolutionCall` 等分桶常量
- import 类分桶保持 `Threshold=0`

**5.2.4 门禁测试对齐**（`tests/integration/quality_gate_test.go`）：

- `TestQualityGate_FixtureMode`：新阈值下全绿
- `TestQualityGate_RepoMode`：记录新基线值
- 如果某指标因 Task 2 方法符号补充而变化，调整断言

### 5.3 验证标准

- `make test-integration` 全绿
- 新基线文档记录完整前后对比
- CI 硬门禁对 call 类 dangling/resolution 生效

### 5.4 风险与缓解

| 风险 | 缓解 |
|---|---|
| 新基线跑出后某指标意外不达标 | 回溯 Task 2 真值补充是否引入错误边，或调整阈值 |
| 阈值定太紧导致 flaky | 留 5-10% 安全边际 |

---

## 6. 测试策略

| 层 | 测试文件 | 类型 | 覆盖 |
|---|---|---|---|
| quality | `graph_evaluator_test.go`（扩展） | 单元 | symbol_id 匹配、C++ 重载去重、悬空边排除 |
| models | `graph_metrics_test.go`（扩展） | 单元 | ListExtractedEdges 返回 SourceID/TargetID |
| schema | `mapper_test.go`（扩展） | 单元 | CollectSymbols 递归展平 Children、方法符号进候选集 |
| fixtures | `graph_ground_truth_test.go`（扩展） | 单元 | Java/Kotlin 方法级真值、ResolveTruthIDs 回填 |
| 集成 | `quality_gate_test.go`（改） | 集成 | 新阈值门禁、方法级边消解、基线对比 |
| 集成 | 全量回归 | 集成 | `make test-integration` 确保现有边数/符号数测试不误判 |

**关键测试用例**：
- `TestComputeEdgeMatch_SymbolID_Dedup`：C++ 重载同名符号 → recall ≤ 1.0
- `TestComputeEdgeMatch_DanglingExcluded`：悬空边（TargetID 空）不参与匹配
- `TestCollectSymbols_FlattenChildren`：方法/构造器作为独立符号进入候选集
- `TestResolveEdges_CrossFileMethodCall`：Java/Kotlin 跨文件方法调用边消解成功
- `TestQualityGate_HardThreshold_CallBucket`：call 类 dangling/resolution 硬门禁生效

---

## 7. 不做的事（YAGNI）

- ❌ signature 匹配（symbol_id 已根本解决重名问题）
- ❌ Swift/ObjC/C++ 的 Children 展平（基线未显示此问题）
- ❌ 数据库 schema 变更（symbol_id 已在 edges 表里）
- ❌ API 改动（评估器内部变化不暴露给 API）
- ❌ 前端改动
- ❌ `CheckCallChainConnectivity` 改造（不受 recall>1.0 影响）

---

## 8. 风险与缓解

| 风险 | 缓解 |
|---|---|
| 方法符号 name 与调用边 target 不对齐导致消解失败 | 实现时先验证 name 格式一致性，不一致则调整解析器或 mapper |
| 展平 Children 后符号数激增影响性能 | fixture 数据量小，真实仓库需观察；批量写入逻辑已有批处理优化 |
| 真值 symbol_id 回填逻辑复杂 | 封装为 `fixtures.ResolveTruthIDs` 辅助函数，单元测试覆盖 |
| 现有测试因符号数/边数变化而断言失败 | 现有测试断言"至少有"或"类型正确"，非精确数量；精确断言按实际调整 |
| 新基线某指标意外不达标 | 回溯 Task 2 真值或调整阈值，留安全边际 |
