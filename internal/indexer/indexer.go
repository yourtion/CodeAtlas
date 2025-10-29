package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// Indexer orchestrates the indexing pipeline
type Indexer struct {
	validator    Validator
	writer       *Writer
	graphBuilder *GraphBuilder
	embedder     Embedder
	config       *IndexerConfig
	db           *models.DB
	logger       IndexerLogger
}

// IndexerLogger defines the logging interface for the indexer
type IndexerLogger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	InfoWithFields(msg string, fields ...LogField)
	WarnWithFields(msg string, fields ...LogField)
	ErrorWithFields(msg string, err error, fields ...LogField)
	DebugWithFields(msg string, fields ...LogField)
}

// LogField represents a structured logging field
type LogField struct {
	Key   string
	Value interface{}
}

// IndexerConfig contains configuration options for the indexer
type IndexerConfig struct {
	// Repository information
	RepoID   string `json:"repo_id"`
	RepoName string `json:"repo_name"`
	RepoURL  string `json:"repo_url,omitempty"`
	Branch   string `json:"branch,omitempty"`

	// Processing options
	BatchSize       int  `json:"batch_size"`
	WorkerCount     int  `json:"worker_count"`
	SkipVectors     bool `json:"skip_vectors"`
	Incremental     bool `json:"incremental"`
	UseTransactions bool `json:"use_transactions"`

	// Graph options
	GraphName string `json:"graph_name"`

	// Embedding options
	EmbeddingModel string `json:"embedding_model,omitempty"`
}

// DefaultIndexerConfig returns default configuration for the indexer
func DefaultIndexerConfig() *IndexerConfig {
	return &IndexerConfig{
		BatchSize:       100,
		WorkerCount:     4,
		SkipVectors:     false,
		Incremental:     false,
		UseTransactions: true,
		GraphName:       "code_graph",
	}
}

// NewIndexer creates a new indexer instance
func NewIndexer(db *models.DB, config *IndexerConfig) *Indexer {
	return NewIndexerWithLogger(db, config, &noOpLogger{})
}

// NewIndexerWithLogger creates a new indexer instance with a custom logger
func NewIndexerWithLogger(db *models.DB, config *IndexerConfig, logger IndexerLogger) *Indexer {
	if config == nil {
		config = DefaultIndexerConfig()
	}

	if logger == nil {
		logger = &noOpLogger{}
	}

	// Create validator
	validator := NewSchemaValidator()

	// Create writer with config
	writerConfig := &WriterConfig{
		MaxRetries:     3,
		BaseRetryDelay: 100 * time.Millisecond,
		MaxRetryDelay:  5 * time.Second,
		BatchSize:      config.BatchSize,
	}
	writer := NewWriter(db, writerConfig)

	// Create graph builder with config
	graphConfig := &GraphBuilderConfig{
		GraphName: config.GraphName,
		BatchSize: config.BatchSize,
	}
	graphBuilder := NewGraphBuilder(db, graphConfig)

	// Create embedder with config (if not skipping vectors)
	var embedder Embedder
	if !config.SkipVectors {
		embedderConfig := DefaultEmbedderConfig()
		if config.EmbeddingModel != "" {
			embedderConfig.Model = config.EmbeddingModel
		}
		embedderConfig.BatchSize = config.BatchSize
		vectorRepo := models.NewVectorRepository(db)
		embedder = NewOpenAIEmbedder(embedderConfig, vectorRepo)
	}

	return &Indexer{
		validator:    validator,
		writer:       writer,
		graphBuilder: graphBuilder,
		embedder:     embedder,
		config:       config,
		db:           db,
		logger:       logger,
	}
}

// noOpLogger is a no-op logger implementation
type noOpLogger struct{}

func (l *noOpLogger) Info(msg string, args ...interface{})                          {}
func (l *noOpLogger) Warn(msg string, args ...interface{})                          {}
func (l *noOpLogger) Error(msg string, args ...interface{})                         {}
func (l *noOpLogger) Debug(msg string, args ...interface{})                         {}
func (l *noOpLogger) InfoWithFields(msg string, fields ...LogField)                 {}
func (l *noOpLogger) WarnWithFields(msg string, fields ...LogField)                 {}
func (l *noOpLogger) ErrorWithFields(msg string, err error, fields ...LogField)     {}
func (l *noOpLogger) DebugWithFields(msg string, fields ...LogField)                {}

