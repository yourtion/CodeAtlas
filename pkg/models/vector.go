package models

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
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
	// 判断是否需要 JOIN symbols/files：任一符号/文件维度过滤非空，或显式请求详情。
	needJoin := len(filters.Kind) > 0 || filters.Language != "" || filters.RepoID != "" || filters.WithDetails

	args := []interface{}{formatVectorForPgvector(queryEmbedding)}
	argIndex := 2
	addArg := func(v interface{}) string {
		s := fmt.Sprintf("$%d", argIndex)
		args = append(args, v)
		argIndex++
		return s
	}

	// SELECT 子句：JOIN 时附带符号/文件详情，消除调用方 N+1 查询。
	selectClause := `
		SELECT
			v.vector_id,
			v.entity_id,
			v.entity_type,
			v.content,
			v.model,
			v.chunk_index,
			1 - (v.embedding <=> $1::vector) as similarity`
	if needJoin {
		selectClause += `,
			s.name,
			s.kind,
			s.signature,
			s.docstring,
			f.path as file_path,
			f.language,
			f.repo_id`
	}

	// FROM + JOIN
	fromClause := "\n\t\t\tFROM vectors v"
	if needJoin {
		// LEFT JOIN：保留无符号关联的向量（如未来文件级 embedding），
		// 过滤条件用 WHERE + IS NOT NULL 收紧。
		fromClause += `
			LEFT JOIN symbols s ON v.entity_id = s.symbol_id AND v.entity_type = 'symbol'
			LEFT JOIN files f ON s.file_id = f.file_id`
	}

	// WHERE 子句
	whereClause := "\n\t\t\tWHERE 1=1"
	if filters.EntityType != "" {
		whereClause += fmt.Sprintf(" AND v.entity_type = %s", addArg(filters.EntityType))
	}
	if len(filters.EntityTypes) > 0 {
		whereClause += fmt.Sprintf(" AND v.entity_type = ANY(%s)", addArg(pq.Array(filters.EntityTypes)))
	}
	if filters.Model != "" {
		whereClause += fmt.Sprintf(" AND v.model = %s", addArg(filters.Model))
	}
	if filters.MinSimilarity > 0 {
		whereClause += fmt.Sprintf(" AND (1 - (v.embedding <=> $1::vector)) >= %s", addArg(filters.MinSimilarity))
	}
	// kind/language/repo 过滤（JOIN 列）。过滤时要求 JOIN 命中（非 NULL）。
	if len(filters.Kind) > 0 {
		whereClause += fmt.Sprintf(" AND s.kind = ANY(%s)", addArg(pq.Array(filters.Kind)))
	}
	if filters.Language != "" {
		whereClause += fmt.Sprintf(" AND f.language = %s", addArg(filters.Language))
	}
	if filters.RepoID != "" {
		whereClause += fmt.Sprintf(" AND f.repo_id = %s", addArg(filters.RepoID))
	}

	// ORDER BY + LIMIT（过滤在 LIMIT 前应用，保证返回数满 limit）
	orderBy := "\n\t\t\tORDER BY v.embedding <=> $1::vector"
	limitClause := ""
	if filters.Limit > 0 {
		limitClause = fmt.Sprintf("\n\t\t\tLIMIT %s", addArg(filters.Limit))
	}

	query := selectClause + fromClause + whereClause + orderBy + limitClause

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*VectorSearchResult
	for rows.Next() {
		var result VectorSearchResult
		var err error
		if needJoin {
			err = rows.Scan(
				&result.VectorID, &result.EntityID, &result.EntityType,
				&result.Content, &result.Model, &result.ChunkIndex, &result.Similarity,
				&result.Name, &result.Kind, &result.Signature, &result.Docstring,
				&result.FilePath, &result.Language, &result.RepoID)
		} else {
			err = rows.Scan(
				&result.VectorID, &result.EntityID, &result.EntityType,
				&result.Content, &result.Model, &result.ChunkIndex, &result.Similarity)
		}
		if err != nil {
			return nil, err
		}
		results = append(results, &result)
	}
	return results, rows.Err()
}

