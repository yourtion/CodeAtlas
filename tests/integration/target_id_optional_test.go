package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestTargetIDOptional tests that edges can have optional target_id for external imports
func TestTargetIDOptional(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)
	db := testDB.DB

	ctx := context.Background()
	repoID := uuid.New().String()

	// Create test parse output with external import (no target_id)
	fileID := uuid.New().String()
	symbolID := uuid.New().String()
	edgeID := uuid.New().String()

	parseOutput := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   fileID,
				Path:     "src/main.js",
				Language: "javascript",
				Size:     100,
				Checksum: "abc123",
				Symbols: []schema.Symbol{
					{
						SymbolID: symbolID,
						FileID:   fileID,
						Name:     "main",
						Kind:     schema.SymbolModule,
						Span: schema.Span{
							StartLine: 1,
							EndLine:   1,
							StartByte: 0,
							EndByte:   0,
						},
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{
			{
				EdgeID:       edgeID,
				SourceID:     symbolID,
				TargetID:     "", // Empty target_id for external import
				EdgeType:     schema.EdgeImport,
				SourceFile:   "src/main.js",
				TargetModule: "lodash",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	// Create indexer
	config := &indexer.IndexerConfig{
		RepoID:          repoID,
		RepoName:        "test-repo",
		BatchSize:       100,
		WorkerCount:     1,
		UseTransactions: true,
	}
	idx := indexer.NewIndexer(db, config)

	// Index the parse output
	result, err := idx.Index(ctx, parseOutput)
	if err != nil {
		t.Fatalf("Failed to index: %v", err)
	}

	// Verify indexing succeeded (allow warnings for graph operations)
	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("Expected status 'success' or 'success_with_warnings', got '%s'", result.Status)
		if len(result.Errors) > 0 {
			for _, e := range result.Errors {
				t.Logf("Error: %s - %s", e.Type, e.Message)
			}
		}
	}

	// Verify edge was created in database
	edgeRepo := models.NewEdgeRepository(db)
	edges, err := edgeRepo.GetBySourceID(ctx, symbolID)
	if err != nil {
		t.Fatalf("Failed to get edges: %v", err)
	}

	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(edges))
	}

	edge := edges[0]
	if edge.TargetID != nil {
		t.Errorf("Expected target_id to be nil, got %v", *edge.TargetID)
	}

	if edge.TargetModule == nil || *edge.TargetModule != "lodash" {
		t.Errorf("Expected target_module 'lodash', got %v", edge.TargetModule)
	}

	if edge.EdgeType != "import" {
		t.Errorf("Expected edge_type 'import', got '%s'", edge.EdgeType)
	}

	t.Logf("✅ Successfully created edge with optional target_id")
	t.Logf("   Edge ID: %s", edge.EdgeID)
	t.Logf("   Source ID: %s", edge.SourceID)
	t.Logf("   Target ID: nil")
	t.Logf("   Target Module: %s", *edge.TargetModule)
}

