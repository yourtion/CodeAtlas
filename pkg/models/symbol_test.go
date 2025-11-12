package models

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestSymbolRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-symbol",
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

	// Create test symbol
	symbol := &Symbol{
		SymbolID:        uuid.New().String(),
		FileID:          file.FileID,
		Name:            "main",
		Kind:            "function",
		Signature:       "func main()",
		StartLine:       10,
		EndLine:         20,
		StartByte:       100,
		EndByte:         200,
		Docstring:       "Main entry point",
		SemanticSummary: "Application entry point",
	}

	err = symbolRepo.Create(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}
	defer symbolRepo.Delete(ctx, symbol.SymbolID)

	// Verify the symbol was created
	retrieved, err := symbolRepo.GetByID(ctx, symbol.SymbolID)
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
}

func TestSymbolRepository_GetByFileID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-getbyfileid",
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
		Path:     "src/utils.go",
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
	testSymbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Helper",
			Kind:      "function",
			Signature: "func Helper()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Utility",
			Kind:      "function",
			Signature: "func Utility()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
		},
	}

	for _, s := range testSymbols {
		err = symbolRepo.Create(ctx, s)
		if err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}
		defer symbolRepo.Delete(ctx, s.SymbolID)
	}

	// Get all symbols for file
	symbols, err := symbolRepo.GetByFileID(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to get symbols by file ID: %v", err)
	}

	if len(symbols) != len(testSymbols) {
		t.Errorf("Expected %d symbols, got %d", len(testSymbols), len(symbols))
	}

	// Verify symbols are sorted by start_line, start_byte
	for i := 1; i < len(symbols); i++ {
		if symbols[i-1].StartLine > symbols[i].StartLine {
			t.Error("Symbols are not sorted by start_line")
			break
		}
		if symbols[i-1].StartLine == symbols[i].StartLine &&
			symbols[i-1].StartByte > symbols[i].StartByte {
			t.Error("Symbols are not sorted by start_byte")
			break
		}
	}
}

func TestSymbolRepository_GetByKind(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-getbykind",
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
		Size:     256,
		Checksum: "ghi789",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Create test symbols with different kinds
	testSymbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "User",
			Kind:      "class",
			Signature: "type User struct",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "GetUser",
			Kind:      "function",
			Signature: "func GetUser()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Admin",
			Kind:      "class",
			Signature: "type Admin struct",
			StartLine: 25,
			EndLine:   30,
			StartByte: 250,
			EndByte:   300,
		},
	}

	for _, s := range testSymbols {
		err = symbolRepo.Create(ctx, s)
		if err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}
		defer symbolRepo.Delete(ctx, s.SymbolID)
	}

	// Get class symbols
	classSymbols, err := symbolRepo.GetByKind(ctx, file.FileID, "class")
	if err != nil {
		t.Fatalf("Failed to get class symbols: %v", err)
	}

	if len(classSymbols) != 2 {
		t.Errorf("Expected 2 class symbols, got %d", len(classSymbols))
	}

	// Get function symbols
	funcSymbols, err := symbolRepo.GetByKind(ctx, file.FileID, "function")
	if err != nil {
		t.Fatalf("Failed to get function symbols: %v", err)
	}

	if len(funcSymbols) != 1 {
		t.Errorf("Expected 1 function symbol, got %d", len(funcSymbols))
	}
}

func TestSymbolRepository_GetByName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-getbyname",
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
		Path:     "src/handlers.go",
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
	testSymbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "HandleRequest",
			Kind:      "function",
			Signature: "func HandleRequest()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "HandleResponse",
			Kind:      "function",
			Signature: "func HandleResponse()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
		},
	}

	for _, s := range testSymbols {
		err = symbolRepo.Create(ctx, s)
		if err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}
		defer symbolRepo.Delete(ctx, s.SymbolID)
	}

	// Search by name pattern
	symbols, err := symbolRepo.GetByName(ctx, "Handle%")
	if err != nil {
		t.Fatalf("Failed to get symbols by name: %v", err)
	}

	if len(symbols) < 2 {
		t.Errorf("Expected at least 2 symbols, got %d", len(symbols))
	}
}

func TestSymbolRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-update",
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
		Checksum: "original",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Create test symbol
	symbol := &Symbol{
		SymbolID:        uuid.New().String(),
		FileID:          file.FileID,
		Name:            "main",
		Kind:            "function",
		Signature:       "func main()",
		StartLine:       10,
		EndLine:         20,
		StartByte:       100,
		EndByte:         200,
		Docstring:       "Original docstring",
		SemanticSummary: "Original summary",
	}

	err = symbolRepo.Create(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}
	defer symbolRepo.Delete(ctx, symbol.SymbolID)

	// Update the symbol
	symbol.Signature = "func main() error"
	symbol.EndLine = 25
	symbol.EndByte = 250
	symbol.Docstring = "Updated docstring"
	symbol.SemanticSummary = "Updated summary"

	err = symbolRepo.Update(ctx, symbol)
	if err != nil {
		t.Fatalf("Failed to update symbol: %v", err)
	}

	// Verify the update
	retrieved, err := symbolRepo.GetByID(ctx, symbol.SymbolID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated symbol: %v", err)
	}

	if retrieved.Signature != "func main() error" {
		t.Errorf("Expected signature 'func main() error', got %s", retrieved.Signature)
	}
	if retrieved.EndLine != 25 {
		t.Errorf("Expected end line 25, got %d", retrieved.EndLine)
	}
	if retrieved.Docstring != "Updated docstring" {
		t.Errorf("Expected docstring 'Updated docstring', got %s", retrieved.Docstring)
	}
}

func TestSymbolRepository_BatchCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)

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
		Size:     2048,
		Checksum: "batch123",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Create test symbols
	testSymbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Symbol1",
			Kind:      "function",
			Signature: "func Symbol1()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Symbol2",
			Kind:      "function",
			Signature: "func Symbol2()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Symbol3",
			Kind:      "class",
			Signature: "type Symbol3 struct",
			StartLine: 25,
			EndLine:   30,
			StartByte: 250,
			EndByte:   300,
		},
	}

	// Batch create
	err = symbolRepo.BatchCreate(ctx, testSymbols)
	if err != nil {
		t.Fatalf("Failed to batch create symbols: %v", err)
	}

	// Clean up
	for _, s := range testSymbols {
		defer symbolRepo.Delete(ctx, s.SymbolID)
	}

	// Verify all symbols were created
	symbols, err := symbolRepo.GetByFileID(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to get symbols: %v", err)
	}

	if len(symbols) != len(testSymbols) {
		t.Errorf("Expected %d symbols, got %d", len(testSymbols), len(symbols))
	}
}

func TestSymbolRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-count",
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
		Path:     "src/count.go",
		Language: "go",
		Size:     512,
		Checksum: "count123",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Get initial count
	initialCount, err := symbolRepo.Count(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	// Create test symbols
	testSymbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Count1",
			Kind:      "function",
			Signature: "func Count1()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Count2",
			Kind:      "function",
			Signature: "func Count2()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
		},
	}

	for _, s := range testSymbols {
		err = symbolRepo.Create(ctx, s)
		if err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}
		defer symbolRepo.Delete(ctx, s.SymbolID)
	}

	// Get new count
	newCount, err := symbolRepo.Count(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	if newCount != initialCount+int64(len(testSymbols)) {
		t.Errorf("Expected count %d, got %d", initialCount+int64(len(testSymbols)), newCount)
	}
}

func TestSymbolRepository_CountByKind(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)
	symbolRepo := NewSymbolRepository(testDB.DB)

	// Create test repository and file
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-countbykind",
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
		Path:     "src/kinds.go",
		Language: "go",
		Size:     1024,
		Checksum: "kinds123",
	}
	err = fileRepo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer fileRepo.Delete(ctx, file.FileID)

	// Create test symbols with different kinds
	testSymbols := []*Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Func1",
			Kind:      "function",
			Signature: "func Func1()",
			StartLine: 5,
			EndLine:   10,
			StartByte: 50,
			EndByte:   100,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Func2",
			Kind:      "function",
			Signature: "func Func2()",
			StartLine: 15,
			EndLine:   20,
			StartByte: 150,
			EndByte:   200,
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    file.FileID,
			Name:      "Class1",
			Kind:      "class",
			Signature: "type Class1 struct",
			StartLine: 25,
			EndLine:   30,
			StartByte: 250,
			EndByte:   300,
		},
	}

	for _, s := range testSymbols {
		err = symbolRepo.Create(ctx, s)
		if err != nil {
			t.Fatalf("Failed to create symbol: %v", err)
		}
		defer symbolRepo.Delete(ctx, s.SymbolID)
	}

	// Get count by kind
	counts, err := symbolRepo.CountByKind(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to get count by kind: %v", err)
	}

	if counts["function"] != 2 {
		t.Errorf("Expected 2 functions, got %d", counts["function"])
	}
	if counts["class"] != 1 {
		t.Errorf("Expected 1 class, got %d", counts["class"])
	}
}
