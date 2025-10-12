package schema

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
	"github.com/yourtionguo/CodeAtlas/internal/utils"
)

func TestNewSchemaMapper(t *testing.T) {
	mapper := NewSchemaMapper()
	if mapper == nil {
		t.Fatal("NewSchemaMapper returned nil")
	}
	if mapper.symbolIDMap == nil {
		t.Error("symbolIDMap not initialized")
	}
}

func TestMapToSchema(t *testing.T) {
	mapper := NewSchemaMapper()

	// Create a simple parsed file
	content := []byte(`package main

func Hello() string {
	return "world"
}
`)

	parsedFile := &parser.ParsedFile{
		Path:     "test.go",
		Language: "go",
		Content:  content,
		Symbols: []parser.ParsedSymbol{
			{
				Name:      "main",
				Kind:      "package",
				Signature: "package main",
				Span: parser.ParsedSpan{
					StartLine: 1,
					EndLine:   1,
					StartByte: 0,
					EndByte:   12,
				},
			},
			{
				Name:      "Hello",
				Kind:      "function",
				Signature: "func Hello() string",
				Span: parser.ParsedSpan{
					StartLine: 3,
					EndLine:   5,
					StartByte: 14,
					EndByte:   56,
				},
				Docstring: "Hello returns a greeting",
			},
		},
		Dependencies: []parser.ParsedDependency{
			{
				Type:         "import",
				Target:       "fmt",
				TargetModule: "fmt",
			},
		},
	}

	file, edges, err := mapper.MapToSchema(parsedFile)
	if err != nil {
		t.Fatalf("MapToSchema failed: %v", err)
	}

	// Verify file metadata
	if file.Path != "test.go" {
		t.Errorf("Expected path 'test.go', got '%s'", file.Path)
	}
	if file.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", file.Language)
	}
	if file.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), file.Size)
	}
	if file.FileID == "" {
		t.Error("FileID not generated")
	}
	if file.Checksum == "" {
		t.Error("Checksum not generated")
	}

	// Verify symbols
	if len(file.Symbols) != 2 {
		t.Fatalf("Expected 2 symbols, got %d", len(file.Symbols))
	}

	// Check package symbol
	pkgSymbol := file.Symbols[0]
	if pkgSymbol.Name != "main" {
		t.Errorf("Expected symbol name 'main', got '%s'", pkgSymbol.Name)
	}
	if pkgSymbol.Kind != SymbolPackage {
		t.Errorf("Expected kind SymbolPackage, got %s", pkgSymbol.Kind)
	}
	if pkgSymbol.SymbolID == "" {
		t.Error("SymbolID not generated")
	}
	if pkgSymbol.FileID != file.FileID {
		t.Error("Symbol FileID doesn't match file FileID")
	}

	// Check function symbol
	funcSymbol := file.Symbols[1]
	if funcSymbol.Name != "Hello" {
		t.Errorf("Expected symbol name 'Hello', got '%s'", funcSymbol.Name)
	}
	if funcSymbol.Kind != SymbolFunction {
		t.Errorf("Expected kind SymbolFunction, got %s", funcSymbol.Kind)
	}
	if funcSymbol.Signature != "func Hello() string" {
		t.Errorf("Expected signature 'func Hello() string', got '%s'", funcSymbol.Signature)
	}
	if funcSymbol.Docstring != "Hello returns a greeting" {
		t.Errorf("Expected docstring 'Hello returns a greeting', got '%s'", funcSymbol.Docstring)
	}

	// Verify span mapping
	if funcSymbol.Span.StartLine != 3 {
		t.Errorf("Expected start line 3, got %d", funcSymbol.Span.StartLine)
	}
	if funcSymbol.Span.EndLine != 5 {
		t.Errorf("Expected end line 5, got %d", funcSymbol.Span.EndLine)
	}

	// Verify edges
	if len(edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(edges))
	}

	edge := edges[0]
	if edge.EdgeType != EdgeImport {
		t.Errorf("Expected EdgeImport, got %s", edge.EdgeType)
	}
	if edge.TargetModule != "fmt" {
		t.Errorf("Expected target module 'fmt', got '%s'", edge.TargetModule)
	}
	if edge.EdgeID == "" {
		t.Error("EdgeID not generated")
	}
}

