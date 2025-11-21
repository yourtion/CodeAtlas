package indexer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// getEnvInt retrieves an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func TestOpenAIEmbedder_GenerateEmbedding(t *testing.T) {
	// Get vector dimensions from environment (same as database schema)
	vectorDim := getEnvInt("EMBEDDING_DIMENSIONS", 1024)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse request
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Return mock response
		resp := OpenAIEmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: make([]float32, vectorDim),
					Index:     0,
				},
			},
			Model: "test-model",
		}

		// Fill embedding with test data
		for i := range resp.Data[0].Embedding {
			resp.Data[0].Embedding[i] = float32(i) * 0.001
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	vectorRepo := models.NewVectorRepository(db)

	// Create embedder
	config := &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          server.URL,
		Model:                "test-model",
		Dimensions:           vectorDim,
		BatchSize:            10,
		MaxRequestsPerSecond: 100,
		MaxRetries:           3,
		BaseRetryDelay:       10 * time.Millisecond,
		MaxRetryDelay:        100 * time.Millisecond,
		Timeout:              5 * time.Second,
	}

	embedder := NewOpenAIEmbedder(config, vectorRepo)

	// Test embedding generation
	ctx := context.Background()
	content := "func TestFunction() { return 42 }"
	embedding, err := embedder.GenerateEmbedding(ctx, content)

	if err != nil {
		t.Fatalf("Failed to generate embedding: %v", err)
	}

	if len(embedding) != vectorDim {
		t.Errorf("Expected embedding length %d, got %d", vectorDim, len(embedding))
	}
}

func TestOpenAIEmbedder_BatchEmbed(t *testing.T) {
	// Get vector dimensions from environment (same as database schema)
	vectorDim := getEnvInt("EMBEDDING_DIMENSIONS", 1024)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Get input texts
		inputs := req["input"].([]interface{})
		numInputs := len(inputs)

		// Return mock response with multiple embeddings
		resp := OpenAIEmbeddingResponse{
			Object: "list",
			Model:  "test-model",
		}

		for i := 0; i < numInputs; i++ {
			embedding := make([]float32, vectorDim)
			for j := range embedding {
				// Use simpler values to avoid floating-point precision issues
				embedding[j] = float32(j) / 1000.0
			}
			resp.Data = append(resp.Data, struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				Object:    "embedding",
				Embedding: embedding,
				Index:     i,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	vectorRepo := models.NewVectorRepository(db)

	// Create embedder
	config := &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          server.URL,
		Model:                "test-model",
		Dimensions:           vectorDim,
		BatchSize:            10,
		MaxRequestsPerSecond: 100,
		MaxRetries:           3,
		BaseRetryDelay:       10 * time.Millisecond,
		MaxRetryDelay:        100 * time.Millisecond,
		Timeout:              5 * time.Second,
	}

	embedder := NewOpenAIEmbedder(config, vectorRepo)

	// Test batch embedding
	ctx := context.Background()
	texts := []string{
		"func TestFunction1() { return 1 }",
		"func TestFunction2() { return 2 }",
		"func TestFunction3() { return 3 }",
	}

	embeddings, err := embedder.BatchEmbed(ctx, texts)

	if err != nil {
		t.Fatalf("Failed to batch embed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, embedding := range embeddings {
		if len(embedding) != vectorDim {
			t.Errorf("Embedding %d: expected length %d, got %d", i, vectorDim, len(embedding))
		}
	}
}

func TestOpenAIEmbedder_EmbedSymbols(t *testing.T) {
	// Get vector dimensions from environment (same as database schema)
	vectorDim := getEnvInt("EMBEDDING_DIMENSIONS", 1024)

	// Create mock server
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Parse request
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Get input texts
		inputs := req["input"].([]interface{})
		numInputs := len(inputs)

		// Return mock response
		resp := OpenAIEmbeddingResponse{
			Object: "list",
			Model:  "test-model",
		}

		for i := 0; i < numInputs; i++ {
			embedding := make([]float32, vectorDim)
			for j := range embedding {
				// Use simpler values to avoid floating-point precision issues
				embedding[j] = float32(j) / 1000.0
			}
			resp.Data = append(resp.Data, struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				Object:    "embedding",
				Embedding: embedding,
				Index:     i,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	vectorRepo := models.NewVectorRepository(db)

	// Create embedder
	config := &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          server.URL,
		Model:                "test-model",
		Dimensions:           vectorDim,
		BatchSize:            2,
		MaxRequestsPerSecond: 100,
		MaxRetries:           3,
		BaseRetryDelay:       10 * time.Millisecond,
		MaxRetryDelay:        100 * time.Millisecond,
		Timeout:              5 * time.Second,
	}

	embedder := NewOpenAIEmbedder(config, vectorRepo)

	// Create test symbols
	symbols := []schema.Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    uuid.New().String(),
			Name:      "TestFunction1",
			Kind:      schema.SymbolFunction,
			Signature: "func TestFunction1() int",
			Docstring: "Test function 1",
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    uuid.New().String(),
			Name:      "TestFunction2",
			Kind:      schema.SymbolFunction,
			Signature: "func TestFunction2() int",
			Docstring: "Test function 2",
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    uuid.New().String(),
			Name:      "TestFunction3",
			Kind:      schema.SymbolFunction,
			Signature: "func TestFunction3() int",
			Docstring: "Test function 3",
		},
	}

	// Test embedding symbols
	ctx := context.Background()
	result, err := embedder.EmbedSymbols(ctx, symbols)

	if err != nil {
		t.Fatalf("Failed to embed symbols: %v", err)
	}

	if result.VectorsCreated != len(symbols) {
		t.Errorf("Expected %d vectors created, got %d", len(symbols), result.VectorsCreated)
	}

	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors, got %d errors", len(result.Errors))
		for _, err := range result.Errors {
			t.Logf("Error: %s - %s", err.EntityID, err.Message)
		}
	}

	// Verify batching (3 symbols with batch size 2 should make 2 requests)
	expectedRequests := 2
	if requestCount != expectedRequests {
		t.Errorf("Expected %d API requests, got %d", expectedRequests, requestCount)
	}

	// Verify vectors were stored
	for _, symbol := range symbols {
		vectors, err := vectorRepo.GetByEntityID(ctx, symbol.SymbolID, "symbol")
		if err != nil {
			t.Errorf("Failed to get vectors for symbol %s: %v", symbol.SymbolID, err)
		}
		if len(vectors) != 1 {
			t.Errorf("Expected 1 vector for symbol %s, got %d", symbol.SymbolID, len(vectors))
		}
	}
}

