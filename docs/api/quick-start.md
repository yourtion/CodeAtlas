# CodeAtlas API Quick Start Guide

## Prerequisites

- Docker and Docker Compose installed
- Go 1.21+ (for local development)
- REST client (curl, Postman, or VS Code REST Client extension)

## Starting the Server

### Option 1: Using Docker Compose (Recommended)

```bash
# Start database and API server
make docker-up

# Check logs
docker-compose logs -f api

# Stop services
make docker-down
```

### Option 2: Local Development

```bash
# Start PostgreSQL
make docker-up

# In another terminal, run API server
make run-api

# Or with custom configuration
ENABLE_AUTH=true AUTH_TOKENS=dev-token make run-api
```

## Configuration

Create a `.env` file in the project root:

```bash
# Server
API_PORT=8080

# Authentication (optional)
ENABLE_AUTH=false
AUTH_TOKENS=token1,token2,token3

# CORS
CORS_ORIGINS=*

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=codeatlas
DB_NAME=codeatlas
```

## Testing the API

### 1. Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "message": "CodeAtlas API server is running"
}
```

### 2. Create a Repository

```bash
curl -X POST http://localhost:8080/api/v1/repositories \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-project",
    "url": "https://github.com/user/my-project",
    "branch": "main"
  }'
```

Save the `repo_id` from the response.

### 3. Index Code

```bash
curl -X POST http://localhost:8080/api/v1/index \
  -H "Content-Type: application/json" \
  -d '{
    "repo_name": "my-project",
    "repo_url": "https://github.com/user/my-project",
    "branch": "main",
    "parse_output": {
      "files": [
        {
          "path": "main.go",
          "language": "go",
          "size": 1024,
          "checksum": "sha256:abc123",
          "symbols": [
            {
              "name": "main",
              "kind": "function",
              "signature": "func main()",
              "start_line": 1,
              "end_line": 10,
              "docstring": "Main entry point"
            }
          ],
          "edges": []
        }
      ]
    }
  }'
```

### 4. List Repositories

```bash
curl http://localhost:8080/api/v1/repositories
```

### 5. Get Repository Details

```bash
curl http://localhost:8080/api/v1/repositories/<repo-id>
```

## Using with Authentication

When `ENABLE_AUTH=true`, include the Bearer token:

```bash
# Set your token
TOKEN="your-token-here"

# Make authenticated request
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/repositories
```

## Using the example.http File

If you're using VS Code with the REST Client extension:

1. Open `example.http` in VS Code
2. Update the variables at the top:
   ```
   @token = your-actual-token
   @repoId = actual-repo-id
   ```
3. Click "Send Request" above any request

## Common Workflows

### Workflow 1: Index a New Repository

```bash
# 1. Create repository
REPO_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/repositories \
  -H "Content-Type: application/json" \
  -d '{"name": "my-project", "branch": "main"}')

REPO_ID=$(echo $REPO_RESPONSE | jq -r '.repo_id')
echo "Created repository: $REPO_ID"

# 2. Index code (you need to parse your code first)
curl -X POST http://localhost:8080/api/v1/index \
  -H "Content-Type: application/json" \
  -d "{
    \"repo_id\": \"$REPO_ID\",
    \"repo_name\": \"my-project\",
    \"parse_output\": {
      \"files\": [...]
    }
  }"
```

### Workflow 2: Query Relationships

```bash
# 1. Get a symbol ID (from indexing response or search)
SYMBOL_ID="your-symbol-id"

# 2. Find who calls this symbol
curl http://localhost:8080/api/v1/symbols/$SYMBOL_ID/callers

# 3. Find what this symbol calls
curl http://localhost:8080/api/v1/symbols/$SYMBOL_ID/callees

# 4. Find dependencies
curl http://localhost:8080/api/v1/symbols/$SYMBOL_ID/dependencies
```

### Workflow 3: Semantic Search

```bash
# Note: You need to generate embeddings first
# This example assumes you have an embedding service

# 1. Generate embedding for your query
QUERY="authentication middleware"
EMBEDDING=$(python -c "import openai; print(openai.Embedding.create(input='$QUERY', model='text-embedding-ada-002')['data'][0]['embedding'])")

# 2. Search
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d "{
    \"query\": \"$QUERY\",
    \"embedding\": $EMBEDDING,
    \"limit\": 10
  }"
```

## Integration with CLI

The CLI tool can upload repositories to the API:

```bash
# Build CLI
make build-cli

# Upload repository
./bin/cli upload \
  -p /path/to/your/repo \
  -s http://localhost:8080 \
  -t your-token-here
```

## Troubleshooting

### Server won't start

```bash
# Check if port is already in use
lsof -i :8080

# Check database connection
docker-compose ps
docker-compose logs postgres
```

### Database connection errors

```bash
# Restart database
make docker-down
make docker-up

# Check database is ready
docker-compose exec postgres psql -U codeatlas -d codeatlas -c "SELECT 1;"
```

### Authentication errors

```bash
# Verify token is correct
echo $AUTH_TOKENS

# Test without auth
ENABLE_AUTH=false make run-api
```

### CORS errors

```bash
# Allow all origins for development
CORS_ORIGINS=* make run-api

# Or specific origin
CORS_ORIGINS=http://localhost:3000 make run-api
```

## Next Steps

- Read the [API Reference](./api-reference.md) for detailed endpoint documentation
- Check [Middleware Configuration](./middleware-and-configuration.md) for security setup
- Explore [Search and Relationships](./search-and-relationships.md) for advanced queries
- Review the [example.http](../../example.http) file for more examples

## Development Tips

### Enable Verbose Logging

```bash
# Set Gin to debug mode
GIN_MODE=debug make run-api
```

### Hot Reload

Use `air` for hot reload during development:

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

### Database Inspection

```bash
# Connect to database
docker-compose exec postgres psql -U codeatlas -d codeatlas

# List tables
\dt

# Query repositories
SELECT repo_id, name, branch FROM repositories;

# Query symbols
SELECT symbol_id, name, kind FROM symbols LIMIT 10;
```

### API Testing

```bash
# Run API tests
make test-api

# Run all tests
make test-all

# Run with coverage
make test-coverage-all
```

## Production Deployment

For production deployment:

1. **Enable authentication:**
   ```bash
   ENABLE_AUTH=true
   AUTH_TOKENS=secure-token-1,secure-token-2
   ```

2. **Restrict CORS:**
   ```bash
   CORS_ORIGINS=https://your-domain.com
   ```

3. **Use HTTPS:**
   - Deploy behind nginx or Caddy
   - Configure SSL certificates
   - Redirect HTTP to HTTPS

4. **Set up monitoring:**
   - Monitor `/health` endpoint
   - Set up logging aggregation
   - Configure alerts

5. **Database:**
   - Use managed PostgreSQL service
   - Enable backups
   - Configure connection pooling

See [Middleware Configuration](./middleware-and-configuration.md) for more production setup details.
