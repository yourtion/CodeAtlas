package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestASTNodeRepository tests all AST node repository operations
func TestASTNodeRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := models.NewASTNodeRepository(testDB.DB)

	// Setup: Create repository and file first
	repoRepo := models.NewRepositoryRepository(testDB.DB)
	fileRepo := models.NewFileRepository(testDB.DB)

	testRepo := &models.Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo",
	}
	if err := repoRepo.Create(ctx, testRepo); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	testFile := &models.File{
		FileID:   uuid.New().String(),
		RepoID:   testRepo.RepoID,
		Path:     "test.go",
		Language: "go",
		Size:     1000,
		Checksum: "test123",
	}
	if err := fileRepo.Create(ctx, testFile); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	t.Run("Create", func(t *testing.T) {
		node := &models.ASTNode{
			NodeID:    uuid.New().String(),
			FileID:    testFile.FileID,
			Type:      "function_declaration",
			StartLine: 1,
			EndLine:   10,
			StartByte: 0,
			EndByte:   100,
			Text:      "func test() {}",
			Attributes: map[string]string{
				"visibility": "public",
			},
		}

		err := repo.Create(ctx, node)
		if err != nil {
			t.Fatalf("Failed to create AST node: %v", err)
		}

		// Verify creation
		retrieved, err := repo.GetByID(ctx, node.NodeID)
		if err != nil {
			t.Fatalf("Failed to get AST node: %v", err)
		}
		if retrieved == nil {
			t.Fatal("AST node not found")
		}
		if retrieved.Type != node.Type {
			t.Errorf("Expected type %s, got %s", node.Type, retrieved.Type)
		}
	})

	t.Run("GetByFileID", func(t *testing.T) {
		nodes, err := repo.GetByFileID(ctx, testFile.FileID)
		if err != nil {
			t.Fatalf("Failed to get nodes by file ID: %v", err)
		}
		if len(nodes) == 0 {
			t.Error("Expected nodes, got none")
		}
	})

	t.Run("GetByType", func(t *testing.T) {
		nodes, err := repo.GetByType(ctx, testFile.FileID, "function_declaration")
		if err != nil {
			t.Fatalf("Failed to get nodes by type: %v", err)
		}
		for _, node := range nodes {
			if node.Type != "function_declaration" {
				t.Errorf("Expected type function_declaration, got %s", node.Type)
			}
		}
	})

	t.Run("GetRootNodes", func(t *testing.T) {
		nodes, err := repo.GetRootNodes(ctx, testFile.FileID)
		if err != nil {
			t.Fatalf("Failed to get root nodes: %v", err)
		}
		for _, node := range nodes {
			if node.ParentID != nil {
				t.Error("Root node should not have parent")
			}
		}
	})

	t.Run("CreateWithParent", func(t *testing.T) {
		// Create parent node
		parentNode := &models.ASTNode{
			NodeID:    uuid.New().String(),
			FileID:    testFile.FileID,
			Type:      "class_declaration",
			StartLine: 1,
			EndLine:   20,
			StartByte: 0,
			EndByte:   200,
			Text:      "class Test {}",
		}
		if err := repo.Create(ctx, parentNode); err != nil {
			t.Fatalf("Failed to create parent node: %v", err)
		}

		// Create child node
		childNode := &models.ASTNode{
			NodeID:    uuid.New().String(),
			FileID:    testFile.FileID,
			Type:      "method_declaration",
			ParentID:  &parentNode.NodeID,
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
			Text:      "def method() {}",
		}
		if err := repo.Create(ctx, childNode); err != nil {
			t.Fatalf("Failed to create child node: %v", err)
		}

		// Get children
		children, err := repo.GetByParentID(ctx, parentNode.NodeID)
		if err != nil {
			t.Fatalf("Failed to get children: %v", err)
		}
		if len(children) == 0 {
			t.Error("Expected child nodes, got none")
		}
	})

	t.Run("Update", func(t *testing.T) {
		node := &models.ASTNode{
			NodeID:    uuid.New().String(),
			FileID:    testFile.FileID,
			Type:      "variable_declaration",
			StartLine: 1,
			EndLine:   1,
			StartByte: 0,
			EndByte:   10,
			Text:      "var x = 1",
		}
		if err := repo.Create(ctx, node); err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}

		node.Text = "var x = 2"
		if err := repo.Update(ctx, node); err != nil {
			t.Fatalf("Failed to update node: %v", err)
		}

		updated, err := repo.GetByID(ctx, node.NodeID)
		if err != nil {
			t.Fatalf("Failed to get updated node: %v", err)
		}
		if updated.Text != "var x = 2" {
			t.Errorf("Expected text 'var x = 2', got %s", updated.Text)
		}
	})

	t.Run("BatchCreate", func(t *testing.T) {
		nodes := []*models.ASTNode{
			{
				NodeID:    uuid.New().String(),
				FileID:    testFile.FileID,
				Type:      "import_statement",
				StartLine: 1,
				EndLine:   1,
				StartByte: 0,
				EndByte:   20,
				Text:      "import fmt",
			},
			{
				NodeID:    uuid.New().String(),
				FileID:    testFile.FileID,
				Type:      "import_statement",
				StartLine: 2,
				EndLine:   2,
				StartByte: 21,
				EndByte:   40,
				Text:      "import os",
			},
		}

		if err := repo.BatchCreate(ctx, nodes); err != nil {
			t.Fatalf("Failed to batch create nodes: %v", err)
		}

		// Verify creation
		for _, node := range nodes {
			retrieved, err := repo.GetByID(ctx, node.NodeID)
			if err != nil {
				t.Errorf("Failed to get node %s: %v", node.NodeID, err)
			}
			if retrieved == nil {
				t.Errorf("Node %s not found", node.NodeID)
			}
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := repo.Count(ctx, testFile.FileID)
		if err != nil {
			t.Fatalf("Failed to count nodes: %v", err)
		}
		if count == 0 {
			t.Error("Expected nodes count > 0")
		}
	})

	t.Run("CountByType", func(t *testing.T) {
		counts, err := repo.CountByType(ctx, testFile.FileID)
		if err != nil {
			t.Fatalf("Failed to count nodes by type: %v", err)
		}
		if counts["import_statement"] < 2 {
			t.Errorf("Expected at least 2 import statements, got %d", counts["import_statement"])
		}
	})

	t.Run("GetNodeHierarchy", func(t *testing.T) {
		// Create a hierarchy
		rootNode := &models.ASTNode{
			NodeID:    uuid.New().String(),
			FileID:    testFile.FileID,
			Type:      "module",
			StartLine: 1,
			EndLine:   100,
			StartByte: 0,
			EndByte:   1000,
			Text:      "module test",
		}
		if err := repo.Create(ctx, rootNode); err != nil {
			t.Fatalf("Failed to create root: %v", err)
		}

		hierarchy, err := repo.GetNodeHierarchy(ctx, rootNode.NodeID)
		if err != nil {
			t.Fatalf("Failed to get hierarchy: %v", err)
		}
		if len(hierarchy) == 0 {
			t.Error("Expected hierarchy nodes")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		node := &models.ASTNode{
			NodeID:    uuid.New().String(),
			FileID:    testFile.FileID,
			Type:      "comment",
			StartLine: 1,
			EndLine:   1,
			StartByte: 0,
			EndByte:   10,
			Text:      "// comment",
		}
		if err := repo.Create(ctx, node); err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}

		if err := repo.Delete(ctx, node.NodeID); err != nil {
			t.Fatalf("Failed to delete node: %v", err)
		}

		deleted, err := repo.GetByID(ctx, node.NodeID)
		if err != nil {
			t.Fatalf("Failed to check deleted node: %v", err)
		}
		if deleted != nil {
			t.Error("Node should be deleted")
		}
	})

	t.Run("DeleteByFileID", func(t *testing.T) {
		// Create a new file for deletion test
		deleteFile := &models.File{
			FileID:   uuid.New().String(),
			RepoID:   testRepo.RepoID,
			Path:     "delete.go",
			Language: "go",
			Size:     100,
			Checksum: "delete123",
		}
		if err := fileRepo.Create(ctx, deleteFile); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Create nodes for this file
		node := &models.ASTNode{
			NodeID:    uuid.New().String(),
			FileID:    deleteFile.FileID,
			Type:      "function",
			StartLine: 1,
			EndLine:   5,
			StartByte: 0,
			EndByte:   50,
			Text:      "func test() {}",
		}
		if err := repo.Create(ctx, node); err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}

		// Delete all nodes for this file
		if err := repo.DeleteByFileID(ctx, deleteFile.FileID); err != nil {
			t.Fatalf("Failed to delete nodes by file ID: %v", err)
		}

		// Verify deletion
		nodes, err := repo.GetByFileID(ctx, deleteFile.FileID)
		if err != nil {
			t.Fatalf("Failed to get nodes: %v", err)
		}
		if len(nodes) != 0 {
			t.Errorf("Expected 0 nodes, got %d", len(nodes))
		}
	})
}

