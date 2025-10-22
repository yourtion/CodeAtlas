# API Server Middleware and Configuration

## Overview

The CodeAtlas API server includes comprehensive middleware for authentication, CORS, and logging. This document describes how to configure and use these features.

## Middleware Components

### 1. Authentication Middleware

The authentication middleware provides token-based authentication for API endpoints.

**Features:**
- Bearer token authentication
- Configurable token list
- Health check endpoint bypass
- Can be enabled/disabled via configuration

**Configuration:**

```bash
# Enable authentication
ENABLE_AUTH=true

# Comma-separated list of valid tokens
AUTH_TOKENS=token1,token2,token3
```

**Usage:**

```bash
# Make authenticated request
curl -H "Authorization: Bearer token1" http://localhost:8080/api/v1/repositories
```

**Behavior:**
- When disabled: All requests pass through without authentication
- When enabled: All endpoints except `/health` require valid Bearer token
- Invalid/missing tokens return 401 Unauthorized

### 2. CORS Middleware

The CORS middleware handles Cross-Origin Resource Sharing for web clients.

**Features:**
- Configurable allowed origins
- Wildcard support for development
- Preflight request handling
- Credential support

**Configuration:**

```bash
# Allow all origins (development)
CORS_ORIGINS=*

# Allow specific origins (production)
CORS_ORIGINS=http://example.com,https://app.example.com
```

**Default Headers:**
- Methods: GET, POST, PUT, DELETE, OPTIONS
- Headers: Origin, Content-Type, Accept, Authorization
- Credentials: Enabled
- Max Age: 86400 seconds (24 hours)

### 3. Logging Middleware

The logging middleware provides structured HTTP request logging.

**Features:**
- Request method, path, and status code
- Response latency in milliseconds
- Client IP address
- Query parameters
- Error messages
- Log level based on status code

**Log Levels:**
- INFO: 2xx and 3xx status codes
- WARN: 4xx status codes
- ERROR: 5xx status codes

**Example Log Output:**

```
INFO: 2025/10/22 21:45:26 method=GET path=/api/v1/repositories status=200 latency_ms=45 client_ip=192.168.1.100
WARN: 2025/10/22 21:45:27 method=POST path=/api/v1/index status=400 latency_ms=12 client_ip=192.168.1.100
ERROR: 2025/10/22 21:45:28 method=GET path=/api/v1/search status=500 latency_ms=234 client_ip=192.168.1.100 errors=database connection failed
```

## Server Configuration

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `API_PORT` | Server port | `8080` | `8080` |
| `ENABLE_AUTH` | Enable authentication | `false` | `true` |
| `AUTH_TOKENS` | Valid API tokens (comma-separated) | `""` | `token1,token2` |
| `CORS_ORIGINS` | Allowed CORS origins (comma-separated) | `*` | `http://localhost:3000` |

### Configuration Examples

#### Development Configuration

```bash
# .env.development
API_PORT=8080
ENABLE_AUTH=false
CORS_ORIGINS=*
```

#### Production Configuration

```bash
# .env.production
API_PORT=8080
ENABLE_AUTH=true
AUTH_TOKENS=prod-token-1,prod-token-2,prod-token-3
CORS_ORIGINS=https://app.example.com,https://dashboard.example.com
```

## API Endpoints

### Public Endpoints

These endpoints are accessible without authentication:

- `GET /health` - Health check endpoint

### Protected Endpoints

These endpoints require authentication when `ENABLE_AUTH=true`:

**Index:**
- `POST /api/v1/index` - Index parsed code

**Repositories:**
- `GET /api/v1/repositories` - List all repositories
- `GET /api/v1/repositories/:id` - Get repository by ID
- `POST /api/v1/repositories` - Create repository

**Search:**
- `POST /api/v1/search` - Semantic search

**Relationships:**
- `GET /api/v1/symbols/:id/callers` - Get functions that call this symbol
- `GET /api/v1/symbols/:id/callees` - Get functions called by this symbol
- `GET /api/v1/symbols/:id/dependencies` - Get symbol dependencies
- `GET /api/v1/files/:id/symbols` - Get symbols in a file

