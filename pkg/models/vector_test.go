package models

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// Unit tests for vector format conversion functions

func TestFormatVectorForPgvector(t *testing.T) {
	tests := []struct {
		name     string
		input    []float32
		expected string
	}{
		{
			name:     "empty vector",
			input:    []float32{},
			expected: "[]",
		},
		{
			name:     "single element",
			input:    []float32{0.5},
			expected: "[0.5]",
		},
		{
			name:     "multiple elements",
			input:    []float32{0.1, 0.2, 0.3},
			expected: "[0.1,0.2,0.3]",
		},
		{
			name:     "negative values",
			input:    []float32{-0.5, 0.0, 0.5},
			expected: "[-0.5,0,0.5]",
		},
		{
			name:     "small values",
			input:    []float32{0.001, 0.002, 0.003},
			expected: "[0.001,0.002,0.003]",
		},
		{
			name:     "large values",
			input:    []float32{100.5, 200.75, 300.25},
			expected: "[100.5,200.75,300.25]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatVectorForPgvector(tt.input)
			if result != tt.expected {
				t.Errorf("formatVectorForPgvector() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseVectorFromPgvector(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []float32
		wantError bool
	}{
		{
			name:      "empty vector",
			input:     "[]",
			expected:  []float32{},
			wantError: false,
		},
		{
			name:      "single element",
			input:     "[0.5]",
			expected:  []float32{0.5},
			wantError: false,
		},
		{
			name:      "multiple elements",
			input:     "[0.1,0.2,0.3]",
			expected:  []float32{0.1, 0.2, 0.3},
			wantError: false,
		},
		{
			name:      "negative values",
			input:     "[-0.5,0,0.5]",
			expected:  []float32{-0.5, 0.0, 0.5},
			wantError: false,
		},
		{
			name:      "with spaces",
			input:     "[0.1, 0.2, 0.3]",
			expected:  []float32{0.1, 0.2, 0.3},
			wantError: false,
		},
		{
			name:      "invalid format - no brackets",
			input:     "0.1,0.2,0.3",
			expected:  []float32{0.1, 0.2, 0.3},
			wantError: false,
		},
		{
			name:      "invalid element",
			input:     "[0.1,abc,0.3]",
			expected:  nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVectorFromPgvector(tt.input)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("parseVectorFromPgvector() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("parseVectorFromPgvector() unexpected error: %v", err)
				return
			}
			
			if len(result) != len(tt.expected) {
				t.Errorf("parseVectorFromPgvector() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("parseVectorFromPgvector()[%d] = %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestFormatAndParseRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input []float32
	}{
		{
			name:  "empty",
			input: []float32{},
		},
		{
			name:  "single",
			input: []float32{0.5},
		},
		{
			name:  "multiple",
			input: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		},
		{
			name:  "mixed values",
			input: []float32{-1.5, 0.0, 1.5, 100.25, -50.75},
		},
		{
			name:  "typical embedding",
			input: []float32{0.123, 0.456, 0.789, -0.321, -0.654},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Format to pgvector string
			formatted := formatVectorForPgvector(tt.input)
			
			// Parse back to []float32
			parsed, err := parseVectorFromPgvector(formatted)
			if err != nil {
				t.Fatalf("parseVectorFromPgvector() error: %v", err)
			}
			
			// Compare
			if len(parsed) != len(tt.input) {
				t.Errorf("Round trip length mismatch: got %d, want %d", len(parsed), len(tt.input))
				return
			}
			
			for i := range parsed {
				// Allow small floating point differences
				diff := parsed[i] - tt.input[i]
				if diff < -0.0001 || diff > 0.0001 {
					t.Errorf("Round trip value mismatch at index %d: got %v, want %v", i, parsed[i], tt.input[i])
				}
			}
		})
	}
}

// Integration tests for VectorRepository (require database)

func TestVectorRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create repository and file first
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-vector-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	fileRepo := NewFileRepository(testDB.DB)
	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repository.RepoID,
		Path:     "test.go",
		Language: "go",
		Size:     1024,
		Checksum: "abc123",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	symbolRepo := NewSymbolRepository(testDB.DB)
	symbol := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "TestFunction",
		Kind:      "function",
		Signature: "func TestFunction()",
		StartLine: 10,
		EndLine:   20,
		StartByte: 100,
		EndByte:   200,
	}
	err = symbolRepo.Create(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	// Create vector with 1024 dimensions (as defined in schema)
	vectorRepo := NewVectorRepository(testDB.DB)
	embedding := make([]float32, 1024)
	for i := range embedding {
		embedding[i] = float32(i) / 1000.0
	}
	vector := &Vector{
		VectorID:   uuid.New().String(),
		EntityID:   symbol.SymbolID,
		EntityType: "symbol",
		Embedding:  embedding,
		Content:    "Test function content",
		Model:      "test-model",
		ChunkIndex: 0,
	}

	err = vectorRepo.Create(ctx, vector)
	if err != nil {
		t.Fatalf("Failed to create vector: %v", err)
	}

	// Verify the vector was created
	retrieved, err := vectorRepo.GetByID(ctx, vector.VectorID)
	if err != nil {
		t.Fatalf("Failed to retrieve vector: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Vector not found")
	}

	if retrieved.EntityID != vector.EntityID {
		t.Errorf("Expected entity_id %s, got %s", vector.EntityID, retrieved.EntityID)
	}
	if retrieved.EntityType != vector.EntityType {
		t.Errorf("Expected entity_type %s, got %s", vector.EntityType, retrieved.EntityType)
	}
	if retrieved.Content != vector.Content {
		t.Errorf("Expected content %s, got %s", vector.Content, retrieved.Content)
	}
	if retrieved.Model != vector.Model {
		t.Errorf("Expected model %s, got %s", vector.Model, retrieved.Model)
	}

	// Verify embedding
	if len(retrieved.Embedding) != len(embedding) {
		t.Errorf("Expected embedding length %d, got %d", len(embedding), len(retrieved.Embedding))
	}
	for i := range embedding {
		diff := retrieved.Embedding[i] - embedding[i]
		if diff < -0.0001 || diff > 0.0001 {
			t.Errorf("Embedding mismatch at index %d: got %v, want %v", i, retrieved.Embedding[i], embedding[i])
		}
	}
}

func TestVectorRepository_GetByEntityID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create test data
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-get-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	fileRepo := NewFileRepository(testDB.DB)
	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repository.RepoID,
		Path:     "test.go",
		Language: "go",
		Size:     1024,
		Checksum: "abc123",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	symbolRepo := NewSymbolRepository(testDB.DB)
	symbol := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "TestFunction",
		Kind:      "function",
		Signature: "func TestFunction()",
		StartLine: 10,
		EndLine:   20,
		StartByte: 100,
		EndByte:   200,
	}
	err = symbolRepo.Create(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	// Create multiple vectors for the same entity with 1024 dimensions
	vectorRepo := NewVectorRepository(testDB.DB)
	
	createEmbedding := func(offset int) []float32 {
		emb := make([]float32, 1024)
		for i := range emb {
			emb[i] = float32(i+offset) / 1000.0
		}
		return emb
	}
	
	vectors := []*Vector{
		{
			VectorID:   uuid.New().String(),
			EntityID:   symbol.SymbolID,
			EntityType: "symbol",
			Embedding:  createEmbedding(0),
			Content:    "Chunk 0",
			Model:      "test-model",
			ChunkIndex: 0,
		},
		{
			VectorID:   uuid.New().String(),
			EntityID:   symbol.SymbolID,
			EntityType: "symbol",
			Embedding:  createEmbedding(1000),
			Content:    "Chunk 1",
			Model:      "test-model",
			ChunkIndex: 1,
		},
		{
			VectorID:   uuid.New().String(),
			EntityID:   symbol.SymbolID,
			EntityType: "symbol",
			Embedding:  createEmbedding(2000),
			Content:    "Chunk 2",
			Model:      "test-model",
			ChunkIndex: 2,
		},
	}

	err = vectorRepo.BatchCreate(ctx, vectors)
	if err != nil {
		t.Fatalf("Failed to batch create vectors: %v", err)
	}

	// Retrieve vectors by entity ID
	retrieved, err := vectorRepo.GetByEntityID(ctx, symbol.SymbolID, "symbol")
	if err != nil {
		t.Fatalf("Failed to retrieve vectors by entity ID: %v", err)
	}

	if len(retrieved) != len(vectors) {
		t.Errorf("Expected %d vectors, got %d", len(vectors), len(retrieved))
	}

	// Verify vectors are sorted by chunk_index
	for i := 1; i < len(retrieved); i++ {
		if retrieved[i-1].ChunkIndex > retrieved[i].ChunkIndex {
			t.Error("Vectors are not sorted by chunk_index")
			break
		}
	}

	// Verify embeddings are correctly retrieved
	for i, vec := range retrieved {
		if len(vec.Embedding) != 1024 {
			t.Errorf("Vector %d: expected embedding length 1024, got %d", i, len(vec.Embedding))
		}
	}
}

func TestVectorRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create test data
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-update-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	fileRepo := NewFileRepository(testDB.DB)
	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repository.RepoID,
		Path:     "test.go",
		Language: "go",
		Size:     1024,
		Checksum: "abc123",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	symbolRepo := NewSymbolRepository(testDB.DB)
	symbol := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "TestFunction",
		Kind:      "function",
		Signature: "func TestFunction()",
		StartLine: 10,
		EndLine:   20,
		StartByte: 100,
		EndByte:   200,
	}
	err = symbolRepo.Create(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	// Create vector with 1024 dimensions
	vectorRepo := NewVectorRepository(testDB.DB)
	originalEmbedding := make([]float32, 1024)
	for i := range originalEmbedding {
		originalEmbedding[i] = float32(i) / 1000.0
	}
	
	vector := &Vector{
		VectorID:   uuid.New().String(),
		EntityID:   symbol.SymbolID,
		EntityType: "symbol",
		Embedding:  originalEmbedding,
		Content:    "Original content",
		Model:      "original-model",
		ChunkIndex: 0,
	}

	err = vectorRepo.Create(ctx, vector)
	if err != nil {
		t.Fatalf("Failed to create vector: %v", err)
	}

	// Update the vector
	updatedEmbedding := make([]float32, 1024)
	for i := range updatedEmbedding {
		updatedEmbedding[i] = float32(i+1000) / 1000.0
	}
	vector.Embedding = updatedEmbedding
	vector.Content = "Updated content"
	vector.Model = "updated-model"

	err = vectorRepo.Update(ctx, vector)
	if err != nil {
		t.Fatalf("Failed to update vector: %v", err)
	}

	// Verify the update
	retrieved, err := vectorRepo.GetByID(ctx, vector.VectorID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated vector: %v", err)
	}

	if retrieved.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got %s", retrieved.Content)
	}
	if retrieved.Model != "updated-model" {
		t.Errorf("Expected model 'updated-model', got %s", retrieved.Model)
	}

	// Verify embedding was updated
	if len(retrieved.Embedding) != 1024 {
		t.Errorf("Expected embedding length 1024, got %d", len(retrieved.Embedding))
	}
	// Check a few sample values
	for i := 0; i < 10; i++ {
		expected := float32(i+1000) / 1000.0
		diff := retrieved.Embedding[i] - expected
		if diff < -0.001 || diff > 0.001 {
			t.Errorf("Embedding mismatch at index %d: got %v, want %v", i, retrieved.Embedding[i], expected)
		}
	}
}

func TestVectorRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create test data
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-delete-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	fileRepo := NewFileRepository(testDB.DB)
	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repository.RepoID,
		Path:     "test.go",
		Language: "go",
		Size:     1024,
		Checksum: "abc123",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	symbolRepo := NewSymbolRepository(testDB.DB)
	symbol := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "TestFunction",
		Kind:      "function",
		Signature: "func TestFunction()",
		StartLine: 10,
		EndLine:   20,
		StartByte: 100,
		EndByte:   200,
	}
	err = symbolRepo.Create(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	// Create vector with 1024 dimensions
	vectorRepo := NewVectorRepository(testDB.DB)
	embedding := make([]float32, 1024)
	for i := range embedding {
		embedding[i] = float32(i) / 1000.0
	}
	
	vector := &Vector{
		VectorID:   uuid.New().String(),
		EntityID:   symbol.SymbolID,
		EntityType: "symbol",
		Embedding:  embedding,
		Content:    "Test content",
		Model:      "test-model",
		ChunkIndex: 0,
	}

	err = vectorRepo.Create(ctx, vector)
	if err != nil {
		t.Fatalf("Failed to create vector: %v", err)
	}

	// Delete the vector
	err = vectorRepo.Delete(ctx, vector.VectorID)
	if err != nil {
		t.Fatalf("Failed to delete vector: %v", err)
	}

	// Verify the vector is gone
	retrieved, err := vectorRepo.GetByID(ctx, vector.VectorID)
	if err != nil {
		t.Fatalf("Failed to check if vector exists: %v", err)
	}
	if retrieved != nil {
		t.Error("Vector should have been deleted")
	}
}
