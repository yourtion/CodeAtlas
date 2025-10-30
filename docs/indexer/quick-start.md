# Quick Start Guide

Get the Knowledge Graph Indexer up and running in minutes.

## Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for building from source)
- 4GB RAM minimum
- 10GB disk space

## Installation

### Option 1: Using Make (Recommended)

```bash
# Clone repository
git clone https://github.com/yourtionguo/CodeAtlas.git
cd CodeAtlas

# Build CLI and API
make build

# Start database
make docker-up

# Run API server
make run-api
```

### Option 2: Using Docker Compose

```bash
# Start all services
docker-compose up -d

# Check status
docker-compose ps
```

### Option 3: Manual Setup

```bash
# Build CLI
go build -o bin/cli ./cmd/cli

# Build API
go build -o bin/api ./cmd/api

# Start PostgreSQL with extensions
docker run -d \
  --name codeatlas-postgres \
  -e POSTGRES_DB=codeatlas \
  -e POSTGRES_USER=codeatlas \
  -e POSTGRES_PASSWORD=codeatlas \
  -p 5432:5432 \
  postgres:17-bookworm

# Install extensions
docker exec -it codeatlas-postgres psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS vector;"
docker exec -it codeatlas-postgres psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS age;"

# Run API server
./bin/api
```

## Verify Installation

### Check API Server

```bash
# Health check
curl http://localhost:8080/health

# Expected output:
# {"status":"ok","database":"connected","timestamp":"2025-10-29T10:30:00Z"}
```

### Check Database

```bash
# Connect to database
docker exec -it codeatlas-postgres psql -U codeatlas -d codeatlas

# Check extensions
SELECT * FROM pg_extension WHERE extname IN ('vector', 'age');

# Check tables
\dt

# Exit
\q
```

## First Indexing

### Step 1: Parse a Repository

```bash
# Parse a sample repository
./bin/cli parse --path /path/to/your/repo --output parsed.json

# Example output:
# Parsing repository...
# Parsed 150 files, 1250 symbols, 3400 edges
# Output written to parsed.json
```

### Step 2: Index to Knowledge Graph

```bash
# Index the parsed output
./bin/cli index \
  --input parsed.json \
  --repo-name "my-project" \
  --repo-url "https://github.com/user/my-project" \
  --api-url http://localhost:8080

# Example output:
# Indexing to knowledge graph...
# Repository ID: 550e8400-e29b-41d4-a716-446655440000
# Files processed: 150
# Symbols created: 1250
# Edges created: 3400
# Vectors created: 1250
# Duration: 45.2s
# ✓ Indexing completed successfully
```

### Step 3: Verify Indexing

```bash
# List repositories
curl http://localhost:8080/api/v1/repositories

# Get repository details
curl http://localhost:8080/api/v1/repositories/<repo-id>

# Search code
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "authentication function",
    "filters": {"limit": 5}
  }'
```

## Quick Examples

### Example 1: Index Go Project

```bash
# Parse and index in one command
./bin/cli index \
  --path ./mygoproject \
  --language go \
  --repo-name "My Go Project" \
  --api-url http://localhost:8080
```

### Example 2: Index with Skip Vectors (Fast)

```bash
# Skip embedding generation for faster indexing
./bin/cli index \
  --path ./large-project \
  --skip-vectors \
  --api-url http://localhost:8080
```

### Example 3: Incremental Update

```bash
# Initial index
./bin/cli index --path ./myproject --api-url http://localhost:8080

# Make changes to code
# ...

# Re-index only changed files
./bin/cli index --path ./myproject --incremental --api-url http://localhost:8080
```

### Example 4: Search Indexed Code

```bash
# Semantic search
./bin/cli search \
  --query "database connection" \
  --language go \
  --limit 10 \
  --api-url http://localhost:8080
```

## Configuration

### Environment Variables

Create a `.env` file:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=codeatlas
DB_NAME=codeatlas

# API Server
API_PORT=8080
ENABLE_AUTH=false

# Indexer
INDEXER_BATCH_SIZE=100
INDEXER_WORKER_COUNT=4
INDEXER_SKIP_VECTORS=false

# Embeddings (optional)
EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
EMBEDDING_DIMENSIONS=768
```

Load environment variables:

```bash
# Load .env file
export $(cat .env | xargs)

