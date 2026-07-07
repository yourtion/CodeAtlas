# 质量评估系统设计

- **日期**: 2026-07-07
- **状态**: 已批准（设计阶段），待写实现计划
- **范围**: 新建 `internal/quality/` 评估领域包 + `codeatlas eval` CLI + 集成测试门禁，覆盖依赖图质量与检索质量两个环节
- **不做**: QA 端到端评估、解析器改造、结构断言硬门禁、评估报告持久化、Web UI

---

## 1. 背景与目标

### 1.1 前序迭代

- **PR#1**（`fix/correctness-bugs-phase1`）：检索质量地基（BM25 + 混合重排、多粒度 embedding、多跳调用链 CTE、去 AGE 改关系表）
- **PR#2**（`feat/qa-context-engine`）：QA 上下文引擎（`POST /api/v1/qa` + `codeatlas ask`，纯检索上下文组装，1 跳图谱扩展）

PR#2 的集成测试代码已写（`TestVectorSearch_RepoIDsFilter`、`TestHybridRetriever_Integration`），但因本地 Docker 不可用未在真 DB 跑通验证。依赖图地基（关系表真源 + 多跳 CTE + Tree-sitter 边提取）已铺好，"精准"指边的覆盖率与准确率，而非从零建图。

### 1.2 本轮定位

**先建指标体系，再内建评估系统，既指导迭代又做门禁**。核心交付是评估系统本身，依赖图和 PR#2 集成测试是被评估/被验证的对象。依赖图具体改造点由测量结果驱动，这轮不预设改哪里——可能跑出某处特别差就顺手修，也可能只先把指标跑起来建立基线。

PR#2 的集成测试验证需求并入评估系统的 fixture 真值，不重复劳动。

### 1.3 成功标准

1. `codeatlas eval` CLI 可用：`--repo` 和 `--fixtures` 两模式都能跑通，输出 text/json 报告，门禁失败 exit 1
2. 依赖图指标全覆盖：5 个结构断言类 + 3 个 fixture 真值类指标全部实现并能产出数值
3. 检索指标全覆盖：recall@k、MRR、neighbor_hit_rate、mode 对比全部实现
4. PR#2 集成验证并入完成：`TestQualityGate_PR2Regression` 跑通，RepoIDs 过滤和 retrieval 端到端正确性由 recall 真值验证
5. 真实仓库基线建立：对 CodeAtlas 自身代码库跑一次 `eval --repo`，记录基线报告，作为下一轮门禁收紧的参照
6. 门禁阈值生效：fixture 真值类指标在 CI 上作为硬门禁，不达标 CI 红
7. 单元测试覆盖率 ≥ 90%（quality 包新增代码）
8. `make test-integration` 全绿（本地验证 + CI 验证）

---

## 2. 架构分层

```
cmd/cli/eval_command.go              ← CLI: codeatlas eval（编排，无业务）
        │
        ▼
internal/quality/                    ← 新建领域包（指标体系内聚）
  ├── metrics.go                     ← 指标定义 + MetricValue/Report/Summary + 阈值常量
  ├── graph_evaluator.go             ← 依赖图评估（调 models 拿数据算指标）
  ├── retrieval_evaluator.go         ← 检索评估（调 retrieval 跑查询对真值）
  ├── report.go                      ← Evaluate 顶层编排 + Report 序列化 + 阈值对比
  └── fixtures/                      ← 真值定义（集中管理）
        ├── graph_ground_truth.go    ← 各 fixture 期望的边
        └── retrieval_ground_truth.go← 各 fixture+query 期望命中符号
        │
        ▼
pkg/models/graph_metrics.go          ← 新增：聚合查询方法（CountEdgesByType 等）
pkg/models/edge.go                   ← 已有：GetCallersWithDetails/GetCalleesWithDetails
pkg/models/symbol.go                 ← 已有
pkg/models/vector.go                 ← 已有：HybridSearch/KeywordSearch（被 retrieval 层调用）
internal/retrieval/hybrid_retriever.go ← 已有，被检索评估复用
```

### 2.1 分层职责

