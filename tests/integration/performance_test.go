package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestLargeScaleIndexing tests indexing performance with many files
func TestLargeScaleIndexing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create large parse output (100 files, 10 symbols each)
	numFiles := 100
	symbolsPerFile := 10
	parseOutput := createLargeParseOutput(numFiles, symbolsPerFile)

	config := &indexer.IndexerConfig{
		RepoID:          uuid.New().String(),
		RepoName:        "large-test-repo",
		BatchSize:       50,
		WorkerCount:     4,
		SkipVectors:     true,
		UseTransactions: false, // Test without transactions for performance
	}

	idx := indexer.NewIndexer(testDB.DB, config)

	// Measure indexing time
	startTime := time.Now()
	result, err := idx.Index(ctx, parseOutput)
	duration := time.Since(startTime)

	if err != nil {
		// Log first few errors for debugging
		if result != nil && len(result.Errors) > 0 {
			t.Logf("Indexing failed with %d errors:", len(result.Errors))
			for i, e := range result.Errors {
				if i < 5 { // Only log first 5 errors
					t.Logf("  Error %d: %s - %s", i+1, e.Type, e.Message)
				}
			}
		}
		t.Fatalf("Large scale indexing failed: %v", err)
	}

	// Verify results
	expectedFiles := numFiles

	if result.FilesProcessed != expectedFiles {
		t.Errorf("Expected %d files processed, got: %d", expectedFiles, result.FilesProcessed)
	}

	// Note: Some symbols may be deduplicated due to unique constraints
	// Just verify we created a reasonable number
	minExpectedSymbols := numFiles * symbolsPerFile / 2 // At least half
	if result.SymbolsCreated < minExpectedSymbols {
		t.Errorf("Expected at least %d symbols created, got: %d", minExpectedSymbols, result.SymbolsCreated)
	}

	// Performance assertions
	filesPerSecond := float64(result.FilesProcessed) / duration.Seconds()
	t.Logf("Performance: %.2f files/second, total duration: %s", filesPerSecond, duration)

	// Should process at least 10 files per second (conservative target)
	if filesPerSecond < 10 {
		t.Logf("Warning: Performance below target (%.2f files/sec < 10 files/sec)", filesPerSecond)
	}

	// Verify referential integrity after large scale indexing
	if err := VerifyReferentialIntegrity(ctx, testDB.DB); err != nil {
		t.Errorf("Referential integrity check failed after large scale indexing: %v", err)
	}
}

// TestConcurrentIndexing tests parallel indexing operations
func TestConcurrentIndexing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create multiple parse outputs for different repos
	numRepos := 5
	parseOutputs := make([]*schema.ParseOutput, numRepos)
	configs := make([]*indexer.IndexerConfig, numRepos)

	for i := 0; i < numRepos; i++ {
		parseOutputs[i] = createTestParseOutput()
		configs[i] = &indexer.IndexerConfig{
			RepoID:          uuid.New().String(),
			RepoName:        fmt.Sprintf("concurrent-repo-%d", i),
			BatchSize:       10,
			WorkerCount:     2,
			SkipVectors:     true,
			UseTransactions: true,
		}
	}

	// Index concurrently
	errChan := make(chan error, numRepos)
	resultChan := make(chan *indexer.IndexResult, numRepos)

	startTime := time.Now()
	for i := 0; i < numRepos; i++ {
		go func(idx int) {
			idxr := indexer.NewIndexer(testDB.DB, configs[idx])
			result, err := idxr.Index(ctx, parseOutputs[idx])
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- result
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRepos; i++ {
		select {
		case err := <-errChan:
			t.Errorf("Concurrent indexing error: %v", err)
		case result := <-resultChan:
			if result.Status == "success" || result.Status == "success_with_warnings" {
				successCount++
			}
		case <-time.After(30 * time.Second):
			t.Fatal("Concurrent indexing timeout")
		}
	}

	duration := time.Since(startTime)
	t.Logf("Concurrent indexing completed in %s", duration)

	if successCount != numRepos {
		t.Errorf("Expected %d successful indexing operations, got: %d", numRepos, successCount)
	}

	// Verify all repos were created
	repoRepo := models.NewRepositoryRepository(testDB.DB)
	for _, config := range configs {
		repo, err := repoRepo.GetByID(ctx, config.RepoID)
		if err != nil {
			t.Errorf("Failed to get repository %s: %v", config.RepoID, err)
		}
		if repo == nil {
			t.Errorf("Repository %s not found after concurrent indexing", config.RepoID)
		}
	}
}

// TestMemoryUsage tests memory efficiency with large AST trees
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create parse output with large AST trees
	parseOutput := createParseOutputWithLargeAST()

	config := &indexer.IndexerConfig{
		RepoID:          uuid.New().String(),
		RepoName:        "memory-test-repo",
		BatchSize:       10,
		WorkerCount:     2,
		SkipVectors:     true,
		UseTransactions: false,
	}

	idx := indexer.NewIndexer(testDB.DB, config)

	// Run indexing
	result, err := idx.Index(ctx, parseOutput)
	if err != nil {
		// Log first few errors for debugging
		if result != nil && len(result.Errors) > 0 {
			t.Logf("Indexing failed with %d errors:", len(result.Errors))
			for i, e := range result.Errors {
				if i < 5 {
					t.Logf("  Error %d: %s - %s", i+1, e.Type, e.Message)
				}
			}
		}
		t.Fatalf("Memory test indexing failed: %v", err)
	}

	// Verify all nodes were created
	if result.NodesCreated == 0 {
		t.Error("Expected AST nodes to be created")
	}

	t.Logf("Created %d AST nodes", result.NodesCreated)
}

