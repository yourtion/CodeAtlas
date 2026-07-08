package models

import "context"

// 本文件提供仓库级图指标聚合查询，为 internal/quality 的 GraphEvaluator 提供原始数据。
//
// 仓库范围通过 files.repo_id 过滤：边经由 source 符号所属文件关联到仓库，
// 符号直接通过 file_id 关联文件。所有方法均带 ctx 与 repoID 参数。

// CountEdgesByType 按 edge_type 分组统计指定仓库的边数。
// 返回 map[edge_type]count。
func CountEdgesByType(ctx context.Context, r *EdgeRepository, repoID string) (map[string]int, error) {
	query := `
		SELECT e.edge_type, COUNT(*)
		FROM edges e
		JOIN symbols s ON e.source_id = s.symbol_id
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1
		GROUP BY e.edge_type
	`
	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var edgeType string
		var count int
		if err := rows.Scan(&edgeType, &count); err != nil {
			return nil, err
		}
		result[edgeType] = count
	}
	return result, rows.Err()
}

// CountDanglingEdges 按 edge_type 分组统计 target_id IS NULL 的边数
// （未解析符号的边，如指向外部依赖的 import）。
func CountDanglingEdges(ctx context.Context, r *EdgeRepository, repoID string) (map[string]int, error) {
	query := `
		SELECT e.edge_type, COUNT(*)
		FROM edges e
		JOIN symbols s ON e.source_id = s.symbol_id
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1 AND e.target_id IS NULL
		GROUP BY e.edge_type
	`
	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var edgeType string
		var count int
		if err := rows.Scan(&edgeType, &count); err != nil {
			return nil, err
		}
		result[edgeType] = count
	}
	return result, rows.Err()
}

// CountCrossFileEdges 统计 source_file ≠ target_file 的边数（跨文件依赖）。
func CountCrossFileEdges(ctx context.Context, r *EdgeRepository, repoID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM edges e
		JOIN symbols s ON e.source_id = s.symbol_id
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1
		  AND e.target_file IS NOT NULL
		  AND e.source_file IS NOT NULL
		  AND e.source_file <> e.target_file
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	return count, err
}

// CountTotalSymbols 统计指定仓库的总符号数。
func CountTotalSymbols(ctx context.Context, r *SymbolRepository, repoID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM symbols s
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	return count, err
}

// CountOrphanSymbols 统计无任何出入边的孤立符号数
// （既不作为任何边的 source，也不作为任何边的 target）。
func CountOrphanSymbols(ctx context.Context, r *SymbolRepository, repoID string) (int, error) {
	// 子查询 JOIN symbols+files 按 repo_id 过滤，避免多 repo 场景下其它仓库的边
	// 把本仓库符号误判为非孤立。target_id 子查询显式排除 NULL 以规避 NOT IN 的 NULL 中毒。
	query := `
		SELECT COUNT(*)
		FROM symbols s
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1
		  AND s.symbol_id NOT IN (
			SELECT e.source_id FROM edges e
			JOIN symbols es ON e.source_id = es.symbol_id
			JOIN files ef ON es.file_id = ef.file_id
			WHERE ef.repo_id = $1
		  )
		  AND s.symbol_id NOT IN (
			SELECT e.target_id FROM edges e
			JOIN symbols ts ON e.target_id = ts.symbol_id
			JOIN files tf ON ts.file_id = tf.file_id
			WHERE tf.repo_id = $1 AND e.target_id IS NOT NULL
		  )
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	return count, err
}

// ChainSpec 调用链端点对（models 层定义，避免循环依赖 quality 包）。
type ChainSpec struct {
	StartName string
	EndName   string
	StartFile string
	EndFile   string
}

// CheckSingleChainConnectivity 用递归 CTE 查 start 是否能经 call 边到达 end。
func CheckSingleChainConnectivity(ctx context.Context, r *EdgeRepository, repoID string, c ChainSpec) (bool, error) {
	query := `
		WITH RECURSIVE reach AS (
			SELECT s.symbol_id FROM symbols s
			JOIN files f ON s.file_id = f.file_id
			WHERE f.repo_id = $1 AND s.name = $2 AND f.path = $3
			UNION
			SELECT e.target_id FROM reach r
			JOIN edges e ON e.source_id = r.symbol_id
			WHERE e.edge_type = 'call' AND e.target_id IS NOT NULL
		)
		SELECT EXISTS(
			SELECT 1 FROM reach r
			JOIN symbols s ON r.symbol_id = s.symbol_id
			JOIN files f ON s.file_id = f.file_id
			WHERE s.name = $4 AND f.path = $5
		)
	`
	var connected bool
	err := r.db.QueryRowContext(ctx, query, repoID, c.StartName, c.StartFile, c.EndName, c.EndFile).Scan(&connected)
	return connected, err
}

// ExtractedEdge models 层的提取边（供 quality 层转换）。
type ExtractedEdge struct {
	SourceID   string
	SourceName string
	EdgeType   string
	TargetID   string
	TargetName string
}

// ListExtractedEdges 返回仓库内所有提取出的边（source_id/edge_type/target_id +
// 对应 name，用于 symbol_id 精确匹配解决 C++ 重载同名问题）。
//
// target_id 取解析后的目标符号 ID；若未解析到（悬空），COALESCE 回空串。
// target_name 取解析后的目标符号名；悬空时回退到 target_module
// （对 import 边而言 target_module 是有意义的标识，如 "java.util.ArrayList"），
// 仅供调试日志——匹配只用 symbol_id 三元组。
func ListExtractedEdges(ctx context.Context, r *EdgeRepository, repoID string) ([]ExtractedEdge, error) {
	query := `
		SELECT e.source_id, s_source.name, e.edge_type,
		       COALESCE(e.target_id, ''), COALESCE(s_target.name, COALESCE(e.target_module, ''))
		FROM edges e
		JOIN symbols s_source ON e.source_id = s_source.symbol_id
		JOIN files f ON s_source.file_id = f.file_id
		LEFT JOIN symbols s_target ON e.target_id = s_target.symbol_id
		WHERE f.repo_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ExtractedEdge
	for rows.Next() {
		var e ExtractedEdge
		if err := rows.Scan(&e.SourceID, &e.SourceName, &e.EdgeType, &e.TargetID, &e.TargetName); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
