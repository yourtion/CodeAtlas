package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/api/handlers"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockEmbedder is a test embedder that returns a fixed embedding
type mockEmbedder struct {
	embedding []float32
}

func (m *mockEmbedder) GenerateEmbedding(ctx context.Context, content string) ([]float32, error) {
	return m.embedding, nil
}

func (m *mockEmbedder) EmbedSymbols(ctx context.Context, symbols []schema.Symbol) (*indexer.EmbedResult, error) {
	result := &indexer.EmbedResult{
		VectorsCreated: len(symbols),
		Duration:       0,
		Errors:         []indexer.EmbedError{},
	}
	return result, nil
}

func (m *mockEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = m.embedding
	}
	return embeddings, nil
}

// TestIndexHandlerIntegration tests the index handler with real database
func TestIndexHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	// Create handler with nil embedder config (will skip vectors)
	handler := handlers.NewIndexHandler(testDB.DB, nil)

	// Create test request
	parseOutput := createTestParseOutput()
	reqBody := handlers.IndexRequest{
		RepoName:    "test-api-repo",
		RepoURL:     "https://github.com/test/api-repo",
		Branch:      "main",
		ParseOutput: *parseOutput,
		Options: handlers.IndexOptions{
			Incremental: false,
			SkipVectors: true,
			BatchSize:   10,
			WorkerCount: 2,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/index", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Create router and handle request
	router := gin.New()
	router.POST("/api/v1/index", handler.Index)
	router.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Parse response
	var response handlers.IndexResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response
	if response.Status != "success" && response.Status != "success_with_warnings" {
		t.Errorf("Expected success status, got: %s", response.Status)
	}

	if response.FilesProcessed != len(parseOutput.Files) {
		t.Errorf("Expected %d files processed, got: %d", len(parseOutput.Files), response.FilesProcessed)
	}

	// Verify data was actually written to database
	ctx := context.Background()
	repoRepo := models.NewRepositoryRepository(testDB.DB)
	repo, err := repoRepo.GetByID(ctx, response.RepoID)
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}
	if repo == nil {
		t.Fatal("Repository not found in database")
	}
}

// TestSearchHandlerIntegration tests the search handler with real database
func TestSearchHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Get vector dimension from environment (same as schema initialization)
	vectorDim := getEnvInt("EMBEDDING_DIMENSIONS", 1024)

	// Setup test data
	repoRepo := models.NewRepositoryRepository(testDB.DB)
	fileRepo := models.NewFileRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)
	vectorRepo := models.NewVectorRepository(testDB.DB)

	// Create repository
	repo := &models.Repository{
		RepoID: uuid.New().String(),
		Name:   "search-test-repo",
	}
	if err := repoRepo.Create(ctx, repo); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create file
	file := &models.File{
		FileID:   uuid.New().String(),
		RepoID:   repo.RepoID,
		Path:     "test.go",
		Language: "go",
		Size:     1000,
		Checksum: "test123",
	}
	if err := fileRepo.Create(ctx, file); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create symbol
	symbol := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "SearchableFunction",
		Kind:      "function",
		Signature: "func SearchableFunction()",
		StartLine: 1,
		EndLine:   10,
		Docstring: "This is a searchable function",
	}
	if err := symbolRepo.Create(ctx, symbol); err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	// Create vector
	embedding := make([]float32, vectorDim)
	for i := range embedding {
		embedding[i] = 0.1
	}
	vector := &models.Vector{
		VectorID:   uuid.New().String(),
		EntityID:   symbol.SymbolID,
		EntityType: "symbol",
		Embedding:  embedding,
		Content:    symbol.Docstring,
		Model:      "test-model",
	}
	if err := vectorRepo.Create(ctx, vector); err != nil {
		t.Fatalf("Failed to create vector: %v", err)
	}

	// Use custom handler with mock embedder that returns our test embedding
	handler := handlers.NewSearchHandlerWithEmbedder(testDB.DB, &mockEmbedder{embedding: embedding})

	// Create search request
	reqBody := handlers.SearchRequest{
		Query: "searchable function",
		Limit: 10,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Create router and handle request
	router := gin.New()
	router.POST("/api/v1/search", handler.Search)
	router.ServeHTTP(w, req)

	// Check response - should return 200 OK with search results
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Parse response
	var response handlers.SearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response
	if len(response.Results) == 0 {
		t.Error("Expected search results, got none")
	}

	// Verify result contains our symbol
	found := false
	for _, result := range response.Results {
		if result.SymbolID == symbol.SymbolID {
			found = true
			if result.Name != symbol.Name {
				t.Errorf("Expected symbol name %s, got: %s", symbol.Name, result.Name)
			}
			// Cosine similarity ranges from -1 to 1
			if result.Similarity < -1 || result.Similarity > 1 {
				t.Errorf("Invalid similarity score (should be between -1 and 1): %f", result.Similarity)
			}
		}
	}

	if !found {
		t.Error("Expected to find our test symbol in search results")
	}
}

