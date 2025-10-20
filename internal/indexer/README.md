# Indexer Package

The indexer package provides the orchestration layer for transforming parsed code structures into a queryable knowledge base. It coordinates validation, database writes, graph construction, and vector embedding generation.

## Overview

The indexer implements a multi-stage pipeline:

1. **Validation** - Validates input against schema constraints
2. **Repository Metadata** - Creates or updates repository records
3. **Database Writing** - Persists files, symbols, AST nodes, and edges
4. **Graph Building** - Constructs AGE graph nodes and relationships
5. **Embedding Generation** - Creates vector embeddings for semantic search

## Components

### Indexer

The main orchestrator that coordinates all indexing operations.

```go
type Indexer struct {
    validator    Validator
    writer       *Writer
    graphBuilder *GraphBuilder
    embedder     Embedder
    config       *IndexerConfig
    db           *models.DB
}
```

### Configuration

```go
type IndexerConfig struct {
    // Repository information
    RepoID   string
    RepoName string
    RepoURL  string
    Branch   string

    // Processing options
    BatchSize       int  // Items per batch (default: 100)
    WorkerCount     int  // Parallel workers (default: 4)
    SkipVectors     bool // Skip embedding generation
    Incremental     bool // Only process changed files
    UseTransactions bool // Use database transactions

    // Graph options
    GraphName string // AGE graph name (default: "code_graph")

    // Embedding options
    EmbeddingModel string // Model for embeddings
}
```

## Usage

### Basic Indexing

```go
// Create database connection
db, err := models.NewDB()
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Configure indexer
config := &indexer.IndexerConfig{
    RepoID:      "repo-123",
    RepoName:    "my-project",
    BatchSize:   100,
    WorkerCount: 4,
}

// Create indexer
idx := indexer.NewIndexer(db, config)

// Index parsed output
ctx := context.Background()
result, err := idx.Index(ctx, parseOutput)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Indexed %d files, %d symbols\n", 
    result.FilesProcessed, result.SymbolsCreated)
```

### Incremental Indexing

Only process files that have changed based on checksums:

```go
config := indexer.DefaultIndexerConfig()
config.Incremental = true // Enable incremental mode

idx := indexer.NewIndexer(db, config)
result, err := idx.Index(ctx, parseOutput)

// Only changed files will be processed
fmt.Printf("Processed %d changed files\n", result.FilesProcessed)
```

### Progress Tracking

Monitor indexing progress in real-time:

```go
progressChan := make(chan indexer.IndexProgress, 10)

go func() {
    for progress := range progressChan {
        fmt.Printf("[%s] %s (%.0f%%)\n", 
            progress.Stage, 
            progress.Message, 
            progress.Progress)
    }
}()

result, err := idx.IndexWithProgress(ctx, parseOutput, progressChan)
```

### Skip Vector Embeddings

For faster indexing without semantic search:

```go
config := indexer.DefaultIndexerConfig()
config.SkipVectors = true // Skip embedding generation

idx := indexer.NewIndexer(db, config)
result, err := idx.Index(ctx, parseOutput)
```

### Parallel Processing

Use multiple workers for large codebases:

```go
config := indexer.DefaultIndexerConfig()
config.WorkerCount = 8  // Use 8 parallel workers
config.BatchSize = 50   // Process 50 items per batch

idx := indexer.NewIndexer(db, config)
result, err := idx.Index(ctx, parseOutput)
```

## Index Result

The `IndexResult` structure contains detailed information about the indexing operation:

```go
type IndexResult struct {
    RepoID         string                 // Repository ID
    Status         string                 // success, partial_success, failed
    FilesProcessed int                    // Number of files indexed
    SymbolsCreated int                    // Number of symbols created
    NodesCreated   int                    // Number of AST nodes created
    EdgesCreated   int                    // Number of edges created
    VectorsCreated int                    // Number of embeddings generated
    Duration       time.Duration          // Total indexing time
    Errors         []*IndexerError        // Errors encountered
    Summary        map[string]interface{} // Additional statistics
}
```

### Status Values

- `success` - All operations completed successfully
- `success_with_warnings` - Completed with non-fatal errors (e.g., embedding failures)
- `partial_success` - Completed with some non-retryable errors
- `failed` - Critical failure (e.g., validation failed)

## Error Handling

The indexer collects errors throughout the pipeline and categorizes them:

```go
type IndexerError struct {
    Type      IndexerErrorType // validation, database, graph, embedding, etc.
    Message   string           // Error description
    EntityID  string           // Related entity ID
    FilePath  string           // Related file path
    Cause     error            // Underlying error
    Retryable bool             // Whether error is retryable
}
```

### Error Types

- `validation` - Schema validation errors (non-retryable)
- `database` - Database operation errors (may be retryable)
- `graph` - Graph construction errors (retryable)
- `embedding` - Embedding generation errors (retryable)
- `transaction` - Transaction errors (non-retryable)

### Error Collection

Errors are collected and summarized in the result:

```go
result, err := idx.Index(ctx, parseOutput)

// Check for errors
if len(result.Errors) > 0 {
    for _, err := range result.Errors {
        log.Printf("[%s] %s: %s\n", err.Type, err.EntityID, err.Message)
    }
}

// Check error summary
errorSummary := result.Summary["error_types"].(map[string]int)
fmt.Printf("Validation errors: %d\n", errorSummary["validation"])
fmt.Printf("Database errors: %d\n", errorSummary["database"])
```

