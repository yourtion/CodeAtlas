# QA 上下文引擎设计（补全 Phase 1）

- **日期**: 2026-07-06
- **状态**: 已批准（设计阶段），待写实现计划
- **范围**: 新增「自然语言问答上下文组装」能力，补全 README 路线图 Phase 1 的最后一公里
- **不做**: 服务端 LLM 生成、Agentic RAG、跨语言互操作增强

---

## 1. 背景与目标

上一轮 MR（`fix/correctness-bugs-phase1`）完成了检索质量基础设施：BM25 关键词召回 + 混合重排、多粒度 embedding、多跳调用链查询（递归 CTE）、过滤下沉 SQL 消除 N+1。`internal/qa/` 和 `internal/retrieval/` 两个目录至今为空。

本设计把已有的检索 + 图谱能力接成「问一个自然语言问题 → 拿到带引用与调用脉络的上下文」的最后一公里，补全 Phase 1。

### 1.1 核心定位

**纯检索上下文组装（不做生成）**。服务端不绑定 LLM、不承担 token 成本，只做高质量上下文提供者，让用户把组装好的上下文喂给 Cursor / Copilot / ChatGPT 等任意工具。职责清晰、可测试性强、零外部依赖。

### 1.2 成功标准

1. `POST /api/v1/qa` 接受自然语言问题，返回结构化上下文块 + 拼好的 Markdown prompt
2. 上下文 = hybrid 检索 Top-K 命中 + 每个命中的 1 跳 callers/callees 图谱邻居
3. 支持多 `repo_id` 过滤
4. `codeatlas ask` CLI 命令可一键产出 prompt 文本
5. 单元测试覆盖各层（无 DB），集成测试覆盖端到端（真 DB）
6. 现有 search handler 平滑迁移到多 repo 过滤

---

## 2. 架构分层（方案 A）

三层职责严格分离，每层可独立测试：

```
cmd/cli/ask_command.go                ← CLI: codeatlas ask（HTTP 调用）
        │
        ▼
internal/api/handlers/qa_handler.go   ← HTTP 边界，纯编排，无业务逻辑
        │
        ▼
internal/qa/service.go                ← 编排 + 格式化（Ask 流程、prompt 拼接）
        │
        ▼
internal/retrieval/hybrid_retriever.go ← 检索 + 图谱组装（数据获取）
        │
        ▼
pkg/models/vector.go                  ← 已有：HybridSearch/KeywordSearch/SimilaritySearchWithFilters
pkg/models/edge.go                    ← 已有：GetCallersWithDetails/GetCalleesWithDetails
```

### 2.1 各层职责

| 层 | 包 | 职责 | 不做什么 |
|---|---|---|---|
| **retrieval** | `internal/retrieval` | 输入查询 + filters，输出 `[]ContextBlock`（命中符号 + 1 跳图谱邻居）。含 mode 分发、多 repo 过滤、邻居限流去重 | 不碰 HTTP、不碰格式化、不碰 prompt |
| **qa** | `internal/qa` | 调 retriever 拿 block，拼 Markdown prompt，装结构化 JSON 响应，token 截断，按需取源码 | 不碰数据库、不碰 HTTP |
| **handler** | `internal/api/handlers` | JSON 绑定、调 service、返回响应、错误码 | 无业务逻辑 |

### 2.2 依赖注入

retrieval 和 qa 都用**接口**做依赖注入（同项目 Embedder/Parser 约定），便于单测 mock。

---

## 3. 核心数据结构

### 3.1 retrieval 层（`internal/retrieval/retriever.go`）