| 层 | 职责 | 不做什么 |
|---|---|---|
| `quality` | 定义指标、跑评估、产出 Report、阈值对比 | 不碰 HTTP、不碰 DB 连接管理（DB 由调用方传入） |
| `graph_evaluator` | 给定 repoID + DB，算依赖图指标 | 不算检索指标 |
| `retrieval_evaluator` | 给定 fixture 真值 + retriever，算 recall@k / MRR | 不算图指标 |
| `eval_command` | 解析 flag、连 DB / 构造 retriever、调 quality 评估、输出报告 | 无指标计算逻辑 |
| 集成测试 | 调 quality 包 `Evaluate`，断言 Report ≥ 阈值 | 不重复实现指标 |

### 2.2 依赖注入

`quality` 包用接口收窄 models/retrieval 依赖（同项目 Embedder/Parser/Retriever 约定），便于单测 mock。评估器构造函数接收已构造好的依赖，不自己建连接。

### 2.3 关键设计点

fixture 真值和真实仓库走同一 Evaluator 接口——区别只在"真值来源"：fixture 真值是 `quality/fixtures` 里的静态数据；真实仓库无真值，只跑结构断言类指标，跳过 recall/MRR。用 `EvalMode`（`"fixture" | "repo"`）区分，Evaluator 内部按 mode 选算哪些指标。

---

## 3. 依赖图指标定义

分两类——**结构断言类**（无需真值，真实仓库可跑）和 **fixture 真值类**（需手工标注，门禁用）。

### 3.1 结构断言类指标（真实仓库 + fixture 都跑）

| 指标 | 定义 | 计算方式 | 健康基线（初定） |
|---|---|---|---|
| **悬空边率** `dangling_edge_ratio` | `target_id IS NULL` 的边 / 总边数 | `COUNT(target_id IS NULL) / COUNT(*)`，按 edge_type 分桶 | < 30%（import 类允许较高，call 类应很低） |
| **符号消解率** `symbol_resolution_rate` | `target_id IS NOT NULL` 的边 / 总边数 | 1 - 悬空边率，按 edge_type 分桶 | > 70% |
| **孤立符号率** `orphan_symbol_ratio` | 无任何出入边的符号 / 总符号数 | `COUNT(无 in/out edge) / COUNT(symbols)` | < 40%（入口符号合理孤立，但过高说明图断裂） |
| **edge_type 分布** `edge_type_distribution` | 各 edge_type 的计数与占比 | `GROUP BY edge_type` | 无阈值，观察用 |
| **跨文件连接率** `cross_file_connectivity` | source_file ≠ target_file 的边 / 总边数 | `COUNT(source_file ≠ target_file) / COUNT(*)` | > 20%（过低说明跨文件关系没提出来） |

**分桶原则**：悬空边率和符号消解率按 `edge_type` 分桶报告（import 悬空是正常的——外部依赖；call 悬空是问题——说明符号没解析到）。Report 里既有总值也有分桶值。

### 3.2 fixture 真值类指标（仅 fixture 模式跑）

| 指标 | 定义 | 计算方式 | 门禁阈值（初定） |
|---|---|---|---|
| **边召回率** `edge_recall` | 真值边中被提取出来的比例 | `|提取边 ∩ 真值边| / |真值边|` | ≥ 90% |
| **边准确率** `edge_precision` | 提取边中正确的比例 | `|提取边 ∩ 真值边| / |提取边|` | ≥ 85% |
| **调用链连通性** `call_chain_connectivity` | 真值里 A→B→C 链路在图里能走通的比例 | 多跳 CTE 查真值链路端点对，看是否连通 | ≥ 95% |

**边匹配规则**：真值边用 `(source_file, source_name, edge_type, target_name)` 四元组表示，不依赖 symbol_id（symbol_id 是入库后才有）。匹配时按四元组在提取边里找。`target_name` 允许模糊（如 `c_init` 匹配 `c_init`，外部库函数 `malloc` 匹配 target_module 含 `malloc`）。

### 3.3 真值来源

**现有可复用真值**：`tests/integration/call_analysis_fixtures_test.go` 里已有 `expectedCCalls` 等散落的期望列表（如 `cpp_calls_c.cpp` 的 15 个期望 call）。评估系统把这些系统化成 `quality/fixtures/graph_ground_truth.go` 里的结构化真值：