// KeywordSearch 基于全文检索（content_tsv）的关键词召回。
// 查询文本先用 split_identifier 拆分驼峰/下划线，再 plainto_tsquery。
// 返回按 ts_rank 排序的结果，score 落入 Similarity 字段以便与向量结果统一处理。
// filters 的 kind/language/repo 同样在 SQL 层应用。
//
// 能力差异（与 SimilaritySearchWithFilters）：本方法仅支持 EntityType（单值），
// 不支持 EntityTypes（多值）/Model/MinSimilarity。原因是这些过滤项当前无关键词
// 检索调用方使用，且 ts_rank 与 cosine 距离的 MinSimilarity 阈值语义不同。
// 如未来需要，应在此扩展并同步更新 search_handler 的 keyword 分支。
func (r *VectorRepository) KeywordSearch(ctx context.Context, query string, filters VectorSearchFilters) ([]*VectorSearchResult, error) {
	needJoin := len(filters.Kind) > 0 || filters.Language != "" || filters.RepoID != "" || filters.WithDetails

	args := []interface{}{query}
	argIndex := 2
	addArg := func(v interface{}) string {
		s := fmt.Sprintf("$%d", argIndex)
		args = append(args, v)
		argIndex++
		return s
	}

	selectClause := `
		SELECT
			v.vector_id,
			v.entity_id,
			v.entity_type,
			v.content,
			v.model,
			v.chunk_index,
			ts_rank(v.content_tsv, plainto_tsquery('simple', split_identifier($1))) as similarity`
	if needJoin {
		selectClause += `,
			s.name,
			s.kind,
			s.signature,
			s.docstring,
			f.path as file_path,
			f.language,
			f.repo_id`
	}

	fromClause := "\n\t\t\tFROM vectors v"
	if needJoin {
		fromClause += `
			LEFT JOIN symbols s ON v.entity_id = s.symbol_id AND v.entity_type = 'symbol'
			LEFT JOIN files f ON s.file_id = f.file_id`
	}

	whereClause := "\n\t\t\tWHERE v.content_tsv @@ plainto_tsquery('simple', split_identifier($1))"
	if filters.EntityType != "" {
		whereClause += fmt.Sprintf(" AND v.entity_type = %s", addArg(filters.EntityType))
	}
	if len(filters.Kind) > 0 {
		whereClause += fmt.Sprintf(" AND s.kind = ANY(%s)", addArg(pq.Array(filters.Kind)))
	}
	if filters.Language != "" {
		whereClause += fmt.Sprintf(" AND f.language = %s", addArg(filters.Language))
	}
	if filters.RepoID != "" {
		whereClause += fmt.Sprintf(" AND f.repo_id = %s", addArg(filters.RepoID))
	}

	orderBy := "\n\t\t\tORDER BY similarity DESC"
	limitClause := ""
	if filters.Limit > 0 {
		limitClause = fmt.Sprintf("\n\t\t\tLIMIT %s", addArg(filters.Limit))
	}

	q := selectClause + fromClause + whereClause + orderBy + limitClause
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*VectorSearchResult
	for rows.Next() {
		var result VectorSearchResult
		if needJoin {
			if err := rows.Scan(
				&result.VectorID, &result.EntityID, &result.EntityType,
				&result.Content, &result.Model, &result.ChunkIndex, &result.Similarity,
				&result.Name, &result.Kind, &result.Signature, &result.Docstring,
				&result.FilePath, &result.Language, &result.RepoID); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(
				&result.VectorID, &result.EntityID, &result.EntityType,
				&result.Content, &result.Model, &result.ChunkIndex, &result.Similarity); err != nil {
				return nil, err
			}
		}
		results = append(results, &result)
	}
	return results, rows.Err()
}

// HybridSearchResult 是混合检索单条结果，融合向量相似度与关键词相关度。
type HybridSearchResult struct {
	VectorSearchResult
	VectorScore  float64 `json:"vector_score"`  // 向量相似度 [0,1]
	KeywordScore float64 `json:"keyword_score"` // 关键词 ts_rank 归一化后 [0,1]
}

