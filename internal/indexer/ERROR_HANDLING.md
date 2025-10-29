# Error Handling and Logging Guide

## Overview

The indexer implements comprehensive error handling and structured logging to ensure robust operation and easy troubleshooting. This document describes the error handling patterns, logging strategies, and best practices.

## Error Types

### IndexerError

The `IndexerError` type provides structured error information with context:

```go
type IndexerError struct {
    Type      IndexerErrorType  // Category of error
    Message   string            // Human-readable message
    EntityID  string            // ID of affected entity (optional)
    FilePath  string            // Path of affected file (optional)
    Cause     error             // Underlying error (optional)
    Retryable bool              // Whether operation can be retried
}
```

### Error Categories

| Type | Description | Retryable | Use Case |
|------|-------------|-----------|----------|
| `ErrorTypeValidation` | Schema validation failures | No | Invalid input data, missing required fields |
| `ErrorTypeDatabase` | Database operation failures | Depends | Connection issues, constraint violations |
| `ErrorTypeGraph` | Graph construction failures | Yes | AGE graph operations, Cypher query errors |
| `ErrorTypeEmbedding` | Vector embedding failures | Yes | API failures, rate limits, model errors |
| `ErrorTypeTransaction` | Transaction failures | No | Commit/rollback errors, deadlocks |
| `ErrorTypeNotFound` | Entity not found | No | Missing references, deleted entities |
| `ErrorTypeConflict` | Constraint conflicts | No | Duplicate keys, unique violations |
| `ErrorTypeTimeout` | Operation timeouts | Yes | Network timeouts, slow queries |
| `ErrorTypeConnection` | Connection failures | Yes | Database unavailable, network issues |

### Creating Errors

Use constructor functions to create typed errors:

```go
// Validation error
err := NewValidationError(
    "missing required field 'name'",
    "symbol-123",
    "file.go",
    nil,
)

// Database error (retryable)
err := NewDatabaseError(
    "connection timeout",
    "file-456",
    "",
    cause,
    true, // retryable
)

// Graph error (always retryable)
err := NewGraphError(
    "failed to create node",
    "symbol-789",
    "file.go",
    cause,
)

// Embedding error
err := NewEmbeddingError(
    "API rate limit exceeded",
    "symbol-123",
    "",
    cause,
    true, // retryable
)
```

## Error Collection

### ErrorCollector

The `ErrorCollector` aggregates errors during batch operations:

```go
collector := NewErrorCollector()

// Add errors as they occur
for _, item := range items {
    if err := processItem(item); err != nil {
        collector.Add(err)
    }
}

// Check if any errors occurred
if collector.HasErrors() {
    fmt.Printf("Encountered %d errors\n", collector.Count())
    
    // Get error summary by type
    summary := collector.Summary()
    for errType, count := range summary {
        fmt.Printf("%s: %d\n", errType, count)
    }
    
    // Filter retryable errors
    retryable := collector.FilterRetryable()
    if len(retryable) > 0 {
        // Retry these operations
    }
    
    // Group errors by type
    groups := collector.GroupByType()
    for errType, errors := range groups {
        fmt.Printf("%s errors:\n", errType)
        for _, err := range errors {
            fmt.Printf("  - %v\n", err)
        }
    }
}
```

### Methods

- `Add(err error)` - Add an error to the collection
- `HasErrors() bool` - Check if any errors exist
- `Count() int` - Get total error count
- `First() error` - Get the first error
- `Errors() []error` - Get all errors
- `Clear()` - Remove all errors
- `FilterRetryable() []error` - Get only retryable errors
- `FilterNonRetryable() []error` - Get only non-retryable errors
- `GroupByType() map[IndexerErrorType][]error` - Group errors by type
- `Summary() map[string]int` - Get error counts by type

## Structured Logging

### Logger Interface

The indexer uses the `IndexerLogger` interface for logging:

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
```

### Structured Fields

Use `LogField` to add context to log messages:

```go
logger.InfoWithFields("indexing started",
    LogField{Key: "repo_id", Value: repoID},
    LogField{Key: "files_count", Value: len(files)},
    LogField{Key: "incremental", Value: true},
)

