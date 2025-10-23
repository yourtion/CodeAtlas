# CodeAtlas API Reference

## Base URL

```
http://localhost:8080
```

## Authentication

Most endpoints require authentication when `ENABLE_AUTH=true`. Include the Bearer token in the Authorization header:

```
Authorization: Bearer <your-token>
```

## Common Response Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Missing or invalid token |
| 404 | Not Found |
| 500 | Internal Server Error |

## Endpoints

### Health Check

#### GET /health

Check if the API server is running. No authentication required.

**Response:**

```json
{
  "status": "ok",
  "message": "CodeAtlas API server is running"
}
```

---

### Index

#### POST /api/v1/index

Index parsed code into the knowledge graph.

**Request Body:**

```json
{
  "repo_id": "optional-uuid",
  "repo_name": "my-project",
  "repo_url": "https://github.com/user/repo",
  "branch": "main",
  "commit_hash": "abc123",
  "parse_output": {
    "files": [
      {
        "path": "src/main.go",
        "language": "go",
        "size": 1024,
        "checksum": "sha256:...",
        "symbols": [
          {
            "name": "main",
            "kind": "function",
            "signature": "func main()",
            "start_line": 10,
            "end_line": 20,
            "docstring": "Main entry point",
            "semantic_summary": "Application entry point that initializes the server"
          }
        ],
        "edges": [
          {
            "source_symbol": "main",
            "target_symbol": "initServer",
            "edge_type": "call"
          }
        ]
      }
    ]
  },
  "options": {
    "incremental": false,
    "skip_vectors": false,
    "batch_size": 100,
    "worker_count": 4,
    "embedding_model": "text-embedding-ada-002"
  }
}
```

**Required Fields:**
- `repo_name`: Repository name
- `parse_output`: Parsed code structure
- `parse_output.files`: At least one file

**Optional Fields:**
- `repo_id`: Auto-generated if not provided
- `repo_url`: Repository URL
- `branch`: Default "main"
- `commit_hash`: Current commit
- `options`: Indexing configuration

**Response (200 OK):**

```json
{
  "repo_id": "uuid-here",
  "status": "success",
  "files_processed": 10,
  "symbols_created": 45,
  "edges_created": 78,
  "vectors_created": 45,
  "errors": [],
  "duration": "2.5s"
}
```

**Response (207 Multi-Status):**

```json
{
  "repo_id": "uuid-here",
  "status": "partial_success",
  "files_processed": 10,
  "symbols_created": 40,
  "edges_created": 70,
  "vectors_created": 40,
  "errors": [
    {
      "type": "vector_error",
      "message": "Failed to generate embedding",
      "entity_id": "symbol-123",
      "file_path": "src/utils.go",
      "retryable": true
    }
  ],
  "duration": "2.8s"
}
```

**Status Values:**
- `success`: All operations completed successfully
- `partial_success`: Some operations failed but indexing continued
- `success_with_warnings`: Completed with non-critical warnings
- `failed`: Indexing failed

---

### Repositories

#### GET /api/v1/repositories

List all repositories.

**Response (200 OK):**

```json
{
  "repositories": [
    {
      "repo_id": "uuid-1",
      "name": "my-project",
      "url": "https://github.com/user/repo",
      "branch": "main",
      "commit_hash": "abc123",
      "metadata": {
        "language": "go",
        "stars": 100
      },
      "created_at": "2025-10-22T10:00:00Z",
      "updated_at": "2025-10-22T12:00:00Z"
    }
  ],
  "total": 1
}
```

#### GET /api/v1/repositories/:id

Get a specific repository by ID.

**Path Parameters:**
- `id`: Repository UUID

**Response (200 OK):**

```json
{
  "repo_id": "uuid-1",
  "name": "my-project",
  "url": "https://github.com/user/repo",
  "branch": "main",
  "commit_hash": "abc123",
  "metadata": {
    "language": "go",
    "stars": 100
  },
  "created_at": "2025-10-22T10:00:00Z",
  "updated_at": "2025-10-22T12:00:00Z"
}
```

**Response (404 Not Found):**

```json
{
  "error": "Repository not found"
}
```

#### POST /api/v1/repositories

Create a new repository.

**Request Body:**

```json
{
  "name": "my-project",
  "url": "https://github.com/user/repo",
  "branch": "main"
}
```

**Required Fields:**
- `name`: Repository name

**Optional Fields:**
- `url`: Repository URL
- `branch`: Default "main"

**Response (201 Created):**

```json
{
  "repo_id": "uuid-1",
  "name": "my-project",
  "url": "https://github.com/user/repo",
  "branch": "main",
  "commit_hash": "",
  "metadata": {},
  "created_at": "2025-10-22T10:00:00Z",
  "updated_at": "2025-10-22T10:00:00Z"
}
```

---

### Search

#### POST /api/v1/search

Perform semantic search across code symbols.

**Request Body:**

```json
{
  "query": "authentication middleware",
  "embedding": [0.1, 0.2, 0.3, ...],
  "repo_id": "uuid-1",
  "language": "go",
  "kind": ["function", "class"],
  "limit": 10
}
```

**Required Fields:**
- `query`: Search query text
- `embedding`: Query embedding vector (768 dimensions)

**Optional Fields:**
- `repo_id`: Filter by repository
- `language`: Filter by programming language
- `kind`: Filter by symbol types
- `limit`: Max results (default 10)

**Response (200 OK):**

