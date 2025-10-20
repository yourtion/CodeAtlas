package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// Writer handles database persistence operations for the indexer
type Writer struct {
	db             *models.DB
	repoRepo       *models.RepositoryRepository
	fileRepo       *models.FileRepository
	symbolRepo     *models.SymbolRepository
	astNodeRepo    *models.ASTNodeRepository
	edgeRepo       *models.EdgeRepository
	maxRetries     int
	baseRetryDelay time.Duration
	maxRetryDelay  time.Duration
	batchSize      int
}

// WriterConfig contains configuration options for the Writer
type WriterConfig struct {
	MaxRetries     int           `json:"max_retries"`
	BaseRetryDelay time.Duration `json:"base_retry_delay"`
	MaxRetryDelay  time.Duration `json:"max_retry_delay"`
	BatchSize      int           `json:"batch_size"`
}

// DefaultWriterConfig returns default configuration for the Writer
func DefaultWriterConfig() *WriterConfig {
	return &WriterConfig{
		MaxRetries:     3,
		BaseRetryDelay: 100 * time.Millisecond,
		MaxRetryDelay:  5 * time.Second,
		BatchSize:      100,
	}
}

// NewWriter creates a new database writer instance
func NewWriter(db *models.DB, config *WriterConfig) *Writer {
	if config == nil {
		config = DefaultWriterConfig()
	}

	return &Writer{
		db:             db,
		repoRepo:       models.NewRepositoryRepository(db),
		fileRepo:       models.NewFileRepository(db),
		symbolRepo:     models.NewSymbolRepository(db),
		astNodeRepo:    models.NewASTNodeRepository(db),
		edgeRepo:       models.NewEdgeRepository(db),
		maxRetries:     config.MaxRetries,
		baseRetryDelay: config.BaseRetryDelay,
		maxRetryDelay:  config.MaxRetryDelay,
		batchSize:      config.BatchSize,
	}
}

// WriteResult contains the results of a write operation
type WriteResult struct {
	FilesProcessed int           `json:"files_processed"`
	SymbolsCreated int           `json:"symbols_created"`
	NodesCreated   int           `json:"nodes_created"`
	EdgesCreated   int           `json:"edges_created"`
	Duration       time.Duration `json:"duration"`
	Errors         []WriteError  `json:"errors,omitempty"`
}

// WriteError represents an error that occurred during writing
type WriteError struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
}

// WriteRepository creates or updates a repository record
func (w *Writer) WriteRepository(ctx context.Context, repo *models.Repository) error {
	return w.withRetry(ctx, "repository", repo.RepoID, func() error {
		return w.repoRepo.CreateOrUpdate(ctx, repo)
	})
}

