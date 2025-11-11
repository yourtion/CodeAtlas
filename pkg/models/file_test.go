package models

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestFileRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)

	// Create test repository first
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-file",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repoRepo.Delete(ctx, repo.RepoID)

	// Create test file
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

	// Verify the file was created
	retrieved, err := fileRepo.GetByID(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to retrieve file: %v", err)
	}

	if retrieved == nil {
		t.Fatal("File not found")
	}

	if retrieved.Path != file.Path {
		t.Errorf("Expected path %s, got %s", file.Path, retrieved.Path)
	}
	if retrieved.Language != file.Language {
		t.Errorf("Expected language %s, got %s", file.Language, retrieved.Language)
	}
	if retrieved.Size != file.Size {
		t.Errorf("Expected size %d, got %d", file.Size, retrieved.Size)
	}
	if retrieved.Checksum != file.Checksum {
		t.Errorf("Expected checksum %s, got %s", file.Checksum, retrieved.Checksum)
	}
}

func TestFileRepository_GetByPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)

	// Create test repository
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-getbypath",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repoRepo.Delete(ctx, repo.RepoID)

	// Create test file
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

	// Retrieve by path
	retrieved, err := fileRepo.GetByPath(ctx, repo.RepoID, "src/utils.go")
	if err != nil {
		t.Fatalf("Failed to retrieve file by path: %v", err)
	}

	if retrieved == nil {
		t.Fatal("File not found by path")
	}

	if retrieved.FileID != file.FileID {
		t.Errorf("Expected file ID %s, got %s", file.FileID, retrieved.FileID)
	}
}

func TestFileRepository_GetByRepoID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)

	// Create test repository
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-getbyrepoid",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repoRepo.Delete(ctx, repo.RepoID)

	// Create test files
	testFiles := []*File{
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "src/main.go",
			Language: "go",
			Size:     1024,
			Checksum: "abc123",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "src/utils.go",
			Language: "go",
			Size:     512,
			Checksum: "def456",
		},
	}

	for _, f := range testFiles {
		err = fileRepo.Create(ctx, f)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		defer fileRepo.Delete(ctx, f.FileID)
	}

	// Get all files for repository
	files, err := fileRepo.GetByRepoID(ctx, repo.RepoID)
	if err != nil {
		t.Fatalf("Failed to get files by repo ID: %v", err)
	}

	if len(files) != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), len(files))
	}

	// Verify files are sorted by path
	for i := 1; i < len(files); i++ {
		if files[i-1].Path > files[i].Path {
			t.Error("Files are not sorted by path")
			break
		}
	}
}

func TestFileRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)

	// Create test repository
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

	// Create test file
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

	// Update the file
	file.Size = 2048
	file.Checksum = "updated"

	err = fileRepo.Update(ctx, file)
	if err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	// Verify the update
	retrieved, err := fileRepo.GetByID(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated file: %v", err)
	}

	if retrieved.Size != 2048 {
		t.Errorf("Expected size 2048, got %d", retrieved.Size)
	}
	if retrieved.Checksum != "updated" {
		t.Errorf("Expected checksum 'updated', got %s", retrieved.Checksum)
	}
}

func TestFileRepository_BatchCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)

	// Create test repository
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

	// Create test files
	testFiles := []*File{
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "src/file1.go",
			Language: "go",
			Size:     100,
			Checksum: "check1",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "src/file2.go",
			Language: "go",
			Size:     200,
			Checksum: "check2",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "src/file3.go",
			Language: "go",
			Size:     300,
			Checksum: "check3",
		},
	}

	// Batch create
	err = fileRepo.BatchCreate(ctx, testFiles)
	if err != nil {
		t.Fatalf("Failed to batch create files: %v", err)
	}

	// Clean up
	for _, f := range testFiles {
		defer fileRepo.Delete(ctx, f.FileID)
	}

	// Verify all files were created
	files, err := fileRepo.GetByRepoID(ctx, repo.RepoID)
	if err != nil {
		t.Fatalf("Failed to get files: %v", err)
	}

	if len(files) != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), len(files))
	}
}

func TestFileRepository_GetFilesByLanguage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)

	// Create test repository
	repo := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-language",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err := repoRepo.Create(ctx, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repoRepo.Delete(ctx, repo.RepoID)

	// Create test files with different languages
	testFiles := []*File{
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "src/main.go",
			Language: "go",
			Size:     1024,
			Checksum: "go1",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "src/utils.go",
			Language: "go",
			Size:     512,
			Checksum: "go2",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "src/main.py",
			Language: "python",
			Size:     256,
			Checksum: "py1",
		},
	}

	for _, f := range testFiles {
		err = fileRepo.Create(ctx, f)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		defer fileRepo.Delete(ctx, f.FileID)
	}

	// Get Go files
	goFiles, err := fileRepo.GetFilesByLanguage(ctx, repo.RepoID, "go")
	if err != nil {
		t.Fatalf("Failed to get Go files: %v", err)
	}

	if len(goFiles) != 2 {
		t.Errorf("Expected 2 Go files, got %d", len(goFiles))
	}

	// Get Python files
	pyFiles, err := fileRepo.GetFilesByLanguage(ctx, repo.RepoID, "python")
	if err != nil {
		t.Fatalf("Failed to get Python files: %v", err)
	}

	if len(pyFiles) != 1 {
		t.Errorf("Expected 1 Python file, got %d", len(pyFiles))
	}
}

func TestFileRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repoRepo := NewRepositoryRepository(testDB.DB)
	fileRepo := NewFileRepository(testDB.DB)

	// Create test repository
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

	// Get initial count
	initialCount, err := fileRepo.Count(ctx, repo.RepoID)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	// Create test files
	testFiles := []*File{
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "file1.go",
			Language: "go",
			Size:     100,
			Checksum: "c1",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repo.RepoID,
			Path:     "file2.go",
			Language: "go",
			Size:     200,
			Checksum: "c2",
		},
	}

	for _, f := range testFiles {
		err = fileRepo.Create(ctx, f)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		defer fileRepo.Delete(ctx, f.FileID)
	}

	// Get new count
	newCount, err := fileRepo.Count(ctx, repo.RepoID)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	if newCount != initialCount+int64(len(testFiles)) {
		t.Errorf("Expected count %d, got %d", initialCount+int64(len(testFiles)), newCount)
	}
}