logger.ErrorWithFields("database write failed", err,
    LogField{Key: "entity_type", Value: "symbol"},
    LogField{Key: "entity_id", Value: symbolID},
    LogField{Key: "file_path", Value: filePath},
)
```

### Field Types

Supported field value types:
- `string` - Quoted if contains spaces
- `int`, `int64`, `float64` - Numeric values
- `bool` - Boolean values
- `time.Duration` - Formatted as duration string (e.g., "5s")
- `time.Time` - Formatted as RFC3339
- `error` - Quoted error message

### Logger Adapter

Use `LoggerAdapter` to bridge `utils.Logger` with `IndexerLogger`:

```go
utilsLogger := utils.NewLogger(verbose)
indexerLogger := NewLoggerAdapter(utilsLogger)
indexer := NewIndexerWithLogger(db, config, indexerLogger)
```

## Error Handling Patterns

### Validation Errors

Validation errors are non-retryable and should fail fast:

```go
validationResult := validator.Validate(input)
if validationResult.HasErrors() {
    for _, valErr := range validationResult.Errors {
        logger.ErrorWithFields("validation failed", nil,
            LogField{Key: "entity_id", Value: valErr.EntityID},
            LogField{Key: "file_path", Value: valErr.FilePath},
            LogField{Key: "message", Value: valErr.Message},
        )
    }
    return nil, fmt.Errorf("validation failed with %d errors", validationResult.ErrorCount())
}
```

### Database Errors

Database errors may be retryable (connection issues) or non-retryable (constraint violations):

```go
err := writer.WriteFiles(ctx, repoID, files)
if err != nil {
    if isRetryable(err) {
        logger.WarnWithFields("database write failed, will retry", 
            LogField{Key: "attempt", Value: attempt},
            LogField{Key: "error", Value: err.Error()},
        )
        // Retry with backoff
    } else {
        logger.ErrorWithFields("database write failed permanently", err,
            LogField{Key: "repo_id", Value: repoID},
        )
        return err
    }
}
```

### Graph Errors

Graph errors are generally non-fatal and retryable:

```go
graphResult := graphBuilder.CreateNodes(ctx, symbols)
for _, graphErr := range graphResult.Errors {
    logger.WarnWithFields("graph operation failed",
        LogField{Key: "entity_type", Value: graphErr.EntityType},
        LogField{Key: "entity_id", Value: graphErr.EntityID},
        LogField{Key: "message", Value: graphErr.Message},
    )
    // Continue processing, graph is supplementary
}
```

### Embedding Errors

Embedding errors are non-fatal and retryable:

```go
embedResult := embedder.EmbedSymbols(ctx, symbols)
for _, embedErr := range embedResult.Errors {
    logger.WarnWithFields("embedding generation failed",
        LogField{Key: "entity_id", Value: embedErr.EntityID},
        LogField{Key: "message", Value: embedErr.Message},
    )
    // Continue processing, embeddings are optional
}
```

### Batch Operations

Collect errors during batch operations and continue processing:

```go
errorCollector := NewErrorCollector()

for _, file := range files {
    if err := processFile(file); err != nil {
        logger.WarnWithFields("file processing failed",
            LogField{Key: "file_path", Value: file.Path},
            LogField{Key: "error", Value: err.Error()},
        )
        errorCollector.Add(NewDatabaseError(
            "failed to process file",
            file.FileID,
            file.Path,
            err,
            isRetryable(err),
        ))
        continue // Process remaining files
    }
}

// Report summary
if errorCollector.HasErrors() {
    logger.WarnWithFields("batch processing completed with errors",
        LogField{Key: "total_errors", Value: errorCollector.Count()},
        LogField{Key: "error_summary", Value: errorCollector.Summary()},
    )
}
```

## Logging Best Practices

### 1. Use Appropriate Log Levels

- **Debug**: Detailed diagnostic information (verbose mode only)
- **Info**: Normal operational messages (progress, completion)
- **Warn**: Recoverable errors, degraded functionality
- **Error**: Unrecoverable errors, operation failures

### 2. Include Context

Always include relevant context in log messages:

```go
logger.InfoWithFields("processing file",
    LogField{Key: "file_path", Value: file.Path},
    LogField{Key: "file_id", Value: file.FileID},
    LogField{Key: "language", Value: file.Language},
    LogField{Key: "size_bytes", Value: file.Size},
)
```

### 3. Log Operation Boundaries

Log the start and completion of major operations:

```go
logger.InfoWithFields("starting indexing operation",
    LogField{Key: "repo_id", Value: repoID},
    LogField{Key: "files_count", Value: len(files)},
)

// ... perform operation ...