func TestMapSymbol(t *testing.T) {
	mapper := NewSchemaMapper()
	fileID := "test-file-id"

	tests := []struct {
		name         string
		parsedSymbol parser.ParsedSymbol
		expectedKind SymbolKind
		expectedName string
		expectedSig  string
		expectedDoc  string
	}{
		{
			name: "function symbol",
			parsedSymbol: parser.ParsedSymbol{
				Name:      "TestFunc",
				Kind:      "function",
				Signature: "func TestFunc()",
				Docstring: "Test function",
				Span: parser.ParsedSpan{
					StartLine: 1,
					EndLine:   3,
					StartByte: 0,
					EndByte:   50,
				},
			},
			expectedKind: SymbolFunction,
			expectedName: "TestFunc",
			expectedSig:  "func TestFunc()",
			expectedDoc:  "Test function",
		},
		{
			name: "class symbol",
			parsedSymbol: parser.ParsedSymbol{
				Name:      "MyClass",
				Kind:      "class",
				Signature: "class MyClass",
				Span: parser.ParsedSpan{
					StartLine: 5,
					EndLine:   10,
					StartByte: 100,
					EndByte:   200,
				},
			},
			expectedKind: SymbolClass,
			expectedName: "MyClass",
			expectedSig:  "class MyClass",
		},
		{
			name: "interface symbol",
			parsedSymbol: parser.ParsedSymbol{
				Name:      "MyInterface",
				Kind:      "interface",
				Signature: "type MyInterface interface",
				Span: parser.ParsedSpan{
					StartLine: 12,
					EndLine:   15,
					StartByte: 250,
					EndByte:   300,
				},
			},
			expectedKind: SymbolInterface,
			expectedName: "MyInterface",
			expectedSig:  "type MyInterface interface",
		},
		{
			name: "struct symbol",
			parsedSymbol: parser.ParsedSymbol{
				Name:      "MyStruct",
				Kind:      "struct",
				Signature: "type MyStruct struct",
				Span: parser.ParsedSpan{
					StartLine: 20,
					EndLine:   25,
					StartByte: 400,
					EndByte:   500,
				},
			},
			expectedKind: SymbolClass,
			expectedName: "MyStruct",
			expectedSig:  "type MyStruct struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := mapper.mapSymbol(tt.parsedSymbol, fileID)

			if symbol.SymbolID == "" {
				t.Error("SymbolID not generated")
			}
			if symbol.FileID != fileID {
				t.Errorf("Expected FileID '%s', got '%s'", fileID, symbol.FileID)
			}
			if symbol.Name != tt.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tt.expectedName, symbol.Name)
			}
			if symbol.Kind != tt.expectedKind {
				t.Errorf("Expected kind %s, got %s", tt.expectedKind, symbol.Kind)
			}
			if symbol.Signature != tt.expectedSig {
				t.Errorf("Expected signature '%s', got '%s'", tt.expectedSig, symbol.Signature)
			}
			if symbol.Docstring != tt.expectedDoc {
				t.Errorf("Expected docstring '%s', got '%s'", tt.expectedDoc, symbol.Docstring)
			}

			// Verify span mapping
			if symbol.Span.StartLine != tt.parsedSymbol.Span.StartLine {
				t.Errorf("Start line mismatch: expected %d, got %d",
					tt.parsedSymbol.Span.StartLine, symbol.Span.StartLine)
			}
			if symbol.Span.EndLine != tt.parsedSymbol.Span.EndLine {
				t.Errorf("End line mismatch: expected %d, got %d",
					tt.parsedSymbol.Span.EndLine, symbol.Span.EndLine)
			}
		})
	}
}

