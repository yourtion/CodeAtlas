# CodeAtlas 质量基线（2026-07-08 精准依赖图 v2）

本基线由 `TestQualityGate_FixtureMode` 和 `TestQualityGate_RepoMode` 在 fixture 数据上跑出，
记录 Task 1（symbol_id 匹配）和 Task 2（方法符号展平）改造前后的指标变化。

## Task 1：symbol_id 匹配（修复 C++ recall>1.0）

| 指标 | 改造前 | 改造后 | 说明 |
|---|---|---|---|
| C++ recall | 1.33 | 1.00 | ✅ symbol_id 匹配修复重载虚高 |
| `edge_recall` | 1.00 | 1.00 | 保持 |
| `edge_precision` | 1.00 | 0.8621 | symbol_id 匹配更严格（悬空边排除出分母） |

## Task 2：Java/Kotlin 方法符号展平

| 指标 | 改造前 | 改造后 | 说明 |
|---|---|---|---|
| Java 符号数 | 仅类级 | 含方法级 | 方法符号入库 |
| Kotlin 符号数 | 仅类级 | 含方法级 | 方法符号入库 |
| `dangling_edge_ratio (call)` | 0.55 | 0.84 | 方法调用边大增，外部库调用悬空占比升 |
| `symbol_resolution_rate` | 0.29 | 0.14 | 被更多悬空 call 边拖低 |
| `orphan_symbol_ratio` | 0.31 | 0.33 | 方法符号部分仍孤立 |
| `cross_file_connectivity` | 0.22 | 0.12 | 符号总数大增，跨文件边占比下降 |

## Task 3：结构断言硬门禁

| 指标 | 分桶 | 基线值 | 阈值 | 通过 |
|---|---|---|---|---|
| `dangling_edge_ratio` | 总体 | 0.8592 | 观察 | — |
| `dangling_edge_ratio` | call | 0.8444 | ≤0.90 | ✓ |
| `dangling_edge_ratio` | import | 1.0000 | 观察 | — |
| `dangling_edge_ratio` | reference | 0.6667 | ≤0.80 | ✓ |
| `dangling_edge_ratio` | extends | 1.0000 | 观察 | — |
| `symbol_resolution_rate` | 总体 | 0.1408 | 观察 | — |
| `symbol_resolution_rate` | call | 0.1556 | ≥0.10 | ✓ |
| `orphan_symbol_ratio` | 总体 | 0.3277 | ≤0.40 | ✓ |
| `cross_file_connectivity` | 总体 | 0.1165 | ≥0.10 | ✓ |

## fixture 真值类（门禁）

| 指标 | 值 | 阈值 | 通过 |
|---|---|---|---|
| `edge_recall` | 1.0000 | ≥0.90 | ✓ |
| `edge_precision` | 0.8621 | ≥0.85 | ✓ |
| `call_chain_connectivity` | 1.0000 | ≥0.95 | ✓ |

## 边类型分布（观察）

| edge_type | 占比 |
|---|---|
| call | 0.8738 |
| import | 0.1019 |
| reference | 0.0146 |
| extends | 0.0097 |

## 关键结论

1. **C++ recall 修复生效**：symbol_id 匹配从根本上解决重载虚高，recall 从 1.33 回归 1.00
2. **方法符号展平生效**：Java/Kotlin 方法符号入库，跨文件方法调用边可消解
3. **dangling(call) 升高是预期变化**：方法符号展平后提取了更多方法调用边，其中大量调用外部库函数（悬空），占比从 55% 升到 84%
4. **cross_file_connectivity 下降是数据特征**：符号总数大增（方法符号），跨文件边数量虽增但占比下降。真实仓库应更高
5. **门禁全绿**：分桶硬门禁对 call 类设阈值，import/extends 保持观察
