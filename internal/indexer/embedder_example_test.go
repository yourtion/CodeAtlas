package indexer_test

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// ExampleOpenAIEmbedder_GenerateEmbedding demonstrates generating a single embedding
func ExampleOpenAIEmbedder_GenerateEmbedding() {
	// Create database connection (in real usage)
	// db := models.NewDB(...)
	// vectorRepo := models.NewVectorRepository(db)

	// Create embedder configuration
	config := &indexer.EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          "http://localhost:1234/v1/embeddings",
		Model:                "text-embedding-qwen3-embedding-0.6b",
		Dimensions:           1024, // Actual dimension from local model
		BatchSize:            50,
		MaxRequestsPerSecond: 10,
	}

	// Create embedder (vectorRepo would be real in production)
	var vectorRepo *models.VectorRepository
	embedder := indexer.NewOpenAIEmbedder(config, vectorRepo)

	// Generate embedding for code content
	ctx := context.Background()
	content := "func calculateSum(a, b int) int { return a + b }"

	embedding, err := embedder.GenerateEmbedding(ctx, content)
	if err != nil {
		log.Fatalf("Failed to generate embedding: %v", err)
	}

	fmt.Printf("Generated embedding with %d dimensions\n", len(embedding))
	// Output: Generated embedding with 1024 dimensions
}

// ExampleOpenAIEmbedder_BatchEmbed demonstrates batch embedding generation
func ExampleOpenAIEmbedder_BatchEmbed() {
	// Create embedder configuration
	config := &indexer.EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          "http://localhost:1234/v1/embeddings",
		Model:                "text-embedding-qwen3-embedding-0.6b",
		Dimensions:           1024, // Actual dimension from local model
		BatchSize:            50,
		MaxRequestsPerSecond: 10,
	}

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
		log.Fatalf("Failed to batch embed: %v", err)
	}

	fmt.Printf("Generated %d embeddings\n", len(embeddings))
	for i, emb := range embeddings {
		fmt.Printf("Embedding %d: %d dimensions\n", i+1, len(emb))
	}
	// Output:
	// Generated 3 embeddings
	// Embedding 1: 1024 dimensions
	// Embedding 2: 1024 dimensions
	// Embedding 3: 1024 dimensions
}

// ExampleOpenAIEmbedder_EmbedSymbols demonstrates embedding code symbols
func ExampleOpenAIEmbedder_EmbedSymbols() {
	// Note: This example requires a running embedding service and database
	// For demonstration purposes, we show the configuration and usage pattern

	// Create embedder configuration
	config := &indexer.EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          "http://localhost:1234/v1/embeddings",
		Model:                "text-embedding-qwen3-embedding-0.6b",
		Dimensions:           1024,
		BatchSize:            50,
		MaxRequestsPerSecond: 10,
	}

	// In production, create embedder with real vectorRepo
	// db := models.NewDB(...)
	// vectorRepo := models.NewVectorRepository(db)
	// embedder := indexer.NewOpenAIEmbedder(config, vectorRepo)

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
	fmt.Printf("Configured embedder with model: %s\n", config.Model)
	fmt.Printf("Expected dimensions: %d\n", config.Dimensions)
	fmt.Printf("Symbols to embed: %d\n", len(symbols))

	// Output:
	// Configured embedder with model: text-embedding-qwen3-embedding-0.6b
	// Expected dimensions: 1024
	// Symbols to embed: 2
}

// ExampleEmbedderConfig demonstrates configuration options
func ExampleEmbedderConfig() {
	// Default configuration
	defaultConfig := indexer.DefaultEmbedderConfig()
	fmt.Printf("Default backend: %s\n", defaultConfig.Backend)
	fmt.Printf("Default endpoint: %s\n", defaultConfig.APIEndpoint)
	fmt.Printf("Default model: %s\n", defaultConfig.Model)
	fmt.Printf("Default dimensions: %d\n", defaultConfig.Dimensions)
	fmt.Printf("Default batch size: %d\n", defaultConfig.BatchSize)

	// Custom configuration for local deployment
	localConfig := &indexer.EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          "http://localhost:1234/v1/embeddings",
		Model:                "text-embedding-qwen3-embedding-0.6b",
		Dimensions:           768,
		BatchSize:            100,
		MaxRequestsPerSecond: 20,
	}
	fmt.Printf("\nLocal backend: %s\n", localConfig.Backend)
	fmt.Printf("Local endpoint: %s\n", localConfig.APIEndpoint)

	// Output:
	// Default backend: openai
	// Default endpoint: http://localhost:1234/v1/embeddings
	// Default model: text-embedding-qwen3-embedding-0.6b
	// Default dimensions: 768
	// Default batch size: 50
	//
	// Local backend: openai
	// Local endpoint: http://localhost:1234/v1/embeddings
}
