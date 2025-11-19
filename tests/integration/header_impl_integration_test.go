package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestHeaderImplAssociation tests the header-implementation association workflow
func TestHeaderImplAssociation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create test parse output with C header and implementation files
	parseOutput := createHeaderImplParseOutput()

	// Create indexer config
	config := &indexer.IndexerConfig{
		RepoID:          uuid.New().String(),
		RepoName:        "test-header-impl-repo",
		RepoURL:         "https://github.com/test/header-impl",
		Branch:          "main",
		BatchSize:       10,
		WorkerCount:     2,
		SkipVectors:     true, // Skip vectors for faster test
		Incremental:     false,
		UseTransactions: true,
		GraphName:       "test_header_impl_graph",
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

	// Verify header-impl association results
	if result.Summary["header_impl_pairs"] == nil {
		t.Error("Expected header_impl_pairs in summary")
	} else {
		pairs := result.Summary["header_impl_pairs"].(int)
		if pairs != 1 {
			t.Errorf("Expected 1 header-impl pair, got: %d", pairs)
		}
	}

	if result.Summary["header_impl_edges"] == nil {
		t.Error("Expected header_impl_edges in summary")
	} else {
		edges := result.Summary["header_impl_edges"].(int)
		// Should have 1 file-level edge + 1 symbol-level edge
		if edges != 2 {
			t.Errorf("Expected 2 header-impl edges, got: %d", edges)
		}
	}

	// Verify edges in database
	t.Run("VerifyImplementsHeaderEdge", func(t *testing.T) {
		// Query for implements_header edges
		query := `SELECT * FROM edges WHERE edge_type = 'implements_header'`
		rows, err := testDB.DB.QueryContext(ctx, query)
		if err != nil {
			t.Fatalf("Failed to query edges: %v", err)
		}
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
			var edge models.Edge
			var targetID, targetFile *string
			err := rows.Scan(
				&edge.EdgeID,
				&edge.SourceID,
				&targetID,
				&edge.EdgeType,
				&edge.SourceFile,
				&targetFile,
				&edge.TargetModule,
				&edge.CreatedAt,
			)
			if err != nil {
				t.Errorf("Failed to scan edge: %v", err)
				continue
			}

			// Verify edge properties
			if edge.EdgeType != "implements_header" {
				t.Errorf("Expected edge type 'implements_header', got: %s", edge.EdgeType)
			}
			if edge.SourceFile != "test.c" {
				t.Errorf("Expected source file 'test.c', got: %s", edge.SourceFile)
			}
			if targetFile == nil || *targetFile != "test.h" {
				t.Errorf("Expected target file 'test.h', got: %v", targetFile)
			}
		}

		if count != 1 {
			t.Errorf("Expected 1 implements_header edge, got: %d", count)
		}
	})

	t.Run("VerifyImplementsDeclarationEdge", func(t *testing.T) {
		// Query for implements_declaration edges
		query := `SELECT * FROM edges WHERE edge_type = 'implements_declaration'`
		rows, err := testDB.DB.QueryContext(ctx, query)
		if err != nil {
			t.Fatalf("Failed to query edges: %v", err)
		}
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
			var edge models.Edge
			var targetID, targetFile *string
			err := rows.Scan(
				&edge.EdgeID,
				&edge.SourceID,
				&targetID,
				&edge.EdgeType,
				&edge.SourceFile,
				&targetFile,
				&edge.TargetModule,
				&edge.CreatedAt,
			)
			if err != nil {
				t.Errorf("Failed to scan edge: %v", err)
				continue
			}

			// Verify edge properties
			if edge.EdgeType != "implements_declaration" {
				t.Errorf("Expected edge type 'implements_declaration', got: %s", edge.EdgeType)
			}
			if edge.SourceFile != "test.c" {
				t.Errorf("Expected source file 'test.c', got: %s", edge.SourceFile)
			}
			if targetFile == nil || *targetFile != "test.h" {
				t.Errorf("Expected target file 'test.h', got: %v", targetFile)
			}
			
			// Verify source and target IDs are set
			if edge.SourceID == "" {
				t.Error("Expected source ID to be set")
			}
			if targetID == nil || *targetID == "" {
				t.Error("Expected target ID to be set")
			}
		}

		if count != 1 {
			t.Errorf("Expected 1 implements_declaration edge, got: %d", count)
		}
	})
}

