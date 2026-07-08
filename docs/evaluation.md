# 质量评估系统

CodeAtlas 内建质量评估系统，覆盖**依赖图质量**与**检索质量**两个环节，用于迭代观测和 CI 门禁。

## 快速开始

### 评估真实仓库（结构断言，建基线）

```bash
codeatlas eval --repo <repo_id> --db "host=localhost port=5432 user=codeatlas dbname=codeatlas sslmode=disable"
```

### 评估 fixture 真值集（recall/precision/MRR，门禁用）

```bash
codeatlas eval --fixtures --db "host=localhost port=5432 user=codeatlas dbname=codeatlas sslmode=disable"
```

### 输出格式

```bash
codeatlas eval --repo <repo_id> --db "..." --format json   # JSON 格式
codeatlas eval --repo <repo_id> --db "..." --format text   # 默认，文本表格
```

### 指定只跑某一类

```bash
codeatlas eval --repo <repo_id> --only graph      # 只跑依赖图指标
codeatlas eval --fixtures --only retrieval         # 只跑检索指标（需 embedding 环境）
```

## 指标体系

### 依赖图指标

#### 结构断言类（无需真值，真实仓库可跑）

| 指标 | 说明 | 建议基线 |
|---|---|---|
| `dangling_edge_ratio` | `target_id IS NULL` 的边占比，按 edge_type 分桶 | < 30% |
| `symbol_resolution_rate` | `target_id` 已解析的边占比（1 - 悬空边率） | > 70% |
| `orphan_symbol_ratio` | 无出入边的孤立符号占比 | < 40% |
| `cross_file_connectivity` | `source_file ≠ target_file` 的边占比 | > 20% |

#### fixture 真值类（门禁）

| 指标 | 说明 | 阈值 |
|---|---|---|
| `edge_recall` | 真值边被提取的比例 | ≥ 90% |
| `edge_precision` | 提取边匹配真值的比例 | ≥ 85% |
| `call_chain_connectivity` | 真值调用链在图中连通的比例 | ≥ 95% |

### 检索指标（fixture 模式，需 embedding 环境）

| 指标 | 说明 | 阈值 |
|---|---|---|
| `recall@10` | Top-10 命中含真值相关符号的比例 | ≥ 70% |
| `MRR` | 第一个相关符号排名倒数的均值 | ≥ 0.5 |
| `neighbor_hit_rate` | 1 跳邻居含真值相关符号的比例 | ≥ 60% |
| `mode_compare` | hybrid vs vector/keyword 的 recall@k 差值 | 仅观察 |

## 门禁机制

- **fixture 真值类**：CI 硬门禁，不达标 `exit 1`
- **结构断言类**：当前版本仅建基线，不做硬门禁。下一轮有基线数据后收紧为硬门禁

## 评估驱动的 case 补充

评估报告里某指标 Detail 为空或某维度覆盖为 0 时，按需补真值：

- 某 edge_type 无真值 → 补对应 fixture 真值条目
- 某语言无 case → 从现有 fixture 构造 query 真值
- 某指标无法计算 → 补充真值相关符号

## 真值来源

依赖图真值从 `tests/integration/call_analysis_fixtures_test.go` 里散落的 `expectedXxxCalls` 列表系统化迁移而来，存放在 `internal/quality/fixtures/`。

## 相关文件

- `internal/quality/` — 评估领域包（metrics/graph_evaluator/retrieval_evaluator/report/graph_data_fetcher）
- `internal/quality/fixtures/` — 真值数据
- `pkg/models/graph_metrics.go` — 聚合查询方法
- `cmd/cli/eval_command.go` — `codeatlas eval` 命令
- `tests/integration/quality_gate_test.go` — 集成门禁测试

## 已知限制

- **call 类悬空率较高**：`dangling_edge_ratio(call)` 在 fixture 数据上约 55%，主要因为 fixture 代码大量调用标准库/外部函数（strlen/malloc/printf 等），这些函数不在索引范围内。真实仓库的 call 悬空率会低很多。
- **同名符号消歧用首个候选**：C++ 重载、多文件同名符号等场景，消歧策略是"同文件优先 → import 文件优先 → 首个候选+日志"。首个候选可能不是语义上最准确的，但有 WARN 日志可观测。签名消歧留待后续（需解析器产出更丰富的 Target 字段）。
