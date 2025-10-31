-- CodeAtlas Knowledge Graph Database Schema
-- Migration: 02_performance_indexes
-- Description: Additional performance indexes for vector similarity search
-- Version: 1.0.0
-- Note: This migration should be run after initial data load for optimal index creation

-- ============================================================================
-- VECTOR SIMILARITY INDEXES
-- ============================================================================

-- Create IVFFlat index for vector similarity search
-- This index significantly improves performance for semantic search queries
-- The 'lists' parameter should be approximately sqrt(total_rows)
-- Adjust based on your data size:
-- - Small datasets (<100k vectors): lists = 100
-- - Medium datasets (100k-1M vectors): lists = 1000
-- - Large datasets (>1M vectors): lists = 2000+

DO $$
DECLARE
    vector_count INTEGER;
    lists_param INTEGER;
BEGIN
    -- Count existing vectors
    SELECT COUNT(*) INTO vector_count FROM vectors;
    
    -- Calculate optimal lists parameter
    IF vector_count < 100000 THEN
        lists_param := 100;
    ELSIF vector_count < 1000000 THEN
        lists_param := 1000;
    ELSE
        lists_param := 2000;
    END IF;
    
    RAISE NOTICE 'Creating IVFFlat index with lists = % for % vectors', lists_param, vector_count;
    
    -- Create the index
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_vectors_embedding_cosine 
                    ON vectors USING ivfflat (embedding vector_cosine_ops) 
                    WITH (lists = %s)', lists_param);
    
    RAISE NOTICE 'IVFFlat index created successfully';
END $$;

-- ============================================================================
-- ADDITIONAL COMPOSITE INDEXES
-- ============================================================================

-- Composite index for common symbol queries
CREATE INDEX IF NOT EXISTS idx_symbols_file_kind ON symbols(file_id, kind);

-- Composite index for edge traversal queries
CREATE INDEX IF NOT EXISTS idx_edges_source_target ON edges(source_id, target_id);

-- Index for file path prefix searches
CREATE INDEX IF NOT EXISTS idx_files_path_prefix ON files(path text_pattern_ops);

-- ============================================================================
-- STATISTICS UPDATE
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
-- MIGRATION TRACKING
-- ============================================================================

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES ('02_performance_indexes', 'Additional performance indexes for vector similarity search')
ON CONFLICT (version) DO NOTHING;

-- ============================================================================
-- COMPLETION
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 02_performance_indexes completed successfully';
    RAISE NOTICE 'Vector similarity index created';
    RAISE NOTICE 'Additional composite indexes created';
    RAISE NOTICE 'Table statistics updated';
END $$;
