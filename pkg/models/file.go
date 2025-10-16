package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// File represents a file entity in the knowledge graph
type File struct {
	FileID    string    `json:"file_id" db:"file_id"`
	RepoID    string    `json:"repo_id" db:"repo_id"`
	Path      string    `json:"path" db:"path"`
	Language  string    `json:"language" db:"language"`
	Size      int64     `json:"size" db:"size"`
	Checksum  string    `json:"checksum" db:"checksum"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// FileRepository handles CRUD operations for files
type FileRepository struct {
	db *DB
}

// NewFileRepository creates a new file repository
func NewFileRepository(db *DB) *FileRepository {
	return &FileRepository{db: db}
}

// Create inserts a new file record
func (r *FileRepository) Create(ctx context.Context, file *File) error {
	query := `
		INSERT INTO files (file_id, repo_id, path, language, size, checksum, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	now := time.Now()
	file.CreatedAt = now
	file.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		file.FileID, file.RepoID, file.Path, file.Language,
		file.Size, file.Checksum, file.CreatedAt, file.UpdatedAt)
	return err
}

// GetByID retrieves a file by its ID
func (r *FileRepository) GetByID(ctx context.Context, fileID string) (*File, error) {
	query := `
		SELECT file_id, repo_id, path, language, size, checksum, created_at, updated_at
		FROM files WHERE file_id = $1
	`
	var file File
	err := r.db.QueryRowContext(ctx, query, fileID).Scan(
		&file.FileID, &file.RepoID, &file.Path, &file.Language,
		&file.Size, &file.Checksum, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// GetByPath retrieves a file by repository ID and path
func (r *FileRepository) GetByPath(ctx context.Context, repoID, path string) (*File, error) {
	query := `
		SELECT file_id, repo_id, path, language, size, checksum, created_at, updated_at
		FROM files WHERE repo_id = $1 AND path = $2
	`
	var file File
	err := r.db.QueryRowContext(ctx, query, repoID, path).Scan(
		&file.FileID, &file.RepoID, &file.Path, &file.Language,
		&file.Size, &file.Checksum, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// GetByRepoID retrieves all files for a repository
func (r *FileRepository) GetByRepoID(ctx context.Context, repoID string) ([]*File, error) {
	query := `
		SELECT file_id, repo_id, path, language, size, checksum, created_at, updated_at
		FROM files WHERE repo_id = $1 ORDER BY path
	`
	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var file File
		err := rows.Scan(
			&file.FileID, &file.RepoID, &file.Path, &file.Language,
			&file.Size, &file.Checksum, &file.CreatedAt, &file.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, &file)
	}
	return files, rows.Err()
}

// Update updates an existing file record
func (r *FileRepository) Update(ctx context.Context, file *File) error {
	query := `
		UPDATE files 
		SET language = $3, size = $4, checksum = $5, updated_at = $6
		WHERE file_id = $1 AND repo_id = $2
	`
	file.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query,
		file.FileID, file.RepoID, file.Language, file.Size,
		file.Checksum, file.UpdatedAt)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("file not found: %s", file.FileID)
	}
	return nil
}

// Delete removes a file record
func (r *FileRepository) Delete(ctx context.Context, fileID string) error {
	query := `DELETE FROM files WHERE file_id = $1`
	result, err := r.db.ExecContext(ctx, query, fileID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("file not found: %s", fileID)
	}
	return nil
}

// BatchCreate inserts multiple files with ON CONFLICT handling for incremental updates
func (r *FileRepository) BatchCreate(ctx context.Context, files []*File) error {
	if len(files) == 0 {
		return nil
	}

	query := `
		INSERT INTO files (file_id, repo_id, path, language, size, checksum, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (repo_id, path) 
		DO UPDATE SET 
			language = EXCLUDED.language,
			size = EXCLUDED.size,
			checksum = EXCLUDED.checksum,
			updated_at = EXCLUDED.updated_at
		WHERE files.checksum != EXCLUDED.checksum
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, file := range files {
		file.CreatedAt = now
		file.UpdatedAt = now
		_, err := stmt.ExecContext(ctx,
			file.FileID, file.RepoID, file.Path, file.Language,
			file.Size, file.Checksum, file.CreatedAt, file.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert file %s: %w", file.Path, err)
		}
	}

	return nil
}

// BatchCreateTx inserts multiple files within a transaction
func (r *FileRepository) BatchCreateTx(ctx context.Context, tx *sql.Tx, files []*File) error {
	if len(files) == 0 {
		return nil
	}

	query := `
		INSERT INTO files (file_id, repo_id, path, language, size, checksum, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (repo_id, path) 
		DO UPDATE SET 
			language = EXCLUDED.language,
			size = EXCLUDED.size,
			checksum = EXCLUDED.checksum,
			updated_at = EXCLUDED.updated_at
		WHERE files.checksum != EXCLUDED.checksum
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, file := range files {
		file.CreatedAt = now
		file.UpdatedAt = now
		_, err := stmt.ExecContext(ctx,
			file.FileID, file.RepoID, file.Path, file.Language,
			file.Size, file.Checksum, file.CreatedAt, file.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert file %s: %w", file.Path, err)
		}
	}

	return nil
}

// GetChangedFiles returns files that have different checksums (for incremental updates)
func (r *FileRepository) GetChangedFiles(ctx context.Context, repoID string, fileChecksums map[string]string) ([]*File, error) {
	if len(fileChecksums) == 0 {
		return nil, nil
	}

	// Build arrays for the query
	paths := make([]string, 0, len(fileChecksums))
	checksums := make([]string, 0, len(fileChecksums))
	for path, checksum := range fileChecksums {
		paths = append(paths, path)
		checksums = append(checksums, checksum)
	}

	query := `
		SELECT file_id, repo_id, path, language, size, checksum, created_at, updated_at
		FROM files 
		WHERE repo_id = $1 
		AND path = ANY($2) 
		AND checksum != ANY($3)
		ORDER BY path
	`

	rows, err := r.db.QueryContext(ctx, query, repoID, pq.Array(paths), pq.Array(checksums))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var file File
		err := rows.Scan(
			&file.FileID, &file.RepoID, &file.Path, &file.Language,
			&file.Size, &file.Checksum, &file.CreatedAt, &file.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, &file)
	}
	return files, rows.Err()
}

// GetFilesByLanguage retrieves files filtered by language
func (r *FileRepository) GetFilesByLanguage(ctx context.Context, repoID, language string) ([]*File, error) {
	query := `
		SELECT file_id, repo_id, path, language, size, checksum, created_at, updated_at
		FROM files 
		WHERE repo_id = $1 AND language = $2 
		ORDER BY path
	`
	rows, err := r.db.QueryContext(ctx, query, repoID, language)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var file File
		err := rows.Scan(
			&file.FileID, &file.RepoID, &file.Path, &file.Language,
			&file.Size, &file.Checksum, &file.CreatedAt, &file.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, &file)
	}
	return files, rows.Err()
}

// Count returns the total number of files for a repository
func (r *FileRepository) Count(ctx context.Context, repoID string) (int64, error) {
	query := `SELECT COUNT(*) FROM files WHERE repo_id = $1`
	var count int64
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	return count, err
}