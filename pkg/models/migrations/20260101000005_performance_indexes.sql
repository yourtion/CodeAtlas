-- 性能索引补全：复合 / 部分 / 覆盖 / 时间戳 / GIN / hash / 表达式索引
--
-- 历史上这些索引位于 docker/initdb/02_performance_indexes.sql（仅在容器首次
-- 启动时执行）。统一到 goose 真源后，基础单列索引已随 init 迁移落地，但覆盖、
-- 复合、时间戳、GIN、hash、LOWER 表达式等索引遗漏。本迁移把它们补回，确保
-- 关键查询路径的执行计划不回归。
--
-- 幂等：全部使用 IF NOT EXISTS，重复执行安全。
-- 所有索引均 CONCURRENTLY 不必需（ goose 在事务内执行，CONCURRENTLY 不被允许），
-- 对于已含数据的生产库，可在低峰期手工提前 CREATE INDEX CONCURRENTLY 后再跑迁移。

-- +goose Up

-- ---------------------------------------------------------------------------
-- 复合索引（常见组合查询）
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_files_repo_language ON files(repo_id, language);
CREATE INDEX IF NOT EXISTS idx_files_repo_path_pattern ON files(repo_id, path text_pattern_ops);
CREATE INDEX IF NOT EXISTS idx_symbols_name_kind ON symbols(name, kind);
CREATE INDEX IF NOT EXISTS idx_edges_source_type_target ON edges(source_id, edge_type, target_id);
CREATE INDEX IF NOT EXISTS idx_edges_target_type_source ON edges(target_id, edge_type, source_id);

-- ---------------------------------------------------------------------------
-- 部分索引（特定查询加速）
-- ---------------------------------------------------------------------------
-- 有 semantic_summary 的符号（embedding 生成时常用）
CREATE INDEX IF NOT EXISTS idx_symbols_with_summary ON symbols(symbol_id, file_id)
WHERE semantic_summary IS NOT NULL AND semantic_summary != '';
-- 无 target 的外部引用边
CREATE INDEX IF NOT EXISTS idx_edges_external ON edges(source_id, target_module, edge_type)
WHERE target_id IS NULL AND target_module IS NOT NULL;

-- ---------------------------------------------------------------------------
-- 覆盖索引（index-only scan 常见字段）
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_symbols_name_covering ON symbols(name)
INCLUDE (symbol_id, file_id, kind, signature, start_line, end_line);
CREATE INDEX IF NOT EXISTS idx_files_repo_covering ON files(repo_id)
INCLUDE (file_id, path, language, size, checksum);

-- ---------------------------------------------------------------------------
-- 时间戳索引（按时间倒序查询）
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_repositories_created_at ON repositories(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_repositories_updated_at ON repositories(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_updated_at ON files(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_symbols_created_at ON symbols(created_at DESC);

-- ---------------------------------------------------------------------------
-- GIN 索引（JSONB 查询）
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_repositories_metadata_gin ON repositories USING gin(metadata);
CREATE INDEX IF NOT EXISTS idx_ast_nodes_attributes_gin ON ast_nodes USING gin(attributes);

-- ---------------------------------------------------------------------------
-- Hash 索引（纯等值查询）
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_files_checksum_hash ON files USING hash(checksum);
CREATE INDEX IF NOT EXISTS idx_symbols_kind_hash ON symbols USING hash(kind);
CREATE INDEX IF NOT EXISTS idx_edges_type_hash ON edges USING hash(edge_type);

-- ---------------------------------------------------------------------------
-- 表达式索引（大小写不敏感查询）
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_symbols_name_lower ON symbols(LOWER(name));
CREATE INDEX IF NOT EXISTS idx_files_path_lower ON files(LOWER(path));

-- ---------------------------------------------------------------------------
-- 统计信息目标（提升高频列的查询规划质量）
-- ---------------------------------------------------------------------------
ALTER TABLE symbols ALTER COLUMN name SET STATISTICS 1000;
ALTER TABLE symbols ALTER COLUMN kind SET STATISTICS 1000;
ALTER TABLE files ALTER COLUMN path SET STATISTICS 1000;
ALTER TABLE files ALTER COLUMN language SET STATISTICS 1000;
ALTER TABLE edges ALTER COLUMN edge_type SET STATISTICS 1000;

ANALYZE repositories;
ANALYZE files;
ANALYZE symbols;
ANALYZE ast_nodes;
ANALYZE edges;
ANALYZE vectors;
ANALYZE docstrings;
ANALYZE summaries;


-- +goose Down

DROP INDEX IF EXISTS idx_files_path_lower;
DROP INDEX IF EXISTS idx_symbols_name_lower;
DROP INDEX IF EXISTS idx_edges_type_hash;
DROP INDEX IF EXISTS idx_symbols_kind_hash;
DROP INDEX IF EXISTS idx_files_checksum_hash;
DROP INDEX IF EXISTS idx_ast_nodes_attributes_gin;
DROP INDEX IF EXISTS idx_repositories_metadata_gin;
DROP INDEX IF EXISTS idx_symbols_created_at;
DROP INDEX IF EXISTS idx_files_updated_at;
DROP INDEX IF EXISTS idx_files_created_at;
DROP INDEX IF EXISTS idx_repositories_updated_at;
DROP INDEX IF EXISTS idx_repositories_created_at;
DROP INDEX IF EXISTS idx_files_repo_covering;
DROP INDEX IF EXISTS idx_symbols_name_covering;
DROP INDEX IF EXISTS idx_edges_external;
DROP INDEX IF EXISTS idx_symbols_with_summary;
DROP INDEX IF EXISTS idx_edges_target_type_source;
DROP INDEX IF EXISTS idx_edges_source_type_target;
DROP INDEX IF EXISTS idx_symbols_name_kind;
DROP INDEX IF EXISTS idx_files_repo_path_pattern;
DROP INDEX IF EXISTS idx_files_repo_language;
-- 统计目标不主动回滚（PG 默认 100，差异可忽略）