logger.InfoWithFields("indexing operation completed",
    LogField{Key: "repo_id", Value: repoID},
    LogField{Key: "duration", Value: duration},
    LogField{Key: "status", Value: status},
)
```

### 4. Log Errors with Full Context

Include entity IDs, file paths, and error details:

```go
logger.ErrorWithFields("failed to write symbol", err,
    LogField{Key: "symbol_id", Value: symbol.SymbolID},
    LogField{Key: "symbol_name", Value: symbol.Name},
    LogField{Key: "file_id", Value: symbol.FileID},
    LogField{Key: "file_path", Value: filePath},
)
```

### 5. Use Structured Fields Consistently

Use consistent field names across the codebase:

- `repo_id` - Repository identifier
- `file_id` - File identifier
- `file_path` - File path
- `symbol_id` - Symbol identifier
- `entity_id` - Generic entity identifier
- `entity_type` - Type of entity (file, symbol, node, edge)
- `error` - Error message
- `duration` - Operation duration
- `status` - Operation status
- `count` - Count of items

## Testing Error Handling

### Unit Tests

Test error creation and formatting:

```go
func TestIndexerError(t *testing.T) {
    cause := errors.New("underlying error")
    err := NewValidationError("invalid data", "entity-123", "file.go", cause)
    
    assert.Equal(t, ErrorTypeValidation, err.Type)
    assert.Equal(t, "invalid data", err.Message)
    assert.Equal(t, "entity-123", err.EntityID)
    assert.Equal(t, "file.go", err.FilePath)
    assert.Equal(t, cause, err.Cause)
    assert.False(t, err.Retryable)
}
```

### Integration Tests

Test error handling in real scenarios:

```go
func TestIndexerErrorHandling(t *testing.T) {
    // Create indexer with test database
    indexer := NewIndexer(testDB, config)
    
    // Test with invalid input
    result, err := indexer.Index(ctx, invalidInput)
    assert.Error(t, err)
    assert.Equal(t, "failed", result.Status)
    assert.Greater(t, len(result.Errors), 0)
    
    // Verify error types
    for _, err := range result.Errors {
        assert.NotEmpty(t, err.Type)
        assert.NotEmpty(t, err.Message)
    }
}
```

## Monitoring and Observability

### Metrics to Track

1. **Error Rates**
   - Total errors per operation
   - Errors by type
   - Retryable vs non-retryable errors

2. **Operation Success**
   - Success rate
   - Partial success rate
   - Complete failure rate

3. **Performance**
   - Operation duration
   - Items processed per second
   - Error recovery time

### Log Aggregation

Structure logs for easy aggregation and analysis:

```
INFO: indexing operation completed repo_id=abc-123 duration=5.2s status=success files_processed=150 symbols_created=1200
WARN: graph operation failed entity_type=node entity_id=symbol-456 message="node creation timeout"
ERROR: database write failed error="connection refused" repo_id=abc-123 entity_type=file entity_id=file-789
```

### Alerting

Set up alerts for:
- High error rates (> 5% of operations)
- Repeated non-retryable errors
- Database connection failures
- Embedding API failures
- Operation timeouts

## Example: Complete Error Handling Flow

```go
func (idx *Indexer) Index(ctx context.Context, input *schema.ParseOutput) (*IndexResult, error) {
    startTime := time.Now()
    errorCollector := NewErrorCollector()
    
    // Log operation start
    idx.logger.InfoWithFields("starting indexing operation",
        LogField{Key: "repo_id", Value: idx.config.RepoID},
        LogField{Key: "files_count", Value: len(input.Files)},
    )
    
    // Validate input
    validationResult := idx.validator.Validate(input)
    if validationResult.HasErrors() {
        idx.logger.ErrorWithFields("validation failed", nil,
            LogField{Key: "error_count", Value: validationResult.ErrorCount()},
        )
        for _, valErr := range validationResult.Errors {
            errorCollector.Add(NewValidationError(
                valErr.Message,
                valErr.EntityID,
                valErr.FilePath,
                nil,
            ))
        }
        return &IndexResult{Status: "failed", Errors: convertErrors(errorCollector.Errors())}, 
            fmt.Errorf("validation failed")
    }
    
    // Write data with error collection
    writeResult, err := idx.writeData(ctx, input.Files, input.Relationships)
    if err != nil {
        idx.logger.ErrorWithFields("write operation failed", err,
            LogField{Key: "repo_id", Value: idx.config.RepoID},
        )
        errorCollector.Add(NewDatabaseError("write failed", "", "", err, true))
    }
    
    // Collect write errors
    for _, writeErr := range writeResult.Errors {
        idx.logger.WarnWithFields("write error",
            LogField{Key: "entity_type", Value: writeErr.EntityType},
            LogField{Key: "entity_id", Value: writeErr.EntityID},
        )
        errorCollector.Add(NewDatabaseError(
            writeErr.Message,
            writeErr.EntityID,
            "",
            nil,
            writeErr.Retryable,
        ))
    }
    
    // Determine status
    status := "success"
    if errorCollector.HasErrors() {
        nonRetryable := errorCollector.FilterNonRetryable()
        if len(nonRetryable) > 0 {
            status = "partial_success"
        } else {
            status = "success_with_warnings"
        }
    }
    
    // Log completion
    idx.logger.InfoWithFields("indexing operation completed",
        LogField{Key: "repo_id", Value: idx.config.RepoID},
        LogField{Key: "status", Value: status},
        LogField{Key: "duration", Value: time.Since(startTime)},
        LogField{Key: "total_errors", Value: errorCollector.Count()},
    )
    
    return &IndexResult{
        Status:   status,
        Duration: time.Since(startTime),
        Errors:   convertErrors(errorCollector.Errors()),
    }, nil
}
```
