package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// RepositoryHandler handles repository operations
type RepositoryHandler struct {
	repoRepository *models.RepositoryRepository
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(db *models.DB) *RepositoryHandler {
	return &RepositoryHandler{
		repoRepository: models.NewRepositoryRepository(db),
	}
}

// RepositoryResponse represents a repository in API responses
type RepositoryResponse struct {
	RepoID     string                 `json:"repo_id"`
	Name       string                 `json:"name"`
	URL        string                 `json:"url,omitempty"`
	Branch     string                 `json:"branch"`
	CommitHash string                 `json:"commit_hash,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// ListRepositoriesResponse represents the response for listing repositories
type ListRepositoriesResponse struct {
	Repositories []RepositoryResponse `json:"repositories"`
	Total        int                  `json:"total"`
}

// GetAll handles GET /api/v1/repositories
func (h *RepositoryHandler) GetAll(c *gin.Context) {
	ctx := context.Background()
	
	repos, err := h.repoRepository.GetAll(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve repositories",
			"details": err.Error(),
		})
		return
	}

	// Convert to response format
	response := ListRepositoriesResponse{
		Repositories: make([]RepositoryResponse, len(repos)),
		Total:        len(repos),
	}

	for i, repo := range repos {
		response.Repositories[i] = RepositoryResponse{
			RepoID:     repo.RepoID,
			Name:       repo.Name,
			URL:        repo.URL,
			Branch:     repo.Branch,
			CommitHash: repo.CommitHash,
			Metadata:   repo.Metadata,
			CreatedAt:  repo.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:  repo.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetByID handles GET /api/v1/repositories/:id
func (h *RepositoryHandler) GetByID(c *gin.Context) {
	repoID := c.Param("id")
	if repoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Repository ID is required",
		})
		return
	}

	ctx := context.Background()
	repo, err := h.repoRepository.GetByID(ctx, repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve repository",
			"details": err.Error(),
		})
		return
	}

	if repo == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Repository not found",
		})
		return
	}

	// Convert to response format
	response := RepositoryResponse{
		RepoID:     repo.RepoID,
		Name:       repo.Name,
		URL:        repo.URL,
		Branch:     repo.Branch,
		CommitHash: repo.CommitHash,
		Metadata:   repo.Metadata,
		CreatedAt:  repo.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  repo.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	c.JSON(http.StatusOK, response)
}