// TestTargetIDRequired tests that non-import edges with empty target_id are accepted
// as dangling edges (Task 3: validator 放宽——非 import 边空 target_id 不再阻塞写入，
// 悬空边是合法状态，供 symbol_resolution_rate 指标观测）。
func TestTargetIDRequired(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)
	db := testDB.DB

	ctx := context.Background()
	repoID := uuid.New().String()

	// Create test parse output with call edge without target_id（悬空 call 边，合法）
	fileID := uuid.New().String()
	symbolID := uuid.New().String()
	edgeID := uuid.New().String()

	parseOutput := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   fileID,
				Path:     "src/main.js",
				Language: "javascript",
				Size:     100,
				Checksum: "abc123",
				Symbols: []schema.Symbol{
					{
						SymbolID: symbolID,
						FileID:   fileID,
						Name:     "main",
						Kind:     schema.SymbolFunction,
						Span: schema.Span{
							StartLine: 1,
							EndLine:   10,
							StartByte: 0,
							EndByte:   100,
						},
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{
			{
				EdgeID:     edgeID,
				SourceID:   symbolID,
				TargetID:   "", // Empty target_id：悬空 call 边（如调用未索引的标准库函数）
				EdgeType:   schema.EdgeCall,
				SourceFile: "src/main.js",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	// Create indexer
	config := &indexer.IndexerConfig{
		RepoID:          repoID,
		RepoName:        "test-repo",
		BatchSize:       100,
		WorkerCount:     1,
		UseTransactions: true,
	}
	idx := indexer.NewIndexer(db, config)

	// 悬空 call 边应被接受（不再阻塞写入）。
	result, err := idx.Index(ctx, parseOutput)
	if err != nil {
		t.Fatalf("悬空 call 边应被接受，但索引失败: %v", err)
	}
	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("Expected status 'success' or 'success_with_warnings', got '%s'", result.Status)
	}

	// 验证悬空边已入库（target_id 为 NULL）。
	edgeRepo := models.NewEdgeRepository(db)
	edges, err := edgeRepo.GetBySourceID(ctx, symbolID)
	if err != nil {
		t.Fatalf("Failed to get edges: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(edges))
	}
	if edges[0].TargetID != nil {
		t.Errorf("悬空边 target_id 应为 nil，got %v", *edges[0].TargetID)
	}

	t.Logf("✅ 正确接受悬空 call 边（target_id 空），供 symbol_resolution_rate 指标观测")
}

// TestMixedEdges tests indexing with both internal and external dependencies
func TestMixedEdges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)
	db := testDB.DB

	ctx := context.Background()
	repoID := uuid.New().String()

	// Create test parse output with mixed edges
	fileID := uuid.New().String()
	symbolID1 := uuid.New().String()
	symbolID2 := uuid.New().String()
	edgeID1 := uuid.New().String()
	edgeID2 := uuid.New().String()

	parseOutput := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   fileID,
				Path:     "src/main.js",
				Language: "javascript",
				Size:     200,
				Checksum: "abc123",
				Symbols: []schema.Symbol{
					{
						SymbolID: symbolID1,
						FileID:   fileID,
						Name:     "main",
						Kind:     schema.SymbolFunction,
						Span: schema.Span{
							StartLine: 1,
							EndLine:   10,
							StartByte: 0,
							EndByte:   100,
						},
					},
					{
						SymbolID: symbolID2,
						FileID:   fileID,
						Name:     "helper",
						Kind:     schema.SymbolFunction,
						Span: schema.Span{
							StartLine: 12,
							EndLine:   20,
							StartByte: 102,
							EndByte:   200,
						},
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{
			{
				EdgeID:       edgeID1,
				SourceID:     symbolID1,
				TargetID:     "", // External import
				EdgeType:     schema.EdgeImport,
				SourceFile:   "src/main.js",
				TargetModule: "lodash",
			},
			{
				EdgeID:     edgeID2,
				SourceID:   symbolID1,
				TargetID:   symbolID2, // Internal call
				EdgeType:   schema.EdgeCall,
				SourceFile: "src/main.js",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	// Create indexer
	config := &indexer.IndexerConfig{
		RepoID:          repoID,
		RepoName:        "test-repo",
		BatchSize:       100,
		WorkerCount:     1,
		UseTransactions: true,
	}
	idx := indexer.NewIndexer(db, config)

	// Index the parse output
	result, err := idx.Index(ctx, parseOutput)
	if err != nil {
		t.Fatalf("Failed to index: %v", err)
	}

	if result.Status != "success" && result.Status != "success_with_warnings" {
		t.Errorf("Expected status 'success' or 'success_with_warnings', got '%s'", result.Status)
	}

	// Verify both edges were created
	edgeRepo := models.NewEdgeRepository(db)
	edges, err := edgeRepo.GetBySourceID(ctx, symbolID1)
	if err != nil {
		t.Fatalf("Failed to get edges: %v", err)
	}

	if len(edges) != 2 {
		t.Fatalf("Expected 2 edges, got %d", len(edges))
	}

	// Check external import edge
	var externalEdge, internalEdge *models.Edge
	for _, e := range edges {
		if e.EdgeType == "import" {
			externalEdge = e
		} else if e.EdgeType == "call" {
			internalEdge = e
		}
	}

	if externalEdge == nil {
		t.Fatal("External import edge not found")
	}
	if externalEdge.TargetID != nil {
		t.Errorf("External edge should have nil target_id, got %v", *externalEdge.TargetID)
	}
	if externalEdge.TargetModule == nil || *externalEdge.TargetModule != "lodash" {
		t.Errorf("External edge should have target_module 'lodash', got %v", externalEdge.TargetModule)
	}

	// Check internal call edge
	if internalEdge == nil {
		t.Fatal("Internal call edge not found")
	}
	if internalEdge.TargetID == nil || *internalEdge.TargetID != symbolID2 {
		t.Errorf("Internal edge should have target_id '%s', got %v", symbolID2, internalEdge.TargetID)
	}

	t.Logf("✅ Successfully indexed mixed internal and external edges")
	t.Logf("   External import: %s -> (lodash)", externalEdge.SourceID)
	t.Logf("   Internal call: %s -> %s", internalEdge.SourceID, *internalEdge.TargetID)
}
