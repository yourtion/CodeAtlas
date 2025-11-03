# Logging Configuration

CodeAtlas uses a structured logging system that can be controlled through environment variables and configuration.

## Database Logging

Database operations (connections, schema initialization, optimizations) use a configurable logger that can be controlled:

### Disable Logging (Default in Tests)

```go
import "github.com/yourtionguo/CodeAtlas/pkg/models"

// Disable all database logging
models.SetDBLogger(nil)
```

### Enable Verbose Logging

Set the `DB_VERBOSE` environment variable:

```bash
export DB_VERBOSE=true
go test ./...
```

Or in code:

```go
import (
    "github.com/yourtionguo/CodeAtlas/pkg/models"
    "github.com/yourtionguo/CodeAtlas/internal/utils"
)

// Enable verbose database logging
logger := utils.NewLogger(true)  // verbose=true
models.SetDBLogger(logger)
```

### Custom Logger

```go
import (
    "github.com/yourtionguo/CodeAtlas/pkg/models"
    "github.com/yourtionguo/CodeAtlas/internal/utils"
)

// Create custom logger
logger := utils.NewLogger(false)  // verbose=false for production
models.SetDBLogger(logger)
```

## Log Levels

The logger supports multiple levels:

- **Debug**: Detailed information for debugging (only shown when verbose=true)
- **Info**: General informational messages
- **Warn**: Warning messages for non-critical issues
- **Error**: Error messages for failures

## Test Environment

In tests, database logging is automatically disabled to reduce noise:

```go
func SetupTestDB(t *testing.T) *TestDB {
    // Disable database logging during tests
    models.SetDBLogger(nil)
    // ... rest of setup
}
```

## Production Environment

For production, use non-verbose logging:

```bash
# Don't set DB_VERBOSE, or set it to false
export DB_VERBOSE=false
```

## Indexer Logging

The indexer uses its own logger interface that can be configured:

```go
import "github.com/yourtionguo/CodeAtlas/internal/indexer"

config := &indexer.IndexerConfig{
    // ... other config
}

// Use custom logger
idx := indexer.NewIndexer(db, config)
idx.SetLogger(myLogger)  // Implement indexer.Logger interface
```

## Examples

### Quiet Tests

```bash
# Run tests with minimal output
go test ./...
```

### Verbose Tests

```bash
# Run tests with database debug logging
DB_VERBOSE=true go test -v ./...
```

### Production with Logging

```bash
# Run with info/warn/error logging only
DB_VERBOSE=false ./bin/api
```