```go
type GraphGroundTruth struct {
    FixtureFile string       // "tests/fixtures/cpp/cpp_calls_c.cpp"
    Edges       []ExpectedEdge
}
type ExpectedEdge struct {
    SourceName string   // "CppClass::CppMethod"
    EdgeType   string   // "call"
    TargetName string   // "c_init"
    Optional   bool     // true = 提到了不算漏（如标准库 strlen）
}
```

**这轮覆盖的 fixture**：先用现有 `call_analysis_fixtures_test.go` 涉及的 fixture（跨语言 + 单语言），不新造 fixture 文件。真值从那些 `expectedXxxCalls` 列表迁移过来。

### 3.4 门禁 vs 观察

- fixture 真值类指标做**硬门禁**（不达标 CI 红）
- 结构断言类指标这轮**只建立基线、不做硬门禁**（报告里记录值，阈值设为"建议值"不强制）——因为还没有真实仓库的基线数据，强行定阈值会误伤。下一轮有了基线再收紧

---

## 4. 检索质量指标定义

检索评估**必须有真值**（无真值无法算 recall/MRR），所以只跑 fixture 模式。

### 4.1 指标定义

| 指标 | 定义 | 计算方式 | 门禁阈值（初定） |
|---|---|---|---|
| **recall@k** | Top-K 命中里包含真值相关符号的比例 | `|命中 ∩ 真值相关| / |真值相关|`，k=10 | ≥ 70% |
| **MRR** (Mean Reciprocal Rank) | 第一个真值相关符号的排名倒数的均值 | `mean(1/rank_of_first_relevant)` | ≥ 0.5 |
| **邻居扩展命中率** `neighbor_hit_rate` | 1 跳 callers/callees 里含真值相关符号的比例 | `|邻居 ∩ 真值相关| / |真值相关|` | ≥ 60%（图谱扩展的价值证明） |
| **mode 对比** | hybrid vs vector vs keyword 的 recall@k 差值 | 各 mode 各自算 recall，报告差值 | 无阈值，观察 hybrid 是否优于单模（验证 PR#1 的混合重排） |

**"真值相关符号"定义**：对一个 query，预先标注"这个问题的答案应该包含哪些符号"。如 query "C++ 如何调用 C 函数" → 真值相关 = `[CppClass::CppMethod, c_init, c_process_string]`。

### 4.2 检索真值结构

```go
// internal/quality/fixtures/retrieval_ground_truth.go
type RetrievalGroundTruth struct {
    Query           string       // "C++ 如何调用 C 函数"
    RelevantSymbols []string     // 真值相关符号名（按 name 匹配，不依赖 symbol_id）
    RelevantFiles   []string     // 真值相关文件（辅助匹配，符号名歧义时用文件消歧）
    Repos           []string     // 涉及的 repo（fixture 标识）
}
```

**匹配规则**：命中符号按 `name` 匹配真值；同名歧义时（如多个 `init` 函数）用 `file_path` 消歧。匹配在 retrieval_evaluator 内完成，不污染 retrieval 层。

### 4.3 检索真值来源

这轮**新构造少量检索真值**，来源是现有 fixture 的自然问题：

| Query | 真值相关符号 | 来源 fixture |
|---|---|---|
| "C++ 如何调用 C 函数" | `CppClass::CppMethod`, `c_init`, `c_process_string` | `cpp_calls_c.cpp` + `c_library.h` |
| "Kotlin 调用 Java 的哪些方法" | （从 `kotlin_calls_java.kt` 提取） | `kotlin_calls_java.kt` + `java_library.java` |
| "Swift 如何互操作 Objective-C" | （从 fixture 提取） | `swift_calls_objc.swift` + `objc_class.h/m` |
| "Go 函数调用关系" | （从单语言 Go fixture 提取） | `call_analysis_single_language` 的 Go case |

**真值数量**：初定 6-10 个 query，覆盖跨语言（3）+ 单语言（3-4）+ 多 repo 过滤（1-2）。这轮先把架子搭起来跑通，真值可以少但要准。

