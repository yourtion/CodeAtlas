package indexer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestNewIndexer tests indexer creation
func TestNewIndexer(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"

	indexer := NewIndexer(db, config)

	if indexer == nil {
		t.Fatal("expected indexer to be created")
	}

	if indexer.validator == nil {
		t.Error("expected validator to be initialized")
	}

	if indexer.writer == nil {
		t.Error("expected writer to be initialized")
	}

	if indexer.graphBuilder == nil {
		t.Error("expected graph builder to be initialized")
	}

	if indexer.embedder == nil {
		t.Error("expected embedder to be initialized when SkipVectors is false")
	}
}

// TestNewIndexerWithSkipVectors tests indexer creation with vectors disabled
func TestNewIndexerWithSkipVectors(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.SkipVectors = true

	indexer := NewIndexer(db, config)

	if indexer.embedder != nil {
		t.Error("expected embedder to be nil when SkipVectors is true")
	}
}

// TestIndexValidInput tests indexing with valid input
func TestIndexValidInput(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.SkipVectors = true // Skip vectors for faster test
	config.WorkerCount = 1

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	// Create test input
	input := createTestParseOutput()

	// Index the data
	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be returned")
	}

	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("expected status to be success, got: %s", result.Status)
	}

	if result.FilesProcessed != len(input.Files) {
		t.Errorf("expected %d files processed, got: %d", len(input.Files), result.FilesProcessed)
	}

	if result.SymbolsCreated == 0 {
		t.Error("expected symbols to be created")
	}

	if result.Duration == 0 {
		t.Error("expected duration to be recorded")
	}
}

// TestIndexInvalidInput tests indexing with invalid input
func TestIndexInvalidInput(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	// Create invalid input (missing required fields)
	input := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   "", // Missing file ID
				Path:     "test.go",
				Language: "go",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	// Index should fail validation
	result, err := indexer.Index(ctx, input)
	if err == nil {
		t.Error("expected error for invalid input")
	}

	if result == nil {
		t.Fatal("expected result to be returned even on error")
	}

	if result.Status != "failed" {
		t.Errorf("expected status to be failed, got: %s", result.Status)
	}

	if len(result.Errors) == 0 {
		t.Error("expected validation errors to be reported")
	}
}

// TestIndexIncremental tests incremental indexing
func TestIndexIncremental(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.SkipVectors = true
	config.Incremental = false // First index without incremental

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	// First index
	input := createTestParseOutput()
	result1, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("first index failed: %v", err)
	}

	if result1.FilesProcessed != len(input.Files) {
		t.Errorf("expected %d files processed, got: %d", len(input.Files), result1.FilesProcessed)
	}

	// Second index with incremental (same checksums)
	config.Incremental = true
	indexer = NewIndexer(db, config)

	result2, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("second index failed: %v", err)
	}

	// With incremental and same checksums, should process 0 files
	if result2.FilesProcessed != 0 {
		t.Errorf("expected 0 files processed with incremental, got: %d", result2.FilesProcessed)
	}

	// Modify a file checksum
	input.Files[0].Checksum = "new-checksum"

	result3, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("third index failed: %v", err)
	}

	// Should process only the changed file
	if result3.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed with incremental, got: %d", result3.FilesProcessed)
	}
}

// TestIndexWithProgress tests indexing with progress tracking
func TestIndexWithProgress(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.SkipVectors = true

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	input := createTestParseOutput()

	// Create progress channel
	progressChan := make(chan IndexProgress, 10)
	var progressUpdates []IndexProgress

	// Collect progress updates in goroutine
	go func() {
		for progress := range progressChan {
			progressUpdates = append(progressUpdates, progress)
		}
	}()

	// Index with progress
	result, err := indexer.IndexWithProgress(ctx, input, progressChan)
	if err != nil {
		t.Fatalf("index with progress failed: %v", err)
	}

	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("expected success status, got: %s", result.Status)
	}

	// Wait a bit for progress updates to be collected
	time.Sleep(100 * time.Millisecond)

	if len(progressUpdates) == 0 {
		t.Error("expected progress updates to be sent")
	}

	// Check for expected stages
	stages := make(map[string]bool)
	for _, progress := range progressUpdates {
		stages[progress.Stage] = true
	}

	expectedStages := []string{"validation", "repository", "writing"}
	for _, stage := range expectedStages {
		if !stages[stage] {
			t.Errorf("expected stage %s in progress updates", stage)
		}
	}
}

