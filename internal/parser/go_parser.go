package parser

import (
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// DetailedParseError represents a detailed parsing error with location information
type DetailedParseError struct {
	File    string
	Line    int
	Column  int
	Message string
	Type    string // filesystem, parse, mapping
}

func (e *DetailedParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.File, e.Message)
}

// ParsedFile represents the internal representation of a parsed file
type ParsedFile struct {
	Path         string
	Language     string
	Content      []byte
	Checksum     string
	RootNode     *sitter.Node
	Symbols      []ParsedSymbol
	Dependencies []ParsedDependency
}

// ParsedSymbol represents a code symbol (function, class, etc.)
type ParsedSymbol struct {
	Name      string
	Kind      string
	Signature string
	Span      ParsedSpan
	Docstring string
	Node      *sitter.Node
	Children  []ParsedSymbol
}

// ParsedSpan represents the location of a code element
type ParsedSpan struct {
	StartLine int
	EndLine   int
	StartByte int
	EndByte   int
}

// ParsedDependency represents a relationship between code elements
type ParsedDependency struct {
	Type         string // import, call, extends, etc.
	Source       string // Symbol name
	Target       string // Symbol/module name
	TargetModule string // For imports
	IsExternal   bool   // True if this is an external dependency (e.g., npm package, third-party library)
}

// GoParser parses Go source code using Tree-sitter
type GoParser struct {
	tsParser *TreeSitterParser
}

// NewGoParser creates a new Go parser
func NewGoParser(tsParser *TreeSitterParser) *GoParser {
	return &GoParser{
		tsParser: tsParser,
	}
}

