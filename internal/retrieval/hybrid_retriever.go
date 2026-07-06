package retrieval

import (
	"context"
	"sort"
	"sync"

	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// 编译期断言：确保 *HybridRetriever 满足 Retriever 接口。
// Query 方法签名与接口不一致时编译即失败。
var _ Retriever = (*HybridRetriever)(nil)

// HybridRetriever 是 Retriever 的默认实现：mode 分发 + 1 跳图谱扩展。
//
// 它把原本散落在 search_handler 里的检索逻辑下沉为可复用的纯逻辑层，
// 并在其上叠加"1 跳 callers/callees"的图谱上下文，供 QA 引擎拼装 prompt。
//
// 三个依赖均为接口，便于单测注入 mock：
//   - vectorRepo：向量/关键词/混合召回
//   - edgeRepo：1 跳图谱（callers/callees）
//   - embedder：把 query 文本转向量
type HybridRetriever struct {
	vectorRepo VectorSearcher
	edgeRepo   EdgeExpander
	embedder   indexer.Embedder
	config     HybridRetrieverConfig
}

// NewHybridRetriever 构造一个 HybridRetriever。
// config 传入零值时会用 DefaultHybridRetrieverConfig 兜底。
func NewHybridRetriever(vectorRepo VectorSearcher, edgeRepo EdgeExpander, embedder indexer.Embedder, config HybridRetrieverConfig) *HybridRetriever {
	// 关键配置缺省时回退默认，避免运行期出现 limit=0 / 并发=0 等异常。
	if config.DefaultLimit <= 0 {
		config.DefaultLimit = 10
	}
	if config.NeighborLimit <= 0 {
		config.NeighborLimit = 5
	}
	if config.EdgeConcurrency <= 0 {
		config.EdgeConcurrency = 4
	}
	if config.WeightVector <= 0 && config.WeightKeyword <= 0 {
		config.WeightVector = 0.7
		config.WeightKeyword = 0.3
	}
	return &HybridRetriever{
		vectorRepo: vectorRepo,
		edgeRepo:   edgeRepo,
		embedder:   embedder,
		config:     config,
	}
}

// Query 执行检索 + 1 跳图谱扩展，返回拼装好的 ContextBlock 列表。
//
// 流程：
//  1. 默认值填充（mode/limit）
//  2. 按 mode 分发到 keyword/vector/hybrid 三条检索路径
//  3. 检索结果转 ContextBlock
//  4. 若 ExpandHops > 0，并发拉取每个 block 的 callers/callees（失败静默跳过）
func (r *HybridRetriever) Query(ctx context.Context, req RetrievalRequest) ([]ContextBlock, error) {
	// 1. 默认值填充
	mode := req.Mode
	if mode == "" {
		mode = "hybrid"
	}
	limit := req.Limit
	if limit <= 0 {
		limit = r.config.DefaultLimit
	}

	// 2. 构建过滤：与 search_handler 等价，kind/language/repo 全部下沉 SQL。
	filters := models.VectorSearchFilters{
		EntityType:  "symbol",
		Limit:       limit,
		Kind:        req.Kind,
		Language:    req.Language,
		RepoIDs:     req.RepoIDs,
		WithDetails: true,
	}

	// 3. mode 分发
	hybridResults, err := r.search(ctx, mode, req.Query, filters)
	if err != nil {
		return nil, err
	}

	// 4. 转 ContextBlock
	blocks := make([]ContextBlock, 0, len(hybridResults))
	for _, h := range hybridResults {
		blocks = append(blocks, ContextBlock{
			Symbol:     toContextSymbol(&h.VectorSearchResult),
			Similarity: h.Similarity,
			MatchMode:  mode,
			ChunkID:    h.VectorID,
		})
	}

	// 5. 1 跳图谱扩展
	if req.ExpandHops > 0 {
		r.expandGraph(ctx, blocks, req)
	}

	return blocks, nil
}

// search 按 mode 分发到三条检索路径，返回统一的 HybridSearchResult。
//
// 三条路径：
//   - keyword：纯关键词召回（无需 embedding），ts_rank 归一化到 [0,1]
//   - vector：纯向量召回
//   - hybrid（默认/其它）：向量 + 关键词 + 加权重排
func (r *HybridRetriever) search(ctx context.Context, mode, query string, filters models.VectorSearchFilters) ([]*models.HybridSearchResult, error) {
	switch mode {
	case "keyword":
		kwResults, err := r.vectorRepo.KeywordSearch(ctx, query, filters)
		if err != nil {
			return nil, err
		}
		// ts_rank 原始量纲不可控（可能远大于 1），除以本批 max 归一化到 [0,1]，
		// 与 vector / hybrid 模式保持同量纲。
		kwMax := 0.0
		for _, kw := range kwResults {
			if kw.Similarity > kwMax {
				kwMax = kw.Similarity
			}
		}
		results := make([]*models.HybridSearchResult, 0, len(kwResults))
		for _, kw := range kwResults {
			score := kw.Similarity
			if kwMax > 0 {
				score /= kwMax
			}
			kw.Similarity = score
			results = append(results, &models.HybridSearchResult{
				VectorSearchResult: *kw, KeywordScore: score,
			})
		}
		return results, nil

	case "vector":
		embedding, err := r.embedder.GenerateEmbedding(ctx, query)
		if err != nil {
			return nil, err
		}
		vecResults, err := r.vectorRepo.SimilaritySearchWithFilters(ctx, embedding, filters)
		if err != nil {
			return nil, err
		}
		results := make([]*models.HybridSearchResult, 0, len(vecResults))
		for _, v := range vecResults {
			results = append(results, &models.HybridSearchResult{
				VectorSearchResult: *v, VectorScore: v.Similarity,
			})
		}
		return results, nil

	default: // hybrid
		embedding, err := r.embedder.GenerateEmbedding(ctx, query)
		if err != nil {
			return nil, err
		}
		return r.vectorRepo.HybridSearch(ctx, query, embedding, filters, r.config.WeightVector, r.config.WeightKeyword)
	}
}

// expandGraph 对每个 block 并发拉取 callers/callees，就地写回 block。
//
// 并发控制：用带 buffer 的 channel 做信号量（容量 = EdgeConcurrency），
// 限制同时在途的图谱查询数。用 sync.WaitGroup 等待全部完成。
//
// 故障策略：图谱查询失败静默跳过（不中断整体）——某条边的查询失败只导致
// 该 block 的对应邻居为空，不影响其它 block。因此不用 errgroup（它会
// 在首个错误时取消整组）。
//
// 方向开关：ExpandCallers/ExpandCallees 直接按布尔值控制。字段注释虽标注
// "默认 true"，但 bool 零值为 false 无法三态区分，故以调用方显式填入为准。
func (r *HybridRetriever) expandGraph(ctx context.Context, blocks []ContextBlock, req RetrievalRequest) {
	// 信号量：缓冲 channel，写入即占槽，读出即释放。
	sem := make(chan struct{}, r.config.EdgeConcurrency)
	var wg sync.WaitGroup

	for i := range blocks {
		// 闭包捕获 i（range 变量复用陷阱）
		i := i
		symbolID := blocks[i].Symbol.SymbolID
		if symbolID == "" {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			// 每个方向单独占一次信号量槽，使 callers/callees 并行度更均匀。
			if req.ExpandCallers {
				sem <- struct{}{}
				callers, err := r.edgeRepo.GetCallersWithDetails(ctx, symbolID)
				<-sem
				if err == nil {
					blocks[i].Callers = topNeighbors(callers, r.config.NeighborLimit)
				}
				// err != nil 静默跳过
			}
			if req.ExpandCallees {
				sem <- struct{}{}
				callees, err := r.edgeRepo.GetCalleesWithDetails(ctx, symbolID)
				<-sem
				if err == nil {
					blocks[i].Callees = topNeighbors(callees, r.config.NeighborLimit)
				}
				// err != nil 静默跳过
			}
		}()
	}
	wg.Wait()
}

// toContextSymbol 把检索结果转成图谱/检索共用的符号视图。
// 字段映射：EntityID → SymbolID，其余 Name/Kind/Signature/FilePath/Language/Docstring 直接拷贝。
func toContextSymbol(v *models.VectorSearchResult) ContextSymbol {
	return ContextSymbol{
		SymbolID:  v.EntityID,
		Name:      v.Name,
		Kind:      v.Kind,
		Signature: v.Signature,
		FilePath:  v.FilePath,
		Language:  v.Language,
		Docstring: v.Docstring,
	}
}

// topNeighbors 从带详情的边结果中选出 Top-limit 个邻居，转为 ContextSymbol。
//
// EdgeWithDetails 不含调用频次列，无法按"热度"排序；这里按符号名字典序稳定排序
// 后截断，保证输出确定且可复现（便于测试与调试）。
func topNeighbors(edges []*models.EdgeWithDetails, limit int) []ContextSymbol {
	if len(edges) == 0 {
		return nil
	}
	sort.SliceStable(edges, func(i, j int) bool {
		return edges[i].Name < edges[j].Name
	})
	if limit > 0 && len(edges) > limit {
		edges = edges[:limit]
	}
	out := make([]ContextSymbol, 0, len(edges))
	for _, e := range edges {
		out = append(out, ContextSymbol{
			SymbolID:  e.SymbolID,
			Name:      e.Name,
			Kind:      e.Kind,
			Signature: e.Signature,
			FilePath:  e.FilePath,
		})
	}
	return out
}
