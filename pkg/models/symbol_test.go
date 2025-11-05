package models

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestSymbolRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create repository and file first
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-" + repoID[:8],
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

	repo := NewSymbolRepository(testDB.DB)

	symbol := &Symbol{
		SymbolID:        uuid.New().String(),
		FileID:          file.FileID,
		Name:            "TestFunction",
		Kind:            "function",
		Signature:       "func TestFunction() error",
		StartLine:       10,
		EndLine:         20,
		StartByte:       100,
		EndByte:         200,
		Docstring:       "Test function documentation",
		SemanticSummary: "A test function that does testing",
	}

	err = repo.Create(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	// Verify the symbol was created
	retrieved, err := repo.GetByID(ctx, symbol.SymbolID)
	if err != nil {
		t.Fatalf("Failed to retrieve symbol: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Symbol not found")
	}

	if retrieved.Name != symbol.Name {
		t.Errorf("Expected name %s, got %s", symbol.Name, retrieved.Name)
	}
	if retrieved.Kind != symbol.Kind {
		t.Errorf("Expected kind %s, got %s", symbol.Kind, retrieved.Kind)
	}
	if retrieved.Signature != symbol.Signature {
		t.Errorf("Expected signature %s, got %s", symbol.Signature, retrieved.Signature)
	}
	if retrieved.StartLine != symbol.StartLine {
		t.Errorf("Expected start line %d, got %d", symbol.StartLine, retrieved.StartLine)
	}
	if retrieved.EndLine != symbol.EndLine {
		t.Errorf("Expected end line %d, got %d", symbol.EndLine, retrieved.EndLine)
	}
	if retrieved.Docstring != symbol.Docstring {
		t.Errorf("Expected docstring %s, got %s", symbol.Docstring, retrieved.Docstring)
	}
}

func TestSymbolRepository_GetByFileID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create repository and file first
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-fileid-" + repoID[:8],
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

	repo := NewSymbolRepository(testDB.DB)

	symbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Function1",
			Kind:      "function",
			Signature: "func Function1()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Function2",
			Kind:      "function",
			Signature: "func Function2()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "TestClass",
			Kind:      "class",
			Signature: "class TestClass",
			StartLine: 25,
			EndLine:   35,
			StartByte: 250,
			EndByte:   350,
		},
	}

	err = repo.BatchCreate(ctx, symbols)
	if err != nil {
		t.Fatalf("Failed to batch create symbols: %v", err)
	}

	// Retrieve symbols by file ID
	retrieved, err := repo.GetByFileID(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to retrieve symbols by file ID: %v", err)
	}

	if len(retrieved) != len(symbols) {
		t.Errorf("Expected %d symbols, got %d", len(symbols), len(retrieved))
	}

	// Verify symbols are sorted by start line
	for i := 1; i < len(retrieved); i++ {
		if retrieved[i-1].StartLine > retrieved[i].StartLine {
			t.Error("Symbols are not sorted by start line")
			break
		}
	}
}

func TestSymbolRepository_GetByKind(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create repository and file first
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-kind-" + repoID[:8],
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

	repo := NewSymbolRepository(testDB.DB)

	symbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Function1",
			Kind:      "function",
			Signature: "func Function1()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Function2",
			Kind:      "function",
			Signature: "func Function2()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "TestClass",
			Kind:      "class",
			Signature: "class TestClass",
			StartLine: 25,
			EndLine:   35,
			StartByte: 250,
			EndByte:   350,
		},
	}

	err = repo.BatchCreate(ctx, symbols)
	if err != nil {
		t.Fatalf("Failed to batch create symbols: %v", err)
	}

	// Get function symbols
	functions, err := repo.GetByKind(ctx, file.FileID, "function")
	if err != nil {
		t.Fatalf("Failed to get function symbols: %v", err)
	}

	if len(functions) != 2 {
		t.Errorf("Expected 2 function symbols, got %d", len(functions))
	}

	for _, symbol := range functions {
		if symbol.Kind != "function" {
			t.Errorf("Expected kind 'function', got %s", symbol.Kind)
		}
	}

	// Get class symbols
	classes, err := repo.GetByKind(ctx, file.FileID, "class")
	if err != nil {
		t.Fatalf("Failed to get class symbols: %v", err)
	}

	if len(classes) != 1 {
		t.Errorf("Expected 1 class symbol, got %d", len(classes))
	}
}

func TestSymbolRepository_GetByName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create repository and file first
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-name-" + repoID[:8],
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

	repo := NewSymbolRepository(testDB.DB)

	symbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "TestFunction",
			Kind:      "function",
			Signature: "func TestFunction()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "TestClass",
			Kind:      "class",
			Signature: "class TestClass",
			StartLine: 15,
			EndLine:   25,
			StartByte: 150,
			EndByte:   250,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "AnotherFunction",
			Kind:      "function",
			Signature: "func AnotherFunction()",
			StartLine: 30,
			EndLine:   35,
			StartByte: 300,
			EndByte:   350,
		},
	}

	err = repo.BatchCreate(ctx, symbols)
	if err != nil {
		t.Fatalf("Failed to batch create symbols: %v", err)
	}

	// Search for symbols with "Test" in the name
	testSymbols, err := repo.GetByName(ctx, "%Test%")
	if err != nil {
		t.Fatalf("Failed to search symbols by name: %v", err)
	}

	if len(testSymbols) < 2 {
		t.Errorf("Expected at least 2 symbols with 'Test' in name, got %d", len(testSymbols))
	}

	// Verify our specific symbols are in the results
	foundTestFunction := false
	foundTestClass := false
	for _, sym := range testSymbols {
		if sym.Name == "TestFunction" && sym.FileID == file.FileID {
			foundTestFunction = true
		}
		if sym.Name == "TestClass" && sym.FileID == file.FileID {
			foundTestClass = true
		}
	}
	if !foundTestFunction || !foundTestClass {
		t.Error("Expected to find both TestFunction and TestClass in results")
	}

	// Verify results are sorted by name
	for i := 1; i < len(testSymbols); i++ {
		if testSymbols[i-1].Name > testSymbols[i].Name {
			t.Error("Symbols are not sorted by name")
			break
		}
	}
}