// IndexResult contains the results of an indexing operation
type IndexResult struct {
	RepoID         string                 `json:"repo_id"`
	Status         string                 `json:"status"`
	FilesProcessed int                    `json:"files_processed"`
	SymbolsCreated int                    `json:"symbols_created"`
	NodesCreated   int                    `json:"nodes_created"`
	EdgesCreated   int                    `json:"edges_created"`
	VectorsCreated int                    `json:"vectors_created"`
	Duration       time.Duration          `json:"duration"`
	Errors         []*IndexerError        `json:"errors,omitempty"`
	Summary        map[string]interface{} `json:"summary,omitempty"`
}

// Index coordinates the validation → write → graph → embeddings pipeline
func (idx *Indexer) Index(ctx context.Context, input *schema.ParseOutput) (*IndexResult, error) {
	startTime := time.Now()
	result := &IndexResult{
		RepoID:  idx.config.RepoID,
		Status:  "in_progress",
		Summary: make(map[string]interface{}),
	}

	idx.logger.InfoWithFields("starting indexing operation",
		LogField{Key: "repo_id", Value: idx.config.RepoID},
		LogField{Key: "repo_name", Value: idx.config.RepoName},
		LogField{Key: "files_count", Value: len(input.Files)},
		LogField{Key: "relationships_count", Value: len(input.Relationships)},
	)

	// Collect errors throughout the process
	errorCollector := NewErrorCollector()

	// Step 1: Validate input
	idx.logger.Debug("validating input")
	validationResult := idx.validator.Validate(input)
	if validationResult.HasErrors() {
		idx.logger.ErrorWithFields("validation failed", nil,
			LogField{Key: "error_count", Value: validationResult.ErrorCount()},
			LogField{Key: "repo_id", Value: idx.config.RepoID},
		)
		for _, valErr := range validationResult.Errors {
			errorCollector.Add(NewValidationError(
				valErr.Message,
				valErr.EntityID,
				valErr.FilePath,
				nil,
			))
		}
		result.Status = "failed"
		result.Errors = convertErrors(errorCollector.Errors())
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("validation failed with %d errors", validationResult.ErrorCount())
	}
	idx.logger.InfoWithFields("validation completed successfully",
		LogField{Key: "repo_id", Value: idx.config.RepoID},
	)

	// Step 2: Write repository metadata
	idx.logger.Debug("writing repository metadata")
	if err := idx.writeRepository(ctx); err != nil {
		idx.logger.ErrorWithFields("failed to write repository metadata", err,
			LogField{Key: "repo_id", Value: idx.config.RepoID},
		)
		errorCollector.Add(NewDatabaseError(
			"failed to write repository metadata",
			idx.config.RepoID,
			"",
			err,
			true,
		))
		result.Status = "failed"
		result.Errors = convertErrors(errorCollector.Errors())
		result.Duration = time.Since(startTime)
		return result, err
	}
	idx.logger.InfoWithFields("repository metadata written",
		LogField{Key: "repo_id", Value: idx.config.RepoID},
	)

	// Step 3: Process files (with incremental support)
	filesToProcess := input.Files
	if idx.config.Incremental {
		idx.logger.Debug("filtering changed files for incremental indexing")
		filesToProcess = idx.filterChangedFiles(ctx, input.Files)
		idx.logger.InfoWithFields("incremental filtering completed",
			LogField{Key: "total_files", Value: len(input.Files)},
			LogField{Key: "changed_files", Value: len(filesToProcess)},
			LogField{Key: "skipped_files", Value: len(input.Files) - len(filesToProcess)},
		)
	}

	// Step 4: Write data to database
	idx.logger.InfoWithFields("writing data to database",
		LogField{Key: "files_to_process", Value: len(filesToProcess)},
		LogField{Key: "relationships", Value: len(input.Relationships)},
	)
	writeResult, err := idx.writeData(ctx, filesToProcess, input.Relationships)
	if err != nil {
		idx.logger.ErrorWithFields("failed to write data", err,
			LogField{Key: "repo_id", Value: idx.config.RepoID},
		)
		errorCollector.Add(NewDatabaseError(
			"failed to write data",
			"",
			"",
			err,
			true,
		))
	}
	result.FilesProcessed = writeResult.FilesProcessed
	result.SymbolsCreated = writeResult.SymbolsCreated
	result.NodesCreated = writeResult.NodesCreated
	result.EdgesCreated = writeResult.EdgesCreated

	idx.logger.InfoWithFields("data written to database",
		LogField{Key: "files_processed", Value: writeResult.FilesProcessed},
		LogField{Key: "symbols_created", Value: writeResult.SymbolsCreated},
		LogField{Key: "nodes_created", Value: writeResult.NodesCreated},
		LogField{Key: "edges_created", Value: writeResult.EdgesCreated},
		LogField{Key: "write_errors", Value: len(writeResult.Errors)},
	)

	// Collect write errors
	for _, writeErr := range writeResult.Errors {
		idx.logger.WarnWithFields("write error occurred",
			LogField{Key: "entity_type", Value: writeErr.EntityType},
			LogField{Key: "entity_id", Value: writeErr.EntityID},
			LogField{Key: "message", Value: writeErr.Message},
			LogField{Key: "retryable", Value: writeErr.Retryable},
		)
		errorCollector.Add(NewDatabaseError(
			writeErr.Message,
			writeErr.EntityID,
			"",
			nil,
			writeErr.Retryable,
		))
	}

	// Step 5: Build graph (async, non-blocking)
	if idx.graphBuilder != nil {
		idx.logger.Info("building knowledge graph")
		graphResult := idx.buildGraph(ctx, filesToProcess, input.Relationships)
		result.Summary["graph_nodes_created"] = graphResult.NodesCreated
		result.Summary["graph_edges_created"] = graphResult.EdgesCreated

		idx.logger.InfoWithFields("knowledge graph built",
			LogField{Key: "nodes_created", Value: graphResult.NodesCreated},
			LogField{Key: "edges_created", Value: graphResult.EdgesCreated},
			LogField{Key: "graph_errors", Value: len(graphResult.Errors)},
		)

		// Collect graph errors (non-fatal)
		for _, graphErr := range graphResult.Errors {
			idx.logger.WarnWithFields("graph error occurred",
				LogField{Key: "entity_type", Value: graphErr.EntityType},
				LogField{Key: "entity_id", Value: graphErr.EntityID},
				LogField{Key: "message", Value: graphErr.Message},
			)
			errorCollector.Add(NewGraphError(
				graphErr.Message,
				graphErr.EntityID,
				"",
				nil,
			))
		}
	}

	// Step 6: Generate embeddings (async, optional)
	if idx.embedder != nil && !idx.config.SkipVectors {
		idx.logger.Info("generating vector embeddings")
		embedResult := idx.generateEmbeddings(ctx, filesToProcess)
		result.VectorsCreated = embedResult.VectorsCreated

		idx.logger.InfoWithFields("vector embeddings generated",
			LogField{Key: "vectors_created", Value: embedResult.VectorsCreated},
			LogField{Key: "embedding_errors", Value: len(embedResult.Errors)},
		)

		// Collect embedding errors (non-fatal)
		for _, embedErr := range embedResult.Errors {
			idx.logger.WarnWithFields("embedding error occurred",
				LogField{Key: "entity_id", Value: embedErr.EntityID},
				LogField{Key: "message", Value: embedErr.Message},
			)
			errorCollector.Add(NewEmbeddingError(
				embedErr.Message,
				embedErr.EntityID,
				"",
				nil,
				true,
			))
		}
	}

	// Finalize result
	result.Duration = time.Since(startTime)
	result.Errors = convertErrors(errorCollector.Errors())

	// Determine final status
	if errorCollector.HasErrors() {
		nonRetryable := errorCollector.FilterNonRetryable()
		if len(nonRetryable) > 0 {
			result.Status = "partial_success"
			idx.logger.WarnWithFields("indexing completed with errors",
				LogField{Key: "status", Value: result.Status},
				LogField{Key: "total_errors", Value: errorCollector.Count()},
				LogField{Key: "non_retryable_errors", Value: len(nonRetryable)},
			)
		} else {
			result.Status = "success_with_warnings"
			idx.logger.InfoWithFields("indexing completed with warnings",
				LogField{Key: "status", Value: result.Status},
				LogField{Key: "total_warnings", Value: errorCollector.Count()},
			)
		}
	} else {
		result.Status = "success"
		idx.logger.InfoWithFields("indexing completed successfully",
			LogField{Key: "status", Value: result.Status},
		)
	}

	// Add summary statistics
	result.Summary["total_errors"] = errorCollector.Count()
	result.Summary["error_types"] = errorCollector.Summary()
	result.Summary["validation_errors"] = validationResult.ErrorCount()

	idx.logger.InfoWithFields("indexing operation completed",
		LogField{Key: "repo_id", Value: idx.config.RepoID},
		LogField{Key: "status", Value: result.Status},
		LogField{Key: "duration", Value: result.Duration},
		LogField{Key: "files_processed", Value: result.FilesProcessed},
		LogField{Key: "symbols_created", Value: result.SymbolsCreated},
		LogField{Key: "edges_created", Value: result.EdgesCreated},
		LogField{Key: "vectors_created", Value: result.VectorsCreated},
		LogField{Key: "total_errors", Value: errorCollector.Count()},
	)

	return result, nil
}

