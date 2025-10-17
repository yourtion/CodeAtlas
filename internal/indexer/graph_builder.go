package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// GraphBuilder handles AGE graph construction operations
type GraphBuilder struct {
	db        *models.DB
	graphName string
	batchSize int
}

// GraphBuilderConfig contains configuration options for the GraphBuilder
type GraphBuilderConfig struct {
	GraphName string `json:"graph_name"`
	BatchSize int    `json:"batch_size"`
}

// DefaultGraphBuilderConfig returns default configuration for the GraphBuilder
func DefaultGraphBuilderConfig() *GraphBuilderConfig {
	return &GraphBuilderConfig{
		GraphName: "code_graph",
		BatchSize: 100,
	}
}

// NewGraphBuilder creates a new graph builder instance
func NewGraphBuilder(db *models.DB, config *GraphBuilderConfig) *GraphBuilder {
	if config == nil {
		config = DefaultGraphBuilderConfig()
	}

	return &GraphBuilder{
		db:        db,
		graphName: config.GraphName,
		batchSize: config.BatchSize,
	}
}

// GraphBuildResult contains the results of a graph build operation
type GraphBuildResult struct {
	NodesCreated int           `json:"nodes_created"`
	EdgesCreated int           `json:"edges_created"`
	Duration     time.Duration `json:"duration"`
	Errors       []GraphError  `json:"errors,omitempty"`
}

// GraphError represents an error that occurred during graph building
type GraphError struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Message    string `json:"message"`
}

