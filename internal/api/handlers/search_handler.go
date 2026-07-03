package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/search"
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
	RepoID   string   `json:"repo_id,omitempty"`
	Language string   `json:"language,omitempty"`
	Kind     []string `json:"kind,omitempty"`
	Limit    int      `json:"limit,omitempty"`
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

	// Generate embedding from query text using embedding service
	embedding, err := h.embedder.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate embedding",
			"details": err.Error(),
		})
		return
	}

	// Build search filters
	filters := models.VectorSearchFilters{
		EntityType: "symbol",
		Limit:      req.Limit,
	}

	// Perform vector similarity search
	vectorResults, err := h.vectorRepo.SimilaritySearchWithFilters(ctx, embedding, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to perform semantic search",
			"details": err.Error(),
		})
		return
	}

	// 收集候选结果：向量检索结果 + 符号/文件详情。
	// NOTE: 过滤逻辑已抽出到 internal/search.InMemoryFilter，调用点集中于此。
	// 下一站检索质量优化时，把过滤下沉到 SQL（在 SimilaritySearchWithFilters
	// 阶段就应用 kind/language/repo 条件），此处改用 search.SQLFilter 即可，
	// 无需改动 handler。
	candidates := make([]search.Candidate, 0, len(vectorResults))
	for _, vr := range vectorResults {
		symbol, err := h.symbolRepo.GetByID(ctx, vr.EntityID)
		if err != nil || symbol == nil {
			continue
		}
		file, err := h.fileRepo.GetByID(ctx, symbol.FileID)
		if err != nil || file == nil {
			continue
		}
		candidates = append(candidates, search.Candidate{
			SymbolID:   symbol.SymbolID,
			Name:       symbol.Name,
			Kind:       symbol.Kind,
			Signature:  symbol.Signature,
			FilePath:   file.Path,
			Language:   file.Language,
			RepoID:     file.RepoID,
			Docstring:  symbol.Docstring,
			Similarity: vr.Similarity,
		})
	}

	// 应用过滤（当前为内存实现，行为与原内联逻辑一致）
	filter := search.NewInMemoryFilter(search.FilterCriteria{
		Kind:     req.Kind,
		Language: req.Language,
		RepoID:   req.RepoID,
	})
	filtered := filter.Filter(candidates)

	// 转换为响应类型
	results := make([]SearchResult, 0, len(filtered))
	for _, c := range filtered {
		results = append(results, SearchResult{
			SymbolID:   c.SymbolID,
			Name:       c.Name,
			Kind:       c.Kind,
			Signature:  c.Signature,
			FilePath:   c.FilePath,
			Docstring:  c.Docstring,
			Similarity: c.Similarity,
		})
	}

	response := SearchResponse{
		Results: results,
		Total:   len(results),
	}

	c.JSON(http.StatusOK, response)
}