### 4.4 mode 对比的价值

报告里会显示 hybrid/vector/keyword 三种 mode 的 recall@k 对比。如果 hybrid ≤ vector 或 ≤ keyword，说明 PR#1 的混合重排没生效或退化——这是个免费的回归监测，不需要额外成本。

### 4.5 邻居扩展命中率的意义

这个指标直接验证 PR#2 的核心设计——"1 跳图谱扩展是否有价值"。如果 neighbor_hit_rate 很低，说明扩展出来的邻居大多无关，要么是 Top-5 排序策略有问题，要么图谱扩展本身价值有限。指标驱动后续改造方向。

---

## 5. quality 包核心数据结构与接口

### 5.1 核心数据结构（`metrics.go`）

```go
package quality

// EvalMode 区分评估模式
type EvalMode string
const (
    EvalModeFixture EvalMode = "fixture"  // 跑真值类指标（recall/precision/MRR）
    EvalModeRepo    EvalMode = "repo"     // 只跑结构断言类指标（真实仓库无真值）
)

// MetricValue 单个指标的值（支持分桶）
type MetricValue struct {
    Name       string             // "dangling_edge_ratio"
    Category   MetricCategory     // "graph" / "retrieval"，用于报告分组显示
    Value      float64            // 0.0 - 1.0
    Threshold  float64            // 门禁阈值；0 = 仅观察无阈值
    Passed     bool               // Value 是否达标（无阈值时恒 true）
    Bucket     string             // 分桶标签，如 "import"/"call"；空 = 总值
    Detail     map[string]float64 // 子分桶明细，如按 edge_type 分桶时的各值
}

// MetricCategory 指标大类
type MetricCategory string
const (
    CategoryGraph     MetricCategory = "graph"     // 依赖图指标
    CategoryRetrieval MetricCategory = "retrieval" // 检索指标
)

// Report 评估报告（eval CLI 和集成测试共用）
type Report struct {
    Mode       EvalMode        `json:"mode"`
    RepoID     string          `json:"repo_id,omitempty"`
    FixtureSet string          `json:"fixture_set,omitempty"` // 真值集标识
    Metrics    []MetricValue   `json:"metrics"`
    Summary    Summary         `json:"summary"`
}
type Summary struct {
    Total    int `json:"total"`
    Passed   int `json:"passed"`
    Failed   int `json:"failed"`
    NoThresh int `json:"no_threshold"` // 仅观察无阈值的指标数
}
```

### 5.2 Evaluator 接口

```go
// ExpectedChain 真值里的一条调用链（用于 call_chain_connectivity 指标）
// 不依赖 symbol_id，用符号名 + 文件定位
type ExpectedChain struct {
    StartName string   // 链路起点符号名
    EndName   string   // 链路终点符号名
    StartFile string   // 起点所在文件（消歧）
    EndFile   string   // 终点所在文件（消歧）
}

// GraphDataFetcher 收窄 models 依赖，便于 mock
type GraphDataFetcher interface {
    // 结构断言类原始数据
    CountEdgesByType(ctx context.Context, repoID string) (map[string]int, error)         // GROUP BY edge_type
    CountDanglingEdges(ctx context.Context, repoID string) (map[string]int, error)       // 按 edge_type 分桶的 target_id IS NULL 计数
    CountOrphanSymbols(ctx context.Context, repoID string) (int, error)                  // 无出入边的符号数
    CountCrossFileEdges(ctx context.Context, repoID string) (int, error)                 // source_file ≠ target_file
    CountTotalSymbols(ctx context.Context, repoID string) (int, error)
    // 调用链连通性
    CheckCallChainConnectivity(ctx context.Context, repoID string, chains []ExpectedChain) (int, int, error) // (连通数, 总数)
}

// GraphEvaluator 依赖图评估器
type GraphEvaluator struct {
    fetcher GraphDataFetcher
    truth   *GraphGroundTruth  // fixture 模式非空；repo 模式 nil
}

func (e *GraphEvaluator) Evaluate(ctx context.Context, repoID string, mode EvalMode) ([]MetricValue, error)
```