// IndexWithProgress indexes with progress tracking
func (idx *Indexer) IndexWithProgress(ctx context.Context, input *schema.ParseOutput, progressChan chan<- IndexProgress) (*IndexResult, error) {
	startTime := time.Now()

	// Send initial progress
	if progressChan != nil {
		progressChan <- IndexProgress{
			Stage:      "validation",
			Progress:   0,
			TotalFiles: len(input.Files),
			Message:    "Validating input...",
		}
	}

	// Validate
	validationResult := idx.validator.Validate(input)
	if validationResult.HasErrors() {
		if progressChan != nil {
			progressChan <- IndexProgress{
				Stage:    "validation",
				Progress: 0,
				Message:  fmt.Sprintf("Validation failed with %d errors", validationResult.ErrorCount()),
				Error:    true,
			}
		}
		return nil, fmt.Errorf("validation failed")
	}

	if progressChan != nil {
		progressChan <- IndexProgress{
			Stage:    "validation",
			Progress: 100,
			Message:  "Validation complete",
		}
	}

	// Write repository
	if progressChan != nil {
		progressChan <- IndexProgress{
			Stage:    "repository",
			Progress: 0,
			Message:  "Writing repository metadata...",
		}
	}

	if err := idx.writeRepository(ctx); err != nil {
		return nil, err
	}

	if progressChan != nil {
		progressChan <- IndexProgress{
			Stage:    "repository",
			Progress: 100,
			Message:  "Repository metadata written",
		}
	}

	// Process files with progress updates
	filesToProcess := input.Files
	if idx.config.Incremental {
		filesToProcess = idx.filterChangedFiles(ctx, input.Files)
		if progressChan != nil {
			progressChan <- IndexProgress{
				Stage:      "incremental",
				Progress:   100,
				TotalFiles: len(filesToProcess),
				Message:    fmt.Sprintf("Filtered to %d changed files", len(filesToProcess)),
			}
		}
	}

	// Write data with progress
	if progressChan != nil {
		progressChan <- IndexProgress{
			Stage:      "writing",
			Progress:   0,
			TotalFiles: len(filesToProcess),
			Message:    "Writing data to database...",
		}
	}

	writeResult, err := idx.writeData(ctx, filesToProcess, input.Relationships)
	if err != nil {
		return nil, err
	}

	if progressChan != nil {
		progressChan <- IndexProgress{
			Stage:          "writing",
			Progress:       100,
			FilesProcessed: writeResult.FilesProcessed,
			Message:        fmt.Sprintf("Wrote %d files, %d symbols", writeResult.FilesProcessed, writeResult.SymbolsCreated),
		}
	}

	// Build graph
	if idx.graphBuilder != nil {
		if progressChan != nil {
			progressChan <- IndexProgress{
				Stage:    "graph",
				Progress: 0,
				Message:  "Building knowledge graph...",
			}
		}

		graphResult := idx.buildGraph(ctx, filesToProcess, input.Relationships)

		if progressChan != nil {
			progressChan <- IndexProgress{
				Stage:    "graph",
				Progress: 100,
				Message:  fmt.Sprintf("Created %d nodes, %d edges", graphResult.NodesCreated, graphResult.EdgesCreated),
			}
		}
	}

	// Generate embeddings
	if idx.embedder != nil && !idx.config.SkipVectors {
		if progressChan != nil {
			progressChan <- IndexProgress{
				Stage:    "embeddings",
				Progress: 0,
				Message:  "Generating embeddings...",
			}
		}

		embedResult := idx.generateEmbeddings(ctx, filesToProcess)

		if progressChan != nil {
			progressChan <- IndexProgress{
				Stage:    "embeddings",
				Progress: 100,
				Message:  fmt.Sprintf("Generated %d embeddings", embedResult.VectorsCreated),
			}
		}
	}

	// Complete
	if progressChan != nil {
		progressChan <- IndexProgress{
			Stage:    "complete",
			Progress: 100,
			Message:  fmt.Sprintf("Indexing complete in %s", time.Since(startTime)),
		}
		close(progressChan)
	}

	return idx.Index(ctx, input)
}

