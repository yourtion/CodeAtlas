package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// Embedder interface defines methods for generating and storing embeddings
type Embedder interface {
	// GenerateEmbedding creates a vector for text content
	GenerateEmbedding(ctx context.Context, content string) ([]float32, error)

	// EmbedSymbols generates embeddings for symbols with docstrings
	EmbedSymbols(ctx context.Context, symbols []schema.Symbol) (*EmbedResult, error)

	// BatchEmbed processes multiple texts in a single API call
	BatchEmbed(ctx context.Context, texts []string) ([][]float32, error)
}

// EmbedderConfig contains configuration options for the embedder
type EmbedderConfig struct {
	// Backend type: "openai" or "local"
	Backend string `json:"backend"`

	// API endpoint URL (for OpenAI-compatible APIs)
	APIEndpoint string `json:"api_endpoint"`

	// API key for authentication (optional for local)
	APIKey string `json:"api_key"`

	// Model name to use for embeddings
	Model string `json:"model"`

	// Expected embedding dimensions
	Dimensions int `json:"dimensions"`

	// Batch size for API calls
	BatchSize int `json:"batch_size"`

	// Rate limiting: max requests per second
	MaxRequestsPerSecond int `json:"max_requests_per_second"`

	// Retry configuration
	MaxRetries     int           `json:"max_retries"`
	BaseRetryDelay time.Duration `json:"base_retry_delay"`
	MaxRetryDelay  time.Duration `json:"max_retry_delay"`

	// HTTP client timeout
	Timeout time.Duration `json:"timeout"`
}

// DefaultEmbedderConfig returns default configuration
// Note: Adjust Dimensions based on your model:
// - text-embedding-qwen3-embedding-0.6b: 1024 dimensions
// - nomic-embed-text: 768 dimensions
// - text-embedding-3-small (OpenAI): 1536 dimensions
func DefaultEmbedderConfig() *EmbedderConfig {
	return &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          "http://localhost:1234/v1/embeddings",
		Model:                "text-embedding-qwen3-embedding-0.6b",
		Dimensions:           768, // Adjust based on your model
		BatchSize:            50,
		MaxRequestsPerSecond: 10,
		MaxRetries:           3,
		BaseRetryDelay:       100 * time.Millisecond,
		MaxRetryDelay:        5 * time.Second,
		Timeout:              30 * time.Second,
	}
}

// OpenAIEmbedder implements the Embedder interface using OpenAI-compatible API
type OpenAIEmbedder struct {
	config      *EmbedderConfig
	httpClient  *http.Client
	vectorRepo  *models.VectorRepository
	rateLimiter *rateLimiter
	mu          sync.Mutex
}

// NewOpenAIEmbedder creates a new OpenAI-compatible embedder
func NewOpenAIEmbedder(config *EmbedderConfig, vectorRepo *models.VectorRepository) *OpenAIEmbedder {
	if config == nil {
		config = DefaultEmbedderConfig()
	}

	return &OpenAIEmbedder{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		vectorRepo:  vectorRepo,
		rateLimiter: newRateLimiter(config.MaxRequestsPerSecond),
	}
}

// EmbedResult contains the results of an embedding operation
type EmbedResult struct {
	VectorsCreated int           `json:"vectors_created"`
	Duration       time.Duration `json:"duration"`
	Errors         []EmbedError  `json:"errors,omitempty"`
}

// EmbedError represents an error that occurred during embedding
type EmbedError struct {
	EntityID string `json:"entity_id"`
	Message  string `json:"message"`
}

// GenerateEmbedding creates a vector for text content
func (e *OpenAIEmbedder) GenerateEmbedding(ctx context.Context, content string) ([]float32, error) {
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Wait for rate limiter
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Retry logic
	var lastErr error
	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(e.config.BaseRetryDelay) * math.Pow(2, float64(attempt-1)))
			if delay > e.config.MaxRetryDelay {
				delay = e.config.MaxRetryDelay
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		embedding, err := e.callEmbeddingAPI(ctx, []string{content})
		if err == nil && len(embedding) > 0 {
			return embedding[0], nil
		}
		lastErr = err

		// Don't retry on certain errors
		if !e.isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed to generate embedding after %d attempts: %w", e.config.MaxRetries+1, lastErr)
}

// BatchEmbed processes multiple texts in a single API call
func (e *OpenAIEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Wait for rate limiter
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Retry logic
	var lastErr error
	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(e.config.BaseRetryDelay) * math.Pow(2, float64(attempt-1)))
			if delay > e.config.MaxRetryDelay {
				delay = e.config.MaxRetryDelay
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		embeddings, err := e.callEmbeddingAPI(ctx, texts)
		if err == nil {
			return embeddings, nil
		}
		lastErr = err

		// Don't retry on certain errors
		if !e.isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed to batch embed after %d attempts: %w", e.config.MaxRetries+1, lastErr)
}

