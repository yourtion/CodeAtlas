package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

func TestNewAPIClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		options []ClientOption
		want    *APIClient
	}{
		{
			name:    "default client",
			baseURL: "http://localhost:8080",
			options: nil,
			want: &APIClient{
				baseURL:    "http://localhost:8080",
				maxRetries: 3,
			},
		},
		{
			name:    "client with token",
			baseURL: "http://localhost:8080",
			options: []ClientOption{WithToken("test-token")},
			want: &APIClient{
				baseURL:    "http://localhost:8080",
				token:      "test-token",
				maxRetries: 3,
			},
		},
		{
			name:    "client with custom timeout",
			baseURL: "http://localhost:8080",
			options: []ClientOption{WithTimeout(10 * time.Second)},
			want: &APIClient{
				baseURL:    "http://localhost:8080",
				maxRetries: 3,
			},
		},
		{
			name:    "client with max retries",
			baseURL: "http://localhost:8080",
			options: []ClientOption{WithMaxRetries(5)},
			want: &APIClient{
				baseURL:    "http://localhost:8080",
				maxRetries: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAPIClient(tt.baseURL, tt.options...)
			if got.baseURL != tt.want.baseURL {
				t.Errorf("baseURL = %v, want %v", got.baseURL, tt.want.baseURL)
			}
			if got.token != tt.want.token {
				t.Errorf("token = %v, want %v", got.token, tt.want.token)
			}
			if got.maxRetries != tt.want.maxRetries {
				t.Errorf("maxRetries = %v, want %v", got.maxRetries, tt.want.maxRetries)
			}
			if got.httpClient == nil {
				t.Error("httpClient should not be nil")
			}
		})
	}
}

func TestAPIClient_Index(t *testing.T) {
	tests := []struct {
		name           string
		request        *IndexRequest
		serverResponse IndexResponse
		serverStatus   int
		wantErr        bool
	}{
		{
			name: "successful index",
			request: &IndexRequest{
				RepoName: "test-repo",
				ParseOutput: schema.ParseOutput{
					Files: []schema.File{
						{FileID: "file1", Path: "test.go", Language: "go"},
					},
				},
			},
			serverResponse: IndexResponse{
				RepoID:         "repo-123",
				Status:         "success",
				FilesProcessed: 1,
				SymbolsCreated: 5,
				EdgesCreated:   3,
				Duration:       "1s",
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name: "server error",
			request: &IndexRequest{
				RepoName: "test-repo",
				ParseOutput: schema.ParseOutput{
					Files: []schema.File{},
				},
			},
			serverStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/index" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("unexpected method: %s", r.Method)
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK {
					json.NewEncoder(w).Encode(tt.serverResponse)
				} else {
					json.NewEncoder(w).Encode(map[string]string{"error": "server error"})
				}
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, WithMaxRetries(0))
			ctx := context.Background()

			got, err := client.Index(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Index() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.RepoID != tt.serverResponse.RepoID {
					t.Errorf("RepoID = %v, want %v", got.RepoID, tt.serverResponse.RepoID)
				}
				if got.Status != tt.serverResponse.Status {
					t.Errorf("Status = %v, want %v", got.Status, tt.serverResponse.Status)
				}
			}
		})
	}
}

func TestAPIClient_Search(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		embedding      []float32
		filters        SearchFilters
		serverResponse SearchResponse
		serverStatus   int
		wantErr        bool
	}{
		{
			name:      "successful search",
			query:     "test function",
			embedding: []float32{0.1, 0.2, 0.3},
			filters: SearchFilters{
				Limit: 10,
			},
			serverResponse: SearchResponse{
				Results: []SearchResult{
					{
						SymbolID:   "sym-1",
						Name:       "testFunc",
						Kind:       "function",
						Similarity: 0.95,
					},
				},
				Total: 1,
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "server error",
			query:        "test",
			embedding:    []float32{0.1},
			serverStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/search" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("unexpected method: %s", r.Method)
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK {
					json.NewEncoder(w).Encode(tt.serverResponse)
				} else {
					json.NewEncoder(w).Encode(map[string]string{"error": "server error"})
				}
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, WithMaxRetries(0))
			ctx := context.Background()

			got, err := client.Search(ctx, tt.query, tt.embedding, tt.filters)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Total != tt.serverResponse.Total {
					t.Errorf("Total = %v, want %v", got.Total, tt.serverResponse.Total)
				}
				if len(got.Results) != len(tt.serverResponse.Results) {
					t.Errorf("Results length = %v, want %v", len(got.Results), len(tt.serverResponse.Results))
				}
			}
		})
	}
}

func TestAPIClient_GetCallers(t *testing.T) {
	tests := []struct {
		name           string
		symbolID       string
		serverResponse RelationshipResponse
		serverStatus   int
		wantErr        bool
	}{
		{
			name:     "successful get callers",
			symbolID: "sym-123",
			serverResponse: RelationshipResponse{
				Symbols: []RelatedSymbol{
					{
						SymbolID: "caller-1",
						Name:     "callerFunc",
						Kind:     "function",
					},
				},
				Total: 1,
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "symbol not found",
			symbolID:     "invalid",
			serverStatus: http.StatusNotFound,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/api/v1/symbols/" + tt.symbolID + "/callers"
				if r.URL.Path != expectedPath {
					t.Errorf("unexpected path: %s, want %s", r.URL.Path, expectedPath)
				}
				if r.Method != "GET" {
					t.Errorf("unexpected method: %s", r.Method)
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK {
					json.NewEncoder(w).Encode(tt.serverResponse)
				} else {
					json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
				}
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, WithMaxRetries(0))
			ctx := context.Background()

			got, err := client.GetCallers(ctx, tt.symbolID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCallers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Total != tt.serverResponse.Total {
					t.Errorf("Total = %v, want %v", got.Total, tt.serverResponse.Total)
				}
			}
		})
	}
}

