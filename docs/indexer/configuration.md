# Indexer Configuration Guide

Complete configuration reference for the Knowledge Graph Indexer.

## Table of Contents

- [Overview](#overview)
- [Database Configuration](#database-configuration)
- [Indexer Configuration](#indexer-configuration)
- [Embedder Configuration](#embedder-configuration)
- [API Server Configuration](#api-server-configuration)
- [Configuration Examples](#configuration-examples)
- [Performance Tuning](#performance-tuning)

## Overview

The indexer is configured through environment variables with sensible defaults. Configuration affects:

- Database connection and pooling
- Indexing batch size and parallelism
- Embedding generation
- API server behavior

## Database Configuration

### Connection Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DB_HOST` | string | `localhost` | PostgreSQL hostname |
| `DB_PORT` | int | `5432` | PostgreSQL port |
| `DB_USER` | string | `codeatlas` | Database username |
| `DB_PASSWORD` | string | `codeatlas` | Database password |
| `DB_NAME` | string | `codeatlas` | Database name |
| `DB_SSLMODE` | string | `disable` | SSL mode (disable, require, verify-ca, verify-full) |

**Example**:
```bash
export DB_HOST=db.example.com
export DB_PORT=5432
export DB_USER=codeatlas_prod
export DB_PASSWORD=secure-password
export DB_NAME=codeatlas_prod
export DB_SSLMODE=require
```

### Connection Pool Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DB_MAX_OPEN_CONNS` | int | `25` | Maximum open connections |
| `DB_MAX_IDLE_CONNS` | int | `5` | Maximum idle connections |
| `DB_CONN_MAX_LIFETIME` | duration | `5m` | Connection max lifetime |
| `DB_CONN_MAX_IDLE_TIME` | duration | `5m` | Connection max idle time |

**Tuning Guidelines**:

- **High throughput**: Increase `DB_MAX_OPEN_CONNS` to 50-100
- **Low resources**: Decrease to 10-15
- **Idle connections**: Set to 10-20% of max open connections
- **Lifetime**: Increase to 10-15m for stable connections

**Example**:
```bash
# High-throughput configuration
export DB_MAX_OPEN_CONNS=50
export DB_MAX_IDLE_CONNS=10
export DB_CONN_MAX_LIFETIME=10m
export DB_CONN_MAX_IDLE_TIME=10m
```

## Indexer Configuration

### Core Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `INDEXER_BATCH_SIZE` | int | `100` | Entities per batch |
| `INDEXER_WORKER_COUNT` | int | `4` | Concurrent workers |
| `INDEXER_SKIP_VECTORS` | bool | `false` | Skip embedding generation |
| `INDEXER_INCREMENTAL` | bool | `false` | Enable incremental indexing |
| `INDEXER_USE_TRANSACTIONS` | bool | `true` | Use database transactions |
| `INDEXER_GRAPH_NAME` | string | `code_graph` | AGE graph name |

**Batch Size Guidelines**:

- **Small repositories** (<100 files): 50-100
- **Medium repositories** (100-1000 files): 100-200
- **Large repositories** (>1000 files): 200-500
- **Memory constrained**: 25-50

**Worker Count Guidelines**:

- **Default**: Number of CPU cores
- **CPU-bound**: Match CPU core count
- **I/O-bound**: 2x CPU core count
- **Memory constrained**: 2-4 workers

**Example**:
```bash
# Optimized for large repositories
export INDEXER_BATCH_SIZE=200
export INDEXER_WORKER_COUNT=8
export INDEXER_SKIP_VECTORS=false
export INDEXER_INCREMENTAL=true
export INDEXER_USE_TRANSACTIONS=true
export INDEXER_GRAPH_NAME=code_graph
```

### Advanced Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `INDEXER_MAX_RETRIES` | int | `3` | Max retry attempts |
| `INDEXER_BASE_RETRY_DELAY` | duration | `100ms` | Initial retry delay |
| `INDEXER_MAX_RETRY_DELAY` | duration | `5s` | Maximum retry delay |
| `INDEXER_TIMEOUT` | duration | `30m` | Indexing timeout |

**Example**:
```bash
# Aggressive retry configuration
export INDEXER_MAX_RETRIES=5
export INDEXER_BASE_RETRY_DELAY=50ms
export INDEXER_MAX_RETRY_DELAY=10s
export INDEXER_TIMEOUT=60m
```

## Embedder Configuration

### Backend Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `EMBEDDING_BACKEND` | string | `openai` | Backend type (openai, local) |
| `EMBEDDING_API_ENDPOINT` | string | `http://localhost:1234/v1/embeddings` | API endpoint URL |
| `EMBEDDING_API_KEY` | string | `` | API key (optional for local) |
| `EMBEDDING_MODEL` | string | `text-embedding-qwen3-embedding-0.6b` | Model name |
| `EMBEDDING_DIMENSIONS` | int | `768` | Embedding dimensions |

**Supported Models**:

| Model | Dimensions | Backend | Cost |
|-------|------------|---------|------|
| `text-embedding-qwen3-embedding-0.6b` | 768 | Local/OpenAI | Free |
| `nomic-embed-text` | 768 | Local/OpenAI | Free |
| `text-embedding-3-small` | 1536 | OpenAI | $0.02/1M tokens |
| `text-embedding-3-large` | 3072 | OpenAI | $0.13/1M tokens |

**Example - Local Model**:
```bash
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
export EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
export EMBEDDING_DIMENSIONS=768
```

**Example - OpenAI**:
```bash
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=https://api.openai.com/v1/embeddings
export EMBEDDING_API_KEY=sk-your-api-key
export EMBEDDING_MODEL=text-embedding-3-small
export EMBEDDING_DIMENSIONS=1536
```

### Performance Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `EMBEDDING_BATCH_SIZE` | int | `50` | Texts per API call |
| `EMBEDDING_MAX_REQUESTS_PER_SECOND` | int | `10` | Rate limit |
| `EMBEDDING_MAX_RETRIES` | int | `3` | Max retry attempts |
| `EMBEDDING_BASE_RETRY_DELAY` | duration | `100ms` | Initial retry delay |
| `EMBEDDING_MAX_RETRY_DELAY` | duration | `5s` | Maximum retry delay |
| `EMBEDDING_TIMEOUT` | duration | `30s` | HTTP request timeout |

**Tuning Guidelines**:

- **OpenAI API**: Reduce rate limit to 5-10 req/s to avoid rate limiting
- **Local model**: Increase to 20-50 req/s based on hardware
- **Batch size**: Larger batches reduce API calls but increase latency
- **Timeout**: Increase for slow models or networks

**Example**:
```bash
# Optimized for OpenAI API
export EMBEDDING_BATCH_SIZE=100
export EMBEDDING_MAX_REQUESTS_PER_SECOND=50
export EMBEDDING_MAX_RETRIES=5
export EMBEDDING_TIMEOUT=60s
```

## API Server Configuration

### Server Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `API_HOST` | string | `0.0.0.0` | Server bind address |
| `API_PORT` | int | `8080` | Server port |
| `API_TIMEOUT` | duration | `30s` | Request timeout |

**Example**:
```bash
export API_HOST=0.0.0.0
export API_PORT=8080
export API_TIMEOUT=60s
```

### Authentication Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ENABLE_AUTH` | bool | `false` | Enable authentication |
| `AUTH_TOKENS` | string | `` | Comma-separated tokens |

**Example**:
```bash
export ENABLE_AUTH=true
export AUTH_TOKENS="token1,token2,token3"
```

### CORS Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `CORS_ORIGINS` | string | `*` | Allowed origins (comma-separated) |

**Example**:
```bash
export CORS_ORIGINS="http://localhost:3000,https://app.example.com"
```

## Configuration Examples

### Development Environment

```bash
# Database (Docker defaults)
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
export DB_NAME=codeatlas
export DB_MAX_OPEN_CONNS=10

# API Server
export API_PORT=8080
export ENABLE_AUTH=false

# Indexer (fast, no embeddings)
export INDEXER_BATCH_SIZE=50
export INDEXER_WORKER_COUNT=4
export INDEXER_SKIP_VECTORS=true

# No embedding configuration needed
```

### Production Environment

```bash
# Database (production)
export DB_HOST=db.production.example.com
export DB_PORT=5432
export DB_USER=codeatlas_prod
export DB_PASSWORD=<secure-password>
export DB_NAME=codeatlas_prod
export DB_SSLMODE=require
export DB_MAX_OPEN_CONNS=50
export DB_MAX_IDLE_CONNS=10
export DB_CONN_MAX_LIFETIME=10m

# API Server (with auth)
export API_HOST=0.0.0.0
export API_PORT=8080
export ENABLE_AUTH=true
export AUTH_TOKENS=<secure-token-1>,<secure-token-2>
export CORS_ORIGINS=https://app.example.com
export API_TIMEOUT=60s

# Indexer (high performance)
export INDEXER_BATCH_SIZE=200
export INDEXER_WORKER_COUNT=8
export INDEXER_USE_TRANSACTIONS=true
export INDEXER_INCREMENTAL=true

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

```bash
# Database (high connection pool)
export DB_MAX_OPEN_CONNS=100
export DB_MAX_IDLE_CONNS=20

# Indexer (maximum parallelism)
export INDEXER_BATCH_SIZE=500
export INDEXER_WORKER_COUNT=16
export INDEXER_SKIP_VECTORS=true  # Generate later

# Incremental updates
export INDEXER_INCREMENTAL=true
```

### Resource-Constrained Environment

```bash
# Database (minimal pool)
export DB_MAX_OPEN_CONNS=5
export DB_MAX_IDLE_CONNS=2

# Indexer (low resource usage)
export INDEXER_BATCH_SIZE=25
export INDEXER_WORKER_COUNT=2
export INDEXER_SKIP_VECTORS=true

# Embedder (if needed, use local)
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
export EMBEDDING_BATCH_SIZE=10
export EMBEDDING_MAX_REQUESTS_PER_SECOND=2
```

### Local Model Setup

```bash
# Start local embedding server (LM Studio or vLLM)
# LM Studio: Load model and start server on port 1234
# vLLM: docker run -p 8000:8000 vllm/vllm-openai --model text-embedding-qwen3-embedding-0.6b

# Configure indexer
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
export EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
export EMBEDDING_DIMENSIONS=768
export EMBEDDING_BATCH_SIZE=50
export EMBEDDING_MAX_REQUESTS_PER_SECOND=20
```

## Performance Tuning

### Optimize for Speed

```bash
# Skip embeddings initially
export INDEXER_SKIP_VECTORS=true

# Increase batch size
export INDEXER_BATCH_SIZE=500

# Increase workers
export INDEXER_WORKER_COUNT=16

# Increase database connections
export DB_MAX_OPEN_CONNS=100
```

### Optimize for Memory

```bash
# Reduce batch size
export INDEXER_BATCH_SIZE=25

# Reduce workers
export INDEXER_WORKER_COUNT=2

# Reduce database connections
export DB_MAX_OPEN_CONNS=5
```

### Optimize for Cost (Embeddings)

```bash
# Use local model
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings

# Or skip embeddings
export INDEXER_SKIP_VECTORS=true
```

### Optimize for Incremental Updates

```bash
# Enable incremental indexing
export INDEXER_INCREMENTAL=true

# Smaller batch size for faster feedback
export INDEXER_BATCH_SIZE=50

# Moderate workers
export INDEXER_WORKER_COUNT=4
```

## Configuration Files

### Using .env File

Create `.env`:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=codeatlas
DB_NAME=codeatlas

# Indexer
INDEXER_BATCH_SIZE=100
INDEXER_WORKER_COUNT=4

# Embedder
EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
```

Load and run:

```bash
export $(cat .env | xargs)
./bin/api
```

### Using Docker Compose

Edit `docker-compose.yml`:

```yaml
services:
  api:
    image: codeatlas-api:latest
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - INDEXER_BATCH_SIZE=200
      - INDEXER_WORKER_COUNT=8
      - EMBEDDING_API_ENDPOINT=http://embedder:8000/v1/embeddings
    depends_on:
      - postgres
      - embedder
  
  postgres:
    image: postgres:17-bookworm
    environment:
      - POSTGRES_DB=codeatlas
      - POSTGRES_USER=codeatlas
      - POSTGRES_PASSWORD=codeatlas
  
  embedder:
    image: vllm/vllm-openai:latest
    command: --model text-embedding-qwen3-embedding-0.6b
```

## Validation

The configuration system validates settings on startup:

### Common Validation Errors

**Database**:
- `database host cannot be empty` - Set `DB_HOST`
- `database port must be between 1 and 65535` - Check `DB_PORT`
- `max idle connections cannot exceed max open connections` - Adjust `DB_MAX_IDLE_CONNS`

**Indexer**:
- `batch size must be at least 1` - Check `INDEXER_BATCH_SIZE`
- `worker count must be at least 1` - Check `INDEXER_WORKER_COUNT`
- `graph name cannot be empty` - Set `INDEXER_GRAPH_NAME`

**Embedder**:
- `backend must be 'openai' or 'local'` - Check `EMBEDDING_BACKEND`
- `API endpoint cannot be empty` - Set `EMBEDDING_API_ENDPOINT`
- `dimensions must be at least 1` - Check `EMBEDDING_DIMENSIONS`

## Monitoring Configuration

### Log Configuration

```bash
# Set log level
export LOG_LEVEL=debug  # debug, info, warn, error

# Enable structured logging
export LOG_FORMAT=json  # json, text
```

### Metrics Configuration

```bash
# Enable metrics endpoint
export ENABLE_METRICS=true
export METRICS_PORT=9090
```

## Next Steps

- **[Quick Start](./quick-start.md)** - Get started quickly
- **[Performance Tuning](./performance-tuning.md)** - Optimization strategies
- **[Troubleshooting](./troubleshooting.md)** - Common issues
- **[Architecture](./architecture.md)** - System design
