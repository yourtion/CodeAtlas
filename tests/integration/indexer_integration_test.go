package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestEndToEndIndexing tests the complete parse → index → query workflow
func TestEndToEndIndexing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create test parse output
	parseOutput := createTestParseOutput()

	// Create indexer config
	config := &indexer.IndexerConfig{
		RepoID:          uuid.New().String(),
		RepoName:        "test-repo",
		RepoURL:         "https://github.com/test/repo",
		Branch:          "main",
		BatchSize:       10,
		WorkerCount:     2,
		SkipVectors:     true, // Skip vectors for faster test
		Incremental:     false,
		UseTransactions: true,
		GraphName:       "test_graph",
	}

	// Create indexer
	idx := indexer.NewIndexer(testDB.DB, config)

	// Run indexing
	result, err := idx.Index(ctx, parseOutput)
	if err != nil {
		t.Fatalf("Indexing failed: %v", err)
	}

	// Verify result
	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("Expected success status, got: %s", result.Status)
	}

	if result.FilesProcessed != len(parseOutput.Files) {
		t.Errorf("Expected %d files processed, got: %d", len(parseOutput.Files), result.FilesProcessed)
	}

	// Count expected symbols
	expectedSymbols := 0
	for _, file := range parseOutput.Files {
		expectedSymbols += len(file.Symbols)
	}
	if result.SymbolsCreated != expectedSymbols {
		t.Errorf("Expected %d symbols created, got: %d", expectedSymbols, result.SymbolsCreated)
	}

	if result.EdgesCreated != len(parseOutput.Relationships) {
		t.Errorf("Expected %d edges created, got: %d", len(parseOutput.Relationships), result.EdgesCreated)
	}

	// Verify data in database
	t.Run("VerifyRepository", func(t *testing.T) {
		repoRepo := models.NewRepositoryRepository(testDB.DB)
		repo, err := repoRepo.GetByID(ctx, config.RepoID)
		if err != nil {
			t.Fatalf("Failed to get repository: %v", err)
		}
		if repo == nil {
			t.Fatal("Repository not found")
		}
		if repo.Name != config.RepoName {
			t.Errorf("Expected repo name %s, got: %s", config.RepoName, repo.Name)
		}
	})

	t.Run("VerifyFiles", func(t *testing.T) {
		fileRepo := models.NewFileRepository(testDB.DB)
		for _, file := range parseOutput.Files {
			dbFile, err := fileRepo.GetByID(ctx, file.FileID)
			if err != nil {
				t.Errorf("Failed to get file %s: %v", file.FileID, err)
				continue
			}
			if dbFile == nil {
				t.Errorf("File %s not found", file.FileID)
				continue
			}
			if dbFile.Path != file.Path {
				t.Errorf("Expected file path %s, got: %s", file.Path, dbFile.Path)
			}
		}
	})

	t.Run("VerifySymbols", func(t *testing.T) {
		symbolRepo := models.NewSymbolRepository(testDB.DB)
		for _, file := range parseOutput.Files {
			for _, symbol := range file.Symbols {
				dbSymbol, err := symbolRepo.GetByID(ctx, symbol.SymbolID)
				if err != nil {
					t.Errorf("Failed to get symbol %s: %v", symbol.SymbolID, err)
					continue
				}
				if dbSymbol == nil {
					t.Errorf("Symbol %s not found", symbol.SymbolID)
					continue
				}
				if dbSymbol.Name != symbol.Name {
					t.Errorf("Expected symbol name %s, got: %s", symbol.Name, dbSymbol.Name)
				}
			}
		}
	})

	t.Run("VerifyEdges", func(t *testing.T) {
		edgeRepo := models.NewEdgeRepository(testDB.DB)
		for _, edge := range parseOutput.Relationships {
			dbEdge, err := edgeRepo.GetByID(ctx, edge.EdgeID)
			if err != nil {
				t.Errorf("Failed to get edge %s: %v", edge.EdgeID, err)
				continue
			}
			if dbEdge == nil {
				t.Errorf("Edge %s not found", edge.EdgeID)
				continue
			}
			if dbEdge.EdgeType != string(edge.EdgeType) {
				t.Errorf("Expected edge type %s, got: %s", edge.EdgeType, dbEdge.EdgeType)
			}
		}
	})

	t.Run("VerifyReferentialIntegrity", func(t *testing.T) {
		if err := VerifyReferentialIntegrity(ctx, testDB.DB); err != nil {
			t.Errorf("Referential integrity check failed: %v", err)
		}
	})
}

