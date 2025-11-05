package models

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Vector represents a vector embedding entity in the knowledge graph
type Vector struct {
	VectorID   string    `json:"vector_id" db:"vector_id"`
	EntityID   string    `json:"entity_id" db:"entity_id"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	Embedding  []float32 `json:"embedding" db:"embedding"`
	Content    string    `json:"content" db:"content"`
	Model      string    `json:"model" db:"model"`
	ChunkIndex int       `json:"chunk_index" db:"chunk_index"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// VectorRepository handles CRUD operations for vectors
type VectorRepository struct {
	db *DB
}

// NewVectorRepository creates a new vector repository
func NewVectorRepository(db *DB) *VectorRepository {
	return &VectorRepository{db: db}
}

// formatVectorForPgvector converts []float32 to pgvector format string [0.1,0.2,0.3]
func formatVectorForPgvector(embedding []float32) string {
	if len(embedding) == 0 {
		return "[]"
	}
	parts := make([]string, len(embedding))
	for i, v := range embedding {
		parts[i] = strconv.FormatFloat(float64(v), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// parseVectorFromPgvector parses pgvector format string [0.1,0.2,0.3] to []float32
func parseVectorFromPgvector(vectorStr string) ([]float32, error) {
	// Remove brackets
	vectorStr = strings.TrimPrefix(vectorStr, "[")
	vectorStr = strings.TrimSuffix(vectorStr, "]")
	
	if vectorStr == "" {
		return []float32{}, nil
	}
	
	parts := strings.Split(vectorStr, ",")
	result := make([]float32, len(parts))
	
	for i, part := range parts {
		val, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vector element %d: %w", i, err)
		}
		result[i] = float32(val)
	}
	
	return result, nil
}

// Create inserts a new vector record
func (r *VectorRepository) Create(ctx context.Context, vector *Vector) error {
	query := `
		INSERT INTO vectors (vector_id, entity_id, entity_type, embedding, content, model, chunk_index, created_at)
		VALUES ($1, $2, $3, $4::vector, $5, $6, $7, $8)
	`
	vector.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		vector.VectorID, vector.EntityID, vector.EntityType, formatVectorForPgvector(vector.Embedding),
		vector.Content, vector.Model, vector.ChunkIndex, vector.CreatedAt)
	return err
}

// GetByID retrieves a vector by its ID
func (r *VectorRepository) GetByID(ctx context.Context, vectorID string) (*Vector, error) {
	query := `
		SELECT vector_id, entity_id, entity_type, embedding::text, content, model, chunk_index, created_at
		FROM vectors WHERE vector_id = $1
	`
	var vector Vector
	var embeddingStr string
	err := r.db.QueryRowContext(ctx, query, vectorID).Scan(
		&vector.VectorID, &vector.EntityID, &vector.EntityType, &embeddingStr,
		&vector.Content, &vector.Model, &vector.ChunkIndex, &vector.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	embedding, err := parseVectorFromPgvector(embeddingStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedding: %w", err)
	}
	vector.Embedding = embedding
	return &vector, nil
}

// GetByEntityID retrieves all vectors for an entity
func (r *VectorRepository) GetByEntityID(ctx context.Context, entityID, entityType string) ([]*Vector, error) {
	query := `
		SELECT vector_id, entity_id, entity_type, embedding::text, content, model, chunk_index, created_at
		FROM vectors WHERE entity_id = $1 AND entity_type = $2 ORDER BY chunk_index
	`
	rows, err := r.db.QueryContext(ctx, query, entityID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vectors []*Vector
	for rows.Next() {
		var vector Vector
		var embeddingStr string
		err := rows.Scan(
			&vector.VectorID, &vector.EntityID, &vector.EntityType, &embeddingStr,
			&vector.Content, &vector.Model, &vector.ChunkIndex, &vector.CreatedAt)
		if err != nil {
			return nil, err
		}
		
		embedding, err := parseVectorFromPgvector(embeddingStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse embedding for vector %s: %w", vector.VectorID, err)
		}
		vector.Embedding = embedding
		vectors = append(vectors, &vector)
	}
	return vectors, rows.Err()
}

// GetByEntityType retrieves vectors filtered by entity type
func (r *VectorRepository) GetByEntityType(ctx context.Context, entityType string, limit int) ([]*Vector, error) {
	query := `
		SELECT vector_id, entity_id, entity_type, embedding::text, content, model, chunk_index, created_at
		FROM vectors WHERE entity_type = $1 ORDER BY created_at DESC LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, entityType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vectors []*Vector
	for rows.Next() {
		var vector Vector
		var embeddingStr string
		err := rows.Scan(
			&vector.VectorID, &vector.EntityID, &vector.EntityType, &embeddingStr,
			&vector.Content, &vector.Model, &vector.ChunkIndex, &vector.CreatedAt)
		if err != nil {
			return nil, err
		}
		
		embedding, err := parseVectorFromPgvector(embeddingStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse embedding for vector %s: %w", vector.VectorID, err)
		}
		vector.Embedding = embedding
		vectors = append(vectors, &vector)
	}
	return vectors, rows.Err()
}

// Update updates an existing vector record
func (r *VectorRepository) Update(ctx context.Context, vector *Vector) error {
	query := `
		UPDATE vectors 
		SET embedding = $3::vector, content = $4, model = $5, chunk_index = $6
		WHERE vector_id = $1 AND entity_id = $2
	`
	result, err := r.db.ExecContext(ctx, query,
		vector.VectorID, vector.EntityID, formatVectorForPgvector(vector.Embedding),
		vector.Content, vector.Model, vector.ChunkIndex)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("vector not found: %s", vector.VectorID)
	}
	return nil
}

// Delete removes a vector record
func (r *VectorRepository) Delete(ctx context.Context, vectorID string) error {
	query := `DELETE FROM vectors WHERE vector_id = $1`
	result, err := r.db.ExecContext(ctx, query, vectorID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("vector not found: %s", vectorID)
	}
	return nil
}

// BatchCreate inserts multiple vectors with embedding dimension validation
func (r *VectorRepository) BatchCreate(ctx context.Context, vectors []*Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	query := `
		INSERT INTO vectors (vector_id, entity_id, entity_type, embedding, content, model, chunk_index, created_at)
		VALUES ($1, $2, $3, $4::vector, $5, $6, $7, $8)
		ON CONFLICT (vector_id) 
		DO UPDATE SET 
			embedding = EXCLUDED.embedding,
			content = EXCLUDED.content,
			model = EXCLUDED.model
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, vector := range vectors {
		vector.CreatedAt = now
		_, err := stmt.ExecContext(ctx,
			vector.VectorID, vector.EntityID, vector.EntityType, formatVectorForPgvector(vector.Embedding),
			vector.Content, vector.Model, vector.ChunkIndex, vector.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert vector %s: %w", vector.VectorID, err)
		}
	}

	return nil
}

// BatchCreateTx inserts multiple vectors within a transaction
func (r *VectorRepository) BatchCreateTx(ctx context.Context, tx *sql.Tx, vectors []*Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	query := `
		INSERT INTO vectors (vector_id, entity_id, entity_type, embedding, content, model, chunk_index, created_at)
		VALUES ($1, $2, $3, $4::vector, $5, $6, $7, $8)
		ON CONFLICT (entity_id, entity_type, chunk_index) 
		DO UPDATE SET 
			embedding = EXCLUDED.embedding,
			content = EXCLUDED.content,
			model = EXCLUDED.model
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, vector := range vectors {
		vector.CreatedAt = now
		_, err := stmt.ExecContext(ctx,
			vector.VectorID, vector.EntityID, vector.EntityType, formatVectorForPgvector(vector.Embedding),
			vector.Content, vector.Model, vector.ChunkIndex, vector.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert vector %s: %w", vector.VectorID, err)
		}
	}

	return nil
}

// DeleteByEntityID removes all vectors for an entity
func (r *VectorRepository) DeleteByEntityID(ctx context.Context, entityID, entityType string) error {
	query := `DELETE FROM vectors WHERE entity_id = $1 AND entity_type = $2`
	_, err := r.db.ExecContext(ctx, query, entityID, entityType)
	return err
}

// SimilaritySearch performs vector similarity search using pgvector
func (r *VectorRepository) SimilaritySearch(ctx context.Context, queryEmbedding []float32, entityType string, limit int) ([]*VectorSearchResult, error) {
	query := `
		SELECT 
			v.vector_id,
			v.entity_id,
			v.entity_type,
			v.content,
			v.model,
			v.chunk_index,
			1 - (v.embedding <=> $1::vector) as similarity
		FROM vectors v
		WHERE v.entity_type = $2
		ORDER BY v.embedding <=> $1::vector
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, formatVectorForPgvector(queryEmbedding), entityType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*VectorSearchResult
	for rows.Next() {
		var result VectorSearchResult
		err := rows.Scan(
			&result.VectorID, &result.EntityID, &result.EntityType,
			&result.Content, &result.Model, &result.ChunkIndex, &result.Similarity)
		if err != nil {
			return nil, err
		}
		results = append(results, &result)
	}
	return results, rows.Err()
}

// SimilaritySearchWithFilters performs vector similarity search with additional filters
func (r *VectorRepository) SimilaritySearchWithFilters(ctx context.Context, queryEmbedding []float32, filters VectorSearchFilters) ([]*VectorSearchResult, error) {
	baseQuery := `
		SELECT 
			v.vector_id,
			v.entity_id,
			v.entity_type,
			v.content,
			v.model,
			v.chunk_index,
			1 - (v.embedding <=> $1::vector) as similarity
		FROM vectors v
		WHERE 1=1
	`

	args := []interface{}{formatVectorForPgvector(queryEmbedding)}
	argIndex := 2

	if filters.EntityType != "" {
		baseQuery += fmt.Sprintf(" AND v.entity_type = $%d", argIndex)
		args = append(args, filters.EntityType)
		argIndex++
	}

	if len(filters.EntityTypes) > 0 {
		baseQuery += fmt.Sprintf(" AND v.entity_type = ANY($%d)", argIndex)
		args = append(args, pq.Array(filters.EntityTypes))
		argIndex++
	}

	if filters.Model != "" {
		baseQuery += fmt.Sprintf(" AND v.model = $%d", argIndex)
		args = append(args, filters.Model)
		argIndex++
	}

	if filters.MinSimilarity > 0 {
		baseQuery += fmt.Sprintf(" AND (1 - (v.embedding <=> $1::vector)) >= $%d", argIndex)
		args = append(args, filters.MinSimilarity)
		argIndex++
	}

	baseQuery += " ORDER BY v.embedding <=> $1::vector"

	if filters.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
	}

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*VectorSearchResult
	for rows.Next() {
		var result VectorSearchResult
		err := rows.Scan(
			&result.VectorID, &result.EntityID, &result.EntityType,
			&result.Content, &result.Model, &result.ChunkIndex, &result.Similarity)
		if err != nil {
			return nil, err
		}
		results = append(results, &result)
	}
	return results, rows.Err()
}

// GetEmbeddingDimensions returns the dimensions of embeddings for a model
func (r *VectorRepository) GetEmbeddingDimensions(ctx context.Context, model string) (int, error) {
	query := `
		SELECT vector_dims(embedding) as dimensions
		FROM vectors 
		WHERE model = $1 
		LIMIT 1
	`
	var dimensions int
	err := r.db.QueryRowContext(ctx, query, model).Scan(&dimensions)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no vectors found for model: %s", model)
		}
		return 0, err
	}
	return dimensions, nil
}

// Count returns the total number of vectors
func (r *VectorRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM vectors`
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// CountByEntityType returns the count of vectors by entity type
func (r *VectorRepository) CountByEntityType(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT entity_type, COUNT(*) 
		FROM vectors 
		GROUP BY entity_type
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var entityType string
		var count int64
		err := rows.Scan(&entityType, &count)
		if err != nil {
			return nil, err
		}
		counts[entityType] = count
	}
	return counts, rows.Err()
}

// CountByModel returns the count of vectors by model
func (r *VectorRepository) CountByModel(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT model, COUNT(*) 
		FROM vectors 
		GROUP BY model
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var model string
		var count int64
		err := rows.Scan(&model, &count)
		if err != nil {
			return nil, err
		}
		counts[model] = count
	}
	return counts, rows.Err()
}

// VectorSearchResult represents a result from vector similarity search
type VectorSearchResult struct {
	VectorID   string  `json:"vector_id"`
	EntityID   string  `json:"entity_id"`
	EntityType string  `json:"entity_type"`
	Content    string  `json:"content"`
	Model      string  `json:"model"`
	ChunkIndex int     `json:"chunk_index"`
	Similarity float64 `json:"similarity"`
}

// VectorSearchFilters represents filters for vector similarity search
type VectorSearchFilters struct {
	EntityType    string   `json:"entity_type,omitempty"`
	EntityTypes   []string `json:"entity_types,omitempty"`
	Model         string   `json:"model,omitempty"`
	MinSimilarity float64  `json:"min_similarity,omitempty"`
	Limit         int      `json:"limit,omitempty"`
}
