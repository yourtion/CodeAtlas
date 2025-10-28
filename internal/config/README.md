# Configuration Package

This package provides centralized configuration management for CodeAtlas using environment variables with sensible defaults and validation.

## Features

- **Environment-based configuration**: All settings loaded from environment variables
- **Sensible defaults**: Works out-of-the-box for development
- **Validation**: Comprehensive validation on startup to catch configuration errors early
- **Type-safe**: Strongly-typed configuration structs
- **Well-documented**: Clear documentation for all options

## Usage

### Loading Configuration

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

### Configuration Sections

The configuration is organized into four main sections:

1. **Database**: PostgreSQL connection and pool settings
2. **API**: HTTP server configuration
3. **Indexer**: Code indexing pipeline settings
4. **Embedder**: Vector embedding generation settings

### Example

```go
// Access database configuration
fmt.Printf("Connecting to %s:%d\n", cfg.Database.Host, cfg.Database.Port)

// Access API configuration
fmt.Printf("Starting server on %s\n", cfg.API.Address())

// Access indexer configuration
fmt.Printf("Using %d workers with batch size %d\n", 
    cfg.Indexer.WorkerCount, cfg.Indexer.BatchSize)

// Access embedder configuration
fmt.Printf("Using %s model with %d dimensions\n",
    cfg.Embedder.Model, cfg.Embedder.Dimensions)
```

## Configuration Structs

### Config

Main configuration struct containing all subsections:

```go
type Config struct {
    Database DatabaseConfig
    API      APIConfig
    Indexer  IndexerConfig
    Embedder EmbedderConfig
}
```

### DatabaseConfig

PostgreSQL database configuration:

```go
type DatabaseConfig struct {
    Host            string        // DB_HOST
    Port            int           // DB_PORT
    User            string        // DB_USER
    Password        string        // DB_PASSWORD
    Database        string        // DB_NAME
    SSLMode         string        // DB_SSLMODE
    MaxOpenConns    int           // DB_MAX_OPEN_CONNS
    MaxIdleConns    int           // DB_MAX_IDLE_CONNS
    ConnMaxLifetime time.Duration // DB_CONN_MAX_LIFETIME
    ConnMaxIdleTime time.Duration // DB_CONN_MAX_IDLE_TIME
}
```

### APIConfig

API server configuration:

```go
type APIConfig struct {
    Host        string        // API_HOST
    Port        int           // API_PORT
    EnableAuth  bool          // ENABLE_AUTH
    AuthTokens  []string      // AUTH_TOKENS (comma-separated)
    CORSOrigins []string      // CORS_ORIGINS (comma-separated)
    Timeout     time.Duration // API_TIMEOUT
}
```

### IndexerConfig

Indexer pipeline configuration:

```go
type IndexerConfig struct {
    BatchSize       int    // INDEXER_BATCH_SIZE
    WorkerCount     int    // INDEXER_WORKER_COUNT
    SkipVectors     bool   // INDEXER_SKIP_VECTORS
    Incremental     bool   // INDEXER_INCREMENTAL
    UseTransactions bool   // INDEXER_USE_TRANSACTIONS
    GraphName       string // INDEXER_GRAPH_NAME
    EmbeddingModel  string // INDEXER_EMBEDDING_MODEL
}
```

### EmbedderConfig

Embedder configuration:

```go
type EmbedderConfig struct {
    Backend              string        // EMBEDDING_BACKEND
    APIEndpoint          string        // EMBEDDING_API_ENDPOINT
    APIKey               string        // EMBEDDING_API_KEY
    Model                string        // EMBEDDING_MODEL
    Dimensions           int           // EMBEDDING_DIMENSIONS
    BatchSize            int           // EMBEDDING_BATCH_SIZE
    MaxRequestsPerSecond int           // EMBEDDING_MAX_REQUESTS_PER_SECOND
    MaxRetries           int           // EMBEDDING_MAX_RETRIES
    BaseRetryDelay       time.Duration // EMBEDDING_BASE_RETRY_DELAY
    MaxRetryDelay        time.Duration // EMBEDDING_MAX_RETRY_DELAY
    Timeout              time.Duration // EMBEDDING_TIMEOUT
}
```

## Validation

The configuration system validates all settings on load:

```go
cfg, err := config.LoadConfig()
if err != nil {
    // Configuration validation failed
    log.Fatal(err)
}
```

Common validation errors:

- Empty required fields (host, user, database name)
- Invalid port numbers (must be 1-65535)
- Invalid connection pool settings (idle > max)
- Authentication enabled without tokens
- Invalid embedder backend type
- Invalid batch sizes or worker counts

## Helper Methods

### ConnectionString

Generate PostgreSQL connection string:

```go
connStr := cfg.Database.ConnectionString()
// Returns: "host=localhost port=5432 user=codeatlas password=codeatlas dbname=codeatlas sslmode=disable"
```

### Address

Get API server address:

```go
address := cfg.API.Address()
// Returns: "0.0.0.0:8080"
```

## Environment Variables

All configuration is loaded from environment variables. See the main [Configuration Guide](../../docs/configuration.md) for complete documentation.

### Quick Reference

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=codeatlas
DB_NAME=codeatlas

# API
API_PORT=8080
ENABLE_AUTH=false

# Indexer
INDEXER_BATCH_SIZE=100
INDEXER_WORKER_COUNT=4

# Embedder
EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
EMBEDDING_DIMENSIONS=768
```

## Testing

The package includes comprehensive tests:

```bash
go test ./internal/config/...
```

Tests cover:
- Default value loading
- Custom environment variable parsing
- Configuration validation
- Helper functions
- Type conversions

## Best Practices

1. **Load once**: Load configuration at application startup
2. **Validate early**: Configuration is validated on load
3. **Use defaults**: Default values work for development
4. **Document overrides**: Keep track of non-default values
5. **Secure secrets**: Never commit passwords or API keys

## Examples

### Development

```bash
# Use defaults
go run cmd/api/main.go
```

### Production

```bash
# Set production values
export DB_HOST=db.production.example.com
export DB_PASSWORD=secure-password
export ENABLE_AUTH=true
export AUTH_TOKENS=token1,token2
go run cmd/api/main.go
```

### Docker

```yaml
services:
  api:
    image: codeatlas-api
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - API_PORT=8080
      - INDEXER_BATCH_SIZE=200
```

## See Also

- [Configuration Guide](../../docs/configuration.md) - Complete configuration documentation
- [.env.example](../../.env.example) - Example configuration file
