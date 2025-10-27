# API Client

The `client` package provides an HTTP client for CLI tools to communicate with the CodeAtlas API server. It implements all API endpoints with built-in retry logic, connection pooling, and authentication support.

## Features

- **Comprehensive API Coverage**: All API endpoints (index, search, relationships, file symbols)
- **Retry Logic**: Exponential backoff for transient failures
- **Connection Pooling**: Efficient HTTP connection management
- **Authentication**: Bearer token support
- **Configurable Timeouts**: Customizable request timeouts
- **Health Checks**: Server health monitoring
- **Error Handling**: Detailed error responses with retry information

## Installation

```go
import "github.com/yourtionguo/CodeAtlas/pkg/client"
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/yourtionguo/CodeAtlas/pkg/client"
)

func main() {
    // Create a new API client
    apiClient := client.NewAPIClient("http://localhost:8080")
    
    ctx := context.Background()
    
    // Check server health
    if err := apiClient.Health(ctx); err != nil {
        log.Fatalf("Server is not healthy: %v", err)
    }
    
    log.Println("Server is healthy!")
}
```

### With Authentication

```go
apiClient := client.NewAPIClient(
    "http://localhost:8080",
    client.WithToken("your-api-token"),
)
```

### With Custom Configuration

```go
apiClient := client.NewAPIClient(
    "http://localhost:8080",
    client.WithTimeout(10*time.Minute),
    client.WithMaxRetries(5),
    client.WithToken("your-api-token"),
)
```

## API Methods

### Index

Index a repository with parsed code output:

```go
req := &client.IndexRequest{
    RepoName: "my-project",
    RepoURL:  "https://github.com/user/my-project",
    Branch:   "main",
    ParseOutput: parseOutput, // schema.ParseOutput
    Options: client.IndexOptions{
        Incremental:    false,
        SkipVectors:    false,
        BatchSize:      100,
        WorkerCount:    4,
        EmbeddingModel: "text-embedding-3-small",
    },
}

resp, err := apiClient.Index(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Indexed %d files, created %d symbols\n", 
    resp.FilesProcessed, resp.SymbolsCreated)
```

### Search

Perform semantic search across code:

```go
embedding := []float32{0.1, 0.2, 0.3} // Generate from query text
filters := client.SearchFilters{
    RepoID:   "repo-123",
    Language: "go",
    Kind:     []string{"function", "class"},
    Limit:    10,
}

resp, err := apiClient.Search(ctx, "authentication function", embedding, filters)
if err != nil {
    log.Fatal(err)
}

for _, result := range resp.Results {
    fmt.Printf("%s (%s) - similarity: %.2f\n", 
        result.Name, result.Kind, result.Similarity)
}
```

### Get Callers

Find all functions that call a specific symbol:

```go
resp, err := apiClient.GetCallers(ctx, "symbol-id-123")
if err != nil {
    log.Fatal(err)
}

for _, symbol := range resp.Symbols {
    fmt.Printf("%s in %s\n", symbol.Name, symbol.FilePath)
}
```

### Get Callees

Find all functions called by a specific symbol:

```go
resp, err := apiClient.GetCallees(ctx, "symbol-id-123")
if err != nil {
    log.Fatal(err)
}

for _, symbol := range resp.Symbols {
    fmt.Printf("%s in %s\n", symbol.Name, symbol.FilePath)
}
```

### Get Dependencies

Find all dependencies of a symbol (imports, extends, implements):

```go
resp, err := apiClient.GetDependencies(ctx, "symbol-id-123")
if err != nil {
    log.Fatal(err)
}

for _, dep := range resp.Dependencies {
    fmt.Printf("%s (%s) via %s\n", dep.Name, dep.Kind, dep.EdgeType)
}
```

### Get File Symbols

Retrieve all symbols in a file:

```go
resp, err := apiClient.GetFileSymbols(ctx, "file-id-123")
if err != nil {
    log.Fatal(err)
}

for _, symbol := range resp.Symbols {
    fmt.Printf("%s (%s) at line %d\n", 
        symbol.Name, symbol.Kind, symbol.StartLine)
}
```

### Health Check

Check if the API server is healthy:

```go
if err := apiClient.Health(ctx); err != nil {
    log.Printf("Server is unhealthy: %v", err)
} else {
    log.Println("Server is healthy")
}
```

## Configuration Options

### WithTimeout

Set the HTTP client timeout:

```go
client.WithTimeout(5 * time.Minute)
```

### WithToken

Set the authentication token:

```go
client.WithToken("your-api-token")
```

### WithMaxRetries

Set the maximum number of retry attempts:

```go
client.WithMaxRetries(5)
```

## Error Handling

The client returns detailed error information:

```go
resp, err := apiClient.Index(ctx, req)
if err != nil {
    if apiErr, ok := err.(*client.APIError); ok {
        fmt.Printf("API error (status %d): %s\n", 
            apiErr.StatusCode, apiErr.Message)
        if apiErr.Details != nil {
            fmt.Printf("Details: %v\n", apiErr.Details)
        }
    } else {
        fmt.Printf("Network error: %v\n", err)
    }
    return
}
```

## Retry Logic

The client automatically retries failed requests with exponential backoff:

- Retries on server errors (5xx status codes)
- Retries on rate limiting (429 status code)
- Retries on network errors
- Exponential backoff: 1s, 2s, 4s, 8s, ... (max 30s)
- Configurable max retries (default: 3)

Non-retryable errors (4xx except 429) fail immediately.

## Connection Pooling

The client uses connection pooling for efficient HTTP communication:

- Max idle connections: 100
- Max idle connections per host: 10
- Idle connection timeout: 90 seconds

## Thread Safety

The `APIClient` is safe for concurrent use by multiple goroutines.

## Testing

Run the tests:

```bash
go test ./pkg/client/...
```

Run with coverage:

```bash
go test ./pkg/client/... -cover
```

## Examples

See `example_test.go` for comprehensive usage examples.

## Requirements

This client implements the requirements from the Knowledge Graph Indexer specification:

- **4.1-4.8**: CLI index command and API communication
- **6.1-6.6**: Search and relationship queries
- **7.1**: Error handling with retry logic

## License

See the main project LICENSE file.
