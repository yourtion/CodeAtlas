package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

func TestStreamProcessor_StreamFiles(t *testing.T) {
	config := DefaultStreamConfig()
	config.MaxMemoryMB = 10
	config.MaxGoroutines = 4

	sp := NewStreamProcessor(config)

	// Create test files
	files := make([]schema.File, 100)
	for i := range files {
		files[i] = schema.File{
			FileID:   string(rune(i)),
			Path:     "/test/file.go",
			Language: "go",
			Size:     1024,
			Checksum: "abc123",
		}
	}

	// Process files
	processedCount := 0
	err := sp.StreamFiles(context.Background(), files, func(ctx context.Context, file schema.File) error {
		processedCount++
		time.Sleep(1 * time.Millisecond) // Simulate work
		return nil
	})

	if err != nil {
		t.Fatalf("StreamFiles failed: %v", err)
	}

	if processedCount != len(files) {
		t.Errorf("Expected %d files processed, got %d", len(files), processedCount)
	}
}

func TestStreamProcessor_StreamASTNodes(t *testing.T) {
	config := DefaultStreamConfig()
	sp := NewStreamProcessor(config)

	// Create test nodes
	nodes := make([]schema.ASTNode, 1000)
	for i := range nodes {
		nodes[i] = schema.ASTNode{
			NodeID: string(rune(i)),
			FileID: "file1",
			Type:   "function",
			Span: schema.Span{
				StartLine: i,
				EndLine:   i + 10,
			},
		}
	}

	// Process nodes
	processedCount := 0
	err := sp.StreamASTNodes(context.Background(), nodes, 100, func(ctx context.Context, batch []schema.ASTNode) error {
		processedCount += len(batch)
		return nil
	})

	if err != nil {
		t.Fatalf("StreamASTNodes failed: %v", err)
	}

	if processedCount != len(nodes) {
		t.Errorf("Expected %d nodes processed, got %d", len(nodes), processedCount)
	}
}

func TestStreamProcessor_MemoryBackpressure(t *testing.T) {
	config := DefaultStreamConfig()
	config.MaxMemoryMB = 1 // Very low limit to trigger backpressure
	config.MaxGoroutines = 2

	sp := NewStreamProcessor(config)

	// Create large files
	files := make([]schema.File, 10)
	for i := range files {
		files[i] = schema.File{
			FileID:   string(rune(i)),
			Path:     "/test/file.go",
			Language: "go",
			Size:     1024 * 1024, // 1 MB
			Checksum: "abc123",
		}
	}

	// Process should complete despite memory pressure
	err := sp.StreamFiles(context.Background(), files, func(ctx context.Context, file schema.File) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Fatalf("StreamFiles failed under memory pressure: %v", err)
	}

	stats := sp.GetMemoryStats()
	if stats.CurrentMemoryMB > config.MaxMemoryMB*2 {
		t.Errorf("Memory usage exceeded limits: %d MB (max: %d MB)", stats.CurrentMemoryMB, config.MaxMemoryMB)
	}
}

func TestStreamProcessor_ContextCancellation(t *testing.T) {
	config := DefaultStreamConfig()
	sp := NewStreamProcessor(config)

	ctx, cancel := context.WithCancel(context.Background())

	files := make([]schema.File, 100)
	for i := range files {
		files[i] = schema.File{
			FileID: string(rune(i)),
		}
	}

	// Cancel after processing a few files
	processedCount := 0
	err := sp.StreamFiles(ctx, files, func(ctx context.Context, file schema.File) error {
		processedCount++
		if processedCount == 10 {
			cancel()
		}
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestMemoryStats(t *testing.T) {
	stats := MemoryStats{
		CurrentMemoryMB:  50,
		MaxMemoryMB:      100,
		ActiveGoroutines: 5,
		MaxGoroutines:    10,
	}

	if pressure := stats.MemoryPressure(); pressure != 50.0 {
		t.Errorf("Expected memory pressure 50%%, got %.1f%%", pressure)
	}

	if pressure := stats.GoroutinePressure(); pressure != 50.0 {
		t.Errorf("Expected goroutine pressure 50%%, got %.1f%%", pressure)
	}
}
