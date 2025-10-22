# Search and Relationship API Documentation

## Overview

This document describes the search and relationship query endpoints for the CodeAtlas knowledge graph API.

## Base URL

```
http://localhost:8080/api/v1
```

## Endpoints

### 1. Semantic Search

Search for code symbols using semantic similarity.

**Endpoint:** `POST /api/v1/search`

**Request Body:**

```json
{
  "query": "string (required) - Natural language search query",
  "embedding": [0.1, 0.2, ...] (required) - Query embedding vector",
  "repo_id": "string (optional) - Filter by repository ID",
  "language": "string (optional) - Filter by programming language (e.g., 'go', 'python')",
  "kind": ["string"] (optional) - Filter by symbol kinds (e.g., ['function', 'class'])",
  "limit": 10 (optional) - Maximum number of results (default: 10)"
}
```

**Response:**

```json
{
  "results": [
    {
      "symbol_id": "string - Unique symbol identifier",
      "name": "string - Symbol name",
      "kind": "string - Symbol kind (function, class, method, etc.)",
      "signature": "string - Symbol signature",
      "file_path": "string - File path relative to repository root",
      "docstring": "string - Documentation string (optional)",
      "similarity": 0.95 (float - Similarity score 0-1)"
    }
  ],
  "total": 1 (int - Total number of results)
}
```

**Example:**

```http
POST /api/v1/search
Content-Type: application/json

{
  "query": "function to parse JSON data",
  "embedding": [0.1, 0.2, 0.3, ...],
  "language": "go",
  "kind": ["function"],
  "limit": 5
}
```

**Status Codes:**
- `200 OK` - Search successful
- `400 Bad Request` - Invalid request (missing query or embedding)
- `500 Internal Server Error` - Search failed

---

### 2. Get Symbol Callers

Find all functions that call a specific symbol.

**Endpoint:** `GET /api/v1/symbols/:id/callers`

**Path Parameters:**
- `id` (required) - Symbol ID

**Response:**

```json
{
  "symbols": [
    {
      "symbol_id": "string - Caller symbol ID",
      "name": "string - Caller name",
      "kind": "string - Caller kind",
      "file_path": "string - File path",
      "signature": "string - Caller signature"
    }
  ],
  "total": 1 (int - Total number of callers)
}
```

**Example:**

```http
GET /api/v1/symbols/func-123/callers
```

**Status Codes:**
- `200 OK` - Query