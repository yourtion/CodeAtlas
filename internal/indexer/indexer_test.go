package indexer

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*models.DB, func()) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we should skip database tests
	if os.Getenv("SKIP_DB_TESTS") == "1" {
		t.Skip("Skipping database tests (SKIP_DB_TESTS=1)")
	}

	// Connect to database (uses environment variables)
	db, err := models.NewDB()
	if err != nil {
		t.Skipf("Failed to connect to test database: %v (set DB_* env vars or SKIP_DB_TESTS=1)", err)
		return nil, func() {}
	}

	// Initialize schema
	ctx := context.Background()
	schemaManager := models.NewSchemaManager(db)
	if err := schemaManager.InitializeSchema(ctx); err != nil {
		db.Close()
		t.Skipf("Failed to initialize schema: %v", err)
		return nil, func() {}
	}

	// Return cleanup function
	cleanup := func() {
		if db != nil {
			// Clean up test data
			ctx := context.Background()
			db.ExecContext(ctx, "TRUNCATE TABLE repositories CASCADE")
			db.Close()
		}
	}

	return db, cleanup
}

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

	// Create valid input first
	input := createTestParseOutput()

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("index failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be returned")
	}

	// Check result structure
	if result.Status == "" {
		t.Error("expected status to be set")
	}

	if result.Summary == nil {
		t.Fatal("expected summary to be populated")
	}

	// Check error summary fields exist (even if 0 errors)
	if result.Summary["total_errors"] == nil {
		t.Error("expected total_errors in summary")
	}

	if result.Summary["error_types"] == nil {
		t.Error("expected error_types in summary")
	}

	if result.Summary["validation_errors"] == nil {
		t.Error("expected validation_errors in summary")
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

	// Expected files = input files + external file (if external dependencies exist)
	expectedFiles := len(input.Files)
	// Check if external file was created
	hasExternalFile := false
	for _, f := range files {
		if f.Path == "__external__" {
			hasExternalFile = true
			expectedFiles++
			break
		}
	}

	if len(files) != expectedFiles {
		t.Errorf("expected %d files in database (including external file: %v), got: %d", expectedFiles, hasExternalFile, len(files))
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

// TestIndexWithGraphBuilder tests indexing with graph building
func TestIndexWithGraphBuilder(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.SkipVectors = true

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	// Create test input with relationships
	input := createTestParseOutputWithRelationships()

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("index with graph failed: %v", err)
	}

	if result.Summary["graph_nodes_created"] == nil {
		t.Error("expected graph nodes to be created")
	}

	if result.Summary["graph_edges_created"] == nil {
		t.Error("expected graph edges to be created")
	}
}

// TestIndexWithEmbeddings tests indexing with embedding generation
func TestIndexWithEmbeddings(t *testing.T) {
	// Skip if no OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping embedding test (no OPENAI_API_KEY)")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.SkipVectors = false
	config.WorkerCount = 2

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	input := createTestParseOutput()

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("index with embeddings failed: %v", err)
	}

	if result.VectorsCreated == 0 {
		t.Error("expected vectors to be created")
	}
}

// TestIndexContextCancellation tests context cancellation during indexing
func TestIndexContextCancellation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.SkipVectors = true

	indexer := NewIndexer(db, config)

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	input := createTestParseOutput()

	_, err := indexer.Index(ctx, input)
	// Should handle cancellation gracefully
	if err != nil {
		t.Logf("index with cancelled context returned error (expected): %v", err)
	}
}

// TestIndexLargeDataset tests indexing with a large dataset
func TestIndexLargeDataset(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "large-repo"
	config.SkipVectors = true
	config.BatchSize = 50
	config.WorkerCount = 4

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	// Create large dataset (200 files)
	input := createTestParseOutputWithMultipleFiles(200)

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("large dataset index failed: %v", err)
	}

	if result.FilesProcessed != 200 {
		t.Errorf("expected 200 files processed, got: %d", result.FilesProcessed)
	}

	if result.Duration == 0 {
		t.Error("expected duration to be recorded")
	}

	t.Logf("Indexed 200 files in %s", result.Duration)
}

// TestIndexBatchProcessing tests batch processing logic
func TestIndexBatchProcessing(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "batch-test-repo"
	config.SkipVectors = true
	config.BatchSize = 10 // Small batch size to test batching

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	// Create 25 files (will require 3 batches with size 10)
	input := createTestParseOutputWithMultipleFiles(25)

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("batch processing failed: %v", err)
	}

	if result.FilesProcessed != 25 {
		t.Errorf("expected 25 files processed, got: %d", result.FilesProcessed)
	}
}

// TestIndexEmptyInput tests indexing with empty input
func TestIndexEmptyInput(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "empty-repo"
	config.SkipVectors = true

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	input := &schema.ParseOutput{
		Files:         []schema.File{},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   0,
			SuccessCount: 0,
			FailureCount: 0,
		},
	}

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("empty input index failed: %v", err)
	}

	if result.FilesProcessed != 0 {
		t.Errorf("expected 0 files processed, got: %d", result.FilesProcessed)
	}

	if result.Status != "success" {
		t.Errorf("expected success status for empty input, got: %s", result.Status)
	}
}

