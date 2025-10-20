package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Symbol represents a code symbol entity in the knowledge graph
type Symbol struct {
	SymbolID        string    `json:"symbol_id" db:"symbol_id"`
	FileID          string    `json:"file_id" db:"file_id"`
	Name            string    `json:"name" db:"name"`
	Kind            string    `json:"kind" db:"kind"`
	Signature       string    `json:"signature" db:"signature"`
	StartLine       int       `json:"start_line" db:"start_line"`
	EndLine         int       `json:"end_line" db:"end_line"`
	StartByte       int       `json:"start_byte" db:"start_byte"`
	EndByte         int       `json:"end_byte" db:"end_byte"`
	Docstring       string    `json:"docstring" db:"docstring"`
	SemanticSummary string    `json:"semantic_summary" db:"semantic_summary"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// SymbolRepository handles CRUD operations for symbols
type SymbolRepository struct {
	db *DB
}

// NewSymbolRepository creates a new symbol repository
func NewSymbolRepository(db *DB) *SymbolRepository {
	return &SymbolRepository{db: db}
}

// Create inserts a new symbol record
func (r *SymbolRepository) Create(ctx context.Context, symbol *Symbol) error {
	query := `
		INSERT INTO symbols (symbol_id, file_id, name, kind, signature, start_line, end_line, 
			start_byte, end_byte, docstring, semantic_summary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	symbol.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		symbol.SymbolID, symbol.FileID, symbol.Name, symbol.Kind, symbol.Signature,
		symbol.StartLine, symbol.EndLine, symbol.StartByte, symbol.EndByte,
		symbol.Docstring, symbol.SemanticSummary, symbol.CreatedAt)
	return err
}

// GetByID retrieves a symbol by its ID
func (r *SymbolRepository) GetByID(ctx context.Context, symbolID string) (*Symbol, error) {
	query := `
		SELECT symbol_id, file_id, name, kind, signature, start_line, end_line,
			start_byte, end_byte, docstring, semantic_summary, created_at
		FROM symbols WHERE symbol_id = $1
	`
	var symbol Symbol
	err := r.db.QueryRowContext(ctx, query, symbolID).Scan(
		&symbol.SymbolID, &symbol.FileID, &symbol.Name, &symbol.Kind, &symbol.Signature,
		&symbol.StartLine, &symbol.EndLine, &symbol.StartByte, &symbol.EndByte,
		&symbol.Docstring, &symbol.SemanticSummary, &symbol.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &symbol, nil
}

// GetByFileID retrieves all symbols for a file
func (r *SymbolRepository) GetByFileID(ctx context.Context, fileID string) ([]*Symbol, error) {
	query := `
		SELECT symbol_id, file_id, name, kind, signature, start_line, end_line,
			start_byte, end_byte, docstring, semantic_summary, created_at
		FROM symbols WHERE file_id = $1 ORDER BY start_line, start_byte
	`
	rows, err := r.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []*Symbol
	for rows.Next() {
		var symbol Symbol
		err := rows.Scan(
			&symbol.SymbolID, &symbol.FileID, &symbol.Name, &symbol.Kind, &symbol.Signature,
			&symbol.StartLine, &symbol.EndLine, &symbol.StartByte, &symbol.EndByte,
			&symbol.Docstring, &symbol.SemanticSummary, &symbol.CreatedAt)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, &symbol)
	}
	return symbols, rows.Err()
}

// GetByKind retrieves symbols filtered by kind
func (r *SymbolRepository) GetByKind(ctx context.Context, fileID, kind string) ([]*Symbol, error) {
	query := `
		SELECT symbol_id, file_id, name, kind, signature, start_line, end_line,
			start_byte, end_byte, docstring, semantic_summary, created_at
		FROM symbols WHERE file_id = $1 AND kind = $2 ORDER BY start_line, start_byte
	`
	rows, err := r.db.QueryContext(ctx, query, fileID, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []*Symbol
	for rows.Next() {
		var symbol Symbol
		err := rows.Scan(
			&symbol.SymbolID, &symbol.FileID, &symbol.Name, &symbol.Kind, &symbol.Signature,
			&symbol.StartLine, &symbol.EndLine, &symbol.StartByte, &symbol.EndByte,
			&symbol.Docstring, &symbol.SemanticSummary, &symbol.CreatedAt)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, &symbol)
	}
	return symbols, rows.Err()
}

