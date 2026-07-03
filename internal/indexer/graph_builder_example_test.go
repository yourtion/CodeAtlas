package indexer_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// skipIfNoDB 在 short 模式、SKIP_DB_TESTS=1 或连不上数据库时跳过测试。
// GraphBuilder 的集成测试依赖真实 AGE 图数据库，无法离线运行。
func skipIfNoDB(t *testing.T) *models.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("SKIP_DB_TESTS") == "1" {
		t.Skip("Skipping database tests (SKIP_DB_TESTS=1)")
	}
	db, err := models.NewDB()
	if err != nil {
		t.Skipf("Failed to connect to test database: %v (set DB_* env vars or SKIP_DB_TESTS=1)", err)
	}
	return db
}

// TestGraphBuilderExample 覆盖 GraphBuilder 的基本用法：初始化图、创建节点、创建边。
// 原 ExampleGraphBuilder 用 log.Fatalf 处理连库失败，会在无 DB 环境让整个测试进程崩溃。
func TestGraphBuilderExample(t *testing.T) {
	db := skipIfNoDB(t)
	defer db.Close()

	gb := indexer.NewGraphBuilder(db, nil)
	ctx := context.Background()

	if err := gb.InitGraph(ctx); err != nil {
		t.Fatalf("Failed to initialize graph: %v", err)
	}

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

	result, err := gb.CreateNodes(ctx, symbols)
	if err != nil {
		t.Fatalf("Failed to create nodes: %v", err)
	}
	fmt.Printf("Created %d nodes\n", result.NodesCreated)

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
		t.Fatalf("Failed to create edges: %v", err)
	}
	fmt.Printf("Created %d edges\n", edgeResult.EdgesCreated)
}

// TestGraphBuilderCustomConfig 覆盖使用自定义配置创建 GraphBuilder。
func TestGraphBuilderCustomConfig(t *testing.T) {
	db := skipIfNoDB(t)
	defer db.Close()

	config := &indexer.GraphBuilderConfig{
		GraphName: "my_custom_graph",
		BatchSize: 50,
	}
	gb := indexer.NewGraphBuilder(db, config)

	ctx := context.Background()
	if err := gb.InitGraph(ctx); err != nil {
		t.Fatalf("Failed to initialize custom graph: %v", err)
	}

	fmt.Println("Custom graph initialized successfully")
}

// TestGraphBuilderUpdateProperties 覆盖更新节点属性。
func TestGraphBuilderUpdateProperties(t *testing.T) {
	db := skipIfNoDB(t)
	defer db.Close()

	gb := indexer.NewGraphBuilder(db, nil)
	ctx := context.Background()

	if err := gb.InitGraph(ctx); err != nil {
		t.Fatalf("Failed to initialize graph: %v", err)
	}

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

	if _, err := gb.CreateNodes(ctx, []schema.Symbol{symbol}); err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	props := map[string]interface{}{
		"version":     "2.0",
		"description": "Updated function",
	}

	if err := gb.UpdateNodeProperties(ctx, "func-update-example", props); err != nil {
		t.Fatalf("Failed to update properties: %v", err)
	}

	fmt.Println("Node properties updated successfully")
}
