# Logging and Error Handling Architecture

## Overview

CodeAtlas uses a unified logging and error handling architecture across all components. This document describes the architecture, conventions, and best practices.

## Architecture

### Core Components

```
internal/utils/
├── logger.go              # Core logger implementation with structured logging
└── logger_structured_test.go  # Tests for structured logging

internal/indexer/
├── errors.go              # Indexer-specific error types
├── errors_test.go         # Error handling tests
├── logger_adapter.go      # Adapter for indexer logger interface
└── logger_adapter_test.go # Adapter tests

internal/schema/
└── types.go               # Schema-level error types (ParseError)

internal/parser/
└── go_parser.go           # Parser-specific error types (DetailedParseError)

pkg/client/
└── api_client.go          # API client error types (APIError, IndexError)

internal/api/handlers/
└── index_handler.go       # API handler error types (IndexError)
```

## Logger Architecture

### 1. Core Logger (`internal/utils/logger.go`)

The core logger provides structured logging with multiple levels:

```go
type Logger struct {
    verbose bool
    infoLog *log.Logger
    warnLog *log.Logger
    errLog  *log.Logger
    dbgLog  *log.Logger
}

type Field struct {
    Key   string
    Value interface{}
}
```

**Features:**
- Basic logging: `Info()`, `Warn()`, `Error()`, `Debug()`
- Formatted logging: `Infof()`, `Warnf()`, `Errorf()`, `Debugf()`
- Structured logging: `InfoWithFields()`, `WarnWithFields()`, `ErrorWithFields()`, `DebugWithFields()`
- Verbose mode support for debug logging
- Smart value formatting (strings, durations, times, errors)

**Usage:**
```go
logger := utils.NewLogger(verbose)

// Basic logging
logger.Info("operation started")
logger.Warn("potential issue detected")
logger.Error("operation failed")
logger.Debug("detailed debug info")

// Structured logging
logger.InfoWithFields("indexing started",
    utils.Field{Key: "repo_id", Value: repoID},
    utils.Field{Key: "files_count", Value: len(files)},
)

logger.ErrorWithFields("database write failed", err,
    utils.Field{Key: "entity_id", Value: entityID},
    utils.Field{Key: "file_path", Value: filePath},
)
```

### 2. Indexer Logger Interface (`internal/indexer/indexer.go`)

The indexer defines its own logger interface for dependency injection:

```go
type IndexerLogger interface {
    Info(msg string, args ...interface{})
    Warn(msg string, args ...interface{})
    Error(msg string, args ...interface{})
    Debug(msg string, args ...interface{})
    InfoWithFields(msg string, fields ...LogField)
    WarnWithFields(msg string, fields ...LogField)
    ErrorWithFields(msg string, err error, fields ...LogField)
    DebugWithFields(msg string, fields ...LogField)
}

type LogField struct {
    Key   string
    Value interface{}
}
```

### 3. Logger Adapter (`internal/indexer/logger_adapter.go`)

The adapter bridges `utils.Logger` with `IndexerLogger`:

```go
type LoggerAdapter struct {
    logger *utils.Logger
}

func NewLoggerAdapter(logger *utils.Logger) *LoggerAdapter {
    return &LoggerAdapter{logger: logger}
}
```

**Usage:**
```go
utilsLogger := utils.NewLogger(verbose)
indexerLogger := indexer.NewLoggerAdapter(utilsLogger)
idx := indexer.NewIndexerWithLogger(db, config, indexerLogger)
```

### 4. Parser Progress Logger (`internal/parser/parser_pool.go`)

The parser uses a specialized progress logger interface:

```go
type ProgressLogger interface {
    LogProgress(file string, current, total int)
    LogError(file string, err error)
}

type DefaultProgressLogger struct {
    mu sync.Mutex
}
```

## Error Handling Architecture

### 1. Indexer Errors (`internal/indexer/errors.go`)

The most comprehensive error system for indexing operations:

```go
type IndexerError struct {
    Type      IndexerErrorType
    Message   string
    EntityID  string
    FilePath  string
    Cause     error
    Retryable bool
}

type IndexerErrorType string

const (
    ErrorTypeValidation  IndexerErrorType = "validation"
    ErrorTypeDatabase    IndexerErrorType = "database"
    ErrorTypeGraph       IndexerErrorType = "graph"
    ErrorTypeEmbedding   IndexerErrorType = "embedding"
    ErrorTypeTransaction IndexerErrorType = "transaction"
    ErrorTypeNotFound    IndexerErrorType = "not_found"
    ErrorTypeConflict    IndexerErrorType = "conflict"
    ErrorTypeTimeout     IndexerErrorType = "timeout"
    ErrorTypeConnection  IndexerErrorType = "connection"
)
```

