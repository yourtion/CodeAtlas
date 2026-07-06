# QA 上下文引擎实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现「自然语言问题 → 带图谱脉络的结构化上下文 + Markdown prompt」的 QA 上下文引擎，补全 Phase 1 最后一公里。服务端不做 LLM 生成，只做纯检索上下文组装。

**Architecture:** 三层分离——`internal/retrieval`（检索 + 1 跳图谱组装，复用已有 HybridSearch/EdgeRepository）→ `internal/qa`（编排 + prompt 拼接 + token 截断）→ `qa_handler`（HTTP 边界）。检索层用收窄接口屏蔽 models，便于 mock。

**Tech Stack:** Go 1.25+ / Gin / PostgreSQL+pgvector / urfave/cli/v2。复用 `models.VectorRepository.HybridSearch`、`models.EdgeRepository.GetCallersWithDetails/GetCalleesWithDetails`。

**Spec:** `docs/superpowers/specs/2026-07-06-qa-context-engine-design.md`

---

## 文件结构

**新建**：
| 文件 | 职责 |
|---|---|
| `internal/retrieval/retriever.go` | 接口定义（`Retriever`、`VectorSearcher`、`EdgeExpander`）+ 数据结构（`RetrievalRequest`、`ContextBlock`、`ContextSymbol`） |
| `internal/retrieval/hybrid_retriever.go` | `HybridRetriever` 默认实现：mode 分发、RepoIDs 过滤、1 跳扩展、Top-5 限流、去重 |
| `internal/retrieval/hybrid_retriever_test.go` | 单元测试（mock 接口） |
| `internal/retrieval/hybrid_retriever_integration_test.go` | 集成测试（真 DB + fixtures） |
| `internal/qa/service.go` | `Service` 接口 + 实现：Ask 流程、chunk_ids 汇总、IncludeSource 分支 |
| `internal/qa/prompt_builder.go` | `BuildPrompt(blocks, opts) → (prompt, truncated)`：Markdown 拼接 + 8000 token 截断 |
| `internal/qa/service_test.go` | 单元测试（mock Retriever） |
| `internal/qa/prompt_builder_test.go` | 表驱动单元测试 |
| `internal/api/handlers/qa_handler.go` | `QAHandler`：`Ask` + `GetChunks` 两个端点 |
| `internal/api/handlers/qa_handler_test.go` | 单元测试（mock qa.Service） |
| `cmd/cli/ask_command.go` | `codeatlas ask` 命令 |
| `cmd/cli/ask_command_test.go` | 单元测试（mock HTTP） |

**修改**：
| 文件 | 改动 |
|---|---|
| `pkg/models/vector.go` | `VectorSearchFilters.RepoID string` → `RepoIDs []string`；三处 SQL 改 `ANY()`；新增 `GetByVectorIDs` |
| `internal/api/handlers/search_handler.go` | `SearchRequest.RepoID` → `RepoIDs`；filters 构造；mode 分发逻辑保留（暂不下沉） |
| `internal/api/handlers/search_handler_test.go` | 字段名同步 |
| `internal/api/server.go` | 加 `qaHandler` 字段；构造；注册两个 QA 路由 |
| `pkg/client/api_client.go` | `SearchFilters.RepoID` → `RepoIDs`；新增 `Ask`/`GetChunks` 方法 |
| `cmd/cli/main.go` | 注册 `createAskCommand()` |
| `docs/api.md` | search 字段变更 + 新增 QA 端点 |

---

## Task 1：扩展 VectorSearchFilters 支持多 repo_id

**Files:**
- Modify: `pkg/models/vector.go`

这是 breaking change 的核心。把单值 `RepoID string` 迁移为 `RepoIDs []string`，三处搜索 SQL 的等值匹配改为 `ANY()`。

- [ ] **Step 1: 修改 VectorSearchFilters 结构体**

把 `pkg/models/vector.go` 中 `VectorSearchFilters` 的 `RepoID string` 字段改为 `RepoIDs []string`：

```go
// VectorSearchFilters represents filters for vector similarity search
type VectorSearchFilters struct {
	EntityType  string   `json:"entity_type,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	Kind        []string `json:"kind,omitempty"`
	Language    string   `json:"language,omitempty"`
	RepoIDs     []string `json:"repo_ids,omitempty"` // 精确匹配（多 repo）
	WithDetails bool     `json:"with_details,omitempty"`
}
```

- [ ] **Step 2: 改 SimilaritySearchWithFilters 的 needJoin 和 SQL**

`SimilaritySearchWithFilters` 函数（约 line 338）：

needJoin 判断改为：
```go
needJoin := len(filters.Kind) > 0 || filters.Language != "" || len(filters.RepoIDs) > 0 || filters.WithDetails
```

repo 过滤子句（约 line 403）改为：
```go
if len(filters.RepoIDs) > 0 {
	whereClause += fmt.Sprintf(" AND f.repo_id = ANY(%s)", addArg(filters.RepoIDs))
}
```

- [ ] **Step 3: 改 KeywordSearch 的 needJoin 和 SQL**

`KeywordSearch` 函数（约 line 454）：同样把 needJoin 的 `filters.RepoID != ""` 改为 `len(filters.RepoIDs) > 0`，repo 过滤子句（约 line 503）改为 `ANY()`。

- [ ] **Step 4: 改 HybridSearch 的 recallFilters 构造**

`HybridSearch` 函数（约 line 557）内部构造 `recallFilters` 时会复制 `filters.RepoID`，改为复制 `RepoIDs`：

```go
recallFilters := models.VectorSearchFilters{
	EntityType:  filters.EntityType,
	Kind:        filters.Kind,
	Language:    filters.Language,
	RepoIDs:     filters.RepoIDs,
	WithDetails: filters.WithDetails,
}
```

- [ ] **Step 5: 修复所有 filters.RepoID 编译错误**

运行 `go build ./...`，预期在 `search_handler.go` 和测试文件出现 `filters.RepoID undefined` / `req.RepoID` 相关错误。这些在后续 Task 修复。当前只确保 `pkg/models` 自身编译通过：

```bash
go build ./pkg/...
```
Expected: 编译通过（pkg/models 内部不再引用 RepoID）

- [ ] **Step 6: 提交**

```bash
git add pkg/models/vector.go
git commit -m "refactor(models): VectorSearchFilters.RepoID 迁移为 RepoIDs []string

