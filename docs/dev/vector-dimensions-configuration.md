# Vector Dimensions Configuration

This document explains how to configure vector dimensions in CodeAtlas to match your embedding model.

## Overview

CodeAtlas uses PostgreSQL with pgvector extension to store semantic embeddings. Different embedding models produce vectors of different dimensions:

| Model | Dimensions |
|-------|-----------|
| nomic-embed-text | 768 |
| text-embedding-qwen3-embedding-0.6b | 1024 |
| text-embedding-3-small (OpenAI) | 1536 |
| text-embedding-3-large (OpenAI) | 3072 |

The database schema must be configured with the correct vector dimension to match your embedding model.

## Quick Start

### For New Database

Set the dimension in your `.env` file before initializing:

```bash
# .env
EMBEDDING_DIMENSIONS=1536
EMBEDDING_MODEL=text-embedding-3-small

# Initialize database
make docker-up
make init-db
```

### For Existing Database

Change the dimension using the alter command:

```bash
# Change to 1536 dimensions (OpenAI)
make alter-vector-dimension VECTOR_DIM=1536

# Or use environment variable
export EMBEDDING_DIMENSIONS=1536
make alter-vector-dimension-from-env
```

## Configuration Methods

### Method 1: Using Makefile (Recommended)

```bash
# Change dimension to 1536 (OpenAI text-embedding-3-small)
make alter-vector-dimension VECTOR_DIM=1536

# Or use environment variable
export EMBEDDING_DIMENSIONS=1536
make alter-vector-dimension-from-env

# If you have existing data and want to clear it
make alter-vector-dimension-force VECTOR_DIM=1536
```

### Method 2: Using Go Tool

```bash
# Check what would change (dry-run)
go run scripts/alter_vector_dimension.go -dimension 1536 -dry-run

# Apply the change
go run scripts/alter_vector_dimension.go -dimension 1536

# Force change (truncates existing vectors)
go run scripts/alter_vector_dimension.go -dimension 1536 -force
```

### Method 3: Direct SQL

```bash
# Connect to database
psql -U codeatlas -d codeatlas

# Check current dimension
SELECT format_type(atttypid, atttypmod) 
FROM pg_attribute
WHERE attrelid = 'vectors'::regclass AND attname = 'embedding';

# Alter dimension (if table is empty or you've truncated it)
ALTER TABLE vectors ALTER COLUMN embedding TYPE vector(1536);

# Or use the SQL script
psql -U codeatlas -d codeatlas -v dimension=1536 -f scripts/alter_vector_dimension.sql
```

## Testing with Different Dimensions

When running tests, set the `VECTOR_DIMENSIONS` environment variable:

```bash
# Run tests with 1536 dimensions
export VECTOR_DIMENSIONS=1536
make test-integration

# Or inline
VECTOR_DIMENSIONS=768 go test ./tests/integration/...
```

The test utilities will automatically use the specified dimension when creating test databases.

## Docker Compose Setup

Update your `.env` file to match your embedding model:

```bash
# .env file
EMBEDDING_MODEL=text-embedding-3-small
EMBEDDING_DIMENSIONS=1536
VECTOR_DIMENSIONS=1536
```

Then regenerate schema and restart:

```bash
make generate-schema-from-env
make docker-down
make docker-up
make init-db
```

## Production Deployment

### Option 1: Pre-generate Schema

Before deployment, generate the schema with your desired dimensions:

```bash
# Generate schema
make generate-schema VECTOR_DIM=1536

# Commit the generated files
git add docker/initdb/01_init_schema.sql deployments/migrations/01_init_schema.sql
git commit -m "Update schema for 1536-dimensional vectors"
```

### Option 2: Dynamic Generation

Set `VECTOR_DIMENSIONS` in your deployment environment and generate schema during deployment:

```bash
# In your deployment script
export VECTOR_DIMENSIONS=1536
make generate-schema-from-env
make init-db
```

## Changing Vector Dimensions

⚠️ **Warning**: Changing vector dimensions will require re-indexing your data.

### Option 1: Alter Existing Table (Recommended)

If you have no data or can afford to lose existing vectors:

```bash
# Clear vectors and change dimension
make alter-vector-dimension-force VECTOR_DIM=1536

# Update your .env file
sed -i '' 's/EMBEDDING_DIMENSIONS=.*/EMBEDDING_DIMENSIONS=1536/' .env

# Re-index your repositories
./bin/cli parse -p /path/to/repo
```

### Option 2: Fresh Database

If you want a completely fresh start:

```bash
# Stop and remove database
make docker-down
docker volume rm codeatlas_pgdata

# Update configuration
export EMBEDDING_DIMENSIONS=1536

# Recreate database
make docker-up
make init-db

# Re-index your repositories
./bin/cli parse -p /path/to/repo
```

## Verification

After initialization, verify the vector dimension:

```sql
-- Connect to database
psql -U codeatlas -d codeatlas

-- Check vector column type
SELECT 
    table_name, 
    column_name, 
    data_type,
    udt_name
FROM information_schema.columns
WHERE table_name = 'vectors' AND column_name = 'embedding';

-- Should show: vector(1536) or your configured dimension
```

## Common Issues

### Dimension Mismatch Error

```
ERROR: dimension of vector (768) does not match column dimension (1536)
```

**Solution**: The embedding dimension doesn't match the database schema. Either:
- Regenerate schema with correct dimension
- Change embedding model to match schema

### Missing VECTOR_DIMENSIONS

```
Error: vector dimension must be specified
```

**Solution**: Set the environment variable:
```bash
export VECTOR_DIMENSIONS=1536
```

### Test Failures

If integration tests fail with dimension errors:

```bash
# Set dimension for tests
export VECTOR_DIMENSIONS=1024
make test-integration
```

## Best Practices

1. **Keep dimensions consistent**: Ensure `EMBEDDING_DIMENSIONS` and `VECTOR_DIMENSIONS` match
2. **Document your choice**: Add a comment in `.env` explaining which model you're using
3. **Version control**: Commit generated schema files for reproducibility
4. **Test before production**: Verify dimension configuration in staging environment
5. **Plan migrations**: Changing dimensions requires data migration

## Examples

### Local Development with Qwen

```bash
# .env
EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
EMBEDDING_DIMENSIONS=1024
VECTOR_DIMENSIONS=1024

# Setup
make generate-schema-from-env
make docker-up
make init-db
```

### Production with OpenAI

```bash
# .env
EMBEDDING_MODEL=text-embedding-3-small
EMBEDDING_API_KEY=sk-...
EMBEDDING_DIMENSIONS=1536
VECTOR_DIMENSIONS=1536

# Setup
make generate-schema-from-env
make docker-up
make init-db
```

### Testing with Custom Dimension

```bash
# Generate test schema
VECTOR_DIMENSIONS=768 make generate-schema-from-env

# Run tests
VECTOR_DIMENSIONS=768 make test-integration
```
