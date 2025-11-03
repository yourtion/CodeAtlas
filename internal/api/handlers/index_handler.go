package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// IndexHandler handles indexing operations
type IndexHandler struct {
	db *models.DB
}

// NewIndexHandler creates a new index handler
func NewIndexHandler(db *models.DB) *IndexHandler {
	return &IndexHandler{db: db}
}

// IndexRequest represents the request body for POST /api/v1/index
type IndexRequest struct {
	RepoID      string              `json:"repo_id,omitempty"`
	RepoName    string              `json:"repo_name" binding:"required"`
	RepoURL     string              `json:"repo_url,omitempty"`
	Branch      string              `json:"branch,omitempty"`
	CommitHash  string              `json:"commit_hash,omitempty"`
	ParseOutput schema.ParseOutput  `json:"parse_output" binding:"required"`
	Options     IndexOptions        `json:"options,omitempty"`
}

// IndexOptions contains optional configuration for indexing
type IndexOptions struct {
	Incremental    bool   `json:"incremental"`
	SkipVectors    bool   `json:"skip_vectors"`
	BatchSize      int    `json:"batch_size"`
	WorkerCount    int    `json:"worker_count"`
	EmbeddingModel string `json:"embedding_model,omitempty"`
}

// IndexResponse represents the response for POST /api/v1/index
type IndexResponse struct {
	RepoID         string        `json:"repo_id"`
	Status         string        `json:"status"`
	FilesProcessed int           `json:"files_processed"`
	SymbolsCreated int           `json:"symbols_created"`
	EdgesCreated   int           `json:"edges_created"`
	VectorsCreated int           `json:"vectors_created"`
	Errors         []IndexError  `json:"errors,omitempty"`
	Duration       string        `json:"duration"`
}

// IndexError represents an error that occurred during indexing
type IndexError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	EntityID  string `json:"entity_id,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	Retryable bool   `json:"retryable"`
}

// Index handles POST /api/v1/index
func (h *IndexHandler) Index(c *gin.Context) {
	var req IndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate parse output
	if len(req.ParseOutput.Files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Parse output must contain at least one file",
		})
		return
	}

	// Generate repo ID if not provided
	if req.RepoID == "" {
		req.RepoID = uuid.New().String()
	}

	// Set default branch if not provided
	if req.Branch == "" {
		req.Branch = "main"
	}

	// Create indexer config
	config := &indexer.IndexerConfig{
		RepoID:          req.RepoID,
		RepoName:        req.RepoName,
		RepoURL:         req.RepoURL,
		Branch:          req.Branch,
		BatchSize:       req.Options.BatchSize,
		WorkerCount:     req.Options.WorkerCount,
		SkipVectors:     req.Options.SkipVectors,
		Incremental:     req.Options.Incremental,
		UseTransactions: true,
		GraphName:       "code_graph",
		EmbeddingModel:  req.Options.EmbeddingModel,
	}

	// Set defaults for config
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = 4
	}

	// Create indexer
	idx := indexer.NewIndexer(h.db, config)

	// Run indexing
	ctx := context.Background()
	result, err := idx.Index(ctx, &req.ParseOutput)
	
	if err != nil {
		// Log the error for debugging
		fmt.Printf("DEBUG Index handler: indexing error: %v\n", err)
		if result != nil {
			fmt.Printf("DEBUG Index handler: result status: %s, errors: %d\n", result.Status, len(result.Errors))
			for i, e := range result.Errors {
				fmt.Printf("DEBUG Index handler: error %d: type=%s, message=%s\n", i, e.Type, e.Message)
			}
		}
		
		// Check if it's a validation error
		if result != nil && result.Status == "failed" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Indexing failed due to validation errors",
				"details": convertIndexErrors(result.Errors),
			})
			return
		}

		// Other errors
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Indexing failed",
			"details": err.Error(),
		})
		return
	}

	// Build response
	response := IndexResponse{
		RepoID:         result.RepoID,
		Status:         result.Status,
		FilesProcessed: result.FilesProcessed,
		SymbolsCreated: result.SymbolsCreated,
		EdgesCreated:   result.EdgesCreated,
		VectorsCreated: result.VectorsCreated,
		Duration:       result.Duration.String(),
		Errors:         convertIndexErrors(result.Errors),
	}

	// Determine HTTP status code based on result status
	statusCode := http.StatusOK
	switch result.Status {
	case "success":
		statusCode = http.StatusOK
	case "partial_success":
		statusCode = http.StatusMultiStatus
	case "success_with_warnings":
		statusCode = http.StatusOK
	case "failed":
		statusCode = http.StatusInternalServerError
	}

	c.JSON(statusCode, response)
}

// convertIndexErrors converts indexer errors to API error format
func convertIndexErrors(errors []*indexer.IndexerError) []IndexError {
	if errors == nil {
		return nil
	}

	result := make([]IndexError, len(errors))
	for i, err := range errors {
		result[i] = IndexError{
			Type:      string(err.Type),
			Message:   err.Message,
			EntityID:  err.EntityID,
			FilePath:  err.FilePath,
			Retryable: err.Retryable,
		}
	}
	return result
}