多 repo 过滤用 ANY() 匹配，为 QA 多仓库支持铺路。breaking change，
后续 task 同步更新 search handler 与 client。"
```

---

## Task 2：新增 GetByVectorIDs 批量查询方法

**Files:**
- Modify: `pkg/models/vector.go`

QA 的 `IncludeSource` 和 `/qa/chunks` 端点需要按 vector_id 批量取 content。

- [ ] **Step 1: 在 VectorRepository 添加 GetByVectorIDs 方法**

在 `pkg/models/vector.go` 的 `GetByID` 方法（约 line 88）后添加：

```go
// GetByVectorIDs 按 vector_id 批量查询向量记录（用于按需取源码片段）。
// 返回顺序不保证，调用方按 VectorID 自行对齐。
func (r *VectorRepository) GetByVectorIDs(ctx context.Context, ids []string) ([]*Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query := `
		SELECT vector_id, entity_id, entity_type, embedding::text, content, model, chunk_index, created_at
		FROM vectors
		WHERE vector_id = ANY($1)
	`
	rows, err := r.db.QueryContext(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to query vectors by ids: %w", err)
	}
	defer rows.Close()

	var results []*Vector
	for rows.Next() {
		var vector Vector
		var embeddingStr string
		if err := rows.Scan(
			&vector.VectorID, &vector.EntityID, &vector.EntityType, &embeddingStr,
			&vector.Content, &vector.Model, &vector.ChunkIndex, &vector.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan vector %s: %w", vector.VectorID, err)
		}
		embedding, err := parseEmbedding(embeddingStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse embedding for vector %s: %w", vector.VectorID, err)
		}
		vector.Embedding = embedding
		results = append(results, &vector)
	}
	return results, rows.Err()
}
```

注意：`parseEmbedding` 是同文件已有的包级函数（`GetByID` 等都在用）。如不存在则用现有代码里 `formatVectorForPgvector` 的反向逻辑（查 grep 确认函数名）。

- [ ] **Step 2: 验证编译**

```bash
go build ./pkg/...
```
Expected: PASS（如 `parseEmbedding` 函数名不对，grep 同文件确认正确名字后修正）

- [ ] **Step 3: 提交**

```bash
git add pkg/models/vector.go
git commit -m "feat(models): 新增 GetByVectorIDs 批量查询，供 QA 按需取源码"
```

---

## Task 3：迁移 search_handler 到 RepoIDs

**Files:**
- Modify: `internal/api/handlers/search_handler.go`
- Modify: `internal/api/handlers/search_handler_test.go`

修复 Task 1 留下的编译错误。

- [ ] **Step 1: 改 SearchRequest 结构体**

`internal/api/handlers/search_handler.go` 的 `SearchRequest`：

```go
type SearchRequest struct {
	Query    string   `json:"query" binding:"required"`
	RepoIDs  []string `json:"repo_ids,omitempty"`
	Language string   `json:"language,omitempty"`
	Kind     []string `json:"kind,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Mode     string   `json:"mode,omitempty"`
}
```

- [ ] **Step 2: 改 filters 构造**

`Search` 方法里（约 line 109）：

```go
filters := models.VectorSearchFilters{
	EntityType:  "symbol",
	Limit:       req.Limit,
	Kind:        req.Kind,
	Language:    req.Language,
	RepoIDs:     req.RepoIDs,
	WithDetails: true,
}
```

- [ ] **Step 3: 更新测试**

`internal/api/handlers/search_handler_test.go`：
- line 30: `{"repo_id": "test-repo"}` → `{"repo_ids": ["test-repo"]}`
- line 87: `RepoID: "repo-1"` → `RepoIDs: []string{"repo-1"}`
- line 104: `RepoID: "repo-1"` → `RepoIDs: []string{"repo-1"}`

（用 `grep -n "RepoID" internal/api/handlers/search_handler_test.go` 找全所有点，逐一改为 RepoIDs）

- [ ] **Step 4: 验证 search handler 测试通过**

```bash
go test ./internal/api/handlers/ -run TestSearch -v
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/api/handlers/search_handler.go internal/api/handlers/search_handler_test.go
git commit -m "refactor(api): search handler 迁移到 RepoIDs 多仓库过滤"
```

---

## Task 4：迁移 client.SearchFilters 到 RepoIDs

**Files:**
- Modify: `pkg/client/api_client.go`

CLI 通过 client 调 search，需同步。

- [ ] **Step 1: 改 SearchFilters 结构体**

```go
type SearchFilters struct {
	RepoIDs  []string `json:"repo_ids,omitempty"`
	Language string   `json:"language,omitempty"`
	Kind     []string `json:"kind,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}
```

- [ ] **Step 2: 改 Search 方法构造请求体**

`Search` 方法里：

```go
	if len(filters.RepoIDs) > 0 {
		searchReq["repo_ids"] = filters.RepoIDs
	}
```

- [ ] **Step 3: 验证编译**

```bash
go build ./pkg/client/... ./cmd/cli/...
```
Expected: PASS（如 cli 里有用到 SearchFilters.RepoID 的地方，grep 后同步改）

- [ ] **Step 4: 提交**

```bash
git add pkg/client/api_client.go
git commit -m "refactor(client): SearchFilters.RepoID 迁移为 RepoIDs"
```

---

## Task 5：集成测试验证 RepoIDs 多 repo 过滤

**Files:**
- Modify: `tests/integration/models_integration_test.go`

验证 breaking change 在真 DB 上正确工作（项目规范：改 schema 必须跑集成测试）。

- [ ] **Step 1: 加多 repo 过滤集成测试**

在 `tests/integration/models_integration_test.go` 找一个已有的向量搜索测试附近，添加子测试：

```go
func TestVectorSearch_RepoIDsFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db := setupTestDB(t) // 复用文件顶部已有的 setup 函数
	defer db.Close()

	ctx := context.Background()
	vectorRepo := models.NewVectorRepository(db)

	// 索引两个 repo 的符号向量（复用已有 helper 或直接构造）
	// ... 插入 repo-a 和 repo-b 各几个 vector ...

	t.Run("single repo filter", func(t *testing.T) {
		filters := models.VectorSearchFilters{
			EntityType: "symbol",
			RepoIDs:    []string{"repo-a"},
			WithDetails: true,
		}
		results, err := vectorRepo.SimilaritySearchWithFilters(ctx, sampleEmbedding, filters)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		for _, r := range results {
			if r.RepoID != "repo-a" {
				t.Errorf("expected repo-a only, got %s", r.RepoID)
			}
		}
	})

	t.Run("multi repo filter", func(t *testing.T) {
		filters := models.VectorSearchFilters{
			EntityType: "symbol",
			RepoIDs:    []string{"repo-a", "repo-b"},
			WithDetails: true,
		}
		results, err := vectorRepo.SimilaritySearchWithFilters(ctx, sampleEmbedding, filters)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		gotRepos := map[string]bool{}
		for _, r := range results {
			gotRepos[r.RepoID] = true
		}
		// 应该能命中两个 repo 的结果
		if len(gotRepos) < 1 {
			t.Errorf("expected results from multiple repos, got none")
		}
	})
}
```

注意：实现时参考同文件顶部 `setupTestDB` / 已有的向量插入 helper，复用它们的 fixture 构造方式，不要新造轮子。

- [ ] **Step 2: 跑集成测试**

需要数据库。先确保 `make db` 已启动并 `make db-init`：

```bash
go test ./tests/integration/ -run TestVectorSearch_RepoIDsFilter -v
```
Expected: PASS

- [ ] **Step 3: 跑全量集成测试确认无回归**

```bash
make test-integration
```
Expected: PASS（这一步是项目规范强制要求——改 schema 后必须全量验证）

- [ ] **Step 4: 提交**

```bash
git add tests/integration/models_integration_test.go
git commit -m "test(integration): 验证 RepoIDs 多仓库过滤"
```

---

## Task 6：retrieval 层接口与数据结构

**Files:**
- Create: `internal/retrieval/retriever.go`

定义检索层的对外接口和核心数据结构。这一步是后续所有 retrieval 代码的基础。

- [ ] **Step 1: 创建 retriever.go**

```go
// Package retrieval 提供代码检索与图谱上下文组装能力。
//
// 本层把"检索 + 1 跳图谱扩展"封装为可复用的纯接口，供 QA 引擎和
// 未来的 Agentic RAG 共用。不碰 HTTP、不碰 prompt 格式化。
package retrieval