// TestIncrementalIndexing tests incremental indexing with file modifications
func TestIncrementalIndexing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Initial indexing
	parseOutput := createTestParseOutput()
	config := &indexer.IndexerConfig{
		RepoID:          uuid.New().String(),
		RepoName:        "test-repo",
		BatchSize:       10,
		WorkerCount:     2,
		SkipVectors:     true,
		Incremental:     false,
		UseTransactions: true,
	}

	idx := indexer.NewIndexer(testDB.DB, config)
	result1, err := idx.Index(ctx, parseOutput)
	if err != nil {
		t.Fatalf("Initial indexing failed: %v", err)
	}

	initialFilesProcessed := result1.FilesProcessed

	// Modify one file (change checksum)
	parseOutput.Files[0].Checksum = "modified_checksum_123"
	parseOutput.Files[0].Symbols[0].Name = "ModifiedFunction"

	// Re-index with incremental flag
	config.Incremental = true
	idx = indexer.NewIndexer(testDB.DB, config)
	result2, err := idx.Index(ctx, parseOutput)
	if err != nil {
		t.Fatalf("Incremental indexing failed: %v", err)
	}

	// Should only process the modified file
	if result2.FilesProcessed >= initialFilesProcessed {
		t.Errorf("Expected fewer files processed in incremental mode, got: %d (initial: %d)",
			result2.FilesProcessed, initialFilesProcessed)
	}

	// Verify modified symbol
	symbolRepo := models.NewSymbolRepository(testDB.DB)
	dbSymbol, err := symbolRepo.GetByID(ctx, parseOutput.Files[0].Symbols[0].SymbolID)
	if err != nil {
		t.Fatalf("Failed to get modified symbol: %v", err)
	}
	if dbSymbol.Name != "ModifiedFunction" {
		t.Errorf("Expected modified symbol name 'ModifiedFunction', got: %s", dbSymbol.Name)
	}
}

// TestAPIEndpoints tests API handlers with sample data
func TestAPIEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Index test data first
	parseOutput := createTestParseOutput()
	config := &indexer.IndexerConfig{
		RepoID:          uuid.New().String(),
		RepoName:        "test-repo",
		BatchSize:       10,
		WorkerCount:     2,
		SkipVectors:     true,
		UseTransactions: true,
	}

	idx := indexer.NewIndexer(testDB.DB, config)
	_, err := idx.Index(ctx, parseOutput)
	if err != nil {
		t.Fatalf("Indexing failed: %v", err)
	}

	t.Run("GetRepository", func(t *testing.T) {
		repoRepo := models.NewRepositoryRepository(testDB.DB)
		repo, err := repoRepo.GetByID(ctx, config.RepoID)
		if err != nil {
			t.Fatalf("Failed to get repository: %v", err)
		}
		if repo == nil {
			t.Fatal("Repository not found")
		}
	})

	t.Run("GetFileSymbols", func(t *testing.T) {
		symbolRepo := models.NewSymbolRepository(testDB.DB)
		fileID := parseOutput.Files[0].FileID
		symbols, err := symbolRepo.GetByFileID(ctx, fileID)
		if err != nil {
			t.Fatalf("Failed to get file symbols: %v", err)
		}
		if len(symbols) == 0 {
			t.Error("Expected symbols for file, got none")
		}
	})

	t.Run("GetSymbolsByKind", func(t *testing.T) {
		symbolRepo := models.NewSymbolRepository(testDB.DB)
		fileID := parseOutput.Files[0].FileID
		symbols, err := symbolRepo.GetByKind(ctx, fileID, "function")
		if err != nil {
			t.Fatalf("Failed to get symbols by kind: %v", err)
		}
		for _, symbol := range symbols {
			if symbol.Kind != "function" {
				t.Errorf("Expected kind 'function', got: %s", symbol.Kind)
			}
		}
	})
}

