package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/utils"
	"github.com/yourtionguo/CodeAtlas/pkg/client"
)

// createSearchCommand creates the search CLI command
func createSearchCommand() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search code using semantic search",
		Description: `Perform semantic search across indexed code using natural language queries.
   This command generates embeddings for your query and searches for similar
   code symbols using vector similarity.

   The search supports filtering by repository, language, and symbol kind
   (function, class, interface, variable, etc.).

EXAMPLES:
   # Basic semantic search
   codeatlas search --query "function that handles user authentication"

   # Search within a specific repository
   codeatlas search --query "parse JSON data" --repo-id abc-123

   # Search for specific symbol types
   codeatlas search --query "database connection" --kind function,class

   # Search in specific language
   codeatlas search --query "HTTP request handler" --language go

   # Limit number of results
   codeatlas search --query "error handling" --limit 5

ENVIRONMENT VARIABLES:
   CODEATLAS_API_URL        Default API server URL
   CODEATLAS_API_TOKEN      API authentication token
   EMBEDDING_API_ENDPOINT   Embedding API endpoint (default: http://localhost:1234/v1/embeddings)
   EMBEDDING_MODEL          Embedding model name (default: text-embedding-qwen3-embedding-0.6b)`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "query",
				Aliases:  []string{"q"},
				Usage:    "Search query (natural language)",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "repo-id",
				Aliases: []string{"r"},
				Usage:   "Filter by repository ID",
			},
			&cli.StringFlag{
				Name:    "language",
				Aliases: []string{"l"},
				Usage:   "Filter by programming language (e.g., go, python, javascript)",
			},
			&cli.StringFlag{
				Name:    "kind",
				Aliases: []string{"k"},
				Usage:   "Filter by symbol kind (comma-separated: function,class,interface,variable)",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "Maximum number of results to return",
				Value:   10,
			},
			&cli.StringFlag{
				Name:  "api-url",
				Usage: "API server URL (can also use CODEATLAS_API_URL env var)",
			},
			&cli.StringFlag{
				Name:  "api-token",
				Usage: "API authentication token (can also use CODEATLAS_API_TOKEN env var)",
			},
			&cli.StringFlag{
				Name:  "embedding-endpoint",
				Usage: "Embedding API endpoint (can also use EMBEDDING_API_ENDPOINT env var)",
				Value: "http://localhost:1234/v1/embeddings",
			},
			&cli.StringFlag{
				Name:  "embedding-model",
				Usage: "Embedding model name (can also use EMBEDDING_MODEL env var)",
				Value: "text-embedding-qwen3-embedding-0.6b",
			},
			&cli.IntFlag{
				Name:  "embedding-dimensions",
				Usage: "Embedding dimensions (must match model)",
				Value: 768,
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Request timeout",
				Value: 30 * time.Second,
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose logging",
			},
		},
		Action: executeSearchCommand,
	}
}

// executeSearchCommand executes the search command
func executeSearchCommand(c *cli.Context) error {
	query := c.String("query")
	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	// Get API URL from flag or environment
	apiURL := c.String("api-url")
	if apiURL == "" {
		apiURL = os.Getenv("CODEATLAS_API_URL")
		if apiURL == "" {
			return fmt.Errorf("API URL must be specified via --api-url flag or CODEATLAS_API_URL environment variable")
		}
	}

	// Get API token from flag or environment
	apiToken := c.String("api-token")
	if apiToken == "" {
		apiToken = os.Getenv("CODEATLAS_API_TOKEN")
	}

	// Get embedding configuration from flags or environment
	embeddingEndpoint := c.String("embedding-endpoint")
	if envEndpoint := os.Getenv("EMBEDDING_API_ENDPOINT"); envEndpoint != "" {
		embeddingEndpoint = envEndpoint
	}

	embeddingModel := c.String("embedding-model")
	if envModel := os.Getenv("EMBEDDING_MODEL"); envModel != "" {
		embeddingModel = envModel
	}

	// Create logger
	verbose := c.Bool("verbose")
	logger := utils.NewLogger(verbose)

	logger.Info("Searching for: %s", query)

	// Create embedder for query vectorization
	embedderConfig := &indexer.EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          embeddingEndpoint,
		Model:                embeddingModel,
		Dimensions:           c.Int("embedding-dimensions"),
		BatchSize:            1,
		MaxRequestsPerSecond: 10,
		MaxRetries:           3,
		Timeout:              c.Duration("timeout"),
	}

	embedder := indexer.NewOpenAIEmbedder(embedderConfig, nil)

	// Generate embedding for query
	logger.Info("Generating embedding for query...")
	ctx := context.Background()
	embedding, err := embedder.GenerateEmbedding(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}
	logger.Debug("Generated embedding with %d dimensions", len(embedding))

	// Create API client
	clientOpts := []client.ClientOption{
		client.WithTimeout(c.Duration("timeout")),
		client.WithMaxRetries(3),
	}
	if apiToken != "" {
		clientOpts = append(clientOpts, client.WithToken(apiToken))
	}

	apiClient := client.NewAPIClient(apiURL, clientOpts...)

	// Check API health
	logger.Info("Checking API server health...")
	if err := apiClient.Health(ctx); err != nil {
		return fmt.Errorf("API server health check failed: %w", err)
	}
	logger.Debug("API server is healthy")

	// Build search filters
	filters := client.SearchFilters{
		RepoID:   c.String("repo-id"),
		Language: c.String("language"),
		Limit:    c.Int("limit"),
	}

	// Parse kind filter
	if kindStr := c.String("kind"); kindStr != "" {
		filters.Kind = strings.Split(kindStr, ",")
		// Trim whitespace from each kind
		for i := range filters.Kind {
			filters.Kind[i] = strings.TrimSpace(filters.Kind[i])
		}
	}

	// Perform search
	logger.Info("Searching...")
	startTime := time.Now()

	searchResp, err := apiClient.Search(ctx, query, embedding, filters)
	if err != nil {
		return fmt.Errorf("search request failed: %w", err)
	}

	duration := time.Since(startTime)

	// Display results
	displaySearchResults(searchResp, query, duration, logger)

	return nil
}

// displaySearchResults displays the search results
func displaySearchResults(resp *client.SearchResponse, query string, duration time.Duration, logger *utils.Logger) {
	fmt.Println("\n=== Search Results ===")
	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Found: %d results (in %v)\n", resp.Total, duration)
	fmt.Println()

	if len(resp.Results) == 0 {
		fmt.Println("No results found.")
		fmt.Println("\nTips:")
		fmt.Println("  - Try a different query")
		fmt.Println("  - Remove filters to broaden the search")
		fmt.Println("  - Ensure the repository has been indexed with embeddings")
		return
	}

	for i, result := range resp.Results {
		fmt.Printf("%d. %s (%s)\n", i+1, result.Name, result.Kind)
		fmt.Printf("   File: %s\n", result.FilePath)
		if result.Signature != "" {
			fmt.Printf("   Signature: %s\n", truncateString(result.Signature, 100))
		}
		fmt.Printf("   Similarity: %.2f%%\n", result.Similarity*100)
		if result.Docstring != "" {
			fmt.Printf("   Doc: %s\n", truncateString(result.Docstring, 150))
		}
		fmt.Printf("   ID: %s\n", result.SymbolID)
		fmt.Println()
	}

	fmt.Println("---")
	fmt.Printf("Showing %d of %d results\n", len(resp.Results), resp.Total)
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	// Replace newlines with spaces for display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")

	// Collapse multiple spaces
	s = strings.Join(strings.Fields(s), " ")

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
