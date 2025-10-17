package indexer_test

import (
	"context"
	"fmt"
	"log"

	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// ExampleGraphBuilder demonstrates basic usage of the GraphBuilder
func ExampleGraphBuilder() {
	// Connect to database
	db, err := models.NewDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create graph builder with default config
	gb := indexer.NewGraphBuilder(db, nil)

	ctx := context.Background()

	// Initialize the graph
	err = gb.InitGraph(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize graph: %v", err)
	}

	// Create some test symbols
	symbols := []schema.Symbol{
		{
			SymbolID:  "func-001",
			FileID:    "file-001",
			Name:      "calculateSum",
			Kind:      schema.SymbolFunction,
			Signature: "func calculateSum(a, b int) int",
			Span: schema.Span{
				StartLine: 10,
				EndLine:   15,
				StartByte: 100,
				EndByte:   200,
			},
		},
		{
			SymbolID:  "func-002",
			FileID:    "file-001",
			Name:      "processData",
			Kind:      schema.SymbolFunction,
			Signature: "func processData(data []int) int",
			Span: schema.Span{
				StartLine: 20,
				EndLine:   30,
				StartByte: 250,
				EndByte:   400,
			},
		},
	}

	// Create nodes in the graph
	result, err := gb.CreateNodes(ctx, symbols)
	if err != nil {
		log.Fatalf("Failed to create nodes: %v", err)
	}

	fmt.Printf("Created %d nodes\n", result.NodesCreated)

	// Create edges between symbols
	edges := []schema.DependencyEdge{
		{
			EdgeID:     "edge-001",
			SourceID:   "func-002",
			TargetID:   "func-001",
			EdgeType:   schema.EdgeCall,
			SourceFile: "main.go",
			TargetFile: "main.go",
		},
	}

	edgeResult, err := gb.CreateEdges(ctx, edges)
	if err != nil {
		log.Fatalf("Failed to create edges: %v", err)
	}

	fmt.Printf("Created %d edges\n", edgeResult.EdgesCreated)

	// Output:
	// Created 2 nodes
	// Created 1 edges
}

// ExampleGraphBuilder_customConfig demonstrates using custom configuration
func ExampleGraphBuilder_customConfig() {
	db, err := models.NewDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create graph builder with custom config
	config := &indexer.GraphBuilderConfig{
		GraphName: "my_custom_graph",
		BatchSize: 50,
	}
	gb := indexer.NewGraphBuilder(db, config)

	ctx := context.Background()

	// Initialize the custom graph
	err = gb.InitGraph(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize graph: %v", err)
	}

	fmt.Println("Custom graph initialized successfully")

	// Output:
	// Custom graph initialized successfully
}

// ExampleGraphBuilder_updateProperties demonstrates updating node properties
func ExampleGraphBuilder_updateProperties() {
	db, err := models.NewDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	gb := indexer.NewGraphBuilder(db, nil)
	ctx := context.Background()

	// Initialize graph
	err = gb.InitGraph(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize graph: %v", err)
	}

	// Create a test symbol
	symbol := schema.Symbol{
		SymbolID:  "func-update-example",
		FileID:    "file-001",
		Name:      "exampleFunction",
		Kind:      schema.SymbolFunction,
		Signature: "func exampleFunction()",
		Span: schema.Span{
			StartLine: 1,
			EndLine:   10,
			StartByte: 0,
			EndByte:   100,
		},
	}

	_, err = gb.CreateNodes(ctx, []schema.Symbol{symbol})
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	// Update node properties
	props := map[string]interface{}{
		"version":     "2.0",
		"description": "Updated function",
	}

	err = gb.UpdateNodeProperties(ctx, "func-update-example", props)
	if err != nil {
		log.Fatalf("Failed to update properties: %v", err)
	}

	fmt.Println("Node properties updated successfully")

	// Output:
	// Node properties updated successfully
}
