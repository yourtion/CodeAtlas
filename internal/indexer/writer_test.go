package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWriter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test with default config
	writer := NewWriter(db, nil)
	assert.NotNil(t, writer)
	assert.Equal(t, 3, writer.maxRetries)
	assert.Equal(t, 100*time.Millisecond, writer.baseRetryDelay)
	assert.Equal(t, 5*time.Second, writer.maxRetryDelay)
	assert.Equal(t, 100, writer.batchSize)

	// Test with custom config
	config := &WriterConfig{
		MaxRetries:     5,
		BaseRetryDelay: 200 * time.Millisecond,
		MaxRetryDelay:  10 * time.Second,
		BatchSize:      50,
	}
	writer = NewWriter(db, config)
	assert.Equal(t, 5, writer.maxRetries)
	assert.Equal(t, 200*time.Millisecond, writer.baseRetryDelay)
	assert.Equal(t, 10*time.Second, writer.maxRetryDelay)
	assert.Equal(t, 50, writer.batchSize)
}

func TestWriteRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	repo := &models.Repository{
		RepoID:     uuid.New().String(),
		Name:       "test-repo",
		URL:        "https://github.com/test/repo",
		Branch:     "main",
		CommitHash: "abc123",
		Metadata:   map[string]interface{}{"test": "value"},
	}

	err := writer.WriteRepository(ctx, repo)
	require.NoError(t, err)

	// Verify repository was created
	repoRepo := models.NewRepositoryRepository(db)
	retrieved, err := repoRepo.GetByID(ctx, repo.RepoID)
	require.NoError(t, err)
	assert.Equal(t, repo.Name, retrieved.Name)
	assert.Equal(t, repo.URL, retrieved.URL)
}

func TestWriteFiles(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	// Create test repository
	repoID := uuid.New().String()
	repo := &models.Repository{
		RepoID: repoID,
		Name:   "test-repo",
	}
	err := writer.WriteRepository(ctx, repo)
	require.NoError(t, err)

	// Create test files
	files := []schema.File{
		{
			FileID:   uuid.New().String(),
			Path:     "main.go",
			Language: "go",
			Size:     1024,
			Checksum: "checksum1",
		},
		{
			FileID:   uuid.New().String(),
			Path:     "utils.go",
			Language: "go",
			Size:     512,
			Checksum: "checksum2",
		},
	}

	result, err := writer.WriteFiles(ctx, repoID, files)
	require.NoError(t, err)
	assert.Equal(t, 2, result.FilesProcessed)
	assert.Empty(t, result.Errors)
	assert.Greater(t, result.Duration, time.Duration(0))

	// Verify files were created
	fileRepo := models.NewFileRepository(db)
	for _, file := range files {
		retrieved, err := fileRepo.GetByID(ctx, file.FileID)
		require.NoError(t, err)
		assert.Equal(t, file.Path, retrieved.Path)
		assert.Equal(t, file.Language, retrieved.Language)
		assert.Equal(t, file.Size, retrieved.Size)
		assert.Equal(t, file.Checksum, retrieved.Checksum)
	}
}

func TestWriteFilesWithTransaction(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	// Create test repository
	repoID := uuid.New().String()
	repo := &models.Repository{
		RepoID: repoID,
		Name:   "test-repo",
	}
	err := writer.WriteRepository(ctx, repo)
	require.NoError(t, err)

	// Create test files
	files := []schema.File{
		{
			FileID:   uuid.New().String(),
			Path:     "main.go",
			Language: "go",
			Size:     1024,
			Checksum: "checksum1",
		},
	}

	result, err := writer.WriteFilesWithTransaction(ctx, repoID, files)
	require.NoError(t, err)
	assert.Equal(t, 1, result.FilesProcessed)
	assert.Empty(t, result.Errors)

	// Verify file was created
	fileRepo := models.NewFileRepository(db)
	retrieved, err := fileRepo.GetByID(ctx, files[0].FileID)
	require.NoError(t, err)
	assert.Equal(t, files[0].Path, retrieved.Path)
}

