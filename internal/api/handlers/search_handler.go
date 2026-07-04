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

	// 构建检索过滤：kind/language/repo 全部下沉到 SQL（JOIN symbols/files），
	// 过滤在 LIMIT 前应用，保证返回数满 limit（修复原先"先取 limit 再内存
	// 过滤、过滤掉的不补位、导致结果数失真"的缺陷）。
	// JOIN 同时返回符号/文件详情，消除原先每条结果的 2 次 N+1 查询。
	filters := models.VectorSearchFilters{
		EntityType:  "symbol",
		Limit:       req.Limit,
		Kind:        req.Kind,
		Language:    req.Language,
		RepoID:      req.RepoID,
		WithDetails: true, // 顺带取出 name/kind/signature/docstring/file_path/language/repo
	}

	// Perform vector similarity search (过滤已在 SQL 层应用)
	vectorResults, err := h.vectorRepo.SimilaritySearchWithFilters(ctx, embedding, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to perform semantic search",
			"details": err.Error(),
		})
		return
	}

	// 向量检索已 JOIN 返回全部所需详情，直接构造响应。
	results := make([]SearchResult, 0, len(vectorResults))
	for _, vr := range vectorResults {
		results = append(results, SearchResult{
			SymbolID:   vr.EntityID,
			Name:       vr.Name,
			Kind:       vr.Kind,
			Signature:  vr.Signature,
			FilePath:   vr.FilePath,
			Docstring:  vr.Docstring,
			Similarity: vr.Similarity,
		})
	}

	response := SearchResponse{
		Results: results,
		Total:   len(results),
	}

	c.JSON(http.StatusOK, response)
}