// writeRepository writes repository metadata
func (idx *Indexer) writeRepository(ctx context.Context) error {
	// Generate repo ID if not provided
	if idx.config.RepoID == "" {
		idx.config.RepoID = uuid.New().String()
	}

	repo := &models.Repository{
		RepoID: idx.config.RepoID,
		Name:   idx.config.RepoName,
		URL:    idx.config.RepoURL,
		Branch: idx.config.Branch,
	}

	return idx.writer.WriteRepository(ctx, repo)
}

// filterChangedFiles filters files based on checksum for incremental indexing
func (idx *Indexer) filterChangedFiles(ctx context.Context, files []schema.File) []schema.File {
	fileRepo := models.NewFileRepository(idx.db)
	var changedFiles []schema.File

	for _, file := range files {
		// Check if file exists with same checksum
		existingFile, err := fileRepo.GetByID(ctx, file.FileID)
		if err != nil || existingFile == nil {
			// File doesn't exist, include it
			changedFiles = append(changedFiles, file)
			continue
		}

		// Check if checksum changed
		if existingFile.Checksum != file.Checksum {
			changedFiles = append(changedFiles, file)
		}
	}

	return changedFiles
}

// writeData writes files, symbols, nodes, and edges to database
func (idx *Indexer) writeData(ctx context.Context, files []schema.File, edges []schema.DependencyEdge) (*WriteResult, error) {
	result := &WriteResult{}

	if idx.config.UseTransactions {
		return idx.writeDataWithTransaction(ctx, files, edges)
	}

	// Write files
	filesResult, err := idx.writer.WriteFiles(ctx, idx.config.RepoID, files)
	if err != nil {
		return result, err
	}
	result.FilesProcessed = filesResult.FilesProcessed
	result.Errors = append(result.Errors, filesResult.Errors...)

	// Collect all symbols and nodes from files
	var allSymbols []schema.Symbol
	var allNodes []schema.ASTNode
	for _, file := range files {
		allSymbols = append(allSymbols, file.Symbols...)
		allNodes = append(allNodes, file.Nodes...)
	}

	// Write symbols
	symbolsResult, err := idx.writer.WriteSymbols(ctx, allSymbols)
	if err != nil {
		return result, err
	}
	result.SymbolsCreated = symbolsResult.SymbolsCreated
	result.Errors = append(result.Errors, symbolsResult.Errors...)

	// Write AST nodes
	nodesResult, err := idx.writer.WriteASTNodes(ctx, allNodes)
	if err != nil {
		return result, err
	}
	result.NodesCreated = nodesResult.NodesCreated
	result.Errors = append(result.Errors, nodesResult.Errors...)

	// Write edges
	edgesResult, err := idx.writer.WriteEdges(ctx, edges)
	if err != nil {
		return result, err
	}
	result.EdgesCreated = edgesResult.EdgesCreated
	result.Errors = append(result.Errors, edgesResult.Errors...)

	return result, nil
}