```go
// RetrievalRunner 收窄 retrieval.Retriever 依赖
type RetrievalRunner interface {
    Query(ctx context.Context, req retrieval.RetrievalRequest) ([]retrieval.ContextBlock, error)
}

// RetrievalEvaluator 检索评估器
type RetrievalEvaluator struct {
    runner RetrievalRunner
    truths []RetrievalGroundTruth
    modes  []string // 评估哪些 mode：["hybrid","vector","keyword"]
}

func (e *RetrievalEvaluator) Evaluate(ctx context.Context, repoIDs []string) ([]MetricValue, error)
```

### 5.3 顶层编排（`report.go`）

```go
// EvaluateConfig 一次评估的配置
type EvaluateConfig struct {
    Mode         EvalMode
    RepoID       string          // repo 模式必填
    RepoIDs      []string        // fixture 模式下检索评估的多 repo
    FixtureSet   string          // fixture 模式标识
    RunRetrieval bool            // 是否跑检索评估（repo 模式默认 false）
    RetrievalModes []string      // 默认 ["hybrid","vector","keyword"]
}

// Evaluate 顶层入口，CLI 和集成测试共用
func Evaluate(ctx context.Context, cfg EvaluateConfig, graphEval *GraphEvaluator, retrievalEval *RetrievalEvaluator) (*Report, error)
```

### 5.4 关键设计点

1. **MetricValue 的分桶**：一个指标可能产生多个 MetricValue（如 `dangling_edge_ratio` 会有总值 + 按 edge_type 的分桶值各一个），Report 里是扁平列表，前端/CLI 按需聚合显示
2. **阈值与门禁解耦**：阈值存在 `MetricValue.Threshold`，由 Evaluator 填入（来自 `metrics.go` 的常量定义）。集成测试可以临时覆盖阈值——`Report` 提供 `OverrideThreshold(name string, newThreshold float64)` 方法，重新计算对应指标的 `Passed`，便于不同 fixture 放宽
3. **repo 模式跳过 retrieval**：真实仓库无检索真值，`Evaluate` 在 repo 模式下 `RunRetrieval=false`，只跑结构断言。CLI 的 `--repo` 模式自动如此
4. **接口收窄**：`GraphDataFetcher` / `RetrievalRunner` 都是收窄接口，屏蔽 `*models.DB` / `*retrieval.HybridRetriever` 内部，单测可 mock（同 PR#2 的 `VectorSearcher`/`EdgeExpander` 约定）
5. **models 层新增方法**：`GraphDataFetcher` 需要的 `CountEdgesByType`/`CountDanglingEdges`/`CountOrphanSymbols`/`CountCrossFileEdges`/`CountTotalSymbols` 是 `pkg/models/graph_metrics.go` 新增的聚合查询方法，复用现有 DB 连接，纯 SQL

### 5.5 GraphDataFetcher 实现方式

聚合查询方法放 `pkg/models/graph_metrics.go`（`EdgeRepository` 和 `SymbolRepository` 已经很大，不再往里塞）。Fetcher 适配器在 `internal/quality/graph_evaluator.go` 里用一个 `defaultGraphFetcher` 结构组合 `*EdgeRepository` + `*SymbolRepository` 实现接口。models 层只加查询方法、quality 层持有适配器，职责清晰。

---

## 6. CLI 与集成测试门禁

### 6.1 `codeatlas eval` 命令（`cmd/cli/eval_command.go`）

参考 `impact_command` / `ask_command` 风格。

```bash
# 评估真实仓库（只跑结构断言，建立基线）
codeatlas eval --repo <repo_id> --db <conn>
codeatlas eval --repo <repo_id> --db <conn> --format json

# 评估 fixture 真值集（跑 recall/precision/MRR，门禁用）
codeatlas eval --fixtures --db <conn> --format text

# 指定只跑某一类
codeatlas eval --repo <repo_id> --only graph
codeatlas eval --fixtures --only retrieval
```

**Flags**：