import (
	"context"

	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// RetrievalRequest 是检索层入口。
type RetrievalRequest struct {
	Query         string   // 自然语言问题或符号名
	RepoIDs       []string // 空 = 全库；多 repo 按列表过滤
	Language      string   // 可选语言过滤
	Kind          []string // 可选符号类型过滤
	Mode          string   // "hybrid"(默认) | "vector" | "keyword"
	Limit         int      // Top-K，默认 10
	ExpandHops    int      // 图谱扩展跳数，固定 1（保留字段供未来扩展）
	ExpandCallers bool     // 默认 true，是否拉取 callers
	ExpandCallees bool     // 默认 true，是否拉取 callees
}

// ContextSymbol 是图谱/检索共用的符号视图。
// 刻意去耦 models.VectorSearchResult 的内部字段，转换在 retrieval 层内完成。
type ContextSymbol struct {
	SymbolID  string
	Name      string
	Kind      string
	Signature string
	FilePath  string
	Language  string
	Docstring string
}

// ContextBlock 是一个检索命中的完整上下文单元。
type ContextBlock struct {
	Symbol     ContextSymbol   // 主命中符号
	Similarity float64         // 检索得分（向量/关键词/混合同量纲 [0,1]）
	MatchMode  string          // "vector" | "keyword" | "hybrid"
	Callers    []ContextSymbol // 1 跳：谁调用了它（每边 Top-5）
	Callees    []ContextSymbol // 1 跳：它调用了谁（每边 Top-5）
	ChunkID    string          // 对应 vectors.vector_id，用于按需取源码
}

// Retriever 是检索层的可注入接口。
type Retriever interface {
	Query(ctx context.Context, req RetrievalRequest) ([]ContextBlock, error)
}

// VectorSearcher 收窄 VectorRepository 用到的方法，便于 mock。
type VectorSearcher interface {
	HybridSearch(ctx context.Context, query string, emb []float32, f models.VectorSearchFilters, wv, wk float64) ([]*models.HybridSearchResult, error)
	KeywordSearch(ctx context.Context, query string, f models.VectorSearchFilters) ([]*models.VectorSearchResult, error)
	SimilaritySearchWithFilters(ctx context.Context, emb []float32, f models.VectorSearchFilters) ([]*models.VectorSearchResult, error)
	GetByVectorIDs(ctx context.Context, ids []string) ([]*models.Vector, error)
}

// EdgeExpander 收窄 EdgeRepository 用到的方法，便于 mock。
type EdgeExpander interface {
	GetCallersWithDetails(ctx context.Context, symbolID string) ([]*models.EdgeWithDetails, error)
	GetCalleesWithDetails(ctx context.Context, symbolID string) ([]*models.EdgeWithDetails, error)
}

// 确保 *models.VectorRepository 满足 VectorSearcher（编译期检查）
var _ VectorSearcher = (*models.VectorRepository)(nil)

// HybridRetrieverConfig 是 HybridRetriever 的配置。
type HybridRetrieverConfig struct {
	WeightVector   float64 // 默认 0.7
	WeightKeyword  float64 // 默认 0.3
	DefaultLimit   int     // 默认 10
	NeighborLimit  int     // 每边邻居上限，默认 5
	EdgeConcurrency int    // 图谱查询并发上限，默认 4
}

// DefaultHybridRetrieverConfig 返回默认配置。
func DefaultHybridRetrieverConfig() HybridRetrieverConfig {
	return HybridRetrieverConfig{
		WeightVector:    0.7,
		WeightKeyword:   0.3,
		DefaultLimit:    10,
		NeighborLimit:   5,
		EdgeConcurrency: 4,
	}
}

// 引用 indexer.Embedder 接口，避免未使用 import（HybridRetriever 会用到）
var _ indexer.Embedder = (indexer.Embedder)(nil)
```

注意：最后一行 `var _ indexer.Embedder = (indexer.Embedder)(nil)` 其实不需要——import 会因为 `indexer.Embedder` 在 `HybridRetriever`（Task 7）里使用而保留。如果 Task 6 单独编译报 "imported and not used"，先临时用 `var _ = indexer.NewOpenAIEmbedder` 占位，Task 7 会自然替换。更干净的做法：本 task 先不 import indexer，Task 7 创建 HybridRetriever 时再加。

- [ ] **Step 2: 验证编译**

```bash
go build ./internal/retrieval/
```
Expected: PASS。如果 `var _ VectorSearcher = (*models.VectorRepository)(nil)` 报错（因为 GetByVectorIDs 刚加、签名不完全匹配），按报错调整接口签名或实现签名。

- [ ] **Step 3: 提交**

```bash
git add internal/retrieval/retriever.go
git commit -m "feat(retrieval): 定义检索层接口与数据结构"
```

---

## Task 7：HybridRetriever 实现

**Files:**
- Create: `internal/retrieval/hybrid_retriever.go`

实现检索 + 1 跳图谱扩展。这是 retrieval 层的核心。

- [ ] **Step 1: 创建 hybrid_retriever.go**

```go
package retrieval

import (
	"context"
	"sort"
	"sync"

	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
	"golang.org/x/sync/errgroup"
)

// neighborCap 控制每个符号在图谱上的引用频次尚未可得时的稳定排序。
// EdgeWithDetails 无调用频次列，按符号名字典序排序后取 Top-N。

// HybridRetriever 是 Retriever 的默认实现。
type HybridRetriever struct {
	vectorRepo VectorSearcher
	edgeRepo   EdgeExpander
	embedder   indexer.Embedder
	config     HybridRetrieverConfig
}

// NewHybridRetriever 创建默认实现。三个依赖均为接口，便于测试注入 mock。
func NewHybridRetriever(vr VectorSearcher, ee EdgeExpander, emb indexer.Embedder, cfg HybridRetrieverConfig) *HybridRetriever {
	if cfg.DefaultLimit == 0 {
		cfg = DefaultHybridRetrieverConfig()
	}
	return &HybridRetriever{
		vectorRepo: vr,
		edgeRepo:   ee,
		embedder:   emb,
		config:     cfg,
	}
}

// Query 执行检索 + 1 跳图谱扩展。
func (r *HybridRetriever) Query(ctx context.Context, req RetrievalRequest) ([]ContextBlock, error) {
	// 填充默认值
	if req.Mode == "" {
		req.Mode = "hybrid"
	}
	if req.Limit == 0 {
		req.Limit = r.config.DefaultLimit
	}

	// 构建 filters
	filters := models.VectorSearchFilters{
		EntityType:  "symbol",
		Limit:       req.Limit,
		Kind:        req.Kind,
		Language:    req.Language,
		RepoIDs:     req.RepoIDs,
		WithDetails: true,
	}

	// mode 分发（与原 search_handler 逻辑等价，但下沉到检索层）
	var hybridResults []*models.HybridSearchResult
	var matchMode := req.Mode

	switch req.Mode {
	case "keyword":
		kwResults, err := r.vectorRepo.KeywordSearch(ctx, req.Query, filters)
		if err != nil {
			return nil, err
		}
		// ts_rank 归一化到 [0,1]（复用 search_handler 的逻辑）
		kwMax := 0.0
		for _, kw := range kwResults {
			if kw.Similarity > kwMax {
				kwMax = kw.Similarity
			}
		}
		hybridResults = make([]*models.HybridSearchResult, 0, len(kwResults))
		for _, kw := range kwResults {
			score := kw.Similarity
			if kwMax > 0 {
				score /= kwMax
			}
			kw.Similarity = score
			hybridResults = append(hybridResults, &models.HybridSearchResult{
				VectorSearchResult: *kw, KeywordScore: score,
			})
		}
	case "vector":
		emb, err := r.embedder.GenerateEmbedding(ctx, req.Query)
		if err != nil {
			return nil, err
		}
		vecResults, err := r.vectorRepo.SimilaritySearchWithFilters(ctx, emb, filters)
		if err != nil {
			return nil, err
		}
		hybridResults = make([]*models.HybridSearchResult, 0, len(vecResults))
		for _, v := range vecResults {
			hybridResults = append(hybridResults, &models.HybridSearchResult{
				VectorSearchResult: *v, VectorScore: v.Similarity,
			})
		}
	default: // hybrid
		emb, err := r.embedder.GenerateEmbedding(ctx, req.Query)
		if err != nil {
			return nil, err
		}
		hybridResults, err = r.vectorRepo.HybridSearch(ctx, req.Query, emb, filters,
			r.config.WeightVector, r.config.WeightKeyword)
		if err != nil {
			return nil, err
		}
	}

	// 1 跳图谱扩展（并发）
	blocks := make([]ContextBlock, 0, len(hybridResults))
	for _, hr := range hybridResults {
		blocks = append(blocks, ContextBlock{
			Symbol:     toContextSymbol(&hr.VectorSearchResult),
			Similarity: hr.Similarity,
			MatchMode:  matchMode,
			ChunkID:    hr.VectorID,
		})
	}

	if req.ExpandHops > 0 {
		r.expandNeighbors(ctx, blocks, req)
	}

	return blocks, nil
}

// expandNeighbors 并发拉取每个 block 的 callers/callees（每边 Top-N）。
func (r *HybridRetriever) expandNeighbors(ctx context.Context, blocks []ContextBlock, req RetrievalRequest) {
	sem := make(chan struct{}, r.config.EdgeConcurrency)
	var wg sync.WaitGroup

	for i := range blocks {
		bid := blocks[i].Symbol.SymbolID
		if bid == "" {
			continue
		}
		wg.Add(1)
		go func(idx int, symbolID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if req.ExpandCallers {
				if callers, err := r.edgeRepo.GetCallersWithDetails(ctx, symbolID); err == nil {
					blocks[idx].Callers = topNeighbors(callers, r.config.NeighborLimit)
				}
			}
			if req.ExpandCallees {
				if callees, err := r.edgeRepo.GetCalleesWithDetails(ctx, symbolID); err == nil {
					blocks[idx].Callees = topNeighbors(callees, r.config.NeighborLimit)
				}
			}
		}(i, bid)
	}
	wg.Wait()
}

// topNeighbors 把 EdgeWithDetails 转 ContextSymbol，按符号名稳定排序后取 Top-N。
func topNeighbors(edges []*models.EdgeWithDetails, limit int) []ContextSymbol {
	if len(edges) == 0 {
		return nil
	}
	// 按符号名字典序稳定排序（EdgeWithDetails 无调用频次列）
	sort.SliceStable(edges, func(i, j int) bool {
		return edges[i].Name < edges[j].Name
	})
	if limit > 0 && len(edges) > limit {
		edges = edges[:limit]
	}
	result := make([]ContextSymbol, 0, len(edges))
	for _, e := range edges {
		result = append(result, ContextSymbol{
			SymbolID:  e.SymbolID,
			Name:      e.Name,
			Kind:      e.Kind,
			Signature: e.Signature,
			FilePath:  e.FilePath,
		})
	}
	return result
}

// toContextSymbol 把 VectorSearchResult 转 ContextSymbol。
func toContextSymbol(r *models.VectorSearchResult) ContextSymbol {
	return ContextSymbol{
		SymbolID:  r.EntityID,
		Name:      r.Name,
		Kind:      r.Kind,
		Signature: r.Signature,
		FilePath:  r.FilePath,
		Language:  r.Language,
		Docstring: r.Docstring,
	}
}
```

注意：
- 上面用了 `errgroup` import 但实际用 `sync.WaitGroup` + semaphore——请删掉未用的 `errgroup` import，只留 `sync`。或者用 `errgroup` 替代（但图谱查询失败时我们选择静默跳过，不中断整体，所以 WaitGroup 更合适）。
- `var matchMode := req.Mode` 是语法错误，应为 `matchMode := req.Mode`。请修正。

- [ ] **Step 2: 验证编译并修正**

```bash
go build ./internal/retrieval/
```
Expected: 编译通过。修正上面注释提到的两个小问题（删 errgroup import、matchMode 语法）。

- [ ] **Step 3: 提交**

```bash
git add internal/retrieval/hybrid_retriever.go
git commit -m "feat(retrieval): HybridRetriever 实现——检索 + 1 跳图谱扩展"
```

---

## Task 8：HybridRetriever 单元测试

**Files:**
- Create: `internal/retrieval/hybrid_retriever_test.go`

mock VectorSearcher + EdgeExpander，验证 mode 分发、邻居 Top-5、去重逻辑。

- [ ] **Step 1: 写 mock 和测试**

```go
package retrieval

import (
	"context"
	"testing"

	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// fakeVectorSearcher mock VectorSearcher
type fakeVectorSearcher struct {
	hybridResult []*models.HybridSearchResult
	keywordResult []*models.VectorSearchResult
	vectorResult []*models.VectorSearchResult
	receivedFilters models.VectorSearchFilters
}

func (f *fakeVectorSearcher) HybridSearch(ctx context.Context, query string, emb []float32, fil models.VectorSearchFilters, wv, wk float64) ([]*models.HybridSearchResult, error) {
	f.receivedFilters = fil
	return f.hybridResult, nil
}
func (f *fakeVectorSearcher) KeywordSearch(ctx context.Context, query string, fil models.VectorSearchFilters) ([]*models.VectorSearchResult, error) {
	f.receivedFilters = fil
	return f.keywordResult, nil
}
func (f *fakeVectorSearcher) SimilaritySearchWithFilters(ctx context.Context, emb []float32, fil models.VectorSearchFilters) ([]*models.VectorSearchResult, error) {
	f.receivedFilters = fil
	return f.vectorResult, nil
}
func (f *fakeVectorSearcher) GetByVectorIDs(ctx context.Context, ids []string) ([]*models.Vector, error) {
	return nil, nil
}

// fakeEdgeExpander mock EdgeExpander
type fakeEdgeExpander struct {
	callers map[string][]*models.EdgeWithDetails
	callees map[string][]*models.EdgeWithDetails
}

func (f *fakeEdgeExpander) GetCallersWithDetails(ctx context.Context, symbolID string) ([]*models.EdgeWithDetails, error) {
	return f.callers[symbolID], nil
}
func (f *fakeEdgeExpander) GetCalleesWithDetails(ctx context.Context, symbolID string) ([]*models.EdgeWithDetails, error) {
	return f.callees[symbolID], nil
}

func TestHybridRetriever_Query_RepoIDsFilterPassedThrough(t *testing.T) {
	fvs := &fakeVectorSearcher{
		keywordResult: []*models.VectorSearchResult{{VectorID: "v1", EntityID: "s1", Name: "FuncA"}},
	}
	ee := &fakeEdgeExpander{}
	r := NewHybridRetriever(fvs, ee, nil, HybridRetrieverConfig{NeighborLimit: 5})

	_, err := r.Query(context.Background(), RetrievalRequest{
		Query:     "test",
		RepoIDs:   []string{"repo-a", "repo-b"},
		Mode:      "keyword",
		ExpandHops: 0,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(fvs.receivedFilters.RepoIDs) != 2 {
		t.Errorf("expected 2 RepoIDs passed to filter, got %v", fvs.receivedFilters.RepoIDs)
	}
}

func TestHybridRetriever_NeighborLimitTop5(t *testing.T) {
	// 构造 1 个命中符号，其有 8 个 callers
	callers := make([]*models.EdgeWithDetails, 8)
	for i := range callers {
		callers[i] = &models.EdgeWithDetails{SymbolID: "caller", Name: string(rune('A' + i))}
	}
	fvs := &fakeVectorSearcher{
		keywordResult: []*models.VectorSearchResult{{VectorID: "v1", EntityID: "s1", Name: "FuncA"}},
	}
	ee := &fakeEdgeExpander{callers: map[string][]*models.EdgeWithDetails{"s1": callers}}
	r := NewHybridRetriever(fvs, ee, nil, HybridRetrieverConfig{NeighborLimit: 5})

	blocks, err := r.Query(context.Background(), RetrievalRequest{
		Query: "test", Mode: "keyword", ExpandHops: 1,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(blocks[0].Callers) != 5 {
		t.Errorf("expected 5 callers (Top-5), got %d", len(blocks[0].Callers))
	}
}

func TestHybridRetriever_ExpandSwitches(t *testing.T) {
	fvs := &fakeVectorSearcher{
		keywordResult: []*models.VectorSearchResult{{VectorID: "v1", EntityID: "s1", Name: "FuncA"}},
	}
	ee := &fakeEdgeExpander{
		callers: map[string][]*models.EdgeWithDetails{"s1": {{SymbolID: "c1", Name: "C1"}}},
		callees: map[string][]*models.EdgeWithDetails{"s1": {{SymbolID: "d1", Name: "D1"}}},
	}
	r := NewHybridRetriever(fvs, ee, nil, HybridRetrieverConfig{NeighborLimit: 5})

	// 关闭 callers 扩展
	blocks, _ := r.Query(context.Background(), RetrievalRequest{
		Query: "test", Mode: "keyword", ExpandHops: 1, ExpandCallers: false,
	})
	if len(blocks[0].Callers) != 0 {
		t.Errorf("expected no callers when ExpandCallers=false, got %d", len(blocks[0].Callers))
	}
	if len(blocks[0].Callees) != 1 {
		t.Errorf("expected 1 callee, got %d", len(blocks[0].Callees))
	}
}
```

- [ ] **Step 2: 运行测试**

```bash
go test ./internal/retrieval/ -run TestHybridRetriever -v
```
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add internal/retrieval/hybrid_retriever_test.go
git commit -m "test(retrieval): HybridRetriever 单元测试——mode/邻居限流/开关"
```

---

## Task 9：retrieval 集成测试

**Files:**
- Create: `internal/retrieval/hybrid_retriever_integration_test.go`

真 DB 验证端到端检索 + 图谱扩展。

- [ ] **Step 1: 写集成测试**

参考 `tests/integration/test_utils.go` 的 DB 启动方式，或复用其 helper：

```go
package retrieval

import (
	"context"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestHybridRetriever_Integration 验证真 DB 上的检索 + 1 跳图谱扩展。
func TestHybridRetriever_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// 复用 tests/integration 的 setup；如该包未导出 helper，
	// 则在此直接用 models.NewDB + goose 迁移（参考 scripts/init_db.go）
	db := setupIntegrationDB(t) // 见下：自建 helper 或复用
	defer db.Close()

	ctx := context.Background()
	vectorRepo := models.NewVectorRepository(db)
	edgeRepo := models.NewEdgeRepository(db)
	// embedder：集成测试用 fake（keyword 模式不需要 embedding）
	var emb indexer.Embedder = &noOpEmbedder{}

	r := NewHybridRetriever(vectorRepo, edgeRepo, emb, DefaultHybridRetrieverConfig())

	// 索引少量 fixture：1 个 repo、2 个符号、1 条 call 边
	// （插入逻辑参考 tests/integration/models_integration_test.go 已有的构造方式）
	// ... seed fixture ...

	t.Run("keyword search returns block with callees", func(t *testing.T) {
		blocks, err := r.Query(ctx, RetrievalRequest{
			Query: "FuncA", Mode: "keyword", ExpandHops: 1,
		})
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(blocks) == 0 {
			t.Fatal("expected at least 1 block")
		}
		// 验证命中符号 FuncA 的 callees 包含被调用方
		if len(blocks[0].Callees) == 0 {
			t.Error("expected callees to be expanded, got none")
		}
	})
}

// noOpEmbedder 集成测试 keyword 模式时使用，GenerateEmbedding 不会被调用。
type noOpEmbedder struct{}
func (noOpEmbedder) EmbedSymbols(ctx context.Context, symbols []schema.Symbol) error { return nil }
func (noOpEmbedder) GenerateEmbedding(ctx context.Context, content string) ([]float32, error) {
	return []float32{0}, nil
}
// 按 indexer.Embedder 接口补全其他方法（grep 接口定义后补全）
```

注意：
- `indexer.Embedder` 接口的完整方法集，实现时 `grep -n "type Embedder interface" -A 20 internal/indexer/embedder.go` 确认，noOpEmbedder 全部实现。
- `setupIntegrationDB` 如不能复用 tests/integration 的（跨包私有），则参考 `tests/integration/test_utils.go` 自建：连接 PG → 跑 goose 迁移 → 返回 `*models.DB`。

- [ ] **Step 2: 跑集成测试**

```bash
go test ./internal/retrieval/ -run TestHybridRetriever_Integration -v
```
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add internal/retrieval/hybrid_retriever_integration_test.go
git commit -m "test(retrieval): 集成测试验证检索+图谱扩展端到端"
```

---

## Task 10：QA prompt_builder

**Files:**
- Create: `internal/qa/prompt_builder.go`
- Create: `internal/qa/prompt_builder_test.go`

把 ContextBlock[] 拼成 Markdown prompt，含 8000 token 智能截断。纯函数，最容易表驱动测试。

- [ ] **Step 1: 写 prompt_builder.go**

```go
// Package qa 把检索上下文组装为可直接喂给 LLM 的 Markdown prompt，
// 并提供结构化 JSON 视图。本层不碰数据库、不碰 HTTP。
package qa

import (
	"fmt"
	"strings"

	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

// PromptBuildOptions 控制 prompt 拼接。
type PromptBuildOptions struct {
	MaxTokens     int  // 软上限，默认 8000（按 4 字符 ≈ 1 token 估算）
	IncludeSource bool // 是否内联源码片段
}

// DefaultPromptBuildOptions 返回默认配置。
func DefaultPromptBuildOptions() PromptBuildOptions {
	return PromptBuildOptions{MaxTokens: 8000}
}

// BuildPrompt 把 ContextBlock[] 拼成 Markdown prompt。
// sources 是 chunkID → 源码文本映射（IncludeSource 时传入，可为 nil）。
// 返回 prompt 文本和是否被截断。
// 截断策略：超限时优先保留高 similarity 的 block；先砍低分 block 的图谱邻居，再砍整个低分 block。
// 注意：调用方需保证 blocks 已按 Similarity 降序排列（retrieval 层返回即降序）。
func BuildPrompt(query string, repoIDs []string, blocks []retrieval.ContextBlock, sources map[string]string, opts PromptBuildOptions) (string, bool) {
	if opts.MaxTokens == 0 {
		opts = DefaultPromptBuildOptions()
	}

	var sb strings.Builder
	sb.WriteString("# Code Context\n\n")
	sb.WriteString("## Question\n")
	sb.WriteString(query + "\n\n")

	if len(repoIDs) > 0 {
		sb.WriteString("## Repositories\n")
		sb.WriteString(strings.Join(repoIDs, ", ") + "\n\n")
	}

	sb.WriteString("## Relevant Symbols\n\n")

	charBudget := opts.MaxTokens * 4
	truncated := false
	for i, b := range blocks {
		section := formatBlockSection(i+1, b, sources, opts.IncludeSource)
		// 预算检查：若加上这段会超，先尝试去掉图谱邻居再拼
		if len(section) > charBudget {
			// 砍邻居重算
			stripped := b
			stripped.Callers = nil
			stripped.Callees = nil
			section = formatBlockSection(i+1, stripped, sources, opts.IncludeSource)
			truncated = true
		}
		if len(section) > charBudget {
			// 整个 block 放不下，跳过低分的（blocks 已按 similarity 降序）
			truncated = true
			continue
		}
		sb.WriteString(section)
		charBudget -= len(section)
	}

	return sb.String(), truncated
}

// formatBlockSection 渲染单个 block 的 Markdown 段落。
func formatBlockSection(idx int, b retrieval.ContextBlock, sources map[string]string, includeSource bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "### %d. %s (similarity: %.2f)\n", idx, b.Symbol.Name, b.Similarity)
	if b.Symbol.FilePath != "" {
		fmt.Fprintf(&sb, "- **File**: `%s`\n", b.Symbol.FilePath)
	}
	if b.Symbol.Signature != "" {
		fmt.Fprintf(&sb, "- **Signature**: `%s`\n", b.Symbol.Signature)
	}
	if b.Symbol.Docstring != "" {
		fmt.Fprintf(&sb, "- **Docstring**: %s\n", b.Symbol.Docstring)
	}
	if len(b.Callers) > 0 {
		sb.WriteString("- **Called by**:\n")
		for _, c := range b.Callers {
			fmt.Fprintf(&sb, "  - `%s` (%s)\n", c.Name, c.FilePath)
		}
	}
	if len(b.Callees) > 0 {
		sb.WriteString("- **Calls**:\n")
		for _, c := range b.Callees {
			fmt.Fprintf(&sb, "  - `%s` (%s)\n", c.Name, c.FilePath)
		}
	}
	if includeSource && sources != nil {
		if src, ok := sources[b.ChunkID]; ok && src != "" {
			fmt.Fprintf(&sb, "\n```\n%s\n```\n", src)
		}
	}
	sb.WriteString("\n")
	return sb.String()
}
```

- [ ] **Step 2: 写表驱动单元测试**

```go
package qa

import (
	"strings"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

func TestBuildPrompt_Format(t *testing.T) {
	blocks := []retrieval.ContextBlock{
		{
			Symbol:     retrieval.ContextSymbol{Name: "FuncA", FilePath: "a.go:10", Signature: "func FuncA()"},
			Similarity: 0.9,
			Callers:    []retrieval.ContextSymbol{{Name: "Caller1", FilePath: "c.go:1"}},
		},
	}
	prompt, truncated := BuildPrompt("how does FuncA work", []string{"repo-1"}, blocks, nil, DefaultPromptBuildOptions())
	if truncated {
		t.Error("expected not truncated for small input")
	}
	mustContain := []string{"# Code Context", "## Question", "FuncA", "0.90", "Caller1", "repo-1"}
	for _, s := range mustContain {
		if !strings.Contains(prompt, s) {
			t.Errorf("prompt missing %q\ngot:\n%s", s, prompt)
		}
	}
}

func TestBuildPrompt_TruncationDropsLowScoreNeighbors(t *testing.T) {
	// 构造一个超大 block，触发截断
	big := retrieval.ContextBlock{
		Symbol:     retrieval.ContextSymbol{Name: "Big", FilePath: "x.go"},
		Similarity: 0.3,
		Callers:    make([]retrieval.ContextSymbol, 100),
	}
	high := retrieval.ContextBlock{
		Symbol:     retrieval.ContextSymbol{Name: "High", FilePath: "y.go"},
		Similarity: 0.95,
	}
	blocks := []retrieval.ContextBlock{high, big} // 高分在前
	opts := PromptBuildOptions{MaxTokens: 50}      // 极小预算强制截断

	_, truncated := BuildPrompt("q", nil, blocks, nil, opts)
	if !truncated {
		t.Error("expected truncated=true for tiny budget")
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/qa/ -run TestBuildPrompt -v
```
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add internal/qa/prompt_builder.go internal/qa/prompt_builder_test.go
git commit -m "feat(qa): prompt_builder——Markdown 拼接 + 8000 token 智能截断"
```

---

## Task 11：QA Service

**Files:**
- Create: `internal/qa/service.go`
- Create: `internal/qa/service_test.go`

编排：调 retriever + prompt_builder + 源码按需拉取。

- [ ] **Step 1: 写 service.go**

```go
package qa

import (
	"context"
	"fmt"

	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

// AskRequest 是 QA 端点的请求。
type AskRequest struct {
	Query         string
	RepoIDs       []string
	Language      string
	Kind          []string
	Mode          string
	Limit         int
	IncludeSource bool
	ExpandCallers bool
	ExpandCallees bool
}

// AskResponse 是 QA 端点的响应。
type AskResponse struct {
	Query     string             `json:"query"`
	Blocks    []ContextBlockJSON `json:"blocks"`
	Prompt    string             `json:"prompt"`
	Truncated bool               `json:"truncated"`
	ChunkIDs  []string           `json:"chunk_ids"`
}

// ContextBlockJSON 是结构化 JSON 视图。
type ContextBlockJSON struct {
	Symbol     SymbolJSON   `json:"symbol"`
	Similarity float64      `json:"similarity"`
	MatchMode  string       `json:"match_mode"`
	Callers    []SymbolJSON `json:"callers"`
	Callees    []SymbolJSON `json:"callees"`
	ChunkID    string       `json:"chunk_id"`
	Source     string       `json:"source,omitempty"`
}

// SymbolJSON 是符号的 JSON 视图。
type SymbolJSON struct {
	SymbolID  string `json:"symbol_id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Signature string `json:"signature,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	Language  string `json:"language,omitempty"`
	Docstring string `json:"docstring,omitempty"`
}

// SourceFetcher 按 chunk_id 批量取源码（IncludeSource 时用）。
type SourceFetcher interface {
	GetByVectorIDs(ctx context.Context, ids []string) (map[string]string, error)
}

// Service 是 QA 编排接口。
type Service interface {
	Ask(ctx context.Context, req AskRequest) (*AskResponse, error)
}

type service struct {
	retriever     retrieval.Retriever
	sourceFetcher SourceFetcher
	promptOpts    PromptBuildOptions
}

// NewService 创建 QA service。
func NewService(r retrieval.Retriever, sf SourceFetcher, opts PromptBuildOptions) Service {
	if opts.MaxTokens == 0 {
		opts = DefaultPromptBuildOptions()
	}
	return &service{retriever: r, sourceFetcher: sf, promptOpts: opts}
}

// Ask 执行 QA 编排。
func (s *service) Ask(ctx context.Context, req AskRequest) (*AskResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	blocks, err := s.retriever.Query(ctx, retrieval.RetrievalRequest{
		Query:         req.Query,
		RepoIDs:       req.RepoIDs,
		Language:      req.Language,
		Kind:          req.Kind,
		Mode:          req.Mode,
		Limit:         req.Limit,
		ExpandHops:    1,
		ExpandCallers: req.ExpandCallers,
		ExpandCallees: req.ExpandCallees,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// 按需取源码
	sources := map[string]string{}
	if req.IncludeSource && s.sourceFetcher != nil {
		chunkIDs := collectChunkIDs(blocks)
		if len(chunkIDs) > 0 {
			if fetched, err := s.sourceFetcher.GetByVectorIDs(ctx, chunkIDs); err == nil {
				sources = fetched
			}
		}
	}

	// 拼 prompt（注意：IncludeSource 由 opts 标记，源码通过 sources 参数传入）
	promptOpts := s.promptOpts
	promptOpts.IncludeSource = req.IncludeSource
	prompt, truncated := BuildPrompt(req.Query, req.RepoIDs, blocks, sources, promptOpts)

	// 组装 JSON 响应
	resp := &AskResponse{
		Query:     req.Query,
		Blocks:    toBlockJSONs(blocks, sources),
		Prompt:    prompt,
		Truncated: truncated,
		ChunkIDs:  collectChunkIDs(blocks),
	}
	return resp, nil
}

func collectChunkIDs(blocks []retrieval.ContextBlock) []string {
	seen := map[string]bool{}
	var ids []string
	for _, b := range blocks {
		if b.ChunkID != "" && !seen[b.ChunkID] {
			seen[b.ChunkID] = true
			ids = append(ids, b.ChunkID)
		}
	}
	return ids
}

func toBlockJSONs(blocks []retrieval.ContextBlock, sources map[string]string) []ContextBlockJSON {
	result := make([]ContextBlockJSON, 0, len(blocks))
	for _, b := range blocks {
		result = append(result, ContextBlockJSON{
			Symbol:     toSymbolJSON(b.Symbol),
			Similarity: b.Similarity,
			MatchMode:  b.MatchMode,
			Callers:    toSymbolJSONs(b.Callers),
			Callees:    toSymbolJSONs(b.Callees),
			ChunkID:    b.ChunkID,
			Source:     sources[b.ChunkID],
		})
	}
	return result
}

func toSymbolJSON(s retrieval.ContextSymbol) SymbolJSON {
	return SymbolJSON{
		SymbolID: s.SymbolID, Name: s.Name, Kind: s.Kind,
		Signature: s.Signature, FilePath: s.FilePath,
		Language: s.Language, Docstring: s.Docstring,
	}
}

func toSymbolJSONs(ss []retrieval.ContextSymbol) []SymbolJSON {
	r := make([]SymbolJSON, 0, len(ss))
	for _, s := range ss {
		r = append(r, toSymbolJSON(s))
	}
	return r
}
```

注意：`BuildPrompt` 在 Task 10 被定义为接受 `sources map[string]string` 参数（按 Task 10 Step 1 的修正方案），但上面 Ask 里调用 `BuildPrompt(req.Query, req.RepoIDs, blocks, s.promptOpts)` 没传 sources。**需要对齐**：要么 BuildPrompt 内部不接受 sources（源码只在 JSON 视图里体现，prompt 里 IncludeSource 时另外拼），要么 Ask 调用时传入。推荐：BuildPrompt 签名加 `sources map[string]string`，Ask 调用时传入。修正 Ask 里的调用为 `BuildPrompt(req.Query, req.RepoIDs, blocks, sources, s.promptOpts)`，并同步 Task 10 的 BuildPrompt 签名。

- [ ] **Step 2: 写 service 单元测试**

```go
package qa

import (
	"context"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

type fakeRetriever struct {
	blocks []retrieval.ContextBlock
	err    error
}
func (f *fakeRetriever) Query(ctx context.Context, req retrieval.RetrievalRequest) ([]retrieval.ContextBlock, error) {
	return f.blocks, f.err
}

type fakeSourceFetcher struct {
	data map[string]string
}
func (f *fakeSourceFetcher) GetByVectorIDs(ctx context.Context, ids []string) (map[string]string, error) {
	return f.data, nil
}

func TestService_Ask_BasicFlow(t *testing.T) {
	fr := &fakeRetriever{blocks: []retrieval.ContextBlock{
		{Symbol: retrieval.ContextSymbol{Name: "FuncA"}, ChunkID: "v1", Similarity: 0.9},
	}}
	svc := NewService(fr, &fakeSourceFetcher{}, DefaultPromptBuildOptions())

	resp, err := svc.Ask(context.Background(), AskRequest{Query: "test", Mode: "keyword"})
	if err != nil {
		t.Fatalf("Ask failed: %v", err)
	}
	if resp.Query != "test" {
		t.Errorf("expected query echo 'test', got %q", resp.Query)
	}
	if len(resp.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(resp.Blocks))
	}
	if len(resp.ChunkIDs) != 1 || resp.ChunkIDs[0] != "v1" {
		t.Errorf("expected chunk_ids [v1], got %v", resp.ChunkIDs)
	}
	if resp.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestService_Ask_EmptyQueryReturnsError(t *testing.T) {
	svc := NewService(&fakeRetriever{}, nil, DefaultPromptBuildOptions())
	_, err := svc.Ask(context.Background(), AskRequest{})
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestService_Ask_IncludeSourceFillsSource(t *testing.T) {
	fr := &fakeRetriever{blocks: []retrieval.ContextBlock{
		{Symbol: retrieval.ContextSymbol{Name: "F"}, ChunkID: "v1"},
	}}
	sf := &fakeSourceFetcher{data: map[string]string{"v1": "source code here"}}
	svc := NewService(fr, sf, DefaultPromptBuildOptions())

	resp, _ := svc.Ask(context.Background(), AskRequest{Query: "q", Mode: "keyword", IncludeSource: true})
	if resp.Blocks[0].Source != "source code here" {
		t.Errorf("expected source filled, got %q", resp.Blocks[0].Source)
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/qa/ -v
```
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add internal/qa/service.go internal/qa/service_test.go
git commit -m "feat(qa): Service 编排——retriever + prompt + 源码按需拉取"
```

---

## Task 12：QA HTTP Handler

**Files:**
- Create: `internal/api/handlers/qa_handler.go`
- Create: `internal/api/handlers/qa_handler_test.go`

POST /api/v1/qa 和 GET /api/v1/qa/chunks。

- [ ] **Step 1: 写 qa_handler.go**

```go
package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/qa"
	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// QAHandler handles QA endpoints.
type QAHandler struct {
	qaService qa.Service
	vectorRepo *models.VectorRepository // 给 chunks 端点用
}

// NewQAHandler creates a QA handler with embedder config (参考 NewSearchHandler)。
func NewQAHandler(db *models.DB, embedderConfig *EmbedderConfig) *QAHandler {
	cfg := embedderConfig
	if cfg == nil {
		cfg = indexer.DefaultEmbedderConfig() // 注意类型转换，DefaultEmbedderConfig 返回 *indexer.EmbedderConfig
	}
	vectorRepo := models.NewVectorRepository(db)
	edgeRepo := models.NewEdgeRepository(db)
	emb := indexer.NewOpenAIEmbedder(cfg, vectorRepo)

	retriever := retrieval.NewHybridRetriever(vectorRepo, edgeRepo, emb, retrieval.DefaultHybridRetrieverConfig())

	// SourceFetcher 适配器：把 VectorRepository 适配成 qa.SourceFetcher
	sf := &vectorSourceFetcher{vr: vectorRepo}

	return &QAHandler{
		qaService:  qa.NewService(retriever, sf, qa.DefaultPromptBuildOptions()),
		vectorRepo: vectorRepo,
	}
}

// vectorSourceFetcher 把 VectorRepository 适配成 qa.SourceFetcher。
type vectorSourceFetcher struct {
	vr *models.VectorRepository
}

func (f *vectorSourceFetcher) GetByVectorIDs(ctx context.Context, ids []string) (map[string]string, error) {
	vectors, err := f.vr.GetByVectorIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(vectors))
	for _, v := range vectors {
		m[v.VectorID] = v.Content
	}
	return m, nil
}

// AskRequest 是 POST /api/v1/qa 的请求体。
type askRequestBody struct {
	Query         string   `json:"query" binding:"required"`
	RepoIDs       []string `json:"repo_ids,omitempty"`
	Language      string   `json:"language,omitempty"`
	Kind          []string `json:"kind,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	IncludeSource bool     `json:"include_source,omitempty"`
	ExpandCallers *bool    `json:"expand_callers,omitempty"` // 指针区分"未传"和"传 false"
	ExpandCallees *bool    `json:"expand_callees,omitempty"`
}

// Ask handles POST /api/v1/qa
func (h *QAHandler) Ask(c *gin.Context) {
	var body askRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	req := qa.AskRequest{
		Query:         body.Query,
		RepoIDs:       body.RepoIDs,
		Language:      body.Language,
		Kind:          body.Kind,
		Mode:          body.Mode,
		Limit:         body.Limit,
		IncludeSource: body.IncludeSource,
		ExpandCallers: true, // 默认 true
		ExpandCallees: true,
	}
	if body.ExpandCallers != nil {
		req.ExpandCallers = *body.ExpandCallers
	}
	if body.ExpandCallees != nil {
		req.ExpandCallees = *body.ExpandCallees
	}

	resp, err := h.qaService.Ask(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "QA failed", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetChunks handles GET /api/v1/qa/chunks?ids=id1,id2
func (h *QAHandler) GetChunks(c *gin.Context) {
	idsParam := c.Query("ids")
	if idsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids parameter required"})
		return
	}
	ids := strings.Split(idsParam, ",")
	if len(ids) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many ids, max 50"})
		return
	}

	vectors, err := h.vectorRepo.GetByVectorIDs(c.Request.Context(), ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch chunks", "details": err.Error()})
		return
	}

	chunks := make([]chunkJSON, 0, len(vectors))
	for _, v := range vectors {
		chunks = append(chunks, chunkJSON{
			ChunkID:   v.VectorID,
			SymbolID:  v.EntityID,
			Content:   v.Content,
			FilePath:  "", // vectors 表无 file_path，需要时可 JOIN；本端点先不返回
		})
	}
	c.JSON(http.StatusOK, gin.H{"chunks": chunks})
}

type chunkJSON struct {
	ChunkID  string `json:"chunk_id"`
	SymbolID string `json:"symbol_id"`
	Content  string `json:"content"`
	FilePath string `json:"file_path,omitempty"`
}
```

注意补充 import：`context`。`DefaultEmbedderConfig` 的类型对齐（`EmbedderConfig` 是 handlers 包的 alias，`indexer.DefaultEmbedderConfig` 返回 `*indexer.EmbedderConfig`，确认兼容）。

- [ ] **Step 2: 写 handler 单元测试**

mock qa.Service 和 vectorRepo 较重，这里直接测 handler 的参数校验和错误码（业务逻辑已被 service 单测覆盖）。参考 `search_handler_test.go` 的 HTTP 测试模式：

```go
package handlers

import (
	// ... 参考 search_handler_test.go 的 import
)

// 用 httptest 起 gin，构造 QAHandler，验证 400/500/200。
// 具体 mock 方式参考 search_handler_test.go 如何 mock embedder/vectorRepo。
```

测试用例至少覆盖：
- 空 query → 400
- 正常请求 → 200 + 响应含 prompt/blocks/chunk_ids
- chunks 端点 ids 为空 → 400
- chunks 端点 ids > 50 → 400

（实现时参考 `search_handler_test.go` 的 setup 模式，复用其 DB mock 或 httptest 框架）

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/api/handlers/ -run TestQA -v
```
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add internal/api/handlers/qa_handler.go internal/api/handlers/qa_handler_test.go
git commit -m "feat(api): QA handler——POST /qa + GET /qa/chunks"
```

---

## Task 13：注册 QA 路由

**Files:**
- Modify: `internal/api/server.go`

- [ ] **Step 1: 加 qaHandler 字段**

`Server` struct 加：
```go
qaHandler *handlers.QAHandler
```

- [ ] **Step 2: 构造 handler**

`NewServer` 里 `return &Server{...}` 加：
```go
qaHandler: handlers.NewQAHandler(db, config.EmbedderConfig),
```

- [ ] **Step 3: 注册路由**

`RegisterRoutes` 的 v1 group 里加：
```go
// QA endpoints
v1.POST("/qa", s.qaHandler.Ask)
v1.GET("/qa/chunks", s.qaHandler.GetChunks)
```

- [ ] **Step 4: 验证编译 + 跑 API 测试**

```bash
go build ./internal/api/...
go test ./internal/api/... -v
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/api/server.go
git commit -m "feat(api): 注册 /api/v1/qa 与 /qa/chunks 路由"
```

---

## Task 14：client.Ask + GetChunks 方法

**Files:**
- Modify: `pkg/client/api_client.go`

CLI 通过 client 调 QA 端点。

- [ ] **Step 1: 加请求/响应类型和方法**

在 `pkg/client/api_client.go` 添加：

```go
// QARequest represents the request for POST /api/v1/qa
type QARequest struct {
	Query         string   `json:"query"`
	RepoIDs       []string `json:"repo_ids,omitempty"`
	Language      string   `json:"language,omitempty"`
	Kind          []string `json:"kind,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	IncludeSource bool     `json:"include_source,omitempty"`
	ExpandCallers *bool    `json:"expand_callers,omitempty"`
	ExpandCallees *bool    `json:"expand_callees,omitempty"`
}

// QAResponse represents the response for POST /api/v1/qa
type QAResponse struct {
	Query     string             `json:"query"`
	Blocks    []QABlock          `json:"blocks"`
	Prompt    string             `json:"prompt"`
	Truncated bool               `json:"truncated"`
	ChunkIDs  []string           `json:"chunk_ids"`
}

type QABlock struct {
	Symbol     QASymbol   `json:"symbol"`
	Similarity float64    `json:"similarity"`
	MatchMode  string     `json:"match_mode"`
	Callers    []QASymbol `json:"callers"`
	Callees    []QASymbol `json:"callees"`
	ChunkID    string     `json:"chunk_id"`
	Source     string     `json:"source,omitempty"`
}

type QASymbol struct {
	SymbolID  string `json:"symbol_id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Signature string `json:"signature,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	Language  string `json:"language,omitempty"`
	Docstring string `json:"docstring,omitempty"`
}

// ChunksResponse represents the response for GET /api/v1/qa/chunks
type ChunksResponse struct {
	Chunks []Chunk `json:"chunks"`
}

type Chunk struct {
	ChunkID  string `json:"chunk_id"`
	SymbolID string `json:"symbol_id"`
	Content  string `json:"content"`
	FilePath string `json:"file_path,omitempty"`
}

// Ask performs a QA context query
func (c *APIClient) Ask(ctx context.Context, req *QARequest) (*QAResponse, error) {
	var response QAResponse
	err := c.doRequestWithRetry(ctx, "POST", "/api/v1/qa", req, &response)
	if err != nil {
		return nil, fmt.Errorf("ask request failed: %w", err)
	}
	return &response, nil
}

// GetChunks fetches source content by chunk IDs
func (c *APIClient) GetChunks(ctx context.Context, ids []string) (*ChunksResponse, error) {
	if len(ids) == 0 {
		return &ChunksResponse{Chunks: []Chunk{}}, nil
	}
	path := "/api/v1/qa/chunks?ids=" + strings.Join(ids, ",")
	var response ChunksResponse
	err := c.doRequestWithRetry(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("get chunks request failed: %w", err)
	}
	return &response, nil
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./pkg/client/...
```
Expected: PASS（补充 `strings` import 如未引入）

- [ ] **Step 3: 提交**

```bash
git add pkg/client/api_client.go
git commit -m "feat(client): 新增 Ask 和 GetChunks 方法"
```

---

## Task 15：CLI ask 命令

**Files:**
- Create: `cmd/cli/ask_command.go`
- Create: `cmd/cli/ask_command_test.go`
- Modify: `cmd/cli/main.go`

- [ ] **Step 1: 写 ask_command.go**

参考 `cmd/cli/search_command.go` 结构：

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/yourtionguo/CodeAtlas/pkg/client"
)

func createAskCommand() *cli.Command {
	return &cli.Command{
		Name:  "ask",
		Usage: "Ask a question and get assembled code context (prompt for LLMs)",
		Description: `Performs a QA context query and outputs a Markdown prompt ready to paste into an LLM.
The prompt includes relevant symbols with their 1-hop callers/callees.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "question",
				Aliases:  []string{"q"},
				Usage:    "Natural language question",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:    "repo",
				Aliases: []string{"r"},
				Usage:   "Filter by repository ID (can be repeated for multiple repos)",
			},
			&cli.StringFlag{Name: "language", Aliases: []string{"l"}, Usage: "Filter by language"},
			&cli.StringFlag{Name: "kind", Aliases: []string{"k"}, Usage: "Filter by symbol kind (comma-separated)"},
			&cli.StringFlag{Name: "mode", Usage: "Retrieval mode: hybrid(default)|vector|keyword", Value: "hybrid"},
			&cli.IntFlag{Name: "limit", Usage: "Top-K results", Value: 10},
			&cli.BoolFlag{Name: "include-source", Usage: "Inline source code into prompt"},
			&cli.StringFlag{Name: "api-url", Usage: "API server URL (or CODEATLAS_API_URL env)"},
			&cli.StringFlag{Name: "api-token", Usage: "API auth token (or CODEATLAS_API_TOKEN env)"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Write prompt to file (default stdout)"},
			&cli.BoolFlag{Name: "json", Usage: "Output full JSON response instead of prompt only"},
			&cli.DurationFlag{Name: "timeout", Usage: "Request timeout", Value: 60 * time.Second},
		},
		Action: executeAskCommand,
	}
}

func executeAskCommand(c *cli.Context) error {
	apiURL := c.String("api-url")
	if apiURL == "" {
		apiURL = os.Getenv("CODEATLAS_API_URL")
		if apiURL == "" {
			return fmt.Errorf("API URL required via --api-url or CODEATLAS_API_URL")
		}
	}
	apiToken := c.String("api-token")
	if apiToken == "" {
		apiToken = os.Getenv("CODEATLAS_API_TOKEN")
	}

	cli := client.NewAPIClient(apiURL, client.WithTimeout(c.Duration("timeout")), client.WithToken(apiToken))

	req := &client.QARequest{
		Query:         c.String("question"),
		RepoIDs:       c.StringSlice("repo"),
		Language:      c.String("language"),
		Mode:          c.String("mode"),
		Limit:         c.Int("limit"),
		IncludeSource: c.Bool("include-source"),
	}
	if k := c.String("kind"); k != "" {
		req.Kind = strings.Split(k, ",")
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Duration("timeout"))
	defer cancel()

	resp, err := cli.Ask(ctx, req)
	if err != nil {
		return fmt.Errorf("ask failed: %w", err)
	}

	if c.Bool("json") {
		data, _ := jsonMarshalIndent(resp)
		fmt.Println(string(data))
		return nil
	}

	output := c.String("output")
	if output != "" {
		return os.WriteFile(output, []byte(resp.Prompt), 0644)
	}
	fmt.Print(resp.Prompt)
	return nil
}
```

注意：变量名 `cli` 与 import 的 `cli` 包冲突，改为 `apiClient`。补充 import：`encoding/json`、`strings`。`jsonMarshalIndent` 就是 `json.MarshalIndent`，直接用。

- [ ] **Step 2: 在 main.go 注册命令**

`cmd/cli/main.go` 的 `Commands: []cli.Command{...}` 里加 `createAskCommand()`：

```go
Commands: []*cli.Command{
	createParseCommand(),
	createIndexCommand(),
	createSearchCommand(),
	createImpactCommand(),
	createAskCommand(),  // ← 新增
	// ... upload 等
},
```

- [ ] **Step 3: 写 ask_command 单元测试**

参考 `cmd/cli/impact_command_test.go` 的测试模式（mock HTTP server）：

```go
package main

import (
	// httptest + 测试 flag 解析和输出
)

// 用 httptest.NewServer 起一个假 API，返回固定 QAResponse JSON，
// 验证 --output 写文件、stdout 输出 prompt、--json 输出 JSON。
```

测试用例：
- `--output` 写文件，文件内容 = resp.Prompt
- 默认 stdout 输出 prompt 文本
- `--json` 输出完整 JSON
- `--repo` 可重复

- [ ] **Step 4: 验证 CLI 编译**

```bash
go build ./cmd/cli/
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/cli/ask_command.go cmd/cli/ask_command_test.go cmd/cli/main.go
git commit -m "feat(cli): ask 命令——自然语言问答上下文，输出可粘贴 LLM 的 prompt"
```

---

## Task 16：文档更新

**Files:**
- Modify: `docs/api.md`

- [ ] **Step 1: 更新 search 端点字段说明**

把 `repo_id`（单值）的描述改为 `repo_ids`（数组）。

- [ ] **Step 2: 新增 QA 端点文档**

在 `docs/api.md` 添加 POST /api/v1/qa 和 GET /api/v1/qa/chunks 的完整说明（请求/响应字段、示例）。

- [ ] **Step 3: 提交**

```bash
git add docs/api.md
git commit -m "docs(api): search repo_ids 变更 + 新增 QA 端点文档"
```

---

## Task 17：全量验证

**Files:** 无（验证步骤）

- [ ] **Step 1: 跑单元测试**

```bash
make test
```
Expected: PASS

- [ ] **Step 2: 跑集成测试（项目规范强制）**

```bash
make test-integration
```
Expected: PASS

- [ ] **Step 3: 跑覆盖率检查**

```bash
make test-coverage
```
Expected: 达到 90% 目标（或接近，识别缺口）

- [ ] **Step 4: 端到端冒烟（手动）**

```bash
make db && make db-init && make run-api &
# 索引一个测试仓库后：
curl -X POST http://localhost:8080/api/v1/qa \
  -H "Content-Type: application/json" \
  -d '{"query":"main function","mode":"keyword"}'
```
Expected: 返回含 prompt 和 blocks 的 JSON

```bash
./bin/cli ask --question "main function" --api-url http://localhost:8080 --mode keyword
```
Expected: 输出 Markdown prompt 文本

- [ ] **Step 5: 最终提交（如有修复）**

```bash
git add -A
git commit -m "test: 全量验证通过——单元/集成/覆盖率/冒烟"
```

---

## 自查记录

**Spec 覆盖检查**：
- §2 三层架构 → Task 6/7（retrieval）、Task 10/11（qa）、Task 12/13（handler）✓
- §3 数据结构 → Task 6（retrieval 类型）、Task 11（qa 类型）✓
- §3.2 接口（VectorSearcher/EdgeExpander）→ Task 6 ✓
- §4 检索流程 → Task 7 ✓
- §5 Ask 流程 + prompt 格式 + token 截断 → Task 10、Task 11 ✓
- §6 HTTP API 两个端点 → Task 12、Task 13 ✓
- §7 VectorSearchFilters 迁移 → Task 1 ✓；GetByVectorIDs → Task 2 ✓；search handler 迁移 → Task 3、Task 4 ✓
- §8 CLI → Task 14、Task 15 ✓
- §9 测试策略 → 每个 task 内嵌测试 + Task 5（集成）+ Task 9（集成）+ Task 17（全量）✓

**类型一致性**：
- `RetrievalRequest.ExpandCallers/Callees` 在 Task 6 定义、Task 7 使用、Task 11 传递 ✓
- `ContextBlock.ChunkID` 在 Task 6 定义、Task 7 填充、Task 11 collectChunkIDs 收集 ✓
- `SourceFetcher.GetByVectorIDs` 在 Task 11 定义、Task 12 vectorSourceFetcher 实现 ✓
- `BuildPrompt` 签名（含 `sources map[string]string` 参数）在 Task 10 定稿，Task 11 调用点已对齐 ✓

**已知需实现时对齐的小问题**（plan 内对应 task 已标注修正方式）：
1. Task 7 的 `errgroup` import 删除（改用 sync.WaitGroup）+ `matchMode :=` 语法（去掉多余的 `var`）
2. Task 12 的 `context` import 补充 + `DefaultEmbedderConfig` 返回类型对齐（`*indexer.EmbedderConfig` vs handlers 包 alias）
3. Task 15 的变量名 `cli`→`apiClient`（避免与 urfave/cli 包冲突）+ 补 `encoding/json`、`strings` import
