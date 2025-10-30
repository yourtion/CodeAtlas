# Indexer API Reference

Complete reference for all Knowledge Graph Indexer API endpoints.

## Base URL

```
http://localhost:8080
```

## Authentication

When authentication is enabled, include a Bearer token in the Authorization header:

```
Authorization: Bearer <your-token>
```

Enable authentication with environment variables:

```bash
export ENABLE_AUTH=true
export AUTH_TOKENS="token1,token2,token3"
```

## Endpoints

### Health Check

Check API server health and database connectivity.

**Endpoint**: `GET /health`

**Authentication**: Not required

**Response**:

```json
{
  "status": "ok",
  "database": "connected",
  "timestamp": "2025-10-29T10:30:00Z"
}
```

**Status Codes**:
- `200 OK`: Server is healthy
- `503 Service Unavailable`: Database connection failed

**Example**:

```bash
curl http://localhost:8080/health
```

---

### Index Repository

Index parsed code output into the knowledge graph.

**Endpoint**: `POST /api/v1/index`

**Authentication**: Required (if enabled)

**Request Body**:

```json
{
  "repo_id": "550e8400-e29b-41d4-a716-446655440000",
  "repo_name": "my-project",
  "repo_url": "https://github.com/user/my-project",
  "branch": "main",
  "commit_hash": "abc123def456",
  "parse_output": {
    "files": [...],
    "relationships": [...],
    "metadata": {...}
  },
  "options": {
    "incremental": false,
    "skip_vectors": false,
    "batch_size": 100,
    "embedding_model": "text-embedding-qwen3-embedding-0.6b"
  }
}
```

