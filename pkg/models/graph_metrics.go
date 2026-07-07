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
	query := `
		SELECT COUNT(*)
		FROM symbols s
		JOIN files f ON s.file_id = f.file_id
		WHERE f.repo_id = $1
		  AND s.symbol_id NOT IN (SELECT source_id FROM edges WHERE source_id IS NOT NULL)
		  AND s.symbol_id NOT IN (SELECT target_id FROM edges WHERE target_id IS NOT NULL)
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	return count, err
}