// TestRelationshipHandlerIntegration tests relationship queries
func TestRelationshipHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Setup test data with relationships
	repoRepo := models.NewRepositoryRepository(testDB.DB)
	fileRepo := models.NewFileRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)
	edgeRepo := models.NewEdgeRepository(testDB.DB)

	// Create repository
	repo := &models.Repository{
		RepoID: uuid.New().String(),
		Name:   "relationship-test-repo",
	}
	if err := repoRepo.Create(ctx, repo); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create file
	file := &models.File{
		FileID:   uuid.New().String(),
		RepoID:   repo.RepoID,
		Path:     "test.go",
		Language: "go",
		Size:     1000,
		Checksum: "test123",
	}
	if err := fileRepo.Create(ctx, file); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create symbols
	caller := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "CallerFunction",
		Kind:      "function",
		Signature: "func CallerFunction()",
		StartLine: 1,
		EndLine:   10,
	}
	callee := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "CalleeFunction",
		Kind:      "function",
		Signature: "func CalleeFunction()",
		StartLine: 11,
		EndLine:   20,
	}

	if err := symbolRepo.Create(ctx, caller); err != nil {
		t.Fatalf("Failed to create caller symbol: %v", err)
	}
	if err := symbolRepo.Create(ctx, callee); err != nil {
		t.Fatalf("Failed to create callee symbol: %v", err)
	}

	// Create edge (caller calls callee)
	edge := &models.Edge{
		EdgeID:     uuid.New().String(),
		SourceID:   caller.SymbolID,
		TargetID:   &callee.SymbolID,
		EdgeType:   "calls",
		SourceFile: file.Path,
		TargetFile: &file.Path,
	}
	if err := edgeRepo.Create(ctx, edge); err != nil {
		t.Fatalf("Failed to create edge: %v", err)
	}

	// Test GetCallees
	t.Run("GetCallees", func(t *testing.T) {
		edges, err := edgeRepo.GetBySourceID(ctx, caller.SymbolID)
		if err != nil {
			t.Fatalf("Failed to get callees: %v", err)
		}

		if len(edges) == 0 {
			t.Fatal("Expected callees, got none")
		}

		found := false
		for _, e := range edges {
			if e.TargetID != nil && *e.TargetID == callee.SymbolID {
				found = true
				if e.EdgeType != "calls" {
					t.Errorf("Expected edge type 'calls', got: %s", e.EdgeType)
				}
			}
		}

		if !found {
			t.Error("Expected to find callee in edges")
		}
	})

	// Test GetCallers
	t.Run("GetCallers", func(t *testing.T) {
		edges, err := edgeRepo.GetByTargetID(ctx, callee.SymbolID)
		if err != nil {
			t.Fatalf("Failed to get callers: %v", err)
		}

		if len(edges) == 0 {
			t.Fatal("Expected callers, got none")
		}

		found := false
		for _, e := range edges {
			if e.SourceID == caller.SymbolID {
				found = true
				if e.EdgeType != "calls" {
					t.Errorf("Expected edge type 'calls', got: %s", e.EdgeType)
				}
			}
		}

		if !found {
			t.Error("Expected to find caller in edges")
		}
	})
}

// TestInvalidRequests tests error handling for invalid API requests
func TestInvalidRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	handler := handlers.NewIndexHandler(testDB.DB, nil)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name:           "Empty request",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing repo name",
			requestBody: map[string]interface{}{
				"parse_output": map[string]interface{}{
					"files":         []interface{}{},
					"relationships": []interface{}{},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty parse output",
			requestBody: map[string]interface{}{
				"repo_name": "test-repo",
				"parse_output": map[string]interface{}{
					"files":         []interface{}{},
					"relationships": []interface{}{},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/index", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/api/v1/index", handler.Index)
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got: %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