```json
{
  "results": [
    {
      "symbol_id": "symbol-123",
      "name": "AuthMiddleware",
      "kind": "function",
      "signature": "func AuthMiddleware() gin.HandlerFunc",
      "file_path": "internal/api/middleware/auth.go",
      "docstring": "AuthMiddleware provides token-based authentication",
      "similarity": 0.92
    }
  ],
  "total": 1
}
```

---

### Relationships

#### GET /api/v1/symbols/:id/callers

Get all functions that call the specified symbol.

**Path Parameters:**
- `id`: Symbol UUID

**Response (200 OK):**

```json
{
  "symbols": [
    {
      "symbol_id": "symbol-456",
      "name": "SetupRouter",
      "kind": "function",
      "file_path": "internal/api/server.go",
      "signature": "func (s *Server) SetupRouter() *gin.Engine"
    }
  ],
  "total": 1
}
```

**Response (404 Not Found):**

```json
{
  "error": "Symbol not found"
}
```

#### GET /api/v1/symbols/:id/callees

Get all functions called by the specified symbol.

**Path Parameters:**
- `id`: Symbol UUID

**Response (200 OK):**

```json
{
  "symbols": [
    {
      "symbol_id": "symbol-789",
      "name": "validateToken",
      "kind": "function",
      "file_path": "internal/api/middleware/auth.go",
      "signature": "func validateToken(token string) bool"
    }
  ],
  "total": 1
}
```

#### GET /api/v1/symbols/:id/dependencies

Get all dependencies of the specified symbol (imports, extends, implements).

**Path Parameters:**
- `id`: Symbol UUID

**Response (200 OK):**

```json
{
  "dependencies": [
    {
      "symbol_id": "symbol-999",
      "name": "gin.HandlerFunc",
      "kind": "type",
      "file_path": "vendor/github.com/gin-gonic/gin/context.go",
      "module": "github.com/gin-gonic/gin",
      "edge_type": "import",
      "signature": "type HandlerFunc func(*Context)"
    }
  ],
  "total": 1
}
```

#### GET /api/v1/files/:id/symbols

Get all symbols defined in a file.

**Path Parameters:**
- `id`: File UUID

**Response (200 OK):**

```json
{
  "symbols": [
    {
      "symbol_id": "symbol-123",
      "name": "AuthMiddleware",
      "kind": "function",
      "signature": "func AuthMiddleware() gin.HandlerFunc",
      "start_line": 10,
      "end_line": 50,
      "docstring": "AuthMiddleware provides token-based authentication",
      "semantic_summary": "Validates Bearer tokens and returns 401 for invalid requests"
    }
  ],
  "total": 1
}
```

**Response (404 Not Found):**

```json
{
  "error": "File not found"
}
```

---

### Files

#### POST /api/v1/files

Create a new file record.

**Request Body:**

```json
{
  "repository_id": "uuid-1",
  "path": "src/main.go",
  "content": "package main...",
  "language": "go",
  "size": 1024
}
```

**Required Fields:**
- `repository_id`: Repository UUID
- `path`: File path

**Optional Fields:**
- `content`: File content
- `language`: Programming language
- `size`: File size in bytes

**Response (201 Created):**

```json
{
  "file_id": "file-uuid",
  "repo_id": "uuid-1",
  "path": "src/main.go",
  "language": "go",
  "size": 1024,
  "checksum": "",
  "created_at": "2025-10-22T10:00:00Z",
  "updated_at": "2025-10-22T10:00:00Z"
}
```

---

### Commits

#### POST /api/v1/commits

Create a new commit record.

**Request Body:**

```json
{
  "repository_id": 1,
  "hash": "abc123def456",
  "author": "John Doe",
  "email": "john@example.com",
  "message": "Add authentication middleware",
  "timestamp": "2025-10-22T10:00:00Z"
}
```

**Required Fields:**
- `repository_id`: Repository ID
- `hash`: Commit hash
- `author`: Author name
- `email`: Author email
- `message`: Commit message
- `timestamp`: Commit timestamp

**Response (501 Not Implemented):**

```json
{
  "error": "Not implemented yet"
}
```

---

## Error Responses

### 400 Bad Request

```json
{
  "error": "Invalid request body",
  "details": "missing required field: repo_name"
}
```

### 401 Unauthorized

```json
{
  "error": "Missing authorization header"
}
```

```json
{
  "error": "Invalid authorization header format. Expected: Bearer <token>"
}
```

```json
{
  "error": "Invalid or expired token"
}
```

### 404 Not Found

```json
{
  "error": "Repository not found"
}
```

### 500 Internal Server Error

```json
{
  "error": "Failed to retrieve repositories",
  "details": "database connection failed"
}
```

---

## Data Models

### Symbol Kinds

- `function`: Function or method
- `class`: Class or struct
- `interface`: Interface
- `variable`: Variable or constant
- `type`: Type definition
- `module`: Module or package

### Edge Types

- `call`: Function call relationship
- `import`: Import/dependency relationship
- `extends`: Inheritance relationship
- `implements`: Interface implementation
- `reference`: Reference to another symbol

### Languages

Supported programming languages:
- `go`
- `python`
- `javascript`
- `typescript`
- `java`
- `c`
- `cpp`
- `rust`
- And more...

---

## Rate Limiting

Currently, there is no rate limiting implemented. This may be added in future versions.

---

## Pagination

Currently, pagination is not implemented for list endpoints. All results are returned in a single response. This may be added in future versions with `page` and `page_size` query parameters.

---

## Versioning

The API is currently at version 1 (`/api/v1/`). Breaking changes will result in a new version (`/api/v2/`).