# Run API server
./bin/api
```

### Using Docker Compose

Edit `docker-compose.yml`:

```yaml
services:
  api:
    environment:
      - DB_HOST=postgres
      - INDEXER_BATCH_SIZE=200
      - INDEXER_WORKER_COUNT=8
```

## Common Workflows

### Workflow 1: Development Setup

```bash
# 1. Start services
make docker-up

# 2. Run API server
make run-api

# 3. Index your project
./bin/cli index --path . --api-url http://localhost:8080

# 4. Start web frontend (optional)
cd web
pnpm install
pnpm dev
```

### Workflow 2: CI/CD Integration

```yaml
# .github/workflows/index.yml
name: Index Codebase
on: [push]

jobs:
  index:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Build CLI
        run: make build-cli
      
      - name: Parse code
        run: ./bin/cli parse --path . --output parsed.json
      
      - name: Index to staging
        run: |
          ./bin/cli index \
            --input parsed.json \
            --repo-name "${{ github.repository }}" \
            --repo-url "${{ github.repositoryUrl }}" \
            --branch "${{ github.ref_name }}" \
            --commit-hash "${{ github.sha }}" \
            --api-url ${{ secrets.STAGING_API_URL }} \
            --api-token ${{ secrets.STAGING_API_TOKEN }}
```

### Workflow 3: Production Deployment

```bash
# 1. Build production images
docker build -f deployments/Dockerfile.api -t codeatlas-api:latest .
docker build -f deployments/Dockerfile.cli -t codeatlas-cli:latest .

# 2. Push to registry
docker push codeatlas-api:latest
docker push codeatlas-cli:latest

# 3. Deploy with docker-compose
docker-compose -f docker-compose.prod.yml up -d

# 4. Verify deployment
curl https://api.example.com/health
```

## Troubleshooting

### Issue: API Server Not Starting

```bash
# Check if port is in use
lsof -i :8080

# Check database connectivity
docker exec -it codeatlas-postgres psql -U codeatlas -d codeatlas -c "SELECT 1;"

# Check logs
docker logs codeatlas-api
```

### Issue: Database Connection Failed

```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Start PostgreSQL
make docker-up

# Check connection
psql -h localhost -U codeatlas -d codeatlas
```

### Issue: Indexing Fails

```bash
# Enable verbose logging
./bin/cli index --path . --verbose --api-url http://localhost:8080

# Check parse output
./bin/cli parse --path . --output parsed.json
cat parsed.json | jq .

# Check API server logs
docker logs codeatlas-api
```

### Issue: Slow Indexing

```bash
# Skip embeddings
./bin/cli index --path . --skip-vectors --api-url http://localhost:8080

# Increase batch size
./bin/cli index --path . --batch-size 200 --api-url http://localhost:8080

# Increase workers
./bin/cli index --path . --workers 8 --api-url http://localhost:8080
```

## Next Steps

### Learn More

- **[Architecture Overview](./architecture.md)** - System design and components
- **[API Reference](./api-reference.md)** - Complete API documentation
- **[CLI Documentation](./cli-index-command.md)** - CLI usage guide
- **[Configuration Guide](./configuration.md)** - Tuning options

### Advanced Topics

- **[Incremental Indexing](./incremental-indexing.md)** - Efficient updates
- **[Vector Embeddings](./vector-embeddings.md)** - Semantic search setup
- **[Graph Queries](./graph-queries.md)** - Cypher query examples
- **[Performance Tuning](./performance-tuning.md)** - Optimization strategies

### Examples

- **[API Examples](./api-examples.md)** - Practical API usage
- **[Integration Examples](./integration-examples.md)** - CI/CD integration
- **[Query Examples](./query-examples.md)** - Search and graph queries

## Getting Help

- **Documentation**: This directory
- **GitHub Issues**: Report bugs and request features
- **Discussions**: Ask questions and share ideas
- **Examples**: See `example.http` for API examples

## What's Next?

Now that you have the indexer running:

1. **Index your repositories**: Start with small projects and scale up
2. **Explore the API**: Try different search queries and relationship queries
3. **Integrate with CI/CD**: Automate indexing on code changes
4. **Tune performance**: Adjust configuration for your workload
5. **Build applications**: Use the API to build code intelligence tools

Happy indexing! 🚀
