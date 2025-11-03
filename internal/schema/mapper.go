package schema

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
	"github.com/yourtionguo/CodeAtlas/internal/utils"
)

// SchemaMapper transforms parsed files into the unified schema format
type SchemaMapper struct {
	// Map to track symbol IDs for dependency resolution
	symbolIDMap map[string]string
}

// NewSchemaMapper creates a new schema mapper
func NewSchemaMapper() *SchemaMapper {
	return &SchemaMapper{
		symbolIDMap: make(map[string]string),
	}
}

// MapToSchema transforms a ParsedFile into a schema.File
func (m *SchemaMapper) MapToSchema(parsed *parser.ParsedFile) (*File, []DependencyEdge, error) {
	// Generate deterministic file ID based on path and checksum
	// This ensures the same file always gets the same ID
	checksum := utils.SHA256Checksum(parsed.Content)
	fileID := utils.GenerateDeterministicUUID(fmt.Sprintf("file:%s:%s", parsed.Path, checksum))

	file := &File{
		FileID:   fileID,
		Path:     parsed.Path,
		Language: parsed.Language,
		Size:     int64(len(parsed.Content)),
		Checksum: checksum,
		Nodes:    []ASTNode{},
		Symbols:  []Symbol{},
	}

	// Reset symbol ID map for this file
	m.symbolIDMap = make(map[string]string)

	// Map symbols
	for _, parsedSymbol := range parsed.Symbols {
		symbol := m.mapSymbol(parsedSymbol, fileID)
		file.Symbols = append(file.Symbols, symbol)

		// Store symbol ID for dependency resolution
		m.symbolIDMap[parsedSymbol.Name] = symbol.SymbolID
	}

	// Map AST nodes if root node exists
	if parsed.RootNode != nil {
		astNodes := m.mapASTNodes(parsed.RootNode, fileID, "", parsed.Content)
		file.Nodes = astNodes
	}

	// Map dependencies
	edges := m.mapDependencies(parsed.Dependencies, fileID, parsed.Path)

	return file, edges, nil
}

// mapSymbol transforms a ParsedSymbol into a schema.Symbol
func (m *SchemaMapper) mapSymbol(parsed parser.ParsedSymbol, fileID string) Symbol {
	// Generate deterministic UUID based on file_id, name, start_line, and start_byte
	// This ensures the same symbol always gets the same ID across multiple parses
	symbolKey := fmt.Sprintf("%s:%s:%d:%d", fileID, parsed.Name, parsed.Span.StartLine, parsed.Span.StartByte)
	symbolID := utils.GenerateDeterministicUUID(symbolKey)

	// Map symbol kind
	kind := m.mapSymbolKind(parsed.Kind)

	// Map span
	span := Span{
		StartLine: parsed.Span.StartLine,
		EndLine:   parsed.Span.EndLine,
		StartByte: parsed.Span.StartByte,
		EndByte:   parsed.Span.EndByte,
	}

	symbol := Symbol{
		SymbolID:  symbolID,
		FileID:    fileID,
		Name:      parsed.Name,
		Kind:      kind,
		Signature: parsed.Signature,
		Span:      span,
		Docstring: parsed.Docstring,
	}

	return symbol
}

// mapSymbolKind maps parser symbol kinds to schema symbol kinds
func (m *SchemaMapper) mapSymbolKind(kind string) SymbolKind {
	switch kind {
	case "function", "method":
		return SymbolFunction
	case "class", "struct":
		return SymbolClass
	case "interface":
		return SymbolInterface
	case "variable", "field", "type":
		return SymbolVariable
	case "package":
		return SymbolPackage
	case "module":
		return SymbolModule
	default:
		return SymbolVariable
	}
}

// mapASTNodes recursively transforms Tree-sitter nodes into schema.ASTNode
func (m *SchemaMapper) mapASTNodes(node *sitter.Node, fileID string, parentID string, content []byte) []ASTNode {
	if node == nil {
		return nil
	}

	var nodes []ASTNode

	// Create node for current Tree-sitter node
	nodeID := utils.GenerateUUID()

	span := Span{
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		StartByte: int(node.StartByte()),
		EndByte:   int(node.EndByte()),
	}

	// Extract text for small nodes (< 100 bytes)
	text := ""
	nodeSize := int(node.EndByte() - node.StartByte())
	if nodeSize < 100 {
		text = node.Content(content)
	}

	astNode := ASTNode{
		NodeID:     nodeID,
		FileID:     fileID,
		Type:       node.Type(),
		Span:       span,
		ParentID:   parentID,
		Text:       text,
		Attributes: make(map[string]string),
	}

	// Add node type as attribute
	if node.IsNamed() {
		astNode.Attributes["named"] = "true"
	}

	nodes = append(nodes, astNode)

	// Recursively process children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childNodes := m.mapASTNodes(child, fileID, nodeID, content)
		nodes = append(nodes, childNodes...)
	}

	return nodes
}

// mapDependencies transforms ParsedDependency into schema.DependencyEdge
func (m *SchemaMapper) mapDependencies(dependencies []parser.ParsedDependency, fileID string, filePath string) []DependencyEdge {
	var edges []DependencyEdge

	for _, dep := range dependencies {
		edge := m.mapDependency(dep, fileID, filePath)
		if edge != nil {
			edges = append(edges, *edge)
		}
	}

	return edges
}

// mapDependency transforms a single ParsedDependency into a schema.DependencyEdge
func (m *SchemaMapper) mapDependency(dep parser.ParsedDependency, fileID string, filePath string) *DependencyEdge {
	edgeID := utils.GenerateUUID()

	// Map edge type
	edgeType := m.mapEdgeType(dep.Type)

	// Resolve source and target symbol IDs
	sourceID := m.resolveSymbolID(dep.Source)
	targetID := m.resolveSymbolID(dep.Target)

	// For imports, we may not have a target symbol ID (external module)
	// In that case, we still create the edge with target module information
	if dep.Type == "import" {
		edge := DependencyEdge{
			EdgeID:       edgeID,
			SourceID:     sourceID,
			TargetID:     targetID,
			EdgeType:     edgeType,
			SourceFile:   filePath,
			TargetModule: dep.TargetModule,
		}
		return &edge
	}

	// For other edge types, we need both source and target IDs
	if sourceID == "" || targetID == "" {
		// Cannot create edge without both IDs
		return nil
	}

	edge := DependencyEdge{
		EdgeID:     edgeID,
		SourceID:   sourceID,
		TargetID:   targetID,
		EdgeType:   edgeType,
		SourceFile: filePath,
	}

	return &edge
}

// mapEdgeType maps parser dependency types to schema edge types
func (m *SchemaMapper) mapEdgeType(depType string) EdgeType {
	switch depType {
	case "import":
		return EdgeImport
	case "call":
		return EdgeCall
	case "extends":
		return EdgeExtends
	case "implements":
		return EdgeImplements
	default:
		return EdgeReference
	}
}

// resolveSymbolID looks up the symbol ID for a given symbol name
func (m *SchemaMapper) resolveSymbolID(symbolName string) string {
	if symbolName == "" {
		return ""
	}
	return m.symbolIDMap[symbolName]
}