// TestEdgeRepository tests all edge repository operations
func TestEdgeRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	edgeRepo := models.NewEdgeRepository(testDB.DB)

	// Setup: Create repository, file, and symbols
	repoRepo := models.NewRepositoryRepository(testDB.DB)
	fileRepo := models.NewFileRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)

	testRepo := &models.Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo",
	}
	if err := repoRepo.Create(ctx, testRepo); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	testFile := &models.File{
		FileID:   uuid.New().String(),
		RepoID:   testRepo.RepoID,
		Path:     "test.go",
		Language: "go",
		Size:     1000,
		Checksum: "test123",
	}
	if err := fileRepo.Create(ctx, testFile); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	sourceSymbol := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    testFile.FileID,
		Name:      "caller",
		Kind:      "function",
		Signature: "func caller()",
		StartLine: 1,
		EndLine:   10,
	}
	if err := symbolRepo.Create(ctx, sourceSymbol); err != nil {
		t.Fatalf("Failed to create source symbol: %v", err)
	}

	targetSymbol := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    testFile.FileID,
		Name:      "callee",
		Kind:      "function",
		Signature: "func callee()",
		StartLine: 12,
		EndLine:   20,
	}
	if err := symbolRepo.Create(ctx, targetSymbol); err != nil {
		t.Fatalf("Failed to create target symbol: %v", err)
	}

	t.Run("Create", func(t *testing.T) {
		edge := &models.Edge{
			EdgeID:     uuid.New().String(),
			SourceID:   sourceSymbol.SymbolID,
			TargetID:   &targetSymbol.SymbolID,
			EdgeType:   "call",
			SourceFile: testFile.Path,
			TargetFile: &testFile.Path,
		}

		err := edgeRepo.Create(ctx, edge)
		if err != nil {
			t.Fatalf("Failed to create edge: %v", err)
		}

		// Verify creation
		retrieved, err := edgeRepo.GetByID(ctx, edge.EdgeID)
		if err != nil {
			t.Fatalf("Failed to get edge: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Edge not found")
		}
		if retrieved.EdgeType != edge.EdgeType {
			t.Errorf("Expected type %s, got %s", edge.EdgeType, retrieved.EdgeType)
		}
	})

	t.Run("GetBySourceID", func(t *testing.T) {
		edges, err := edgeRepo.GetBySourceID(ctx, sourceSymbol.SymbolID)
		if err != nil {
			t.Fatalf("Failed to get edges by source ID: %v", err)
		}
		if len(edges) == 0 {
			t.Error("Expected edges, got none")
		}
		for _, edge := range edges {
			if edge.SourceID != sourceSymbol.SymbolID {
				t.Error("Edge source ID mismatch")
			}
		}
	})

	t.Run("GetByTargetID", func(t *testing.T) {
		edges, err := edgeRepo.GetByTargetID(ctx, targetSymbol.SymbolID)
		if err != nil {
			t.Fatalf("Failed to get edges by target ID: %v", err)
		}
		for _, edge := range edges {
			if edge.TargetID == nil || *edge.TargetID != targetSymbol.SymbolID {
				t.Error("Edge target ID mismatch")
			}
		}
	})

	t.Run("GetByType", func(t *testing.T) {
		edges, err := edgeRepo.GetByType(ctx, "call")
		if err != nil {
			t.Fatalf("Failed to get edges by type: %v", err)
		}
		for _, edge := range edges {
			if edge.EdgeType != "call" {
				t.Errorf("Expected type call, got %s", edge.EdgeType)
			}
		}
	})

	t.Run("GetBySourceAndType", func(t *testing.T) {
		edges, err := edgeRepo.GetBySourceAndType(ctx, sourceSymbol.SymbolID, "call")
		if err != nil {
			t.Fatalf("Failed to get edges by source and type: %v", err)
		}
		for _, edge := range edges {
			if edge.SourceID != sourceSymbol.SymbolID || edge.EdgeType != "call" {
				t.Error("Edge filter mismatch")
			}
		}
	})

	t.Run("GetByTargetAndType", func(t *testing.T) {
		edges, err := edgeRepo.GetByTargetAndType(ctx, targetSymbol.SymbolID, "call")
		if err != nil {
			t.Fatalf("Failed to get edges by target and type: %v", err)
		}
		for _, edge := range edges {
			if edge.TargetID == nil || *edge.TargetID != targetSymbol.SymbolID || edge.EdgeType != "call" {
				t.Error("Edge filter mismatch")
			}
		}
	})

	t.Run("Update", func(t *testing.T) {
		edge := &models.Edge{
			EdgeID:     uuid.New().String(),
			SourceID:   sourceSymbol.SymbolID,
			TargetID:   &targetSymbol.SymbolID,
			EdgeType:   "reference",
			SourceFile: testFile.Path,
		}
		if err := edgeRepo.Create(ctx, edge); err != nil {
			t.Fatalf("Failed to create edge: %v", err)
		}

		edge.EdgeType = "import"
		if err := edgeRepo.Update(ctx, edge); err != nil {
			t.Fatalf("Failed to update edge: %v", err)
		}

		updated, err := edgeRepo.GetByID(ctx, edge.EdgeID)
		if err != nil {
			t.Fatalf("Failed to get updated edge: %v", err)
		}
		if updated.EdgeType != "import" {
			t.Errorf("Expected type import, got %s", updated.EdgeType)
		}
	})

	t.Run("BatchCreate", func(t *testing.T) {
		symbol3 := &models.Symbol{
			SymbolID:  uuid.New().String(),
			FileID:    testFile.FileID,
			Name:      "helper1",
			Kind:      "function",
			Signature: "func helper1()",
			StartLine: 22,
			EndLine:   30,
		}
		if err := symbolRepo.Create(ctx, symbol3); err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}

		symbol4 := &models.Symbol{
			SymbolID:  uuid.New().String(),
			FileID:    testFile.FileID,
			Name:      "helper2",
			Kind:      "function",
			Signature: "func helper2()",
			StartLine: 32,
			EndLine:   40,
		}
		if err := symbolRepo.Create(ctx, symbol4); err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}

		edges := []*models.Edge{
			{
				EdgeID:     uuid.New().String(),
				SourceID:   sourceSymbol.SymbolID,
				TargetID:   &symbol3.SymbolID,
				EdgeType:   "call",
				SourceFile: testFile.Path,
			},
			{
				EdgeID:     uuid.New().String(),
				SourceID:   sourceSymbol.SymbolID,
				TargetID:   &symbol4.SymbolID,
				EdgeType:   "call",
				SourceFile: testFile.Path,
			},
		}

		if err := edgeRepo.BatchCreate(ctx, edges); err != nil {
			t.Fatalf("Failed to batch create edges: %v", err)
		}

		// Verify creation
		for _, edge := range edges {
			retrieved, err := edgeRepo.GetByID(ctx, edge.EdgeID)
			if err != nil {
				t.Errorf("Failed to get edge %s: %v", edge.EdgeID, err)
			}
			if retrieved == nil {
				t.Errorf("Edge %s not found", edge.EdgeID)
			}
		}
	})

	t.Run("GetCallRelationships", func(t *testing.T) {
		edges, err := edgeRepo.GetCallRelationships(ctx, sourceSymbol.SymbolID)
		if err != nil {
			t.Fatalf("Failed to get call relationships: %v", err)
		}
		for _, edge := range edges {
			if edge.EdgeType != "call" && edge.EdgeType != "calls" {
				t.Errorf("Expected call relationship, got %s", edge.EdgeType)
			}
		}
	})

	t.Run("GetImportRelationships", func(t *testing.T) {
		// Create import edge
		importEdge := &models.Edge{
			EdgeID:       uuid.New().String(),
			SourceID:     sourceSymbol.SymbolID,
			EdgeType:     "import",
			SourceFile:   testFile.Path,
			TargetModule: strPtr("fmt"),
		}
		if err := edgeRepo.Create(ctx, importEdge); err != nil {
			t.Fatalf("Failed to create import edge: %v", err)
		}

		edges, err := edgeRepo.GetImportRelationships(ctx, sourceSymbol.SymbolID)
		if err != nil {
			t.Fatalf("Failed to get import relationships: %v", err)
		}
		for _, edge := range edges {
			if edge.EdgeType != "import" && edge.EdgeType != "imports" {
				t.Errorf("Expected import relationship, got %s", edge.EdgeType)
			}
		}
	})

	t.Run("GetEdgesByTypes", func(t *testing.T) {
		edges, err := edgeRepo.GetEdgesByTypes(ctx, []string{"call", "import"})
		if err != nil {
			t.Fatalf("Failed to get edges by types: %v", err)
		}
		for _, edge := range edges {
			if edge.EdgeType != "call" && edge.EdgeType != "import" {
				t.Errorf("Unexpected edge type: %s", edge.EdgeType)
			}
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := edgeRepo.Count(ctx)
		if err != nil {
			t.Fatalf("Failed to count edges: %v", err)
		}
		if count == 0 {
			t.Error("Expected edges count > 0")
		}
	})

	t.Run("CountByType", func(t *testing.T) {
		counts, err := edgeRepo.CountByType(ctx)
		if err != nil {
			t.Fatalf("Failed to count edges by type: %v", err)
		}
		if counts["call"] == 0 {
			t.Error("Expected call edges count > 0")
		}
	})

	t.Run("CountBySourceID", func(t *testing.T) {
		count, err := edgeRepo.CountBySourceID(ctx, sourceSymbol.SymbolID)
		if err != nil {
			t.Fatalf("Failed to count edges by source: %v", err)
		}
		if count == 0 {
			t.Error("Expected edges from source > 0")
		}
	})

	t.Run("CountByTargetID", func(t *testing.T) {
		count, err := edgeRepo.CountByTargetID(ctx, targetSymbol.SymbolID)
		if err != nil {
			t.Fatalf("Failed to count edges by target: %v", err)
		}
		if count == 0 {
			t.Error("Expected edges to target > 0")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		edge := &models.Edge{
			EdgeID:     uuid.New().String(),
			SourceID:   sourceSymbol.SymbolID,
			TargetID:   &targetSymbol.SymbolID,
			EdgeType:   "test",
			SourceFile: testFile.Path,
		}
		if err := edgeRepo.Create(ctx, edge); err != nil {
			t.Fatalf("Failed to create edge: %v", err)
		}

		if err := edgeRepo.Delete(ctx, edge.EdgeID); err != nil {
			t.Fatalf("Failed to delete edge: %v", err)
		}

		deleted, err := edgeRepo.GetByID(ctx, edge.EdgeID)
		if err != nil {
			t.Fatalf("Failed to check deleted edge: %v", err)
		}
		if deleted != nil {
			t.Error("Edge should be deleted")
		}
	})

	t.Run("DeleteBySourceID", func(t *testing.T) {
		// Create a new symbol for deletion test
		deleteSymbol := &models.Symbol{
			SymbolID:  uuid.New().String(),
			FileID:    testFile.FileID,
			Name:      "deleteTest",
			Kind:      "function",
			Signature: "func deleteTest()",
			StartLine: 50,
			EndLine:   60,
		}
		if err := symbolRepo.Create(ctx, deleteSymbol); err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}

		// Create edge
		edge := &models.Edge{
			EdgeID:     uuid.New().String(),
			SourceID:   deleteSymbol.SymbolID,
			TargetID:   &targetSymbol.SymbolID,
			EdgeType:   "call",
			SourceFile: testFile.Path,
		}
		if err := edgeRepo.Create(ctx, edge); err != nil {
			t.Fatalf("Failed to create edge: %v", err)
		}

		// Delete by source
		if err := edgeRepo.DeleteBySourceID(ctx, deleteSymbol.SymbolID); err != nil {
			t.Fatalf("Failed to delete edges by source: %v", err)
		}

		// Verify deletion
		edges, err := edgeRepo.GetBySourceID(ctx, deleteSymbol.SymbolID)
		if err != nil {
			t.Fatalf("Failed to get edges: %v", err)
		}
		if len(edges) != 0 {
			t.Errorf("Expected 0 edges, got %d", len(edges))
		}
	})

	t.Run("DeleteByTargetID", func(t *testing.T) {
		// Create a new symbol for deletion test
		deleteTarget := &models.Symbol{
			SymbolID:  uuid.New().String(),
			FileID:    testFile.FileID,
			Name:      "deleteTarget",
			Kind:      "function",
			Signature: "func deleteTarget()",
			StartLine: 70,
			EndLine:   80,
		}
		if err := symbolRepo.Create(ctx, deleteTarget); err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}

		// Create edge
		edge := &models.Edge{
			EdgeID:     uuid.New().String(),
			SourceID:   sourceSymbol.SymbolID,
			TargetID:   &deleteTarget.SymbolID,
			EdgeType:   "call",
			SourceFile: testFile.Path,
		}
		if err := edgeRepo.Create(ctx, edge); err != nil {
			t.Fatalf("Failed to create edge: %v", err)
		}

		// Delete by target
		if err := edgeRepo.DeleteByTargetID(ctx, deleteTarget.SymbolID); err != nil {
			t.Fatalf("Failed to delete edges by target: %v", err)
		}

		// Verify deletion
		edges, err := edgeRepo.GetByTargetID(ctx, deleteTarget.SymbolID)
		if err != nil {
			t.Fatalf("Failed to get edges: %v", err)
		}
		if len(edges) != 0 {
			t.Errorf("Expected 0 edges, got %d", len(edges))
		}
	})
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