```go
// RetrievalRequest 是检索层入口
type RetrievalRequest struct {
    Query       string   // 自然语言问题或符号名
    RepoIDs     []string // 空 = 全库；多 repo 按列表过滤
    Language    string   // 可选语言过滤
    Kind        []string // 可选符号类型过滤（function/class/...）
    Mode        string   // "hybrid"(默认) | "vector" | "keyword"
    Limit       int      // Top-K，默认 10
    ExpandHops  int      // 图谱扩展跳数，本迭代固定为 1（保留字段供未来扩展）
    ExpandCallers bool   // 默认 true，是否拉取 callers
    ExpandCallees bool   // 默认 true，是否拉取 callees
}

// ContextBlock 是一个检索命中的完整上下文单元
type ContextBlock struct {
    Symbol     ContextSymbol   // 主命中符号
    Similarity float64         // 检索得分（向量/关键词/混合同量纲 [0,1]）
    MatchMode  string          // "vector" | "keyword" | "hybrid"
    Callers    []ContextSymbol // 1 跳：谁调用了它（每边 Top-5）
    Callees    []ContextSymbol // 1 跳：它调用了谁（每边 Top-5）
    ChunkID    string          // 对应 vectors.vector_id，用于按需取源码
}

// ContextSymbol 是图谱/检索共用的符号视图
// 刻意去耦 models.VectorSearchResult 的内部字段，转换在 retrieval 层内完成
type ContextSymbol struct {
    SymbolID  string
    Name      string
    Kind      string
    Signature string
    FilePath  string
    Language  string
    Docstring string
}
```

### 3.2 retrieval 层接口

```go
// Retriever 是检索层的可注入接口
type Retriever interface {
    Query(ctx context.Context, req RetrievalRequest) ([]ContextBlock, error)
}

// HybridRetriever 默认实现，组合两个已有仓库
type HybridRetriever struct {
    vectorRepo VectorSearcher   // 收窄的接口，屏蔽 *models.VectorRepository
    edgeRepo   EdgeExpander     // 收窄的接口，屏蔽 *models.EdgeRepository
    embedder   indexer.Embedder // 复用现有接口做 query embedding
}

// VectorSearcher 收窄 VectorRepository 用到的方法，便于 mock
type VectorSearcher interface {
    HybridSearch(ctx context.Context, query string, emb []float32, f models.VectorSearchFilters, wv, wk float64) ([]*models.HybridSearchResult, error)
    KeywordSearch(ctx context.Context, query string, f models.VectorSearchFilters) ([]*models.VectorSearchResult, error)
    SimilaritySearchWithFilters(ctx context.Context, emb []float32, f models.VectorSearchFilters) ([]*models.VectorSearchResult, error)
    // GetByVectorIDs 需在 VectorRepository 新增（现仅有单个 GetByID）。
    // 给 IncludeSource / chunks 端点用，按 vector_id 批量取 content。
    GetByVectorIDs(ctx context.Context, ids []string) ([]*models.Vector, error)
}

// EdgeExpander 收窄 EdgeRepository 用到的方法
type EdgeExpander interface {
    GetCallersWithDetails(ctx context.Context, symbolID string) ([]*models.EdgeWithDetails, error)
    GetCalleesWithDetails(ctx context.Context, symbolID string) ([]*models.EdgeWithDetails, error)
}
```

### 3.3 qa 层（`internal/qa/service.go`）

```go
type AskRequest struct {
    Query         string
    RepoIDs       []string
    Language      string
    Kind          []string
    Mode          string   // 默认 hybrid
    Limit         int      // 默认 10
    IncludeSource bool     // 默认 false：prompt 不含源码；true 时内联源码片段
    ExpandCallers bool     // 默认 true
    ExpandCallees bool     // 默认 true
}

type AskResponse struct {
    Query     string             `json:"query"`     // 回显
    Blocks    []ContextBlockJSON `json:"blocks"`    // 结构化上下文块
    Prompt    string             `json:"prompt"`    // 拼好的 Markdown prompt
    Truncated bool               `json:"truncated"` // prompt 是否因超长被截断
    ChunkIDs  []string           `json:"chunk_ids"` // 所有 block 的 chunk_id 去重汇总
}

type ContextBlockJSON struct {
    Symbol     SymbolJSON   `json:"symbol"`
    Similarity float64      `json:"similarity"`
    MatchMode  string       `json:"match_mode"`
    Callers    []SymbolJSON `json:"callers"`
    Callees    []SymbolJSON `json:"callees"`
    ChunkID    string       `json:"chunk_id"`
    Source     string       `json:"source,omitempty"` // IncludeSource=true 才有
}
```

