package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// CParser parses C source code using Tree-sitter
type CParser struct {
	tsParser *TreeSitterParser
}

// NewCParser creates a new C parser
func NewCParser(tsParser *TreeSitterParser) *CParser {
	return &CParser{
		tsParser: tsParser,
	}
}

// Parse parses a C file and extracts symbols and dependencies
func (p *CParser) Parse(file ScannedFile) (*ParsedFile, error) {
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
	rootNode, parseErr := p.tsParser.Parse(content, "c")

	parsedFile := &ParsedFile{
		Path:     file.Path,
		Language: "c",
		Content:  content,
		RootNode: rootNode,
	}

	// If we have no root node at all, return error immediately
	if rootNode == nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to parse C file: %v", parseErr),
			Type:    "parse",
		}
	}

	// Determine if this is a header or implementation file
	isHeader := strings.HasSuffix(file.Path, ".h")

	// Extract includes
	if err := p.extractIncludes(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract functions (both declarations and definitions)
	if err := p.extractFunctions(rootNode, parsedFile, content, isHeader); err != nil {
		// Non-fatal, continue
	}

	// Extract structs
	if err := p.extractStructs(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract unions
	if err := p.extractUnions(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract enums
	if err := p.extractEnums(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract typedefs
	if err := p.extractTypedefs(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract macros
	if err := p.extractMacros(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract global variables
	if err := p.extractGlobalVariables(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract call relationships (only for implementation files)
	if !isHeader {
		if err := p.extractCallRelationships(rootNode, parsedFile, content); err != nil {
			// Non-fatal, continue
		}
	}

	// Return parse error if there was one, but with partial results
	if parseErr != nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("syntax error in C file: %v", parseErr),
			Type:    "parse",
		}
	}

	return parsedFile, nil
}

// Helper functions for node traversal and span conversion

// extractSignature extracts a clean signature from a node
func (p *CParser) extractSignature(node *sitter.Node, content []byte) string {
	// Get the first line or up to opening brace
	nodeText := node.Content(content)
	lines := strings.Split(nodeText, "\n")

	signature := ""
	for _, line := range lines {
		signature += line
		// Stop at opening brace or semicolon
		if strings.Contains(line, "{") || strings.Contains(line, ";") {
			// Remove the brace/semicolon and everything after
			if idx := strings.Index(signature, "{"); idx != -1 {
				signature = signature[:idx]
			}
			if idx := strings.Index(signature, ";"); idx != -1 {
				signature = signature[:idx+1]
			}
			break
		}
		signature += " "
	}

	return strings.TrimSpace(signature)
}

// findContainingFunction finds the name of the function containing a node
func (p *CParser) findContainingFunction(node *sitter.Node, parsedFile *ParsedFile) string {
	current := node.Parent()

	for current != nil {
		// Check if this is a function definition
		if current.Type() == "function_definition" {
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

// findPairedFile locates the corresponding .h or .c file
func (p *CParser) findPairedFile(currentPath string) string {
	dir := filepath.Dir(currentPath)
	baseName := strings.TrimSuffix(filepath.Base(currentPath), filepath.Ext(currentPath))

	if strings.HasSuffix(currentPath, ".h") {
		// Look for implementation file
		implPath := filepath.Join(dir, baseName+".c")
		if _, err := os.Stat(implPath); err == nil {
			return implPath
		}
	} else {
		// Look for header file
		headerPath := filepath.Join(dir, baseName+".h")
		if _, err := os.Stat(headerPath); err == nil {
			return headerPath
		}
	}

	return ""
}

// extractIncludes extracts #include statements with system/local classification
func (p *CParser) extractIncludes(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for preproc_include directives
	includeQuery := `(preproc_include) @include.decl`

	matches, err := p.tsParser.Query(rootNode, includeQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			includeNode := capture.Node

			// Look for system_lib_string (<...>) or string_literal ("...")
			var includePath string
			var isSystem bool

			for i := 0; i < int(includeNode.ChildCount()); i++ {
				child := includeNode.Child(i)
				if child.Type() == "system_lib_string" {
					// System include like <stdio.h>
					text := child.Content(content)
					includePath = strings.Trim(text, "<>")
					isSystem = true
				} else if child.Type() == "string_literal" {
					// Local include like "myheader.h"
					text := child.Content(content)
					includePath = strings.Trim(text, "\"")
					isSystem = false
				}
			}

			if includePath != "" {
				dependency := ParsedDependency{
					Type:         "import",
					Source:       "",
					Target:       includePath,
					TargetModule: includePath,
					IsExternal:   !isSystem && p.isExternalImport(includePath),
				}

				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractFunctions extracts function declarations and definitions
func (p *CParser) extractFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte, isHeader bool) error {
	// Query for function declarations (in headers)
	declQuery := `(declaration
		declarator: (function_declarator
			declarator: (identifier) @func.name)) @func.decl`

	// Query for function definitions (in implementation files)
	defQuery := `(function_definition
		declarator: (function_declarator
			declarator: (identifier) @func.name)) @func.def`

	// Extract declarations
	declMatches, err := p.tsParser.Query(rootNode, declQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range declMatches {
		var funcNode *sitter.Node
		var funcName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // func.name
				funcName = capture.Node.Content(content)
			} else if capture.Index == 1 { // func.decl
				funcNode = capture.Node
			}
		}

		if funcNode != nil && funcName != "" {
			signature := p.extractSignature(funcNode, content)
			docstring := p.extractDoxygenComments(funcNode, content)

			symbol := ParsedSymbol{
				Name:      funcName,
				Kind:      "function_declaration",
				Signature: signature,
				Span:      nodeToSpan(funcNode),
				Docstring: docstring,
				Node:      funcNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	// Extract definitions
	defMatches, err := p.tsParser.Query(rootNode, defQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range defMatches {
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
			signature := p.extractSignature(funcNode, content)
			docstring := p.extractDoxygenComments(funcNode, content)

			// Check if this is a static function
			isStatic := p.isStaticFunction(funcNode, content)
			kind := "function"
			if isStatic {
				kind = "static_function"
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

	return nil
}

// isStaticFunction checks if a function has static storage class
func (p *CParser) isStaticFunction(funcNode *sitter.Node, content []byte) bool {
	// Look for storage_class_specifier with "static"
	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child.Type() == "storage_class_specifier" {
			if strings.Contains(child.Content(content), "static") {
				return true
			}
		}
	}
	return false
}

// extractStructs extracts struct declarations with fields
func (p *CParser) extractStructs(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for struct specifiers
	structQuery := `(struct_specifier
		name: (type_identifier) @struct.name
		body: (field_declaration_list) @struct.body) @struct.def`

	matches, err := p.tsParser.Query(rootNode, structQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var structNode *sitter.Node
		var structName string
		var structBody *sitter.Node

		for _, capture := range match.Captures {
			if capture.Index == 0 { // struct.name
				structName = capture.Node.Content(content)
			} else if capture.Index == 1 { // struct.body
				structBody = capture.Node
			} else if capture.Index == 2 { // struct.def
				structNode = capture.Node
			}
		}

		if structNode != nil && structName != "" {
			fields := p.extractStructFields(structBody, content)
			docstring := p.extractDoxygenComments(structNode, content)
			signature := fmt.Sprintf("struct %s", structName)

			symbol := ParsedSymbol{
				Name:      structName,
				Kind:      "struct",
				Signature: signature,
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

// extractStructFields extracts fields from a struct body
func (p *CParser) extractStructFields(bodyNode *sitter.Node, content []byte) []ParsedSymbol {
	if bodyNode == nil {
		return nil
	}

	var fields []ParsedSymbol

	// Iterate through field declarations
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)
		if child.Type() == "field_declaration" {
			// Extract all field identifiers (can be multiple in one declaration)
			var fieldNames []string
			var fieldType string

			// First pass: get the type
			for j := 0; j < int(child.ChildCount()); j++ {
				fieldChild := child.Child(j)
				if fieldChild.Type() == "type_identifier" || fieldChild.Type() == "primitive_type" {
					if fieldType == "" {
						fieldType = fieldChild.Content(content)
					}
				} else if fieldChild.Type() == "struct_specifier" {
					// Handle nested struct types
					fieldType = "struct"
				}
			}

			// Second pass: get all field names
			for j := 0; j < int(child.ChildCount()); j++ {
				fieldChild := child.Child(j)
				if fieldChild.Type() == "field_identifier" {
					fieldNames = append(fieldNames, fieldChild.Content(content))
				} else if fieldChild.Type() == "array_declarator" {
					// Handle array fields like char name[50]
					for k := 0; k < int(fieldChild.ChildCount()); k++ {
						arrayChild := fieldChild.Child(k)
						if arrayChild.Type() == "field_identifier" {
							fieldNames = append(fieldNames, arrayChild.Content(content))
						}
					}
				}
			}

			// Create a symbol for each field
			for _, fieldName := range fieldNames {
				signature := fmt.Sprintf("%s %s", fieldType, fieldName)
				field := ParsedSymbol{
					Name:      fieldName,
					Kind:      "field",
					Signature: signature,
					Span:      nodeToSpan(child),
					Node:      child,
				}
				fields = append(fields, field)
			}
		}
	}

	return fields
}

// extractUnions extracts union declarations
func (p *CParser) extractUnions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for union specifiers
	unionQuery := `(union_specifier
		name: (type_identifier) @union.name
		body: (field_declaration_list) @union.body) @union.def`

	matches, err := p.tsParser.Query(rootNode, unionQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var unionNode *sitter.Node
		var unionName string
		var unionBody *sitter.Node

		for _, capture := range match.Captures {
			if capture.Index == 0 { // union.name
				unionName = capture.Node.Content(content)
			} else if capture.Index == 1 { // union.body
				unionBody = capture.Node
			} else if capture.Index == 2 { // union.def
				unionNode = capture.Node
			}
		}

		if unionNode != nil && unionName != "" {
			fields := p.extractStructFields(unionBody, content) // Reuse struct field extraction
			docstring := p.extractDoxygenComments(unionNode, content)
			signature := fmt.Sprintf("union %s", unionName)

			symbol := ParsedSymbol{
				Name:      unionName,
				Kind:      "union",
				Signature: signature,
				Span:      nodeToSpan(unionNode),
				Docstring: docstring,
				Node:      unionNode,
				Children:  fields,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractEnums extracts enum declarations
func (p *CParser) extractEnums(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for enum specifiers
	enumQuery := `(enum_specifier
		name: (type_identifier) @enum.name
		body: (enumerator_list) @enum.body) @enum.def`

	matches, err := p.tsParser.Query(rootNode, enumQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var enumNode *sitter.Node
		var enumName string
		var enumBody *sitter.Node

		for _, capture := range match.Captures {
			if capture.Index == 0 { // enum.name
				enumName = capture.Node.Content(content)
			} else if capture.Index == 1 { // enum.body
				enumBody = capture.Node
			} else if capture.Index == 2 { // enum.def
				enumNode = capture.Node
			}
		}

		if enumNode != nil && enumName != "" {
			enumerators := p.extractEnumerators(enumBody, content)
			docstring := p.extractDoxygenComments(enumNode, content)
			signature := fmt.Sprintf("enum %s", enumName)

			symbol := ParsedSymbol{
				Name:      enumName,
				Kind:      "enum",
				Signature: signature,
				Span:      nodeToSpan(enumNode),
				Docstring: docstring,
				Node:      enumNode,
				Children:  enumerators,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractEnumerators extracts enumerator constants from an enum body
func (p *CParser) extractEnumerators(bodyNode *sitter.Node, content []byte) []ParsedSymbol {
	if bodyNode == nil {
		return nil
	}

	var enumerators []ParsedSymbol

	// Iterate through enumerators
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)
		if child.Type() == "enumerator" {
			// Extract enumerator name
			var enumName string

			for j := 0; j < int(child.ChildCount()); j++ {
				enumChild := child.Child(j)
				if enumChild.Type() == "identifier" {
					enumName = enumChild.Content(content)
					break
				}
			}

			if enumName != "" {
				signature := child.Content(content)
				enumerator := ParsedSymbol{
					Name:      enumName,
					Kind:      "enum_constant",
					Signature: signature,
					Span:      nodeToSpan(child),
					Node:      child,
				}
				enumerators = append(enumerators, enumerator)
			}
		}
	}

	return enumerators
}

// extractTypedefs extracts typedef declarations
func (p *CParser) extractTypedefs(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for type definitions
	typedefQuery := `(type_definition
		declarator: (type_identifier) @typedef.name) @typedef.def`

	matches, err := p.tsParser.Query(rootNode, typedefQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var typedefNode *sitter.Node
		var typedefName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // typedef.name
				typedefName = capture.Node.Content(content)
			} else if capture.Index == 1 { // typedef.def
				typedefNode = capture.Node
			}
		}

		if typedefNode != nil && typedefName != "" {
			signature := p.extractSignature(typedefNode, content)
			docstring := p.extractDoxygenComments(typedefNode, content)

			symbol := ParsedSymbol{
				Name:      typedefName,
				Kind:      "typedef",
				Signature: signature,
				Span:      nodeToSpan(typedefNode),
				Docstring: docstring,
				Node:      typedefNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractMacros extracts #define macros
func (p *CParser) extractMacros(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for preproc_def (simple macros) and preproc_function_def (function-like macros)
	macroQuery := `(preproc_def
		name: (identifier) @macro.name) @macro.def`

	matches, err := p.tsParser.Query(rootNode, macroQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var macroNode *sitter.Node
		var macroName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // macro.name
				macroName = capture.Node.Content(content)
			} else if capture.Index == 1 { // macro.def
				macroNode = capture.Node
			}
		}

		if macroNode != nil && macroName != "" {
			signature := strings.TrimSpace(macroNode.Content(content))
			docstring := p.extractDoxygenComments(macroNode, content)

			symbol := ParsedSymbol{
				Name:      macroName,
				Kind:      "macro",
				Signature: signature,
				Span:      nodeToSpan(macroNode),
				Docstring: docstring,
				Node:      macroNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	// Query for function-like macros
	funcMacroQuery := `(preproc_function_def
		name: (identifier) @macro.name) @macro.def`

	matches, err = p.tsParser.Query(rootNode, funcMacroQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var macroNode *sitter.Node
		var macroName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // macro.name
				macroName = capture.Node.Content(content)
			} else if capture.Index == 1 { // macro.def
				macroNode = capture.Node
			}
		}

		if macroNode != nil && macroName != "" {
			signature := strings.TrimSpace(macroNode.Content(content))
			docstring := p.extractDoxygenComments(macroNode, content)

			symbol := ParsedSymbol{
				Name:      macroName,
				Kind:      "function_macro",
				Signature: signature,
				Span:      nodeToSpan(macroNode),
				Docstring: docstring,
				Node:      macroNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractGlobalVariables extracts global variable declarations
func (p *CParser) extractGlobalVariables(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Recursively process all declaration nodes
	var processNode func(*sitter.Node)
	processNode = func(node *sitter.Node) {
		// Look for declarations that are not functions
		if node.Type() == "declaration" {
			// Check if this is a variable declaration (not a function declaration)
			isFunctionDecl := false
			for j := 0; j < int(node.ChildCount()); j++ {
				declChild := node.Child(j)
				if declChild.Type() == "function_declarator" {
					isFunctionDecl = true
					break
				}
			}

			if !isFunctionDecl {
				// Extract variable name
				var varNames []string
				var varType string

				// Helper function to recursively find identifiers
				var findIdentifiers func(*sitter.Node)
				findIdentifiers = func(n *sitter.Node) {
					if n.Type() == "identifier" {
						// Make sure this identifier is not part of a type
						parent := n.Parent()
						if parent != nil && parent.Type() != "type_identifier" {
							varNames = append(varNames, n.Content(content))
						}
					}
					for k := 0; k < int(n.ChildCount()); k++ {
						findIdentifiers(n.Child(k))
					}
				}

				for j := 0; j < int(node.ChildCount()); j++ {
					declChild := node.Child(j)
					if declChild.Type() == "init_declarator" {
						// Look for identifier in init_declarator
						findIdentifiers(declChild)
					} else if declChild.Type() == "identifier" {
						// Direct identifier (for simple declarations)
						varNames = append(varNames, declChild.Content(content))
					} else if declChild.Type() == "type_identifier" || declChild.Type() == "primitive_type" {
						if varType == "" {
							varType = declChild.Content(content)
						}
					}
				}

				// Check if this is an extern variable
				isExtern := p.isExternVariable(node, content)

				for _, varName := range varNames {
					if varName != "" {
						signature := p.extractSignature(node, content)
						docstring := p.extractDoxygenComments(node, content)

						kind := "global_variable"
						if isExtern {
							kind = "extern_variable"
						}

						symbol := ParsedSymbol{
							Name:      varName,
							Kind:      kind,
							Signature: signature,
							Span:      nodeToSpan(node),
							Docstring: docstring,
							Node:      node,
						}

						parsedFile.Symbols = append(parsedFile.Symbols, symbol)
					}
				}
			}
		}

		// Recursively process children
		for i := 0; i < int(node.ChildCount()); i++ {
			processNode(node.Child(i))
		}
	}

	processNode(rootNode)
	return nil
}

// isExternVariable checks if a variable has extern storage class
func (p *CParser) isExternVariable(declNode *sitter.Node, content []byte) bool {
	// Look for storage_class_specifier with "extern"
	for i := 0; i < int(declNode.ChildCount()); i++ {
		child := declNode.Child(i)
		if child.Type() == "storage_class_specifier" {
			if strings.Contains(child.Content(content), "extern") {
				return true
			}
		}
	}
	return false
}

// extractDoxygenComments extracts Doxygen documentation
func (p *CParser) extractDoxygenComments(node *sitter.Node, content []byte) string {
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

			// Handle Doxygen-style comments (/** ... */ or ///)
			if strings.HasPrefix(commentText, "/**") || strings.HasPrefix(commentText, "/*!") {
				commentText = strings.TrimPrefix(commentText, "/**")
				commentText = strings.TrimPrefix(commentText, "/*!")
				commentText = strings.TrimSuffix(commentText, "*/")
				commentText = strings.TrimSpace(commentText)

				// Clean up documentation formatting
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
			} else if strings.HasPrefix(commentText, "///") || strings.HasPrefix(commentText, "//!") {
				commentText = strings.TrimPrefix(commentText, "///")
				commentText = strings.TrimPrefix(commentText, "//!")
				commentText = strings.TrimSpace(commentText)
			} else {
				// Handle regular comments
				commentText = strings.TrimPrefix(commentText, "//")
				commentText = strings.TrimPrefix(commentText, "/*")
				commentText = strings.TrimSuffix(commentText, "*/")
				commentText = strings.TrimSpace(commentText)
			}

			comments = append([]string{commentText}, comments...)
		} else if sibling.Type() != "comment" {
			break
		}
	}

	return strings.Join(comments, "\n")
}

// extractCallRelationships extracts function calls
func (p *CParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for call expressions
	callQuery := `(call_expression
		function: (identifier) @call.target)`

	matches, err := p.tsParser.Query(rootNode, callQuery, "c")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			callTarget := capture.Node.Content(content)

			// Find the containing function for this call
			caller := p.findContainingFunction(capture.Node, parsedFile)
			if caller != "" {
				// Resolve the call to a declaration if possible
				resolvedTarget := p.resolveCallToDeclaration(callTarget, parsedFile)

				dependency := ParsedDependency{
					Type:   "call",
					Source: caller,
					Target: resolvedTarget,
				}

				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	// Also handle function pointer calls
	funcPtrQuery := `(call_expression
		function: (pointer_expression) @call.target)`

	matches, err = p.tsParser.Query(rootNode, funcPtrQuery, "c")
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

// matchDeclarationToImplementation matches function declarations with definitions
func (p *CParser) matchDeclarationToImplementation(headerFile, implFile *ParsedFile) {
	// Find function declarations in header
	for _, headerSymbol := range headerFile.Symbols {
		if headerSymbol.Kind == "function_declaration" {
			// Find matching function definition in implementation file
			for _, implSymbol := range implFile.Symbols {
				if (implSymbol.Kind == "function" || implSymbol.Kind == "static_function") &&
					implSymbol.Name == headerSymbol.Name {

					// Compare signatures to ensure they match
					headerSig := p.normalizeSignature(headerSymbol.Signature)
					implSig := p.normalizeSignature(implSymbol.Signature)

					if p.signaturesMatch(headerSig, implSig) {
						// Create implements_declaration edge
						dependency := ParsedDependency{
							Type:   "implements_declaration",
							Source: implSymbol.Name,
							Target: headerSymbol.Name,
						}
						implFile.Dependencies = append(implFile.Dependencies, dependency)
					}
				}
			}
		}
	}

	// Create implements_header edge between files
	if len(headerFile.Symbols) > 0 && len(implFile.Symbols) > 0 {
		dependency := ParsedDependency{
			Type:   "implements_header",
			Source: implFile.Path,
			Target: headerFile.Path,
		}
		implFile.Dependencies = append(implFile.Dependencies, dependency)
	}
}

// extractFunctionSignature creates a comparable function signature
func (p *CParser) extractFunctionSignature(funcNode *sitter.Node, content []byte) string {
	// Extract the function signature components
	var returnType string
	var funcName string
	var params []string

	// Find function_declarator
	var funcDeclarator *sitter.Node
	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child.Type() == "function_declarator" {
			funcDeclarator = child
		} else if child.Type() == "type_identifier" || child.Type() == "primitive_type" {
			if returnType == "" {
				returnType = child.Content(content)
			}
		}
	}

	if funcDeclarator != nil {
		// Extract function name
		for i := 0; i < int(funcDeclarator.ChildCount()); i++ {
			child := funcDeclarator.Child(i)
			if child.Type() == "identifier" {
				funcName = child.Content(content)
			} else if child.Type() == "parameter_list" {
				// Extract parameters
				for j := 0; j < int(child.ChildCount()); j++ {
					param := child.Child(j)
					if param.Type() == "parameter_declaration" {
						paramText := strings.TrimSpace(param.Content(content))
						params = append(params, paramText)
					}
				}
			}
		}
	}

	// Build signature
	signature := fmt.Sprintf("%s %s(%s)", returnType, funcName, strings.Join(params, ", "))
	return signature
}

// normalizeSignature normalizes a function signature for comparison
func (p *CParser) normalizeSignature(signature string) string {
	// Remove extra whitespace
	signature = strings.Join(strings.Fields(signature), " ")

	// Remove parameter names, keeping only types
	// This is a simplified approach - a full implementation would parse the signature properly
	signature = strings.TrimSpace(signature)

	return signature
}

// signaturesMatch compares two normalized signatures
func (p *CParser) signaturesMatch(sig1, sig2 string) bool {
	// Extract function name and parameter types from both signatures
	name1, params1 := p.parseSignature(sig1)
	name2, params2 := p.parseSignature(sig2)

	// Names must match
	if name1 != name2 {
		return false
	}

	// Parameter counts must match
	if len(params1) != len(params2) {
		return false
	}

	// Parameter types must match (ignoring parameter names)
	for i := range params1 {
		type1 := p.extractParameterType(params1[i])
		type2 := p.extractParameterType(params2[i])

		if type1 != type2 {
			return false
		}
	}

	return true
}

// parseSignature extracts function name and parameters from a signature
func (p *CParser) parseSignature(signature string) (string, []string) {
	// Find the function name (between return type and opening parenthesis)
	openParen := strings.Index(signature, "(")
	if openParen == -1 {
		return "", nil
	}

	// Extract everything before the parenthesis
	beforeParen := strings.TrimSpace(signature[:openParen])
	parts := strings.Fields(beforeParen)

	// Last part is the function name
	var funcName string
	if len(parts) > 0 {
		funcName = parts[len(parts)-1]
	}

	// Extract parameters (between parentheses)
	closeParen := strings.LastIndex(signature, ")")
	if closeParen == -1 {
		return funcName, nil
	}

	paramsStr := signature[openParen+1 : closeParen]
	if strings.TrimSpace(paramsStr) == "" || strings.TrimSpace(paramsStr) == "void" {
		return funcName, []string{}
	}

	// Split parameters by comma
	params := strings.Split(paramsStr, ",")
	for i := range params {
		params[i] = strings.TrimSpace(params[i])
	}

	return funcName, params
}

// extractParameterType extracts the type from a parameter declaration
func (p *CParser) extractParameterType(param string) string {
	// Remove parameter name, keeping only the type
	// This is a simplified approach - handles basic cases
	parts := strings.Fields(param)

	if len(parts) == 0 {
		return ""
	}

	// For simple types like "int x", return "int"
	// For pointer types like "int *x", return "int *"
	// For complex types, this is a heuristic

	// If last part doesn't contain *, it's likely the parameter name
	if len(parts) > 1 && !strings.Contains(parts[len(parts)-1], "*") {
		// Remove last part (parameter name)
		return strings.Join(parts[:len(parts)-1], " ")
	}

	// Otherwise, return the whole thing
	return param
}

// resolveCallToDeclaration links calls to header declarations
func (p *CParser) resolveCallToDeclaration(call string, currentFile *ParsedFile) string {
	// First, check if the call target is defined in the current file
	for _, symbol := range currentFile.Symbols {
		if symbol.Name == call {
			// Found in current file
			return call
		}
	}

	// Check if there's a paired header file
	pairedHeader := p.findPairedFile(currentFile.Path)
	if pairedHeader != "" {
		// In a real implementation, we would load and parse the header file
		// For now, we just return the call as-is
		// The indexer can handle cross-file resolution later
		return call
	}

	// Check included headers (from dependencies)
	for _, dep := range currentFile.Dependencies {
		if dep.Type == "import" {
			// In a real implementation, we would check if the function is declared
			// in the included header
			// For now, we just return the call as-is
			return call
		}
	}

	// Default: return the call as-is
	return call
}

// isExternalImport classifies includes (system includes as external)
func (p *CParser) isExternalImport(includePath string) bool {
	// Standard C library headers are considered internal (part of the language)
	stdHeaders := map[string]bool{
		"stdio.h":    true,
		"stdlib.h":   true,
		"string.h":   true,
		"math.h":     true,
		"time.h":     true,
		"ctype.h":    true,
		"stddef.h":   true,
		"stdint.h":   true,
		"stdbool.h":  true,
		"assert.h":   true,
		"errno.h":    true,
		"limits.h":   true,
		"float.h":    true,
		"stdarg.h":   true,
		"setjmp.h":   true,
		"signal.h":   true,
		"locale.h":   true,
		"wchar.h":    true,
		"wctype.h":   true,
		"complex.h":  true,
		"fenv.h":     true,
		"inttypes.h": true,
		"iso646.h":   true,
		"stdalign.h": true,
		"stdatomic.h": true,
		"stdnoreturn.h": true,
		"threads.h":  true,
		"uchar.h":    true,
	}

	// Check if it's a standard header
	if stdHeaders[includePath] {
		return false // Standard library is internal
	}

	// POSIX and system headers (usually in angle brackets)
	// These are considered external
	systemPrefixes := []string{
		"sys/",
		"linux/",
		"unix/",
		"windows.h",
		"pthread.h",
		"unistd.h",
		"fcntl.h",
		"dirent.h",
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(includePath, prefix) || includePath == prefix {
			return true
		}
	}

	// If it's a local header (contains .h and no path separator), it's internal
	if strings.HasSuffix(includePath, ".h") && !strings.Contains(includePath, "/") {
		return false
	}

	// Everything else is considered external
	return true
}
