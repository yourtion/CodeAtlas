-- Create CodeAtlas database schema

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS age;

-- Set AGE graph path
SET search_path = ag_catalog, "$user", public;
SELECT create_graph('codeatlas');

-- Repositories table
CREATE TABLE repositories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(512),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Files table
CREATE TABLE files (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER REFERENCES repositories(id),
    path VARCHAR(1024) NOT NULL,
    content TEXT,
    language VARCHAR(50),
    size INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Git commits table
CREATE TABLE commits (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER REFERENCES repositories(id),
    hash VARCHAR(40) NOT NULL,
    author VARCHAR(255),
    email VARCHAR(255),
    message TEXT,
    timestamp TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- File vectors table (for semantic search)
CREATE TABLE file_vectors (
    id SERIAL PRIMARY KEY,
    file_id INTEGER REFERENCES files(id),
    vector vector(768), -- Assuming 768-dimensional embeddings
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_files_repository_id ON files(repository_id);
CREATE INDEX idx_commits_repository_id ON commits(repository_id);
CREATE INDEX idx_file_vectors_file_id ON file_vectors(file_id);
CREATE INDEX idx_file_vectors_vector ON file_vectors USING hnsw (vector vector_l2_ops);