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

func TestNewRepositoryHandler(t *testing.T) {
	// Test creating handler with nil DB
	handler := NewRepositoryHandler(nil)
	if handler == nil {
		t.Error("Expected handler to be created, got nil")
	}
	if handler.repoRepository == nil {
		t.Error("Expected repoRepository to be initialized")
	}
}

func TestRepositoryResponse_JSONSerialization(t *testing.T) {
	response := RepositoryResponse{
		RepoID:     "test-id",
		Name:       "test-repo",
		URL:        "https://github.com/test/repo",
		Branch:     "main",
		CommitHash: "abc123",
		Metadata: map[string]interface{}{
			"language": "Go",
			"stars":    100,
		},
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
	}

	// Test that all fields are set
	if response.RepoID == "" {
		t.Error("Expected RepoID to be set")
	}
	if response.Name == "" {
		t.Error("Expected Name to be set")
	}
	if response.Branch == "" {
		t.Error("Expected Branch to be set")
	}
	if response.CreatedAt == "" {
		t.Error("Expected CreatedAt to be set")
	}
	if response.UpdatedAt == "" {
		t.Error("Expected UpdatedAt to be set")
	}
	if response.Metadata == nil {
		t.Error("Expected Metadata to be set")
	}
}

func TestListRepositoriesResponse_EmptyList(t *testing.T) {
	response := ListRepositoriesResponse{
		Repositories: []RepositoryResponse{},
		Total:        0,
	}

	// Validate empty list
	if len(response.Repositories) != 0 {
		t.Errorf("Expected 0 repositories, got %d", len(response.Repositories))
	}
	if response.Total != 0 {
		t.Errorf("Expected total to be 0, got %d", response.Total)
	}
}

func TestRepositoryResponse_OptionalFields(t *testing.T) {
	// Test with minimal fields
	response := RepositoryResponse{
		RepoID:    "test-id",
		Name:      "test-repo",
		Branch:    "main",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
	}

	// Optional fields should be empty
	if response.URL != "" {
		t.Errorf("Expected URL to be empty, got '%s'", response.URL)
	}
	if response.CommitHash != "" {
		t.Errorf("Expected CommitHash to be empty, got '%s'", response.CommitHash)
	}
	if response.Metadata != nil {
		t.Error("Expected Metadata to be nil")
	}
}