| Flag | 类型 | 说明 |
|---|---|---|
| `--repo` | string | 真实仓库 ID（repo 模式） |
| `--fixtures` | bool | fixture 模式（与 `--repo` 互斥） |
| `--db` | string | 数据库连接串（默认读环境变量 DB_HOST 等，同 indexer 约定） |
| `--only` | string | `graph` / `retrieval` / 空(=全部)，repo 模式默认 graph |
| `--format` | string | `text`(默认) / `json` |
| `--threshold-override` | string | 可选，JSON 文件路径，覆盖默认阈值（迭代调参用，不影响门禁） |

**输出（text 格式）**：

```
CodeAtlas Quality Report
========================
Mode: fixture  FixtureSet: call_analysis
RepoID: -

== Graph Metrics ==
  edge_recall                  0.93  (≥0.90)  ✓
  edge_precision               0.88  (≥0.85)  ✓
  call_chain_connectivity      0.97  (≥0.95)  ✓
  dangling_edge_ratio          0.24  (<0.30)  ✓
    └ import                   0.45  (仅观察)
    └ call                     0.05  (仅观察)
  symbol_resolution_rate       0.76  (>0.70)  ✓
  orphan_symbol_ratio          0.18  (<0.40)  ✓

== Retrieval Metrics ==
  recall@10                    0.75  (≥0.70)  ✓
  MRR                          0.58  (≥0.50)  ✓
  neighbor_hit_rate            0.62  (≥0.60)  ✓
  mode_compare hybrid_vs_vector +0.12  (仅观察)
  mode_compare hybrid_vs_keyword +0.08 (仅观察)

Summary: 11 passed, 0 failed, 4 observed
```

**退出码**：有门禁指标失败 → exit 1；全过或仅观察 → exit 0。CI 用退出码做门禁。

### 6.2 集成测试门禁（`tests/integration/quality_gate_test.go`）

核心思路：集成测试 = 在真 DB 上跑 quality 包 + 断言 Report 全过。不重复实现指标，只做"装真值 → 索引 fixture → Evaluate → 断言"。

```go
func TestQualityGate_FixtureMode(t *testing.T) {
    if testing.Short() { t.Skip("integration") }
    
    db := SetupTestDB(t)
    // 1. 索引 fixture 到测试库（复用现有 indexer 集成测试的索引流程）
    repoID := indexFixtures(t, db, "call_analysis")
    
    // 2. 构造 evaluator
    graphEval := quality.NewGraphEvaluator(
        models.NewGraphDataFetcher(db), 
        quality.Fixtures.CallAnalysis,  // 真值
    )
    retrievalEval := quality.NewRetrievalEvaluator(
        buildRetriever(t, db),
        quality.Fixtures.RetrievalTruths,
    )
    
    // 3. 跑评估
    report, err := quality.Evaluate(ctx, quality.EvaluateConfig{
        Mode: quality.EvalModeFixture,
        FixtureSet: "call_analysis",
        RunRetrieval: true,
    }, graphEval, retrievalEval)
    require.NoError(t, err)
    
    // 4. 门禁断言
    for _, m := range report.Metrics {
        if m.Threshold > 0 && !m.Passed {
            t.Errorf("质量门禁失败: %s = %.2f, 阈值 %.2f", m.Name, m.Value, m.Threshold)
        }
    }
}
```

**门禁场景**：

| 测试 | 模式 | 跑什么 | 门禁 |
|---|---|---|---|
| `TestQualityGate_FixtureMode` | fixture | 图真值 + 检索真值全跑 | 硬断言所有有阈值的指标 |
| `TestQualityGate_RepoMode` | repo | 结构断言类 | 仅断言查询不报错 + Report 结构完整（不卡阈值，建基线） |
| `TestQualityGate_PR2Regression` | fixture | 专项跑 PR#2 遗留：RepoIDs 多 repo 过滤、retrieval 端到端 | 断言 recall@10 ≥ 0.7（PR#2 功能验证并入） |

**第三个测试是 PR#2 集成测试的归属**：PR#2 的 RepoIDs 过滤正确性、retrieval 端到端正确性，都由 retrieval_evaluator 跑 recall 时自然验证（多 repo query 的真值命中即证明过滤正确）。不再单独写 `TestRepoIDsFilter` 之类的用例。

### 6.3 fixture 索引复用