// writeDataWithTransaction writes all data within a single transaction
func (idx *Indexer) writeDataWithTransaction(ctx context.Context, files []schema.File, edges []schema.DependencyEdge) (*WriteResult, error) {
	result := &WriteResult{}

	// Begin transaction
	tx, err := idx.writer.BeginTx(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Write files
	filesResult, err := idx.writer.WriteFilesWithTransaction(ctx, idx.config.RepoID, files)
	if err != nil {
		return result, fmt.Errorf("failed to write files: %w", err)
	}
	result.FilesProcessed = filesResult.FilesProcessed

	// Collect all symbols and nodes
	var allSymbols []schema.Symbol
	var allNodes []schema.ASTNode
	for _, file := range files {
		allSymbols = append(allSymbols, file.Symbols...)
		allNodes = append(allNodes, file.Nodes...)
	}

	// Write symbols
	symbolsResult, err := idx.writer.WriteSymbolsWithTransaction(ctx, allSymbols)
	if err != nil {
		return result, fmt.Errorf("failed to write symbols: %w", err)
	}
	result.SymbolsCreated = symbolsResult.SymbolsCreated

	// Write AST nodes
	nodesResult, err := idx.writer.WriteASTNodesWithTransaction(ctx, allNodes)
	if err != nil {
		return result, fmt.Errorf("failed to write AST nodes: %w", err)
	}
	result.NodesCreated = nodesResult.NodesCreated

	// Write edges
	edgesResult, err := idx.writer.WriteEdgesWithTransaction(ctx, edges)
	if err != nil {
		return result, fmt.Errorf("failed to write edges: %w", err)
	}
	result.EdgesCreated = edgesResult.EdgesCreated

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// buildGraph builds the knowledge graph from symbols and edges
func (idx *Indexer) buildGraph(ctx context.Context, files []schema.File, edges []schema.DependencyEdge) *GraphBuildResult {
	result := &GraphBuildResult{}

	// Collect all symbols
	var allSymbols []schema.Symbol
	for _, file := range files {
		allSymbols = append(allSymbols, file.Symbols...)
	}

	// Create nodes in parallel
	if len(allSymbols) > 0 {
		nodesResult, err := idx.graphBuilder.CreateNodes(ctx, allSymbols)
		if err != nil {
			result.Errors = append(result.Errors, GraphError{
				EntityType: "nodes",
				Message:    err.Error(),
			})
		} else {
			result.NodesCreated = nodesResult.NodesCreated
			result.Errors = append(result.Errors, nodesResult.Errors...)
		}
	}

	// Create edges in parallel
	if len(edges) > 0 {
		edgesResult, err := idx.graphBuilder.CreateEdges(ctx, edges)
		if err != nil {
			result.Errors = append(result.Errors, GraphError{
				EntityType: "edges",
				Message:    err.Error(),
			})
		} else {
			result.EdgesCreated = edgesResult.EdgesCreated
			result.Errors = append(result.Errors, edgesResult.Errors...)
		}
	}

	return result
}

// generateEmbeddings generates vector embeddings for symbols
func (idx *Indexer) generateEmbeddings(ctx context.Context, files []schema.File) *EmbedResult {
	result := &EmbedResult{}

	// Collect all symbols
	var allSymbols []schema.Symbol
	for _, file := range files {
		allSymbols = append(allSymbols, file.Symbols...)
	}

	if len(allSymbols) == 0 {
		return result
	}

	// Process symbols in parallel batches
	if idx.config.WorkerCount > 1 {
		return idx.generateEmbeddingsParallel(ctx, allSymbols)
	}

	// Sequential processing
	embedResult, err := idx.embedder.EmbedSymbols(ctx, allSymbols)
	if err != nil {
		result.Errors = append(result.Errors, EmbedError{
			Message: err.Error(),
		})
	} else {
		result.VectorsCreated = embedResult.VectorsCreated
		result.Errors = append(result.Errors, embedResult.Errors...)
	}

	return result
}

// generateEmbeddingsParallel generates embeddings using parallel workers
func (idx *Indexer) generateEmbeddingsParallel(ctx context.Context, symbols []schema.Symbol) *EmbedResult {
	result := &EmbedResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Calculate batch size per worker
	batchSize := (len(symbols) + idx.config.WorkerCount - 1) / idx.config.WorkerCount

	// Process in parallel
	for i := 0; i < idx.config.WorkerCount; i++ {
		start := i * batchSize
		if start >= len(symbols) {
			break
		}
		end := start + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}

		wg.Add(1)
		go func(batch []schema.Symbol) {
			defer wg.Done()

			embedResult, err := idx.embedder.EmbedSymbols(ctx, batch)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.Errors = append(result.Errors, EmbedError{
					Message: err.Error(),
				})
			} else {
				result.VectorsCreated += embedResult.VectorsCreated
				result.Errors = append(result.Errors, embedResult.Errors...)
			}
		}(symbols[start:end])
	}

	wg.Wait()
	return result
}

// IndexProgress represents progress information during indexing
type IndexProgress struct {
	Stage          string  `json:"stage"`
	Progress       float64 `json:"progress"`
	TotalFiles     int     `json:"total_files,omitempty"`
	FilesProcessed int     `json:"files_processed,omitempty"`
	Message        string  `json:"message"`
	Error          bool    `json:"error,omitempty"`
}

// convertErrors converts internal errors to IndexerError format
func convertErrors(errors []error) []*IndexerError {
	var result []*IndexerError
	for _, err := range errors {
		if indexerErr, ok := err.(*IndexerError); ok {
			result = append(result, indexerErr)
		} else {
			result = append(result, &IndexerError{
				Type:      "unknown",
				Message:   err.Error(),
				Retryable: false,
			})
		}
	}
	return result
}