**Error Collector:**
```go
type ErrorCollector struct {
    errors []error
}

// Methods
- Add(err error)
- HasErrors() bool
- Count() int
- FilterRetryable() []error
- FilterNonRetryable() []error
- GroupByType() map[IndexerErrorType][]error
- Summary() map[string]int
```

### 2. Schema Errors (`internal/schema/types.go`)

Errors for parsing operations:

```go
type ParseError struct {
    File    string
    Line    int
    Column  int
    Message string
    Type    ErrorType
}

type ErrorType string

const (
    ErrorFileSystem ErrorType = "filesystem"
    ErrorParse      ErrorType = "parse"
    ErrorMapping    ErrorType = "mapping"
    ErrorLLM        ErrorType = "llm"
    ErrorOutput     ErrorType = "output"
)
```

### 3. Parser Errors (`internal/parser/go_parser.go`)

Detailed parsing errors with location:

```go
type DetailedParseError struct {
    File    string
    Line    int
    Column  int
    Message string
    Type    string // filesystem, parse, mapping
}
```

### 4. API Client Errors (`pkg/client/api_client.go`)

Errors for API communication:

```go
type APIError struct {
    StatusCode int
    Message    string
    Details    interface{}
}

type IndexError struct {
    Type      string
    Message   string
    EntityID  string
    FilePath  string
    Retryable bool
}
```

### 5. API Handler Errors (`internal/api/handlers/index_handler.go`)

Errors for HTTP API responses:

```go
type IndexError struct {
    Type      string
    Message   string
    EntityID  string
    FilePath  string
    Retryable bool
}
```

## Error Type Mapping

Different components use different error types for their specific needs:

| Component | Error Type | Purpose |
|-----------|-----------|---------|
| Indexer | `IndexerError` | Comprehensive indexing errors with context |
| Schema | `ParseError` | Parsing operation errors |
| Parser | `DetailedParseError` | Detailed parsing errors with location |
| API Client | `APIError`, `IndexError` | HTTP API communication errors |
| API Handler | `IndexError` | HTTP response errors |

## Unified Conventions

### 1. Error Context

All error types should include:
- **Message**: Human-readable error description
- **Type/Category**: Error classification
- **Location**: File path, line number (when applicable)
- **Entity ID**: Affected entity identifier (when applicable)
- **Retryability**: Whether the operation can be retried

### 2. Structured Logging Fields

Use consistent field names across the codebase:

| Field Name | Description | Example |
|------------|-------------|---------|
| `repo_id` | Repository identifier | `"abc-123"` |
| `repo_name` | Repository name | `"MyProject"` |
| `file_id` | File identifier | `"file-456"` |
| `file_path` | File path | `"src/main.go"` |
| `symbol_id` | Symbol identifier | `"symbol-789"` |
| `entity_id` | Generic entity identifier | `"entity-123"` |
| `entity_type` | Type of entity | `"file"`, `"symbol"`, `"node"` |
| `error` | Error message | `"connection refused"` |
| `duration` | Operation duration | `5.2s` |
| `status` | Operation status | `"success"`, `"failed"` |
| `count` | Count of items | `150` |

### 3. Log Levels

Use appropriate log levels:

- **Debug**: Detailed diagnostic information (verbose mode only)
  - Internal state changes
  - Detailed operation steps
  - Performance metrics

- **Info**: Normal operational messages
  - Operation start/completion
  - Progress updates
  - Success messages

- **Warn**: Recoverable errors, degraded functionality
  - Non-fatal errors
  - Retryable failures
  - Performance degradation

- **Error**: Unrecoverable errors, operation failures
  - Fatal errors
  - Non-retryable failures
  - System errors

### 4. Error Handling Patterns

#### Validation Errors (Non-retryable)
```go
if validationResult.HasErrors() {
    logger.ErrorWithFields("validation failed", nil,
        utils.Field{Key: "error_count", Value: validationResult.ErrorCount()},
    )
    return nil, fmt.Errorf("validation failed")
}
```

#### Database Errors (May be retryable)
```go
err := writer.WriteFiles(ctx, repoID, files)
if err != nil {
    logger.ErrorWithFields("database write failed", err,
        utils.Field{Key: "repo_id", Value: repoID},
        utils.Field{Key: "retryable", Value: isRetryable(err)},
    )
    return err
}
```

#### Batch Operations (Collect errors)
```go
errorCollector := indexer.NewErrorCollector()

for _, file := range files {
    if err := processFile(file); err != nil {
        logger.WarnWithFields("file processing failed",
            utils.Field{Key: "file_path", Value: file.Path},
        )
        errorCollector.Add(err)
        continue
    }
}

if errorCollector.HasErrors() {
    logger.WarnWithFields("batch processing completed with errors",
        utils.Field{Key: "total_errors", Value: errorCollector.Count()},
        utils.Field{Key: "error_summary", Value: errorCollector.Summary()},
    )
}
```

