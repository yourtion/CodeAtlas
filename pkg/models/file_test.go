package models

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFileRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create repository first
	repoRepo := NewRepositoryRepository(db)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err = repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	repo := NewFileRepository(db)

	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repository.RepoID,
		Path:     "test/file.go",
		Language: "go",
		Size:     1024,
		Checksum: "abc123",
	}

	err = repo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Verify the file was created
	retrieved, err := repo.GetByID(ctx, file.FileID)
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
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create repository first
	repoRepo := NewRepositoryRepository(db)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-path-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err = repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	repo := NewFileRepository(db)

	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repository.RepoID,
		Path:     "test/unique_path.go",
		Language: "go",
		Size:     512,
		Checksum: "def456",
	}

	err = repo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Retrieve by path
	retrieved, err := repo.GetByPath(ctx, repository.RepoID, file.Path)
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

func TestFileRepository_BatchCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create repository first
	repoRepo := NewRepositoryRepository(db)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-batch-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err = repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	repo := NewFileRepository(db)

	files := []*File{
		{
			FileID:   uuid.New().String(),
			RepoID:   repository.RepoID,
			Path:     "batch/file1.go",
			Language: "go",
			Size:     100,
			Checksum: "batch1",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repository.RepoID,
			Path:     "batch/file2.go",
			Language: "go",
			Size:     200,
			Checksum: "batch2",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repository.RepoID,
			Path:     "batch/file3.py",
			Language: "python",
			Size:     300,
			Checksum: "batch3",
		},
	}

	err = repo.BatchCreate(ctx, files)
	if err != nil {
		t.Fatalf("Failed to batch create files: %v", err)
	}

	// Verify all files were created
	retrievedFiles, err := repo.GetByRepoID(ctx, repository.RepoID)
	if err != nil {
		t.Fatalf("Failed to retrieve files by repo ID: %v", err)
	}

	if len(retrievedFiles) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(retrievedFiles))
	}

	// Verify files are sorted by path
	for i := 1; i < len(retrievedFiles); i++ {
		if retrievedFiles[i-1].Path > retrievedFiles[i].Path {
			t.Error("Files are not sorted by path")
			break
		}
	}
}

func TestFileRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create repository first
	repoRepo := NewRepositoryRepository(db)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-update-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err = repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	repo := NewFileRepository(db)

	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repository.RepoID,
		Path:     "test/update.go",
		Language: "go",
		Size:     1000,
		Checksum: "original",
	}

	err = repo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Update the file
	file.Size = 2000
	file.Checksum = "updated"
	originalUpdatedAt := file.UpdatedAt

	// Wait a bit to ensure updated_at changes
	time.Sleep(10 * time.Millisecond)

	err = repo.Update(ctx, file)
	if err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	// Verify the update
	retrieved, err := repo.GetByID(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated file: %v", err)
	}

	if retrieved.Size != 2000 {
		t.Errorf("Expected size 2000, got %d", retrieved.Size)
	}
	if retrieved.Checksum != "updated" {
		t.Errorf("Expected checksum 'updated', got %s", retrieved.Checksum)
	}
	if !retrieved.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
}

func TestFileRepository_GetFilesByLanguage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create repository first
	repoRepo := NewRepositoryRepository(db)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-lang-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err = repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	repo := NewFileRepository(db)

	files := []*File{
		{
			FileID:   uuid.New().String(),
			RepoID:   repository.RepoID,
			Path:     "lang/file1.go",
			Language: "go",
			Size:     100,
			Checksum: "go1",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repository.RepoID,
			Path:     "lang/file2.go",
			Language: "go",
			Size:     200,
			Checksum: "go2",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repository.RepoID,
			Path:     "lang/file3.py",
			Language: "python",
			Size:     300,
			Checksum: "py1",
		},
	}

	err = repo.BatchCreate(ctx, files)
	if err != nil {
		t.Fatalf("Failed to batch create files: %v", err)
	}

	// Get Go files
	goFiles, err := repo.GetFilesByLanguage(ctx, repository.RepoID, "go")
	if err != nil {
		t.Fatalf("Failed to get Go files: %v", err)
	}

	if len(goFiles) != 2 {
		t.Errorf("Expected 2 Go files, got %d", len(goFiles))
	}

	for _, file := range goFiles {
		if file.Language != "go" {
			t.Errorf("Expected language 'go', got %s", file.Language)
		}
	}

	// Get Python files
	pythonFiles, err := repo.GetFilesByLanguage(ctx, repository.RepoID, "python")
	if err != nil {
		t.Fatalf("Failed to get Python files: %v", err)
	}

	if len(pythonFiles) != 1 {
		t.Errorf("Expected 1 Python file, got %d", len(pythonFiles))
	}
}

func TestFileRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create repository first
	repoRepo := NewRepositoryRepository(db)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-count-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err = repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	repo := NewFileRepository(db)

	// Initial count should be 0
	count, err := repo.Count(ctx, repository.RepoID)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Add some files
	files := []*File{
		{
			FileID:   uuid.New().String(),
			RepoID:   repository.RepoID,
			Path:     "count/file1.go",
			Language: "go",
			Size:     100,
			Checksum: "count1",
		},
		{
			FileID:   uuid.New().String(),
			RepoID:   repository.RepoID,
			Path:     "count/file2.go",
			Language: "go",
			Size:     200,
			Checksum: "count2",
		},
	}

	err = repo.BatchCreate(ctx, files)
	if err != nil {
		t.Fatalf("Failed to batch create files: %v", err)
	}

	// Count should now be 2
	count, err = repo.Count(ctx, repository.RepoID)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestFileRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create repository first
	repoRepo := NewRepositoryRepository(db)
	repoID := uuid.New().String()
	repository := &Repository{
		RepoID: repoID,
		Name:   "test-repo-delete-" + repoID[:8],
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}
	err = repoRepo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	repo := NewFileRepository(db)

	file := &File{
		FileID:   uuid.New().String(),
		RepoID:   repository.RepoID,
		Path:     "test/delete.go",
		Language: "go",
		Size:     500,
		Checksum: "delete",
	}

	err = repo.Create(ctx, file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Delete the file
	err = repo.Delete(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Verify the file is gone
	retrieved, err := repo.GetByID(ctx, file.FileID)
	if err != nil {
		t.Fatalf("Failed to check if file exists: %v", err)
	}
	if retrieved != nil {
		t.Error("File should have been deleted")
	}
}