func TestAPIClient_GetCallees(t *testing.T) {
	tests := []struct {
		name           string
		symbolID       string
		serverResponse RelationshipResponse
		serverStatus   int
		wantErr        bool
	}{
		{
			name:     "successful get callees",
			symbolID: "sym-123",
			serverResponse: RelationshipResponse{
				Symbols: []RelatedSymbol{
					{
						SymbolID: "callee-1",
						Name:     "calleeFunc",
						Kind:     "function",
					},
				},
				Total: 1,
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/api/v1/symbols/" + tt.symbolID + "/callees"
				if r.URL.Path != expectedPath {
					t.Errorf("unexpected path: %s, want %s", r.URL.Path, expectedPath)
				}

				w.WriteHeader(tt.serverStatus)
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, WithMaxRetries(0))
			ctx := context.Background()

			got, err := client.GetCallees(ctx, tt.symbolID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCallees() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.Total != tt.serverResponse.Total {
				t.Errorf("Total = %v, want %v", got.Total, tt.serverResponse.Total)
			}
		})
	}
}

func TestAPIClient_GetDependencies(t *testing.T) {
	tests := []struct {
		name           string
		symbolID       string
		serverResponse DependencyResponse
		serverStatus   int
		wantErr        bool
	}{
		{
			name:     "successful get dependencies",
			symbolID: "sym-123",
			serverResponse: DependencyResponse{
				Dependencies: []Dependency{
					{
						Name:     "fmt",
						Kind:     "module",
						EdgeType: "import",
					},
				},
				Total: 1,
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/api/v1/symbols/" + tt.symbolID + "/dependencies"
				if r.URL.Path != expectedPath {
					t.Errorf("unexpected path: %s, want %s", r.URL.Path, expectedPath)
				}

				w.WriteHeader(tt.serverStatus)
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, WithMaxRetries(0))
			ctx := context.Background()

			got, err := client.GetDependencies(ctx, tt.symbolID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.Total != tt.serverResponse.Total {
				t.Errorf("Total = %v, want %v", got.Total, tt.serverResponse.Total)
			}
		})
	}
}

func TestAPIClient_GetFileSymbols(t *testing.T) {
	tests := []struct {
		name           string
		fileID         string
		serverResponse SymbolsResponse
		serverStatus   int
		wantErr        bool
	}{
		{
			name:   "successful get file symbols",
			fileID: "file-123",
			serverResponse: SymbolsResponse{
				Symbols: []SymbolInfo{
					{
						SymbolID:  "sym-1",
						Name:      "testFunc",
						Kind:      "function",
						StartLine: 10,
						EndLine:   20,
					},
				},
				Total: 1,
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/api/v1/files/" + tt.fileID + "/symbols"
				if r.URL.Path != expectedPath {
					t.Errorf("unexpected path: %s, want %s", r.URL.Path, expectedPath)
				}

				w.WriteHeader(tt.serverStatus)
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, WithMaxRetries(0))
			ctx := context.Background()

			got, err := client.GetFileSymbols(ctx, tt.fileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFileSymbols() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.Total != tt.serverResponse.Total {
				t.Errorf("Total = %v, want %v", got.Total, tt.serverResponse.Total)
			}
		})
	}
}

func TestAPIClient_Health(t *testing.T) {
	tests := []struct {
		name         string
		serverStatus int
		wantErr      bool
	}{
		{
			name:         "healthy server",
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "unhealthy server",
			serverStatus: http.StatusServiceUnavailable,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/health" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			client := NewAPIClient(server.URL)
			ctx := context.Background()

			err := client.Health(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Health() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPIClient_RetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "temporary error"})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(RelationshipResponse{
				Symbols: []RelatedSymbol{},
				Total:   0,
			})
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, WithMaxRetries(3))
	ctx := context.Background()

	_, err := client.GetCallers(ctx, "test-id")
	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestAPIClient_Authentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RelationshipResponse{})
	}))
	defer server.Close()

	t.Run("with token", func(t *testing.T) {
		client := NewAPIClient(server.URL, WithToken("test-token"), WithMaxRetries(0))
		ctx := context.Background()

		_, err := client.GetCallers(ctx, "test-id")
		if err != nil {
			t.Errorf("Expected success with token, got error: %v", err)
		}
	})

	t.Run("without token", func(t *testing.T) {
		client := NewAPIClient(server.URL, WithMaxRetries(0))
		ctx := context.Background()

		_, err := client.GetCallers(ctx, "test-id")
		if err == nil {
			t.Error("Expected error without token")
		}
	})
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *APIError
		wantMsg string
	}{
		{
			name: "error without details",
			err: &APIError{
				StatusCode: 404,
				Message:    "not found",
			},
			wantMsg: "API error (status 404): not found",
		},
		{
			name: "error with details",
			err: &APIError{
				StatusCode: 400,
				Message:    "validation failed",
				Details:    "missing required field",
			},
			wantMsg: "API error (status 400): validation failed - missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}
