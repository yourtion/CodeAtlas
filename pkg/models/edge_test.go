package models

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestEdgeRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)
	edgeRepo := NewEdgeRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-edge",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repoRepo.Delete(ctx, repo.RepoID)

	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repo.RepoID,
		Path:     "src/main.go",
		Language: "go",
		Size:     1024,
		Checksum: "abc123",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Create test symbols
	sourceSymbol := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "main",
		Kind:      "function",
		Signature: "func main()",
		StartLine: 10,
		EndLine:   20,
		StartByte: 100,
		EndByte:   200,
	}
	err = symbolRepo.Create(ctx, sourceSymbol)
	if err != nil {
		t.Fatalf("Failed to create source symbol: %v", err)
	}
	defer symbolRepo.Delete(ctx, sourceSymbol.SymbolID)

	targetSymbol := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "helper",
		Kind:      "function",
		Signature: "func helper()",
		StartLine: 25,
		EndLine:   30,
		StartByte: 250,
		EndByte:   300,
	}
	err = symbolRepo.Create(ctx, targetSymbol)
	if err != nil {
		t.Fatalf("Failed to create target symbol: %v", err)
	}
	defer symbolRepo.Delete(ctx, targetSymbol.SymbolID)

	// Create test edge
	edge := &Edge{
		EdgeID:     uuid.New().String(),
		SourceID:   sourceSymbol.SymbolID,
		TargetID:   &targetSymbol.SymbolID,
		EdgeType:   "call",
		SourceFile: file.Path,
		TargetFile: &file.Path,
	}

	err = edgeRepo.Create(ctx, edge)
	if err != nil {
		t.Fatalf("Failed to create edge: %v", err)
	}
	defer edgeRepo.Delete(ctx, edge.EdgeID)

	// Verify the edge was created
	retrieved, err := edgeRepo.GetByID(ctx, edge.EdgeID)
	if err != nil {
		t.Fatalf("Failed to retrieve edge: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Edge not found")
	}

	if retrieved.SourceID != edge.SourceID {
		t.Errorf("Expected source ID %s, got %s", edge.SourceID, retrieved.SourceID)
	}
	if retrieved.TargetID == nil || *retrieved.TargetID != *edge.TargetID {
		t.Errorf("Expected target ID %s, got %v", *edge.TargetID, retrieved.TargetID)
	}
	if retrieved.EdgeType != edge.EdgeType {
		t.Errorf("Expected edge type %s, got %s", edge.EdgeType, retrieved.EdgeType)
	}
}

func TestEdgeRepository_GetBySourceID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)
	edgeRepo := NewEdgeRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-getbysource",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repoRepo.Delete(ctx, repo.RepoID)

	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repo.RepoID,
		Path:     "src/caller.go",
		Language: "go",
		Size:     512,
		Checksum: "def456",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Create test symbols
	sourceSymbol := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "caller",
		Kind:      "function",
		Signature: "func caller()",
		StartLine: 5,
		EndLine:   10,
		StartByte: 50,
		EndByte:   100,
	}
	err = symbolRepo.Create(ctx, sourceSymbol)
	if err != nil {
		t.Fatalf("Failed to create source symbol: %v", err)
	}
	defer symbolRepo.Delete(ctx, sourceSymbol.SymbolID)

	targetSymbol1 := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "callee1",
		Kind:      "function",
		Signature: "func callee1()",
		StartLine: 15,
		EndLine:   20,
		StartByte: 150,
		EndByte:   200,
	}
	err = symbolRepo.Create(ctx, targetSymbol1)
	if err != nil {
		t.Fatalf("Failed to create target symbol 1: %v", err)
	}
	defer symbolRepo.Delete(ctx, targetSymbol1.SymbolID)

	targetSymbol2 := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "callee2",
		Kind:      "function",
		Signature: "func callee2()",
		StartLine: 25,
		EndLine:   30,
		StartByte: 250,
		EndByte:   300,
	}
	err = symbolRepo.Create(ctx, targetSymbol2)
	if err != nil {
		t.Fatalf("Failed to create target symbol 2: %v", err)
	}
	defer symbolRepo.Delete(ctx, targetSymbol2.SymbolID)

	// Create test edges
	testEdges := []*Edge{
		{
			EdgeID:     uuid.New().String(),
			SourceID:   sourceSymbol.SymbolID,
			TargetID:   &targetSymbol1.SymbolID,
			EdgeType:   "call",
			SourceFile: file.Path,
			TargetFile: &file.Path,
		},
		{
			EdgeID:     uuid.New().String(),
			SourceID:   sourceSymbol.SymbolID,
			TargetID:   &targetSymbol2.SymbolID,
			EdgeType:   "call",
			SourceFile: file.Path,
			TargetFile: &file.Path,
		},
	}

	for _, e := range testEdges {
		err = edgeRepo.Create(ctx, e)
		if err != nil {
			t.Fatalf("Failed to create edge: %v", err)
		}
		defer edgeRepo.Delete(ctx, e.EdgeID)
	}

	// Get edges by source ID
	edges, err := edgeRepo.GetBySourceID(ctx, sourceSymbol.SymbolID)
	if err != nil {
		t.Fatalf("Failed to get edges by source ID: %v", err)
	}

	if len(edges) != len(testEdges) {
		t.Errorf("Expected %d edges, got %d", len(testEdges), len(edges))
	}
}

