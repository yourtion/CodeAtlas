-- CodeAtlas 初始数据库 Schema
-- 合并自原 docker/initdb 与 deployments/migrations 两套并行定义，统一为唯一真源。
-- 由 goose 嵌入二进制执行（pkg/models/migrations.go），goose 自动维护 goose_db_version 表。

-- +goose Up

-- ============================================================================
-- 扩展
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS age;

LOAD 'age';
SET search_path = ag_catalog, "$user", public;

-- ============================================================================
-- 核心表
-- ============================================================================

SET search_path = public, ag_catalog, "$user";

-- repositories: 仓库元数据
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

-- files: 源代码文件（checksum 用于增量索引）
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

-- symbols: 代码符号（函数、类、变量等）
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
CREATE INDEX IF NOT EXISTS idx_symbols_file_kind ON symbols(file_id, kind);

-- ast_nodes: 完整语法树
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

-- edges: 关系边（call, import, extends, implements）
-- 注意：target_id 使用 ON DELETE SET NULL（保留边的 source 记录，便于追溯调用方），
-- 此前 Go 内联 schema 误用 CASCADE，会丢失调用关系历史。
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
CREATE INDEX IF NOT EXISTS idx_edges_source_target ON edges(source_id, target_id);

-- ============================================================================
-- 向量与摘要存储
-- ============================================================================

-- vectors: 语义嵌入向量（pgvector）
-- 维度固定为 1024，匹配默认嵌入模型 text-embedding-qwen3-embedding-0.6b。
-- 如需切换维度（如 nomic-embed-text 768），应编写新的 ALTER 迁移，不在此处参数化。
CREATE TABLE IF NOT EXISTS vectors (
    vector_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    embedding vector(1024),
    content TEXT NOT NULL,
    model VARCHAR(100) NOT NULL,
    chunk_index INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_entity_chunk UNIQUE(entity_id, entity_type, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_vectors_entity ON vectors(entity_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_vectors_model ON vectors(model);

-- docstrings: 符号文档字符串
CREATE TABLE IF NOT EXISTS docstrings (
    doc_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol_id UUID NOT NULL REFERENCES symbols(symbol_id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_docstrings_symbol ON docstrings(symbol_id);

-- summaries: 实体摘要（部分索引用于带 docstring 的查询优化）
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
CREATE INDEX IF NOT EXISTS idx_symbols_with_docstring ON symbols(docstring) WHERE docstring IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_edges_with_target ON edges(target_id) WHERE target_id IS NOT NULL;

-- 文件路径前缀搜索
CREATE INDEX IF NOT EXISTS idx_files_path_prefix ON files(path text_pattern_ops);

-- ============================================================================
-- AGE 图
-- ============================================================================

SELECT * FROM ag_catalog.create_graph('code_graph');

-- ============================================================================
-- updated_at 触发器（DB 为真源，应用层不再赋值）
-- ============================================================================

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

DROP TRIGGER IF EXISTS update_repositories_updated_at ON repositories;
CREATE TRIGGER update_repositories_updated_at BEFORE UPDATE ON repositories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_files_updated_at ON files;
CREATE TRIGGER update_files_updated_at BEFORE UPDATE ON files
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 视图
-- ============================================================================

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
-- 授权（应用用户 codeatlas）
-- ============================================================================

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO codeatlas;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO codeatlas;
GRANT USAGE ON SCHEMA ag_catalog TO codeatlas;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA ag_catalog TO codeatlas;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA ag_catalog TO codeatlas;

-- 更新表统计信息，便于查询规划器
ANALYZE repositories;
ANALYZE files;
ANALYZE symbols;
ANALYZE ast_nodes;
ANALYZE edges;
ANALYZE vectors;
ANALYZE docstrings;
ANALYZE summaries;


-- +goose Down

DROP VIEW IF EXISTS edges_with_symbols;
DROP VIEW IF EXISTS symbols_with_files;
DROP TRIGGER IF EXISTS update_files_updated_at ON files;
DROP TRIGGER IF EXISTS update_repositories_updated_at ON repositories;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS summaries;
DROP TABLE IF EXISTS docstrings;
DROP TABLE IF EXISTS vectors;
DROP TABLE IF EXISTS edges;
DROP TABLE IF EXISTS ast_nodes;
DROP TABLE IF EXISTS symbols;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS repositories;