---

## 4. 检索层流程（`internal/retrieval/hybrid_retriever.go`）

`Query` 方法步骤：

1. **mode 分发**（默认 hybrid）：把现有 `search_handler.go` 里的 mode 分发逻辑下沉到此层。handler 不再做检索分发。
2. **构建 filters**：按 `RepoIDs` 构造 `models.VectorSearchFilters`（多 repo 用 `ANY()` 匹配，见 §7）
3. **执行检索**：
   - `hybrid`：调 `HybridSearch(query, emb, filters, 0.7, 0.3)`（权重与现有 search handler 一致）
   - `vector`：调 `SimilaritySearchWithFilters`
   - `keyword`：调 `KeywordSearch`（无需 embedding）
4. **1 跳图谱扩展**：对每个命中符号，若 `ExpandHops > 0` 且对应开关开，调 `GetCallersWithDetails` + `GetCalleesWithDetails`
5. **邻居限流**：callers 和 callees 各取 Top-5，防止热点函数拖出几十个邻居导致 prompt 爆炸。排序依据：`EdgeWithDetails` 按调用频次降序（频次相同按符号名），频次信息由现有多跳查询的计数列提供；若该列不可得则退化为按符号名字典序稳定排序
6. **去重**：同一符号可能作为多个命中的邻居出现，跨 block 去重
7. **并发控制**：Top-K=10 时最多 20 个 edge 查询，用 `errgroup` + 信号量限制 4 并发（复用项目已有并发模式）
8. **转换**：`VectorSearchResult` / `EdgeWithDetails` → `ContextSymbol`，填充 `ChunkID`（来自 `VectorSearchResult.VectorID`）
9. 返回 `[]ContextBlock`

---

## 5. QA 层与 Prompt 格式

### 5.1 Ask 流程（`internal/qa/service.go`）

1. 构造 `RetrievalRequest`（转 AskRequest 字段）
2. `retriever.Query()` → `[]ContextBlock`
3. 若 `IncludeSource`：用 `chunk_id` 批量调 `GetByVectorIDs`，把 `vectors.content` 挂到 block.Source
4. `prompt_builder` 拼 Markdown prompt
5. 汇总 `chunk_ids`（去重）
6. 装结构化 JSON 响应

### 5.2 Prompt 格式（`internal/qa/prompt_builder.go`）

```markdown
# Code Context

## Question
<用户的问题>

## Repositories
repo-a, repo-b

## Relevant Symbols

### 1. FunctionName (similarity: 0.87)
- **File**: `path/to/file.go:42`
- **Signature**: `func FunctionName(args) ReturnType`
- **Docstring**: 这是函数的文档说明...
- **Called by**:
  - `CallerA` (path/to/caller.go:10)
  - `CallerB` (path/to/caller.go:25)
- **Calls**:
  - `CalleeX` (path/to/callee.go:5)

### 2. AnotherClass (similarity: 0.82)
- ...
```

`IncludeSource=true` 时，每个符号段落末尾追加 fenced 代码块（来自 `vectors.content`）。

### 5.3 Token 预算与智能截断

- 软上限默认 **8000 token**（按 4 字符 ≈ 1 token 估算）
- 超限策略：优先保留高 similarity 的 block；截断低分 block 的图谱邻居（先砍 callers/callees，再砍 block 本身）
- 响应带 `Truncated: true` 标记

---

## 6. HTTP API

在 `internal/api/server.go` 的 `RegisterRoutes` 注册两个端点：

### 6.1 `POST /api/v1/qa`