func TestEdgeRepository_GetByTargetID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)
	edgeRepo := NewEdgeRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-getbytarget",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repoRepo.Delete(ctx, repo.RepoID)

	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repo.RepoID,
		Path:     "src/callee.go",
		Language: "go",
		Size:     256,
		Checksum: "ghi789",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Create test symbols
	targetSymbol := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "callee",
		Kind:      "function",
		Signature: "func callee()",
		StartLine: 5,
		EndLine:   10,
		StartByte: 50,
		EndByte:   100,
	}
	err = symbolRepo.Create(ctx, targetSymbol)
	if err != nil {
		t.Fatalf("Failed to create target symbol: %v", err)
	}
	defer symbolRepo.Delete(ctx, targetSymbol.SymbolID)

	sourceSymbol1 := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "caller1",
		Kind:      "function",
		Signature: "func caller1()",
		StartLine: 15,
		EndLine:   20,
		StartByte: 150,
		EndByte:   200,
	}
	err = symbolRepo.Create(ctx, sourceSymbol1)
	if err != nil {
		t.Fatalf("Failed to create source symbol 1: %v", err)
	}
	defer symbolRepo.Delete(ctx, sourceSymbol1.SymbolID)

	sourceSymbol2 := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "caller2",
		Kind:      "function",
		Signature: "func caller2()",
		StartLine: 25,
		EndLine:   30,
		StartByte: 250,
		EndByte:   300,
	}
	err = symbolRepo.Create(ctx, sourceSymbol2)
	if err != nil {
		t.Fatalf("Failed to create source symbol 2: %v", err)
	}
	defer symbolRepo.Delete(ctx, sourceSymbol2.SymbolID)

	// Create test edges
	testEdges := []*Edge{
		{
			EdgeID:     uuid.New().String(),
			SourceID:   sourceSymbol1.SymbolID,
			TargetID:   &targetSymbol.SymbolID,
			EdgeType:   "call",
			SourceFile: file.Path,
			TargetFile: &file.Path,
		},
		{
			EdgeID:     uuid.New().String(),
			SourceID:   sourceSymbol2.SymbolID,
			TargetID:   &targetSymbol.SymbolID,
			EdgeType:   "call",
			SourceFile: file.Path,
			TargetFile: &file.Path,
		},
	}

	for _, e := range testEdges {
		err = edgeRepo.Create(ctx, e)
		if err != nil {
			t.Fatalf("Failed to create edge: %v", err)
		}
		defer edgeRepo.Delete(ctx, e.EdgeID)
	}

	// Get edges by target ID
	edges, err := edgeRepo.GetByTargetID(ctx, targetSymbol.SymbolID)
	if err != nil {
		t.Fatalf("Failed to get edges by target ID: %v", err)
	}

	if len(edges) != len(testEdges) {
		t.Errorf("Expected %d edges, got %d", len(testEdges), len(edges))
	}
}

