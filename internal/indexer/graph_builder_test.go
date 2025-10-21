package indexer

import (
	"context"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func TestNewGraphBuilder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := models.NewDB()
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name   string
		config *GraphBuilderConfig
		want   *GraphBuilder
	}{
		{
			name:   "with default config",
			config: nil,
			want: &GraphBuilder{
				db:        db,
				graphName: "code_graph",
				batchSize: 100,
			},
		},
		{
			name: "with custom config",
			config: &GraphBuilderConfig{
				GraphName: "test_graph",
				BatchSize: 50,
			},
			want: &GraphBuilder{
				db:        db,
				graphName: "test_graph",
				batchSize: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGraphBuilder(db, tt.config)
			if got.graphName != tt.want.graphName {
				t.Errorf("NewGraphBuilder() graphName = %v, want %v", got.graphName, tt.want.graphName)
			}
			if got.batchSize != tt.want.batchSize {
				t.Errorf("NewGraphBuilder() batchSize = %v, want %v", got.batchSize, tt.want.batchSize)
			}
		})
	}
}

func TestGraphBuilder_InitGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := models.NewDB()
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer db.Close()

	gb := NewGraphBuilder(db, &GraphBuilderConfig{
		GraphName: "test_init_graph",
		BatchSize: 100,
	})

	ctx := context.Background()

	// Test graph initialization
	err = gb.InitGraph(ctx)
	if err != nil {
		t.Fatalf("InitGraph() error = %v", err)
	}

	// Test idempotency - should not error on second call
	err = gb.InitGraph(ctx)
	if err != nil {
		t.Errorf("InitGraph() second call error = %v", err)
	}
}