索引 fixture 的逻辑**复用 `tests/integration/indexer_integration_test.go` 已有的索引流程**。如果现有 helper 不够通用，在 `test_utils.go` 抽一个 `IndexFixtureSet(t, db, setName) repoID`，不重写索引逻辑。

### 6.4 评估驱动的 case 补充机制

评估跑起来后如果发现覆盖不足，按这个原则补 case：

- **某 edge_type 无真值**（如 `extends`/`implements` 没 fixture 覆盖）→ 补对应 fixture 的真值条目，优先复用现有 fixture 文件（只加真值标注，不造新文件）
- **某语言无 case**（如 Python 没有 retrieval 真值）→ 从现有 Python fixture 构造 1-2 个 query 真值
- **某指标无法计算**（如 neighbor_hit_rate 因真值相关符号太少算不出意义）→ 补充真值相关符号

补 case 的判断标准：评估报告里某指标 Detail 为空、或 fixture 覆盖率统计显示某维度为 0。这些会在 Report 里体现为"覆盖缺口"，不靠人肉记忆。

---

## 7. 文件清单与变更影响

### 7.1 新建文件

| 文件 | 说明 |
|---|---|
| `internal/quality/metrics.go` | 指标定义、MetricValue/Report/Summary 结构、阈值常量 |
| `internal/quality/graph_evaluator.go` | GraphEvaluator + GraphDataFetcher 接口 + defaultGraphFetcher 适配器 |
| `internal/quality/retrieval_evaluator.go` | RetrievalEvaluator + RetrievalRunner 接口 |
| `internal/quality/report.go` | Evaluate 顶层编排 + Report 序列化 + 阈值对比 |
| `internal/quality/fixtures/graph_ground_truth.go` | 依赖图真值（从现有 expectedXxxCalls 迁移） |
| `internal/quality/fixtures/retrieval_ground_truth.go` | 检索真值（6-10 个 query） |
| `internal/quality/metrics_test.go` | 单元：MetricValue 阈值判断、分桶聚合、Report 序列化 |
| `internal/quality/graph_evaluator_test.go` | 单元：mock GraphDataFetcher，各指标计算正确性 |
| `internal/quality/retrieval_evaluator_test.go` | 单元：mock RetrievalRunner，recall/MRR/neighbor_hit 计算 |
| `internal/quality/report_test.go` | 单元：Evaluate 编排、mode 分发、mode 对比 |
| `cmd/cli/eval_command.go` | `codeatlas eval` 命令实现 |
| `cmd/cli/eval_command_test.go` | 单元：flag 解析、输出格式、退出码 |
| `tests/integration/quality_gate_test.go` | 集成：fixture/repo 模式门禁 + PR#2 回归 |

### 7.2 修改文件

| 文件 | 改动 |
|---|---|
| `pkg/models/graph_metrics.go`（新增） | 5 个聚合查询方法：CountEdgesByType/CountDanglingEdges/CountOrphanSymbols/CountCrossFileEdges/CountTotalSymbols |
| `cmd/cli/main.go` | 注册 `eval` 命令 |
| `docs/evaluation.md`（新增） | 评估系统文档（指标定义、CLI 用法、门禁机制） |
| `docs/cli.md` | 补 `eval` 命令说明 |

### 7.3 不改的文件（隔离边界）

- `pkg/models/vector.go` — 检索方法已齐全，retrieval_evaluator 经 `retrieval.HybridRetriever` 调用，不直接碰 vector repo
- `internal/retrieval/` — 完全复用，不改
- `internal/qa/` — 不涉及（QA 端到端评估留下一轮）
- `internal/parser/` — 不改（依赖图改造由指标驱动，这轮不预设改解析器）
- `pkg/models/edge.go` / `pkg/models/symbol.go` — 不改（聚合查询放新文件 graph_metrics.go）

### 7.4 Breaking change 分析

**本迭代无 breaking change**：

1. `pkg/models/` 新增方法是纯增量（聚合查询方法在新文件），不改现有方法签名，不影响现有调用方
2. `internal/quality/` 是全新包，不触碰现有包的公开 API
3. `VectorSearchFilters` 不动（PR#2 已改完 RepoIDs，这轮复用）
4. CLI 新增命令，不改现有命令