func TestOpenAIEmbedder_RetryLogic(t *testing.T) {
	// Get vector dimensions from environment (same as database schema)
	vectorDim := getEnvInt("EMBEDDING_DIMENSIONS", 1024)

	// Create mock server that fails first 2 times
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Return 503 error
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service temporarily unavailable"))
			return
		}

		// Return success on 3rd attempt
		resp := OpenAIEmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: make([]float32, vectorDim),
					Index:     0,
				},
			},
			Model: "test-model",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	vectorRepo := models.NewVectorRepository(db)

	// Create embedder
	config := &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          server.URL,
		Model:                "test-model",
		Dimensions:           vectorDim,
		BatchSize:            10,
		MaxRequestsPerSecond: 100,
		MaxRetries:           3,
		BaseRetryDelay:       10 * time.Millisecond,
		MaxRetryDelay:        100 * time.Millisecond,
		Timeout:              5 * time.Second,
	}

	embedder := NewOpenAIEmbedder(config, vectorRepo)

	// Test embedding with retries
	ctx := context.Background()
	content := "func TestFunction() { return 42 }"
	embedding, err := embedder.GenerateEmbedding(ctx, content)

	if err != nil {
		t.Fatalf("Failed to generate embedding after retries: %v", err)
	}

	if len(embedding) != vectorDim {
		t.Errorf("Expected embedding length %d, got %d", vectorDim, len(embedding))
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestOpenAIEmbedder_RateLimiting(t *testing.T) {
	// Get vector dimensions from environment (same as database schema)
	vectorDim := getEnvInt("EMBEDDING_DIMENSIONS", 1024)

	// Create mock server
	requestTimes := []time.Time{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())

		resp := OpenAIEmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: make([]float32, vectorDim),
					Index:     0,
				},
			},
			Model: "test-model",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	vectorRepo := models.NewVectorRepository(db)

	// Create embedder with low rate limit
	config := &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          server.URL,
		Model:                "test-model",
		Dimensions:           vectorDim,
		BatchSize:            1,
		MaxRequestsPerSecond: 2, // 2 requests per second
		MaxRetries:           3,
		BaseRetryDelay:       10 * time.Millisecond,
		MaxRetryDelay:        100 * time.Millisecond,
		Timeout:              5 * time.Second,
	}

	embedder := NewOpenAIEmbedder(config, vectorRepo)

	// Make 4 requests
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		_, err := embedder.GenerateEmbedding(ctx, "test content")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	// Verify rate limiting (should take at least 1 second for 4 requests at 2 req/s)
	if len(requestTimes) != 4 {
		t.Fatalf("Expected 4 requests, got %d", len(requestTimes))
	}

	duration := requestTimes[3].Sub(requestTimes[0])
	minDuration := time.Second
	if duration < minDuration {
		t.Errorf("Rate limiting not working: 4 requests completed in %v (expected >= %v)", duration, minDuration)
	}
}

