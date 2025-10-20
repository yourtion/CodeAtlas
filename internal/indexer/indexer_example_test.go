package indexer_test

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// ExampleIndexer_Index demonstrates basic indexing usage
func ExampleIndexer_Index() {
	// Create database connection
	db, err := models.NewDB()
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}
	defer db.Close()

	// Configure indexer
	config := &indexer.IndexerConfig{
		RepoID:      uuid.New().String(),
		RepoName:    "example-repo",
		RepoURL:     "https://github.com/example/repo",
		Branch:      "main",
		BatchSize:   100,
		WorkerCount: 4,
		SkipVectors: false,
		Incremental: false,
		GraphName:   "code_graph",
	}

	// Create indexer
	idx := indexer.NewIndexer(db, config)

	// Create sample parse output
	fileID := uuid.New().String()
	symbolID := uuid.New().String()

	input := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   fileID,
				Path:     "main.go",
				Language: "go",
				Size:     1024,
				Checksum: "abc123",
				Symbols: []schema.Symbol{
					{
						SymbolID:  symbolID,
						FileID:    fileID,
						Name:      "main",
						Kind:      schema.SymbolFunction,
						Signature: "func main()",
						Span: schema.Span{
							StartLine: 5,
							EndLine:   10,
							StartByte: 50,
							EndByte:   150,
						},
						Docstring: "Main entry point",
					},
				},
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	// Index the data
	ctx := context.Background()
	result, err := idx.Index(ctx, input)
	if err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		return
	}

	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Files processed: %d\n", result.FilesProcessed)
	fmt.Printf("Symbols created: %d\n", result.SymbolsCreated)
	fmt.Printf("Duration: %s\n", result.Duration)
}

// ExampleIndexer_IndexWithProgress demonstrates indexing with progress tracking
func ExampleIndexer_IndexWithProgress() {
	db, err := models.NewDB()
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}
	defer db.Close()

	config := indexer.DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "example-repo"

	idx := indexer.NewIndexer(db, config)

	// Create progress channel
	progressChan := make(chan indexer.IndexProgress, 10)

	// Monitor progress in goroutine
	go func() {
		for progress := range progressChan {
			fmt.Printf("[%s] %s (%.0f%%)\n",
				progress.Stage,
				progress.Message,
				progress.Progress)
		}
	}()

	// Create sample input
	input := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   uuid.New().String(),
				Path:     "example.go",
				Language: "go",
				Size:     512,
				Checksum: "xyz789",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	ctx := context.Background()
	result, err := idx.IndexWithProgress(ctx, input, progressChan)
	if err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		return
	}

	fmt.Printf("Final status: %s\n", result.Status)
}

// ExampleIndexer_Index_incremental demonstrates incremental indexing
func ExampleIndexer_Index_incremental() {
	db, err := models.NewDB()
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}
	defer db.Close()

	config := indexer.DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "example-repo"
	config.Incremental = true // Enable incremental mode

	idx := indexer.NewIndexer(db, config)

	// First index
	input := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   uuid.New().String(),
				Path:     "file1.go",
				Language: "go",
				Size:     1024,
				Checksum: "checksum1",
			},
			{
				FileID:   uuid.New().String(),
				Path:     "file2.go",
				Language: "go",
				Size:     2048,
				Checksum: "checksum2",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   2,
			SuccessCount: 2,
			FailureCount: 0,
		},
	}

	ctx := context.Background()

	// First indexing - processes all files
	result1, _ := idx.Index(ctx, input)
	fmt.Printf("First index: %d files processed\n", result1.FilesProcessed)

	// Second indexing with same checksums - processes 0 files
	result2, _ := idx.Index(ctx, input)
	fmt.Printf("Second index (no changes): %d files processed\n", result2.FilesProcessed)

	// Modify one file's checksum
	input.Files[0].Checksum = "new-checksum"

	// Third indexing - processes only changed file
	result3, _ := idx.Index(ctx, input)
	fmt.Printf("Third index (1 file changed): %d files processed\n", result3.FilesProcessed)
}

// ExampleIndexer_Index_skipVectors demonstrates indexing without embeddings
func ExampleIndexer_Index_skipVectors() {
	db, err := models.NewDB()
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}
	defer db.Close()

	config := indexer.DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "example-repo"
	config.SkipVectors = true // Skip embedding generation

	idx := indexer.NewIndexer(db, config)

	input := &schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   uuid.New().String(),
				Path:     "example.go",
				Language: "go",
				Size:     1024,
				Checksum: "abc123",
			},
		},
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
	}

	ctx := context.Background()
	result, err := idx.Index(ctx, input)
	if err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		return
	}

	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Vectors created: %d (should be 0)\n", result.VectorsCreated)
}

// ExampleIndexer_Index_parallel demonstrates parallel processing
func ExampleIndexer_Index_parallel() {
	db, err := models.NewDB()
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}
	defer db.Close()

	config := indexer.DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "example-repo"
	config.WorkerCount = 8 // Use 8 parallel workers
	config.BatchSize = 50  // Process 50 items per batch

	idx := indexer.NewIndexer(db, config)

	// Create input with many files
	files := make([]schema.File, 100)
	for i := 0; i < 100; i++ {
		files[i] = schema.File{
			FileID:   uuid.New().String(),
			Path:     fmt.Sprintf("file%d.go", i),
			Language: "go",
			Size:     1024,
			Checksum: fmt.Sprintf("checksum%d", i),
		}
	}

	input := &schema.ParseOutput{
		Files: files,
		Metadata: schema.ParseMetadata{
			Version:      "1.0",
			Timestamp:    time.Now(),
			TotalFiles:   100,
			SuccessCount: 100,
			FailureCount: 0,
		},
	}

	ctx := context.Background()
	startTime := time.Now()
	result, err := idx.Index(ctx, input)
	if err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		return
	}

	fmt.Printf("Processed %d files in %s\n", result.FilesProcessed, time.Since(startTime))
	fmt.Printf("Throughput: %.2f files/sec\n",
		float64(result.FilesProcessed)/time.Since(startTime).Seconds())
}
