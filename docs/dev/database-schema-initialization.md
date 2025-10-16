# Database Schema Initialization

This document describes the database schema initialization process for CodeAtlas Knowledge Graph Indexer.

## Overview

The database schema consists of:
- **Core tables**: repositories, files, symbols, ast_nodes, edges
- **Vector storage**: vectors, docstrings, summaries
- **Extensions**: pgvector (semantic search), Apache AGE (graph database)
- **AGE graph**: code_graph with vertex and edge labels

## Prerequisites

- PostgreSQL 17+
- pgvector extension
- Apache AGE extension

## Initialization Methods

### Method 1: Automatic Initialization via API Server

The schema is automatically initialized when the API server starts:

```bash
# Start all services (database + API)
docker-compose up -d

# Or start API server directly
make run-api
```

The API server will:
1. Wait for database to be ready (with retries)
2. Initialize schema (extensions, tables, indexes, AGE graph)
3. Run health check
4. Log database statistics

### Method 2: Docker Initialization Script

The schema is automatically initialized when starting the database with Docker Compose:

```bash
docker-compose up -d db
```

The initialization script `docker/initdb/01_init_schema.sql` runs automatically on first database startup.

### Method 3: Manual Initialization Tool

Use the standalone initialization tool for manual maintenance:

```bash
# Build the tool
make build-init-db

# Initialize database
make init-db

# Initialize with statistics
make init-db-stats

# Create vector index (after data is loaded)
make init-db-with-index

# Or run directly with custom options
./bin/init-db -max-retries 10 -retry-delay 2 -stats
./bin/init-db -create-vector-index -vector-index-lists 100
```

Tool Flags:
- `-max-retries`: Maximum connection retry attempts (default: 10)
- `-retry-delay`: Delay between retries in seconds (default: 2)
- `-create-vector-index`: Create IVFFlat vector similarity index
- `-vector-index-lists`: Number of lists for vector index (default: 100)
- `-stats`: Show database statistics after initialization

### Method 4: Programmatic Initialization

Use the Go API in your application:

```go
package main

import (
    "context"
    "log"
    "time"
    
    "codeatlas/pkg/models"
)

func main() {
    // Wait for database to be ready
    db, err := models.WaitForDatabase(10, 2*time.Second)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Initialize schema
    sm := models.NewSchemaManager(db)
    ctx := context.Background()
    
    if err := sm.InitializeSchema(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Run health check
    if err := sm.HealthCheck(ctx); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Database initialized successfully")
}
```

## Database Schema

### Core Tables

#### repositories
Stores repository metadata.

