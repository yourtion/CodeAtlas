package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRepositoryHandler_GetByID_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create handler with nil DB (won't be used for validation)
	handler := NewRepositoryHandler(nil)

	// Create test router
	router := gin.New()
	router.GET("/api/v1/repositories/:id", handler.GetByID)

	tests := []struct {
		name           string
		repoID         string
		expectedStatus int
	}{
		{
			name:           "empty repository ID",
			repoID:         "",
			expectedStatus: http.StatusNotFound, // Gin returns 404 for missing param
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req, _ := http.NewRequest("GET", "/api/v1/repositories/"+tt.repoID, nil)
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRepositoryResponse_Structure(t *testing.T) {
	response := RepositoryResponse{
		RepoID:     "test-id",
		Name:       "test-repo",
		URL:        "https://github.com/test/repo",
		Branch:     "main",
		CommitHash: "abc123",
		Metadata:   map[string]interface{}{"key": "value"},
		CreatedAt:  "2024-01-01T00:00:00Z",
		UpdatedAt:  "2024-01-01T00:00:00Z",
	}

	// Validate structure
	if response.RepoID == "" {
		t.Error("Expected RepoID to be set")
	}
	if response.Name == "" {
		t.Error("Expected Name to be set")
	}
	if response.Branch == "" {
		t.Error("Expected Branch to be set")
	}
}

func TestListRepositoriesResponse_Structure(t *testing.T) {
	response := ListRepositoriesResponse{
		Repositories: []RepositoryResponse{
			{
				RepoID: "test-id-1",
				Name:   "test-repo-1",
				Branch: "main",
			},
			{
				RepoID: "test-id-2",
				Name:   "test-repo-2",
				Branch: "develop",
			},
		},
		Total: 2,
	}

	// Validate structure
	if len(response.Repositories) != response.Total {
		t.Errorf("Expected %d repositories, got %d", response.Total, len(response.Repositories))
	}
	if response.Total != 2 {
		t.Errorf("Expected total to be 2, got %d", response.Total)
	}
}