### 7.5 验证要求（项目规范）

- 改了 `pkg/models/`（新增聚合查询）→ 必须本地跑 `make test-integration` 全量验证，不能只跑 `make test`（`-short` 会 skip 集成测试）
- `internal/quality/` 单元测试用 mock，跑 `make test` 即可
- 集成测试 `quality_gate_test.go` 是新增，跑 `make test-integration` 验证门禁逻辑
- CI 跑全量测试 + 覆盖率，本迭代新增代码覆盖率目标遵循项目 90% 约定

---

## 8. 测试策略

遵循项目规范：先写代码后写测试、单元无 DB、集成需 DB、表驱动 + 子测试 + 接口 mock。

| 层 | 测试文件 | 类型 | 覆盖要点 |
|---|---|---|---|
| quality/metrics | `metrics_test.go` | 单元 | MetricValue.Threshold 判断逻辑（≥/≤方向）、分桶聚合、Summary 计数 |
| quality/graph | `graph_evaluator_test.go` | 单元 | mock GraphDataFetcher：各结构断言指标计算、fixture 真值匹配（四元组）、mode 分发（repo 模式跳过真值类） |
| quality/retrieval | `retrieval_evaluator_test.go` | 单元 | mock RetrievalRunner：recall@k/MRR/neighbor_hit 计算、mode 对比差值、多 repo query |
| quality/report | `report_test.go` | 单元 | Evaluate 编排：fixture 模式跑两类、repo 模式只跑图、Report JSON 序列化、阈值覆盖 |
| cli | `eval_command_test.go` | 单元 | mock DB/retriever：flag 解析、repo/fixtures 互斥、format 切换、退出码（门禁失败 exit 1） |
| models | `graph_metrics_test.go`（新增） | 集成 | 真 DB：5 个聚合查询方法正确性（CountEdgesByType 分桶、DanglingEdges、OrphanSymbols、CrossFile、TotalSymbols） |
| 集成门禁 | `quality_gate_test.go` | 集成 | fixture 模式全指标门禁、repo 模式建基线、PR#2 回归（RepoIDs 过滤 + retrieval 端到端并入） |

**测试间独立性**：每个集成测试用例建独立测试库（复用 `SetupTestDB` 模式），不依赖其他测试的数据。fixture 索引用 helper 重复跑，不共享状态。

---

## 9. 不做的事（YAGNI）

- ❌ QA 端到端评估（问题→上下文是否含答案，留下一轮）
- ❌ 依赖图解析器改造（这轮只建指标和基线，改造点由测量结果驱动，下一轮再做）
- ❌ 结构断言类指标做硬门禁（这轮只建基线，下一轮有数据后收紧）
- ❌ 2 跳及以上图谱扩展评估（`ExpandHops` 固定 1，评估也只评 1 跳）
- ❌ 评估报告持久化/历史趋势（这轮报告是即时的，不存库不做趋势图）
- ❌ 跨语言互操作专项评估指标（用现有跨语言 fixture 的真值覆盖即可，不单独定义"互操作准确率"指标）
- ❌ 评估系统 Web UI（纯 CLI + 测试，无前端）

---

## 10. 风险与缓解

| 风险 | 缓解 |
|---|---|
| fixture 真值标注有误，导致门禁误伤 | 真值从现有 `expectedXxxCalls` 迁移（已验证过），新构造的检索真值少量且人工核对 |
| 阈值定得不合理（太高全红、太低形同虚设） | 阈值标"初定"，先跑出基线再调；结构断言类这轮不做硬门禁就是为这个 |
| 聚合查询在大仓库上慢 | 5 个查询都是 `COUNT(*)` + `GROUP BY`，有现成索引；若慢加 `repo_id` 过滤索引即可，这轮不预先优化 |
| PR#2 回归并入后遗漏原测试场景 | `TestQualityGate_PR2Regression` 的真值覆盖原 `TestVectorSearch_RepoIDsFilter` 和 `TestHybridRetriever_Integration` 的断言点，迁移后**保留原测试文件**直到确认新门禁覆盖等价，下一轮再删，避免这轮就破坏现有测试资产 |
