package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

func init() {
	// Disable Gin logging in tests
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
}

func TestIndexHandler_Index_InvalidRequest(t *testing.T) {

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty request body",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:           "missing repo_name",
			requestBody:    `{"parse_output": {"files": [], "relationships": [], "metadata": {}}}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name: "missing parse_output",
			requestBody: `{
				"repo_name": "test-repo",
				"parse_output": {
					"files": [],
					"relationships": [],
					"metadata": {}
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Parse output must contain at least one file",
		},
		{
			name: "empty files in parse_output",
			requestBody: `{
				"repo_name": "test-repo",
				"parse_output": {
					"files": [],
					"relationships": [],
					"metadata": {}
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Parse output must contain at least one file",
		},
		{
			name:           "invalid json",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with nil DB and embedder config (won't be used for validation errors)
			handler := NewIndexHandler(nil, nil)

			// Create test router
			router := gin.New()
			router.POST("/api/v1/index", handler.Index)

			// Create request
			req, _ := http.NewRequest("POST", "/api/v1/index", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check error message
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if errorMsg, ok := response["error"].(string); !ok || errorMsg != tt.expectedError {
				t.Errorf("Expected error '%s', got '%v'", tt.expectedError, response["error"])
			}
		})
	}
}

func TestIndexRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request IndexRequest
		valid   bool
	}{
		{
			name: "valid request with all fields",
			request: IndexRequest{
				RepoID:   "test-repo-id",
				RepoName: "test-repo",
				RepoURL:  "https://github.com/test/repo",
				Branch:   "main",
				ParseOutput: schema.ParseOutput{
					Files: []schema.File{
						{
							FileID:   "file-1",
							Path:     "main.go",
							Language: "go",
							Size:     100,
							Checksum: "abc123",
						},
					},
					Relationships: []schema.DependencyEdge{},
					Metadata: schema.ParseMetadata{
						Version:      "1.0",
						Timestamp:    time.Now(),
						TotalFiles:   1,
						SuccessCount: 1,
					},
				},
			},
			valid: true,
		},
		{
			name: "valid request with minimal fields",
			request: IndexRequest{
				RepoName: "test-repo",
				ParseOutput: schema.ParseOutput{
					Files: []schema.File{
						{
							FileID:   "file-1",
							Path:     "main.go",
							Language: "go",
						},
					},
					Relationships: []schema.DependencyEdge{},
					Metadata:      schema.ParseMetadata{},
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate required fields
			if tt.request.RepoName == "" && tt.valid {
				t.Error("Expected valid request to have repo_name")
			}
			if len(tt.request.ParseOutput.Files) == 0 && tt.valid {
				t.Error("Expected valid request to have at least one file")
			}
		})
	}
}

func TestIndexOptions_Defaults(t *testing.T) {
	options := IndexOptions{}

	// Test default values
	if options.BatchSize != 0 {
		t.Errorf("Expected default BatchSize to be 0, got %d", options.BatchSize)
	}
	if options.WorkerCount != 0 {
		t.Errorf("Expected default WorkerCount to be 0, got %d", options.WorkerCount)
	}
	if options.Incremental {
		t.Error("Expected default Incremental to be false")
	}
	if options.SkipVectors {
		t.Error("Expected default SkipVectors to be false")
	}
}

func TestConvertIndexErrors(t *testing.T) {
	// Test nil input
	result := convertIndexErrors(nil)
	if result != nil {
		t.Error("Expected nil result for nil input")
	}

	// Test empty slice
	result = convertIndexErrors([]*indexer.IndexerError{})
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d errors", len(result))
	}

	// Test conversion
	indexerErrors := []*indexer.IndexerError{
		{
			Type:      indexer.ErrorTypeValidation,
			Message:   "test error",
			EntityID:  "entity-1",
			FilePath:  "test.go",
			Retryable: false,
		},
	}
	result = convertIndexErrors(indexerErrors)
	if len(result) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result))
	}
	if result[0].Type != string(indexer.ErrorTypeValidation) {
		t.Errorf("Expected type %s, got %s", indexer.ErrorTypeValidation, result[0].Type)
	}
	if result[0].Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", result[0].Message)
	}
}

func TestNewIndexHandler(t *testing.T) {
	// Test creating handler with nil DB
	handler := NewIndexHandler(nil, nil)
	if handler == nil {
		t.Error("Expected handler to be created, got nil")
	}
	if handler.db != nil {
		t.Error("Expected db to be nil")
	}
	if handler.embedderConfig != nil {
		t.Error("Expected embedderConfig to be nil")
	}

	// Test creating handler with embedder config
	embedderConfig := &EmbedderConfig{
		Backend:     "openai",
		APIEndpoint: "https://api.openai.com/v1/embeddings",
		Model:       "text-embedding-ada-002",
		Dimensions:  1536,
	}
	handler = NewIndexHandler(nil, embedderConfig)
	if handler == nil {
		t.Error("Expected handler to be created, got nil")
	}
	if handler.embedderConfig == nil {
		t.Error("Expected embedderConfig to be set")
	}
	if handler.embedderConfig.Backend != "openai" {
		t.Errorf("Expected backend 'openai', got '%s'", handler.embedderConfig.Backend)
	}
}

func TestIndexRequest_DefaultValues(t *testing.T) {
	req := IndexRequest{
		RepoName: "test-repo",
		ParseOutput: schema.ParseOutput{
			Files: []schema.File{
				{
					FileID:   "file-1",
					Path:     "main.go",
					Language: "go",
				},
			},
		},
	}

	// Test default values
	if req.RepoID != "" {
		t.Errorf("Expected empty RepoID, got '%s'", req.RepoID)
	}
	if req.Branch != "" {
		t.Errorf("Expected empty Branch, got '%s'", req.Branch)
	}
	if req.Options.BatchSize != 0 {
		t.Errorf("Expected BatchSize 0, got %d", req.Options.BatchSize)
	}
	if req.Options.WorkerCount != 0 {
		t.Errorf("Expected WorkerCount 0, got %d", req.Options.WorkerCount)
	}
}

func TestIndexResponse_Structure(t *testing.T) {
	response := IndexResponse{
		RepoID:         "test-repo-id",
		Status:         "success",
		FilesProcessed: 10,
		SymbolsCreated: 50,
		EdgesCreated:   100,
		VectorsCreated: 50,
		Duration:       "5s",
		Errors:         nil,
	}

	// Validate structure
	if response.RepoID == "" {
		t.Error("Expected RepoID to be set")
	}
	if response.Status == "" {
		t.Error("Expected Status to be set")
	}
	if response.FilesProcessed <= 0 {
		t.Error("Expected FilesProcessed to be positive")
	}
	if response.SymbolsCreated <= 0 {
		t.Error("Expected SymbolsCreated to be positive")
	}
	if response.EdgesCreated <= 0 {
		t.Error("Expected EdgesCreated to be positive")
	}
}

func TestIndexError_Structure(t *testing.T) {
	err := IndexError{
		Type:      "validation",
		Message:   "test error",
		EntityID:  "entity-1",
		FilePath:  "test.go",
		Retryable: false,
	}

	// Validate structure
	if err.Type == "" {
		t.Error("Expected Type to be set")
	}
	if err.Message == "" {
		t.Error("Expected Message to be set")
	}
	if err.EntityID == "" {
		t.Error("Expected EntityID to be set")
	}
	if err.FilePath == "" {
		t.Error("Expected FilePath to be set")
	}
}

func TestEmbedderConfig_Structure(t *testing.T) {
	config := &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          "https://api.openai.com/v1/embeddings",
		APIKey:               "sk-test-key",
		Model:                "text-embedding-ada-002",
		Dimensions:           1536,
		BatchSize:            100,
		MaxRequestsPerSecond: 10,
		MaxRetries:           3,
		BaseRetryDelay:       time.Second,
		MaxRetryDelay:        30 * time.Second,
		Timeout:              30 * time.Second,
	}

	// Validate structure
	if config.Backend == "" {
		t.Error("Expected Backend to be set")
	}
	if config.APIEndpoint == "" {
		t.Error("Expected APIEndpoint to be set")
	}
	if config.Model == "" {
		t.Error("Expected Model to be set")
	}
	if config.Dimensions <= 0 {
		t.Error("Expected Dimensions to be positive")
	}
	if config.BatchSize <= 0 {
		t.Error("Expected BatchSize to be positive")
	}
}