**Request Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo_id` | string | No | Repository UUID (auto-generated if not provided) |
| `repo_name` | string | Yes | Repository name |
| `repo_url` | string | No | Repository URL |
| `branch` | string | No | Git branch (default: "main") |
| `commit_hash` | string | No | Git commit hash |
| `parse_output` | object | Yes | Parsed code output from CLI parser |
| `options` | object | No | Indexing options |

**Options Fields**:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `incremental` | boolean | false | Only process changed files |
| `skip_vectors` | boolean | false | Skip embedding generation |
| `batch_size` | integer | 100 | Batch size for processing |
| `embedding_model` | string | "" | Override default embedding model |

**Response**:

```json
{
  "repo_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "success",
  "files_processed": 150,
  "symbols_created": 1250,
  "edges_created": 3400,
  "vectors_created": 1250,
  "errors": [],
  "duration": "45.2s"
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `repo_id` | string | Repository UUID |
| `status` | string | "success", "partial", or "failed" |
| `files_processed` | integer | Number of files indexed |
| `symbols_created` | integer | Number of symbols created |
| `edges_created` | integer | Number of edges created |
| `vectors_created` | integer | Number of embeddings generated |
| `errors` | array | List of errors encountered |
| `duration` | string | Total indexing time |

**Error Response**:

```json
{
  "error": "validation failed",
  "message": "invalid parse output: missing required field 'files'",
  "details": {
    "field": "parse_output.files",
    "constraint": "required"
  }
}
```

**Status Codes**:
- `200 OK`: Indexing completed successfully
- `207 Multi-Status`: Partial success with some errors
- `400 Bad Request`: Invalid request body or validation failed
- `401 Unauthorized`: Missing or invalid authentication token
- `500 Internal Server Error`: Server error during indexing

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/index \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token1" \
  -d @index-request.json
```

---

### List Repositories

Get a list of all indexed repositories.

**Endpoint**: `GET /api/v1/repositories`

**Authentication**: Required (if enabled)

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 50 | Maximum number of results |
| `offset` | integer | 0 | Pagination offset |

**Response**:

```json
{
  "repositories": [
    {
      "repo_id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "my-project",
      "url": "https://github.com/user/my-project",
      "branch": "main",
      "commit_hash": "abc123def456",
      "created_at": "2025-10-29T10:00:00Z",
      "updated_at": "2025-10-29T10:30:00Z",
      "file_count": 150,
      "symbol_count": 1250
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

**Status Codes**:
- `200 OK`: Success
- `401 Unauthorized`: Missing or invalid authentication token
- `500 Internal Server Error`: Server error

**Example**:

```bash
curl http://localhost:8080/api/v1/repositories \
  -H "Authorization: Bearer token1"
```

---

### Get Repository

Get details of a specific repository.

**Endpoint**: `GET /api/v1/repositories/:id`

**Authentication**: Required (if enabled)

**Path Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Repository UUID |

**Response**:

```json
{
  "repo_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "my-project",
  "url": "https://github.com/user/my-project",
  "branch": "main",
  "commit_hash": "abc123def456",
  "metadata": {
    "languages": ["go", "javascript"],
    "total_lines": 50000
  },
  "created_at": "2025-10-29T10:00:00Z",
  "updated_at": "2025-10-29T10:30:00Z",
  "statistics": {
    "files": 150,
    "symbols": 1250,
    "edges": 3400,
    "vectors": 1250
  }
}
```

**Status Codes**:
- `200 OK`: Success
- `401 Unauthorized`: Missing or invalid authentication token
- `404 Not Found`: Repository not found
- `500 Internal Server Error`: Server error

**Example**:

```bash
curl http://localhost:8080/api/v1/repositories/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer token1"
```

---

### Semantic Search

Search indexed code using natural language queries.

**Endpoint**: `POST /api/v1/search`

**Authentication**: Required (if enabled)

**Request Body**:

```json
{
  "query": "user authentication function",
  "embedding": [0.1, 0.2, 0.3, ...],
  "filters": {
    "repo_id": "550e8400-e29b-41d4-a716-446655440000",
    "language": "go",
    "kind": ["function", "class"],
    "limit": 10
  }
}
```

**Request Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | string | Yes | Natural language search query |
| `embedding` | array | No | Pre-computed embedding vector (if not provided, will be generated) |
| `filters` | object | No | Search filters |

**Filter Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `repo_id` | string | Filter by repository UUID |
| `language` | string | Filter by language (go, javascript, python) |
| `kind` | array | Filter by symbol kind (function, class, interface, variable) |
| `limit` | integer | Maximum results (default: 10, max: 100) |

**Response**:

```json
{
  "results": [
    {
      "symbol_id": "770e8400-e29b-41d4-a716-446655440002",
      "name": "AuthenticateUser",
      "kind": "function",
      "signature": "func AuthenticateUser(username string, password string) (bool, error)",
      "file_path": "internal/auth/authenticate.go",
      "docstring": "AuthenticateUser verifies user credentials against the database",
      "similarity": 0.92
    }
  ],
  "total": 1,
  "query_time": "0.15s"
}
```

**Status Codes**:
- `200 OK`: Success
- `400 Bad Request`: Invalid query or filters
- `401 Unauthorized`: Missing or invalid authentication token
- `500 Internal Server Error`: Server error

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token1" \
  -d '{
    "query": "database connection",
    "filters": {
      "language": "go",
      "limit": 5
    }
  }'
```

---

### Get Symbol Callers

Find all functions that call a specific symbol.

**Endpoint**: `GET /api/v1/symbols/:id/callers`

**Authentication**: Required (if enabled)

**Path Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Symbol UUID |

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 50 | Maximum number of results |

**Response**:

```json
{
  "symbol_id": "770e8400-e29b-41d4-a716-446655440002",
  "callers": [
    {
      "symbol_id": "880e8400-e29b-41d4-a716-446655440003",
      "name": "LoginHandler",
      "kind": "function",
      "signature": "func LoginHandler(w http.ResponseWriter, r *http.Request)",
      "file_path": "internal/api/handlers/login.go",
      "call_site": {
        "line": 42,
        "context": "authenticated, err := AuthenticateUser(username, password)"
      }
    }
  ],
  "total": 1
}
```

**Status Codes**:
- `200 OK`: Success
- `401 Unauthorized`: Missing or invalid authentication token
- `404 Not Found`: Symbol not found
- `500 Internal Server Error`: Server error

**Example**:

```bash
curl http://localhost:8080/api/v1/symbols/770e8400-e29b-41d4-a716-446655440002/callers \
  -H "Authorization: Bearer token1"
```

---

### Get Symbol Callees

Find all functions called by a specific symbol.

**Endpoint**: `GET /api/v1/symbols/:id/callees`

**Authentication**: Required (if enabled)

**Path Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Symbol UUID |

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 50 | Maximum number of results |

**Response**:

```json
{
  "symbol_id": "880e8400-e29b-41d4-a716-446655440003",
  "callees": [
    {
      "symbol_id": "770e8400-e29b-41d4-a716-446655440002",
      "name": "AuthenticateUser",
      "kind": "function",
      "signature": "func AuthenticateUser(username string, password string) (bool, error)",
      "file_path": "internal/auth/authenticate.go"
    },
    {
      "symbol_id": "990e8400-e29b-41d4-a716-446655440004",
      "name": "CreateSession",
      "kind": "function",
      "signature": "func CreateSession(userID string) (string, error)",
      "file_path": "internal/session/session.go"
    }
  ],
  "total": 2
}
```

**Status Codes**:
- `200 OK`: Success
- `401 Unauthorized`: Missing or invalid authentication token
- `404 Not Found`: Symbol not found
- `500 Internal Server Error`: Server error

**Example**:

```bash
curl http://localhost:8080/api/v1/symbols/880e8400-e29b-41d4-a716-446655440003/callees \
  -H "Authorization: Bearer token1"
```

---

### Get Symbol Dependencies

Find all dependencies of a specific symbol.

**Endpoint**: `GET /api/v1/symbols/:id/dependencies`

**Authentication**: Required (if enabled)

**Path Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Symbol UUID |

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `depth` | integer | 1 | Dependency depth (1-5) |
| `limit` | integer | 50 | Maximum number of results |

**Response**:

```json
{
  "symbol_id": "880e8400-e29b-41d4-a716-446655440003",
  "dependencies": [
    {
      "symbol_id": "770e8400-e29b-41d4-a716-446655440002",
      "name": "AuthenticateUser",
      "kind": "function",
      "file_path": "internal/auth/authenticate.go",
      "relationship": "calls",
      "depth": 1
    },
    {
      "symbol_id": "aa0e8400-e29b-41d4-a716-446655440005",
      "name": "ValidatePassword",
      "kind": "function",
      "file_path": "internal/auth/validate.go",
      "relationship": "calls",
      "depth": 2
    }
  ],
  "total": 2
}
```

**Status Codes**:
- `200 OK`: Success
- `400 Bad Request`: Invalid depth parameter
- `401 Unauthorized`: Missing or invalid authentication token
- `404 Not Found`: Symbol not found
- `500 Internal Server Error`: Server error

**Example**:

```bash
curl "http://localhost:8080/api/v1/symbols/880e8400-e29b-41d4-a716-446655440003/dependencies?depth=2" \
  -H "Authorization: Bearer token1"
```

---

### Get File Symbols

Get all symbols defined in a specific file.

**Endpoint**: `GET /api/v1/files/:id/symbols`

**Authentication**: Required (if enabled)

**Path Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | File UUID |

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `kind` | string | all | Filter by symbol kind |

**Response**:

```json
{
  "file_id": "550e8400-e29b-41d4-a716-446655440000",
  "path": "internal/auth/authenticate.go",
  "language": "go",
  "symbols": [
    {
      "symbol_id": "770e8400-e29b-41d4-a716-446655440002",
      "name": "AuthenticateUser",
      "kind": "function",
      "signature": "func AuthenticateUser(username string, password string) (bool, error)",
      "start_line": 10,
      "end_line": 25,
      "docstring": "AuthenticateUser verifies user credentials"
    },
    {
      "symbol_id": "bb0e8400-e29b-41d4-a716-446655440006",
      "name": "User",
      "kind": "struct",
      "signature": "type User struct",
      "start_line": 30,
      "end_line": 35
    }
  ],
  "total": 2
}
```

**Status Codes**:
- `200 OK`: Success
- `401 Unauthorized`: Missing or invalid authentication token
- `404 Not Found`: File not found
- `500 Internal Server Error`: Server error

**Example**:

```bash
curl http://localhost:8080/api/v1/files/550e8400-e29b-41d4-a716-446655440000/symbols \
  -H "Authorization: Bearer token1"
```

---

## Error Responses

All endpoints return consistent error responses:

```json
{
  "error": "error_type",
  "message": "Human-readable error message",
  "details": {
    "field": "specific_field",
    "constraint": "validation_rule"
  }
}
```

### Common Error Types

| Error Type | HTTP Status | Description |
|------------|-------------|-------------|
| `validation_error` | 400 | Request validation failed |
| `authentication_error` | 401 | Missing or invalid token |
| `not_found` | 404 | Resource not found |
| `conflict` | 409 | Resource already exists |
| `internal_error` | 500 | Server error |
| `database_error` | 500 | Database operation failed |

## Rate Limiting

The API implements rate limiting per client:

- **Default**: 100 requests per minute
- **Burst**: 20 requests

Rate limit headers are included in responses:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1698580800
```

## Pagination

List endpoints support pagination:

```bash
# Get first page
curl "http://localhost:8080/api/v1/repositories?limit=10&offset=0"

# Get second page
curl "http://localhost:8080/api/v1/repositories?limit=10&offset=10"
```

## CORS

Configure allowed origins with environment variable:

```bash
export CORS_ORIGINS="http://localhost:3000,https://app.example.com"
```

## Examples

See `example.http` in the project root for complete request examples that can be used with VS Code REST Client extension.

## Client Libraries

### Go

```go
import "github.com/yourtionguo/CodeAtlas/pkg/client"

client := client.NewAPIClient("http://localhost:8080", "token1")

// Index repository
resp, err := client.Index(ctx, &client.IndexRequest{
    RepoName: "my-project",
    ParseOutput: parseOutput,
})

// Search
results, err := client.Search(ctx, "authentication", client.SearchFilters{
    Language: "go",
    Limit: 10,
})
```

### Python

```python
import requests

# Index repository
response = requests.post(
    'http://localhost:8080/api/v1/index',
    headers={'Authorization': 'Bearer token1'},
    json={
        'repo_name': 'my-project',
        'parse_output': parse_output
    }
)

# Search
response = requests.post(
    'http://localhost:8080/api/v1/search',
    headers={'Authorization': 'Bearer token1'},
    json={
        'query': 'authentication',
        'filters': {'language': 'go', 'limit': 10}
    }
)
```

### cURL

```bash
# Index
curl -X POST http://localhost:8080/api/v1/index \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token1" \
  -d @request.json

# Search
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token1" \
  -d '{"query": "authentication", "filters": {"limit": 10}}'
```

## Next Steps

- **[API Examples](./api-examples.md)** - Practical usage examples
- **[CLI Documentation](./cli-index-command.md)** - CLI usage
- **[Configuration](./configuration.md)** - Server configuration
- **[Troubleshooting](./troubleshooting.md)** - Common issues
