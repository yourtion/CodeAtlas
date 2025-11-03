package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// SearchHandler handles search operations
type SearchHandler struct {
	vectorRepo *models.VectorRepository
	symbolRepo *models.SymbolRepository
	fileRepo   *models.FileRepository
	embedder   indexer.Embedder
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(db *models.DB) *SearchHandler {
	return &SearchHandler{
		vectorRepo: models.NewVectorRepository(db),
		symbolRepo: models.NewSymbolRepository(db),
		fileRepo:   models.NewFileRepository(db),
		embedder:   indexer.NewOpenAIEmbedder(indexer.DefaultEmbedderConfig(), models.NewVectorRepository(db)),
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

	// Fetch symbol details and apply additional filters
	results := make([]SearchResult, 0)
	for _, vr := range vectorResults {
		// Get symbol details
		symbol, err := h.symbolRepo.GetByID(ctx, vr.EntityID)
		if err != nil {
			continue // Skip on error
		}
		if symbol == nil {
			continue
		}

		// Apply kind filter
		if len(req.Kind) > 0 {
			kindMatch := false
			for _, k := range req.Kind {
				if symbol.Kind == k {
					kindMatch = true
					break
				}
			}
			if !kindMatch {
				continue
			}
		}

		// Get file details for path and language filter
		file, err := h.fileRepo.GetByID(ctx, symbol.FileID)
		if err != nil {
			continue
		}
		if file == nil {
			continue
		}

		// Apply repo filter
		if req.RepoID != "" && file.RepoID != req.RepoID {
			continue
		}

		// Apply language filter
		if req.Language != "" && file.Language != req.Language {
			continue
		}

		// Build result
		result := SearchResult{
			SymbolID:   symbol.SymbolID,
			Name:       symbol.Name,
			Kind:       symbol.Kind,
			Signature:  symbol.Signature,
			FilePath:   file.Path,
			Docstring:  symbol.Docstring,
			Similarity: vr.Similarity,
		}
		results = append(results, result)
	}

	response := SearchResponse{
		Results: results,
		Total:   len(results),
	}

	c.JSON(http.StatusOK, response)
}