**Files:**
- `POST /api/v1/files` - Create file

**Commits:**
- `POST /api/v1/commits` - Create commit

## Starting the Server

### Using Make

```bash
# Start with default configuration
make run-api

# Start with Docker
make docker-up
```

### Using Go

```bash
# Development
go run cmd/api/main.go

# Production build
go build -o bin/api cmd/api/main.go
./bin/api
```

### Using Docker

```bash
# Build image
docker build -f deployments/Dockerfile.api -t codeatlas-api .

# Run container
docker run -p 8080:8080 \
  -e ENABLE_AUTH=true \
  -e AUTH_TOKENS=token1,token2 \
  -e CORS_ORIGINS=http://localhost:3000 \
  codeatlas-api
```

## Testing

### Health Check

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

### Authenticated Request

```bash
# Without token (when auth is enabled)
curl http://localhost:8080/api/v1/repositories
# Returns: 401 Unauthorized

# With valid token
curl -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/v1/repositories
# Returns: 200 OK with repository list
```

### CORS Preflight

```bash
curl -X OPTIONS http://localhost:8080/api/v1/repositories \
  -H "Origin: http://example.com" \
  -H "Access-Control-Request-Method: GET"
```

## Security Best Practices

### Production Deployment

1. **Always enable authentication in production:**
   ```bash
   ENABLE_AUTH=true
   ```

2. **Use strong, random tokens:**
   ```bash
   # Generate secure tokens
   openssl rand -hex 32
   ```

3. **Restrict CORS origins:**
   ```bash
   # Don't use wildcard in production
   CORS_ORIGINS=https://your-domain.com
   ```

4. **Use HTTPS:**
   - Deploy behind a reverse proxy (nginx, Caddy)
   - Enable TLS/SSL certificates
   - Redirect HTTP to HTTPS

5. **Rotate tokens regularly:**
   - Update AUTH_TOKENS periodically
   - Implement token expiration if needed

### Token Management

**Storing Tokens:**
- Use environment variables or secret management systems
- Never commit tokens to version control
- Use different tokens for different environments

**Distributing Tokens:**
- Share tokens securely (encrypted channels)
- Document which tokens are for which clients
- Revoke tokens when no longer needed

## Troubleshooting

### Authentication Issues

**Problem:** Getting 401 Unauthorized
- Check if `ENABLE_AUTH=true`
- Verify token is in `AUTH_TOKENS` list
- Ensure Authorization header format: `Bearer <token>`
- Check for whitespace in token

**Problem:** Health check returns 401
- Health check should always work without auth
- Check middleware order in server setup

### CORS Issues

**Problem:** CORS errors in browser
- Verify origin is in `CORS_ORIGINS` list
- Check browser console for specific error
- Ensure preflight requests are handled

**Problem:** Credentials not working
- CORS middleware sets `Access-Control-Allow-Credentials: true`
- Ensure client sends credentials with request

### Logging Issues

**Problem:** No logs appearing
- Check log output destination (stdout/stderr)
- Verify logging middleware is registered
- Check log level configuration

## Middleware Order

The middleware is applied in this order:

1. **Recovery** - Panic recovery
2. **Logging** - Request logging
3. **CORS** - CORS headers
4. **Authentication** - Token validation
5. **Routes** - Application routes

This order ensures:
- Panics are caught and logged
- All requests are logged
- CORS is handled before auth
- Auth is checked before route handlers

## Custom Middleware

To add custom middleware:

```go
// internal/api/middleware/custom.go
package middleware

import "github.com/gin-gonic/gin"

func Custom() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Pre-processing
        
        c.Next()
        
        // Post-processing
    }
}
```

Register in server setup:

```go
// internal/api/server.go
func (s *Server) SetupRouter() *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(middleware.Logging())
    r.Use(middleware.CORS(corsConfig))
    r.Use(middleware.Custom()) // Add here
    r.Use(middleware.Auth(authConfig))
    s.RegisterRoutes(r)
    return r
}
```
