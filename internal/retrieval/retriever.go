// Package retrieval 提供代码检索与图谱上下文组装能力。
//
// 本层把"检索 + 1 跳图谱扩展"封装为可复用的纯接口，供 QA 引擎和
// 未来的 Agentic RAG 共用。不碰 HTTP、不碰 prompt 格式化。
package retrieval

import (
	"context"

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
	Similarity float64         // 检索得分
	MatchMode  string          // "vector" | "keyword" | "hybrid"
	Callers    []ContextSymbol // 1 跳：谁调用了它（每边 Top-5）
	Callees    []ContextSymbol // 1 跳：它调用了谁（每边 Top-5）
	ChunkID    string          // 对应 vectors.vector_id
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

// HybridRetrieverConfig 是 HybridRetriever 的配置。
type HybridRetrieverConfig struct {
	WeightVector    float64 // 默认 0.7
	WeightKeyword   float64 // 默认 0.3
	DefaultLimit    int     // 默认 10
	NeighborLimit   int     // 每边邻居上限，默认 5
	EdgeConcurrency int     // 图谱查询并发上限，默认 4
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

// 编译期断言：确保 *models.VectorRepository 与 *models.EdgeRepository 满足本层接口。
// 一旦仓库方法签名变更导致不再满足，编译会立即失败，提醒对齐接口。
var (
	_ VectorSearcher = (*models.VectorRepository)(nil)
	_ EdgeExpander   = (*models.EdgeRepository)(nil)
)
