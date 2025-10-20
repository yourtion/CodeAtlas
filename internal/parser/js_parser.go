package parser

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// JSParser parses JavaScript/TypeScript source code using Tree-sitter
type JSParser struct {
	tsParser *TreeSitterParser
}

// NewJSParser creates a new JavaScript/TypeScript parser
func NewJSParser(tsParser *TreeSitterParser) *JSParser {
	return &JSParser{
		tsParser: tsParser,
	}
}

// Parse parses a JavaScript/TypeScript file and extracts symbols and dependencies
func (p *JSParser) Parse(file ScannedFile) (*ParsedFile, error) {
	// Read file content
	content, err := readFileContent(file.AbsPath)
	if err != nil {
		return nil, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to read file: %v", err),
			Type:    "filesystem",
		}
	}

	// Determine language (js vs ts)
	language := p.detectLanguage(file.Language)

	// Parse with Tree-sitter
	rootNode, parseErr := p.tsParser.Parse(content, language)

	parsedFile := &ParsedFile{
		Path:     file.Path,
		Language: language,
		Content:  content,
		RootNode: rootNode,
	}

	// If we have no root node at all, return error immediately
	if rootNode == nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to parse %s file: %v", language, parseErr),
			Type:    "parse",
		}
	}

	// Extract imports (ES6 and CommonJS)
	if err := p.extractImports(rootNode, parsedFile, content, language); err != nil {
		// Non-fatal, continue
	}

	// Extract exports
	if err := p.extractExports(rootNode, parsedFile, content, language); err != nil {
		// Non-fatal, continue
	}

	// Extract functions
	if err := p.extractFunctions(rootNode, parsedFile, content, language); err != nil {
		// Non-fatal, continue
	}

	// Extract arrow functions
	if err := p.extractArrowFunctions(rootNode, parsedFile, content, language); err != nil {
		// Non-fatal, continue
	}

	// Extract classes
	if err := p.extractClasses(rootNode, parsedFile, content, language); err != nil {
		// Non-fatal, continue
	}

	// Extract call relationships
	if err := p.extractCallRelationships(rootNode, parsedFile, content, language); err != nil {
		// Non-fatal, continue
	}

	// Return parse error if there was one, but with partial results
	if parseErr != nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("syntax error in %s file: %v", language, parseErr),
			Type:    "parse",
		}
	}

	return parsedFile, nil
}

// detectLanguage determines whether to use JavaScript or TypeScript parser
func (p *JSParser) detectLanguage(lang string) string {
	switch lang {
	case "typescript", "ts", "tsx":
		return "typescript"
	default:
		return "javascript"
	}
}

