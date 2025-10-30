# Knowledge Graph Indexer Documentation

The Knowledge Graph Indexer transforms parsed code structures into a queryable knowledge base, enabling semantic search, graph traversal, and relational queries across your codebase.

## Table of Contents

### Getting Started
- **[Quick Start Guide](./quick-start.md)** - Get indexing in minutes
- **[Architecture Overview](./architecture.md)** - System design and components
- **[Configuration Guide](./configuration.md)** - All configuration options

### API Documentation
- **[API Reference](./api-reference.md)** - Complete endpoint documentation
- **[Request/Response Examples](./api-examples.md)** - Practical API usage examples

### CLI Documentation
- **[CLI Index Command](./cli-index-command.md)** - Index repositories via CLI
- **[CLI Search Command](./cli-search-command.md)** - Search indexed code

### Advanced Topics
- **[Incremental Indexing](./incremental-indexing.md)** - Efficient updates
- **[Vector Embeddings](./vector-embeddings.md)** - Semantic search setup
- **[Graph Queries](./graph-queries.md)** - Cypher query examples
- **[Performance Tuning](./performance-tuning.md)** - Optimization strategies

### Troubleshooting
- **[Troubleshooting Guide](./troubleshooting.md)** - Common issues and solutions
- **[Error Reference](./error-reference.md)** - Error codes and meanings

## Overview

The Knowledge Graph Indexer is a multi-layered system that:

1. **Validates** parsed code output against the unified schema
2. **Persists** entities to PostgreSQL (files, symbols, AST nodes, edges)
3. **Builds** graph relationships using Apache AGE
4. **Generates** vector embeddings for semantic search using pgvector

### Key Features

- **Multi-modal Storage**: Combines relational (PostgreSQL), graph (AGE), and vector (pgvector) databases
- **Incremental Updates**: Only processes changed files using checksums
- **Batch Processing**: Efficient bulk operations for large codebases
- **Parallel Execution**: Configurable worker count for concurrent processing
- **Error Resilience**: Continues processing on partial failures
- **Semantic Search**: Natural language code queries via embeddings

## Quick Start

### 1. Start the API Server

```bash
# Start database
make docker-up

# Run API server
make run-api
```

The API server will be available at `http://localhost:8080`.

### 2. Index a Repository

```bash
# Parse and index in one command
./bin/cli index --path /path/to/repo --api-url http://localhost:8080

# Or index from existing parse output
./bin/cli parse --path /path/to/repo --output parsed.json
./bin/cli index --input parsed.json --api-url http://localhost:8080
```

### 3. Search Indexed Code

```bash
# Semantic search
./bin/cli search --query "authentication function" --api-url http://localhost:8080

# Filter by language
./bin/cli search --query "database connection" --language go --api-url http://localhost:8080
```

## Architecture

```
┌─────────────┐
│  CLI Tool   │
└──────┬──────┘
       │ HTTP/JSON
       ↓
┌─────────────────────────────────────────┐
│           API Server                    │
│  ┌────────────────────────────────┐    │
│  │  Index Handler                 │    │
│  │  • Validate schema             │    │
│  │  • Coordinate pipeline         │    │
│  └────────────────────────────────┘    │
│                ↓                        │
│  ┌────────────────────────────────┐    │
│  │  Indexer Orchestrator          │    │
│  │  • Batch processing            │    │
│  │  • Parallel execution          │    │
│  │  • Error collection            │    │
│  └────────────────────────────────┘    │
│         ↓          ↓          ↓         │
│  ┌─────────┐ ┌──────────┐ ┌─────────┐ │
│  │ Writer  │ │  Graph   │ │Embedder │ │
│  │         │ │ Builder  │ │         │ │
│  └─────────┘ └──────────┘ └─────────┘ │
└─────────────────────────────────────────┘
         ↓          ↓          ↓
┌─────────────────────────────────────────┐
│         Storage Layer                   │
│  ┌──────────┐ ┌──────┐ ┌──────────┐   │
│  │PostgreSQL│ │ AGE  │ │ pgvector │   │
│  └──────────┘ └──────┘ └──────────┘   │
└─────────────────────────────────────────┘
```

## Core Components

### 1. Schema Validator
Validates parsed output against the unified schema, checking:
- Required fields (file_id, path, language)
- Referential integrity (symbol_id references in edges)
- Data types and constraints