func TestGraphBuilder_CreateNodes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := models.NewDB()
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer db.Close()

	gb := NewGraphBuilder(db, &GraphBuilderConfig{
		GraphName: "test_create_nodes",
		BatchSize: 2,
	})

	ctx := context.Background()

	// Initialize graph
	err = gb.InitGraph(ctx)
	if err != nil {
		t.Fatalf("InitGraph() error = %v", err)
	}

	tests := []struct {
		name    string
		symbols []schema.Symbol
		wantErr bool
	}{
		{
			name:    "empty symbols",
			symbols: []schema.Symbol{},
			wantErr: false,
		},
		{
			name: "single function symbol",
			symbols: []schema.Symbol{
				{
					SymbolID:  "func-001",
					FileID:    "file-001",
					Name:      "testFunction",
					Kind:      schema.SymbolFunction,
					Signature: "func testFunction() error",
					Span: schema.Span{
						StartLine: 10,
						EndLine:   20,
						StartByte: 100,
						EndByte:   200,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple symbols with different kinds",
			symbols: []schema.Symbol{
				{
					SymbolID:  "class-001",
					FileID:    "file-001",
					Name:      "TestClass",
					Kind:      schema.SymbolClass,
					Signature: "class TestClass",
					Span: schema.Span{
						StartLine: 1,
						EndLine:   50,
						StartByte: 0,
						EndByte:   500,
					},
				},
				{
					SymbolID:  "interface-001",
					FileID:    "file-001",
					Name:      "TestInterface",
					Kind:      schema.SymbolInterface,
					Signature: "interface TestInterface",
					Span: schema.Span{
						StartLine: 60,
						EndLine:   70,
						StartByte: 600,
						EndByte:   700,
					},
				},
				{
					SymbolID:  "var-001",
					FileID:    "file-001",
					Name:      "testVariable",
					Kind:      schema.SymbolVariable,
					Signature: "var testVariable string",
					Span: schema.Span{
						StartLine: 80,
						EndLine:   80,
						StartByte: 800,
						EndByte:   850,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gb.CreateNodes(ctx, tt.symbols)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.NodesCreated != len(tt.symbols) {
					t.Errorf("CreateNodes() created %d nodes, want %d", result.NodesCreated, len(tt.symbols))
				}
				if len(result.Errors) > 0 {
					t.Errorf("CreateNodes() had %d errors: %v", len(result.Errors), result.Errors)
				}
			}
		})
	}
}

func TestGraphBuilder_CreateEdges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := models.NewDB()
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer db.Close()

	gb := NewGraphBuilder(db, &GraphBuilderConfig{
		GraphName: "test_create_edges",
		BatchSize: 2,
	})

	ctx := context.Background()

	// Initialize graph
	err = gb.InitGraph(ctx)
	if err != nil {
		t.Fatalf("InitGraph() error = %v", err)
	}

	// Create test nodes first
	symbols := []schema.Symbol{
		{
			SymbolID:  "func-source",
			FileID:    "file-001",
			Name:      "sourceFunction",
			Kind:      schema.SymbolFunction,
			Signature: "func sourceFunction()",
			Span:      schema.Span{StartLine: 1, EndLine: 10, StartByte: 0, EndByte: 100},
		},
		{
			SymbolID:  "func-target",
			FileID:    "file-001",
			Name:      "targetFunction",
			Kind:      schema.SymbolFunction,
			Signature: "func targetFunction()",
			Span:      schema.Span{StartLine: 20, EndLine: 30, StartByte: 200, EndByte: 300},
		},
	}

	_, err = gb.CreateNodes(ctx, symbols)
	if err != nil {
		t.Fatalf("CreateNodes() error = %v", err)
	}

	tests := []struct {
		name    string
		edges   []schema.DependencyEdge
		wantErr bool
	}{
		{
			name:    "empty edges",
			edges:   []schema.DependencyEdge{},
			wantErr: false,
		},
		{
			name: "single call edge",
			edges: []schema.DependencyEdge{
				{
					EdgeID:     "edge-001",
					SourceID:   "func-source",
					TargetID:   "func-target",
					EdgeType:   schema.EdgeCall,
					SourceFile: "test.go",
					TargetFile: "test.go",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple edges with different types",
			edges: []schema.DependencyEdge{
				{
					EdgeID:     "edge-002",
					SourceID:   "func-source",
					TargetID:   "func-target",
					EdgeType:   schema.EdgeCall,
					SourceFile: "test.go",
					TargetFile: "test.go",
				},
				{
					EdgeID:     "edge-003",
					SourceID:   "func-source",
					TargetID:   "func-target",
					EdgeType:   schema.EdgeReference,
					SourceFile: "test.go",
					TargetFile: "test.go",
				},
			},
			wantErr: false,
		},
		{
			name: "edge without target ID (external dependency - skipped)",
			edges: []schema.DependencyEdge{
				{
					EdgeID:       "edge-004",
					SourceID:     "func-source",
					TargetID:     "",
					EdgeType:     schema.EdgeImport,
					SourceFile:   "test.go",
					TargetModule: "external/package",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gb.CreateEdges(ctx, tt.edges)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateEdges() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Count edges with target ID (external dependencies are skipped)
				expectedEdges := 0
				for _, edge := range tt.edges {
					if edge.TargetID != "" {
						expectedEdges++
					}
				}

				if result.EdgesCreated != expectedEdges {
					t.Errorf("CreateEdges() created %d edges, want %d", result.EdgesCreated, expectedEdges)
				}
				if len(result.Errors) > 0 {
					t.Errorf("CreateEdges() had %d errors: %v", len(result.Errors), result.Errors)
				}
			}
		})
	}
}

func TestGraphBuilder_UpdateNodeProperties(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := models.NewDB()
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer db.Close()

	gb := NewGraphBuilder(db, &GraphBuilderConfig{
		GraphName: "test_update_props",
		BatchSize: 100,
	})

	ctx := context.Background()

	// Initialize graph and create a test node
	err = gb.InitGraph(ctx)
	if err != nil {
		t.Fatalf("InitGraph() error = %v", err)
	}

	symbol := schema.Symbol{
		SymbolID:  "func-update",
		FileID:    "file-001",
		Name:      "updateFunction",
		Kind:      schema.SymbolFunction,
		Signature: "func updateFunction()",
		Span:      schema.Span{StartLine: 1, EndLine: 10, StartByte: 0, EndByte: 100},
	}

	_, err = gb.CreateNodes(ctx, []schema.Symbol{symbol})
	if err != nil {
		t.Fatalf("CreateNodes() error = %v", err)
	}

	tests := []struct {
		name     string
		symbolID string
		props    map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "update string property",
			symbolID: "func-update",
			props: map[string]interface{}{
				"description": "Updated description",
			},
			wantErr: false,
		},
		{
			name:     "update multiple properties",
			symbolID: "func-update",
			props: map[string]interface{}{
				"version":  "2.0",
				"modified": true,
			},
			wantErr: false,
		},
		{
			name:     "update non-existent node",
			symbolID: "non-existent",
			props: map[string]interface{}{
				"test": "value",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gb.UpdateNodeProperties(ctx, tt.symbolID, tt.props)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateNodeProperties() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGraphBuilder_MapSymbolKindToLabel(t *testing.T) {
	gb := &GraphBuilder{}

	tests := []struct {
		name string
		kind schema.SymbolKind
		want string
	}{
		{
			name: "function kind",
			kind: schema.SymbolFunction,
			want: "Function",
		},
		{
			name: "class kind",
			kind: schema.SymbolClass,
			want: "Class",
		},
		{
			name: "interface kind",
			kind: schema.SymbolInterface,
			want: "Interface",
		},
		{
			name: "variable kind",
			kind: schema.SymbolVariable,
			want: "Variable",
		},
		{
			name: "module kind",
			kind: schema.SymbolModule,
			want: "Module",
		},
		{
			name: "package kind",
			kind: schema.SymbolPackage,
			want: "Module",
		},
		{
			name: "unknown kind",
			kind: schema.SymbolKind("unknown"),
			want: "Symbol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gb.mapSymbolKindToLabel(tt.kind)
			if got != tt.want {
				t.Errorf("mapSymbolKindToLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGraphBuilder_MapEdgeTypeToRelationship(t *testing.T) {
	gb := &GraphBuilder{}

	tests := []struct {
		name     string
		edgeType schema.EdgeType
		want     string
	}{
		{
			name:     "call edge",
			edgeType: schema.EdgeCall,
			want:     "CALLS",
		},
		{
			name:     "import edge",
			edgeType: schema.EdgeImport,
			want:     "IMPORTS",
		},
		{
			name:     "extends edge",
			edgeType: schema.EdgeExtends,
			want:     "EXTENDS",
		},
		{
			name:     "implements edge",
			edgeType: schema.EdgeImplements,
			want:     "IMPLEMENTS",
		},
		{
			name:     "reference edge",
			edgeType: schema.EdgeReference,
			want:     "REFERENCES",
		},
		{
			name:     "unknown edge",
			edgeType: schema.EdgeType("unknown"),
			want:     "RELATES_TO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gb.mapEdgeTypeToRelationship(tt.edgeType)
			if got != tt.want {
				t.Errorf("mapEdgeTypeToRelationship() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeCypherString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no special characters",
			input: "simple string",
			want:  "simple string",
		},
		{
			name:  "single quote",
			input: "it's a test",
			want:  "it\\'s a test",
		},
		{
			name:  "multiple single quotes",
			input: "it's a 'test' string",
			want:  "it\\'s a \\'test\\' string",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeCypherString(tt.input)
			if got != tt.want {
				t.Errorf("escapeCypherString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGraphBuilder_DeleteNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := models.NewDB()
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer db.Close()

	gb := NewGraphBuilder(db, &GraphBuilderConfig{
		GraphName: "test_delete_node",
		BatchSize: 100,
	})

	ctx := context.Background()

	// Initialize graph and create a test node
	err = gb.InitGraph(ctx)
	if err != nil {
		t.Fatalf("InitGraph() error = %v", err)
	}

	symbol := schema.Symbol{
		SymbolID:  "func-delete",
		FileID:    "file-001",
		Name:      "deleteFunction",
		Kind:      schema.SymbolFunction,
		Signature: "func deleteFunction()",
		Span:      schema.Span{StartLine: 1, EndLine: 10, StartByte: 0, EndByte: 100},
	}

	_, err = gb.CreateNodes(ctx, []schema.Symbol{symbol})
	if err != nil {
		t.Fatalf("CreateNodes() error = %v", err)
	}

	// Test deletion
	err = gb.DeleteNode(ctx, "func-delete")
	if err != nil {
		t.Errorf("DeleteNode() error = %v", err)
	}

	// Verify node is deleted
	node, err := gb.GetNodeBySymbolID(ctx, "func-delete")
	if err != nil {
		t.Errorf("GetNodeBySymbolID() error = %v", err)
	}
	if node != nil {
		t.Errorf("DeleteNode() did not delete the node")
	}
}

func TestGraphBuilder_DeleteEdge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := models.NewDB()
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer db.Close()

	gb := NewGraphBuilder(db, &GraphBuilderConfig{
		GraphName: "test_delete_edge",
		BatchSize: 100,
	})

	ctx := context.Background()

	// Initialize graph
	err = gb.InitGraph(ctx)
	if err != nil {
		t.Fatalf("InitGraph() error = %v", err)
	}

	// Create test nodes
	symbols := []schema.Symbol{
		{
			SymbolID:  "func-del-source",
			FileID:    "file-001",
			Name:      "sourceFunction",
			Kind:      schema.SymbolFunction,
			Signature: "func sourceFunction()",
			Span:      schema.Span{StartLine: 1, EndLine: 10, StartByte: 0, EndByte: 100},
		},
		{
			SymbolID:  "func-del-target",
			FileID:    "file-001",
			Name:      "targetFunction",
			Kind:      schema.SymbolFunction,
			Signature: "func targetFunction()",
			Span:      schema.Span{StartLine: 20, EndLine: 30, StartByte: 200, EndByte: 300},
		},
	}

	_, err = gb.CreateNodes(ctx, symbols)
	if err != nil {
		t.Fatalf("CreateNodes() error = %v", err)
	}

	// Create test edge
	edges := []schema.DependencyEdge{
		{
			EdgeID:     "edge-delete",
			SourceID:   "func-del-source",
			TargetID:   "func-del-target",
			EdgeType:   schema.EdgeCall,
			SourceFile: "test.go",
			TargetFile: "test.go",
		},
	}

	_, err = gb.CreateEdges(ctx, edges)
	if err != nil {
		t.Fatalf("CreateEdges() error = %v", err)
	}

	// Test deletion
	err = gb.DeleteEdge(ctx, "edge-delete")
	if err != nil {
		t.Errorf("DeleteEdge() error = %v", err)
	}
}