// extractImports extracts ES6 import statements and CommonJS require calls
func (p *JSParser) extractImports(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte, language string) error {
	// ES6 imports
	importQuery := `(import_statement source: (string) @import.source)`

	matches, err := p.tsParser.Query(rootNode, importQuery, language)
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			importPath := strings.Trim(capture.Node.Content(content), "\"'`")

			dependency := ParsedDependency{
				Type:         "import",
				Target:       importPath,
				TargetModule: importPath,
			}

			parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
		}
	}

	// CommonJS require
	requireQuery := `(call_expression
		function: (identifier) @func.name (#eq? @func.name "require")
		arguments: (arguments (string) @require.source))`

	matches, err = p.tsParser.Query(rootNode, requireQuery, language)
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			if capture.Index == 1 { // require.source
				requirePath := strings.Trim(capture.Node.Content(content), "\"'`")

				dependency := ParsedDependency{
					Type:         "import",
					Target:       requirePath,
					TargetModule: requirePath,
				}

				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractExports extracts export statements
func (p *JSParser) extractExports(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte, language string) error {
	// Export declarations (export function, export class, etc.)
	exportQuery := `(export_statement) @export`

	matches, err := p.tsParser.Query(rootNode, exportQuery, language)
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			exportText := capture.Node.Content(content)

			// Create a symbol for the export
			symbol := ParsedSymbol{
				Name:      "export",
				Kind:      "export",
				Signature: strings.Split(exportText, "\n")[0],
				Span:      nodeToSpan(capture.Node),
				Node:      capture.Node,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractFunctions extracts function declarations
func (p *JSParser) extractFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte, language string) error {
	// Regular function declarations
	funcQuery := `(function_declaration name: (identifier) @func.name) @func.def`

	matches, err := p.tsParser.Query(rootNode, funcQuery, language)
	if err != nil {
		return err
	}

	for _, match := range matches {
		var funcNode *sitter.Node
		var funcName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // func.name
				funcName = capture.Node.Content(content)
			} else if capture.Index == 1 { // func.def
				funcNode = capture.Node
			}
		}

		if funcNode != nil && funcName != "" {
			signature := p.extractFunctionSignature(funcNode, content, language)
			docstring := p.extractJSDoc(funcNode, content)
			isAsync := p.isAsyncFunction(funcNode, content)
			isGenerator := p.isGeneratorFunction(funcNode, content)

			kind := "function"
			if isAsync {
				kind = "async_function"
			}
			if isGenerator {
				kind = "generator_function"
			}

			symbol := ParsedSymbol{
				Name:      funcName,
				Kind:      kind,
				Signature: signature,
				Span:      nodeToSpan(funcNode),
				Docstring: docstring,
				Node:      funcNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	// Function expressions assigned to variables (const x = function() {})
	funcExprQuery := `(variable_declarator
		name: (identifier) @var.name
		value: (function_expression) @func.def)`

	matches, err = p.tsParser.Query(rootNode, funcExprQuery, language)
	if err != nil {
		return err
	}

	for _, match := range matches {
		var funcNode *sitter.Node
		var funcName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // var.name
				funcName = capture.Node.Content(content)
			} else if capture.Index == 1 { // func.def
				funcNode = capture.Node
			}
		}

		if funcNode != nil && funcName != "" {
			// Get the variable declarator parent for docstring
			varDeclarator := funcNode.Parent()
			var docstringNode *sitter.Node
			if varDeclarator != nil {
				// Go up to lexical_declaration or variable_declaration
				lexDecl := varDeclarator.Parent()
				if lexDecl != nil {
					docstringNode = lexDecl
				}
			}

			signature := p.extractFunctionSignature(funcNode, content, language)
			docstring := ""
			if docstringNode != nil {
				docstring = p.extractJSDoc(docstringNode, content)
			}
			isAsync := p.isAsyncFunction(funcNode, content)

			kind := "function"
			if isAsync {
				kind = "async_function"
			}

			symbol := ParsedSymbol{
				Name:      funcName,
				Kind:      kind,
				Signature: fmt.Sprintf("const %s = %s", funcName, signature),
				Span:      nodeToSpan(funcNode),
				Docstring: docstring,
				Node:      funcNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractArrowFunctions extracts arrow function expressions
func (p *JSParser) extractArrowFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte, language string) error {
	// Arrow functions assigned to variables
	arrowQuery := `(variable_declarator
		name: (identifier) @var.name
		value: (arrow_function) @arrow.def)`

	matches, err := p.tsParser.Query(rootNode, arrowQuery, language)
	if err != nil {
		return err
	}

	for _, match := range matches {
		var arrowNode *sitter.Node
		var arrowName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // var.name
				arrowName = capture.Node.Content(content)
			} else if capture.Index == 1 { // arrow.def
				arrowNode = capture.Node
			}
		}

		if arrowNode != nil && arrowName != "" {
			// Get the variable declarator parent for docstring
			varDeclarator := arrowNode.Parent()
			var docstringNode *sitter.Node
			if varDeclarator != nil {
				// Go up to lexical_declaration or variable_declaration
				lexDecl := varDeclarator.Parent()
				if lexDecl != nil {
					docstringNode = lexDecl
				}
			}

			signature := p.extractArrowFunctionSignature(arrowNode, arrowName, content, language)
			docstring := ""
			if docstringNode != nil {
				docstring = p.extractJSDoc(docstringNode, content)
			}
			isAsync := p.isAsyncFunction(arrowNode, content)

			kind := "arrow_function"
			if isAsync {
				kind = "async_arrow_function"
			}

			symbol := ParsedSymbol{
				Name:      arrowName,
				Kind:      kind,
				Signature: signature,
				Span:      nodeToSpan(arrowNode),
				Docstring: docstring,
				Node:      arrowNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractClasses extracts class declarations
func (p *JSParser) extractClasses(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte, language string) error {
	// Use different queries for JavaScript vs TypeScript
	var classQuery string
	if language == "typescript" {
		classQuery = `(class_declaration name: [(type_identifier) (identifier)] @class.name) @class.def`
	} else {
		classQuery = `(class_declaration name: (identifier) @class.name) @class.def`
	}

	matches, err := p.tsParser.Query(rootNode, classQuery, language)
	if err != nil {
		return err
	}

	for _, match := range matches {
		var classNode *sitter.Node
		var className string

		// Captures are returned in document order, but Index tells us which capture it is
		// In the query: (class_declaration name: (identifier) @class.name) @class.def
		// @class.name gets index 0, @class.def gets index 1
		for _, capture := range match.Captures {
			if capture.Index == 0 { // class.name
				className = capture.Node.Content(content)
			} else if capture.Index == 1 { // class.def
				classNode = capture.Node
			}
		}

		if classNode != nil && className != "" {
			methods := p.extractClassMethods(classNode, content, language)
			properties := p.extractClassProperties(classNode, content)
			heritage := p.extractClassHeritage(classNode, content)
			docstring := p.extractJSDoc(classNode, content)

			signature := fmt.Sprintf("class %s", className)
			if heritage != "" {
				signature = fmt.Sprintf("class %s extends %s", className, heritage)
			}

			symbol := ParsedSymbol{
				Name:      className,
				Kind:      "class",
				Signature: signature,
				Span:      nodeToSpan(classNode),
				Docstring: docstring,
				Node:      classNode,
				Children:  append(methods, properties...),
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)

			// Add inheritance dependency if present
			if heritage != "" {
				dependency := ParsedDependency{
					Type:   "extends",
					Source: className,
					Target: heritage,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractClassMethods extracts methods from a class
func (p *JSParser) extractClassMethods(classNode *sitter.Node, content []byte, language string) []ParsedSymbol {
	var methods []ParsedSymbol

	// Find class body
	classBody := findChildByType(classNode, "class_body")
	if classBody == nil {
		return methods
	}

	// Iterate through class body children
	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "method_definition" {
			methodName := ""
			isStatic := false
			isAsync := false

			// Check for static keyword
			for j := 0; j < int(child.ChildCount()); j++ {
				methodChild := child.Child(j)
				if methodChild.Type() == "static" {
					isStatic = true
				} else if methodChild.Type() == "async" {
					isAsync = true
				} else if methodChild.Type() == "property_identifier" {
					methodName = methodChild.Content(content)
				}
			}

			if methodName != "" {
				signature := p.extractMethodSignature(child, content, language)
				docstring := p.extractJSDoc(child, content)

				kind := "method"
				if isStatic {
					kind = "static_method"
				}
				if isAsync {
					kind = "async_method"
				}

				method := ParsedSymbol{
					Name:      methodName,
					Kind:      kind,
					Signature: signature,
					Span:      nodeToSpan(child),
					Docstring: docstring,
					Node:      child,
				}

				methods = append(methods, method)
			}
		}
	}

	return methods
}

// extractClassProperties extracts properties from a class
func (p *JSParser) extractClassProperties(classNode *sitter.Node, content []byte) []ParsedSymbol {
	var properties []ParsedSymbol

	// Find class body
	classBody := findChildByType(classNode, "class_body")
	if classBody == nil {
		return properties
	}

	// Iterate through class body children
	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "field_definition" || child.Type() == "public_field_definition" {
			propertyName := ""
			isStatic := false

			// Check for static keyword and property name
			for j := 0; j < int(child.ChildCount()); j++ {
				propChild := child.Child(j)
				if propChild.Type() == "static" {
					isStatic = true
				} else if propChild.Type() == "property_identifier" {
					propertyName = propChild.Content(content)
				}
			}

			if propertyName != "" {
				signature := strings.TrimSpace(child.Content(content))
				if len(signature) > 100 {
					signature = signature[:100] + "..."
				}

				kind := "property"
				if isStatic {
					kind = "static_property"
				}

				property := ParsedSymbol{
					Name:      propertyName,
					Kind:      kind,
					Signature: signature,
					Span:      nodeToSpan(child),
					Node:      child,
				}

				properties = append(properties, property)
			}
		}
	}

	return properties
}

// extractClassHeritage extracts the parent class name
func (p *JSParser) extractClassHeritage(classNode *sitter.Node, content []byte) string {
	// Find class_heritage node
	heritage := findChildByType(classNode, "class_heritage")
	if heritage == nil {
		return ""
	}

	// Find the identifier or member_expression (parent class name)
	for i := 0; i < int(heritage.ChildCount()); i++ {
		child := heritage.Child(i)
		if child.Type() == "identifier" || child.Type() == "member_expression" {
			return child.Content(content)
		}
	}

	return ""
}

// extractFunctionSignature extracts the function signature
func (p *JSParser) extractFunctionSignature(funcNode *sitter.Node, content []byte, language string) string {
	// Get the first line of the function
	signature := strings.Split(funcNode.Content(content), "\n")[0]

	// For TypeScript, include type annotations
	if language == "typescript" {
		signature = p.extractTypeAnnotations(funcNode, signature, content)
	}

	return strings.TrimSpace(signature)
}

// extractArrowFunctionSignature extracts arrow function signature
func (p *JSParser) extractArrowFunctionSignature(arrowNode *sitter.Node, name string, content []byte, language string) string {
	arrowText := arrowNode.Content(content)

	// Get the first line or up to the arrow
	lines := strings.Split(arrowText, "\n")
	signature := lines[0]

	// For TypeScript, include type annotations
	if language == "typescript" {
		signature = p.extractTypeAnnotations(arrowNode, signature, content)
	}

	return fmt.Sprintf("const %s = %s", name, strings.TrimSpace(signature))
}

// extractMethodSignature extracts method signature
func (p *JSParser) extractMethodSignature(methodNode *sitter.Node, content []byte, language string) string {
	signature := strings.Split(methodNode.Content(content), "\n")[0]

	// For TypeScript, include type annotations
	if language == "typescript" {
		signature = p.extractTypeAnnotations(methodNode, signature, content)
	}

	return strings.TrimSpace(signature)
}

// extractTypeAnnotations extracts TypeScript type annotations
func (p *JSParser) extractTypeAnnotations(node *sitter.Node, signature string, content []byte) string {
	// Look for type_annotation nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_annotation" {
			typeText := child.Content(content)
			// If not already in signature, append it
			if !strings.Contains(signature, typeText) {
				signature += " " + typeText
			}
		}
	}

	return signature
}

// extractJSDoc extracts JSDoc comments before a node
func (p *JSParser) extractJSDoc(node *sitter.Node, content []byte) string {
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

			// Handle JSDoc comments (/** ... */)
			if strings.HasPrefix(commentText, "/**") {
				commentText = strings.TrimPrefix(commentText, "/**")
				commentText = strings.TrimSuffix(commentText, "*/")
				commentText = strings.TrimSpace(commentText)

				// Clean up JSDoc formatting
				lines := strings.Split(commentText, "\n")
				var cleanLines []string
				for _, line := range lines {
					line = strings.TrimSpace(line)
					line = strings.TrimPrefix(line, "*")
					line = strings.TrimSpace(line)
					if line != "" {
						cleanLines = append(cleanLines, line)
					}
				}
				commentText = strings.Join(cleanLines, "\n")
			} else {
				// Handle single-line comments
				commentText = strings.TrimPrefix(commentText, "//")
				commentText = strings.TrimSpace(commentText)
			}

			comments = append([]string{commentText}, comments...)
		} else if sibling.Type() != "comment" {
			break
		}
	}

	return strings.Join(comments, "\n")
}

// isAsyncFunction checks if a function is async
func (p *JSParser) isAsyncFunction(funcNode *sitter.Node, content []byte) bool {
	// Check if parent or node itself has async keyword
	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child.Type() == "async" {
			return true
		}
	}

	// Check parent node
	parent := funcNode.Parent()
	if parent != nil {
		for i := 0; i < int(parent.ChildCount()); i++ {
			child := parent.Child(i)
			if child.Type() == "async" {
				return true
			}
		}
	}

	return false
}

// isGeneratorFunction checks if a function is a generator
func (p *JSParser) isGeneratorFunction(funcNode *sitter.Node, content []byte) bool {
	// Look for generator marker (*)
	funcText := funcNode.Content(content)
	return strings.Contains(funcText, "function*")
}

// extractCallRelationships extracts function call relationships
func (p *JSParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte, language string) error {
	// Query for call expressions
	query := `(call_expression function: [(identifier) (member_expression)] @call.target)`

	matches, err := p.tsParser.Query(rootNode, query, language)
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			callTarget := capture.Node.Content(content)

			// Find the containing function for this call
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
func (p *JSParser) findContainingFunction(node *sitter.Node, parsedFile *ParsedFile) string {
	current := node.Parent()

	for current != nil {
		// Check if this is a function, arrow function, or method
		nodeType := current.Type()
		if nodeType == "function_declaration" ||
			nodeType == "function_expression" ||
			nodeType == "arrow_function" ||
			nodeType == "method_definition" {
			// Find the matching symbol in our parsed symbols
			for _, symbol := range parsedFile.Symbols {
				if symbol.Node == current {
					return symbol.Name
				}
				// Also check children for nested symbols
				for _, child := range symbol.Children {
					if child.Node == current {
						return child.Name
					}
				}
			}
		}
		current = current.Parent()
	}

	return ""
}