// TestIndexParallelProcessing tests parallel embedding generation
func TestIndexParallelProcessing(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.WorkerCount = 4
	config.SkipVectors = true // Skip actual embeddings for test

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	// Create input with multiple files
	input := createTestParseOutputWithMultipleFiles(10)

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("parallel index failed: %v", err)
	}

	if result.FilesProcessed != len(input.Files) {
		t.Errorf("expected %d files processed, got: %d", len(input.Files), result.FilesProcessed)
	}
}

// TestIndexErrorCollection tests error collection during indexing
func TestIndexErrorCollection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.SkipVectors = true

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	// Create input with some invalid data
	input := createTestParseOutput()
	// Add an edge with invalid source ID (referential integrity error)
	input.Relationships = append(input.Relationships, schema.DependencyEdge{
		EdgeID:     uuid.New().String(),
		SourceID:   "invalid-source-id",
		TargetID:   "invalid-target-id",
		EdgeType:   schema.EdgeCall,
		SourceFile: "test.go",
	})

	result, err := indexer.Index(ctx, input)
	// Should not fail completely, but collect errors
	if err != nil {
		t.Logf("index returned error (expected): %v", err)
	}

	if result == nil {
		t.Fatal("expected result even with errors")
	}

	// Check that errors were collected
	if len(result.Errors) == 0 {
		t.Error("expected errors to be collected")
	}

	// Check error summary
	if result.Summary["total_errors"] == nil {
		t.Error("expected error summary to be populated")
	}
}

// TestIndexWithTransactions tests transactional indexing
func TestIndexWithTransactions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.UseTransactions = true
	config.SkipVectors = true

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	input := createTestParseOutput()

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("transactional index failed: %v", err)
	}

	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("expected success status, got: %s", result.Status)
	}

	// Verify data was committed
	fileRepo := models.NewFileRepository(db)
	files, err := fileRepo.GetByRepoID(ctx, config.RepoID)
	if err != nil {
		t.Fatalf("failed to get files: %v", err)
	}

	if len(files) != len(input.Files) {
		t.Errorf("expected %d files in database, got: %d", len(input.Files), len(files))
	}
}

// TestIndexWithoutTransactions tests non-transactional indexing
func TestIndexWithoutTransactions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.UseTransactions = false
	config.SkipVectors = true

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	input := createTestParseOutput()

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("non-transactional index failed: %v", err)
	}

	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("expected success status, got: %s", result.Status)
	}
}

// Helper functions
// Note: setupTestDB is defined in writer_test.go and shared across test files

func createTestParseOutput() *schema.ParseOutput {
	fileID := uuid.New().String()
	symbolID := uuid.New().String()
	nodeID := uuid.New().String()

	return &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   fileID,
				Path:     "test.go",
				Language: "go",
				Size:     1024,
				Checksum: "abc123",
				Symbols: []schema.Symbol{
					{
						SymbolID:  symbolID,
						FileID:    fileID,
						Name:      "TestFunction",
						Kind:      schema.SymbolFunction,
						Signature: "func TestFunction() error",
						Span: schema.Span{
							StartLine: 10,
							EndLine:   20,
							StartByte: 100,
							EndByte:   200,
						},
						Docstring:       "Test function documentation",
						SemanticSummary: "A test function",
					},
				},
				Nodes: []schema.ASTNode{
					{
						NodeID: nodeID,
						FileID: fileID,
						Type:   "function_declaration",
						Span: schema.Span{
							StartLine: 10,
							EndLine:   20,
							StartByte: 100,
							EndByte:   200,
						},
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}
}

func createTestParseOutputWithMultipleFiles(count int) *schema.ParseOutput {
	output := &schema.ParseOutput{
		Files:         make([]schema.File, count),
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   count,
			SuccessCount: count,
			FailureCount: 0,
		},
	}

	for i := 0; i < count; i++ {
		fileID := uuid.New().String()
		symbolID := uuid.New().String()

		output.Files[i] = schema.File{
			FileID:   fileID,
			Path:     fmt.Sprintf("test%d.go", i),
			Language: "go",
			Size:     1024,
			Checksum: fmt.Sprintf("checksum%d", i),
			Symbols: []schema.Symbol{
				{
					SymbolID:  symbolID,
					FileID:    fileID,
					Name:      fmt.Sprintf("TestFunction%d", i),
					Kind:      schema.SymbolFunction,
					Signature: fmt.Sprintf("func TestFunction%d() error", i),
					Span: schema.Span{
						StartLine: 10,
						EndLine:   20,
						StartByte: 100,
						EndByte:   200,
					},
				},
			},
			Nodes: []schema.ASTNode{},
		}
	}

	return output
}