// TestVectorSearch tests semantic search functionality
func TestVectorSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create test vectors
	vectorRepo := models.NewVectorRepository(testDB.DB)

	// Create test symbols first
	symbolRepo := models.NewSymbolRepository(testDB.DB)
	fileRepo := models.NewFileRepository(testDB.DB)
	repoRepo := models.NewRepositoryRepository(testDB.DB)

	// Create repository
	repo := &models.Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo",
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

	// Create symbols with embeddings
	symbols := []*models.Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "TestFunction",
			Kind:      "function",
			Signature: "func TestFunction()",
			StartLine: 1,
			EndLine:   10,
			Docstring: "This is a test function",
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "AnotherFunction",
			Kind:      "function",
			Signature: "func AnotherFunction()",
			StartLine: 11,
			EndLine:   20,
			Docstring: "This is another function",
		},
	}

	for _, symbol := range symbols {
		if err := symbolRepo.Create(ctx, symbol); err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}

		// Create vector embedding (random for test)
		embedding := make([]float32, 768)
		for i := range embedding {
			embedding[i] = float32(i) / 768.0
		}

		vector := &models.Vector{
			EntityID:   symbol.SymbolID,
			EntityType: "symbol",
			Embedding:  embedding,
			Content:    symbol.Docstring,
			Model:      "test-model",
		}
		if err := vectorRepo.Create(ctx, vector); err != nil {
			t.Fatalf("Failed to create vector: %v", err)
		}
	}

	// Test similarity search
	t.Run("SimilaritySearch", func(t *testing.T) {
		queryEmbedding := make([]float32, 768)
		for i := range queryEmbedding {
			queryEmbedding[i] = float32(i) / 768.0
		}

		results, err := vectorRepo.SimilaritySearch(ctx, queryEmbedding, "symbol", 10)
		if err != nil {
			t.Fatalf("Similarity search failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected search results, got none")
		}

		// Verify results have similarity scores
		for _, result := range results {
			if result.Similarity < 0 || result.Similarity > 1 {
				t.Errorf("Invalid similarity score: %f", result.Similarity)
			}
		}
	})
}

// TestRelationshipQueries tests callers, callees, and dependencies
func TestRelationshipQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create test data with relationships
	parseOutput := createTestParseOutputWithRelationships()
	config := &indexer.IndexerConfig{
		RepoID:          uuid.New().String(),
		RepoName:        "test-repo",
		BatchSize:       10,
		WorkerCount:     2,
		SkipVectors:     true,
		UseTransactions: true,
	}

	idx := indexer.NewIndexer(testDB.DB, config)
	_, err := idx.Index(ctx, parseOutput)
	if err != nil {
		t.Fatalf("Indexing failed: %v", err)
	}

	edgeRepo := models.NewEdgeRepository(testDB.DB)

	t.Run("GetCallees", func(t *testing.T) {
		// Get the first symbol (caller)
		sourceSymbolID := parseOutput.Files[0].Symbols[0].SymbolID

		// Find edges where this symbol is the source
		edges, err := edgeRepo.GetBySourceID(ctx, sourceSymbolID)
		if err != nil {
			t.Fatalf("Failed to get callees: %v", err)
		}

		if len(edges) == 0 {
			t.Error("Expected callees, got none")
		}

		// Verify edge types
		for _, edge := range edges {
			if edge.EdgeType != "calls" && edge.EdgeType != "imports" {
				t.Errorf("Unexpected edge type: %s", edge.EdgeType)
			}
		}
	})

	t.Run("GetCallers", func(t *testing.T) {
		// Get the second symbol (callee)
		targetSymbolID := parseOutput.Files[0].Symbols[1].SymbolID

		// Find edges where this symbol is the target
		edges, err := edgeRepo.GetByTargetID(ctx, targetSymbolID)
		if err != nil {
			t.Fatalf("Failed to get callers: %v", err)
		}

		// May or may not have callers depending on test data
		for _, edge := range edges {
			if edge.TargetID == nil || *edge.TargetID != targetSymbolID {
				t.Error("Edge target ID mismatch")
			}
		}
	})

	t.Run("GetEdgesByType", func(t *testing.T) {
		edges, err := edgeRepo.GetByType(ctx, "calls")
		if err != nil {
			t.Fatalf("Failed to get edges by type: %v", err)
		}

		for _, edge := range edges {
			if edge.EdgeType != "calls" {
				t.Errorf("Expected edge type 'calls', got: %s", edge.EdgeType)
			}
		}
	})
}

