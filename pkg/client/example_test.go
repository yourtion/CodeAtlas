package client_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/client"
)

// Example demonstrates basic usage of the API client
func Example() {
	// Create a new API client with custom options
	apiClient := client.NewAPIClient(
		"http://localhost:8080",
		client.WithTimeout(10*time.Minute),
		client.WithToken("your-api-token"),
		client.WithMaxRetries(3),
	)

	ctx := context.Background()

	// Check server health
	if err := apiClient.Health(ctx); err != nil {
		log.Fatalf("API server is not healthy: %v", err)
	}

	// Index a repository
	indexReq := &client.IndexRequest{
		RepoName: "my-project",
		RepoURL:  "https://github.com/user/my-project",
		Branch:   "main",
		ParseOutput: schema.ParseOutput{
			Files: []schema.File{
				{
					FileID:   "file-1",
					Path:     "main.go",
					Language: "go",
					Size:     1024,
					Checksum: "abc123",
					Symbols: []schema.Symbol{
						{
							SymbolID:  "sym-1",
							FileID:    "file-1",
							Name:      "main",
							Kind:      schema.SymbolFunction,
							Signature: "func main()",
							Span: schema.Span{
								StartLine: 10,
								EndLine:   20,
								StartByte: 100,
								EndByte:   200,
							},
						},
					},
				},
			},
		},
		Options: client.IndexOptions{
			Incremental:    false,
			SkipVectors:    false,
			BatchSize:      100,
			WorkerCount:    4,
			EmbeddingModel: "text-embedding-3-small",
		},
	}

	indexResp, err := apiClient.Index(ctx, indexReq)
	if err != nil {
		log.Fatalf("Failed to index repository: %v", err)
	}

	fmt.Printf("Indexed repository: %s\n", indexResp.RepoID)
	fmt.Printf("Files processed: %d\n", indexResp.FilesProcessed)
	fmt.Printf("Symbols created: %d\n", indexResp.SymbolsCreated)

	// Perform semantic search
	embedding := []float32{0.1, 0.2, 0.3} // In practice, generate from query
	searchResp, err := apiClient.Search(ctx, "find main function", embedding, client.SearchFilters{
		Language: "go",
		Kind:     []string{"function"},
		Limit:    10,
	})
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Printf("Found %d results\n", searchResp.Total)
	for _, result := range searchResp.Results {
		fmt.Printf("- %s (%s) in %s\n", result.Name, result.Kind, result.FilePath)
	}

	// Get callers of a symbol
	callersResp, err := apiClient.GetCallers(ctx, "sym-1")
	if err != nil {
		log.Fatalf("Failed to get callers: %v", err)
	}

	fmt.Printf("Found %d callers\n", callersResp.Total)

	// Get callees of a symbol
	calleesResp, err := apiClient.GetCallees(ctx, "sym-1")
	if err != nil {
		log.Fatalf("Failed to get callees: %v", err)
	}

	fmt.Printf("Found %d callees\n", calleesResp.Total)

	// Get dependencies
	depsResp, err := apiClient.GetDependencies(ctx, "sym-1")
	if err != nil {
		log.Fatalf("Failed to get dependencies: %v", err)
	}

	fmt.Printf("Found %d dependencies\n", depsResp.Total)
	for _, dep := range depsResp.Dependencies {
		fmt.Printf("- %s (%s) via %s\n", dep.Name, dep.Kind, dep.EdgeType)
	}

	// Get symbols in a file
	symbolsResp, err := apiClient.GetFileSymbols(ctx, "file-1")
	if err != nil {
		log.Fatalf("Failed to get file symbols: %v", err)
	}

	fmt.Printf("Found %d symbols in file\n", symbolsResp.Total)
	for _, symbol := range symbolsResp.Symbols {
		fmt.Printf("- %s (%s) at line %d\n", symbol.Name, symbol.Kind, symbol.StartLine)
	}
}

// ExampleNewAPIClient demonstrates creating a client with various options
func ExampleNewAPIClient() {
	// Basic client
	client1 := client.NewAPIClient("http://localhost:8080")
	fmt.Printf("Client created with base URL: %s\n", "http://localhost:8080")

	// Client with authentication
	client2 := client.NewAPIClient(
		"http://localhost:8080",
		client.WithToken("my-secret-token"),
	)
	_ = client2

	// Client with custom timeout and retries
	client3 := client.NewAPIClient(
		"http://localhost:8080",
		client.WithTimeout(30*time.Second),
		client.WithMaxRetries(5),
	)
	_ = client3

	_ = client1
	// Output: Client created with base URL: http://localhost:8080
}

// ExampleAPIClient_Index demonstrates indexing a repository
func ExampleAPIClient_Index() {
	apiClient := client.NewAPIClient("http://localhost:8080")
	ctx := context.Background()

	req := &client.IndexRequest{
		RepoName: "example-repo",
		ParseOutput: schema.ParseOutput{
			Files: []schema.File{
				{FileID: "f1", Path: "main.go", Language: "go"},
			},
		},
		Options: client.IndexOptions{
			BatchSize:   100,
			WorkerCount: 4,
		},
	}

	resp, err := apiClient.Index(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Files: %d\n", resp.FilesProcessed)
}

// ExampleAPIClient_Search demonstrates semantic search
func ExampleAPIClient_Search() {
	apiClient := client.NewAPIClient("http://localhost:8080")
	ctx := context.Background()

	embedding := []float32{0.1, 0.2, 0.3}
	filters := client.SearchFilters{
		Language: "go",
		Kind:     []string{"function", "class"},
		Limit:    5,
	}

	resp, err := apiClient.Search(ctx, "authentication function", embedding, filters)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d results\n", resp.Total)
}

// ExampleAPIClient_GetCallers demonstrates finding callers of a symbol
func ExampleAPIClient_GetCallers() {
	apiClient := client.NewAPIClient("http://localhost:8080")
	ctx := context.Background()

	resp, err := apiClient.GetCallers(ctx, "symbol-id-123")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d callers\n", resp.Total)
	for _, symbol := range resp.Symbols {
		fmt.Printf("- %s in %s\n", symbol.Name, symbol.FilePath)
	}
}

// ExampleAPIClient_Health demonstrates health check
func ExampleAPIClient_Health() {
	apiClient := client.NewAPIClient("http://localhost:8080")
	ctx := context.Background()

	if err := apiClient.Health(ctx); err != nil {
		fmt.Println("Server is unhealthy")
		return
	}

	fmt.Println("Server is healthy")
}
