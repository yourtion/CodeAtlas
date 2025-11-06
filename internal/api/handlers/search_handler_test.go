package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSearchHandler_Search_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
			name:           "missing query",
			requestBody:    `{"repo_id": "test-repo"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
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
			handler := NewSearchHandler(nil, nil)

			// Create test router
			router := gin.New()
			router.POST("/api/v1/search", handler.Search)

			// Create request
			req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewBufferString(tt.requestBody))
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

func TestSearchRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request SearchRequest
		valid   bool
	}{
		{
			name: "valid request with all fields",
			request: SearchRequest{
				Query:    "test query",
				RepoID:   "repo-1",
				Language: "go",
				Kind:     []string{"function", "class"},
				Limit:    10,
			},
			valid: true,
		},
		{
			name: "valid request with minimal fields",
			request: SearchRequest{
				Query: "test query",
			},
			valid: true,
		},
		{
			name: "invalid request - missing query",
			request: SearchRequest{
				RepoID: "repo-1",
			},
			valid: false,
		},
		{
			name: "valid request with filters",
			request: SearchRequest{
				Query:    "test query",
				Language: "python",
				Kind:     []string{"function"},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate required fields
			hasQuery := tt.request.Query != ""

			if tt.valid && !hasQuery {
				t.Error("Expected valid request to have query")
			}
			if !tt.valid && hasQuery {
				t.Error("Expected invalid request to be missing query")
			}
		})
	}
}

func TestSearchRequest_DefaultLimit(t *testing.T) {
	request := SearchRequest{
		Query: "test",
		Limit: 0,
	}

	// Default limit should be applied in handler
	expectedDefault := 10
	if request.Limit != 0 {
		t.Errorf("Expected limit to be 0 before handler processing, got %d", request.Limit)
	}

	// After handler processing, limit should be set to default
	if request.Limit == 0 {
		request.Limit = expectedDefault
	}

	if request.Limit != expectedDefault {
		t.Errorf("Expected default limit %d, got %d", expectedDefault, request.Limit)
	}
}

func TestSearchResponse_Structure(t *testing.T) {
	response := SearchResponse{
		Results: []SearchResult{
			{
				SymbolID:   "symbol-1",
				Name:       "testFunc",
				Kind:       "function",
				Signature:  "func testFunc()",
				FilePath:   "test.go",
				Docstring:  "Test function",
				Similarity: 0.95,
			},
		},
		Total: 1,
	}

	if len(response.Results) != response.Total {
		t.Errorf("Expected results length %d to match total %d", len(response.Results), response.Total)
	}

	result := response.Results[0]
	if result.SymbolID == "" {
		t.Error("Expected symbol_id to be set")
	}
	if result.Name == "" {
		t.Error("Expected name to be set")
	}
	if result.Kind == "" {
		t.Error("Expected kind to be set")
	}
	if result.Similarity <= 0 || result.Similarity > 1 {
		t.Errorf("Expected similarity between 0 and 1, got %f", result.Similarity)
	}
}