// TestIndexDuplicateFiles tests handling of duplicate file IDs
func TestIndexDuplicateFiles(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "duplicate-test-repo"
	config.SkipVectors = true

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	// First index
	input := createTestParseOutput()
	result1, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("first index failed: %v", err)
	}

	// Second index with same file IDs (should update)
	result2, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("second index failed: %v", err)
	}

	// Both should succeed
	if result1.Status != "success" && result1.Status != "success_with_warnings" {
		t.Errorf("first index status: %s", result1.Status)
	}

	if result2.Status != "success" && result2.Status != "success_with_warnings" {
		t.Errorf("second index status: %s", result2.Status)
	}
}

// TestIndexProgressStages tests all progress stages
func TestIndexProgressStages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "progress-test-repo"
	config.SkipVectors = true

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	input := createTestParseOutputWithMultipleFiles(5)

	progressChan := make(chan IndexProgress, 20)
	var progressUpdates []IndexProgress

	go func() {
		for progress := range progressChan {
			progressUpdates = append(progressUpdates, progress)
		}
	}()

	result, err := indexer.IndexWithProgress(ctx, input, progressChan)
	if err != nil {
		t.Fatalf("index with progress failed: %v", err)
	}

	// Wait for progress updates
	time.Sleep(200 * time.Millisecond)

	if len(progressUpdates) == 0 {
		t.Fatal("expected progress updates")
	}

	// Verify we got all expected stages
	stagesSeen := make(map[string]bool)
	for _, p := range progressUpdates {
		stagesSeen[p.Stage] = true
		t.Logf("Progress: stage=%s, progress=%.1f%%, message=%s", p.Stage, p.Progress, p.Message)
	}

	expectedStages := []string{"validation", "repository", "writing", "complete"}
	for _, stage := range expectedStages {
		if !stagesSeen[stage] {
			t.Errorf("missing expected stage: %s", stage)
		}
	}

	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("expected success, got: %s", result.Status)
	}
}

// TestIndexWithNilConfig tests indexer with nil config (should use defaults)
func TestIndexWithNilConfig(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	indexer := NewIndexer(db, nil)

	if indexer.config == nil {
		t.Fatal("expected default config to be created")
	}

	if indexer.config.BatchSize != 100 {
		t.Errorf("expected default batch size 100, got: %d", indexer.config.BatchSize)
	}

	if indexer.config.WorkerCount != 4 {
		t.Errorf("expected default worker count 4, got: %d", indexer.config.WorkerCount)
	}
}

// TestIndexResultSummary tests result summary population
func TestIndexResultSummary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "summary-test-repo"
	config.SkipVectors = true

	indexer := NewIndexer(db, config)
	ctx := context.Background()

	input := createTestParseOutput()

	result, err := indexer.Index(ctx, input)
	if err != nil {
		t.Fatalf("index failed: %v", err)
	}

	// Check summary fields
	if result.Summary == nil {
		t.Fatal("expected summary to be populated")
	}

	if result.Summary["total_errors"] == nil {
		t.Error("expected total_errors in summary")
	}

	if result.Summary["error_types"] == nil {
		t.Error("expected error_types in summary")
	}

	if result.Summary["validation_errors"] == nil {
		t.Error("expected validation_errors in summary")
	}
}

// Helper functions

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
						NodeID:     nodeID,
						FileID:     fileID,
						Type:       "function_declaration",
						Attributes: map[string]string{},
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

func createTestParseOutputWithRelationships() *schema.ParseOutput {
	fileID1 := uuid.New().String()
	fileID2 := uuid.New().String()
	symbolID1 := uuid.New().String()
	symbolID2 := uuid.New().String()

	return &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   fileID1,
				Path:     "main.go",
				Language: "go",
				Size:     1024,
				Checksum: "checksum1",
				Symbols: []schema.Symbol{
					{
						SymbolID:  symbolID1,
						FileID:    fileID1,
						Name:      "Main",
						Kind:      schema.SymbolFunction,
						Signature: "func Main()",
						Span: schema.Span{
							StartLine: 10,
							EndLine:   20,
							StartByte: 100,
							EndByte:   200,
						},
					},
				},
				Nodes: []schema.ASTNode{},
			},
			{
				FileID:   fileID2,
				Path:     "utils.go",
				Language: "go",
				Size:     512,
				Checksum: "checksum2",
				Symbols: []schema.Symbol{
					{
						SymbolID:  symbolID2,
						FileID:    fileID2,
						Name:      "Helper",
						Kind:      schema.SymbolFunction,
						Signature: "func Helper() string",
						Span: schema.Span{
							StartLine: 5,
							EndLine:   10,
							StartByte: 50,
							EndByte:   100,
						},
					},
				},
				Nodes: []schema.ASTNode{},
			},
		},
		Relationships: []schema.DependencyEdge{
			{
				EdgeID:     uuid.New().String(),
				SourceID:   symbolID1,
				TargetID:   symbolID2,
				EdgeType:   schema.EdgeCall,
				SourceFile: "main.go",
				TargetFile: "utils.go",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   2,
			SuccessCount: 2,
			FailureCount: 0,
		},
	}
}
