-- CodeAtlas Knowledge Graph Database Schema
-- This script initializes the complete database schema including:
-- - Core tables (repositories, files, symbols, ast_nodes, edges)
-- - Vector storage (vectors, docstrings, summaries)
-- - AGE graph schema
-- - Performance indexes

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

-- Enable pgvector for semantic search
CREATE EXTENSION IF NOT EXISTS vector;

-- Enable Apache AGE for graph database capabilities
CREATE EXTENSION IF NOT EXISTS age;

-- Load AGE into search path
LOAD 'age';
SET search_path = ag_catalog, "$user", public;

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- Reset search path to create tables in public schema
SET search_path = public, ag_catalog, "$user";

-- Repositories table
CREATE TABLE IF NOT EXISTS repositories (
    repo_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    url TEXT,
    branch VARCHAR(255) DEFAULT 'main',
    commit_hash VARCHAR(64),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_repo_name UNIQUE(name)
);

CREATE INDEX IF NOT EXISTS idx_repositories_name ON repositories(name);
CREATE INDEX IF NOT EXISTS idx_repositories_commit ON repositories(commit_hash);

-- Files table
CREATE TABLE IF NOT EXISTS files (
    file_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id UUID NOT NULL REFERENCES repositories(repo_id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    language VARCHAR(50) NOT NULL,
    size BIGINT NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_repo_file_path UNIQUE(repo_id, path)
);

CREATE INDEX IF NOT EXISTS idx_files_repo ON files(repo_id);
CREATE INDEX IF NOT EXISTS idx_files_checksum ON files(checksum);
CREATE INDEX IF NOT EXISTS idx_files_language ON files(language);
CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);

-- Symbols table
CREATE TABLE IF NOT EXISTS symbols (
    symbol_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    kind VARCHAR(50) NOT NULL,
    signature TEXT,
    start_line INT NOT NULL,
    end_line INT NOT NULL,
    start_byte INT NOT NULL,
    end_byte INT NOT NULL,
    docstring TEXT,
    semantic_summary TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_file_symbol_location UNIQUE(file_id, name, start_line, start_byte)
);

CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);
CREATE INDEX IF NOT EXISTS idx_symbols_location ON symbols(file_id, start_line, end_line);

-- AST Nodes table
CREATE TABLE IF NOT EXISTS ast_nodes (
    node_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    parent_id UUID REFERENCES ast_nodes(node_id) ON DELETE CASCADE,
    start_line INT NOT NULL,
    end_line INT NOT NULL,
    start_byte INT NOT NULL,
    end_byte INT NOT NULL,
    text TEXT,
    attributes JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ast_nodes_file ON ast_nodes(file_id);
CREATE INDEX IF NOT EXISTS idx_ast_nodes_parent ON ast_nodes(parent_id);
CREATE INDEX IF NOT EXISTS idx_ast_nodes_type ON ast_nodes(type);
CREATE INDEX IF NOT EXISTS idx_ast_nodes_location ON ast_nodes(file_id, start_line, end_line);

-- Dependency Edges table
CREATE TABLE IF NOT EXISTS edges (
    edge_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES symbols(symbol_id) ON DELETE CASCADE,
    target_id UUID REFERENCES symbols(symbol_id) ON DELETE SET NULL,
    edge_type VARCHAR(50) NOT NULL,
    source_file TEXT NOT NULL,
    target_file TEXT,
    target_module TEXT,
    line_number INT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(edge_type);
CREATE INDEX IF NOT EXISTS idx_edges_source_type ON edges(source_id, edge_type);
CREATE INDEX IF NOT EXISTS idx_edges_target_type ON edges(target_id, edge_type);

-- ============================================================================
-- VECTOR STORAGE TABLES
-- ============================================================================

-- Vectors table for semantic embeddings
CREATE TABLE IF NOT EXISTS vectors (
    vector_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    embedding vector(768),
    content TEXT NOT NULL,
    model VARCHAR(100) NOT NULL,
    chunk_index INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_entity_chunk UNIQUE(entity_id, entity_type, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_vectors_entity ON vectors(entity_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_vectors_model ON vectors(model);

-- Create IVFFlat index for vector similarity search
-- Note: This should be created after data is populated for better performance
-- Uncomment after initial data load:
-- CREATE INDEX IF NOT EXISTS idx_vectors_embedding ON vectors 
--   USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Docstrings table
CREATE TABLE IF NOT EXISTS docstrings (
    doc_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol_id UUID NOT NULL REFERENCES symbols(symbol_id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_docstrings_symbol ON docstrings(symbol_id);

-- Summaries table
CREATE TABLE IF NOT EXISTS summaries (
    summary_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    summary_type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_entity_summary UNIQUE(entity_id, entity_type, summary_type)
);

CREATE INDEX IF NOT EXISTS idx_summaries_entity ON summaries(entity_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_summaries_type ON summaries(summary_type);

-- ============================================================================
-- AGE GRAPH SCHEMA
-- ============================================================================

-- Create the code graph if it doesn't exist
SELECT * FROM ag_catalog.create_graph('code_graph');

-- Note: AGE vertex and edge labels are created dynamically when first used
-- The following labels will be used in the application:
--
-- Vertex Labels:
--   - Function: Function symbols
--   - Class: Class symbols
--   - Interface: Interface symbols
--   - Variable: Variable symbols
--   - Module: Module/file symbols
--
-- Edge Labels:
--   - CALLS: Function call relationships
--   - IMPORTS: Import/dependency relationships
--   - EXTENDS: Class inheritance relationships
--   - IMPLEMENTS: Interface implementation relationships
--   - REFERENCES: General symbol reference relationships

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add triggers for updated_at columns
CREATE TRIGGER update_repositories_updated_at BEFORE UPDATE ON repositories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_files_updated_at BEFORE UPDATE ON files
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

-- View for symbols with file information
CREATE OR REPLACE VIEW symbols_with_files AS
SELECT 
    s.symbol_id,
    s.name,
    s.kind,
    s.signature,
    s.start_line,
    s.end_line,
    s.docstring,
    f.file_id,
    f.path as file_path,
    f.language,
    r.repo_id,
    r.name as repo_name
FROM symbols s
JOIN files f ON s.file_id = f.file_id
JOIN repositories r ON f.repo_id = r.repo_id;

-- View for edges with symbol details
CREATE OR REPLACE VIEW edges_with_symbols AS
SELECT 
    e.edge_id,
    e.edge_type,
    e.source_file,
    e.target_file,
    e.target_module,
    e.line_number,
    s1.symbol_id as source_symbol_id,
    s1.name as source_name,
    s1.kind as source_kind,
    s2.symbol_id as target_symbol_id,
    s2.name as target_name,
    s2.kind as target_kind
FROM edges e
JOIN symbols s1 ON e.source_id = s1.symbol_id
LEFT JOIN symbols s2 ON e.target_id = s2.symbol_id;

-- ============================================================================
-- GRANTS (for application user)
-- ============================================================================

-- Grant necessary permissions to the codeatlas user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO codeatlas;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO codeatlas;
GRANT USAGE ON SCHEMA ag_catalog TO codeatlas;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA ag_catalog TO codeatlas;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA ag_catalog TO codeatlas;

-- ============================================================================
-- COMPLETION
-- ============================================================================

-- Log schema initialization
DO $$
BEGIN
    RAISE NOTICE 'CodeAtlas database schema initialized successfully';
    RAISE NOTICE 'Extensions: pgvector, age';
    RAISE NOTICE 'Tables: repositories, files, symbols, ast_nodes, edges, vectors, docstrings, summaries';
    RAISE NOTICE 'Graph: code_graph';
END $$;