### 2. Database Writer
Persists entities to PostgreSQL with:
- Batch insert operations
- ON CONFLICT handling for incremental updates
- Transaction management
- Retry logic with exponential backoff

### 3. Graph Builder
Constructs AGE graph nodes and edges:
- Maps symbol kinds to graph labels (Function, Class, Interface, Variable)
- Maps edge types to relationships (CALLS, IMPORTS, EXTENDS, IMPLEMENTS)
- Stores node and edge properties

### 4. Vector Embedder
Generates semantic embeddings:
- Supports local models (via LM Studio, vLLM)
- Supports OpenAI API
- Batch processing for efficiency
- Rate limiting and retry logic

### 5. Indexer Orchestrator
Coordinates the indexing pipeline:
- Parallel processing with configurable workers
- Incremental indexing using checksums
- Progress tracking and result summary
- Error collection and reporting

## Storage Schema

### PostgreSQL Tables

- **repositories**: Repository metadata
- **files**: File entities with checksums
- **symbols**: Functions, classes, interfaces, variables
- **ast_nodes**: Abstract syntax tree nodes
- **edges**: Dependency relationships
- **vectors**: Semantic embeddings (pgvector)
- **docstrings**: Documentation strings
- **summaries**: Semantic summaries

### AGE Graph Schema

```cypher
// Node Labels
(:Function {symbol_id, name, signature, file_path, start_line, end_line})
(:Class {symbol_id, name, signature, file_path, start_line, end_line})
(:Interface {symbol_id, name, signature, file_path, start_line, end_line})
(:Variable {symbol_id, name, signature, file_path, start_line, end_line})
(:Module {symbol_id, name, path})

// Relationship Types
(:Function)-[:CALLS]->(:Function)
(:Module)-[:IMPORTS]->(:Module)
(:Class)-[:EXTENDS]->(:Class)
(:Class)-[:IMPLEMENTS]->(:Interface)
(:Symbol)-[:REFERENCES]->(:Symbol)
```

## Common Workflows

### Index a New Repository

```bash
# 1. Parse the repository
./bin/cli parse --path /path/to/repo --output parsed.json

# 2. Index to knowledge graph
./bin/cli index \
  --input parsed.json \
  --repo-name "my-project" \
  --repo-url "https://github.com/user/repo" \
  --api-url http://localhost:8080
```

### Incremental Update

```bash
# Only process changed files
./bin/cli index \
  --path /path/to/repo \
  --incremental \
  --api-url http://localhost:8080
```

### Index Without Embeddings

```bash
# Skip vector generation for faster indexing
./bin/cli index \
  --path /path/to/repo \
  --skip-vectors \
  --api-url http://localhost:8080
```

### Search Indexed Code

```bash
# Semantic search
./bin/cli search \
  --query "user authentication" \
  --limit 10 \
  --api-url http://localhost:8080

# Filter by repository
./bin/cli search \
  --query "database query" \
  --repo-id <uuid> \
  --api-url http://localhost:8080
```

### Query Relationships

```bash
# Find callers of a function
curl http://localhost:8080/api/v1/symbols/<symbol-id>/callers

# Find callees
curl http://localhost:8080/api/v1/symbols/<symbol-id>/callees

# Find dependencies
curl http://localhost:8080/api/v1/symbols/<symbol-id>/dependencies
```

## Performance Characteristics

### Throughput
- **Small projects** (<100 files): <10 seconds
- **Medium projects** (100-1000 files): <2 minutes
- **Large projects** (1000+ files): <10 minutes

### Resource Usage
- **Memory**: ~100MB per 1000 files
- **Database**: ~1MB per 100 symbols
- **Embeddings**: ~3KB per symbol (768 dimensions)

### Optimization Tips
1. Use incremental indexing for updates
2. Skip embeddings initially, generate later
3. Increase batch size for large repositories
4. Adjust worker count based on CPU cores
5. Use connection pooling for concurrent operations

## Next Steps

- **[Quick Start Guide](./quick-start.md)** - Detailed setup instructions
- **[API Reference](./api-reference.md)** - Complete API documentation
- **[CLI Index Command](./cli-index-command.md)** - CLI usage guide
- **[Configuration Guide](./configuration.md)** - Tuning options
- **[Troubleshooting](./troubleshooting.md)** - Common issues

## Support

- **GitHub Issues**: Report bugs and request features
- **Documentation**: This directory
- **Examples**: See `example.http` for API examples

## License

MIT License - See [LICENSE](../../LICENSE) for details.