func TestSymbolRepository_GetSymbolsWithDocstrings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create repository and file first
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-doc-" + repoID[:8],
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

	repo := NewSymbolRepository(testDB.DB)

	symbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "DocumentedFunction",
			Kind:      "function",
			Signature: "func DocumentedFunction()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
			Docstring: "This function has documentation",
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "UndocumentedFunction",
			Kind:      "function",
			Signature: "func UndocumentedFunction()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
			// No docstring
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "EmptyDocFunction",
			Kind:      "function",
			Signature: "func EmptyDocFunction()",
			StartLine: 25,
			EndLine:   30,
			StartByte: 250,
			EndByte:   300,
			Docstring: "", // Empty docstring
		},
	}

	err = repo.BatchCreate(ctx, symbols)
	if err != nil {
		t.Fatalf("Failed to batch create symbols: %v", err)
	}

	// Get symbols with docstrings
	documented, err := repo.GetSymbolsWithDocstrings(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to get symbols with docstrings: %v", err)
	}

	if len(documented) != 1 {
		t.Errorf("Expected 1 documented symbol, got %d", len(documented))
	}

	if len(documented) > 0 && documented[0].Name != "DocumentedFunction" {
		t.Errorf("Expected DocumentedFunction, got %s", documented[0].Name)
	}
}

func TestSymbolRepository_CountByKind(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create repository and file first
	repoRepo := NewRepositoryRepository(testDB.DB)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-count-" + repoID[:8],
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

	repo := NewSymbolRepository(testDB.DB)

	symbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Function1",
			Kind:      "function",
			Signature: "func Function1()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Function2",
			Kind:      "function",
			Signature: "func Function2()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "TestClass",
			Kind:      "class",
			Signature: "class TestClass",
			StartLine: 25,
			EndLine:   35,
			StartByte: 250,
			EndByte:   350,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "TestVariable",
			Kind:      "variable",
			Signature: "var TestVariable",
			StartLine: 40,
			EndLine:   40,
			StartByte: 400,
			EndByte:   420,
		},
	}

	err = repo.BatchCreate(ctx, symbols)
	if err != nil {
		t.Fatalf("Failed to batch create symbols: %v", err)
	}

	// Get counts by kind
	counts, err := repo.CountByKind(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to get counts by kind: %v", err)
	}

	expectedCounts := map[string]int64{
		"function": 2,
		"class":    1,
		"variable": 1,
	}

	for kind, expectedCount := range expectedCounts {
		if counts[kind] != expectedCount {
			t.Errorf("Expected %d %s symbols, got %d", expectedCount, kind, counts[kind])
		}
	}
}

func TestSymbolRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create repository and file first
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

	repo := NewSymbolRepository(testDB.DB)

	symbol := &Symbol{
		SymbolID:        uuid.New().String(),
		FileID:          file.FileID,
		Name:            "OriginalFunction",
		Kind:            "function",
		Signature:       "func OriginalFunction()",
		StartLine:       10,
		EndLine:         20,
		StartByte:       100,
		EndByte:         200,
		Docstring:       "Original documentation",
		SemanticSummary: "Original summary",
	}

	err = repo.Create(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	// Update the symbol
	symbol.Name = "UpdatedFunction"
	symbol.Signature = "func UpdatedFunction() error"
	symbol.Docstring = "Updated documentation"
	symbol.SemanticSummary = "Updated summary"

	err = repo.Update(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to update symbol: %v", err)
	}

	// Verify the update
	retrieved, err := repo.GetByID(ctx, symbol.SymbolID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated symbol: %v", err)
	}

	if retrieved.Name != "UpdatedFunction" {
		t.Errorf("Expected name 'UpdatedFunction', got %s", retrieved.Name)
	}
	if retrieved.Signature != "func UpdatedFunction() error" {
		t.Errorf("Expected updated signature, got %s", retrieved.Signature)
	}
	if retrieved.Docstring != "Updated documentation" {
		t.Errorf("Expected updated docstring, got %s", retrieved.Docstring)
	}
	if retrieved.SemanticSummary != "Updated summary" {
		t.Errorf("Expected updated summary, got %s", retrieved.SemanticSummary)
	}
}

func TestSymbolRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// Create repository and file first
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

	repo := NewSymbolRepository(testDB.DB)

	symbol := &Symbol{
		SymbolID:  uuid.New().String(),
		FileID:    file.FileID,
		Name:      "DeleteMe",
		Kind:      "function",
		Signature: "func DeleteMe()",
		StartLine: 10,
		EndLine:   20,
		StartByte: 100,
		EndByte:   200,
	}

	err = repo.Create(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	// Delete the symbol
	err = repo.Delete(ctx, symbol.SymbolID)
	if err != nil {
		t.Fatalf("Failed to delete symbol: %v", err)
	}

	// Verify the symbol is gone
	retrieved, err := repo.GetByID(ctx, symbol.SymbolID)
	if err != nil {
		t.Fatalf("Failed to check if symbol exists: %v", err)
	}
	if retrieved != nil {
		t.Error("Symbol should have been deleted")
	}
}
