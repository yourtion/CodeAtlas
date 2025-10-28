# Configuration Guide

This document describes all configuration options available in CodeAtlas. Configuration is managed through environment variables with sensible defaults.

## Table of Contents

- [Database Configuration](#database-configuration)
- [API Server Configuration](#api-server-configuration)
- [Indexer Configuration](#indexer-configuration)
- [Embedder Configuration](#embedder-configuration)
- [Configuration Examples](#configuration-examples)

## Database Configuration

PostgreSQL database connection and pool settings.

### Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DB_HOST` | string | `localhost` | PostgreSQL server hostname |
| `DB_PORT` | int | `5432` | PostgreSQL server port |
| `DB_USER` | string | `codeatlas` | Database username |
| `DB_PASSWORD` | string | `codeatlas` | Database password |
| `DB_NAME` | string | `codeatlas` | Database name |
| `DB_SSLMODE` | string | `disable` | SSL mode (`disable`, `require`, `verify-ca`, `verify-full`) |
| `DB_MAX_OPEN_CONNS` | int | `25` | Maximum number of open connections to the database |
| `DB_MAX_IDLE_CONNS` | int | `5` | Maximum number of idle connections in the pool |
| `DB_CONN_MAX_LIFETIME` | duration | `5m` | Maximum lifetime of a connection |
| `DB_CONN_MAX_IDLE_TIME` | duration | `5m` | Maximum idle time before closing a connection |

### Connection Pool Tuning

For high-throughput scenarios, consider increasing connection pool limits:

```bash
export DB_MAX_OPEN_CONNS=50
export DB_MAX_IDLE_CONNS=10
export DB_CONN_MAX_LIFETIME=10m
```

For low-resource environments, reduce the pool size:

```bash
export DB_MAX_OPEN_CONNS=10
export DB_MAX_IDLE_CONNS=2
```

## API Server Configuration

HTTP API server settings.

### Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `API_HOST` | string | `0.0.0.0` | Server bind address |
| `API_PORT` | int | `8080` | Server port |
| `ENABLE_AUTH` | bool | `false` | Enable authentication middleware |
| `AUTH_TOKENS` | string | `` | Comma-separated list of valid auth tokens |
| `CORS_ORIGINS` | string | `*` | Comma-separated list of allowed CORS origins |
| `API_TIMEOUT` | duration | `30s` | Request timeout |

### Authentication

To enable authentication:

```bash
export ENABLE_AUTH=true
export AUTH_TOKENS="token1,token2,token3"
```

Clients must include the token in the `Authorization` header:

```
Authorization: Bearer token1
```

### CORS Configuration

To restrict CORS origins:

```bash
export CORS_ORIGINS="http://localhost:3000,https://app.example.com"
```

## Indexer Configuration

Code indexing pipeline settings.

### Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `INDEXER_BATCH_SIZE` | int | `100` | Number of entities to process in each batch |
| `INDEXER_WORKER_COUNT` | int | `4` | Number of concurrent workers for parallel processing |
| `INDEXER_SKIP_VECTORS` | bool | `false` | Skip vector embedding generation |
| `INDEXER_INCREMENTAL` | bool | `false` | Enable incremental indexing (only process changed files) |
| `INDEXER_USE_TRANSACTIONS` | bool | `true` | Use database transactions for atomic operations |
| `INDEXER_GRAPH_NAME` | string | `code_graph` | Apache AGE graph name |
| `INDEXER_EMBEDDING_MODEL` | string | `` | Override embedding model (optional) |

### Performance Tuning

For large codebases, increase batch size and worker count:

```bash
export INDEXER_BATCH_SIZE=200
export INDEXER_WORKER_COUNT=8
```

For faster indexing without embeddings:

```bash
export INDEXER_SKIP_VECTORS=true
```

For incremental updates:

```bash
export INDEXER_INCREMENTAL=true
```

### Transaction Management

Transactions ensure atomicity but may impact performance for very large batches. To disable:

```bash
export INDEXER_USE_TRANSACTIONS=false
```

## Embedder Configuration

Vector embedding generation settings.

### Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `EMBEDDING_BACKEND` | string | `openai` | Backend type (`openai` or `local`) |
| `EMBEDDING_API_ENDPOINT` | string | `http://localhost:1234/v1/embeddings` | API endpoint URL |
| `EMBEDDING_API_KEY` | string | `` | API key for authentication (optional for local) |
| `EMBEDDING_MODEL` | string | `text-embedding-qwen3-embedding-0.6b` | Model name |
| `EMBEDDING_DIMENSIONS` | int | `768` | Expected embedding dimensions |
| `EMBEDDING_BATCH_SIZE` | int | `50` | Number of texts to embed in each API call |
| `EMBEDDING_MAX_REQUESTS_PER_SECOND` | int | `10` | Rate limit for API requests |
| `EMBEDDING_MAX_RETRIES` | int | `3` | Maximum retry attempts for failed requests |
| `EMBEDDING_BASE_RETRY_DELAY` | duration | `100ms` | Initial retry delay (exponential backoff) |
| `EMBEDDING_MAX_RETRY_DELAY` | duration | `5s` | Maximum retry delay |
| `EMBEDDING_TIMEOUT` | duration | `30s` | HTTP request timeout |

### Supported Models

The embedding dimensions must match your model:

| Model | Dimensions | Backend |
|-------|------------|---------|
| `text-embedding-qwen3-embedding-0.6b` | 768 | Local/OpenAI-compatible |
| `nomic-embed-text` | 768 | Local/OpenAI-compatible |
| `text-embedding-3-small` | 1536 | OpenAI |
| `text-embedding-3-large` | 3072 | OpenAI |

### OpenAI Configuration

For OpenAI API:

```bash
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=https://api.openai.com/v1/embeddings
export EMBEDDING_API_KEY=sk-...
export EMBEDDING_MODEL=text-embedding-3-small
export EMBEDDING_DIMENSIONS=1536
```

### Local Model Configuration

For local models via LM Studio or vLLM:

```bash
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
export EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
export EMBEDDING_DIMENSIONS=768
```

### Rate Limiting

To avoid hitting API rate limits:

```bash
export EMBEDDING_MAX_REQUESTS_PER_SECOND=5
export EMBEDDING_BATCH_SIZE=25
```

## Configuration Examples

### Development Environment

Minimal configuration for local development:

```bash
# Database (using Docker Compose defaults)
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
export DB_NAME=codeatlas

# API Server
export API_PORT=8080

# Indexer (fast, no embeddings)
export INDEXER_SKIP_VECTORS=true
export INDEXER_BATCH_SIZE=50
```

### Production Environment

Optimized configuration for production:

```bash
# Database (production PostgreSQL)
export DB_HOST=db.production.example.com
export DB_PORT=5432
export DB_USER=codeatlas_prod
export DB_PASSWORD=<secure-password>
export DB_NAME=codeatlas_prod
export DB_SSLMODE=require
export DB_MAX_OPEN_CONNS=50
export DB_MAX_IDLE_CONNS=10

# API Server (with authentication)
export API_HOST=0.0.0.0
export API_PORT=8080
export ENABLE_AUTH=true
export AUTH_TOKENS=<secure-token-1>,<secure-token-2>
export CORS_ORIGINS=https://app.example.com

# Indexer (high performance)
export INDEXER_BATCH_SIZE=200
export INDEXER_WORKER_COUNT=8
export INDEXER_USE_TRANSACTIONS=true

# Embedder (OpenAI)
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=https://api.openai.com/v1/embeddings
export EMBEDDING_API_KEY=<openai-api-key>
export EMBEDDING_MODEL=text-embedding-3-small
export EMBEDDING_DIMENSIONS=1536
export EMBEDDING_BATCH_SIZE=100
export EMBEDDING_MAX_REQUESTS_PER_SECOND=50
```

### High-Throughput Indexing

Configuration for indexing large codebases quickly:

```bash
# Database (high connection pool)
export DB_MAX_OPEN_CONNS=100
export DB_MAX_IDLE_CONNS=20

# Indexer (maximum parallelism)
export INDEXER_BATCH_SIZE=500
export INDEXER_WORKER_COUNT=16
export INDEXER_SKIP_VECTORS=true  # Generate embeddings later

# Process incrementally
export INDEXER_INCREMENTAL=true
```

### Resource-Constrained Environment

Configuration for limited resources:

```bash
# Database (minimal pool)
export DB_MAX_OPEN_CONNS=5
export DB_MAX_IDLE_CONNS=2

# Indexer (low resource usage)
export INDEXER_BATCH_SIZE=25
export INDEXER_WORKER_COUNT=2
export INDEXER_SKIP_VECTORS=true

# Embedder (if needed, use local model)
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
export EMBEDDING_BATCH_SIZE=10
export EMBEDDING_MAX_REQUESTS_PER_SECOND=2
```

## Configuration Validation

The configuration system validates all settings on startup. Common validation errors:

### Database Validation Errors

- `database host cannot be empty` - Set `DB_HOST`
- `database port must be between 1 and 65535` - Check `DB_PORT`
- `database max idle connections cannot exceed max open connections` - Adjust `DB_MAX_IDLE_CONNS`

### API Validation Errors

- `API port must be between 1 and 65535` - Check `API_PORT`
- `authentication is enabled but no auth tokens are configured` - Set `AUTH_TOKENS` when `ENABLE_AUTH=true`

### Indexer Validation Errors

- `indexer batch size must be at least 1` - Check `INDEXER_BATCH_SIZE`
- `indexer worker count must be at least 1` - Check `INDEXER_WORKER_COUNT`
- `indexer graph name cannot be empty` - Set `INDEXER_GRAPH_NAME`

### Embedder Validation Errors

- `embedder backend must be 'openai' or 'local'` - Check `EMBEDDING_BACKEND`
- `embedder API endpoint cannot be empty` - Set `EMBEDDING_API_ENDPOINT`
- `embedder dimensions must be at least 1` - Check `EMBEDDING_DIMENSIONS`

## Loading Configuration in Code

### Go Application

```go
import "github.com/yourtionguo/CodeAtlas/internal/config"

// Load configuration from environment
cfg, err := config.LoadConfig()
if err != nil {
    log.Fatal("Failed to load configuration:", err)
}

// Use configuration
db, err := sql.Open("postgres", cfg.Database.ConnectionString())
```

### CLI Tool

Configuration is automatically loaded when using the CLI:

```bash
# Override defaults with environment variables
export INDEXER_BATCH_SIZE=200
codeatlas index --path /path/to/repo
```

### Docker Compose

```yaml
services:
  api:
    image: codeatlas-api
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - API_PORT=8080
      - INDEXER_BATCH_SIZE=100
```

## Best Practices

1. **Use environment-specific configurations**: Maintain separate `.env` files for development, staging, and production
2. **Secure sensitive values**: Never commit passwords or API keys to version control
3. **Monitor resource usage**: Adjust connection pools and worker counts based on actual usage
4. **Start conservative**: Begin with default values and tune based on performance metrics
5. **Document overrides**: Keep a record of non-default configuration values and their rationale
6. **Validate early**: Run configuration validation before deploying to catch errors early

## Troubleshooting

### Connection Pool Exhaustion

If you see "too many connections" errors:

```bash
export DB_MAX_OPEN_CONNS=50  # Increase pool size
export DB_MAX_IDLE_CONNS=10
```

### Slow Indexing

To improve indexing performance:

```bash
export INDEXER_BATCH_SIZE=200      # Larger batches
export INDEXER_WORKER_COUNT=8      # More parallelism
export INDEXER_SKIP_VECTORS=true   # Skip embeddings initially
```

### Embedding API Rate Limits

If hitting rate limits:

```bash
export EMBEDDING_MAX_REQUESTS_PER_SECOND=5  # Reduce request rate
export EMBEDDING_BATCH_SIZE=25              # Smaller batches
export EMBEDDING_MAX_RETRIES=5              # More retries
```

### Memory Issues

To reduce memory usage:

```bash
export INDEXER_BATCH_SIZE=50       # Smaller batches
export INDEXER_WORKER_COUNT=2      # Fewer workers
export DB_MAX_OPEN_CONNS=10        # Smaller pool
```
