package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Repository represents a code repository entity in the knowledge graph
type Repository struct {
	RepoID     string                 `json:"repo_id" db:"repo_id"`
	Name       string                 `json:"name" db:"name"`
	URL        string                 `json:"url" db:"url"`
	Branch     string                 `json:"branch" db:"branch"`
	CommitHash string                 `json:"commit_hash" db:"commit_hash"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

// RepositoryRepository handles CRUD operations for repositories
type RepositoryRepository struct {
	db *DB
}

// NewRepositoryRepository creates a new repository repository
func NewRepositoryRepository(db *DB) *RepositoryRepository {
	return &RepositoryRepository{db: db}
}

// Create inserts a new repository record
func (r *RepositoryRepository) Create(ctx context.Context, repo *Repository) error {
	query := `
		INSERT INTO repositories (repo_id, name, url, branch, commit_hash, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	now := time.Now()
	repo.CreatedAt = now
	repo.UpdatedAt = now

	// Convert metadata to JSON
	var metadataJSON []byte
	var err error
	if repo.Metadata != nil {
		metadataJSON, err = json.Marshal(repo.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		repo.RepoID, repo.Name, repo.URL, repo.Branch,
		repo.CommitHash, metadataJSON, repo.CreatedAt, repo.UpdatedAt)
	return err
}

// GetByID retrieves a repository by its ID
func (r *RepositoryRepository) GetByID(ctx context.Context, repoID string) (*Repository, error) {
	query := `
		SELECT repo_id, name, url, branch, commit_hash, metadata, created_at, updated_at
		FROM repositories WHERE repo_id = $1
	`
	var repo Repository
	var metadataJSON []byte
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(
		&repo.RepoID, &repo.Name, &repo.URL, &repo.Branch,
		&repo.CommitHash, &metadataJSON, &repo.CreatedAt, &repo.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Unmarshal metadata
	if metadataJSON != nil {
		err = json.Unmarshal(metadataJSON, &repo.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &repo, nil
}

// GetByName retrieves a repository by its name
func (r *RepositoryRepository) GetByName(ctx context.Context, name string) (*Repository, error) {
	query := `
		SELECT repo_id, name, url, branch, commit_hash, metadata, created_at, updated_at
		FROM repositories WHERE name = $1
	`
	var repo Repository
	var metadataJSON []byte
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&repo.RepoID, &repo.Name, &repo.URL, &repo.Branch,
		&repo.CommitHash, &metadataJSON, &repo.CreatedAt, &repo.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Unmarshal metadata
	if metadataJSON != nil {
		err = json.Unmarshal(metadataJSON, &repo.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &repo, nil
}

// GetAll retrieves all repositories
func (r *RepositoryRepository) GetAll(ctx context.Context) ([]*Repository, error) {
	query := `
		SELECT repo_id, name, url, branch, commit_hash, metadata, created_at, updated_at
		FROM repositories ORDER BY name
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repositories []*Repository
	for rows.Next() {
		var repo Repository
		var metadataJSON []byte
		err := rows.Scan(
			&repo.RepoID, &repo.Name, &repo.URL, &repo.Branch,
			&repo.CommitHash, &metadataJSON, &repo.CreatedAt, &repo.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal metadata
		if metadataJSON != nil {
			err = json.Unmarshal(metadataJSON, &repo.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		repositories = append(repositories, &repo)
	}
	return repositories, rows.Err()
}

// Update updates an existing repository record
func (r *RepositoryRepository) Update(ctx context.Context, repo *Repository) error {
	query := `
		UPDATE repositories 
		SET name = $2, url = $3, branch = $4, commit_hash = $5, metadata = $6, updated_at = $7
		WHERE repo_id = $1
	`
	repo.UpdatedAt = time.Now()

	// Convert metadata to JSON
	var metadataJSON []byte
	var err error
	if repo.Metadata != nil {
		metadataJSON, err = json.Marshal(repo.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.ExecContext(ctx, query,
		repo.RepoID, repo.Name, repo.URL, repo.Branch,
		repo.CommitHash, metadataJSON, repo.UpdatedAt)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("repository not found: %s", repo.RepoID)
	}
	return nil
}

// Delete removes a repository record
func (r *RepositoryRepository) Delete(ctx context.Context, repoID string) error {
	query := `DELETE FROM repositories WHERE repo_id = $1`
	result, err := r.db.ExecContext(ctx, query, repoID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("repository not found: %s", repoID)
	}
	return nil
}

// CreateOrUpdate creates a new repository or updates an existing one
func (r *RepositoryRepository) CreateOrUpdate(ctx context.Context, repo *Repository) error {
	query := `
		INSERT INTO repositories (repo_id, name, url, branch, commit_hash, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (repo_id) 
		DO UPDATE SET 
			name = EXCLUDED.name,
			url = EXCLUDED.url,
			branch = EXCLUDED.branch,
			commit_hash = EXCLUDED.commit_hash,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	repo.CreatedAt = now
	repo.UpdatedAt = now

	// Convert metadata to JSON
	var metadataJSON []byte
	var err error
	if repo.Metadata != nil {
		metadataJSON, err = json.Marshal(repo.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		repo.RepoID, repo.Name, repo.URL, repo.Branch,
		repo.CommitHash, metadataJSON, repo.CreatedAt, repo.UpdatedAt)
	return err
}

// Count returns the total number of repositories
func (r *RepositoryRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM repositories`
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}