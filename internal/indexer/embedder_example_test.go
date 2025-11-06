package indexer_test

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// getEmbedderConfigFromEnv returns embedder config from environment variables
// Falls back to localhost defaults if not set
func getEmbedderConfigFromEnv() *indexer.EmbedderConfig {
	backend := os.Getenv("EMBEDDING_BACKEND")
	if backend == "" {
		backend = "openai"
	}

	endpoint := os.Getenv("EMBEDDING_API_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:1234/v1/embeddings"
	}

	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-qwen3-embedding-0.6b"
	}

	dimensions := 1024
	if dimStr := os.Getenv("EMBEDDING_DIMENSIONS"); dimStr != "" {
		fmt.Sscanf(dimStr, "%d", &dimensions)
	}

	return &indexer.EmbedderConfig{
		Backend:              backend,
		APIEndpoint:          endpoint,
		APIKey:               os.Getenv("EMBEDDING_API_KEY"),
		Model:                model,
		Dimensions:           dimensions,
		BatchSize:            50,
		MaxRequestsPerSecond: 10,
	}
}

// ExampleOpenAIEmbedder_GenerateEmbedding demonstrates generating a single embedding
// This example requires an embedding service to be available
func ExampleOpenAIEmbedder_GenerateEmbedding() {
	// Skip if INDEXER_SKIP_VECTORS is set
	if os.Getenv("INDEXER_SKIP_VECTORS") == "true" {
		fmt.Println("Skipping: INDEXER_SKIP_VECTORS is enabled")
		return
	}

	// Get configuration from environment
	config := getEmbedderConfigFromEnv()

	// Create embedder (vectorRepo would be real in production)
	var vectorRepo *models.VectorRepository
	embedder := indexer.NewOpenAIEmbedder(config, vectorRepo)

	// Generate embedding for code content
	ctx := context.Background()
	content := "func calculateSum(a, b int) int { return a + b }"

	embedding, err := embedder.GenerateEmbedding(ctx, content)
	if err != nil {
		// If embedding service is not available, skip gracefully
		fmt.Printf("Skipping: embedding service not available (%v)\n", err)
		return
	}

	fmt.Printf("✓ Generated embedding with %d dimensions\n", len(embedding))
}

// ExampleOpenAIEmbedder_BatchEmbed demonstrates batch embedding generation
// This example requires an embedding service to be available
func ExampleOpenAIEmbedder_BatchEmbed() {
	// Skip if INDEXER_SKIP_VECTORS is set
	if os.Getenv("INDEXER_SKIP_VECTORS") == "true" {
		fmt.Println("Skipping: INDEXER_SKIP_VECTORS is enabled")
		return
	}

	// Get configuration from environment
	config := getEmbedderConfigFromEnv()

	// Create embedder
	var vectorRepo *models.VectorRepository
	embedder := indexer.NewOpenAIEmbedder(config, vectorRepo)

	// Batch embed multiple code snippets
	ctx := context.Background()
	texts := []string{
		"func add(a, b int) int { return a + b }",
		"func subtract(a, b int) int { return a - b }",
		"func multiply(a, b int) int { return a * b }",
	}

	embeddings, err := embedder.BatchEmbed(ctx, texts)
	if err != nil {
		// If embedding service is not available, skip gracefully
		fmt.Printf("Skipping: embedding service not available (%v)\n", err)
		return
	}

	fmt.Printf("✓ Generated %d embeddings\n", len(embeddings))
	for i, emb := range embeddings {
		fmt.Printf("  Embedding %d: %d dimensions\n", i+1, len(emb))
	}
}

// ExampleOpenAIEmbedder_EmbedSymbols demonstrates embedding code symbols
// This example shows how to configure the embedder for symbol embedding
func ExampleOpenAIEmbedder_EmbedSymbols() {
	// Get configuration from environment
	config := getEmbedderConfigFromEnv()

	// Create symbols to embed
	symbols := []schema.Symbol{
		{
			SymbolID:  uuid.New().String(),
			FileID:    uuid.New().String(),
			Name:      "calculateSum",
			Kind:      schema.SymbolFunction,
			Signature: "func calculateSum(a, b int) int",
			Docstring: "calculateSum adds two integers and returns the result",
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    uuid.New().String(),
			Name:      "User",
			Kind:      schema.SymbolClass,
			Signature: "type User struct",
			Docstring: "User represents a user in the system",
		},
	}

	// Show configuration
	fmt.Printf("✓ Configured embedder with model: %s\n", config.Model)
	fmt.Printf("✓ Expected dimensions: %d\n", config.Dimensions)
	fmt.Printf("✓ Symbols to embed: %d\n", len(symbols))
}

// ExampleEmbedderConfig demonstrates configuration options
func ExampleEmbedderConfig() {
	// Configuration from environment (used in CI/production)
	envConfig := getEmbedderConfigFromEnv()
	fmt.Printf("Environment configuration:\n")
	fmt.Printf("  Backend: %s\n", envConfig.Backend)
	fmt.Printf("  Endpoint: %s\n", envConfig.APIEndpoint)
	fmt.Printf("  Model: %s\n", envConfig.Model)
	fmt.Printf("  Dimensions: %d\n", envConfig.Dimensions)
	fmt.Printf("  Batch size: %d\n", envConfig.BatchSize)

	// Custom configuration for local deployment
	localConfig := &indexer.EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          "http://localhost:1234/v1/embeddings",
		Model:                "text-embedding-qwen3-embedding-0.6b",
		Dimensions:           768,
		BatchSize:            100,
		MaxRequestsPerSecond: 20,
	}
	fmt.Printf("\nLocal configuration:\n")
	fmt.Printf("  Backend: %s\n", localConfig.Backend)
	fmt.Printf("  Endpoint: %s\n", localConfig.APIEndpoint)
	fmt.Printf("  Dimensions: %d\n", localConfig.Dimensions)
}