// GetByName searches symbols by name pattern
func (r *SymbolRepository) GetByName(ctx context.Context, namePattern string) ([]*Symbol, error) {
	query := `
		SELECT symbol_id, file_id, name, kind, signature, start_line, end_line,
			start_byte, end_byte, docstring, semantic_summary, created_at
		FROM symbols WHERE name ILIKE $1 ORDER BY name
	`
	rows, err := r.db.QueryContext(ctx, query, namePattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []*Symbol
	for rows.Next() {
		var symbol Symbol
		err := rows.Scan(
			&symbol.SymbolID, &symbol.FileID, &symbol.Name, &symbol.Kind, &symbol.Signature,
			&symbol.StartLine, &symbol.EndLine, &symbol.StartByte, &symbol.EndByte,
			&symbol.Docstring, &symbol.SemanticSummary, &symbol.CreatedAt)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, &symbol)
	}
	return symbols, rows.Err()
}

// Update updates an existing symbol record
func (r *SymbolRepository) Update(ctx context.Context, symbol *Symbol) error {
	query := `
		UPDATE symbols 
		SET name = $3, kind = $4, signature = $5, start_line = $6, end_line = $7,
			start_byte = $8, end_byte = $9, docstring = $10, semantic_summary = $11
		WHERE symbol_id = $1 AND file_id = $2
	`
	result, err := r.db.ExecContext(ctx, query,
		symbol.SymbolID, symbol.FileID, symbol.Name, symbol.Kind, symbol.Signature,
		symbol.StartLine, symbol.EndLine, symbol.StartByte, symbol.EndByte,
		symbol.Docstring, symbol.SemanticSummary)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("symbol not found: %s", symbol.SymbolID)
	}
	return nil
}

// Delete removes a symbol record
func (r *SymbolRepository) Delete(ctx context.Context, symbolID string) error {
	query := `DELETE FROM symbols WHERE symbol_id = $1`
	result, err := r.db.ExecContext(ctx, query, symbolID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("symbol not found: %s", symbolID)
	}
	return nil
}