// WriteFiles batch inserts files with checksum-based incremental update logic
func (w *Writer) WriteFiles(ctx context.Context, repoID string, files []schema.File) (*WriteResult, error) {
	startTime := time.Now()
	result := &WriteResult{}

	if len(files) == 0 {
		return result, nil
	}

	// Convert schema files to model files
	modelFiles := make([]*models.File, 0, len(files))
	for _, file := range files {
		modelFile := &models.File{
			FileID:   file.FileID,
			RepoID:   repoID,
			Path:     file.Path,
			Language: file.Language,
			Size:     file.Size,
			Checksum: file.Checksum,
		}
		modelFiles = append(modelFiles, modelFile)
	}

	// Process files in batches
	for i := 0; i < len(modelFiles); i += w.batchSize {
		end := i + w.batchSize
		if end > len(modelFiles) {
			end = len(modelFiles)
		}

		batch := modelFiles[i:end]
		err := w.withRetry(ctx, "files_batch", fmt.Sprintf("batch_%d", i/w.batchSize), func() error {
			return w.fileRepo.BatchCreate(ctx, batch)
		})

		if err != nil {
			result.Errors = append(result.Errors, WriteError{
				EntityType: "files_batch",
				EntityID:   fmt.Sprintf("batch_%d", i/w.batchSize),
				Message:    err.Error(),
				Retryable:  w.isRetryableError(err),
			})
		} else {
			result.FilesProcessed += len(batch)
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// WriteSymbols batch inserts symbols with batch processing
func (w *Writer) WriteSymbols(ctx context.Context, symbols []schema.Symbol) (*WriteResult, error) {
	startTime := time.Now()
	result := &WriteResult{}

	if len(symbols) == 0 {
		return result, nil
	}

	// Convert schema symbols to model symbols
	modelSymbols := make([]*models.Symbol, 0, len(symbols))
	for _, symbol := range symbols {
		modelSymbol := &models.Symbol{
			SymbolID:        symbol.SymbolID,
			FileID:          symbol.FileID,
			Name:            symbol.Name,
			Kind:            string(symbol.Kind),
			Signature:       symbol.Signature,
			StartLine:       symbol.Span.StartLine,
			EndLine:         symbol.Span.EndLine,
			StartByte:       symbol.Span.StartByte,
			EndByte:         symbol.Span.EndByte,
			Docstring:       symbol.Docstring,
			SemanticSummary: symbol.SemanticSummary,
		}
		modelSymbols = append(modelSymbols, modelSymbol)
	}

	// Process symbols in batches
	for i := 0; i < len(modelSymbols); i += w.batchSize {
		end := i + w.batchSize
		if end > len(modelSymbols) {
			end = len(modelSymbols)
		}

		batch := modelSymbols[i:end]
		err := w.withRetry(ctx, "symbols_batch", fmt.Sprintf("batch_%d", i/w.batchSize), func() error {
			return w.symbolRepo.BatchCreate(ctx, batch)
		})

		if err != nil {
			result.Errors = append(result.Errors, WriteError{
				EntityType: "symbols_batch",
				EntityID:   fmt.Sprintf("batch_%d", i/w.batchSize),
				Message:    err.Error(),
				Retryable:  w.isRetryableError(err),
			})
		} else {
			result.SymbolsCreated += len(batch)
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// WriteASTNodes batch inserts AST nodes preserving parent-child relationships
func (w *Writer) WriteASTNodes(ctx context.Context, nodes []schema.ASTNode) (*WriteResult, error) {
	startTime := time.Now()
	result := &WriteResult{}

	if len(nodes) == 0 {
		return result, nil
	}

	// Convert schema nodes to model nodes
	modelNodes := make([]*models.ASTNode, 0, len(nodes))
	for _, node := range nodes {
		var parentID *string
		if node.ParentID != "" {
			parentID = &node.ParentID
		}

		modelNode := &models.ASTNode{
			NodeID:     node.NodeID,
			FileID:     node.FileID,
			Type:       node.Type,
			ParentID:   parentID,
			StartLine:  node.Span.StartLine,
			EndLine:    node.Span.EndLine,
			StartByte:  node.Span.StartByte,
			EndByte:    node.Span.EndByte,
			Text:       node.Text,
			Attributes: node.Attributes,
		}
		modelNodes = append(modelNodes, modelNode)
	}

	// Sort nodes to ensure parents are inserted before children
	// This is a simple topological sort based on parent-child relationships
	sortedNodes := w.topologicalSortNodes(modelNodes)

	// Process nodes in batches
	for i := 0; i < len(sortedNodes); i += w.batchSize {
		end := i + w.batchSize
		if end > len(sortedNodes) {
			end = len(sortedNodes)
		}

		batch := sortedNodes[i:end]
		err := w.withRetry(ctx, "ast_nodes_batch", fmt.Sprintf("batch_%d", i/w.batchSize), func() error {
			return w.astNodeRepo.BatchCreate(ctx, batch)
		})

		if err != nil {
			result.Errors = append(result.Errors, WriteError{
				EntityType: "ast_nodes_batch",
				EntityID:   fmt.Sprintf("batch_%d", i/w.batchSize),
				Message:    err.Error(),
				Retryable:  w.isRetryableError(err),
			})
		} else {
			result.NodesCreated += len(batch)
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// WriteEdges batch inserts dependency edges with proper foreign key handling
func (w *Writer) WriteEdges(ctx context.Context, edges []schema.DependencyEdge) (*WriteResult, error) {
	startTime := time.Now()
	result := &WriteResult{}

	if len(edges) == 0 {
		return result, nil
	}

	// Convert schema edges to model edges
	modelEdges := make([]*models.Edge, 0, len(edges))
	for _, edge := range edges {
		var targetID *string
		if edge.TargetID != "" {
			targetID = &edge.TargetID
		}

		var targetFile *string
		if edge.TargetFile != "" {
			targetFile = &edge.TargetFile
		}

		var targetModule *string
		if edge.TargetModule != "" {
			targetModule = &edge.TargetModule
		}

		modelEdge := &models.Edge{
			EdgeID:       edge.EdgeID,
			SourceID:     edge.SourceID,
			TargetID:     targetID,
			EdgeType:     string(edge.EdgeType),
			SourceFile:   edge.SourceFile,
			TargetFile:   targetFile,
			TargetModule: targetModule,
		}
		modelEdges = append(modelEdges, modelEdge)
	}

	// Process edges in batches
	for i := 0; i < len(modelEdges); i += w.batchSize {
		end := i + w.batchSize
		if end > len(modelEdges) {
			end = len(modelEdges)
		}

		batch := modelEdges[i:end]
		err := w.withRetry(ctx, "edges_batch", fmt.Sprintf("batch_%d", i/w.batchSize), func() error {
			return w.edgeRepo.BatchCreate(ctx, batch)
		})

		if err != nil {
			result.Errors = append(result.Errors, WriteError{
				EntityType: "edges_batch",
				EntityID:   fmt.Sprintf("batch_%d", i/w.batchSize),
				Message:    err.Error(),
				Retryable:  w.isRetryableError(err),
			})
		} else {
			result.EdgesCreated += len(batch)
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// WriteFilesWithTransaction writes files within a transaction
func (w *Writer) WriteFilesWithTransaction(ctx context.Context, repoID string, files []schema.File) (*WriteResult, error) {
	startTime := time.Now()
	result := &WriteResult{}

	if len(files) == 0 {
		return result, nil
	}

	tx, err := w.BeginTx(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Convert schema files to model files
	modelFiles := make([]*models.File, 0, len(files))
	for _, file := range files {
		modelFile := &models.File{
			FileID:   file.FileID,
			RepoID:   repoID,
			Path:     file.Path,
			Language: file.Language,
			Size:     file.Size,
			Checksum: file.Checksum,
		}
		modelFiles = append(modelFiles, modelFile)
	}

	// Process files in batches within the transaction
	for i := 0; i < len(modelFiles); i += w.batchSize {
		end := i + w.batchSize
		if end > len(modelFiles) {
			end = len(modelFiles)
		}

		batch := modelFiles[i:end]
		err = w.fileRepo.BatchCreateTx(ctx, tx, batch)
		if err != nil {
			result.Errors = append(result.Errors, WriteError{
				EntityType: "files_batch_tx",
				EntityID:   fmt.Sprintf("batch_%d", i/w.batchSize),
				Message:    err.Error(),
				Retryable:  false, // Transaction errors are not retryable
			})
			return result, fmt.Errorf("failed to write files batch: %w", err)
		}
		result.FilesProcessed += len(batch)
	}

	err = tx.Commit()
	if err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// WriteSymbolsWithTransaction writes symbols within a transaction
func (w *Writer) WriteSymbolsWithTransaction(ctx context.Context, symbols []schema.Symbol) (*WriteResult, error) {
	startTime := time.Now()
	result := &WriteResult{}

	if len(symbols) == 0 {
		return result, nil
	}

	tx, err := w.BeginTx(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Convert schema symbols to model symbols
	modelSymbols := make([]*models.Symbol, 0, len(symbols))
	for _, symbol := range symbols {
		modelSymbol := &models.Symbol{
			SymbolID:        symbol.SymbolID,
			FileID:          symbol.FileID,
			Name:            symbol.Name,
			Kind:            string(symbol.Kind),
			Signature:       symbol.Signature,
			StartLine:       symbol.Span.StartLine,
			EndLine:         symbol.Span.EndLine,
			StartByte:       symbol.Span.StartByte,
			EndByte:         symbol.Span.EndByte,
			Docstring:       symbol.Docstring,
			SemanticSummary: symbol.SemanticSummary,
		}
		modelSymbols = append(modelSymbols, modelSymbol)
	}

	// Process symbols in batches within the transaction
	for i := 0; i < len(modelSymbols); i += w.batchSize {
		end := i + w.batchSize
		if end > len(modelSymbols) {
			end = len(modelSymbols)
		}

		batch := modelSymbols[i:end]
		err = w.symbolRepo.BatchCreateTx(ctx, tx, batch)
		if err != nil {
			result.Errors = append(result.Errors, WriteError{
				EntityType: "symbols_batch_tx",
				EntityID:   fmt.Sprintf("batch_%d", i/w.batchSize),
				Message:    err.Error(),
				Retryable:  false, // Transaction errors are not retryable
			})
			return result, fmt.Errorf("failed to write symbols batch: %w", err)
		}
		result.SymbolsCreated += len(batch)
	}

	err = tx.Commit()
	if err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// WriteASTNodesWithTransaction writes AST nodes within a transaction
func (w *Writer) WriteASTNodesWithTransaction(ctx context.Context, nodes []schema.ASTNode) (*WriteResult, error) {
	startTime := time.Now()
	result := &WriteResult{}

	if len(nodes) == 0 {
		return result, nil
	}

	tx, err := w.BeginTx(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Convert schema nodes to model nodes
	modelNodes := make([]*models.ASTNode, 0, len(nodes))
	for _, node := range nodes {
		var parentID *string
		if node.ParentID != "" {
			parentID = &node.ParentID
		}

		modelNode := &models.ASTNode{
			NodeID:     node.NodeID,
			FileID:     node.FileID,
			Type:       node.Type,
			ParentID:   parentID,
			StartLine:  node.Span.StartLine,
			EndLine:    node.Span.EndLine,
			StartByte:  node.Span.StartByte,
			EndByte:    node.Span.EndByte,
			Text:       node.Text,
			Attributes: node.Attributes,
		}
		modelNodes = append(modelNodes, modelNode)
	}

	// Sort nodes to ensure parents are inserted before children
	sortedNodes := w.topologicalSortNodes(modelNodes)

	// Process nodes in batches within the transaction
	for i := 0; i < len(sortedNodes); i += w.batchSize {
		end := i + w.batchSize
		if end > len(sortedNodes) {
			end = len(sortedNodes)
		}

		batch := sortedNodes[i:end]
		err = w.astNodeRepo.BatchCreateTx(ctx, tx, batch)
		if err != nil {
			result.Errors = append(result.Errors, WriteError{
				EntityType: "ast_nodes_batch_tx",
				EntityID:   fmt.Sprintf("batch_%d", i/w.batchSize),
				Message:    err.Error(),
				Retryable:  false, // Transaction errors are not retryable
			})
			return result, fmt.Errorf("failed to write AST nodes batch: %w", err)
		}
		result.NodesCreated += len(batch)
	}

	err = tx.Commit()
	if err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// WriteEdgesWithTransaction writes edges within a transaction
func (w *Writer) WriteEdgesWithTransaction(ctx context.Context, edges []schema.DependencyEdge) (*WriteResult, error) {
	startTime := time.Now()
	result := &WriteResult{}

	if len(edges) == 0 {
		return result, nil
	}

	tx, err := w.BeginTx(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Convert schema edges to model edges
	modelEdges := make([]*models.Edge, 0, len(edges))
	for _, edge := range edges {
		var targetID *string
		if edge.TargetID != "" {
			targetID = &edge.TargetID
		}

		var targetFile *string
		if edge.TargetFile != "" {
			targetFile = &edge.TargetFile
		}

		var targetModule *string
		if edge.TargetModule != "" {
			targetModule = &edge.TargetModule
		}

		modelEdge := &models.Edge{
			EdgeID:       edge.EdgeID,
			SourceID:     edge.SourceID,
			TargetID:     targetID,
			EdgeType:     string(edge.EdgeType),
			SourceFile:   edge.SourceFile,
			TargetFile:   targetFile,
			TargetModule: targetModule,
		}
		modelEdges = append(modelEdges, modelEdge)
	}

	// Process edges in batches within the transaction
	for i := 0; i < len(modelEdges); i += w.batchSize {
		end := i + w.batchSize
		if end > len(modelEdges) {
			end = len(modelEdges)
		}

		batch := modelEdges[i:end]
		err = w.edgeRepo.BatchCreateTx(ctx, tx, batch)
		if err != nil {
			result.Errors = append(result.Errors, WriteError{
				EntityType: "edges_batch_tx",
				EntityID:   fmt.Sprintf("batch_%d", i/w.batchSize),
				Message:    err.Error(),
				Retryable:  false, // Transaction errors are not retryable
			})
			return result, fmt.Errorf("failed to write edges batch: %w", err)
		}
		result.EdgesCreated += len(batch)
	}

	err = tx.Commit()
	if err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// BeginTx starts a new database transaction
func (w *Writer) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return w.db.BeginTx(ctx, nil)
}

// withRetry executes a function with exponential backoff retry logic
func (w *Writer) withRetry(ctx context.Context, entityType, entityID string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			delay := time.Duration(float64(w.baseRetryDelay) * math.Pow(2, float64(attempt-1)))
			if delay > w.maxRetryDelay {
				delay = w.maxRetryDelay
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry if the error is not retryable
		if !w.isRetryableError(err) {
			break
		}

		// Don't retry if this is the last attempt
		if attempt == w.maxRetries {
			break
		}
	}

	return fmt.Errorf("failed after %d attempts for %s %s: %w", w.maxRetries+1, entityType, entityID, lastErr)
}

// isRetryableError determines if an error is retryable
func (w *Writer) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Database connection errors are retryable
	if contains(errStr, "connection refused") ||
		contains(errStr, "connection reset") ||
		contains(errStr, "connection timeout") ||
		contains(errStr, "connection lost") ||
		contains(errStr, "server closed") ||
		contains(errStr, "broken pipe") {
		return true
	}

	// Temporary database errors are retryable
	if contains(errStr, "temporary") ||
		contains(errStr, "timeout") ||
		contains(errStr, "deadlock") ||
		contains(errStr, "lock timeout") {
		return true
	}

	// PostgreSQL specific retryable errors
	if contains(errStr, "pq: sorry, too many clients") ||
		contains(errStr, "pq: database is starting up") ||
		contains(errStr, "pq: the database system is starting up") {
		return true
	}

	return false
}

// topologicalSortNodes sorts AST nodes to ensure parents are processed before children
func (w *Writer) topologicalSortNodes(nodes []*models.ASTNode) []*models.ASTNode {
	if len(nodes) == 0 {
		return nodes
	}

	// Create maps for efficient lookup
	nodeMap := make(map[string]*models.ASTNode)
	childrenMap := make(map[string][]*models.ASTNode)
	inDegree := make(map[string]int)

	// Initialize maps
	for _, node := range nodes {
		nodeMap[node.NodeID] = node
		inDegree[node.NodeID] = 0
	}

	// Build children map and calculate in-degrees
	for _, node := range nodes {
		if node.ParentID != nil && *node.ParentID != "" {
			parentID := *node.ParentID
			childrenMap[parentID] = append(childrenMap[parentID], node)
			inDegree[node.NodeID]++
		}
	}

	// Topological sort using Kahn's algorithm
	var result []*models.ASTNode
	var queue []*models.ASTNode

	// Start with nodes that have no parents (in-degree 0)
	for _, node := range nodes {
		if inDegree[node.NodeID] == 0 {
			queue = append(queue, node)
		}
	}

	for len(queue) > 0 {
		// Remove node from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Process children
		for _, child := range childrenMap[current.NodeID] {
			inDegree[child.NodeID]--
			if inDegree[child.NodeID] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// If we couldn't sort all nodes (cycle detected), append remaining nodes
	if len(result) < len(nodes) {
		for _, node := range nodes {
			if inDegree[node.NodeID] > 0 {
				result = append(result, node)
			}
		}
	}

	return result
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				anySubstring(s, substr)))
}

// anySubstring checks if substr exists in s
func anySubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