// InitGraph creates the AGE graph schema if it doesn't exist
func (gb *GraphBuilder) InitGraph(ctx context.Context) error {
	// Set search path to include ag_catalog
	_, err := gb.db.ExecContext(ctx, "LOAD 'age'")
	if err != nil {
		return NewGraphError("failed to load AGE extension", "", "", err)
	}

	_, err = gb.db.ExecContext(ctx, "SET search_path = ag_catalog, \"$user\", public")
	if err != nil {
		return NewGraphError("failed to set search path", "", "", err)
	}

	// Check if graph exists
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM ag_catalog.ag_graph WHERE name = $1)`
	err = gb.db.QueryRowContext(ctx, query, gb.graphName).Scan(&exists)
	if err != nil {
		return NewGraphError("failed to check graph existence", "", "", err)
	}

	// Create graph if it doesn't exist
	if !exists {
		createQuery := fmt.Sprintf("SELECT * FROM ag_catalog.create_graph('%s')", gb.graphName)
		_, err = gb.db.ExecContext(ctx, createQuery)
		if err != nil {
			return NewGraphError("failed to create graph", "", "", err)
		}
	}

	return nil
}

// CreateNodes creates graph vertices for symbols
func (gb *GraphBuilder) CreateNodes(ctx context.Context, symbols []schema.Symbol) (*GraphBuildResult, error) {
	startTime := time.Now()
	result := &GraphBuildResult{}

	if len(symbols) == 0 {
		return result, nil
	}

	// Ensure graph is initialized
	if err := gb.InitGraph(ctx); err != nil {
		return result, err
	}

	// Process symbols in batches
	for i := 0; i < len(symbols); i += gb.batchSize {
		end := i + gb.batchSize
		if end > len(symbols) {
			end = len(symbols)
		}

		batch := symbols[i:end]
		created, errors := gb.createNodeBatch(ctx, batch)
		result.NodesCreated += created
		result.Errors = append(result.Errors, errors...)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// createNodeBatch creates a batch of graph nodes
func (gb *GraphBuilder) createNodeBatch(ctx context.Context, symbols []schema.Symbol) (int, []GraphError) {
	var errors []GraphError
	created := 0

	for _, symbol := range symbols {
		err := gb.createNode(ctx, symbol)
		if err != nil {
			errors = append(errors, GraphError{
				EntityType: "node",
				EntityID:   symbol.SymbolID,
				Message:    err.Error(),
			})
		} else {
			created++
		}
	}

	return created, errors
}

// createNode creates a single graph node for a symbol
func (gb *GraphBuilder) createNode(ctx context.Context, symbol schema.Symbol) error {
	// Map symbol kind to graph label
	label := gb.mapSymbolKindToLabel(symbol.Kind)

	// Escape single quotes in properties
	name := escapeCypherString(symbol.Name)
	signature := escapeCypherString(symbol.Signature)
	filePath := escapeCypherString(getFilePathFromSymbol(symbol))

	// Build Cypher query to create node
	// AGE uses MERGE with SET for upsert behavior
	query := fmt.Sprintf(`
		SELECT * FROM cypher('%s', $$
			MERGE (n:%s {symbol_id: '%s'})
			SET n.name = '%s',
				n.signature = '%s',
				n.file_path = '%s',
				n.start_line = %d,
				n.end_line = %d,
				n.start_byte = %d,
				n.end_byte = %d
			RETURN n
		$$) as (n agtype)
	`, gb.graphName, label, symbol.SymbolID,
		name, signature, filePath,
		symbol.Span.StartLine, symbol.Span.EndLine,
		symbol.Span.StartByte, symbol.Span.EndByte)

	_, err := gb.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create node for symbol %s: %w", symbol.SymbolID, err)
	}

	return nil
}

// CreateEdges creates graph edges for relationships
func (gb *GraphBuilder) CreateEdges(ctx context.Context, edges []schema.DependencyEdge) (*GraphBuildResult, error) {
	startTime := time.Now()
	result := &GraphBuildResult{}

	if len(edges) == 0 {
		return result, nil
	}

	// Ensure graph is initialized
	if err := gb.InitGraph(ctx); err != nil {
		return result, err
	}

	// Process edges in batches
	for i := 0; i < len(edges); i += gb.batchSize {
		end := i + gb.batchSize
		if end > len(edges) {
			end = len(edges)
		}

		batch := edges[i:end]
		created, errors := gb.createEdgeBatch(ctx, batch)
		result.EdgesCreated += created
		result.Errors = append(result.Errors, errors...)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// createEdgeBatch creates a batch of graph edges
func (gb *GraphBuilder) createEdgeBatch(ctx context.Context, edges []schema.DependencyEdge) (int, []GraphError) {
	var errors []GraphError
	created := 0

	for _, edge := range edges {
		// Skip edges without target ID (external dependencies)
		if edge.TargetID == "" {
			continue
		}

		err := gb.createEdge(ctx, edge)
		if err != nil {
			errors = append(errors, GraphError{
				EntityType: "edge",
				EntityID:   edge.EdgeID,
				Message:    err.Error(),
			})
		} else {
			created++
		}
	}

	return created, errors
}

// createEdge creates a single graph edge for a relationship
func (gb *GraphBuilder) createEdge(ctx context.Context, edge schema.DependencyEdge) error {
	// Skip edges without target ID (external dependencies)
	if edge.TargetID == "" {
		return nil
	}

	// Map edge type to relationship type
	relType := gb.mapEdgeTypeToRelationship(edge.EdgeType)

	// Escape single quotes in properties
	sourceFile := escapeCypherString(edge.SourceFile)
	targetFile := escapeCypherString(edge.TargetFile)

	// Build Cypher query to create edge
	// AGE uses MERGE with SET for upsert behavior
	query := fmt.Sprintf(`
		SELECT * FROM cypher('%s', $$
			MATCH (source {symbol_id: '%s'})
			MATCH (target {symbol_id: '%s'})
			MERGE (source)-[r:%s {edge_id: '%s'}]->(target)
			SET r.source_file = '%s',
				r.target_file = '%s'
			RETURN r
		$$) as (r agtype)
	`, gb.graphName, edge.SourceID, edge.TargetID, relType,
		edge.EdgeID, sourceFile, targetFile)

	_, err := gb.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create edge %s: %w", edge.EdgeID, err)
	}

	return nil
}

// UpdateNodeProperties updates properties on an existing node
func (gb *GraphBuilder) UpdateNodeProperties(ctx context.Context, symbolID string, props map[string]interface{}) error {
	// Ensure graph is initialized
	if err := gb.InitGraph(ctx); err != nil {
		return err
	}

	// Build SET clause from properties
	var setClauses []string
	for key, value := range props {
		switch v := value.(type) {
		case string:
			setClauses = append(setClauses, fmt.Sprintf("n.%s = '%s'", key, escapeCypherString(v)))
		case int:
			setClauses = append(setClauses, fmt.Sprintf("n.%s = %d", key, v))
		case bool:
			setClauses = append(setClauses, fmt.Sprintf("n.%s = %t", key, v))
		}
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClause := strings.Join(setClauses, ", ")

	// Build Cypher query to update node properties
	query := fmt.Sprintf(`
		SELECT * FROM cypher('%s', $$
			MATCH (n {symbol_id: '%s'})
			SET %s
			RETURN n
		$$) as (n agtype)
	`, gb.graphName, symbolID, setClause)

	result, err := gb.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to update node properties for symbol %s: %w", symbolID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("node not found for symbol %s", symbolID)
	}

	return nil
}

// DeleteNode removes a node from the graph
func (gb *GraphBuilder) DeleteNode(ctx context.Context, symbolID string) error {
	// Ensure graph is initialized
	if err := gb.InitGraph(ctx); err != nil {
		return err
	}

	// Build Cypher query to delete node and its relationships
	query := fmt.Sprintf(`
		SELECT * FROM cypher('%s', $$
			MATCH (n {symbol_id: '%s'})
			DETACH DELETE n
		$$) as (result agtype)
	`, gb.graphName, symbolID)

	_, err := gb.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete node for symbol %s: %w", symbolID, err)
	}

	return nil
}

// DeleteEdge removes an edge from the graph
func (gb *GraphBuilder) DeleteEdge(ctx context.Context, edgeID string) error {
	// Ensure graph is initialized
	if err := gb.InitGraph(ctx); err != nil {
		return err
	}

	// Build Cypher query to delete edge
	query := fmt.Sprintf(`
		SELECT * FROM cypher('%s', $$
			MATCH ()-[r {edge_id: '%s'}]->()
			DELETE r
		$$) as (result agtype)
	`, gb.graphName, edgeID)

	_, err := gb.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete edge %s: %w", edgeID, err)
	}

	return nil
}

// GetNodeBySymbolID retrieves a node by symbol ID
func (gb *GraphBuilder) GetNodeBySymbolID(ctx context.Context, symbolID string) (map[string]interface{}, error) {
	// Ensure graph is initialized
	if err := gb.InitGraph(ctx); err != nil {
		return nil, err
	}

	// Build Cypher query to get node
	query := fmt.Sprintf(`
		SELECT * FROM cypher('%s', $$
			MATCH (n {symbol_id: '%s'})
			RETURN n
		$$) as (n agtype)
	`, gb.graphName, symbolID)

	var nodeData sql.NullString
	err := gb.db.QueryRowContext(ctx, query).Scan(&nodeData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get node for symbol %s: %w", symbolID, err)
	}

	// Parse the agtype result (simplified - in production, use proper agtype parsing)
	result := make(map[string]interface{})
	if nodeData.Valid {
		result["data"] = nodeData.String
	}

	return result, nil
}

// mapSymbolKindToLabel maps schema symbol kinds to AGE graph labels
func (gb *GraphBuilder) mapSymbolKindToLabel(kind schema.SymbolKind) string {
	switch kind {
	case schema.SymbolFunction:
		return "Function"
	case schema.SymbolClass:
		return "Class"
	case schema.SymbolInterface:
		return "Interface"
	case schema.SymbolVariable:
		return "Variable"
	case schema.SymbolModule, schema.SymbolPackage:
		return "Module"
	default:
		return "Symbol"
	}
}

// mapEdgeTypeToRelationship maps schema edge types to AGE relationship types
func (gb *GraphBuilder) mapEdgeTypeToRelationship(edgeType schema.EdgeType) string {
	switch edgeType {
	case schema.EdgeCall:
		return "CALLS"
	case schema.EdgeImport:
		return "IMPORTS"
	case schema.EdgeExtends:
		return "EXTENDS"
	case schema.EdgeImplements:
		return "IMPLEMENTS"
	case schema.EdgeReference:
		return "REFERENCES"
	default:
		return "RELATES_TO"
	}
}

// escapeCypherString escapes single quotes in Cypher strings
func escapeCypherString(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}

// getFilePathFromSymbol extracts file path from symbol (placeholder - needs actual implementation)
func getFilePathFromSymbol(symbol schema.Symbol) string {
	// In a real implementation, this would look up the file path from the file_id
	// For now, return empty string as file_path will be set separately
	return ""
}
