# Vector Embedder

The vector embedder component generates semantic embeddings for code symbols and stores them in the database for similarity search and retrieval.

## Features

- **OpenAI-Compatible API Support**: Works with any OpenAI-compatible embedding API
- **Batch Processing**: Efficiently processes multiple texts in batches to minimize API calls
- **Rate Limiting**: Built-in token bucket rate limiter to respect API limits
- **Retry Logic**: Automatic retry with exponential backoff for transient failures
- **Dimension Validation**: Validates embedding dimensions before storage
- **Error Handling**: Graceful error handling that doesn't block other operations

## Configuration

### EmbedderConfig

```go
type EmbedderConfig struct {
    Backend              string        // "openai" or "local"
    APIEndpoint          string        // API endpoint URL
    APIKey               string        // API key (optional for local)
    Model                string        // Model name
    Dimensions           int           // Expected embedding dimensions
    BatchSize            int           // Batch size for API calls
    MaxRequestsPerSecond int           // Rate limit
    MaxRetries           int           // Max retry attempts
    BaseRetryDelay       time.Duration // Initial retry delay
    MaxRetryDelay        time.Duration // Maximum retry delay
    Timeout              time.Duration // HTTP client timeout
}
```

### Default Configuration

```go
config := indexer.DefaultEmbedderConfig()
// Backend: "openai"
// APIEndpoint: "http://localhost:1234/v1/embeddings"
// Model: "text-embedding-qwen3-embedding-0.6b"
// Dimensions: 768 (adjust based on your model)
// BatchSize: 50
// MaxRequestsPerSecond: 10
```

**Note**: The actual embedding dimensions depend on the model you use. For example:
- `text-embedding-qwen3-embedding-0.6b`: 1024 dimensions
- `nomic-embed-text`: 768 dimensions
- `text-embedding-3-small` (OpenAI): 1536 dimensions

## Usage

### Basic Embedding Generation

```go
// Create embedder
config := indexer.DefaultEmbedderConfig()
embedder := indexer.NewOpenAIEmbedder(config, vectorRepo)

// Generate single embedding
ctx := context.Background()
content := "func calculateSum(a, b int) int { return a + b }"
embedding, err := embedder.GenerateEmbedding(ctx, content)
if err != nil {
    log.Fatal(err)
}
```

### Batch Embedding

```go
// Batch embed multiple texts
texts := []string{
    "func add(a, b int) int { return a + b }",
    "func subtract(a, b int) int { return a - b }",
    "func multiply(a, b int) int { return a * b }",
}

embeddings, err := embedder.BatchEmbed(ctx, texts)
if err != nil {
    log.Fatal(err)
}
```

### Embedding Symbols

```go
// Create symbols
symbols := []schema.Symbol{
    {
        SymbolID:  uuid.New().String(),
        FileID:    uuid.New().String(),
        Name:      "calculateSum",
        Kind:      schema.SymbolFunction,
        Signature: "func calculateSum(a, b int) int",
        Docstring: "Adds two integers and returns the result",
    },
}

// Embed symbols
result, err := embedder.EmbedSymbols(ctx, symbols)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created %d vectors\n", result.VectorsCreated)
fmt.Printf("Errors: %d\n", len(result.Errors))
```

## Symbol Content Construction

The embedder constructs content for embedding from symbols by combining:

1. **Signature**: Function/class signature
2. **Docstring**: Documentation string
3. **Semantic Summary**: AI-generated summary (if available)

Example:
```
func calculateSum(a, b int) int
Adds two integers and returns the result
Returns the sum of two integer values
```

## API Compatibility

The embedder is compatible with any OpenAI-compatible embedding API, including:

- **OpenAI API**: `https://api.openai.com/v1/embeddings`
- **Local LM Studio**: `http://localhost:1234/v1/embeddings`
- **vLLM**: Custom endpoint with OpenAI-compatible format
- **Text Generation WebUI**: With OpenAI extension

### Request Format

```json
{
  "input": ["text1", "text2"],
  "model": "text-embedding-qwen3-embedding-0.6b"
}
```

