package models

import (
	"database/sql"
	"time"
)

// Repository represents a code repository
type Repository struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// File represents a file in a repository
type File struct {
	ID           int       `json:"id"`
	RepositoryID int       `json:"repository_id"`
	Path         string    `json:"path"`
	Content      string    `json:"content"`
	Language     string    `json:"language"`
	Size         int       `json:"size"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Commit represents a Git commit
type Commit struct {
	ID           int       `json:"id"`
	RepositoryID int       `json:"repository_id"`
	Hash         string    `json:"hash"`
	Author       string    `json:"author"`
	Email        string    `json:"email"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateRepository creates a new repository in the database
func (db *DB) CreateRepository(name, url string) (*Repository, error) {
	var repo Repository
	err := db.QueryRow(
		"INSERT INTO repositories (name, url) VALUES ($1, $2) RETURNING id, name, url, created_at, updated_at",
		name, url,
	).Scan(&repo.ID, &repo.Name, &repo.URL, &repo.CreatedAt, &repo.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &repo, nil
}

// GetRepositoryByID retrieves a repository by its ID
func (db *DB) GetRepositoryByID(id int) (*Repository, error) {
	var repo Repository
	err := db.QueryRow(
		"SELECT id, name, url, created_at, updated_at FROM repositories WHERE id = $1",
		id,
	).Scan(&repo.ID, &repo.Name, &repo.URL, &repo.CreatedAt, &repo.UpdatedAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	return &repo, nil
}

// CreateFile creates a new file in the database
func (db *DB) CreateFile(repoID int, path, content, language string, size int) (*File, error) {
	var file File
	err := db.QueryRow(
		"INSERT INTO files (repository_id, path, content, language, size) VALUES ($1, $2, $3, $4, $5) RETURNING id, repository_id, path, content, language, size, created_at, updated_at",
		repoID, path, content, language, size,
	).Scan(&file.ID, &file.RepositoryID, &file.Path, &file.Content, &file.Language, &file.Size, &file.CreatedAt, &file.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &file, nil
}

// CreateCommit creates a new commit in the database
func (db *DB) CreateCommit(repoID int, hash, author, email, message string, timestamp time.Time) (*Commit, error) {
	var commit Commit
	err := db.QueryRow(
		"INSERT INTO commits (repository_id, hash, author, email, message, timestamp) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, repository_id, hash, author, email, message, timestamp, created_at",
		repoID, hash, author, email, message, timestamp,
	).Scan(&commit.ID, &commit.RepositoryID, &commit.Hash, &commit.Author, &commit.Email, &commit.Message, &commit.Timestamp, &commit.CreatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &commit, nil
}