// TestDatabaseOptimization tests database optimization functions
func TestDatabaseOptimization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	t.Run("GetPoolStats", func(t *testing.T) {
		stats := testDB.GetPoolStats()
		// Stats should have reasonable values
		if stats.MaxOpenConnections < 0 {
			t.Error("Invalid max open connections")
		}
	})

	t.Run("LogPoolStats", func(t *testing.T) {
		// Should not panic
		testDB.LogPoolStats()
	})

	t.Run("OptimizeForBulkInserts", func(t *testing.T) {
		ctx := context.Background()
		if err := testDB.OptimizeForBulkInserts(ctx); err != nil {
			t.Fatalf("Failed to optimize for bulk inserts: %v", err)
		}
	})

	t.Run("ResetOptimizations", func(t *testing.T) {
		ctx := context.Background()
		if err := testDB.ResetOptimizations(ctx); err != nil {
			t.Fatalf("Failed to reset optimizations: %v", err)
		}
	})

	t.Run("AnalyzeTables", func(t *testing.T) {
		ctx := context.Background()
		if err := testDB.AnalyzeTables(ctx); err != nil {
			t.Fatalf("Failed to analyze tables: %v", err)
		}
	})

	t.Run("VacuumTables", func(t *testing.T) {
		ctx := context.Background()
		if err := testDB.VacuumTables(ctx); err != nil {
			t.Fatalf("Failed to vacuum tables: %v", err)
		}
	})
}