func TestOpenAIEmbedder_EmptyContent(t *testing.T) {
	// Create test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	vectorRepo := models.NewVectorRepository(db)

	// Create embedder
	config := DefaultEmbedderConfig()
	embedder := NewOpenAIEmbedder(config, vectorRepo)

	// Test with empty content
	ctx := context.Background()
	_, err := embedder.GenerateEmbedding(ctx, "")

	if err == nil {
		t.Error("Expected error for empty content, got nil")
	}
}

func TestOpenAIEmbedder_DimensionValidation(t *testing.T) {
	// Get vector dimensions from environment (same as database schema)
	vectorDim := getEnvInt("EMBEDDING_DIMENSIONS", 1024)

	// Create mock server that returns wrong dimensions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OpenAIEmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: make([]float32, 512), // Wrong dimensions
					Index:     0,
				},
			},
			Model: "test-model",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	vectorRepo := models.NewVectorRepository(db)

	// Create embedder expecting correct dimensions
	config := &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          server.URL,
		Model:                "test-model",
		Dimensions:           vectorDim,
		BatchSize:            10,
		MaxRequestsPerSecond: 100,
		MaxRetries:           3,
		BaseRetryDelay:       10 * time.Millisecond,
		MaxRetryDelay:        100 * time.Millisecond,
		Timeout:              5 * time.Second,
	}

	embedder := NewOpenAIEmbedder(config, vectorRepo)

	// Create test symbol
	symbol := schema.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    uuid.New().String(),
		Name:      "TestFunction",
		Kind:      schema.SymbolFunction,
		Signature: "func TestFunction() int",
		Docstring: "Test function",
	}

	// Test embedding with dimension mismatch
	ctx := context.Background()
	result, err := embedder.EmbedSymbols(ctx, []schema.Symbol{symbol})

	if err != nil {
		t.Fatalf("EmbedSymbols failed: %v", err)
	}

	// Should have error for dimension mismatch
	if len(result.Errors) == 0 {
		t.Error("Expected dimension validation error, got none")
	}

	if result.VectorsCreated != 0 {
		t.Errorf("Expected 0 vectors created, got %d", result.VectorsCreated)
	}
}

func TestBuildSymbolContent(t *testing.T) {
	config := DefaultEmbedderConfig()
	embedder := NewOpenAIEmbedder(config, nil)

	tests := []struct {
		name     string
		symbol   schema.Symbol
		expected string
	}{
		{
			name: "symbol with all fields",
			symbol: schema.Symbol{
				Signature:       "func TestFunction() int",
				Docstring:       "Test function docstring",
				SemanticSummary: "Returns an integer",
			},
			expected: "func TestFunction() int\nTest function docstring\nReturns an integer",
		},
		{
			name: "symbol with only signature",
			symbol: schema.Symbol{
				Signature: "func TestFunction() int",
			},
			expected: "func TestFunction() int",
		},
		{
			name: "symbol with signature and docstring",
			symbol: schema.Symbol{
				Signature: "func TestFunction() int",
				Docstring: "Test function docstring",
			},
			expected: "func TestFunction() int\nTest function docstring",
		},
		{
			name:     "symbol with no content",
			symbol:   schema.Symbol{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := embedder.buildSymbolContent(tt.symbol)
			if content != tt.expected {
				t.Errorf("Expected content:\n%s\nGot:\n%s", tt.expected, content)
			}
		})
	}
}