func TestWriteSymbols(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	// Create test repository and file
	repoID := uuid.New().String()
	fileID := uuid.New().String()
	
	repo := &models.Repository{
		RepoID: repoID,
		Name:   "test-repo",
	}
	err := writer.WriteRepository(ctx, repo)
	require.NoError(t, err)

	files := []schema.File{
		{
			FileID:   fileID,
			Path:     "main.go",
			Language: "go",
			Size:     1024,
			Checksum: "checksum1",
		},
	}
	_, err = writer.WriteFiles(ctx, repoID, files)
	require.NoError(t, err)

	// Create test symbols
	symbols := []schema.Symbol{
		{
			SymbolID:        uuid.New().String(),
			FileID:          fileID,
			Name:            "main",
			Kind:            schema.SymbolFunction,
			Signature:       "func main()",
			Span:            schema.Span{StartLine: 1, EndLine: 10, StartByte: 0, EndByte: 100},
			Docstring:       "Main function",
			SemanticSummary: "Entry point of the application",
		},
		{
			SymbolID:  uuid.New().String(),
			FileID:    fileID,
			Name:      "Helper",
			Kind:      schema.SymbolClass,
			Signature: "type Helper struct",
			Span:      schema.Span{StartLine: 12, EndLine: 20, StartByte: 102, EndByte: 200},
		},
	}

	result, err := writer.WriteSymbols(ctx, symbols)
	require.NoError(t, err)
	assert.Equal(t, 2, result.SymbolsCreated)
	assert.Empty(t, result.Errors)

	// Verify symbols were created
	symbolRepo := models.NewSymbolRepository(db)
	for _, symbol := range symbols {
		retrieved, err := symbolRepo.GetByID(ctx, symbol.SymbolID)
		require.NoError(t, err)
		assert.Equal(t, symbol.Name, retrieved.Name)
		assert.Equal(t, string(symbol.Kind), retrieved.Kind)
		assert.Equal(t, symbol.Signature, retrieved.Signature)
		assert.Equal(t, symbol.Docstring, retrieved.Docstring)
	}
}

func TestWriteASTNodes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	// Create test repository and file
	repoID := uuid.New().String()
	fileID := uuid.New().String()
	
	repo := &models.Repository{
		RepoID: repoID,
		Name:   "test-repo",
	}
	err := writer.WriteRepository(ctx, repo)
	require.NoError(t, err)

	files := []schema.File{
		{
			FileID:   fileID,
			Path:     "main.go",
			Language: "go",
			Size:     1024,
			Checksum: "checksum1",
		},
	}
	_, err = writer.WriteFiles(ctx, repoID, files)
	require.NoError(t, err)

	// Create test AST nodes with parent-child relationships
	parentID := uuid.New().String()
	childID := uuid.New().String()
	
	nodes := []schema.ASTNode{
		{
			NodeID: childID, // Child node first to test sorting
			FileID: fileID,
			Type:   "identifier",
			Span:   schema.Span{StartLine: 2, EndLine: 2, StartByte: 10, EndByte: 20},
			ParentID: parentID,
			Text:     "main",
		},
		{
			NodeID: parentID, // Parent node second
			FileID: fileID,
			Type:   "function_declaration",
			Span:   schema.Span{StartLine: 1, EndLine: 10, StartByte: 0, EndByte: 100},
			Text:   "func main() {}",
			Attributes: map[string]string{
				"visibility": "public",
			},
		},
	}

	result, err := writer.WriteASTNodes(ctx, nodes)
	require.NoError(t, err)
	assert.Equal(t, 2, result.NodesCreated)
	assert.Empty(t, result.Errors)

	// Verify nodes were created and parent-child relationship is preserved
	astNodeRepo := models.NewASTNodeRepository(db)
	
	parentNode, err := astNodeRepo.GetByID(ctx, parentID)
	require.NoError(t, err)
	assert.Equal(t, "function_declaration", parentNode.Type)
	assert.Nil(t, parentNode.ParentID)
	
	childNode, err := astNodeRepo.GetByID(ctx, childID)
	require.NoError(t, err)
	assert.Equal(t, "identifier", childNode.Type)
	assert.NotNil(t, childNode.ParentID)
	assert.Equal(t, parentID, *childNode.ParentID)
}