func TestEdgeRepository_GetByType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)
	edgeRepo := NewEdgeRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-getbytype",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repoRepo.Delete(ctx, repo.RepoID)

	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repo.RepoID,
		Path:     "src/types.go",
		Language: "go",
		Size:     512,
		Checksum: "jkl012",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Create test symbols
	symbol1 := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "symbol1",
		Kind:      "function",
		Signature: "func symbol1()",
		StartLine: 5,
		EndLine:   10,
		StartByte: 50,
		EndByte:   100,
	}
	err = symbolRepo.Create(ctx, symbol1)
	if err != nil {
		t.Fatalf("Failed to create symbol 1: %v", err)
	}
	defer symbolRepo.Delete(ctx, symbol1.SymbolID)

	symbol2 := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "symbol2",
		Kind:      "function",
		Signature: "func symbol2()",
		StartLine: 15,
		EndLine:   20,
		StartByte: 150,
		EndByte:   200,
	}
	err = symbolRepo.Create(ctx, symbol2)
	if err != nil {
		t.Fatalf("Failed to create symbol 2: %v", err)
	}
	defer symbolRepo.Delete(ctx, symbol2.SymbolID)

	// Create test edges with different types
	testEdges := []*Edge{
		{
			EdgeID:     uuid.New().String(),
			SourceID:   symbol1.SymbolID,
			TargetID:   &symbol2.SymbolID,
			EdgeType:   "call",
			SourceFile: file.Path,
			TargetFile: &file.Path,
		},
		{
			EdgeID:     uuid.New().String(),
			SourceID:   symbol1.SymbolID,
			TargetID:   nil,
			EdgeType:   "import",
			SourceFile: file.Path,
			TargetFile: nil,
		},
	}

	for _, e := range testEdges {
		err = edgeRepo.Create(ctx, e)
		if err != nil {
			t.Fatalf("Failed to create edge: %v", err)
		}
		defer edgeRepo.Delete(ctx, e.EdgeID)
	}

	// Get call edges
	callEdges, err := edgeRepo.GetByType(ctx, "call")
	if err != nil {
		t.Fatalf("Failed to get call edges: %v", err)
	}

	// Should have at least 1 call edge from our test
	found := false
	for _, edge := range callEdges {
		if edge.SourceID == symbol1.SymbolID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find test call edge")
	}
}

func TestEdgeRepository_BatchCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)
	edgeRepo := NewEdgeRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-batch",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repoRepo.Delete(ctx, repo.RepoID)

	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repo.RepoID,
		Path:     "src/batch.go",
		Language: "go",
		Size:     1024,
		Checksum: "batch123",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Create test symbols
	symbols := make([]*Symbol, 3)
	for i := 0; i < 3; i++ {
		symbols[i] = &Symbol{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "symbol" + string(rune('A'+i)),
			Kind:      "function",
			Signature: "func symbol()",
			StartLine: (i + 1) * 10,
			EndLine:   (i+1)*10 + 5,
			StartByte: (i + 1) * 100,
			EndByte:   (i+1)*100 + 50,
		}
		err = symbolRepo.Create(ctx, symbols[i])
		if err != nil {
			t.Fatalf("Failed to create symbol %d: %v", i, err)
		}
		defer symbolRepo.Delete(ctx, symbols[i].SymbolID)
	}

	// Create test edges
	testEdges := []*Edge{
		{
			EdgeID:     uuid.New().String(),
			SourceID:   symbols[0].SymbolID,
			TargetID:   &symbols[1].SymbolID,
			EdgeType:   "call",
			SourceFile: file.Path,
			TargetFile: &file.Path,
		},
		{
			EdgeID:     uuid.New().String(),
			SourceID:   symbols[0].SymbolID,
			TargetID:   &symbols[2].SymbolID,
			EdgeType:   "call",
			SourceFile: file.Path,
			TargetFile: &file.Path,
		},
	}

	// Batch create
	err = edgeRepo.BatchCreate(ctx, testEdges)
	if err != nil {
		t.Fatalf("Failed to batch create edges: %v", err)
	}

	// Clean up
	for _, e := range testEdges {
		defer edgeRepo.Delete(ctx, e.EdgeID)
	}

	// Verify all edges were created
	edges, err := edgeRepo.GetBySourceID(ctx, symbols[0].SymbolID)
	if err != nil {
		t.Fatalf("Failed to get edges: %v", err)
	}

	if len(edges) != len(testEdges) {
		t.Errorf("Expected %d edges, got %d", len(testEdges), len(edges))
	}
}

func TestEdgeRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	edgeRepo := NewEdgeRepository(testDB.DB)

	// Get initial count
	initialCount, err := edgeRepo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	// Count should be non-negative
	if initialCount < 0 {
		t.Errorf("Expected non-negative count, got %d", initialCount)
	}
}