// BatchCreate inserts multiple symbols with ON CONFLICT handling
func (r *SymbolRepository) BatchCreate(ctx context.Context, symbols []*Symbol) error {
	if len(symbols) == 0 {
		return nil
	}

	query := `
		INSERT INTO symbols (symbol_id, file_id, name, kind, signature, start_line, end_line,
			start_byte, end_byte, docstring, semantic_summary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (file_id, name, start_line, start_byte) 
		DO UPDATE SET 
			kind = EXCLUDED.kind,
			signature = EXCLUDED.signature,
			end_line = EXCLUDED.end_line,
			end_byte = EXCLUDED.end_byte,
			docstring = EXCLUDED.docstring,
			semantic_summary = EXCLUDED.semantic_summary
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, symbol := range symbols {
		symbol.CreatedAt = now
		_, err := stmt.ExecContext(ctx,
			symbol.SymbolID, symbol.FileID, symbol.Name, symbol.Kind, symbol.Signature,
			symbol.StartLine, symbol.EndLine, symbol.StartByte, symbol.EndByte,
			symbol.Docstring, symbol.SemanticSummary, symbol.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert symbol %s: %w", symbol.Name, err)
		}
	}

	return nil
}

// BatchCreateTx inserts multiple symbols within a transaction
func (r *SymbolRepository) BatchCreateTx(ctx context.Context, tx *sql.Tx, symbols []*Symbol) error {
	if len(symbols) == 0 {
		return nil
	}

	query := `
		INSERT INTO symbols (symbol_id, file_id, name, kind, signature, start_line, end_line,
			start_byte, end_byte, docstring, semantic_summary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (file_id, name, start_line, start_byte) 
		DO UPDATE SET 
			kind = EXCLUDED.kind,
			signature = EXCLUDED.signature,
			end_line = EXCLUDED.end_line,
			end_byte = EXCLUDED.end_byte,
			docstring = EXCLUDED.docstring,
			semantic_summary = EXCLUDED.semantic_summary
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, symbol := range symbols {
		symbol.CreatedAt = now
		_, err := stmt.ExecContext(ctx,
			symbol.SymbolID, symbol.FileID, symbol.Name, symbol.Kind, symbol.Signature,
			symbol.StartLine, symbol.EndLine, symbol.StartByte, symbol.EndByte,
			symbol.Docstring, symbol.SemanticSummary, symbol.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert symbol %s: %w", symbol.Name, err)
		}
	}

	return nil
}

// DeleteByFileID removes all symbols for a file
func (r *SymbolRepository) DeleteByFileID(ctx context.Context, fileID string) error {
	query := `DELETE FROM symbols WHERE file_id = $1`
	_, err := r.db.ExecContext(ctx, query, fileID)
	return err
}

// GetSymbolsWithDocstrings retrieves symbols that have docstrings
func (r *SymbolRepository) GetSymbolsWithDocstrings(ctx context.Context, fileID string) ([]*Symbol, error) {
	query := `
		SELECT symbol_id, file_id, name, kind, signature, start_line, end_line,
			start_byte, end_byte, docstring, semantic_summary, created_at
		FROM symbols 
		WHERE file_id = $1 AND docstring IS NOT NULL AND docstring != ''
		ORDER BY start_line, start_byte
	`
	rows, err := r.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []*Symbol
	for rows.Next() {
		var symbol Symbol
		err := rows.Scan(
			&symbol.SymbolID, &symbol.FileID, &symbol.Name, &symbol.Kind, &symbol.Signature,
			&symbol.StartLine, &symbol.EndLine, &symbol.StartByte, &symbol.EndByte,
			&symbol.Docstring, &symbol.SemanticSummary, &symbol.CreatedAt)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, &symbol)
	}
	return symbols, rows.Err()
}

// GetSymbolsByKinds retrieves symbols filtered by multiple kinds
func (r *SymbolRepository) GetSymbolsByKinds(ctx context.Context, fileID string, kinds []string) ([]*Symbol, error) {
	if len(kinds) == 0 {
		return nil, nil
	}

	query := `
		SELECT symbol_id, file_id, name, kind, signature, start_line, end_line,
			start_byte, end_byte, docstring, semantic_summary, created_at
		FROM symbols 
		WHERE file_id = $1 AND kind = ANY($2)
		ORDER BY start_line, start_byte
	`
	rows, err := r.db.QueryContext(ctx, query, fileID, pq.Array(kinds))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []*Symbol
	for rows.Next() {
		var symbol Symbol
		err := rows.Scan(
			&symbol.SymbolID, &symbol.FileID, &symbol.Name, &symbol.Kind, &symbol.Signature,
			&symbol.StartLine, &symbol.EndLine, &symbol.StartByte, &symbol.EndByte,
			&symbol.Docstring, &symbol.SemanticSummary, &symbol.CreatedAt)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, &symbol)
	}
	return symbols, rows.Err()
}

// Count returns the total number of symbols for a file
func (r *SymbolRepository) Count(ctx context.Context, fileID string) (int64, error) {
	query := `SELECT COUNT(*) FROM symbols WHERE file_id = $1`
	var count int64
	err := r.db.QueryRowContext(ctx, query, fileID).Scan(&count)
	return count, err
}

// CountByKind returns the count of symbols by kind for a file
func (r *SymbolRepository) CountByKind(ctx context.Context, fileID string) (map[string]int64, error) {
	query := `
		SELECT kind, COUNT(*) 
		FROM symbols 
		WHERE file_id = $1 
		GROUP BY kind
	`
	rows, err := r.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var kind string
		var count int64
		err := rows.Scan(&kind, &count)
		if err != nil {
			return nil, err
		}
		counts[kind] = count
	}
	return counts, rows.Err()
}