func TestMapSymbolKind(t *testing.T) {
	mapper := NewSchemaMapper()

	tests := []struct {
		input    string
		expected SymbolKind
	}{
		{"function", SymbolFunction},
		{"method", SymbolFunction},
		{"class", SymbolClass},
		{"struct", SymbolClass},
		{"interface", SymbolInterface},
		{"variable", SymbolVariable},
		{"field", SymbolVariable},
		{"type", SymbolVariable},
		{"package", SymbolPackage},
		{"module", SymbolModule},
		{"unknown", SymbolVariable}, // default case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapper.mapSymbolKind(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMapASTNodes(t *testing.T) {
	// Create a simple Go parser to get real Tree-sitter nodes
	tsParser := sitter.NewParser()
	lang := golang.GetLanguage()
	if lang == nil {
		t.Fatal("Failed to get Go language")
	}
	tsParser.SetLanguage(lang)

	content := []byte(`package main

func main() {
	x := 1
}
`)

	tree := tsParser.Parse(nil, content)
	if tree == nil {
		t.Fatal("Failed to parse: tree is nil")
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	if rootNode == nil {
		t.Fatal("Root node is nil")
	}

	mapper := NewSchemaMapper()
	fileID := "test-file-id"

	nodes := mapper.mapASTNodes(rootNode, fileID, "", content)

	if len(nodes) == 0 {
		t.Fatal("Expected AST nodes, got none")
	}

	// Check root node
	rootASTNode := nodes[0]
	if rootASTNode.NodeID == "" {
		t.Error("NodeID not generated")
	}
	if rootASTNode.FileID != fileID {
		t.Errorf("Expected FileID '%s', got '%s'", fileID, rootASTNode.FileID)
	}
	if rootASTNode.Type != "source_file" {
		t.Errorf("Expected type 'source_file', got '%s'", rootASTNode.Type)
	}
	if rootASTNode.ParentID != "" {
		t.Error("Root node should not have parent ID")
	}

	// Verify parent-child relationships
	parentFound := false
	for _, node := range nodes[1:] {
		if node.ParentID == rootASTNode.NodeID {
			parentFound = true
			break
		}
	}
	if !parentFound {
		t.Error("No child nodes found with root as parent")
	}

	// Verify span information
	if rootASTNode.Span.StartLine <= 0 {
		t.Error("Invalid start line")
	}
	if rootASTNode.Span.EndLine <= 0 {
		t.Error("Invalid end line")
	}
}

func TestMapASTNodesWithSmallText(t *testing.T) {
	tsParser := sitter.NewParser()
	lang := golang.GetLanguage()
	if lang == nil {
		t.Fatal("Failed to get Go language")
	}
	tsParser.SetLanguage(lang)

	// Small content that should have text extracted
	content := []byte(`package main`)

	tree := tsParser.Parse(nil, content)
	if tree == nil {
		t.Fatal("Failed to parse: tree is nil")
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	if rootNode == nil {
		t.Fatal("Root node is nil")
	}
	mapper := NewSchemaMapper()

	nodes := mapper.mapASTNodes(rootNode, "test-file", "", content)

	// Find a small node with text
	foundSmallNode := false
	for _, node := range nodes {
		if node.Text != "" {
			foundSmallNode = true
			// Verify text is reasonable size
			if len(node.Text) > 100 {
				t.Errorf("Text too large: %d bytes", len(node.Text))
			}
		}
	}

	if !foundSmallNode {
		t.Log("Note: No small nodes found with text (this is acceptable)")
	}
}

func TestMapDependencies(t *testing.T) {
	mapper := NewSchemaMapper()
	fileID := "test-file-id"
	filePath := "test.go"

	// Set up symbol ID map
	mapper.symbolIDMap = map[string]string{
		"main":  "symbol-main-id",
		"Hello": "symbol-hello-id",
		"fmt":   "symbol-fmt-id",
	}

	dependencies := []parser.ParsedDependency{
		{
			Type:         "import",
			Target:       "fmt",
			TargetModule: "fmt",
		},
		{
			Type:   "call",
			Source: "main",
			Target: "Hello",
		},
		{
			Type:   "extends",
			Source: "Child",
			Target: "Parent",
		},
	}

	edges := mapper.mapDependencies(dependencies, fileID, filePath)

	// Should have 2 edges (import and call, extends is skipped due to missing IDs)
	if len(edges) != 2 {
		t.Fatalf("Expected 2 edges, got %d", len(edges))
	}

	// Check import edge
	importEdge := edges[0]
	if importEdge.EdgeType != EdgeImport {
		t.Errorf("Expected EdgeImport, got %s", importEdge.EdgeType)
	}
	if importEdge.TargetModule != "fmt" {
		t.Errorf("Expected target module 'fmt', got '%s'", importEdge.TargetModule)
	}
	if importEdge.SourceFile != filePath {
		t.Errorf("Expected source file '%s', got '%s'", filePath, importEdge.SourceFile)
	}

	// Check call edge
	callEdge := edges[1]
	if callEdge.EdgeType != EdgeCall {
		t.Errorf("Expected EdgeCall, got %s", callEdge.EdgeType)
	}
	if callEdge.SourceID != "symbol-main-id" {
		t.Errorf("Expected source ID 'symbol-main-id', got '%s'", callEdge.SourceID)
	}
	if callEdge.TargetID != "symbol-hello-id" {
		t.Errorf("Expected target ID 'symbol-hello-id', got '%s'", callEdge.TargetID)
	}
}

func TestMapDependency(t *testing.T) {
	mapper := NewSchemaMapper()
	fileID := "test-file-id"
	filePath := "test.go"

	tests := []struct {
		name         string
		dependency   parser.ParsedDependency
		symbolMap    map[string]string
		expectNil    bool
		expectedType EdgeType
	}{
		{
			name: "import dependency",
			dependency: parser.ParsedDependency{
				Type:         "import",
				Target:       "fmt",
				TargetModule: "fmt",
			},
			symbolMap:    map[string]string{},
			expectNil:    false,
			expectedType: EdgeImport,
		},
		{
			name: "call dependency with IDs",
			dependency: parser.ParsedDependency{
				Type:   "call",
				Source: "caller",
				Target: "callee",
			},
			symbolMap: map[string]string{
				"caller": "caller-id",
				"callee": "callee-id",
			},
			expectNil:    false,
			expectedType: EdgeCall,
		},
		{
			name: "call dependency without source ID",
			dependency: parser.ParsedDependency{
				Type:   "call",
				Source: "unknown",
				Target: "callee",
			},
			symbolMap: map[string]string{
				"callee": "callee-id",
			},
			expectNil: true,
		},
		{
			name: "extends dependency",
			dependency: parser.ParsedDependency{
				Type:   "extends",
				Source: "child",
				Target: "parent",
			},
			symbolMap: map[string]string{
				"child":  "child-id",
				"parent": "parent-id",
			},
			expectNil:    false,
			expectedType: EdgeExtends,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper.symbolIDMap = tt.symbolMap

			edge := mapper.mapDependency(tt.dependency, fileID, filePath)

			if tt.expectNil {
				if edge != nil {
					t.Error("Expected nil edge, got non-nil")
				}
				return
			}

			if edge == nil {
				t.Fatal("Expected non-nil edge, got nil")
			}

			if edge.EdgeID == "" {
				t.Error("EdgeID not generated")
			}
			if edge.EdgeType != tt.expectedType {
				t.Errorf("Expected edge type %s, got %s", tt.expectedType, edge.EdgeType)
			}
			if edge.SourceFile != filePath {
				t.Errorf("Expected source file '%s', got '%s'", filePath, edge.SourceFile)
			}
		})
	}
}

func TestMapEdgeType(t *testing.T) {
	mapper := NewSchemaMapper()

	tests := []struct {
		input    string
		expected EdgeType
	}{
		{"import", EdgeImport},
		{"call", EdgeCall},
		{"extends", EdgeExtends},
		{"implements", EdgeImplements},
		{"unknown", EdgeReference}, // default case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapper.mapEdgeType(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestResolveSymbolID(t *testing.T) {
	mapper := NewSchemaMapper()
	mapper.symbolIDMap = map[string]string{
		"func1": "id-1",
		"func2": "id-2",
	}

	tests := []struct {
		name       string
		symbolName string
		expected   string
	}{
		{"existing symbol", "func1", "id-1"},
		{"another existing symbol", "func2", "id-2"},
		{"non-existing symbol", "func3", ""},
		{"empty symbol name", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapper.resolveSymbolID(tt.symbolName)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestUUIDGenerationConsistency(t *testing.T) {
	mapper := NewSchemaMapper()

	content := []byte(`package main

func Test() {}
`)

	parsedFile := &parser.ParsedFile{
		Path:     "test.go",
		Language: "go",
		Content:  content,
		Symbols: []parser.ParsedSymbol{
			{
				Name:      "Test",
				Kind:      "function",
				Signature: "func Test()",
				Span: parser.ParsedSpan{
					StartLine: 3,
					EndLine:   3,
					StartByte: 14,
					EndByte:   28,
				},
			},
		},
	}

	// Map twice
	file1, edges1, _ := mapper.MapToSchema(parsedFile)
	file2, edges2, _ := mapper.MapToSchema(parsedFile)

	// UUIDs should be different between runs
	if file1.FileID == file2.FileID {
		t.Error("FileIDs should be unique across runs")
	}
	if file1.Symbols[0].SymbolID == file2.Symbols[0].SymbolID {
		t.Error("SymbolIDs should be unique across runs")
	}
	if len(edges1) > 0 && len(edges2) > 0 && edges1[0].EdgeID == edges2[0].EdgeID {
		t.Error("EdgeIDs should be unique across runs")
	}
}

func TestChecksumGeneration(t *testing.T) {
	mapper := NewSchemaMapper()

	content1 := []byte(`package main`)
	content2 := []byte(`package test`)

	parsedFile1 := &parser.ParsedFile{
		Path:     "test1.go",
		Language: "go",
		Content:  content1,
	}

	parsedFile2 := &parser.ParsedFile{
		Path:     "test2.go",
		Language: "go",
		Content:  content2,
	}

	file1, _, _ := mapper.MapToSchema(parsedFile1)
	file2, _, _ := mapper.MapToSchema(parsedFile2)

	// Checksums should be different for different content
	if file1.Checksum == file2.Checksum {
		t.Error("Checksums should be different for different content")
	}

	// Checksum should be consistent for same content
	file1Again, _, _ := mapper.MapToSchema(parsedFile1)
	if file1.Checksum != file1Again.Checksum {
		t.Error("Checksum should be consistent for same content")
	}

	// Verify checksum format (SHA256 hex string should be 64 characters)
	if len(file1.Checksum) != 64 {
		t.Errorf("Expected checksum length 64, got %d", len(file1.Checksum))
	}

	// Verify it matches the expected checksum
	expectedChecksum := utils.SHA256Checksum(content1)
	if file1.Checksum != expectedChecksum {
		t.Errorf("Checksum mismatch: expected %s, got %s", expectedChecksum, file1.Checksum)
	}
}

func TestRelationshipExtraction(t *testing.T) {
	mapper := NewSchemaMapper()

	content := []byte(`package main

import "fmt"

func main() {
	hello()
}

func hello() {
	fmt.Println("hello")
}
`)

	parsedFile := &parser.ParsedFile{
		Path:     "test.go",
		Language: "go",
		Content:  content,
		Symbols: []parser.ParsedSymbol{
			{Name: "main", Kind: "package"},
			{Name: "main", Kind: "function"},
			{Name: "hello", Kind: "function"},
		},
		Dependencies: []parser.ParsedDependency{
			{
				Type:         "import",
				Target:       "fmt",
				TargetModule: "fmt",
			},
			{
				Type:   "call",
				Source: "main",
				Target: "hello",
			},
		},
	}

	file, edges, err := mapper.MapToSchema(parsedFile)
	if err != nil {
		t.Fatalf("MapToSchema failed: %v", err)
	}

	// Should have 2 edges (import and call)
	if len(edges) != 2 {
		t.Fatalf("Expected 2 edges, got %d", len(edges))
	}

	// Verify import edge
	var importEdge *DependencyEdge
	for i := range edges {
		if edges[i].EdgeType == EdgeImport {
			importEdge = &edges[i]
			break
		}
	}
	if importEdge == nil {
		t.Fatal("Import edge not found")
	}
	if importEdge.TargetModule != "fmt" {
		t.Errorf("Expected target module 'fmt', got '%s'", importEdge.TargetModule)
	}

	// Verify call edge
	var callEdge *DependencyEdge
	for i := range edges {
		if edges[i].EdgeType == EdgeCall {
			callEdge = &edges[i]
			break
		}
	}
	if callEdge == nil {
		t.Fatal("Call edge not found")
	}

	// Verify source and target are resolved to symbol IDs
	if callEdge.SourceID == "" {
		t.Error("Call edge source ID not resolved")
	}
	if callEdge.TargetID == "" {
		t.Error("Call edge target ID not resolved")
	}

	// Verify the IDs match symbols in the file
	foundSource := false
	foundTarget := false
	for _, symbol := range file.Symbols {
		if symbol.SymbolID == callEdge.SourceID {
			foundSource = true
		}
		if symbol.SymbolID == callEdge.TargetID {
			foundTarget = true
		}
	}
	if !foundSource {
		t.Error("Call edge source ID not found in file symbols")
	}
	if !foundTarget {
		t.Error("Call edge target ID not found in file symbols")
	}
}
