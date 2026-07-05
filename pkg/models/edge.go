package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Edge represents a dependency edge entity in the knowledge graph
type Edge struct {
	EdgeID       string    `json:"edge_id" db:"edge_id"`
	SourceID     string    `json:"source_id" db:"source_id"`
	TargetID     *string   `json:"target_id" db:"target_id"`
	EdgeType     string    `json:"edge_type" db:"edge_type"`
	SourceFile   string    `json:"source_file" db:"source_file"`
	TargetFile   *string   `json:"target_file" db:"target_file"`
	TargetModule *string   `json:"target_module" db:"target_module"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// EdgeRepository handles CRUD operations for edges
type EdgeRepository struct {
	db *DB
}

// NewEdgeRepository creates a new edge repository
func NewEdgeRepository(db *DB) *EdgeRepository {
	return &EdgeRepository{db: db}
}

// Create inserts a new edge record
func (r *EdgeRepository) Create(ctx context.Context, edge *Edge) error {
	query := `
		INSERT INTO edges (edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	edge.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		edge.EdgeID, edge.SourceID, edge.TargetID, edge.EdgeType,
		edge.SourceFile, edge.TargetFile, edge.TargetModule, edge.CreatedAt)
	return err
}

// GetByID retrieves an edge by its ID
func (r *EdgeRepository) GetByID(ctx context.Context, edgeID string) (*Edge, error) {
	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges WHERE edge_id = $1
	`
	var edge Edge
	err := r.db.QueryRowContext(ctx, query, edgeID).Scan(
		&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
		&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &edge, nil
}

// GetBySourceID retrieves all edges originating from a source symbol
func (r *EdgeRepository) GetBySourceID(ctx context.Context, sourceID string) ([]*Edge, error) {
	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges WHERE source_id = $1 ORDER BY edge_type, created_at
	`
	rows, err := r.db.QueryContext(ctx, query, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		err := rows.Scan(
			&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
			&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, rows.Err()
}

// GetByTargetID retrieves all edges pointing to a target symbol
func (r *EdgeRepository) GetByTargetID(ctx context.Context, targetID string) ([]*Edge, error) {
	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges WHERE target_id = $1 ORDER BY edge_type, created_at
	`
	rows, err := r.db.QueryContext(ctx, query, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		err := rows.Scan(
			&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
			&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, rows.Err()
}

// GetByType retrieves edges filtered by type
func (r *EdgeRepository) GetByType(ctx context.Context, edgeType string) ([]*Edge, error) {
	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges WHERE edge_type = $1 ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query, edgeType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		err := rows.Scan(
			&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
			&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, rows.Err()
}

// GetBySourceAndType retrieves edges by source ID and type
func (r *EdgeRepository) GetBySourceAndType(ctx context.Context, sourceID, edgeType string) ([]*Edge, error) {
	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges WHERE source_id = $1 AND edge_type = $2 ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query, sourceID, edgeType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		err := rows.Scan(
			&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
			&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, rows.Err()
}

// GetByTargetAndType retrieves edges by target ID and type
func (r *EdgeRepository) GetByTargetAndType(ctx context.Context, targetID, edgeType string) ([]*Edge, error) {
	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges WHERE target_id = $1 AND edge_type = $2 ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query, targetID, edgeType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		err := rows.Scan(
			&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
			&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, rows.Err()
}

// Update updates an existing edge record
func (r *EdgeRepository) Update(ctx context.Context, edge *Edge) error {
	query := `
		UPDATE edges 
		SET target_id = $3, edge_type = $4, source_file = $5, target_file = $6, target_module = $7
		WHERE edge_id = $1 AND source_id = $2
	`
	result, err := r.db.ExecContext(ctx, query,
		edge.EdgeID, edge.SourceID, edge.TargetID, edge.EdgeType,
		edge.SourceFile, edge.TargetFile, edge.TargetModule)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("edge not found: %s", edge.EdgeID)
	}
	return nil
}

// Delete removes an edge record
func (r *EdgeRepository) Delete(ctx context.Context, edgeID string) error {
	query := `DELETE FROM edges WHERE edge_id = $1`
	result, err := r.db.ExecContext(ctx, query, edgeID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("edge not found: %s", edgeID)
	}
	return nil
}

// BatchCreate inserts multiple edges with proper foreign key handling
func (r *EdgeRepository) BatchCreate(ctx context.Context, edges []*Edge) error {
	if len(edges) == 0 {
		return nil
	}

	query := `
		INSERT INTO edges (edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (edge_id)
		DO UPDATE SET
			target_id = EXCLUDED.target_id,
			edge_type = EXCLUDED.edge_type,
			source_file = EXCLUDED.source_file,
			target_file = EXCLUDED.target_file,
			target_module = EXCLUDED.target_module
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, edge := range edges {
		edge.CreatedAt = now

		_, err := stmt.ExecContext(ctx,
			edge.EdgeID, edge.SourceID, edge.TargetID, edge.EdgeType,
			edge.SourceFile, edge.TargetFile, edge.TargetModule, edge.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert edge %s: %w", edge.EdgeID, err)
		}
	}

	return nil
}

// BatchCreateTx inserts multiple edges within a transaction
func (r *EdgeRepository) BatchCreateTx(ctx context.Context, tx *sql.Tx, edges []*Edge) error {
	if len(edges) == 0 {
		return nil
	}

	query := `
		INSERT INTO edges (edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (edge_id) 
		DO UPDATE SET 
			target_id = EXCLUDED.target_id,
			edge_type = EXCLUDED.edge_type,
			source_file = EXCLUDED.source_file,
			target_file = EXCLUDED.target_file,
			target_module = EXCLUDED.target_module
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, edge := range edges {
		edge.CreatedAt = now
		_, err := stmt.ExecContext(ctx,
			edge.EdgeID, edge.SourceID, edge.TargetID, edge.EdgeType,
			edge.SourceFile, edge.TargetFile, edge.TargetModule, edge.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert edge %s: %w", edge.EdgeID, err)
		}
	}

	return nil
}

// DeleteBySourceID removes all edges originating from a source symbol
func (r *EdgeRepository) DeleteBySourceID(ctx context.Context, sourceID string) error {
	query := `DELETE FROM edges WHERE source_id = $1`
	_, err := r.db.ExecContext(ctx, query, sourceID)
	return err
}

// DeleteByTargetID removes all edges pointing to a target symbol
func (r *EdgeRepository) DeleteByTargetID(ctx context.Context, targetID string) error {
	query := `DELETE FROM edges WHERE target_id = $1`
	_, err := r.db.ExecContext(ctx, query, targetID)
	return err
}

// GetCallRelationships retrieves call relationships (caller -> callee)
func (r *EdgeRepository) GetCallRelationships(ctx context.Context, symbolID string) ([]*Edge, error) {
	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges 
		WHERE (source_id = $1 OR target_id = $1) AND edge_type = 'call'
		ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		err := rows.Scan(
			&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
			&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, rows.Err()
}

// GetImportRelationships retrieves import relationships
func (r *EdgeRepository) GetImportRelationships(ctx context.Context, symbolID string) ([]*Edge, error) {
	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges 
		WHERE (source_id = $1 OR target_id = $1) AND edge_type = 'import'
		ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		err := rows.Scan(
			&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
			&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, rows.Err()
}

// GetInheritanceRelationships retrieves inheritance relationships (extends/implements)
func (r *EdgeRepository) GetInheritanceRelationships(ctx context.Context, symbolID string) ([]*Edge, error) {
	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges 
		WHERE (source_id = $1 OR target_id = $1) AND edge_type IN ('extends', 'implements')
		ORDER BY edge_type, created_at
	`
	rows, err := r.db.QueryContext(ctx, query, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		err := rows.Scan(
			&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
			&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, rows.Err()
}

// GetEdgesByTypes retrieves edges filtered by multiple types
func (r *EdgeRepository) GetEdgesByTypes(ctx context.Context, edgeTypes []string) ([]*Edge, error) {
	if len(edgeTypes) == 0 {
		return nil, nil
	}

	query := `
		SELECT edge_id, source_id, target_id, edge_type, source_file, target_file, target_module, created_at
		FROM edges WHERE edge_type = ANY($1) ORDER BY edge_type, created_at
	`
	rows, err := r.db.QueryContext(ctx, query, pq.Array(edgeTypes))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		err := rows.Scan(
			&edge.EdgeID, &edge.SourceID, &edge.TargetID, &edge.EdgeType,
			&edge.SourceFile, &edge.TargetFile, &edge.TargetModule, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, rows.Err()
}

// Count returns the total number of edges
func (r *EdgeRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM edges`
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// CountByType returns the count of edges by type
func (r *EdgeRepository) CountByType(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT edge_type, COUNT(*) 
		FROM edges 
		GROUP BY edge_type
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var edgeType string
		var count int64
		err := rows.Scan(&edgeType, &count)
		if err != nil {
			return nil, err
		}
		counts[edgeType] = count
	}
	return counts, rows.Err()
}

// CountBySourceID returns the count of outgoing edges for a symbol
func (r *EdgeRepository) CountBySourceID(ctx context.Context, sourceID string) (int64, error) {
	query := `SELECT COUNT(*) FROM edges WHERE source_id = $1`
	var count int64
	err := r.db.QueryRowContext(ctx, query, sourceID).Scan(&count)
	return count, err
}

// CountByTargetID returns the count of incoming edges for a symbol
func (r *EdgeRepository) CountByTargetID(ctx context.Context, targetID string) (int64, error) {
	query := `SELECT COUNT(*) FROM edges WHERE target_id = $1`
	var count int64
	err := r.db.QueryRowContext(ctx, query, targetID).Scan(&count)
	return count, err
}

// EdgeWithDetails 是带符号与文件详情的边结果，用于消除关系查询的 N+1。
// 一次 JOIN 即可取回边两端符号的 name/kind/signature 与文件路径，
// 替代原先逐条 GetByID 的 N+1 查询。
type EdgeWithDetails struct {
	EdgeID      string
	EdgeType    string
	SymbolID    string // 关联符号 ID（caller 查询里是 source，callee/dependency 是 target）
	Name        string
	Kind        string
	Signature   string
	FilePath    string
	SourceFile  string
	TargetFile  *string
	TargetModule *string
}

// edgeDetailsQuery 是 JOIN symbols/files 的公共 SQL 片段。
// symbolAlias 指定取详情的符号端（source 或 target），fileAlias 对应其文件。
const edgeDetailsColumns = `
	SELECT
		e.edge_id, e.edge_type,
		s.symbol_id, s.name, s.kind, s.signature,
		f.path,
		e.source_file, e.target_file, e.target_module
	FROM edges e
	JOIN symbols s ON s.symbol_id = %s
	LEFT JOIN files f ON s.file_id = f.file_id
	WHERE %s
	ORDER BY s.name
`

// GetCallersWithDetails 返回调用给定符号的所有符号（含详情），一次 JOIN 消除 N+1。
// caller 是边的 source，给定符号是 target。
func (r *EdgeRepository) GetCallersWithDetails(ctx context.Context, targetSymbolID string) ([]*EdgeWithDetails, error) {
	query := fmt.Sprintf(edgeDetailsColumns, "e.source_id", "e.target_id = $1 AND e.edge_type = 'call'")
	return r.queryEdgesWithDetails(ctx, query, targetSymbolID)
}

// GetCalleesWithDetails 返回给定符号调用的所有符号（含详情）。
// callee 是边的 target，给定符号是 source。
func (r *EdgeRepository) GetCalleesWithDetails(ctx context.Context, sourceSymbolID string) ([]*EdgeWithDetails, error) {
	query := fmt.Sprintf(edgeDetailsColumns, "e.target_id", "e.source_id = $1 AND e.edge_type = 'call'")
	return r.queryEdgesWithDetails(ctx, query, sourceSymbolID)
}

// GetDependenciesWithDetails 返回给定符号的依赖（import/extends/implements/reference，含详情）。
// 依赖的符号是边的 target；若边无 target_id（外部 import，仅有 target_module）则跳过。
func (r *EdgeRepository) GetDependenciesWithDetails(ctx context.Context, sourceSymbolID string) ([]*EdgeWithDetails, error) {
	query := fmt.Sprintf(`
	SELECT
		e.edge_id, e.edge_type,
		s.symbol_id, s.name, s.kind, s.signature,
		f.path,
		e.source_file, e.target_file, e.target_module
	FROM edges e
	JOIN symbols s ON s.symbol_id = e.target_id
	LEFT JOIN files f ON s.file_id = f.file_id
	WHERE e.source_id = $1
	  AND e.edge_type IN ('import', 'extends', 'implements', 'reference')
	  AND e.target_id IS NOT NULL
	ORDER BY e.edge_type, s.name
	`)
	return r.queryEdgesWithDetails(ctx, query, sourceSymbolID)
}

// GetExternalDependencies 返回给定符号的外部依赖（无 target_id，仅有 target_module，
// 如未解析的 import）。这些边无法 JOIN symbols，单独查询。
func (r *EdgeRepository) GetExternalDependencies(ctx context.Context, sourceSymbolID string) ([]*EdgeWithDetails, error) {
	query := `
	SELECT
		e.edge_id, e.edge_type,
		'' AS symbol_id, '' AS name, '' AS kind, '' AS signature,
		'' AS path,
		e.source_file, e.target_file, e.target_module
	FROM edges e
	WHERE e.source_id = $1
	  AND e.edge_type IN ('import', 'extends', 'implements', 'reference')
	  AND e.target_id IS NULL
	  AND e.target_module IS NOT NULL
	ORDER BY e.edge_type, e.target_module
	`
	return r.queryEdgesWithDetails(ctx, query, sourceSymbolID)
}

// queryEdgesWithDetails 执行 JOIN 查询并扫描为 EdgeWithDetails 切片。
func (r *EdgeRepository) queryEdgesWithDetails(ctx context.Context, query, arg string) ([]*EdgeWithDetails, error) {
	rows, err := r.db.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*EdgeWithDetails
	for rows.Next() {
		var d EdgeWithDetails
		if err := rows.Scan(
			&d.EdgeID, &d.EdgeType,
			&d.SymbolID, &d.Name, &d.Kind, &d.Signature,
			&d.FilePath,
			&d.SourceFile, &d.TargetFile, &d.TargetModule,
		); err != nil {
			return nil, err
		}
		results = append(results, &d)
	}
	return results, rows.Err()
}

// ReachableSymbol 是多跳可达性查询的单条结果，记录从起始符号出发沿调用边
// 递归可达的符号及其最近出现深度。
type ReachableSymbol struct {
	SymbolID  string
	Name      string
	Kind      string
	Signature string
	FilePath  string
	// Depth 是该符号相对起始符号的最短跳数（起始符号自身为 0，直接相邻为 1）。
	Depth int
}

// DefaultTransitiveDepth 是多跳查询的默认最大深度，防止环与大图上的递归爆炸。
// 通过查询参数可覆盖（调用方按需调大）。5 跳覆盖多数"调用链追踪"需求。
const DefaultTransitiveDepth = 5

// MaxTransitiveDepth 是多跳查询的硬上限，防止恶意 ?depth=1e9 触发递归 CTE
// 在密集调用图上产生爆炸性中间结果（DoS 防护）。任何传入的 maxDepth 都会被
// 收敛到该值以内。20 跳足够覆盖大型代码库中最深的调用链，同时保证 CTE 终止开销可控。
const MaxTransitiveDepth = 20

// transitiveQuery 构建递归 CTE 查询并执行。
//
// direction 决定递归方向：
//   - "forward"（传递调用链 callees）：沿 source→target 扩展，返回起始符号"会触及的代码"
//   - "backward"（反向影响 callers）：沿 target→source 扩展，返回"调用起始符号的代码"
//
// 用 UNION（而非 UNION ALL）对 symbol_id 去重，天然防环；保留每符号的最小 depth。
func (r *EdgeRepository) transitiveQuery(ctx context.Context, startSymbolID, direction string, maxDepth int) ([]*ReachableSymbol, error) {
	if maxDepth <= 0 {
		maxDepth = DefaultTransitiveDepth
	}
	// 硬上限：无论调用方传多大，都收敛到 MaxTransitiveDepth 以内，防止递归 CTE 爆炸。
	if maxDepth > MaxTransitiveDepth {
		maxDepth = MaxTransitiveDepth
	}

	// 递归步的"下一跳"列：正向取 target_id，反向取 source_id
	var nextCol string
	if direction == "backward" {
		nextCol = "source_id"
	} else {
		nextCol = "target_id"
	}

	// 起始符号的"对端"列：正向 base 从 source 出发记录 target；反向 base 从 target 出发记录 source
	var baseStartCol, baseNextCol string
	if direction == "backward" {
		// 反向：起始符号在 target_id，沿 source_id 扩展
		baseStartCol, baseNextCol = "target_id", "source_id"
	} else {
		// 正向：起始符号在 source_id，沿 target_id 扩展
		baseStartCol, baseNextCol = "source_id", "target_id"
	}

	query := fmt.Sprintf(`
	WITH RECURSIVE reach AS (
		-- Base case: 从起始符号的直接相邻符号开始（depth=1）。
		-- 不把起始符号自身放入结果（调用方已知），只返回可达的"其他"符号。
		SELECT e.%s AS symbol_id, 1 AS depth
		FROM edges e
		WHERE e.%s = $1 AND e.edge_type = 'call' AND e.%s IS NOT NULL

		UNION
		-- Recursive case: 沿 call 边再走一跳
		SELECT e.%s, r.depth + 1
		FROM reach r
		JOIN edges e ON e.%s = r.symbol_id
		WHERE e.edge_type = 'call' AND e.%s IS NOT NULL AND r.depth < $2
	)
	SELECT r.symbol_id, s.name, s.kind, s.signature, f.path, MIN(r.depth) AS depth
	FROM reach r
	JOIN symbols s ON s.symbol_id = r.symbol_id
	LEFT JOIN files f ON s.file_id = f.file_id
	GROUP BY r.symbol_id, s.name, s.kind, s.signature, f.path
	ORDER BY MIN(r.depth), s.name
	`, baseNextCol, baseStartCol, baseNextCol,
		nextCol, baseStartCol, nextCol)

	rows, err := r.db.QueryContext(ctx, query, startSymbolID, maxDepth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ReachableSymbol
	for rows.Next() {
		var rs ReachableSymbol
		if err := rows.Scan(&rs.SymbolID, &rs.Name, &rs.Kind, &rs.Signature, &rs.FilePath, &rs.Depth); err != nil {
			return nil, err
		}
		results = append(results, &rs)
	}
	return results, rows.Err()
}

// GetTransitiveCallees 返回从 startSymbolID 出发，沿调用边（source→target）
// 递归可达的全部符号（传递调用链）。maxDepth 限制最大跳数（<=0 用默认值）。
//
// 语义："起始符号的执行会触及哪些代码"。例如查 main 的 transitive callees
// 可得到整条调用树（去重，每符号取最短跳数）。
func (r *EdgeRepository) GetTransitiveCallees(ctx context.Context, startSymbolID string, maxDepth int) ([]*ReachableSymbol, error) {
	return r.transitiveQuery(ctx, startSymbolID, "forward", maxDepth)
}

// GetTransitiveCallers 返回沿调用边反向（target→source）递归可达的全部符号
// （反向影响范围）。maxDepth 限制最大跳数（<=0 用默认值）。
//
// 语义："修改起始符号会影响哪些代码"。例如查某底层函数的 transitive callers
// 可得到所有直接/间接依赖它的入口点。
func (r *EdgeRepository) GetTransitiveCallers(ctx context.Context, startSymbolID string, maxDepth int) ([]*ReachableSymbol, error) {
	return r.transitiveQuery(ctx, startSymbolID, "backward", maxDepth)
}
