package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// EmbedderConfig is an alias for indexer.EmbedderConfig for easier imports
type EmbedderConfig = indexer.EmbedderConfig

// SearchHandler handles search operations
type SearchHandler struct {
	vectorRepo *models.VectorRepository
	symbolRepo *models.SymbolRepository
	fileRepo   *models.FileRepository
	embedder   indexer.Embedder
}

// NewSearchHandler creates a new search handler with embedder configuration
func NewSearchHandler(db *models.DB, embedderConfig *EmbedderConfig) *SearchHandler {
	// Use default config if none provided
	if embedderConfig == nil {
		embedderConfig = indexer.DefaultEmbedderConfig()
	}
	
	return &SearchHandler{
		vectorRepo: models.NewVectorRepository(db),
		symbolRepo: models.NewSymbolRepository(db),
		fileRepo:   models.NewFileRepository(db),
		embedder:   indexer.NewOpenAIEmbedder(embedderConfig, models.NewVectorRepository(db)),
	}
}

// NewSearchHandlerWithEmbedder creates a new search handler with custom embedder
func NewSearchHandlerWithEmbedder(db *models.DB, embedder indexer.Embedder) *SearchHandler {
	return &SearchHandler{
		vectorRepo: models.NewVectorRepository(db),
		symbolRepo: models.NewSymbolRepository(db),
		fileRepo:   models.NewFileRepository(db),
		embedder:   embedder,
	}
}

// SearchRequest represents the request body for POST /api/v1/search
type SearchRequest struct {
	Query    string   `json:"query" binding:"required"`
	RepoIDs  []string `json:"repo_ids,omitempty"`
	Language string   `json:"language,omitempty"`
	Kind     []string `json:"kind,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	// Mode 控制检索策略：vector（纯向量）、keyword（纯关键词）、hybrid（混合重排）。
	// 默认 hybrid。hybrid 适合自然语言提问，keyword 适合精确符号名查找。
	Mode string `json:"mode,omitempty"`
}

// SearchResponse represents the response for POST /api/v1/search
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// SearchResult represents a single search result
type SearchResult struct {
	SymbolID   string  `json:"symbol_id"`
	Name       string  `json:"name"`
	Kind       string  `json:"kind"`
	Signature  string  `json:"signature"`
	FilePath   string  `json:"file_path"`
	Docstring  string  `json:"docstring,omitempty"`
	Similarity float64 `json:"similarity"`
}

// Search handles POST /api/v1/search
func (h *SearchHandler) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set default limit
	if req.Limit == 0 {
		req.Limit = 10
	}

	ctx := context.Background()

	// mode 默认 hybrid：向量召回（语义）+ 关键词召回（精确符号名）+ 重排。
	// keyword 模式跳过 embedding 生成，适合精确符号查找且省一次 API 调用。
	mode := req.Mode
	if mode == "" {
		mode = "hybrid"
	}

	// 构建检索过滤：kind/language/repo 全部下沉到 SQL（JOIN symbols/files），
	// 过滤在 LIMIT 前应用，保证返回数满 limit。
	filters := models.VectorSearchFilters{
		EntityType:  "symbol",
		Limit:       req.Limit,
		Kind:        req.Kind,
		Language:    req.Language,
		RepoIDs:     req.RepoIDs,
		WithDetails: true, // JOIN 顺带返回 name/kind/signature/docstring/file_path/language/repo
	}

	// 按 mode 分发到不同检索路径
	var hybridResults []*models.HybridSearchResult
	switch mode {
	case "keyword":
		// 纯关键词召回（无需 embedding）
		kwResults, err := h.vectorRepo.KeywordSearch(ctx, req.Query, filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to perform keyword search",
				"details": err.Error(),
			})
			return
		}
		// ts_rank 原始量纲不可控（可能远大于 1），除以本批 max 归一化到 [0,1]，
		// 与 vector / hybrid 模式保持相似度同量纲（响应字段语义一致）。
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
			kw.Similarity = score // 写回，使响应直接读到归一化后的值
			hybridResults = append(hybridResults, &models.HybridSearchResult{
				VectorSearchResult: *kw, KeywordScore: score,
			})
		}
	case "vector":
		// 纯向量召回
		embedding, err := h.embedder.GenerateEmbedding(ctx, req.Query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to generate embedding",
				"details": err.Error(),
			})
			return
		}
		vecResults, err := h.vectorRepo.SimilaritySearchWithFilters(ctx, embedding, filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to perform semantic search",
				"details": err.Error(),
			})
			return
		}
		hybridResults = make([]*models.HybridSearchResult, 0, len(vecResults))
		for _, v := range vecResults {
			hybridResults = append(hybridResults, &models.HybridSearchResult{
				VectorSearchResult: *v, VectorScore: v.Similarity,
			})
		}
	default:
		// hybrid：向量 + 关键词 + 重排（默认）
		embedding, err := h.embedder.GenerateEmbedding(ctx, req.Query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to generate embedding",
				"details": err.Error(),
			})
			return
		}
		// 权重：向量为主 0.7，关键词为辅 0.3
		hybridResults, err = h.vectorRepo.HybridSearch(ctx, req.Query, embedding, filters, 0.7, 0.3)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to perform hybrid search",
				"details": err.Error(),
			})
			return
		}
	}

	// 构造响应（检索已 JOIN 返回全部所需详情）
	results := make([]SearchResult, 0, len(hybridResults))
	for _, h := range hybridResults {
		results = append(results, SearchResult{
			SymbolID:   h.EntityID,
			Name:       h.Name,
			Kind:       h.Kind,
			Signature:  h.Signature,
			FilePath:   h.FilePath,
			Docstring:  h.Docstring,
			Similarity: h.Similarity,
		})
	}

	response := SearchResponse{
		Results: results,
		Total:   len(results),
	}

	c.JSON(http.StatusOK, response)
}
