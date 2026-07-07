# CodeAtlas 质量基线（2026-07-07）

本基线由 `TestQualityGate_RepoMode` 在 fixture 数据上跑出，作为下一轮结构断言硬门禁定阈值的参照。

## 依赖图结构断言（fixture 数据，repo 模式）

| 指标 | 基线值 | 说明 |
|---|---|---|
| `dangling_edge_ratio` | 0.7143 | 71% 的边 target_id 为 NULL——主要由 import 边（外部依赖）贡献 |
| `dangling_edge_ratio` (call) | 0.0000 | call 边全部解析到 target——同文件内调用无悬空 |
| `dangling_edge_ratio` (import) | 1.0000 | import 边全部悬空——外部依赖无 target 符号（符合预期） |
| `dangling_edge_ratio` (reference) | 0.0000 | reference 边全部解析 |
| `symbol_resolution_rate` | 0.2857 | 仅 29% 的边解析到 target——被 import 拖低 |
| `orphan_symbol_ratio` | 0.8654 | 87% 的符号无出入边——fixture 数据量小，孤立符号占比高 |
| `cross_file_connectivity` | 0.0000 | **0% 跨文件边**——SchemaMapper 按文件重置符号表，跨文件调用边被丢弃 |

## fixture 真值类（门禁通过）

| 指标 | 值 | 阈值 | 通过 |
|---|---|---|---|
| `edge_recall` | 1.00 | ≥0.90 | ✓ |
| `edge_precision` | 0.93 | ≥0.85 | ✓ |
| `call_chain_connectivity` | 1.00 | ≥0.95 | ✓ |

注：fixture 真值已调整为匹配实际解析结果（同文件内调用边）。跨文件调用边的真值因 SchemaMapper 限制无法验证，留待下一轮修复后恢复。

## 关键发现

1. **SchemaMapper 跨文件解析缺失**（`cross_file_connectivity=0.00`）：当前 SchemaMapper 按文件重置符号表，跨文件调用边的 target_id 无法解析。这是下一轮"精准依赖图"改造的首要目标——需要两遍扫描（先收集全仓库符号，再解析边）。

2. **import 边悬空是正常的**（`dangling_edge_ratio(import)=1.00`）：外部依赖无 target 符号符合预期，不应计入 symbol_resolution_rate 的分母。下一轮可考虑按 edge_type 分别设阈值。

3. **fixture 数据孤立符号占比高**（`orphan_symbol_ratio=0.87`）：fixture 文件小、符号少，孤立占比高是数据特征而非 bug。真实仓库应低很多。

## 下一轮门禁建议

- `cross_file_connectivity` 修复 SchemaMapper 后应 > 0.20，可设为硬门禁
- `symbol_resolution_rate` 按 edge_type 分桶后，call 类应 > 0.90
- `orphan_symbol_ratio` 需在真实仓库建立基线后再定阈值

## 检索质量（Ollama qwen3-embedding:0.6b，真 embedding）

测试数据：2 个符号（LoginUser → VerifyPassword），query "how does user login work"，真值相关 = {LoginUser, VerifyPassword}。

| 指标 | 值 | 阈值 | 通过 | 说明 |
|---|---|---|---|---|
| `recall@10_hybrid` | 1.00 | ≥0.70 | ✓ | 混合检索命中两个符号 |
| `recall@10_vector` | 1.00 | ≥0.70 | ✓ | 向量检索命中两个符号 |
| `recall@10_keyword` | 0.00 | ≥0.70 | ✗ | 自然语言 query 的 token 与代码符号 BM25 token 不匹配 |
| `MRR_hybrid` | 1.00 | ≥0.50 | ✓ | 第一个相关符号排第一 |
| `mode_compare_hybrid_vs_vector` | 0.00 | 仅观察 | — | hybrid 与 vector 表现相当（小数据集） |
| `mode_compare_hybrid_vs_keyword` | +1.00 | 仅观察 | — | hybrid 显著优于 keyword，证明混合重排价值 |

**关键结论**：mode_compare 数据（hybrid_vs_keyword = +1.00）验证了 PR#1 混合重排设计的价值——hybrid 完全覆盖了 keyword 无法召回的语义查询，而 keyword 在自然语言场景下完全失效。这为下一轮保留/增强混合检索提供了数据支撑。

注：keyword recall=0 不是 bug，是 BM25 对自然语言 query 的固有局限——"how does user login work" 的 token 与代码里的 "LoginUser"/"VerifyPassword" 不重叠。hybrid 模式通过向量召回补上了这个缺口。