```sql
CREATE TABLE repositories (
    repo_id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url TEXT,
    branch VARCHAR(255) DEFAULT 'main',
    commit_hash VARCHAR(64),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

#### files
Stores file metadata with checksums for incremental updates.

```sql
CREATE TABLE files (
    file_id UUID PRIMARY KEY,
    repo_id UUID NOT NULL REFERENCES repositories(repo_id),
    path TEXT NOT NULL,
    language VARCHAR(50) NOT NULL,
    size BIGINT NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(repo_id, path)
);
```

#### symbols
Stores code symbols (functions, classes, variables, etc.).

```sql
CREATE TABLE symbols (
    symbol_id UUID PRIMARY KEY,
    file_id UUID NOT NULL REFERENCES files(file_id),
    name VARCHAR(255) NOT NULL,
    kind VARCHAR(50) NOT NULL,
    signature TEXT,
    start_line INT NOT NULL,
    end_line INT NOT NULL,
    start_byte INT NOT NULL,
    end_byte INT NOT NULL,
    docstring TEXT,
    semantic_summary TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### ast_nodes
Stores abstract syntax tree nodes.

```sql
CREATE TABLE ast_nodes (
    node_id UUID PRIMARY KEY,
    file_id UUID NOT NULL REFERENCES files(file_id),
    type VARCHAR(100) NOT NULL,
    parent_id UUID REFERENCES ast_nodes(node_id),
    start_line INT NOT NULL,
    end_line INT NOT NULL,
    start_byte INT NOT NULL,
    end_byte INT NOT NULL,
    text TEXT,
    attributes JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### edges
Stores dependency relationships between symbols.

```sql
CREATE TABLE edges (
    edge_id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES symbols(symbol_id),
    target_id UUID REFERENCES symbols(symbol_id),
    edge_type VARCHAR(50) NOT NULL,
    source_file TEXT NOT NULL,
    target_file TEXT,
    target_module TEXT,
    line_number INT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Vector Storage Tables

#### vectors
Stores semantic embeddings for code entities.

```sql
CREATE TABLE vectors (
    vector_id UUID PRIMARY KEY,
    entity_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    embedding vector(768),
    content TEXT NOT NULL,
    model VARCHAR(100) NOT NULL,
    chunk_index INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### docstrings
Stores documentation strings.

```sql
CREATE TABLE docstrings (
    doc_id UUID PRIMARY KEY,
    symbol_id UUID NOT NULL REFERENCES symbols(symbol_id),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### summaries
Stores semantic summaries of code entities.

```sql
CREATE TABLE summaries (
    summary_id UUID PRIMARY KEY,
    entity_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    summary_type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Indexes

Performance indexes are automatically created:

### File Indexes
- `idx_files_repo`: Fast repository file lookups
- `idx_files_checksum`: Incremental update support
- `idx_files_language`: Language filtering
- `idx_files_path`: Path-based queries

### Symbol Indexes
- `idx_symbols_file`: File symbol lookups
- `idx_symbols_name`: Name-based searches
- `idx_symbols_kind`: Kind filtering (function, class, etc.)
- `idx_symbols_location`: Location-based queries

### Edge Indexes
- `idx_edges_source`: Source symbol lookups
- `idx_edges_target`: Target symbol lookups
- `idx_edges_type`: Edge type filtering
- `idx_edges_source_type`: Combined source + type queries
- `idx_edges_target_type`: Combined target + type queries

### Vector Indexes
- `idx_vectors_entity`: Entity-based lookups
- `idx_vectors_embedding`: IVFFlat similarity search (created after data load)

## AGE Graph Schema

The `code_graph` is created with the following structure:

### Vertex Labels
- `Function`: Function symbols
- `Class`: Class symbols
- `Interface`: Interface symbols
- `Variable`: Variable symbols
- `Module`: Module/file symbols

### Edge Labels
- `CALLS`: Function call relationships
- `IMPORTS`: Import/dependency relationships
- `EXTENDS`: Class inheritance relationships
- `IMPLEMENTS`: Interface implementation relationships
- `REFERENCES`: General symbol reference relationships

### Example Cypher Queries

```cypher
-- Find all functions called by a function
MATCH (f:Function {symbol_id: $1})-[:CALLS]->(called:Function)
RETURN called.name, called.file_path;

-- Find call chain depth
MATCH path = (f:Function {name: $1})-[:CALLS*1..5]->(target:Function)
RETURN path, length(path) as depth
ORDER BY depth;

-- Find all classes that implement an interface
MATCH (c:Class)-[:IMPLEMENTS]->(i:Interface {name: $1})
RETURN c.name, c.file_path;
```

## Views

Convenience views for common queries:

### symbols_with_files
Joins symbols with file and repository information.

```sql
SELECT * FROM symbols_with_files WHERE repo_name = 'my-repo';
```

### edges_with_symbols
Joins edges with source and target symbol details.

```sql
SELECT * FROM edges_with_symbols WHERE edge_type = 'call';
```

## Health Check

Verify database health:

```go
sm := models.NewSchemaManager(db)
if err := sm.HealthCheck(ctx); err != nil {
    log.Fatal("Health check failed:", err)
}
```

Health check verifies:
- Database connection
- pgvector extension
- AGE extension
- code_graph existence

## Database Statistics

Get database statistics:

```go
stats, err := sm.GetDatabaseStats(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Repositories: %d\n", stats.RepositoryCount)
fmt.Printf("Files: %d\n", stats.FileCount)
fmt.Printf("Symbols: %d\n", stats.SymbolCount)
fmt.Printf("Edges: %d\n", stats.EdgeCount)
fmt.Printf("Vectors: %d\n", stats.VectorCount)
fmt.Printf("Database Size: %s\n", stats.DatabaseSize)
```

## Vector Index Creation

The vector similarity index should be created after initial data load for optimal performance:

```bash
# Via CLI
./bin/cli init-db --create-vector-index --vector-index-lists 100

# Via Go API
sm.CreateVectorIndex(ctx, 100)
```

The `lists` parameter should be approximately `sqrt(row_count)` for optimal performance.

## Troubleshooting

### Extension Not Found

If pgvector or AGE extensions are not available:

```bash
# Install pgvector
git clone https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install

# Install AGE
git clone https://github.com/apache/age.git
cd age
make
sudo make install
```

### Permission Denied

Ensure the database user has sufficient privileges:

```sql
GRANT ALL PRIVILEGES ON DATABASE codeatlas TO codeatlas;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO codeatlas;
GRANT USAGE ON SCHEMA ag_catalog TO codeatlas;
```

### Connection Refused

Ensure PostgreSQL is running and accessible:

```bash
# Check PostgreSQL status
docker-compose ps

# View logs
docker-compose logs postgres

# Restart database
docker-compose restart postgres
```

## Environment Variables

Configure database connection:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
export DB_NAME=codeatlas
```

## Testing

Run schema initialization tests:

```bash
# Run all schema tests
go test -v ./pkg/models -run TestSchemaManager

# Run specific test
go test -v ./pkg/models -run TestSchemaManager_InitializeSchema

# Skip integration tests
go test -v -short ./pkg/models
```

## Migration Strategy

For future schema changes:

1. Create new migration script: `docker/initdb/02_migration_name.sql`
2. Update `SchemaManager.GetSchemaVersion()` to track version
3. Implement migration logic in `SchemaManager`
4. Test migration on development database
5. Document breaking changes

## Performance Tuning

### Connection Pooling

```go
db.SetMaxOpenConns(20)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

### Index Maintenance

```sql
-- Analyze tables for query optimization
ANALYZE repositories;
ANALYZE files;
ANALYZE symbols;
ANALYZE edges;
ANALYZE vectors;

-- Reindex if needed
REINDEX TABLE vectors;
```

### Vector Index Tuning

```sql
-- Adjust lists parameter based on data size
-- lists = sqrt(row_count) is a good starting point
CREATE INDEX idx_vectors_embedding ON vectors 
USING ivfflat (embedding vector_cosine_ops) 
WITH (lists = 100);
```

## References

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [Apache AGE Documentation](https://age.apache.org/)
- [CodeAtlas Architecture](./README.md)
