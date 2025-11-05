package models

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// Integration tests for RepositoryRepository

func TestRepositoryRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	repository := &Repository{
		RepoID:     uuid.New().String(),
		Name:       "test-repo-create",
		URL:        "https://github.com/test/repo",
		Branch:     "main",
		CommitHash: "abc123",
		Metadata: map[string]interface{}{
			"language": "Go",
			"stars":    100,
		},
	}

	err := repo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Verify the repository was created
	retrieved, err := repo.GetByID(ctx, repository.RepoID)
	if err != nil {
		t.Fatalf("Failed to retrieve repository: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Repository not found")
	}

	if retrieved.Name != repository.Name {
		t.Errorf("Expected name %s, got %s", repository.Name, retrieved.Name)
	}
	if retrieved.URL != repository.URL {
		t.Errorf("Expected URL %s, got %s", repository.URL, retrieved.URL)
	}
	if retrieved.Branch != repository.Branch {
		t.Errorf("Expected branch %s, got %s", repository.Branch, retrieved.Branch)
	}
	if retrieved.CommitHash != repository.CommitHash {
		t.Errorf("Expected commit hash %s, got %s", repository.CommitHash, retrieved.CommitHash)
	}

	// Verify metadata
	if retrieved.Metadata == nil {
		t.Fatal("Metadata is nil")
	}
	if retrieved.Metadata["language"] != "Go" {
		t.Errorf("Expected language 'Go', got %v", retrieved.Metadata["language"])
	}
	// JSON numbers are unmarshaled as float64
	if stars, ok := retrieved.Metadata["stars"].(float64); !ok || stars != 100 {
		t.Errorf("Expected stars 100, got %v", retrieved.Metadata["stars"])
	}

	// Cleanup
	repo.Delete(ctx, repository.RepoID)
}

func TestRepositoryRepository_GetByName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	repoName := "test-repo-getbyname-" + uuid.New().String()[:8]
	repository := &Repository{
		RepoID:     uuid.New().String(),
		Name:       repoName,
		URL:        "https://github.com/test/repo",
		Branch:     "main",
		CommitHash: "def456",
	}

	err := repo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Delete(ctx, repository.RepoID)

	// Retrieve by name
	retrieved, err := repo.GetByName(ctx, repoName)
	if err != nil {
		t.Fatalf("Failed to retrieve repository by name: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Repository not found by name")
	}

	if retrieved.RepoID != repository.RepoID {
		t.Errorf("Expected repo ID %s, got %s", repository.RepoID, retrieved.RepoID)
	}
}

func TestRepositoryRepository_GetAll(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	// Get initial count
	initialRepos, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all repositories: %v", err)
	}
	initialCount := len(initialRepos)

	// Create test repositories
	testRepos := []*Repository{
		{
			RepoID: uuid.New().String(),
			Name:   "test-repo-all-1-" + uuid.New().String()[:8],
			URL:    "https://github.com/test/repo1",
			Branch: "main",
		},
		{
			RepoID: uuid.New().String(),
			Name:   "test-repo-all-2-" + uuid.New().String()[:8],
			URL:    "https://github.com/test/repo2",
			Branch: "develop",
		},
	}

	for _, r := range testRepos {
		err = repo.Create(ctx, r)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		defer repo.Delete(ctx, r.RepoID)
	}

	// Get all repositories
	allRepos, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all repositories: %v", err)
	}

	if len(allRepos) != initialCount+len(testRepos) {
		t.Errorf("Expected %d repositories, got %d", initialCount+len(testRepos), len(allRepos))
	}

	// Verify repositories are sorted by name
	for i := 1; i < len(allRepos); i++ {
		if allRepos[i-1].Name > allRepos[i].Name {
			t.Error("Repositories are not sorted by name")
			break
		}
	}
}

func TestRepositoryRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	repository := &Repository{
		RepoID:     uuid.New().String(),
		Name:       "test-repo-update",
		URL:        "https://github.com/test/repo",
		Branch:     "main",
		CommitHash: "original",
		Metadata: map[string]interface{}{
			"version": "1.0",
		},
	}

	err := repo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Delete(ctx, repository.RepoID)

	// Update the repository
	repository.Name = "test-repo-updated"
	repository.Branch = "develop"
	repository.CommitHash = "updated"
	repository.Metadata = map[string]interface{}{
		"version": "2.0",
		"updated": true,
	}

	err = repo.Update(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to update repository: %v", err)
	}

	// Verify the update
	retrieved, err := repo.GetByID(ctx, repository.RepoID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated repository: %v", err)
	}

	if retrieved.Name != "test-repo-updated" {
		t.Errorf("Expected name 'test-repo-updated', got %s", retrieved.Name)
	}
	if retrieved.Branch != "develop" {
		t.Errorf("Expected branch 'develop', got %s", retrieved.Branch)
	}
	if retrieved.CommitHash != "updated" {
		t.Errorf("Expected commit hash 'updated', got %s", retrieved.CommitHash)
	}

	// Verify metadata was updated
	if retrieved.Metadata["version"] != "2.0" {
		t.Errorf("Expected version '2.0', got %v", retrieved.Metadata["version"])
	}
	if updated, ok := retrieved.Metadata["updated"].(bool); !ok || !updated {
		t.Errorf("Expected updated true, got %v", retrieved.Metadata["updated"])
	}
}

func TestRepositoryRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	repository := &Repository{
		RepoID: uuid.New().String(),
		Name:   "test-repo-delete",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}

	err := repo.Create(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Delete the repository
	err = repo.Delete(ctx, repository.RepoID)
	if err != nil {
		t.Fatalf("Failed to delete repository: %v", err)
	}

	// Verify the repository is gone
	retrieved, err := repo.GetByID(ctx, repository.RepoID)
	if err != nil {
		t.Fatalf("Failed to check if repository exists: %v", err)
	}
	if retrieved != nil {
		t.Error("Repository should have been deleted")
	}
}

func TestRepositoryRepository_CreateOrUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	repoID := uuid.New().String()
	repository := &Repository{
		RepoID:     repoID,
		Name:       "test-repo-upsert",
		URL:        "https://github.com/test/repo",
		Branch:     "main",
		CommitHash: "first",
	}

	// First call should create
	err := repo.CreateOrUpdate(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Delete(ctx, repository.RepoID)

	// Verify it was created
	retrieved, err := repo.GetByID(ctx, repoID)
	if err != nil {
		t.Fatalf("Failed to retrieve repository: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Repository not found after create")
	}
	if retrieved.CommitHash != "first" {
		t.Errorf("Expected commit hash 'first', got %s", retrieved.CommitHash)
	}

	// Second call should update
	repository.CommitHash = "second"
	repository.Branch = "develop"
	err = repo.CreateOrUpdate(ctx, repository)
	if err != nil {
		t.Fatalf("Failed to update repository: %v", err)
	}

	// Verify it was updated
	retrieved, err = repo.GetByID(ctx, repoID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated repository: %v", err)
	}
	if retrieved.CommitHash != "second" {
		t.Errorf("Expected commit hash 'second', got %s", retrieved.CommitHash)
	}
	if retrieved.Branch != "develop" {
		t.Errorf("Expected branch 'develop', got %s", retrieved.Branch)
	}
}

func TestRepositoryRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	// Get initial count
	initialCount, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	// Create test repositories
	testRepos := []*Repository{
		{
			RepoID: uuid.New().String(),
			Name:   "test-repo-count-1-" + uuid.New().String()[:8],
			URL:    "https://github.com/test/repo1",
			Branch: "main",
		},
		{
			RepoID: uuid.New().String(),
			Name:   "test-repo-count-2-" + uuid.New().String()[:8],
			URL:    "https://github.com/test/repo2",
			Branch: "main",
		},
	}

	for _, r := range testRepos {
		err = repo.Create(ctx, r)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		defer repo.Delete(ctx, r.RepoID)
	}

	// Get new count
	newCount, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	if newCount != initialCount+int64(len(testRepos)) {
		t.Errorf("Expected count %d, got %d", initialCount+int64(len(testRepos)), newCount)
	}
}

func TestRepositoryRepository_MetadataHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	tests := []struct {
		name     string
		metadata map[string]interface{}
	}{
		{
			name:     "nil metadata",
			metadata: nil,
		},
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
		},
		{
			name: "complex metadata",
			metadata: map[string]interface{}{
				"language":    "Go",
				"stars":       1000,
				"forks":       50,
				"private":     false,
				"tags":        []string{"backend", "api"},
				"maintainers": []string{"alice", "bob"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &Repository{
				RepoID:   uuid.New().String(),
				Name:     "test-repo-metadata-" + uuid.New().String()[:8],
				URL:      "https://github.com/test/repo",
				Branch:   "main",
				Metadata: tt.metadata,
			}

			err := repo.Create(ctx, repository)
			if err != nil {
				t.Fatalf("Failed to create repository: %v", err)
			}
			defer repo.Delete(ctx, repository.RepoID)

			// Retrieve and verify
			retrieved, err := repo.GetByID(ctx, repository.RepoID)
			if err != nil {
				t.Fatalf("Failed to retrieve repository: %v", err)
			}

			if tt.metadata == nil || len(tt.metadata) == 0 {
				// Should have empty metadata object
				if retrieved.Metadata == nil {
					t.Error("Expected empty metadata object, got nil")
				}
			} else {
				if retrieved.Metadata == nil {
					t.Fatal("Metadata is nil")
				}
				// Verify some fields
				if tt.metadata["language"] != nil {
					if retrieved.Metadata["language"] != tt.metadata["language"] {
						t.Errorf("Language mismatch: got %v, want %v", retrieved.Metadata["language"], tt.metadata["language"])
					}
				}
			}
		})
	}
}

func TestRepositoryRepository_UpdateNonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	repository := &Repository{
		RepoID: uuid.New().String(),
		Name:   "non-existent",
		URL:    "https://github.com/test/repo",
		Branch: "main",
	}

	err := repo.Update(ctx, repository)
	if err == nil {
		t.Error("Expected error when updating non-existent repository, got nil")
	}
}

func TestRepositoryRepository_DeleteNonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()
	repo := NewRepositoryRepository(testDB.DB)

	err := repo.Delete(ctx, uuid.New().String())
	if err == nil {
		t.Error("Expected error when deleting non-existent repository, got nil")
	}
}
