package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// APIClient provides HTTP client for CLI to communicate with API server
type APIClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
	maxRetries int
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string, options ...ClientOption) *APIClient {
	client := &APIClient{
		baseURL:    baseURL,
		maxRetries: 3,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	// Apply options
	for _, opt := range options {
		opt(client)
	}

	return client
}

// ClientOption is a function that configures the API client
type ClientOption func(*APIClient)

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *APIClient) {
		c.httpClient.Timeout = timeout
	}
}

// WithToken sets the authentication token
func WithToken(token string) ClientOption {
	return func(c *APIClient) {
		c.token = token
	}
}

// WithMaxRetries sets the maximum number of retry attempts
func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *APIClient) {
		c.maxRetries = maxRetries
	}
}

// IndexRequest represents the request body for POST /api/v1/index
type IndexRequest struct {
	RepoID      string             `json:"repo_id,omitempty"`
	RepoName    string             `json:"repo_name"`
	RepoURL     string             `json:"repo_url,omitempty"`
	Branch      string             `json:"branch,omitempty"`
	CommitHash  string             `json:"commit_hash,omitempty"`
	ParseOutput schema.ParseOutput `json:"parse_output"`
	Options     IndexOptions       `json:"options,omitempty"`
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
	RepoID         string       `json:"repo_id"`
	Status         string       `json:"status"`
	FilesProcessed int          `json:"files_processed"`
	SymbolsCreated int          `json:"symbols_created"`
	EdgesCreated   int          `json:"edges_created"`
	VectorsCreated int          `json:"vectors_created"`
	Errors         []IndexError `json:"errors,omitempty"`
	Duration       string       `json:"duration"`
}

// IndexError represents an error that occurred during indexing
type IndexError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	EntityID  string `json:"entity_id,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	Retryable bool   `json:"retryable"`
}

// SearchFilters contains filters for search queries
type SearchFilters struct {
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

// RelationshipResponse represents the response for relationship queries
type RelationshipResponse struct {
	Symbols []RelatedSymbol `json:"symbols"`
	Total   int             `json:"total"`
}

// RelatedSymbol represents a symbol in a relationship query result
type RelatedSymbol struct {
	SymbolID  string `json:"symbol_id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FilePath  string `json:"file_path"`
	Signature string `json:"signature"`
}

// DependencyResponse represents the response for dependency queries
type DependencyResponse struct {
	Dependencies []Dependency `json:"dependencies"`
	Total        int          `json:"total"`
}

// Dependency represents a dependency relationship
type Dependency struct {
	SymbolID  string `json:"symbol_id,omitempty"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FilePath  string `json:"file_path,omitempty"`
	Module    string `json:"module,omitempty"`
	EdgeType  string `json:"edge_type"`
	Signature string `json:"signature,omitempty"`
}

// SymbolsResponse represents the response for file symbols query
type SymbolsResponse struct {
	Symbols []SymbolInfo `json:"symbols"`
	Total   int          `json:"total"`
}

// SymbolInfo represents symbol information
type SymbolInfo struct {
	SymbolID        string `json:"symbol_id"`
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Signature       string `json:"signature"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	Docstring       string `json:"docstring,omitempty"`
	SemanticSummary string `json:"semantic_summary,omitempty"`
}

// Index sends parse output to API server for indexing
func (c *APIClient) Index(ctx context.Context, req *IndexRequest) (*IndexResponse, error) {
	var response IndexResponse
	err := c.doRequestWithRetry(ctx, "POST", "/api/v1/index", req, &response)
	if err != nil {
		return nil, fmt.Errorf("index request failed: %w", err)
	}
	return &response, nil
}

// Search performs semantic search across code
func (c *APIClient) Search(ctx context.Context, query string, embedding []float32, filters SearchFilters) (*SearchResponse, error) {
	searchReq := map[string]interface{}{
		"query":     query,
		"embedding": embedding,
	}

	if filters.RepoID != "" {
		searchReq["repo_id"] = filters.RepoID
	}
	if filters.Language != "" {
		searchReq["language"] = filters.Language
	}
	if len(filters.Kind) > 0 {
		searchReq["kind"] = filters.Kind
	}
	if filters.Limit > 0 {
		searchReq["limit"] = filters.Limit
	}

	var response SearchResponse
	err := c.doRequestWithRetry(ctx, "POST", "/api/v1/search", searchReq, &response)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	return &response, nil
}

// GetCallers finds functions that call the specified symbol
func (c *APIClient) GetCallers(ctx context.Context, symbolID string) (*RelationshipResponse, error) {
	var response RelationshipResponse
	path := fmt.Sprintf("/api/v1/symbols/%s/callers", symbolID)
	err := c.doRequestWithRetry(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("get callers request failed: %w", err)
	}
	return &response, nil
}

// GetCallees finds functions called by the specified symbol
func (c *APIClient) GetCallees(ctx context.Context, symbolID string) (*RelationshipResponse, error) {
	var response RelationshipResponse
	path := fmt.Sprintf("/api/v1/symbols/%s/callees", symbolID)
	err := c.doRequestWithRetry(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("get callees request failed: %w", err)
	}
	return &response, nil
}

// GetDependencies finds dependencies of the specified symbol
func (c *APIClient) GetDependencies(ctx context.Context, symbolID string) (*DependencyResponse, error) {
	var response DependencyResponse
	path := fmt.Sprintf("/api/v1/symbols/%s/dependencies", symbolID)
	err := c.doRequestWithRetry(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("get dependencies request failed: %w", err)
	}
	return &response, nil
}

// GetFileSymbols retrieves all symbols in a file
func (c *APIClient) GetFileSymbols(ctx context.Context, fileID string) (*SymbolsResponse, error) {
	var response SymbolsResponse
	path := fmt.Sprintf("/api/v1/files/%s/symbols", fileID)
	err := c.doRequestWithRetry(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("get file symbols request failed: %w", err)
	}
	return &response, nil
}

// Health checks API server health
func (c *APIClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// doRequestWithRetry performs an HTTP request with exponential backoff retry logic
func (c *APIClient) doRequestWithRetry(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.doRequest(ctx, method, path, body, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !c.isRetryable(err) {
			return err
		}
	}

	return fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// doRequest performs a single HTTP request
func (c *APIClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error response
		var errResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				return &APIError{
					StatusCode: resp.StatusCode,
					Message:    errMsg,
					Details:    errResp["details"],
				}
			}
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	// Parse response
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// isRetryable determines if an error is retryable
func (c *APIClient) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for API errors
	if apiErr, ok := err.(*APIError); ok {
		// Retry on server errors (5xx) and rate limiting (429)
		return apiErr.StatusCode >= 500 || apiErr.StatusCode == 429
	}

	// Retry on network errors
	return true
}

// APIError represents an error response from the API
type APIError struct {
	StatusCode int
	Message    string
	Details    interface{}
}

func (e *APIError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("API error (status %d): %s - %v", e.StatusCode, e.Message, e.Details)
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}