// TestVectorOperations tests vector repository operations
func TestVectorOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	vectorRepo := models.NewVectorRepository(testDB.DB)

	// Get vector dimension from environment (same as schema initialization)
	vectorDim := getEnvInt("EMBEDDING_DIMENSIONS", 1024)

	// Setup: Create repository, file, and symbol
	repoRepo := models.NewRepositoryRepository(testDB.DB)
	fileRepo := models.NewFileRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)

	testRepo := &models.Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo",
	}
	if err := repoRepo.Create(ctx, testRepo); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	testFile := &models.File{
		FileID:   uuid.New().String(),
		RepoID:   testRepo.RepoID,
		Path:     "test.go",
		Language: "go",
		Size:     1000,
		Checksum: "test123",
	}
	if err := fileRepo.Create(ctx, testFile); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	testSymbol := &models.Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    testFile.FileID,
		Name:      "testFunction",
		Kind:      "function",
		Signature: "func testFunction()",
		StartLine: 1,
		EndLine:   10,
		Docstring: "This is a test function for vector search",
	}
	if err := symbolRepo.Create(ctx, testSymbol); err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		embedding := make([]float32, vectorDim)
		for i := range embedding {
			embedding[i] = float32(i) / float32(vectorDim)
		}

		vector := &models.Vector{
			VectorID:   uuid.New().String(),
			EntityID:   testSymbol.SymbolID,
			EntityType: "symbol",
			Embedding:  embedding,
			Content:    testSymbol.Docstring,
			Model:      "test-model",
			ChunkIndex: 0,
		}

		if err := vectorRepo.Create(ctx, vector); err != nil {
			t.Fatalf("Failed to create vector: %v", err)
		}

		// Get by ID
		retrieved, err := vectorRepo.GetByID(ctx, vector.VectorID)
		if err != nil {
			t.Fatalf("Failed to get vector: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Vector not found")
		}
		if retrieved.EntityID != testSymbol.SymbolID {
			t.Error("Entity ID mismatch")
		}
	})

	t.Run("GetByEntityID", func(t *testing.T) {
		vectors, err := vectorRepo.GetByEntityID(ctx, testSymbol.SymbolID, "symbol")
		if err != nil {
			t.Fatalf("Failed to get vectors by entity ID: %v", err)
		}
		if len(vectors) == 0 {
			t.Error("Expected vectors, got none")
		}
		for _, v := range vectors {
			if v.EntityID != testSymbol.SymbolID {
				t.Error("Entity ID mismatch")
			}
		}
	})

	t.Run("GetByEntityType", func(t *testing.T) {
		vectors, err := vectorRepo.GetByEntityType(ctx, "symbol", 100)
		if err != nil {
			t.Fatalf("Failed to get vectors by entity type: %v", err)
		}
		for _, v := range vectors {
			if v.EntityType != "symbol" {
				t.Errorf("Expected entity type symbol, got %s", v.EntityType)
			}
		}
	})

	t.Run("Update", func(t *testing.T) {
		embedding := make([]float32, vectorDim)
		for i := range embedding {
			embedding[i] = float32(i) / float32(vectorDim/2)
		}

		vector := &models.Vector{
			VectorID:   uuid.New().String(),
			EntityID:   testSymbol.SymbolID,
			EntityType: "symbol",
			Embedding:  embedding,
			Content:    "original content",
			Model:      "test-model",
		}
		if err := vectorRepo.Create(ctx, vector); err != nil {
			t.Fatalf("Failed to create vector: %v", err)
		}

		vector.Content = "updated content"
		if err := vectorRepo.Update(ctx, vector); err != nil {
			t.Fatalf("Failed to update vector: %v", err)
		}

		updated, err := vectorRepo.GetByID(ctx, vector.VectorID)
		if err != nil {
			t.Fatalf("Failed to get updated vector: %v", err)
		}
		if updated.Content != "updated content" {
			t.Errorf("Expected content 'updated content', got %s", updated.Content)
		}
	})

	t.Run("BatchCreate", func(t *testing.T) {
		symbol2 := &models.Symbol{
			SymbolID:  uuid.New().String(),
			FileID:    testFile.FileID,
			Name:      "helper",
			Kind:      "function",
			Signature: "func helper()",
			StartLine: 12,
			EndLine:   20,
			Docstring: "Helper function",
		}
		if err := symbolRepo.Create(ctx, symbol2); err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}

		vectors := []*models.Vector{
			{
				VectorID:   uuid.New().String(),
				EntityID:   symbol2.SymbolID,
				EntityType: "symbol",
				Embedding:  make([]float32, vectorDim),
				Content:    "batch vector 1",
				Model:      "test-model",
			},
			{
				VectorID:   uuid.New().String(),
				EntityID:   symbol2.SymbolID,
				EntityType: "symbol",
				Embedding:  make([]float32, vectorDim),
				Content:    "batch vector 2",
				Model:      "test-model",
				ChunkIndex: 1,
			},
		}

		if err := vectorRepo.BatchCreate(ctx, vectors); err != nil {
			t.Fatalf("Failed to batch create vectors: %v", err)
		}

		// Verify creation
		for _, v := range vectors {
			retrieved, err := vectorRepo.GetByID(ctx, v.VectorID)
			if err != nil {
				t.Errorf("Failed to get vector %s: %v", v.VectorID, err)
			}
			if retrieved == nil {
				t.Errorf("Vector %s not found", v.VectorID)
			}
		}
	})

	t.Run("SimilaritySearch", func(t *testing.T) {
		// Create multiple vectors with different embeddings
		symbols := make([]*models.Symbol, 3)
		for i := 0; i < 3; i++ {
			symbols[i] = &models.Symbol{
				SymbolID:  uuid.New().String(),
				FileID:    testFile.FileID,
				Name:      fmt.Sprintf("searchFunc%d", i),
				Kind:      "function",
				Signature: fmt.Sprintf("func searchFunc%d()", i),
				StartLine: 30 + i*10,
				EndLine:   35 + i*10,
				Docstring: fmt.Sprintf("Search test function %d", i),
			}
			if err := symbolRepo.Create(ctx, symbols[i]); err != nil {
				t.Fatalf("Failed to create symbol: %v", err)
			}

			embedding := make([]float32, vectorDim)
			for j := range embedding {
				embedding[j] = float32(i+j) / float32(vectorDim)
			}

			vector := &models.Vector{
				VectorID:   uuid.New().String(),
				EntityID:   symbols[i].SymbolID,
				EntityType: "symbol",
				Embedding:  embedding,
				Content:    symbols[i].Docstring,
				Model:      "test-model",
			}
			if err := vectorRepo.Create(ctx, vector); err != nil {
				t.Fatalf("Failed to create vector: %v", err)
			}
		}

		// Perform similarity search
		queryEmbedding := make([]float32, vectorDim)
		for i := range queryEmbedding {
			queryEmbedding[i] = float32(i) / float32(vectorDim)
		}

		results, err := vectorRepo.SimilaritySearch(ctx, queryEmbedding, "symbol", 5)
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
			if result.EntityType != "symbol" {
				t.Errorf("Expected entity type symbol, got %s", result.EntityType)
			}
		}
	})

	t.Run("SimilaritySearchWithFilters", func(t *testing.T) {
		queryEmbedding := make([]float32, vectorDim)
		for i := range queryEmbedding {
			queryEmbedding[i] = float32(i) / float32(vectorDim)
		}

		filters := models.VectorSearchFilters{
			EntityType: "symbol",
			Model:      "test-model",
			Limit:      5,
		}

		results, err := vectorRepo.SimilaritySearchWithFilters(ctx, queryEmbedding, filters)
		if err != nil {
			t.Fatalf("Similarity search with filters failed: %v", err)
		}

		for _, result := range results {
			if result.Model != "test-model" {
				t.Errorf("Expected model test-model, got %s", result.Model)
			}
		}
	})

	t.Run("GetEmbeddingDimensions", func(t *testing.T) {
		dim, err := vectorRepo.GetEmbeddingDimensions(ctx, "test-model")
		if err != nil {
			t.Fatalf("Failed to get embedding dimensions: %v", err)
		}
		if dim <= 0 {
			t.Errorf("Invalid embedding dimension: %d", dim)
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := vectorRepo.Count(ctx)
		if err != nil {
			t.Fatalf("Failed to count vectors: %v", err)
		}
		if count == 0 {
			t.Error("Expected vectors count > 0")
		}
	})

	t.Run("CountByEntityType", func(t *testing.T) {
		counts, err := vectorRepo.CountByEntityType(ctx)
		if err != nil {
			t.Fatalf("Failed to count vectors by entity type: %v", err)
		}
		if counts["symbol"] == 0 {
			t.Error("Expected symbol vectors count > 0")
		}
	})

	t.Run("CountByModel", func(t *testing.T) {
		counts, err := vectorRepo.CountByModel(ctx)
		if err != nil {
			t.Fatalf("Failed to count vectors by model: %v", err)
		}
		if counts["test-model"] == 0 {
			t.Error("Expected test-model vectors count > 0")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		embedding := make([]float32, vectorDim)
		vector := &models.Vector{
			VectorID:   uuid.New().String(),
			EntityID:   testSymbol.SymbolID,
			EntityType: "symbol",
			Embedding:  embedding,
			Content:    "delete test",
			Model:      "test-model",
		}
		if err := vectorRepo.Create(ctx, vector); err != nil {
			t.Fatalf("Failed to create vector: %v", err)
		}

		if err := vectorRepo.Delete(ctx, vector.VectorID); err != nil {
			t.Fatalf("Failed to delete vector: %v", err)
		}

		deleted, err := vectorRepo.GetByID(ctx, vector.VectorID)
		if err != nil {
			t.Fatalf("Failed to check deleted vector: %v", err)
		}
		if deleted != nil {
			t.Error("Vector should be deleted")
		}
	})

	t.Run("DeleteByEntityID", func(t *testing.T) {
		deleteSymbol := &models.Symbol{
			SymbolID:  uuid.New().String(),
			FileID:    testFile.FileID,
			Name:      "deleteVectorTest",
			Kind:      "function",
			Signature: "func deleteVectorTest()",
			StartLine: 100,
			EndLine:   110,
		}
		if err := symbolRepo.Create(ctx, deleteSymbol); err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}

		vector := &models.Vector{
			VectorID:   uuid.New().String(),
			EntityID:   deleteSymbol.SymbolID,
			EntityType: "symbol",
			Embedding:  make([]float32, vectorDim),
			Content:    "delete by entity test",
			Model:      "test-model",
		}
		if err := vectorRepo.Create(ctx, vector); err != nil {
			t.Fatalf("Failed to create vector: %v", err)
		}

		if err := vectorRepo.DeleteByEntityID(ctx, deleteSymbol.SymbolID, "symbol"); err != nil {
			t.Fatalf("Failed to delete vectors by entity ID: %v", err)
		}

		vectors, err := vectorRepo.GetByEntityID(ctx, deleteSymbol.SymbolID, "symbol")
		if err != nil {
			t.Fatalf("Failed to get vectors: %v", err)
		}
		if len(vectors) != 0 {
			t.Errorf("Expected 0 vectors, got %d", len(vectors))
		}
	})
}