**请求**：
```json
{
  "query": "用户登录流程涉及哪些函数",
  "repo_ids": ["repo-uuid-1", "repo-uuid-2"],
  "language": "go",
  "kind": ["function"],
  "mode": "hybrid",
  "limit": 10,
  "include_source": false,
  "expand_callers": true,
  "expand_callees": true
}
```

**响应 200**：
```json
{
  "query": "...",
  "blocks": [{ "symbol": {...}, "similarity": 0.87, "match_mode": "hybrid",
               "callers": [...], "callees": [...], "chunk_id": "vec-xxx" }],
  "prompt": "# Code Context\n...",
  "truncated": false,
  "chunk_ids": ["vec-xxx", "vec-yyy"]
}
```

**错误**：400（query 为空等参数错误）/ 500（检索或 embedding 失败）

### 6.2 `GET /api/v1/qa/chunks`

按 chunk_id 批量取源码（默认路径不取源码时的按需拉取入口）。

**请求**：`GET /api/v1/qa/chunks?ids=id1,id2,id3`

**响应 200**：
```json
{
  "chunks": [
    { "chunk_id": "id1", "symbol_id": "sym-1", "content": "...", "file_path": "..." }
  ]
}
```

**错误**：400（ids 为空或超过上限 50）/ 500

### 6.3 Handler 构造

`QAHandler` 持有 `qa.Service`，构造方式参考 `NewSearchHandler`：从 DB + EmbedderConfig 组装 retriever → service。`Ask` 和 `GetChunks` 两个方法。

---

## 7. 变更影响分析（breaking change）

### 7.1 `VectorSearchFilters` 迁移（核心 breaking change）

`models.VectorSearchFilters.RepoID string` → **`RepoIDs []string`**。三处 SQL 等值匹配改为 `ANY()`：

| 文件 | 当前 | 改为 |
|---|---|---|
| `pkg/models/vector.go:340` | `filters.RepoID != ""` (needJoin 判断) | `len(filters.RepoIDs) > 0` |
| `pkg/models/vector.go:403-404` | `AND f.repo_id = $N` (SimilaritySearchWithFilters) | `AND f.repo_id = ANY($N)` |
| `pkg/models/vector.go:503-504` | `AND f.repo_id = $N` (KeywordSearch) | `AND f.repo_id = ANY($N)` |
| `pkg/models/vector.go` HybridSearch recallFilters | 同上 | 同上 |

### 7.2 search_handler 调用点

| 文件 | 当前 | 改为 |
|---|---|---|
| `internal/api/handlers/search_handler.go:109` | `RepoID: req.RepoID` | `RepoIDs: req.RepoIDs` |
| `SearchRequest` 结构 | `RepoID string json:"repo_id"` | `RepoIDs []string json:"repo_ids"` |
| `internal/api/handlers/search_handler_test.go:30,87,104` | `repo_id` / `RepoID:` | `repo_ids` / `RepoIDs:` |

### 7.3 隔离边界（不动）

grep 确认大量 `RepoID` 字段属于**仓库实体的主键**（`Repository.RepoID`、`IndexerConfig.RepoID`、`IndexRequest.RepoID` 等），是不同结构的同名字段，**绝不能动**。本 breaking change 只动检索过滤这一处。

### 7.4 文档同步

`docs/api.md` 的 search 端点字段说明（`repo_id` → `repo_ids`）+ 新增 QA 端点文档。

### 7.5 验证要求（项目规范）

改了 `VectorSearchFilters` 后，必须本地跑 `make test-integration` 全量验证，不能只跑 `make test`（`-short` 会 skip 集成测试）。CI 跑全量测试 + 覆盖率。

---

## 8. CLI（`cmd/cli/ask_command.go`）

参考 `cmd/cli/search_command.go` 风格。

