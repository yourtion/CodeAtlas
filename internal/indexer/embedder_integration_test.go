// +build integration

package indexer

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestIntegration_OpenAIEmbedder tests the embedder against a real vLLM service
func TestIntegration_OpenAIEmbedder(t *testing.T) {
	// Create test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	vectorRepo := models.NewVectorRepository(db)

	// Configure embedder to use vLLM service
	config := &EmbedderConfig{
		Backend:              "openai",
		APIEndpoint:          "http://localhost:1234/v1/embeddings",
		Model:                "text-embedding-qwen3-embedding-0.6b",
		Dimensions:           1536, // gte-Qwen2-1.5B-instruct dimensions
		BatchSize:            10,
		MaxRequestsPerSecond: 5,
		MaxRetries:           3,
		BaseRetryDelay:       100 * time.Millisecond,
		MaxRetryDelay:        5 * time.Second,
		Timeout:              30 * time.Second,
	}

	embedder := NewOpenAIEmbedder(config, vectorRepo)

	t.Run("GenerateEmbedding", func(t *testing.T) {
		ctx := context.Background()
		content := "func TestFunction() int { return 42 }"

		embedding, err := embedder.GenerateEmbedding(ctx, content)
		if err != nil {
			t.Fatalf("Failed to generate embedding: %v", err)
		}

		if len(embedding) != config.Dimensions {
			t.Errorf("Expected embedding length %d, got %d", config.Dimensions, len(embedding))
		}

		// Verify embedding has non-zero values
		hasNonZero := false
		for _, val := range embedding {
			if val != 0 {
				hasNonZero = true
				break
			}
		}
		if !hasNonZero {
			t.Error("Embedding contains only zero values")
		}
	})

	t.Run("BatchEmbed", func(t *testing.T) {
		ctx := context.Background()
		texts := []string{
			"func Add(a, b int) int { return a + b }",
			"func Subtract(a, b int) int { return a - b }",
			"func Multiply(a, b int) int { return a * b }",
		}

		embeddings, err := embedder.BatchEmbed(ctx, texts)
		if err != nil {
			t.Fatalf("Failed to batch embed: %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
		}

		for i, embedding := range embeddings {
			if len(embedding) != config.Dimensions {
				t.Errorf("Embedding %d: expected length %d, got %d", i, config.Dimensions, len(embedding))
			}
		}
	})

	t.Run("EmbedSymbols", func(t *testing.T) {
		ctx := context.Background()

		symbols := []schema.Symbol{
			{
				SymbolID:  uuid.New().String(),
				FileID:    uuid.New().String(),
				Name:      "Add",
				Kind:      schema.SymbolFunction,
				Signature: "func Add(a, b int) int",
				Docstring: "Add returns the sum of two integers",
			},
			{
				SymbolID:  uuid.New().String(),
				FileID:    uuid.New().String(),
				Name:      "Subtract",
				Kind:      schema.SymbolFunction,
				Signature: "func Subtract(a, b int) int",
				Docstring: "Subtract returns the difference of two integers",
			},
		}

		result, err := embedder.EmbedSymbols(ctx, symbols)
		if err != nil {
			t.Fatalf("Failed to embed symbols: %v", err)
		}

		if result.VectorsCreated != len(symbols) {
			t.Errorf("Expected %d vectors created, got %d", len(symbols), result.VectorsCreated)
		}

		if len(result.Errors) > 0 {
			t.Errorf("Expected no errors, got %d errors:", len(result.Errors))
			for _, err := range result.Errors {
				t.Logf("  - %s: %s", err.EntityID, err.Message)
			}
		}

		// Verify vectors were stored in database
		for _, symbol := range symbols {
			vectors, err := vectorRepo.GetByEntityID(ctx, symbol.SymbolID, "symbol")
			if err != nil {
				t.Errorf("Failed to get vectors for symbol %s: %v", symbol.SymbolID, err)
				continue
			}
			if len(vectors) != 1 {
				t.Errorf("Expected 1 vector for symbol %s, got %d", symbol.SymbolID, len(vectors))
				continue
			}

			// Verify vector content
			vector := vectors[0]
			if len(vector.Embedding) != config.Dimensions {
				t.Errorf("Vector for symbol %s has wrong dimensions: expected %d, got %d",
					symbol.SymbolID, config.Dimensions, len(vector.Embedding))
			}
			if vector.Model != config.Model {
				t.Errorf("Vector for symbol %s has wrong model: expected %s, got %s",
					symbol.SymbolID, config.Model, vector.Model)
			}
		}
	})

	t.Run("SemanticSimilarity", func(t *testing.T) {
		ctx := context.Background()

		// Generate embeddings for similar and dissimilar texts
		similar1 := "func Add(a, b int) int { return a + b }"
		similar2 := "func Sum(x, y int) int { return x + y }"
		different := "func ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }"

		emb1, err := embedder.GenerateEmbedding(ctx, similar1)
		if err != nil {
			t.Fatalf("Failed to generate embedding 1: %v", err)
		}

		emb2, err := embedder.GenerateEmbedding(ctx, similar2)
		if err != nil {
			t.Fatalf("Failed to generate embedding 2: %v", err)
		}

		emb3, err := embedder.GenerateEmbedding(ctx, different)
		if err != nil {
			t.Fatalf("Failed to generate embedding 3: %v", err)
		}

		// Calculate cosine similarity
		sim12 := cosineSimilarity(emb1, emb2)
		sim13 := cosineSimilarity(emb1, emb3)

		t.Logf("Similarity between similar functions: %.4f", sim12)
		t.Logf("Similarity between different functions: %.4f", sim13)

		// Similar functions should have higher similarity
		if sim12 <= sim13 {
			t.Errorf("Expected similar functions to have higher similarity: %.4f <= %.4f", sim12, sim13)
		}

		// Similarity should be in reasonable range
		if sim12 < 0.5 {
			t.Errorf("Similarity between similar functions too low: %.4f", sim12)
		}
	})
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
