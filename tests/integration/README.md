# Integration Tests

This directory contains integration tests for the CodeAtlas knowledge graph indexer. These tests verify the complete end-to-end functionality including database operations, API handlers, and data integrity.

## Prerequisites

- PostgreSQL 17+ with pgvector extension installed
- Database connection configured via environment variables
- Go 1.21+

## Environment Variables

Set these environment variables to configure the test database connection:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
```

## Running Tests

### Quick Test (Skip Integration Tests)

```bash
go test -v -short ./tests/integration/...
```

### Full Integration Tests

```bash
go test -v ./tests/integration/...
```

### Specific Test

```bash
go test -v ./tests/integration/... -run TestEndToEndIndexing
```

### With Coverage

```bash
go test -v -coverprofile=coverage.out ./tests/integration/...
go tool cover -html=coverage.out
```

## Test Categories

### Core Integration Tests (`indexer_integration_test.go`)

- **TestEndToEndIndexing**: Complete parse → index → query workflow
- **TestIncrementalIndexing**: Tests incremental updates with file modifications
- **TestAPIEndpoints**: Tests API handlers with sample data
- **TestVectorSearch**: Tests semantic search functionality with pgvector
- **TestRelationshipQueries**: Tests callers, callees, and dependency queries

### Performance Tests (`performance_test.go`)

- **TestLargeScaleIndexing**: Tests indexing 100+ files with performance metrics
- **TestConcurrentIndexing**: Tests parallel indexing operations
- **TestMemoryUsage**: Tests memory efficiency with large AST trees
- **TestBatchOptimization**: Tests adaptive batch sizing

### API Integration Tests (`api_integration_test.go`)

- **TestIndexHandlerIntegration**: Tests the index handler with real database
- **TestSearchHandlerIntegration**: Tests the search handler with vector queries
- **TestRelationshipHandlerIntegration**: Tests relationship queries via API
- **TestInvalidRequests**: Tests error handling for invalid API requests

## Test Database Management

Each test creates a unique test database with a random name (e.g., `codeatlas_test_abc123`) and automatically cleans it up after completion. This ensures test isolation and prevents conflicts.

### Manual Cleanup

If tests are interrupted, you may need to manually clean up test databases:

```sql
-- List test databases
SELECT datname FROM pg_database WHERE datname LIKE 'codeatlas_test_%';

-- Drop test databases
DROP DATABASE IF EXISTS codeatlas_test_abc123;
```

## Test Utilities (`test_utils.go`)

- **SetupTestDB**: Creates a test database with full schema
- **TeardownTestDB**: Drops the test database and closes connections
- **CleanupTables**: Truncates all tables for test isolation
- **VerifyReferentialIntegrity**: Checks foreign key relationships

## Referential Integrity Checks

All integration tests verify referential integrity across tables:

- Files reference valid repositories
- Symbols reference valid files
- AST nodes reference valid files
- Edges reference valid source symbols
- Vectors reference valid entities

## Performance Targets

- **Throughput**: 10+ files/second (conservative target)
- **Concurrency**: Support 5+ parallel indexing operations
- **Memory**: Efficient streaming for large AST trees
- **Batch Optimization**: Adaptive batch sizing based on latency

## Troubleshooting

### Database Connection Errors

If you see connection errors, verify:

1. PostgreSQL is running: `pg_isready`
2. Database credentials are correct
3. pgvector extension is installed: `CREATE EXTENSION IF NOT EXISTS vector;`

### Test Timeouts

Integration tests have a 30-second timeout by default. For slower systems:

```bash
go test -v -timeout 60s ./tests/integration/...
```

### Permission Errors

Ensure the database user has permissions to:

- Create databases
- Create extensions
- Create tables and indexes

```sql
ALTER USER codeatlas CREATEDB;
ALTER USER codeatlas WITH SUPERUSER;  -- For extension creation
```

## CI/CD Integration

For CI/CD pipelines, use Docker Compose to set up the test database:

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:17-bookworm
    environment:
      POSTGRES_USER: codeatlas
      POSTGRES_PASSWORD: codeatlas
      POSTGRES_DB: postgres
    ports:
      - "5432:5432"
    command: postgres -c shared_preload_libraries=vector
```

Then run tests:

```bash
docker-compose up -d
go test -v ./tests/integration/...
docker-compose down
```
