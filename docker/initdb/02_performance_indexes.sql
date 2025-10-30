-- Additional Performance Indexes for CodeAtlas
-- This script creates additional indexes to optimize common query patterns

-- ============================================================================
-- COMPOSITE INDEXES FOR COMMON QUERIES
-- ============================================================================

-- Composite index for file lookups by repo and language
CREATE INDEX IF NOT EXISTS idx_files_repo_language ON files(repo_id, language);

-- Composite index for file lookups by repo and path pattern
CREATE INDEX IF NOT EXISTS idx_files_repo_path_pattern ON files(repo_id, path text_pattern_ops);

-- Composite index for symbol lookups by file and kind
CREATE INDEX IF NOT EXISTS idx_symbols_file_kind ON symbols(file_id, kind);

-- Composite index for symbol lookups by name and kind
CREATE INDEX IF NOT EXISTS idx_symbols_name_kind ON symbols(name, kind);

-- Composite index for edge lookups by source and type
CREATE INDEX IF NOT EXISTS idx_edges_source_type_target ON edges(source_id, edge_type, target_id);

-- Composite index for edge lookups by target and type
CREATE INDEX IF NOT EXISTS idx_edges_target_type_source ON edges(target_id, edge_type, source_id);

-- ============================================================================
-- PARTIAL INDEXES FOR SPECIFIC QUERIES
-- ============================================================================

-- Partial index for symbols with docstrings (for embedding generation)
CREATE INDEX IF NOT EXISTS idx_symbols_with_docstring ON symbols(symbol_id, file_id) 
WHERE docstring IS NOT NULL AND docstring != '';

-- Partial index for symbols with semantic summaries
CREATE INDEX IF NOT EXISTS idx_symbols_with_summary ON symbols(symbol_id, file_id) 
WHERE semantic_summary IS NOT NULL AND semantic_summary != '';

-- Partial index for edges with targets (excluding external references)
CREATE INDEX IF NOT EXISTS idx_edges_with_target ON edges(source_id, target_id, edge_type) 
WHERE target_id IS NOT NULL;

-- Partial index for edges without targets (external references)
CREATE INDEX IF NOT EXISTS idx_edges_external ON edges(source_id, target_module, edge_type) 
WHERE target_id IS NULL AND target_module IS NOT NULL;

-- ============================================================================
-- COVERING INDEXES FOR COMMON QUERIES
-- ============================================================================

-- Covering index for symbol search by name (includes commonly accessed fields)
CREATE INDEX IF NOT EXISTS idx_symbols_name_covering ON symbols(name) 
INCLUDE (symbol_id, file_id, kind, signature, start_line, end_line);

-- Covering index for file listing (includes commonly accessed fields)
CREATE INDEX IF NOT EXISTS idx_files_repo_covering ON files(repo_id) 
INCLUDE (file_id, path, language, size, checksum);

-- ============================================================================
-- BTREE INDEXES FOR RANGE QUERIES
-- ============================================================================

-- Index for timestamp-based queries on repositories
CREATE INDEX IF NOT EXISTS idx_repositories_created_at ON repositories(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_repositories_updated_at ON repositories(updated_at DESC);

-- Index for timestamp-based queries on files
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_updated_at ON files(updated_at DESC);

-- Index for timestamp-based queries on symbols
CREATE INDEX IF NOT EXISTS idx_symbols_created_at ON symbols(created_at DESC);

-- ============================================================================
-- GIN INDEXES FOR JSONB QUERIES
-- ============================================================================

-- GIN index for repository metadata queries
CREATE INDEX IF NOT EXISTS idx_repositories_metadata_gin ON repositories USING gin(metadata);

-- GIN index for AST node attributes queries
CREATE INDEX IF NOT EXISTS idx_ast_nodes_attributes_gin ON ast_nodes USING gin(attributes);

-- ============================================================================
-- HASH INDEXES FOR EQUALITY QUERIES
-- ============================================================================

-- Hash index for checksum lookups (exact match only)
CREATE INDEX IF NOT EXISTS idx_files_checksum_hash ON files USING hash(checksum);

-- Hash index for symbol kind lookups
CREATE INDEX IF NOT EXISTS idx_symbols_kind_hash ON symbols USING hash(kind);

-- Hash index for edge type lookups
CREATE INDEX IF NOT EXISTS idx_edges_type_hash ON edges USING hash(edge_type);

-- ============================================================================
-- VECTOR INDEXES (IVFFLAT)
-- ============================================================================

-- Create IVFFlat index for vector similarity search
-- Note: This should be created after data is populated for better performance
-- The number of lists should be approximately sqrt(total_rows)
-- For 100k vectors, use lists=316; for 1M vectors, use lists=1000

-- Uncomment after initial data load and adjust lists parameter:
-- DROP INDEX IF EXISTS idx_vectors_embedding;
-- CREATE INDEX idx_vectors_embedding ON vectors 
--   USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- For better performance with large datasets, consider HNSW index (requires pgvector 0.5.0+):
-- CREATE INDEX idx_vectors_embedding_hnsw ON vectors 
--   USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

-- ============================================================================
-- EXPRESSION INDEXES
-- ============================================================================

-- Index for case-insensitive symbol name searches
CREATE INDEX IF NOT EXISTS idx_symbols_name_lower ON symbols(LOWER(name));

-- Index for case-insensitive file path searches
CREATE INDEX IF NOT EXISTS idx_files_path_lower ON files(LOWER(path));

-- ============================================================================
-- STATISTICS TARGETS
-- ============================================================================

-- Increase statistics target for frequently queried columns
ALTER TABLE symbols ALTER COLUMN name SET STATISTICS 1000;
ALTER TABLE symbols ALTER COLUMN kind SET STATISTICS 1000;
ALTER TABLE files ALTER COLUMN path SET STATISTICS 1000;
ALTER TABLE files ALTER COLUMN language SET STATISTICS 1000;
ALTER TABLE edges ALTER COLUMN edge_type SET STATISTICS 1000;

-- ============================================================================
-- ANALYZE TABLES
-- ============================================================================

-- Update table statistics for query planner
ANALYZE repositories;
ANALYZE files;
ANALYZE symbols;
ANALYZE ast_nodes;
ANALYZE edges;
ANALYZE vectors;
ANALYZE docstrings;
ANALYZE summaries;

-- ============================================================================
-- COMPLETION
-- ============================================================================

DO $
BEGIN
    RAISE NOTICE 'Performance indexes created successfully';
    RAISE NOTICE 'Composite indexes: 6';
    RAISE NOTICE 'Partial indexes: 4';
    RAISE NOTICE 'Covering indexes: 2';
    RAISE NOTICE 'BTree indexes: 6';
    RAISE NOTICE 'GIN indexes: 2';
    RAISE NOTICE 'Hash indexes: 3';
    RAISE NOTICE 'Expression indexes: 2';
    RAISE NOTICE 'Statistics targets updated';
    RAISE NOTICE 'Tables analyzed';
END $;