// HybridSearch 同时执行向量召回与关键词召回，按加权得分重排融合。
//
// weightVector 与 weightKeyword 是两路召回的权重（归一化后使用），
// 默认建议 0.7/0.3（向量为主、关键词为辅）。两路结果分数各自归一化到 [0,1]
// 后再加权求和，避免量纲不一致导致一路压倒另一路。
//
// 若 query 为空则只走向量召回；若 embedding 为空则只走关键词召回。
func (r *VectorRepository) HybridSearch(ctx context.Context, query string, queryEmbedding []float32, filters VectorSearchFilters, weightVector, weightKeyword float64) ([]*HybridSearchResult, error) {
	// 归一化权重
	total := weightVector + weightKeyword
	if total <= 0 {
		weightVector, weightKeyword = 0.7, 0.3
	} else {
		weightVector /= total
		weightKeyword /= total
	}

	// 召回上限：取 limit 的 2 倍作为各路候选，给重排留余量
	recallLimit := filters.Limit * 2
	if recallLimit == 0 {
		recallLimit = 20
	}
	recallFilters := filters
	recallFilters.Limit = recallLimit

	merge := make(map[string]*HybridSearchResult)

	// 向量召回
	if len(queryEmbedding) > 0 {
		vecResults, err := r.SimilaritySearchWithFilters(ctx, queryEmbedding, recallFilters)
		if err != nil {
			return nil, fmt.Errorf("vector recall failed: %w", err)
		}
		vecMax := 0.0
		for _, v := range vecResults {
			if v.Similarity > vecMax {
				vecMax = v.Similarity
			}
		}
		for _, v := range vecResults {
			score := v.Similarity
			if vecMax > 0 {
				score /= vecMax // 归一化
			}
			merge[v.EntityID] = &HybridSearchResult{
				VectorSearchResult: *v, VectorScore: score,
			}
		}
	}

	// 关键词召回
	if query != "" {
		kwResults, err := r.KeywordSearch(ctx, query, recallFilters)
		if err != nil {
			return nil, fmt.Errorf("keyword recall failed: %w", err)
		}
		kwMax := 0.0
		for _, k := range kwResults {
			if k.Similarity > kwMax {
				kwMax = k.Similarity
			}
		}
		for _, k := range kwResults {
			score := k.Similarity
			if kwMax > 0 {
				score /= kwMax // 归一化
			}
			if existing, ok := merge[k.EntityID]; ok {
				existing.KeywordScore = score
				// 关键词命中时，详情以关键词结果补充（两者 JOIN 字段一致）
			} else {
				merge[k.EntityID] = &HybridSearchResult{
					VectorSearchResult: *k, KeywordScore: score,
				}
			}
		}
	}

	return fuseHybridResults(merge, weightVector, weightKeyword, filters.Limit), nil
}

// fuseHybridResults 是 HybridSearch 的纯函数核心：对每路已归一化的分数
// 做加权融合、按总分降序排序、截断到 limit。提取出来便于无 DB 单测。
//
// 输入约定：merge 中每项的 VectorScore / KeywordScore 已经各自完成
// 除以本路 max 的归一化（[0,1]）。
func fuseHybridResults(merge map[string]*HybridSearchResult, weightVector, weightKeyword float64, limit int) []*HybridSearchResult {
	results := make([]*HybridSearchResult, 0, len(merge))
	for _, h := range merge {
		h.Similarity = h.VectorScore*weightVector + h.KeywordScore*weightKeyword
		results = append(results, h)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
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

	// 符号与文件详情（JOIN 查询时填充，避免调用方再做 N+1 查询）。
	// 当查询未 JOIN symbols/files 时为零值。
	Name       string `json:"name,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Signature  string `json:"signature,omitempty"`
	Docstring  string `json:"docstring,omitempty"`
	FilePath   string `json:"file_path,omitempty"`
	Language   string `json:"language,omitempty"`
	RepoID     string `json:"repo_id,omitempty"`
}

// VectorSearchFilters represents filters for vector similarity search
type VectorSearchFilters struct {
	EntityType    string   `json:"entity_type,omitempty"`
	EntityTypes   []string `json:"entity_types,omitempty"`
	Model         string   `json:"model,omitempty"`
	MinSimilarity float64  `json:"min_similarity,omitempty"`
	Limit         int      `json:"limit,omitempty"`

	// 符号/文件维度过滤（通过 JOIN symbols/files 在 SQL 层应用，
	// 替代原本"先取 limit 再内存过滤导致结果数失真"的做法）。
	// 任一非空即触发 JOIN。
	Kind     []string `json:"kind,omitempty"`     // OR 语义
	Language string   `json:"language,omitempty"` // 精确匹配
	RepoID   string   `json:"repo_id,omitempty"`  // 精确匹配

	// WithDetails 控制是否 JOIN 并返回符号/文件详情（Name/Kind/FilePath 等）。
	// 传入 kind/language/repo 任一即隐含 WithDetails=true。
	// 显式设为 true 可在不过滤时也消除调用方的 N+1 查询。
	WithDetails bool `json:"with_details,omitempty"`
}