func TestWriteEdges(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	// Create test repository, file, and symbols
	repoID := uuid.New().String()
	fileID := uuid.New().String()
	sourceSymbolID := uuid.New().String()
	targetSymbolID := uuid.New().String()
	
	repo := &models.Repository{
		RepoID: repoID,
		Name:   "test-repo",
	}
	err := writer.WriteRepository(ctx, repo)
	require.NoError(t, err)

	files := []schema.File{
		{
			FileID:   fileID,
			Path:     "main.go",
			Language: "go",
			Size:     1024,
			Checksum: "checksum1",
		},
	}
	_, err = writer.WriteFiles(ctx, repoID, files)
	require.NoError(t, err)

	symbols := []schema.Symbol{
		{
			SymbolID:  sourceSymbolID,
			FileID:    fileID,
			Name:      "main",
			Kind:      schema.SymbolFunction,
			Signature: "func main()",
			Span:      schema.Span{StartLine: 1, EndLine: 10, StartByte: 0, EndByte: 100},
		},
		{
			SymbolID:  targetSymbolID,
			FileID:    fileID,
			Name:      "helper",
			Kind:      schema.SymbolFunction,
			Signature: "func helper()",
			Span:      schema.Span{StartLine: 12, EndLine: 20, StartByte: 102, EndByte: 200},
		},
	}
	_, err = writer.WriteSymbols(ctx, symbols)
	require.NoError(t, err)

	// Create test edges
	edges := []schema.DependencyEdge{
		{
			EdgeID:     uuid.New().String(),
			SourceID:   sourceSymbolID,
			TargetID:   targetSymbolID,
			EdgeType:   schema.EdgeCall,
			SourceFile: "main.go",
			TargetFile: "main.go",
		},
		{
			EdgeID:       uuid.New().String(),
			SourceID:     sourceSymbolID,
			TargetID:     "", // External dependency
			EdgeType:     schema.EdgeImport,
			SourceFile:   "main.go",
			TargetModule: "fmt",
		},
	}

	result, err := writer.WriteEdges(ctx, edges)
	require.NoError(t, err)
	assert.Equal(t, 2, result.EdgesCreated)
	assert.Empty(t, result.Errors)

	// Verify edges were created
	edgeRepo := models.NewEdgeRepository(db)
	for _, edge := range edges {
		retrieved, err := edgeRepo.GetByID(ctx, edge.EdgeID)
		require.NoError(t, err)
		assert.Equal(t, edge.SourceID, retrieved.SourceID)
		assert.Equal(t, string(edge.EdgeType), retrieved.EdgeType)
		assert.Equal(t, edge.SourceFile, retrieved.SourceFile)
		
		if edge.TargetID != "" {
			assert.NotNil(t, retrieved.TargetID)
			assert.Equal(t, edge.TargetID, *retrieved.TargetID)
		} else {
			assert.Nil(t, retrieved.TargetID)
		}
	}
}

func TestTopologicalSortNodes(t *testing.T) {
	writer := NewWriter(nil, nil)

	// Create nodes with parent-child relationships
	parentID := "parent"
	childID := "child"
	grandchildID := "grandchild"
	
	nodes := []*models.ASTNode{
		{
			NodeID:   grandchildID,
			ParentID: &childID, // Grandchild first
		},
		{
			NodeID:   childID,
			ParentID: &parentID, // Child second
		},
		{
			NodeID: parentID, // Parent last
		},
	}

	sorted := writer.topologicalSortNodes(nodes)
	
	// Verify correct order: parent -> child -> grandchild
	assert.Len(t, sorted, 3)
	assert.Equal(t, parentID, sorted[0].NodeID)
	assert.Equal(t, childID, sorted[1].NodeID)
	assert.Equal(t, grandchildID, sorted[2].NodeID)
}

func TestIsRetryableError(t *testing.T) {
	writer := NewWriter(nil, nil)

	// Test retryable errors
	retryableErrors := []error{
		&mockError{msg: "connection refused"},
		&mockError{msg: "connection timeout"},
		&mockError{msg: "temporary failure"},
		&mockError{msg: "deadlock detected"},
		&mockError{msg: "pq: sorry, too many clients"},
		&mockError{msg: "connection reset"},
	}

	for _, err := range retryableErrors {
		assert.True(t, writer.isRetryableError(err), "Expected %v to be retryable", err)
	}

	// Test non-retryable errors
	nonRetryableErrors := []error{
		&mockError{msg: "syntax error"},
		&mockError{msg: "foreign key constraint"},
		&mockError{msg: "unique constraint violation"},
	}

	for _, err := range nonRetryableErrors {
		assert.False(t, writer.isRetryableError(err), "Expected %v to not be retryable", err)
	}
}

func TestWriteFilesEmptyInput(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	result, err := writer.WriteFiles(ctx, "repo-id", []schema.File{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.FilesProcessed)
	assert.Empty(t, result.Errors)
}

func TestWriteSymbolsEmptyInput(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	result, err := writer.WriteSymbols(ctx, []schema.Symbol{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.SymbolsCreated)
	assert.Empty(t, result.Errors)
}

func TestWriteASTNodesEmptyInput(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	result, err := writer.WriteASTNodes(ctx, []schema.ASTNode{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.NodesCreated)
	assert.Empty(t, result.Errors)
}

func TestWriteEdgesEmptyInput(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	writer := NewWriter(db, nil)
	ctx := context.Background()

	result, err := writer.WriteEdges(ctx, []schema.DependencyEdge{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.EdgesCreated)
	assert.Empty(t, result.Errors)
}

// Mock error for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*models.DB, func()) {
	// This would typically connect to a test database
	// For now, we'll skip actual database tests and focus on unit logic
	t.Skip("Database tests require test database setup")
	return nil, func() {}
}