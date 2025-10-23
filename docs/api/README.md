# CodeAtlas API Documentation

Welcome to the CodeAtlas API documentation. This directory contains comprehensive guides for using and integrating with the CodeAtlas API.

## Documentation Index

### Getting Started

- **[Quick Start Guide](./quick-start.md)** - Get up and running with the API in minutes
  - Server setup and configuration
  - Basic API testing
  - Common workflows
  - Troubleshooting tips

### API Reference

- **[API Reference](./api-reference.md)** - Complete API endpoint documentation
  - All endpoints with request/response examples
  - Authentication details
  - Error codes and responses
  - Data models

### Configuration

- **[Middleware and Configuration](./middleware-and-configuration.md)** - Server configuration guide
  - Authentication middleware
  - CORS configuration
  - Logging setup
  - Security best practices
  - Environment variables

### Advanced Topics

- **[Search and Relationships](./search-and-relationships.md)** - Advanced query capabilities
  - Semantic search
  - Code relationship queries
  - Graph traversal
  - Performance optimization

## Quick Links

### Essential Files

- **[example.http](../../example.http)** - HTTP request examples for testing
  - Ready-to-use API requests
  - Works with VS Code REST Client extension
  - Covers all endpoints

### Code Examples

```bash
# Health check
curl http://localhost:8080/health

# List repositories
curl http://localhost:8080/api/v1/repositories

# With authentication
curl -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/v1/repositories
```

## API Overview

### Base URL

```
http://localhost:8080
```

### Authentication

Most endpoints require Bearer token authentication when enabled:

```
Authorization: Bearer <your-token>
```

### Main Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (no auth) |
| `/api/v1/repositories` | GET | List repositories |
| `/api/v1/repositories/:id` | GET | Get repository |
| `/api/v1/repositories` | POST | Create repository |
| `/api/v1/index` | POST | Index code |
| `/api/v1/search` | POST | Semantic search |
| `/api/v1/symbols/:id/callers` | GET | Get callers |
| `/api/v1/symbols/:id/callees` | GET | Get callees |
| `/api/v1/symbols/:id/dependencies` | GET | Get dependencies |
| `/api/v1/files/:id/symbols` | GET | Get file symbols |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         API Server                          │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                    Middleware                        │  │
│  │  • Recovery  • Logging  • CORS  • Authentication    │  │
│  └──────────────────────────────────────────────────────┘  │
│                            ↓                                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                      Handlers                        │  │
│  │  • Index  • Search  • Relationships  • Repositories │  │
│  └──────────────────────────────────────────────────────┘  │
│                            ↓                                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   Data Layer                         │  │
│  │  • Models  • Repositories  • Database Access        │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                      │
│  • pgvector (embeddings)  • AGE (graph)  • JSONB          │
└─────────────────────────────────────────────────────────────┘
```

## Features

### Core Capabilities

- **Code Indexing**: Index parsed code into knowledge graph
- **Semantic Search**: Vector-based code search
- **Relationship Queries**: Call graphs, dependencies, file symbols
- **Repository Management**: CRUD operations for repositories
- **Authentication**: Token-based API security
- **CORS Support**: Cross-origin requests for web clients

### Technical Features

- RESTful API design
- JSON request/response format
- Bearer token authentication
- Structured logging
- Error handling with detailed messages
- Health check endpoint
- Middleware architecture

## Development

### Running Locally

```bash
# Start database
make docker-up

# Run API server
make run-api

# Run tests
make test-api
```

### Testing with example.http

1. Install VS Code REST Client extension
2. Open `example.http`
3. Update variables (token, IDs)
4. Click "Send Request"

### Building

```bash
# Build API binary
make build-api

# Run binary
./bin/api
```

## Configuration

### Environment Variables

```bash
# Server
API_PORT=8080

# Authentication
ENABLE_AUTH=true
AUTH_TOKENS=token1,token2

# CORS
CORS_ORIGINS=http://localhost:3000

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=codeatlas
DB_NAME=codeatlas
```

See [Middleware Configuration](./middleware-and-configuration.md) for details.

## Integration

### CLI Integration

```bash
# Upload repository via CLI
./bin/cli upload -p /path/to/repo -s http://localhost:8080 -t your-token
```

### Web Frontend Integration

```javascript
// Example: Fetch repositories
const response = await fetch('http://localhost:8080/api/v1/repositories', {
  headers: {
    'Authorization': 'Bearer your-token'
  }
});
const data = await response.json();
```

### Python Integration

```python
import requests

# Search code
response = requests.post(
    'http://localhost:8080/api/v1/search',
    headers={'Authorization': 'Bearer your-token'},
    json={
        'query': 'authentication',
        'embedding': [0.1, 0.2, ...],
        'limit': 10
    }
)
results = response.json()
```

## Support

### Common Issues

- **Connection refused**: Check if server is running (`make run-api`)
- **401 Unauthorized**: Verify token in `AUTH_TOKENS`
- **CORS errors**: Add origin to `CORS_ORIGINS`
- **Database errors**: Ensure PostgreSQL is running (`make docker-up`)

### Getting Help

1. Check the [Quick Start Guide](./quick-start.md)
2. Review [API Reference](./api-reference.md)
3. Try examples in [example.http](../../example.http)
4. Check server logs for errors

## Contributing

When adding new endpoints:

1. Add handler in `internal/api/handlers/`
2. Register route in `internal/api/server.go`
3. Add tests in `internal/api/handlers/*_test.go`
4. Update API documentation
5. Add examples to `example.http`

## Version

Current API version: **v1**

Breaking changes will result in a new version (`/api/v2/`).

## License

See project LICENSE file.