// EmbedSymbols generates embeddings for symbols with docstrings
func (e *OpenAIEmbedder) EmbedSymbols(ctx context.Context, symbols []schema.Symbol) (*EmbedResult, error) {
	startTime := time.Now()
	result := &EmbedResult{}

	if len(symbols) == 0 {
		return result, nil
	}

	// Filter symbols that need embeddings (have docstrings or semantic summaries)
	var symbolsToEmbed []schema.Symbol
	var contents []string
	for _, symbol := range symbols {
		content := e.buildSymbolContent(symbol)
		if content != "" {
			symbolsToEmbed = append(symbolsToEmbed, symbol)
			contents = append(contents, content)
		}
	}

	if len(symbolsToEmbed) == 0 {
		return result, nil
	}

	// Process in batches
	for i := 0; i < len(symbolsToEmbed); i += e.config.BatchSize {
		end := i + e.config.BatchSize
		if end > len(symbolsToEmbed) {
			end = len(symbolsToEmbed)
		}

		batchSymbols := symbolsToEmbed[i:end]
		batchContents := contents[i:end]

		// Generate embeddings for batch
		embeddings, err := e.BatchEmbed(ctx, batchContents)
		if err != nil {
			// Log error but continue with other batches
			for _, symbol := range batchSymbols {
				result.Errors = append(result.Errors, EmbedError{
					EntityID: symbol.SymbolID,
					Message:  fmt.Sprintf("failed to generate embedding: %v", err),
				})
			}
			continue
		}

		// Validate dimensions
		for j, embedding := range embeddings {
			if len(embedding) != e.config.Dimensions {
				result.Errors = append(result.Errors, EmbedError{
					EntityID: batchSymbols[j].SymbolID,
					Message:  fmt.Sprintf("invalid embedding dimensions: expected %d, got %d", e.config.Dimensions, len(embedding)),
				})
				continue
			}

			// Store embedding
			vector := &models.Vector{
				VectorID:   uuid.New().String(),
				EntityID:   batchSymbols[j].SymbolID,
				EntityType: "symbol",
				Embedding:  embedding,
				Content:    batchContents[j],
				Model:      e.config.Model,
				ChunkIndex: 0,
			}

			err := e.vectorRepo.Create(ctx, vector)
			if err != nil {
				result.Errors = append(result.Errors, EmbedError{
					EntityID: batchSymbols[j].SymbolID,
					Message:  fmt.Sprintf("failed to store embedding: %v", err),
				})
			} else {
				result.VectorsCreated++
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// buildSymbolContent constructs content for embedding from symbol
func (e *OpenAIEmbedder) buildSymbolContent(symbol schema.Symbol) string {
	var parts []string

	// Add signature
	if symbol.Signature != "" {
		parts = append(parts, symbol.Signature)
	}

	// Add docstring
	if symbol.Docstring != "" {
		parts = append(parts, symbol.Docstring)
	}

	// Add semantic summary
	if symbol.SemanticSummary != "" {
		parts = append(parts, symbol.SemanticSummary)
	}

	return strings.Join(parts, "\n")
}

// callEmbeddingAPI makes the actual API call to generate embeddings
func (e *OpenAIEmbedder) callEmbeddingAPI(ctx context.Context, texts []string) ([][]float32, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"input": texts,
		"model": e.config.Model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", e.config.APIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.config.APIKey))
	}

	// Make request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp OpenAIEmbeddingResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract embeddings
	embeddings := make([][]float32, len(apiResp.Data))
	for i, data := range apiResp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// isRetryableError determines if an error is retryable
func (e *OpenAIEmbedder) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors are retryable
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "temporary") {
		return true
	}

	// Rate limit errors are retryable
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") {
		return true
	}

	// Server errors are retryable
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	return false
}

// OpenAIEmbeddingResponse represents the API response structure
type OpenAIEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// rateLimiter implements token bucket rate limiting
type rateLimiter struct {
	rate       int
	interval   time.Duration
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(requestsPerSecond int) *rateLimiter {
	return &rateLimiter{
		rate:       requestsPerSecond,
		interval:   time.Second,
		tokens:     requestsPerSecond,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available
func (rl *rateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()

		// Refill tokens based on time elapsed
		now := time.Now()
		elapsed := now.Sub(rl.lastRefill)
		if elapsed >= rl.interval {
			rl.tokens = rl.rate
			rl.lastRefill = now
		}

		// Check if token available
		if rl.tokens > 0 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		// Calculate wait time
		waitTime := rl.interval - elapsed
		rl.mu.Unlock()

		// Wait or check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue loop to try again
		}
	}
}