// Parse parses a Go file and extracts symbols and dependencies
func (p *GoParser) Parse(file ScannedFile) (*ParsedFile, error) {
	// Read file content
	content, err := readFileContent(file.AbsPath)
	if err != nil {
		return nil, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to read file: %v", err),
			Type:    "filesystem",
		}
	}

	// Parse with Tree-sitter
	rootNode, parseErr := p.tsParser.Parse(content, "go")

	parsedFile := &ParsedFile{
		Path:     file.Path,
		Language: "go",
		Content:  content,
		RootNode: rootNode,
	}

	// If we have no root node at all, return error immediately
	if rootNode == nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to parse Go file: %v", parseErr),
			Type:    "parse",
		}
	}

	// Extract package declaration
	if err := p.extractPackage(rootNode, parsedFile); err != nil {
		// Non-fatal, continue
	}

	// Extract imports
	if err := p.extractImports(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract functions
	if err := p.extractFunctions(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract methods
	if err := p.extractMethods(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract structs
	if err := p.extractStructs(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract interfaces
	if err := p.extractInterfaces(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract type aliases
	if err := p.extractTypes(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract call relationships
	if err := p.extractCallRelationships(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Return parse error if there was one, but with partial results
	if parseErr != nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("syntax error in Go file: %v", parseErr),
			Type:    "parse",
		}
	}

	return parsedFile, nil
}

// extractPackage extracts the package declaration
func (p *GoParser) extractPackage(rootNode *sitter.Node, parsedFile *ParsedFile) error {
	// Walk the tree directly to find package_clause (more robust for error recovery)
	for i := 0; i < int(rootNode.ChildCount()); i++ {
		child := rootNode.Child(i)
		if child.Type() == "package_clause" {
			// Find the package_identifier child
			for j := 0; j < int(child.ChildCount()); j++ {
				pkgChild := child.Child(j)
				if pkgChild.Type() == "package_identifier" {
					packageName := pkgChild.Content(parsedFile.Content)

					symbol := ParsedSymbol{
						Name: packageName,
						Kind: "package",
						Span: nodeToSpan(pkgChild),
						Node: pkgChild,
					}

					parsedFile.Symbols = append(parsedFile.Symbols, symbol)
					return nil
				}
			}
		}
	}

	return nil
}

// extractImports extracts import statements
func (p *GoParser) extractImports(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	query := `(import_spec path: (interpreted_string_literal) @import.path)`

	matches, err := p.tsParser.Query(rootNode, query, "go")
	if err != nil {
		return err
	}

	// Find the package symbol to use as the source for imports
	var packageSymbol string
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "package" {
			packageSymbol = symbol.Name
			break
		}
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			importPath := strings.Trim(capture.Node.Content(content), "\"")

			dependency := ParsedDependency{
				Type:         "import",
				Source:       packageSymbol, // Use package as source for file-level imports
				Target:       importPath,
				TargetModule: importPath,
				IsExternal:   p.isExternalImport(importPath, parsedFile.Path),
			}

			parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
		}
	}

	return nil
}

// isExternalImport determines if an import path refers to an external module
func (p *GoParser) isExternalImport(importPath string, currentFilePath string) bool {
	// Standard library packages (no dots in path, or starts with known stdlib prefixes)
	if !strings.Contains(importPath, ".") {
		return false // Standard library is considered internal
	}
	
	// Extract module path from current file to determine internal imports
	// For now, treat all imports with dots as external (third-party)
	// A more sophisticated approach would parse go.mod to determine the module path
	return true
}

// extractFunctions extracts function declarations
func (p *GoParser) extractFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	query := `(function_declaration name: (identifier) @func.name) @func.def`

	matches, err := p.tsParser.Query(rootNode, query, "go")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var funcNode *sitter.Node
		var funcName string

		// Use Index to identify captures: 0=func.name, 1=func.def
		for _, capture := range match.Captures {
			if capture.Index == 0 {
				funcName = capture.Node.Content(content)
			} else if capture.Index == 1 {
				funcNode = capture.Node
			}
		}

		if funcNode != nil && funcName != "" {
			signature := p.extractFunctionSignature(funcNode, content)
			docstring := p.extractDocstring(funcNode, content)

			symbol := ParsedSymbol{
				Name:      funcName,
				Kind:      "function",
				Signature: signature,
				Span:      nodeToSpan(funcNode),
				Docstring: docstring,
				Node:      funcNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractMethods extracts method declarations
func (p *GoParser) extractMethods(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	query := `(method_declaration 
		receiver: (parameter_list) @method.receiver 
		name: (field_identifier) @method.name) @method.def`

	matches, err := p.tsParser.Query(rootNode, query, "go")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var methodNode *sitter.Node
		var methodName string
		var receiver string

		// Use Index to identify captures: 0=receiver, 1=name, 2=def
		for _, capture := range match.Captures {
			if capture.Index == 0 {
				receiver = capture.Node.Content(content)
			} else if capture.Index == 1 {
				methodName = capture.Node.Content(content)
			} else if capture.Index == 2 {
				methodNode = capture.Node
			}
		}

		if methodNode != nil && methodName != "" {
			signature := p.extractMethodSignature(methodNode, receiver, content)
			docstring := p.extractDocstring(methodNode, content)

			symbol := ParsedSymbol{
				Name:      methodName,
				Kind:      "method",
				Signature: signature,
				Span:      nodeToSpan(methodNode),
				Docstring: docstring,
				Node:      methodNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractStructs extracts struct definitions
func (p *GoParser) extractStructs(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	query := `(type_declaration 
		(type_spec 
			name: (type_identifier) @struct.name 
			type: (struct_type) @struct.body)) @struct.def`

	matches, err := p.tsParser.Query(rootNode, query, "go")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var structNode *sitter.Node
		var structName string
		var structBody *sitter.Node

		// Use Index to identify captures: 0=name, 1=body, 2=def
		for _, capture := range match.Captures {
			if capture.Index == 0 {
				structName = capture.Node.Content(content)
			} else if capture.Index == 1 {
				structBody = capture.Node
			} else if capture.Index == 2 {
				structNode = capture.Node
			}
		}

		if structNode != nil && structName != "" {
			fields := p.extractStructFields(structBody, content)
			docstring := p.extractDocstring(structNode, content)

			symbol := ParsedSymbol{
				Name:      structName,
				Kind:      "struct",
				Signature: fmt.Sprintf("type %s struct", structName),
				Span:      nodeToSpan(structNode),
				Docstring: docstring,
				Node:      structNode,
				Children:  fields,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractInterfaces extracts interface definitions
func (p *GoParser) extractInterfaces(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	query := `(type_declaration 
		(type_spec 
			name: (type_identifier) @interface.name 
			type: (interface_type) @interface.body)) @interface.def`

	matches, err := p.tsParser.Query(rootNode, query, "go")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var interfaceNode *sitter.Node
		var interfaceName string
		var interfaceBody *sitter.Node

		// Use Index to identify captures: 0=name, 1=body, 2=def
		for _, capture := range match.Captures {
			if capture.Index == 0 {
				interfaceName = capture.Node.Content(content)
			} else if capture.Index == 1 {
				interfaceBody = capture.Node
			} else if capture.Index == 2 {
				interfaceNode = capture.Node
			}
		}

		if interfaceNode != nil && interfaceName != "" {
			methods := p.extractInterfaceMethods(interfaceBody, content)
			docstring := p.extractDocstring(interfaceNode, content)

			symbol := ParsedSymbol{
				Name:      interfaceName,
				Kind:      "interface",
				Signature: fmt.Sprintf("type %s interface", interfaceName),
				Span:      nodeToSpan(interfaceNode),
				Docstring: docstring,
				Node:      interfaceNode,
				Children:  methods,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractTypes extracts type aliases
func (p *GoParser) extractTypes(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	query := `(type_declaration 
		(type_spec 
			name: (type_identifier) @type.name)) @type.def`

	matches, err := p.tsParser.Query(rootNode, query, "go")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var typeNode *sitter.Node
		var typeName string

		// Use Index to identify captures: 0=name, 1=def
		for _, capture := range match.Captures {
			if capture.Index == 0 {
				typeName = capture.Node.Content(content)
			} else if capture.Index == 1 {
				typeNode = capture.Node
			}
		}

		if typeNode != nil && typeName != "" {
			// Skip if already processed as struct or interface
			typeSpec := findChildByType(typeNode, "type_spec")
			if typeSpec != nil {
				typeType := findChildByType(typeSpec, "struct_type")
				if typeType == nil {
					typeType = findChildByType(typeSpec, "interface_type")
				}
				if typeType != nil {
					continue // Already processed
				}
			}

			docstring := p.extractDocstring(typeNode, content)
			signature := strings.TrimSpace(typeNode.Content(content))

			symbol := ParsedSymbol{
				Name:      typeName,
				Kind:      "type",
				Signature: signature,
				Span:      nodeToSpan(typeNode),
				Docstring: docstring,
				Node:      typeNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractFunctionSignature extracts the full function signature
func (p *GoParser) extractFunctionSignature(funcNode *sitter.Node, content []byte) string {
	// Get the signature line (first line of function)
	signature := strings.Split(funcNode.Content(content), "\n")[0]
	return strings.TrimSpace(signature)
}

// extractMethodSignature extracts the full method signature including receiver
func (p *GoParser) extractMethodSignature(methodNode *sitter.Node, receiver string, content []byte) string {
	signature := strings.Split(methodNode.Content(content), "\n")[0]
	return strings.TrimSpace(signature)
}

// extractStructFields extracts fields from a struct
func (p *GoParser) extractStructFields(structBody *sitter.Node, content []byte) []ParsedSymbol {
	if structBody == nil {
		return nil
	}

	var fields []ParsedSymbol

	// Iterate through children to find field declarations
	for i := 0; i < int(structBody.ChildCount()); i++ {
		child := structBody.Child(i)
		if child.Type() == "field_declaration_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				fieldDecl := child.Child(j)
				if fieldDecl.Type() == "field_declaration" {
					fieldName := ""
					fieldType := ""

					// Extract field name and type
					for k := 0; k < int(fieldDecl.ChildCount()); k++ {
						fieldChild := fieldDecl.Child(k)
						if fieldChild.Type() == "field_identifier" {
							fieldName = fieldChild.Content(content)
						} else if fieldName != "" && fieldType == "" {
							fieldType = fieldChild.Content(content)
						}
					}

					if fieldName != "" {
						field := ParsedSymbol{
							Name:      fieldName,
							Kind:      "field",
							Signature: fmt.Sprintf("%s %s", fieldName, fieldType),
							Span:      nodeToSpan(fieldDecl),
							Node:      fieldDecl,
						}
						fields = append(fields, field)
					}
				}
			}
		}
	}

	return fields
}

// extractInterfaceMethods extracts methods from an interface
func (p *GoParser) extractInterfaceMethods(interfaceBody *sitter.Node, content []byte) []ParsedSymbol {
	if interfaceBody == nil {
		return nil
	}

	var methods []ParsedSymbol

	// Iterate through children to find method elements
	for i := 0; i < int(interfaceBody.ChildCount()); i++ {
		child := interfaceBody.Child(i)
		if child.Type() == "method_elem" || child.Type() == "method_spec" {
			methodName := ""

			// Find method name (first field_identifier child)
			for j := 0; j < int(child.ChildCount()); j++ {
				methodChild := child.Child(j)
				if methodChild.Type() == "field_identifier" {
					methodName = methodChild.Content(content)
					break
				}
			}

			if methodName != "" {
				signature := strings.TrimSpace(child.Content(content))
				method := ParsedSymbol{
					Name:      methodName,
					Kind:      "method",
					Signature: signature,
					Span:      nodeToSpan(child),
					Node:      child,
				}
				methods = append(methods, method)
			}
		}
	}

	return methods
}

// extractDocstring extracts the comment/docstring before a node
func (p *GoParser) extractDocstring(node *sitter.Node, content []byte) string {
	if node == nil {
		return ""
	}

	// Look for comment nodes before this node
	parent := node.Parent()
	if parent == nil {
		return ""
	}

	var comments []string

	// Find the index of the current node
	nodeIndex := -1
	for i := 0; i < int(parent.ChildCount()); i++ {
		if parent.Child(i) == node {
			nodeIndex = i
			break
		}
	}

	if nodeIndex <= 0 {
		return ""
	}

	// Look backwards for comments
	for i := nodeIndex - 1; i >= 0; i-- {
		sibling := parent.Child(i)
		if sibling.Type() == "comment" {
			commentText := sibling.Content(content)
			// Remove comment markers
			commentText = strings.TrimPrefix(commentText, "//")
			commentText = strings.TrimSpace(commentText)
			comments = append([]string{commentText}, comments...)
		} else if sibling.Type() != "comment" {
			break
		}
	}

	return strings.Join(comments, "\n")
}

// extractCallRelationships extracts function call relationships
func (p *GoParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for call expressions
	query := `(call_expression function: [(identifier) (selector_expression)] @call.target)`

	matches, err := p.tsParser.Query(rootNode, query, "go")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			callTarget := capture.Node.Content(content)

			// Find the containing function/method for this call
			caller := p.findContainingFunction(capture.Node, parsedFile)
			if caller != "" {
				dependency := ParsedDependency{
					Type:   "call",
					Source: caller,
					Target: callTarget,
				}

				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// findContainingFunction finds the name of the function/method containing a node
func (p *GoParser) findContainingFunction(node *sitter.Node, parsedFile *ParsedFile) string {
	current := node.Parent()

	for current != nil {
		// Check if this is a function or method declaration
		if current.Type() == "function_declaration" || current.Type() == "method_declaration" {
			// Find the matching symbol in our parsed symbols
			for _, symbol := range parsedFile.Symbols {
				if symbol.Node == current {
					return symbol.Name
				}
			}
		}
		current = current.Parent()
	}

	return ""
}

// Helper functions

func nodeToSpan(node *sitter.Node) ParsedSpan {
	return ParsedSpan{
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		StartByte: int(node.StartByte()),
		EndByte:   int(node.EndByte()),
	}
}

func findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	if node == nil {
		return nil
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}

	return nil
}

func readFileContent(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return content, nil
}