```bash
codeatlas ask \
  --question "用户登录流程涉及哪些函数" \
  --api-url http://localhost:8080 \
  --repo <repo_id>            # 可重复（StringSliceFlag），支持多 repo
  --mode hybrid               # 默认 hybrid
  --limit 10
  --include-source            # 默认 false
  --output prompt.md          # 默认 stdout；指定则写文件
  --json                      # 输出完整 JSON 响应而非仅 prompt
```

**行为**：
- 默认把 `prompt` 文本打到 stdout（方便 `codeatlas ask ... | pbcopy`）
- `--json` 输出完整 JSON 响应
- `--output` 写文件
- 调用 `pkg/client/api_client.go`（上一轮 MR 已加）发 HTTP 请求

---

## 9. 测试策略

遵循项目规范：先写代码后写测试、单元测试无 DB、集成测试需 DB、表驱动 + 子测试 + 接口注入 mock。

| 层 | 测试文件 | 类型 | 覆盖 |
|---|---|---|---|
| retrieval | `internal/retrieval/hybrid_retriever_test.go` | 单元 | mock VectorSearcher+EdgeExpander：mode 分发、RepoIDs 传参、邻居 Top-5 限流、去重、邻居开关 |
| retrieval | `internal/retrieval/hybrid_retriever_integration_test.go` | 集成 | 真 DB：索引 fixture → Query → 验证 ContextBlock 含正确 callers/callees |
| qa | `internal/qa/prompt_builder_test.go` | 单元 | 表驱动：Markdown 格式、token 截断逻辑、Truncated 标记、IncludeSource 内联 |
| qa | `internal/qa/service_test.go` | 单元 | mock Retriever+SourceFetcher：Ask 流程、chunk_ids 汇总、IncludeSource 分支 |
| api | `internal/api/handlers/qa_handler_test.go` | 单元 | mock qa.Service：JSON 绑定、参数校验、错误码、chunks 端点 ids 校验 |
| cli | `cmd/cli/ask_command_test.go` | 单元 | mock HTTP client：flag 解析、repo 可重复、stdout/file/json 输出 |
| models | `pkg/models/vector_test.go`（扩展） | 集成 | 验证 RepoIDs 的 ANY() 在三种搜索里都正确（多 repo 命中） |

集成测试复用 `tests/integration/test_utils.go` 的 DB 启动 + 现有 call_analysis fixtures（已有丰富 caller/callee 关系）。

---

## 10. 文件清单

**新建**：
- `internal/retrieval/retriever.go`（接口 + 数据结构）
- `internal/retrieval/hybrid_retriever.go`（默认实现）
- `internal/retrieval/hybrid_retriever_test.go`
- `internal/retrieval/hybrid_retriever_integration_test.go`
- `internal/qa/service.go`（接口 + 实现）
- `internal/qa/prompt_builder.go`
- `internal/qa/service_test.go`
- `internal/qa/prompt_builder_test.go`
- `internal/api/handlers/qa_handler.go`
- `internal/api/handlers/qa_handler_test.go`
- `cmd/cli/ask_command.go`
- `cmd/cli/ask_command_test.go`

**修改**：
- `pkg/models/vector.go`（VectorSearchFilters.RepoID → RepoIDs，三处 SQL；新增 `GetByVectorIDs` 批量查询方法）
- `internal/api/handlers/search_handler.go`（SearchRequest + filters 构造）
- `internal/api/handlers/search_handler_test.go`
- `internal/api/server.go`（注册 QA 路由 + 构造 handler）
- `cmd/cli/main.go`（注册 ask 命令）
- `docs/api.md`（search 字段变更 + 新增 QA 端点）

---

## 11. 不做的事（YAGNI）

- ❌ 服务端 LLM 生成答案（保持薄上下文提供者定位）
- ❌ Agentic RAG / 多轮工具调用（留待 Phase 4）
- ❌ 2 跳及以上图谱扩展（`ExpandHops` 字段保留但本迭代固定 1）
- ❌ 查询改写 / cross-encoder 重排（留待后续检索质量迭代）
- ❌ 跨语言互操作增强（已有能力直接复用）