// createTestParseOutput creates a sample parse output for testing
func createTestParseOutput() *schema.ParseOutput {
	fileID1 := uuid.New().String()
	fileID2 := uuid.New().String()
	symbolID1 := uuid.New().String()
	symbolID2 := uuid.New().String()
	symbolID3 := uuid.New().String()

	return &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   fileID1,
				Path:     "src/main.go",
				Language: "go",
				Size:     1024,
				Checksum: "abc123",
				Symbols: []schema.Symbol{
					{
						SymbolID:  symbolID1,
						FileID:    fileID1,
						Name:      "main",
						Kind:      "function",
						Signature: "func main()",
						Span: schema.Span{
							StartLine: 10,
							EndLine:   20,
							StartByte: 100,
							EndByte:   200,
						},
						Docstring: "Main function",
					},
					{
						SymbolID:  symbolID2,
						FileID:    fileID1,
						Name:      "helper",
						Kind:      "function",
						Signature: "func helper() string",
						Span: schema.Span{
							StartLine: 22,
							EndLine:   30,
							StartByte: 210,
							EndByte:   300,
						},
						Docstring: "Helper function",
					},
				},
				Nodes: []schema.ASTNode{
					{
						NodeID:   uuid.New().String(),
						FileID:   fileID1,
						Type:     "function_declaration",
						ParentID: "",
						Span: schema.Span{
							StartLine: 10,
							EndLine:   20,
							StartByte: 100,
							EndByte:   200,
						},
						Text: "func main() { ... }",
					},
				},
			},
			{
				FileID:   fileID2,
				Path:     "src/utils.go",
				Language: "go",
				Size:     512,
				Checksum: "def456",
				Symbols: []schema.Symbol{
					{
						SymbolID:  symbolID3,
						FileID:    fileID2,
						Name:      "Utility",
						Kind:      "function",
						Signature: "func Utility() error",
						Span: schema.Span{
							StartLine: 5,
							EndLine:   15,
							StartByte: 50,
							EndByte:   150,
						},
						Docstring: "Utility function",
					},
				},
				Nodes: []schema.ASTNode{},
			},
		},
		Relationships: []schema.DependencyEdge{
			{
				EdgeID:       uuid.New().String(),
				SourceID:     symbolID1,
				TargetID:     symbolID2,
				EdgeType:     "call",
				SourceFile:   "src/main.go",
				TargetFile:   "src/main.go",
				TargetModule: "",
			},
		},
	}
}

// createTestParseOutputWithRelationships creates parse output with more relationships
func createTestParseOutputWithRelationships() *schema.ParseOutput {
	output := createTestParseOutput()

	// Add more relationships
	symbolID1 := output.Files[0].Symbols[0].SymbolID
	symbolID3 := output.Files[1].Symbols[0].SymbolID

	output.Relationships = append(output.Relationships, schema.DependencyEdge{
		EdgeID:       uuid.New().String(),
		SourceID:     symbolID1,
		TargetID:     symbolID3,
		EdgeType:     "call",
		SourceFile:   "src/main.go",
		TargetFile:   "src/utils.go",
		TargetModule: "",
	})

	return output
}