## Best Practices

### 1. Use Structured Logging

✅ **Good:**
```go
logger.InfoWithFields("indexing started",
    utils.Field{Key: "repo_id", Value: repoID},
    utils.Field{Key: "files_count", Value: len(files)},
)
```

❌ **Bad:**
```go
logger.Info(fmt.Sprintf("indexing started for repo %s with %d files", repoID, len(files)))
```

### 2. Include Context in Errors

✅ **Good:**
```go
return indexer.NewDatabaseError(
    "failed to write symbol",
    symbolID,
    filePath,
    err,
    true,
)
```

❌ **Bad:**
```go
return fmt.Errorf("failed to write symbol: %w", err)
```

### 3. Log Operation Boundaries

✅ **Good:**
```go
logger.InfoWithFields("starting operation",
    utils.Field{Key: "operation", Value: "indexing"},
)
// ... perform operation ...
logger.InfoWithFields("operation completed",
    utils.Field{Key: "duration", Value: duration},
    utils.Field{Key: "status", Value: status},
)
```

### 4. Use Error Collectors for Batch Operations

✅ **Good:**
```go
errorCollector := indexer.NewErrorCollector()
for _, item := range items {
    if err := process(item); err != nil {
        errorCollector.Add(err)
        continue // Process remaining items
    }
}
```

❌ **Bad:**
```go
for _, item := range items {
    if err := process(item); err != nil {
        return err // Stops processing
    }
}
```

### 5. Provide Actionable Error Messages

✅ **Good:**
```go
return fmt.Errorf("failed to connect to database at %s:%d: %w", host, port, err)
```

❌ **Bad:**
```go
return fmt.Errorf("database error: %w", err)
```

## Testing

### Logger Tests

```go
// Test structured logging
func TestLoggerInfoWithFields(t *testing.T) {
    var buf bytes.Buffer
    logger := utils.NewLogger(false)
    logger.infoLog = log.New(&buf, "INFO: ", 0)
    
    logger.InfoWithFields("test message",
        utils.Field{Key: "key1", Value: "value1"},
        utils.Field{Key: "key2", Value: 42},
    )
    
    output := buf.String()
    assert.Contains(t, output, "test message")
    assert.Contains(t, output, "key1=value1")
    assert.Contains(t, output, "key2=42")
}
```

### Error Tests

```go
// Test error creation
func TestIndexerError(t *testing.T) {
    cause := errors.New("underlying error")
    err := indexer.NewValidationError("invalid data", "entity-123", "file.go", cause)
    
    assert.Equal(t, indexer.ErrorTypeValidation, err.Type)
    assert.Equal(t, "invalid data", err.Message)
    assert.Equal(t, "entity-123", err.EntityID)
    assert.Equal(t, "file.go", err.FilePath)
    assert.False(t, err.Retryable)
}

// Test error collector
func TestErrorCollector(t *testing.T) {
    collector := indexer.NewErrorCollector()
    
    collector.Add(indexer.NewValidationError("error1", "id1", "file1", nil))
    collector.Add(indexer.NewDatabaseError("error2", "id2", "file2", nil, true))
    
    assert.True(t, collector.HasErrors())
    assert.Equal(t, 2, collector.Count())
    
    retryable := collector.FilterRetryable()
    assert.Len(t, retryable, 1)
}
```

## Migration Guide

### For New Components

1. Use `utils.Logger` for all logging needs
2. Use structured logging with `InfoWithFields()`, etc.
3. Define component-specific error types if needed
4. Use `indexer.ErrorCollector` for batch operations
5. Follow consistent field naming conventions

### For Existing Components

1. Replace custom loggers with `utils.Logger`
2. Convert string formatting to structured logging
3. Add error context (entity IDs, file paths)
4. Implement error collectors for batch operations
5. Add comprehensive tests

## Summary

CodeAtlas uses a unified logging and error handling architecture:

- **Core Logger**: `internal/utils/logger.go` - Structured logging for all components
- **Indexer Errors**: `internal/indexer/errors.go` - Comprehensive error handling with context
- **Error Collectors**: Aggregate errors during batch operations
- **Consistent Conventions**: Unified field names and error patterns
- **Comprehensive Testing**: Full test coverage for all components

This architecture provides:
- ✅ Structured, parseable logs
- ✅ Rich error context
- ✅ Retryability information
- ✅ Batch error handling
- ✅ Easy debugging and monitoring
- ✅ Consistent patterns across codebase