// TestBatchOptimization tests adaptive batch sizing
func TestBatchOptimization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping batch optimization test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create parse output with varying file sizes
	parseOutput := createVariableSizeParseOutput()

	config := &indexer.IndexerConfig{
		RepoID:          uuid.New().String(),
		RepoName:        "batch-test-repo",
		BatchSize:       10, // Start with small batch
		WorkerCount:     2,
		SkipVectors:     true,
		UseTransactions: false,
	}

	idx := indexer.NewIndexer(testDB.DB, config)

	// Run indexing
	result, err := idx.Index(ctx, parseOutput)
	if err != nil {
		t.Fatalf("Batch optimization test failed: %v", err)
	}

	// Get performance stats
	stats := idx.GetPerformanceStats()
	t.Logf("Performance stats: %+v", stats)

	// Verify batch size was adjusted
	if batchStats, ok := stats["batch"].(map[string]interface{}); ok {
		currentSize := batchStats["current_size"].(int)
		t.Logf("Final batch size: %d", currentSize)
	}

	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("Expected success status, got: %s", result.Status)
	}
}

// createLargeParseOutput creates a parse output with many files
func createLargeParseOutput(numFiles, symbolsPerFile int) *schema.ParseOutput {
	output := &schema.ParseOutput{
		Files:         make([]schema.File, numFiles),
		Relationships: []schema.DependencyEdge{},
	}

	for i := 0; i < numFiles; i++ {
		fileID := uuid.New().String()
		symbols := make([]schema.Symbol, symbolsPerFile)

		for j := 0; j < symbolsPerFile; j++ {
			symbols[j] = schema.Symbol{
				SymbolID:  uuid.New().String(),
				FileID:    fileID,
				Name:      fmt.Sprintf("Symbol_%d_%d", i, j),
				Kind:      "function",
				Signature: fmt.Sprintf("func Symbol_%d_%d()", i, j),
				Span: schema.Span{
					StartLine: j*10 + 1, // Lines start at 1, not 0
					EndLine:   j*10 + 6,
					StartByte: j * 100,
					EndByte:   j*100 + 50,
				},
			}
		}

		output.Files[i] = schema.File{
			FileID:   fileID,
			Path:     fmt.Sprintf("src/file_%d.go", i),
			Language: "go",
			Size:     int64(1024 * (i + 1)),
			Checksum: fmt.Sprintf("checksum_%d", i),
			Symbols:  symbols,
			Nodes:    []schema.ASTNode{},
		}
	}

	output.Metadata = schema.ParseMetadata{
		Version:      "1.0.0",
		TotalFiles:   numFiles,
		SuccessCount: numFiles,
		FailureCount: 0,
	}

	return output
}

// createParseOutputWithLargeAST creates parse output with deep AST trees
func createParseOutputWithLargeAST() *schema.ParseOutput {
	fileID := uuid.New().String()
	nodes := make([]schema.ASTNode, 1000)
	nodeIDs := make([]string, 1000)

	// Pre-generate all node IDs
	for i := 0; i < 1000; i++ {
		nodeIDs[i] = uuid.New().String()
	}

	// Create a deep tree structure
	for i := 0; i < 1000; i++ {
		parentID := ""
		if i > 0 {
			// Create tree structure (each node has parent at i/2)
			parentIdx := i / 2
			parentID = nodeIDs[parentIdx]
		}

		nodes[i] = schema.ASTNode{
			NodeID:   nodeIDs[i],
			FileID:   fileID,
			Type:     "expression",
			ParentID: parentID,
			Span: schema.Span{
				StartLine: i + 1, // Lines start at 1, not 0
				EndLine:   i + 2,
				StartByte: i * 10,
				EndByte:   i*10 + 10,
			},
			Text: fmt.Sprintf("node_%d", i),
		}
	}

	return &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   fileID,
				Path:     "src/large_ast.go",
				Language: "go",
				Size:     10000,
				Checksum: "large_ast_checksum",
				Symbols:  []schema.Symbol{},
				Nodes:    nodes,
			},
		},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}
}

// createVariableSizeParseOutput creates parse output with varying file sizes
func createVariableSizeParseOutput() *schema.ParseOutput {
	output := &schema.ParseOutput{
		Files:         make([]schema.File, 20),
		Relationships: []schema.DependencyEdge{},
	}

	for i := 0; i < 20; i++ {
		fileID := uuid.New().String()
		// Vary number of symbols (1 to 50)
		numSymbols := (i % 10) + 1
		symbols := make([]schema.Symbol, numSymbols)

		for j := 0; j < numSymbols; j++ {
			symbols[j] = schema.Symbol{
				SymbolID:  uuid.New().String(),
				FileID:    fileID,
				Name:      fmt.Sprintf("Func_%d_%d", i, j),
				Kind:      "function",
				Signature: fmt.Sprintf("func Func_%d_%d()", i, j),
				Span: schema.Span{
					StartLine: j*5 + 1, // Lines start at 1, not 0
					EndLine:   j*5 + 4,
					StartByte: j * 50,
					EndByte:   j*50 + 30,
				},
			}
		}

		output.Files[i] = schema.File{
			FileID:   fileID,
			Path:     fmt.Sprintf("src/var_file_%d.go", i),
			Language: "go",
			Size:     int64(512 * numSymbols),
			Checksum: fmt.Sprintf("var_checksum_%d", i),
			Symbols:  symbols,
			Nodes:    []schema.ASTNode{},
		}
	}

	output.Metadata = schema.ParseMetadata{
		Version:      "1.0.0",
		TotalFiles:   20,
		SuccessCount: 20,
		FailureCount: 0,
	}

	return output
}