## Performance Considerations

### Batch Size

- Larger batches reduce database round trips but use more memory
- Recommended: 50-200 depending on entity size
- Default: 100

### Worker Count

- More workers increase parallelism for embedding generation
- Limited by CPU cores and API rate limits
- Recommended: 4-8 for most workloads
- Default: 4

### Transactions

- Transactions ensure atomicity but may be slower for large datasets
- Disable for very large imports (>10,000 files)
- Default: enabled

### Incremental Mode

- Significantly faster for re-indexing with few changes
- Uses file checksums to detect changes
- Recommended for CI/CD pipelines
- Default: disabled

## Integration with Other Components

### Validator

Validates input structure and referential integrity:

```go
validator := indexer.NewSchemaValidator()
result := validator.Validate(parseOutput)

if result.HasErrors() {
    for _, err := range result.Errors {
        log.Printf("Validation error: %s\n", err.Message)
    }
}
```

### Writer

Handles database persistence with retry logic:

```go
writer := indexer.NewWriter(db, writerConfig)

// Write files
filesResult, err := writer.WriteFiles(ctx, repoID, files)

// Write symbols
symbolsResult, err := writer.WriteSymbols(ctx, symbols)
```

### Graph Builder

Constructs AGE graph nodes and edges:

```go
graphBuilder := indexer.NewGraphBuilder(db, graphConfig)

// Initialize graph
err := graphBuilder.InitGraph(ctx)

// Create nodes
nodesResult, err := graphBuilder.CreateNodes(ctx, symbols)

// Create edges
edgesResult, err := graphBuilder.CreateEdges(ctx, edges)
```

### Embedder

Generates vector embeddings for semantic search:

```go
embedder := indexer.NewOpenAIEmbedder(embedderConfig, vectorRepo)

// Generate single embedding
embedding, err := embedder.GenerateEmbedding(ctx, content)

// Batch embed
embeddings, err := embedder.BatchEmbed(ctx, texts)

// Embed symbols
embedResult, err := embedder.EmbedSymbols(ctx, symbols)
```

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
go test ./internal/indexer -v

# Run specific test
go test ./internal/indexer -v -run TestIndexValidInput

# Run with coverage
go test ./internal/indexer -cover
```

Note: Database tests require a test database and are skipped by default.

## Examples

See `indexer_example_test.go` for complete examples:

- Basic indexing
- Incremental indexing
- Progress tracking
- Parallel processing
- Skipping embeddings

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Indexer                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │Validator │→ │  Writer  │→ │  Graph   │→ │ Embedder │   │
│  └──────────┘  └──────────┘  │ Builder  │  └──────────┘   │
│                               └──────────┘                   │
└─────────────────────────────────────────────────────────────┘
                          ↓
        ┌─────────────────┴─────────────────┐
        ↓                 ↓                  ↓
   ┌──────────┐    ┌──────────┐      ┌──────────┐
   │PostgreSQL│    │   AGE    │      │ pgvector │
   │(Relational)│  │ (Graph)  │      │(Vectors) │
   └──────────┘    └──────────┘      └──────────┘
```

## Requirements

- PostgreSQL 17+ with pgvector and AGE extensions
- Go 1.21+
- Sufficient memory for batch processing (depends on batch size)

## Configuration Best Practices

### Development

```go
config := &indexer.IndexerConfig{
    BatchSize:       50,
    WorkerCount:     2,
    SkipVectors:     true,  // Faster for development
    Incremental:     false,
    UseTransactions: true,
}
```

### Production

```go
config := &indexer.IndexerConfig{
    BatchSize:       100,
    WorkerCount:     8,
    SkipVectors:     false,
    Incremental:     true,  // Faster re-indexing
    UseTransactions: true,
}
```

### Large Codebases (>10,000 files)

```go
config := &indexer.IndexerConfig{
    BatchSize:       200,
    WorkerCount:     16,
    SkipVectors:     false,
    Incremental:     true,
    UseTransactions: false, // Better performance for bulk
}
```

## Troubleshooting

### Slow Indexing

- Increase `BatchSize` to reduce database round trips
- Increase `WorkerCount` for more parallelism
- Enable `Incremental` mode for re-indexing
- Consider disabling transactions for very large imports

### Memory Issues

- Reduce `BatchSize` to use less memory
- Reduce `WorkerCount` to limit concurrent operations
- Process repository in smaller chunks

### Validation Errors

- Check input structure matches schema
- Verify all required fields are present
- Ensure referential integrity (symbol IDs in edges exist)

### Database Errors

- Check database connection and credentials
- Verify PostgreSQL extensions are installed
- Check database disk space
- Review connection pool settings

### Graph Errors

- Verify AGE extension is installed and loaded
- Check graph schema initialization
- Review Cypher query syntax

### Embedding Errors

- Check embedding API endpoint and credentials
- Verify rate limits are not exceeded
- Check embedding dimensions match configuration
- Review API error messages

## See Also

- [Validator Documentation](./validator.go)
- [Writer Documentation](./writer.go)
- [Graph Builder Documentation](./graph_builder.go)
- [Embedder Documentation](./embedder.go)
- [Error Handling](./errors.go)