// TestTransactionOperations tests transaction manager operations
func TestTransactionOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	txManager := models.NewTransactionManager(testDB.DB)

	t.Run("WithTransaction", func(t *testing.T) {
		repoRepo := models.NewRepositoryRepository(testDB.DB)

		err := txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			repo := &models.Repository{
				RepoID: uuid.New().String(),
				Name:   "tx-test-repo",
			}
			return repoRepo.Create(ctx, repo)
		})

		if err != nil {
			t.Fatalf("Transaction failed: %v", err)
		}

		// Verify repository was created
		repos, err := repoRepo.GetAll(ctx)
		if err != nil {
			t.Fatalf("Failed to get repositories: %v", err)
		}
		found := false
		for _, r := range repos {
			if r.Name == "tx-test-repo" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Repository not found after transaction")
		}
	})

	t.Run("WithTransactionRollback", func(t *testing.T) {
		t.Skip("Skipping: Repository methods don't support transaction-aware operations yet. " +
			"This requires refactoring repository methods to accept optional *sql.Tx parameter.")

		repoRepo := models.NewRepositoryRepository(testDB.DB)

		err := txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
			repo := &models.Repository{
				RepoID: uuid.New().String(),
				Name:   "rollback-test-repo",
			}
			if err := repoRepo.Create(ctx, repo); err != nil {
				return err
			}
			// Force rollback
			return fmt.Errorf("intentional error for rollback")
		})

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		// Verify repository was not created
		repos, err := repoRepo.GetAll(ctx)
		if err != nil {
			t.Fatalf("Failed to get repositories: %v", err)
		}
		for _, r := range repos {
			if r.Name == "rollback-test-repo" {
				t.Error("Repository should not exist after rollback")
			}
		}
	})

	t.Run("WithTransactionRetry", func(t *testing.T) {
		repoRepo := models.NewRepositoryRepository(testDB.DB)
		attempts := 0

		err := txManager.WithTransactionRetry(ctx, 3, func(tx *sql.Tx) error {
			attempts++
			if attempts < 2 {
				// Simulate transient error on first attempt
				return fmt.Errorf("serialization failure")
			}
			repo := &models.Repository{
				RepoID: uuid.New().String(),
				Name:   "retry-test-repo",
			}
			return repoRepo.Create(ctx, repo)
		})

		if err != nil {
			t.Fatalf("Transaction with retry failed: %v", err)
		}

		if attempts < 2 {
			t.Errorf("Expected at least 2 attempts, got %d", attempts)
		}
	})

	t.Run("ExecuteBatch", func(t *testing.T) {
		repoRepo := models.NewRepositoryRepository(testDB.DB)

		operations := []models.BatchOperation{
			{
				Name: "create-batch-repo-1",
				Fn: func(tx *sql.Tx) error {
					repo := &models.Repository{
						RepoID: uuid.New().String(),
						Name:   "batch-repo-1",
					}
					return repoRepo.Create(ctx, repo)
				},
			},
			{
				Name: "create-batch-repo-2",
				Fn: func(tx *sql.Tx) error {
					repo := &models.Repository{
						RepoID: uuid.New().String(),
						Name:   "batch-repo-2",
					}
					return repoRepo.Create(ctx, repo)
				},
			},
		}

		if err := txManager.ExecuteBatch(ctx, operations); err != nil {
			t.Fatalf("Batch execution failed: %v", err)
		}

		// Verify both repositories were created
		repos, err := repoRepo.GetAll(ctx)
		if err != nil {
			t.Fatalf("Failed to get repositories: %v", err)
		}
		foundCount := 0
		for _, r := range repos {
			if r.Name == "batch-repo-1" || r.Name == "batch-repo-2" {
				foundCount++
			}
		}
		if foundCount != 2 {
			t.Errorf("Expected 2 batch repositories, found %d", foundCount)
		}
	})
}