### Response Format

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.1, 0.2, ...],
      "index": 0
    }
  ],
  "model": "text-embedding-qwen3-embedding-0.6b",
  "usage": {
    "prompt_tokens": 10,
    "total_tokens": 10
  }
}
```

## Rate Limiting

The embedder implements token bucket rate limiting:

```go
config := &indexer.EmbedderConfig{
    MaxRequestsPerSecond: 10, // 10 requests per second
}
```

The rate limiter:
- Refills tokens every second
- Blocks when no tokens available
- Respects context cancellation

## Retry Logic

Automatic retry with exponential backoff for:

- Network errors (connection refused, timeout)
- Rate limit errors (429)
- Server errors (500, 502, 503, 504)

Non-retryable errors:
- Client errors (400, 401, 403, 404)
- Invalid request format
- Authentication failures

```go
config := &indexer.EmbedderConfig{
    MaxRetries:     3,
    BaseRetryDelay: 100 * time.Millisecond,
    MaxRetryDelay:  5 * time.Second,
}
```

## Error Handling

The embedder handles errors gracefully:

```go
result, err := embedder.EmbedSymbols(ctx, symbols)
if err != nil {
    log.Fatal(err)
}

// Check for partial failures
if len(result.Errors) > 0 {
    for _, embedErr := range result.Errors {
        log.Printf("Failed to embed %s: %s", embedErr.EntityID, embedErr.Message)
    }
}

// Continue with successfully created vectors
fmt.Printf("Successfully created %d vectors\n", result.VectorsCreated)
```

## Dimension Validation

The embedder validates embedding dimensions before storage:

```go
config := &indexer.EmbedderConfig{
    Dimensions: 768, // Expected dimensions
}

// If API returns different dimensions, error is logged
// and vector is not stored
```

## Performance Considerations

### Batch Size

Larger batch sizes reduce API calls but increase latency:

```go
// Small batch: More API calls, lower latency per batch
config.BatchSize = 10

// Large batch: Fewer API calls, higher latency per batch
config.BatchSize = 100
```

### Rate Limiting

Balance throughput with API limits:

```go
// Conservative: 5 requests/second
config.MaxRequestsPerSecond = 5

// Aggressive: 20 requests/second
config.MaxRequestsPerSecond = 20
```

### Timeout

Set appropriate timeout for API calls:

```go
// Short timeout for fast local models
config.Timeout = 5 * time.Second

// Longer timeout for remote APIs
config.Timeout = 30 * time.Second
```

## Testing

The embedder includes comprehensive tests:

```bash
# Run all embedder tests
go test -v ./internal/indexer -run TestOpenAIEmbedder

# Run specific test
go test -v ./internal/indexer -run TestOpenAIEmbedder_BatchEmbed

# Run with coverage
go test -cover ./internal/indexer -run TestOpenAIEmbedder
```

## Local Development

### Using LM Studio

1. Download and install LM Studio
2. Load an embedding model (e.g., `nomic-embed-text`)
3. Start the local server
4. Configure embedder:

```go
config := &indexer.EmbedderConfig{
    Backend:     "openai",
    APIEndpoint: "http://localhost:1234/v1/embeddings",
    Model:       "nomic-embed-text",
    Dimensions:  768,
}
```

### Using vLLM with Docker

```bash
# Start vLLM server
docker run -d \
  --name vllm-embeddings \
  -p 1234:8000 \
  vllm/vllm-openai:latest \
  --model Qwen/Qwen3-embedding-0.6b \
  --port 8000

# Configure embedder
config := &indexer.EmbedderConfig{
    Backend:     "openai",
    APIEndpoint: "http://localhost:1234/v1/embeddings",
    Model:       "Qwen/Qwen3-embedding-0.6b",
    Dimensions:  768,
}
```

## Integration with Indexer

The embedder integrates with the main indexer pipeline:

```go
// Create indexer with embedder
indexer := &Indexer{
    writer:       writer,
    graphBuilder: graphBuilder,
    embedder:     embedder,
}

// Embeddings are generated after graph construction
result, err := indexer.Index(ctx, parseOutput)
```

## Future Enhancements

- [ ] Support for local embedding models (sentence-transformers)
- [ ] Embedding cache to avoid recomputation
- [ ] Chunking for large code blocks
- [ ] Multi-model support for different entity types
- [ ] Embedding quality metrics
- [ ] Async embedding generation with queue