// createHeaderImplParseOutput creates a test parse output with C header and implementation files
func createHeaderImplParseOutput() *schema.ParseOutput {
	headerFileID := uuid.New().String()
	implFileID := uuid.New().String()
	headerSymbolID := uuid.New().String()
	implSymbolID := uuid.New().String()

	return &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   headerFileID,
				Path:     "test.h",
				Language: "c",
				Size:     100,
				Checksum: "header-checksum",
				Symbols: []schema.Symbol{
					{
						SymbolID:  headerSymbolID,
						FileID:    headerFileID,
						Name:      "myFunction",
						Kind:      "function_declaration",
						Signature: "int myFunction(int x)",
						Span: schema.Span{
							StartLine: 1,
							EndLine:   1,
							StartByte: 0,
							EndByte:   25,
						},
						Docstring: "Function declaration in header",
					},
				},
			},
			{
				FileID:   implFileID,
				Path:     "test.c",
				Language: "c",
				Size:     200,
				Checksum: "impl-checksum",
				Symbols: []schema.Symbol{
					{
						SymbolID:  implSymbolID,
						FileID:    implFileID,
						Name:      "myFunction",
						Kind:      schema.SymbolFunction,
						Signature: "int myFunction(int x)",
						Span: schema.Span{
							StartLine: 5,
							EndLine:   10,
							StartByte: 100,
							EndByte:   200,
						},
						Docstring: "Function implementation",
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			TotalFiles:   2,
			SuccessCount: 2,
			FailureCount: 0,
		},
	}
}

// TestMultipleHeaderImplPairs tests multiple header-implementation pairs
func TestMultipleHeaderImplPairs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create test parse output with multiple C++ header and implementation files
	parseOutput := createMultipleHeaderImplParseOutput()

	// Create indexer config
	config := &indexer.IndexerConfig{
		RepoID:          uuid.New().String(),
		RepoName:        "test-multiple-header-impl",
		RepoURL:         "https://github.com/test/multiple-header-impl",
		Branch:          "main",
		BatchSize:       10,
		WorkerCount:     2,
		SkipVectors:     true,
		Incremental:     false,
		UseTransactions: true,
		GraphName:       "test_multiple_header_impl_graph",
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

	// Verify we found 2 pairs
	if result.Summary["header_impl_pairs"] == nil {
		t.Error("Expected header_impl_pairs in summary")
	} else {
		pairs := result.Summary["header_impl_pairs"].(int)
		if pairs != 2 {
			t.Errorf("Expected 2 header-impl pairs, got: %d", pairs)
		}
	}

	// Verify we created 4 edges (2 file-level + 2 symbol-level)
	if result.Summary["header_impl_edges"] == nil {
		t.Error("Expected header_impl_edges in summary")
	} else {
		edges := result.Summary["header_impl_edges"].(int)
		if edges != 4 {
			t.Errorf("Expected 4 header-impl edges, got: %d", edges)
		}
	}
}

// createMultipleHeaderImplParseOutput creates a test parse output with multiple header-impl pairs
func createMultipleHeaderImplParseOutput() *schema.ParseOutput {
	return &schema.ParseOutput{
		Files: []schema.File{
			// First pair: foo.hpp / foo.cpp
			{
				FileID:   uuid.New().String(),
				Path:     "foo.hpp",
				Language: "cpp",
				Size:     100,
				Checksum: "foo-header-checksum",
				Symbols: []schema.Symbol{
					{
						SymbolID:  uuid.New().String(),
						FileID:    uuid.New().String(),
						Name:      "fooFunction",
						Kind:      "function_declaration",
						Signature: "void fooFunction()",
						Span: schema.Span{
							StartLine: 1,
							EndLine:   1,
						},
					},
				},
			},
			{
				FileID:   uuid.New().String(),
				Path:     "foo.cpp",
				Language: "cpp",
				Size:     200,
				Checksum: "foo-impl-checksum",
				Symbols: []schema.Symbol{
					{
						SymbolID:  uuid.New().String(),
						FileID:    uuid.New().String(),
						Name:      "fooFunction",
						Kind:      schema.SymbolFunction,
						Signature: "void fooFunction()",
						Span: schema.Span{
							StartLine: 5,
							EndLine:   10,
						},
					},
				},
			},
			// Second pair: bar.h / bar.m (Objective-C)
			{
				FileID:   uuid.New().String(),
				Path:     "bar.h",
				Language: "objc",
				Size:     150,
				Checksum: "bar-header-checksum",
				Symbols: []schema.Symbol{
					{
						SymbolID:  uuid.New().String(),
						FileID:    uuid.New().String(),
						Name:      "barMethod",
						Kind:      "method_declaration",
						Signature: "- (void)barMethod",
						Span: schema.Span{
							StartLine: 1,
							EndLine:   1,
						},
					},
				},
			},
			{
				FileID:   uuid.New().String(),
				Path:     "bar.m",
				Language: "objc",
				Size:     250,
				Checksum: "bar-impl-checksum",
				Symbols: []schema.Symbol{
					{
						SymbolID:  uuid.New().String(),
						FileID:    uuid.New().String(),
						Name:      "barMethod",
						Kind:      "method",
						Signature: "- (void)barMethod",
						Span: schema.Span{
							StartLine: 5,
							EndLine:   10,
						},
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			TotalFiles:   4,
			SuccessCount: 4,
			FailureCount: 0,
		},
	}
}
