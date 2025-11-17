package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// CppParser parses C++ source code using Tree-sitter
type CppParser struct {
	tsParser *TreeSitterParser
}

// NewCppParser creates a new C++ parser
func NewCppParser(tsParser *TreeSitterParser) *CppParser {
	return &CppParser{
		tsParser: tsParser,
	}
}

// Parse parses a C++ file and extracts symbols and dependencies
func (p *CppParser) Parse(file ScannedFile) (*ParsedFile, error) {
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
	rootNode, parseErr := p.tsParser.Parse(content, "cpp")

	parsedFile := &ParsedFile{
		Path:     file.Path,
		Language: "cpp",
		Content:  content,
		RootNode: rootNode,
	}

	// If we have no root node at all, return error immediately
	if rootNode == nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to parse C++ file: %v", parseErr),
			Type:    "parse",
		}
	}

	// Determine if this is a header or implementation file
	isHeader := strings.HasSuffix(file.Path, ".h") ||
		strings.HasSuffix(file.Path, ".hpp") ||
		strings.HasSuffix(file.Path, ".hh") ||
		strings.HasSuffix(file.Path, ".hxx")

	// Extract includes
	if err := p.extractIncludes(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract namespaces
	if err := p.extractNamespaces(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract classes
	if err := p.extractClasses(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract templates
	if err := p.extractTemplates(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract functions (both declarations and definitions)
	if err := p.extractFunctions(rootNode, parsedFile, content, isHeader); err != nil {
		// Non-fatal, continue
	}

	// Extract methods
	if err := p.extractMethods(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract operators
	if err := p.extractOperators(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract virtual methods
	if err := p.extractVirtualMethods(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract call relationships (only for implementation files)
	if !isHeader {
		if err := p.extractCallRelationships(rootNode, parsedFile, content); err != nil {
			// Non-fatal, continue
		}
	}

	// Extract inheritance relationships
	if err := p.extractInheritance(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Return parse error if there was one, but with partial results
	if parseErr != nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("syntax error in C++ file: %v", parseErr),
			Type:    "parse",
		}
	}

	return parsedFile, nil
}

// Helper functions for node traversal and span conversion

// extractSignature extracts a clean signature from a node
func (p *CppParser) extractSignature(node *sitter.Node, content []byte) string {
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
func (p *CppParser) findContainingFunction(node *sitter.Node, parsedFile *ParsedFile) string {
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

// findPairedFile locates the corresponding header or implementation file
func (p *CppParser) findPairedFile(currentPath string) string {
	dir := filepath.Dir(currentPath)
	baseName := strings.TrimSuffix(filepath.Base(currentPath), filepath.Ext(currentPath))

	// Check if this is a header file
	if strings.HasSuffix(currentPath, ".h") ||
		strings.HasSuffix(currentPath, ".hpp") ||
		strings.HasSuffix(currentPath, ".hh") ||
		strings.HasSuffix(currentPath, ".hxx") {
		// Look for implementation file
		extensions := []string{".cpp", ".cc", ".cxx", ".c++"}
		for _, ext := range extensions {
			implPath := filepath.Join(dir, baseName+ext)
			if _, err := os.Stat(implPath); err == nil {
				return implPath
			}
		}
	} else {
		// Look for header file
		extensions := []string{".hpp", ".hh", ".hxx", ".h"}
		for _, ext := range extensions {
			headerPath := filepath.Join(dir, baseName+ext)
			if _, err := os.Stat(headerPath); err == nil {
				return headerPath
			}
		}
	}

	return ""
}

// extractDoxygenComments extracts Doxygen documentation
func (p *CppParser) extractDoxygenComments(node *sitter.Node, content []byte) string {
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

// extractIncludes extracts #include statements
func (p *CppParser) extractIncludes(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for preproc_include directives
	includeQuery := `(preproc_include) @include.decl`

	matches, err := p.tsParser.Query(rootNode, includeQuery, "cpp")
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
					// System include like <iostream>
					text := child.Content(content)
					includePath = strings.Trim(text, "<>")
					isSystem = true
				} else if child.Type() == "string_literal" {
					// Local include like "myheader.hpp"
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
					IsExternal:   p.isExternalImport(includePath, isSystem),
				}

				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractNamespaces extracts namespace declarations
func (p *CppParser) extractNamespaces(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for namespace definitions
	namespaceQuery := `(namespace_definition
		name: (identifier) @namespace.name) @namespace.def`

	matches, err := p.tsParser.Query(rootNode, namespaceQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var namespaceNode *sitter.Node
		var namespaceName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // namespace.name
				namespaceName = capture.Node.Content(content)
			} else if capture.Index == 1 { // namespace.def
				namespaceNode = capture.Node
			}
		}

		if namespaceNode != nil && namespaceName != "" {
			docstring := p.extractDoxygenComments(namespaceNode, content)
			signature := fmt.Sprintf("namespace %s", namespaceName)

			symbol := ParsedSymbol{
				Name:      namespaceName,
				Kind:      "namespace",
				Signature: signature,
				Span:      nodeToSpan(namespaceNode),
				Docstring: docstring,
				Node:      namespaceNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractClasses extracts class declarations with member functions and variables
func (p *CppParser) extractClasses(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for class specifiers
	classQuery := `(class_specifier
		name: (type_identifier) @class.name
		body: (field_declaration_list) @class.body) @class.def`

	matches, err := p.tsParser.Query(rootNode, classQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var classNode *sitter.Node
		var className string
		var classBody *sitter.Node

		for _, capture := range match.Captures {
			if capture.Index == 0 { // class.name
				className = capture.Node.Content(content)
			} else if capture.Index == 1 { // class.body
				classBody = capture.Node
			} else if capture.Index == 2 { // class.def
				classNode = capture.Node
			}
		}

		if classNode != nil && className != "" {
			// Extract members (methods and fields)
			members := p.extractClassMembers(classBody, content)
			docstring := p.extractDoxygenComments(classNode, content)
			signature := fmt.Sprintf("class %s", className)

			symbol := ParsedSymbol{
				Name:      className,
				Kind:      "class",
				Signature: signature,
				Span:      nodeToSpan(classNode),
				Docstring: docstring,
				Node:      classNode,
				Children:  members,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	// Also handle struct specifiers (similar to classes in C++)
	structQuery := `(struct_specifier
		name: (type_identifier) @struct.name
		body: (field_declaration_list) @struct.body) @struct.def`

	matches, err = p.tsParser.Query(rootNode, structQuery, "cpp")
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
			members := p.extractClassMembers(structBody, content)
			docstring := p.extractDoxygenComments(structNode, content)
			signature := fmt.Sprintf("struct %s", structName)

			symbol := ParsedSymbol{
				Name:      structName,
				Kind:      "struct",
				Signature: signature,
				Span:      nodeToSpan(structNode),
				Docstring: docstring,
				Node:      structNode,
				Children:  members,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractClassMembers extracts member functions and variables from a class body
func (p *CppParser) extractClassMembers(bodyNode *sitter.Node, content []byte) []ParsedSymbol {
	if bodyNode == nil {
		return nil
	}

	var members []ParsedSymbol

	// Iterate through field declarations
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)

		// Extract member functions
		if child.Type() == "function_definition" {
			funcName := p.extractFunctionName(child, content)
			if funcName != "" {
				signature := p.extractSignature(child, content)
				docstring := p.extractDoxygenComments(child, content)

				// Check if virtual
				isVirtual := p.isVirtualMethod(child, content)
				kind := "method"
				if isVirtual {
					kind = "virtual_method"
				}

				member := ParsedSymbol{
					Name:      funcName,
					Kind:      kind,
					Signature: signature,
					Span:      nodeToSpan(child),
					Docstring: docstring,
					Node:      child,
				}
				members = append(members, member)
			}
		}

		// Extract field declarations
		if child.Type() == "field_declaration" {
			fieldNames := p.extractFieldNames(child, content)
			for _, fieldName := range fieldNames {
				signature := p.extractSignature(child, content)
				docstring := p.extractDoxygenComments(child, content)

				member := ParsedSymbol{
					Name:      fieldName,
					Kind:      "field",
					Signature: signature,
					Span:      nodeToSpan(child),
					Docstring: docstring,
					Node:      child,
				}
				members = append(members, member)
			}
		}

		// Extract access specifiers (public, private, protected)
		if child.Type() == "access_specifier" {
			// We could track access levels here if needed
		}
	}

	return members
}

// extractFunctionName extracts the function name from a function definition
func (p *CppParser) extractFunctionName(funcNode *sitter.Node, content []byte) string {
	// Look for function_declarator
	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child.Type() == "function_declarator" {
			// Find the identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				declChild := child.Child(j)
				if declChild.Type() == "identifier" || declChild.Type() == "field_identifier" {
					return declChild.Content(content)
				}
				// Handle qualified identifiers (ClassName::methodName)
				if declChild.Type() == "qualified_identifier" {
					// Get the last part (method name)
					for k := int(declChild.ChildCount()) - 1; k >= 0; k-- {
						qualChild := declChild.Child(k)
						if qualChild.Type() == "identifier" {
							return qualChild.Content(content)
						}
					}
				}
				// Handle destructor
				if declChild.Type() == "destructor_name" {
					return declChild.Content(content)
				}
			}
		}
	}
	return ""
}

// extractFieldNames extracts field names from a field declaration
func (p *CppParser) extractFieldNames(fieldNode *sitter.Node, content []byte) []string {
	var names []string

	// Recursively search for field_identifier nodes
	var findIdentifiers func(*sitter.Node)
	findIdentifiers = func(node *sitter.Node) {
		if node.Type() == "field_identifier" {
			names = append(names, node.Content(content))
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			findIdentifiers(node.Child(i))
		}
	}

	findIdentifiers(fieldNode)
	return names
}

// isVirtualMethod checks if a method is virtual
func (p *CppParser) isVirtualMethod(funcNode *sitter.Node, content []byte) bool {
	// Look for virtual keyword
	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child.Type() == "virtual" || child.Type() == "virtual_specifier" {
			return true
		}
		// Check for "virtual" in storage class specifier
		if child.Type() == "storage_class_specifier" {
			if strings.Contains(child.Content(content), "virtual") {
				return true
			}
		}
	}
	return false
}

// extractTemplates extracts template declarations (class and function templates)
func (p *CppParser) extractTemplates(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for template declarations
	templateQuery := `(template_declaration) @template.decl`

	matches, err := p.tsParser.Query(rootNode, templateQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			templateNode := capture.Node

			// Extract template name
			templateName := p.extractTemplateName(templateNode, content)
			if templateName != "" {
				signature := p.extractSignature(templateNode, content)
				docstring := p.extractDoxygenComments(templateNode, content)

				// Determine if it's a class template or function template
				kind := "template"
				if p.isClassTemplate(templateNode) {
					kind = "class_template"
				} else if p.isFunctionTemplate(templateNode) {
					kind = "function_template"
				}

				symbol := ParsedSymbol{
					Name:      templateName,
					Kind:      kind,
					Signature: signature,
					Span:      nodeToSpan(templateNode),
					Docstring: docstring,
					Node:      templateNode,
				}

				parsedFile.Symbols = append(parsedFile.Symbols, symbol)
			}
		}
	}

	return nil
}

// extractTemplateName extracts the name from a template declaration
func (p *CppParser) extractTemplateName(templateNode *sitter.Node, content []byte) string {
	// Look for class_specifier or function_definition inside template
	for i := 0; i < int(templateNode.ChildCount()); i++ {
		child := templateNode.Child(i)

		if child.Type() == "class_specifier" || child.Type() == "struct_specifier" {
			// Find the type_identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				classChild := child.Child(j)
				if classChild.Type() == "type_identifier" {
					return classChild.Content(content)
				}
			}
		}

		if child.Type() == "function_definition" {
			return p.extractFunctionName(child, content)
		}
	}

	return ""
}

// isClassTemplate checks if a template is a class template
func (p *CppParser) isClassTemplate(templateNode *sitter.Node) bool {
	for i := 0; i < int(templateNode.ChildCount()); i++ {
		child := templateNode.Child(i)
		if child.Type() == "class_specifier" || child.Type() == "struct_specifier" {
			return true
		}
	}
	return false
}

// isFunctionTemplate checks if a template is a function template
func (p *CppParser) isFunctionTemplate(templateNode *sitter.Node) bool {
	for i := 0; i < int(templateNode.ChildCount()); i++ {
		child := templateNode.Child(i)
		if child.Type() == "function_definition" {
			return true
		}
	}
	return false
}

// extractFunctions extracts function declarations and definitions
func (p *CppParser) extractFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte, isHeader bool) error {
	// Query for function definitions
	funcQuery := `(function_definition
		declarator: (function_declarator) @func.declarator) @func.def`

	matches, err := p.tsParser.Query(rootNode, funcQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var funcNode *sitter.Node
		var funcDeclarator *sitter.Node

		for _, capture := range match.Captures {
			if capture.Index == 0 { // func.declarator
				funcDeclarator = capture.Node
			} else if capture.Index == 1 { // func.def
				funcNode = capture.Node
			}
		}

		if funcNode != nil && funcDeclarator != nil {
			funcName := p.extractFunctionName(funcNode, content)
			if funcName != "" {
				signature := p.extractSignature(funcNode, content)
				docstring := p.extractDoxygenComments(funcNode, content)

				// Check if this is a static function
				isStatic := p.isStaticFunction(funcNode, content)
				kind := "function"
				if isStatic {
					kind = "static_function"
				}

				// Check if this is an inline function
				if p.isInlineFunction(funcNode, content) {
					kind = "inline_function"
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
	}

	// Also extract function declarations (in headers)
	declQuery := `(declaration
		declarator: (function_declarator) @func.declarator) @func.decl`

	matches, err = p.tsParser.Query(rootNode, declQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var declNode *sitter.Node
		var funcDeclarator *sitter.Node

		for _, capture := range match.Captures {
			if capture.Index == 0 { // func.declarator
				funcDeclarator = capture.Node
			} else if capture.Index == 1 { // func.decl
				declNode = capture.Node
			}
		}

		if declNode != nil && funcDeclarator != nil {
			funcName := p.extractFunctionNameFromDeclarator(funcDeclarator, content)
			if funcName != "" {
				signature := p.extractSignature(declNode, content)
				docstring := p.extractDoxygenComments(declNode, content)

				symbol := ParsedSymbol{
					Name:      funcName,
					Kind:      "function_declaration",
					Signature: signature,
					Span:      nodeToSpan(declNode),
					Docstring: docstring,
					Node:      declNode,
				}

				parsedFile.Symbols = append(parsedFile.Symbols, symbol)
			}
		}
	}

	return nil
}

// extractFunctionNameFromDeclarator extracts function name from a declarator
func (p *CppParser) extractFunctionNameFromDeclarator(declarator *sitter.Node, content []byte) string {
	for i := 0; i < int(declarator.ChildCount()); i++ {
		child := declarator.Child(i)
		if child.Type() == "identifier" || child.Type() == "field_identifier" {
			return child.Content(content)
		}
		// Handle qualified identifiers
		if child.Type() == "qualified_identifier" {
			for j := int(child.ChildCount()) - 1; j >= 0; j-- {
				qualChild := child.Child(j)
				if qualChild.Type() == "identifier" {
					return qualChild.Content(content)
				}
			}
		}
	}
	return ""
}

// isStaticFunction checks if a function has static storage class
func (p *CppParser) isStaticFunction(funcNode *sitter.Node, content []byte) bool {
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

// isInlineFunction checks if a function is inline
func (p *CppParser) isInlineFunction(funcNode *sitter.Node, content []byte) bool {
	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child.Type() == "storage_class_specifier" {
			if strings.Contains(child.Content(content), "inline") {
				return true
			}
		}
	}
	return false
}

// extractMethods extracts member function declarations
func (p *CppParser) extractMethods(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Methods are already extracted as part of class extraction
	// This function is here for completeness and could be used for
	// out-of-class method definitions
	return nil
}

// extractOperators extracts operator overloads
func (p *CppParser) extractOperators(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for operator function definitions
	operatorQuery := `(function_definition
		declarator: (function_declarator
			declarator: (operator_name) @op.name)) @op.def`

	matches, err := p.tsParser.Query(rootNode, operatorQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var opNode *sitter.Node
		var opName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // op.name
				opName = capture.Node.Content(content)
			} else if capture.Index == 1 { // op.def
				opNode = capture.Node
			}
		}

		if opNode != nil && opName != "" {
			signature := p.extractSignature(opNode, content)
			docstring := p.extractDoxygenComments(opNode, content)

			symbol := ParsedSymbol{
				Name:      opName,
				Kind:      "operator",
				Signature: signature,
				Span:      nodeToSpan(opNode),
				Docstring: docstring,
				Node:      opNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractVirtualMethods extracts virtual and pure virtual methods
func (p *CppParser) extractVirtualMethods(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Virtual methods are already marked during class member extraction
	// This function could be used to extract additional information about
	// pure virtual methods (= 0) if needed
	return nil
}

// matchDeclarationToImplementation matches declarations with implementations
func (p *CppParser) matchDeclarationToImplementation(headerFile, implFile *ParsedFile) {
	// Find function declarations in header
	for _, headerSymbol := range headerFile.Symbols {
		if headerSymbol.Kind == "function_declaration" {
			// Find matching function definition in implementation file
			for _, implSymbol := range implFile.Symbols {
				if (implSymbol.Kind == "function" || implSymbol.Kind == "static_function" || implSymbol.Kind == "inline_function") &&
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

		// Match class methods
		if headerSymbol.Kind == "class" || headerSymbol.Kind == "struct" {
			// Find matching class in implementation file
			for _, implSymbol := range implFile.Symbols {
				if (implSymbol.Kind == "class" || implSymbol.Kind == "struct") &&
					implSymbol.Name == headerSymbol.Name {

					// Match methods within the class
					p.matchClassMethods(headerSymbol, implSymbol, implFile)
				}
			}

			// Also check for out-of-class method definitions
			p.matchOutOfClassMethods(headerSymbol, implFile)
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

// matchClassMethods matches methods between header and implementation classes
func (p *CppParser) matchClassMethods(headerClass, implClass ParsedSymbol, implFile *ParsedFile) {
	// Match methods in the class bodies
	for _, headerMethod := range headerClass.Children {
		if headerMethod.Kind == "method" || headerMethod.Kind == "virtual_method" {
			for _, implMethod := range implClass.Children {
				if (implMethod.Kind == "method" || implMethod.Kind == "virtual_method") &&
					implMethod.Name == headerMethod.Name {

					// Create implements_declaration edge
					dependency := ParsedDependency{
						Type:   "implements_declaration",
						Source: fmt.Sprintf("%s::%s", implClass.Name, implMethod.Name),
						Target: fmt.Sprintf("%s::%s", headerClass.Name, headerMethod.Name),
					}
					implFile.Dependencies = append(implFile.Dependencies, dependency)
				}
			}
		}
	}
}

// matchOutOfClassMethods matches out-of-class method definitions (ClassName::methodName)
func (p *CppParser) matchOutOfClassMethods(headerClass ParsedSymbol, implFile *ParsedFile) {
	// Look for function definitions with qualified names (ClassName::methodName)
	for _, implSymbol := range implFile.Symbols {
		if implSymbol.Kind == "function" || implSymbol.Kind == "inline_function" {
			// Check if the function name contains :: (qualified identifier)
			if strings.Contains(implSymbol.Signature, headerClass.Name+"::") {
				// Extract the method name
				methodName := p.extractMethodNameFromQualified(implSymbol.Signature, headerClass.Name)

				// Find matching method in header class
				for _, headerMethod := range headerClass.Children {
					if headerMethod.Name == methodName {
						// Create implements_declaration edge
						dependency := ParsedDependency{
							Type:   "implements_declaration",
							Source: fmt.Sprintf("%s::%s", headerClass.Name, methodName),
							Target: fmt.Sprintf("%s::%s", headerClass.Name, headerMethod.Name),
						}
						implFile.Dependencies = append(implFile.Dependencies, dependency)
					}
				}
			}
		}
	}
}

// extractMethodNameFromQualified extracts method name from qualified identifier
func (p *CppParser) extractMethodNameFromQualified(signature, className string) string {
	// Look for ClassName::methodName pattern
	pattern := className + "::"
	idx := strings.Index(signature, pattern)
	if idx == -1 {
		return ""
	}

	// Extract everything after ClassName::
	rest := signature[idx+len(pattern):]

	// Find the method name (up to opening parenthesis)
	parenIdx := strings.Index(rest, "(")
	if parenIdx == -1 {
		return ""
	}

	methodName := strings.TrimSpace(rest[:parenIdx])
	return methodName
}

// resolveTemplateInstantiations links template uses to declarations
func (p *CppParser) resolveTemplateInstantiations(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for template instantiations
	templateInstQuery := `(template_type) @template.inst`

	matches, err := p.tsParser.Query(rootNode, templateInstQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			templateNode := capture.Node

			// Extract template name
			templateName := p.extractTemplateTypeName(templateNode, content)
			if templateName != "" {
				// Find the containing function or class
				container := p.findContainingSymbol(templateNode, parsedFile)

				if container != "" {
					// Create template instantiation dependency
					dependency := ParsedDependency{
						Type:   "uses",
						Source: container,
						Target: templateName,
					}
					parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
				}
			}
		}
	}

	return nil
}

// extractTemplateTypeName extracts the template name from a template_type node
func (p *CppParser) extractTemplateTypeName(templateNode *sitter.Node, content []byte) string {
	// Look for type_identifier
	for i := 0; i < int(templateNode.ChildCount()); i++ {
		child := templateNode.Child(i)
		if child.Type() == "type_identifier" {
			return child.Content(content)
		}
	}
	return ""
}

// findContainingSymbol finds the name of the symbol containing a node
func (p *CppParser) findContainingSymbol(node *sitter.Node, parsedFile *ParsedFile) string {
	current := node.Parent()

	for current != nil {
		// Check if this is a function definition or class
		for _, symbol := range parsedFile.Symbols {
			if symbol.Node == current {
				return symbol.Name
			}
		}
		current = current.Parent()
	}

	return ""
}

// normalizeSignature normalizes a function signature for comparison
func (p *CppParser) normalizeSignature(signature string) string {
	// Remove extra whitespace
	signature = strings.Join(strings.Fields(signature), " ")

	// Remove const, volatile, noexcept, etc.
	signature = strings.ReplaceAll(signature, " const", "")
	signature = strings.ReplaceAll(signature, " volatile", "")
	signature = strings.ReplaceAll(signature, " noexcept", "")
	signature = strings.ReplaceAll(signature, " override", "")
	signature = strings.ReplaceAll(signature, " final", "")

	signature = strings.TrimSpace(signature)

	return signature
}

// signaturesMatch compares two normalized signatures
func (p *CppParser) signaturesMatch(sig1, sig2 string) bool {
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
func (p *CppParser) parseSignature(signature string) (string, []string) {
	// Find the function name (between return type and opening parenthesis)
	openParen := strings.Index(signature, "(")
	if openParen == -1 {
		return "", nil
	}

	// Extract everything before the parenthesis
	beforeParen := strings.TrimSpace(signature[:openParen])
	parts := strings.Fields(beforeParen)

	// Last part is the function name (may include :: for qualified names)
	var funcName string
	if len(parts) > 0 {
		funcName = parts[len(parts)-1]
		// Remove any namespace qualifiers for comparison
		if idx := strings.LastIndex(funcName, "::"); idx != -1 {
			funcName = funcName[idx+2:]
		}
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

	// Split parameters by comma (handling nested templates)
	params := p.splitParameters(paramsStr)
	for i := range params {
		params[i] = strings.TrimSpace(params[i])
	}

	return funcName, params
}

// splitParameters splits parameters by comma, handling nested templates
func (p *CppParser) splitParameters(paramsStr string) []string {
	var params []string
	var current strings.Builder
	depth := 0

	for _, ch := range paramsStr {
		switch ch {
		case '<':
			depth++
			current.WriteRune(ch)
		case '>':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				params = append(params, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		params = append(params, current.String())
	}

	return params
}

// extractParameterType extracts the type from a parameter declaration
func (p *CppParser) extractParameterType(param string) string {
	// Remove parameter name, keeping only the type
	parts := strings.Fields(param)

	if len(parts) == 0 {
		return ""
	}

	// For simple types like "int x", return "int"
	// For pointer types like "int *x" or "int* x", return "int*"
	// For reference types like "int &x" or "int& x", return "int&"
	// For const types like "const int x", return "const int"

	// If last part doesn't contain * or &, it's likely the parameter name
	lastPart := parts[len(parts)-1]
	if !strings.Contains(lastPart, "*") && !strings.Contains(lastPart, "&") &&
		!strings.Contains(lastPart, "<") && !strings.Contains(lastPart, ">") {
		// Remove last part (parameter name)
		if len(parts) > 1 {
			return strings.Join(parts[:len(parts)-1], " ")
		}
	}

	// Otherwise, return the whole thing
	return param
}

// extractCallRelationships extracts function/method calls
func (p *CppParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for call expressions
	callQuery := `(call_expression
		function: (identifier) @call.target)`

	matches, err := p.tsParser.Query(rootNode, callQuery, "cpp")
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

	// Handle qualified function calls (namespace::function or Class::method)
	qualifiedCallQuery := `(call_expression
		function: (qualified_identifier) @call.target)`

	matches, err = p.tsParser.Query(rootNode, qualifiedCallQuery, "cpp")
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

	// Handle member function calls (object.method() or object->method())
	memberCallQuery := `(call_expression
		function: (field_expression) @call.target)`

	matches, err = p.tsParser.Query(rootNode, memberCallQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			callTarget := capture.Node.Content(content)

			// Find the containing function for this call
			caller := p.findContainingFunction(capture.Node, parsedFile)
			if caller != "" {
				// Extract just the method name
				methodName := p.extractMethodNameFromFieldExpression(callTarget)

				dependency := ParsedDependency{
					Type:   "call",
					Source: caller,
					Target: methodName,
				}

				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractMethodNameFromFieldExpression extracts method name from field expression
func (p *CppParser) extractMethodNameFromFieldExpression(fieldExpr string) string {
	// Handle object.method or object->method
	if idx := strings.LastIndex(fieldExpr, "."); idx != -1 {
		return fieldExpr[idx+1:]
	}
	if idx := strings.LastIndex(fieldExpr, "->"); idx != -1 {
		return fieldExpr[idx+2:]
	}
	return fieldExpr
}

// extractInheritance extracts inheritance relationships (including multiple inheritance)
func (p *CppParser) extractInheritance(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for class specifiers with base class lists
	inheritanceQuery := `(class_specifier
		name: (type_identifier) @class.name
		(base_class_clause) @class.bases) @class.def`

	matches, err := p.tsParser.Query(rootNode, inheritanceQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var className string
		var basesNode *sitter.Node

		for _, capture := range match.Captures {
			if capture.Index == 0 { // class.name
				className = capture.Node.Content(content)
			} else if capture.Index == 1 { // class.bases
				basesNode = capture.Node
			}
		}

		if className != "" && basesNode != nil {
			// Extract base classes
			baseClasses := p.extractBaseClasses(basesNode, content)

			for _, baseClass := range baseClasses {
				dependency := ParsedDependency{
					Type:   "extends",
					Source: className,
					Target: baseClass,
				}

				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	// Also handle struct inheritance
	structInheritanceQuery := `(struct_specifier
		name: (type_identifier) @struct.name
		(base_class_clause) @struct.bases) @struct.def`

	matches, err = p.tsParser.Query(rootNode, structInheritanceQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var structName string
		var basesNode *sitter.Node

		for _, capture := range match.Captures {
			if capture.Index == 0 { // struct.name
				structName = capture.Node.Content(content)
			} else if capture.Index == 1 { // struct.bases
				basesNode = capture.Node
			}
		}

		if structName != "" && basesNode != nil {
			// Extract base classes
			baseClasses := p.extractBaseClasses(basesNode, content)

			for _, baseClass := range baseClasses {
				dependency := ParsedDependency{
					Type:   "extends",
					Source: structName,
					Target: baseClass,
				}

				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	// Extract virtual method overrides
	if err := p.extractVirtualMethodOverrides(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	return nil
}

// extractBaseClasses extracts base class names from a base_class_clause
func (p *CppParser) extractBaseClasses(basesNode *sitter.Node, content []byte) []string {
	var baseClasses []string

	// Iterate through base class specifiers
	for i := 0; i < int(basesNode.ChildCount()); i++ {
		child := basesNode.Child(i)

		// Look for type_identifier in base class specifier
		if child.Type() == "type_identifier" {
			baseClasses = append(baseClasses, child.Content(content))
		}

		// Handle nested structures
		for j := 0; j < int(child.ChildCount()); j++ {
			grandchild := child.Child(j)
			if grandchild.Type() == "type_identifier" {
				baseClasses = append(baseClasses, grandchild.Content(content))
			}
		}
	}

	return baseClasses
}

// extractVirtualMethodOverrides extracts virtual method override relationships
func (p *CppParser) extractVirtualMethodOverrides(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for methods with override specifier
	overrideQuery := `(function_definition
		declarator: (function_declarator) @method.declarator
		(virtual_specifier) @method.override) @method.def`

	matches, err := p.tsParser.Query(rootNode, overrideQuery, "cpp")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var methodNode *sitter.Node
		var overrideNode *sitter.Node

		for _, capture := range match.Captures {
			if capture.Index == 1 { // method.override
				overrideNode = capture.Node
			} else if capture.Index == 2 { // method.def
				methodNode = capture.Node
			}
		}

		if methodNode != nil && overrideNode != nil {
			// Check if this is actually an override specifier
			if strings.Contains(overrideNode.Content(content), "override") {
				methodName := p.extractFunctionName(methodNode, content)

				// Find the containing class
				className := p.findContainingClass(methodNode, parsedFile)

				if className != "" && methodName != "" {
					// Create override dependency
					// In a real implementation, we would need to find the base class method
					// For now, we just mark it as an override
					dependency := ParsedDependency{
						Type:   "overrides",
						Source: fmt.Sprintf("%s::%s", className, methodName),
						Target: methodName, // Would need to resolve to base class method
					}

					parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
				}
			}
		}
	}

	return nil
}

// findContainingClass finds the name of the class containing a node
func (p *CppParser) findContainingClass(node *sitter.Node, parsedFile *ParsedFile) string {
	current := node.Parent()

	for current != nil {
		// Check if this is a class or struct
		if current.Type() == "class_specifier" || current.Type() == "struct_specifier" {
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

// resolveCallToDeclaration links calls to header declarations
func (p *CppParser) resolveCallToDeclaration(call string, currentFile *ParsedFile) string {
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

// isExternalImport classifies includes (std:: as internal)
func (p *CppParser) isExternalImport(includePath string, isSystem bool) bool {
	// Standard C++ library headers are considered internal (part of the language)
	stdHeaders := map[string]bool{
		// C++ standard library
		"iostream":     true,
		"fstream":      true,
		"sstream":      true,
		"iomanip":      true,
		"string":       true,
		"vector":       true,
		"list":         true,
		"map":          true,
		"set":          true,
		"queue":        true,
		"stack":        true,
		"deque":        true,
		"array":        true,
		"algorithm":    true,
		"iterator":     true,
		"functional":   true,
		"memory":       true,
		"utility":      true,
		"tuple":        true,
		"optional":     true,
		"variant":      true,
		"any":          true,
		"chrono":       true,
		"thread":       true,
		"mutex":        true,
		"condition_variable": true,
		"atomic":       true,
		"future":       true,
		"exception":    true,
		"stdexcept":    true,
		"typeinfo":     true,
		"type_traits":  true,
		"limits":       true,
		"numeric":      true,
		"cmath":        true,
		"cstdlib":      true,
		"cstdio":       true,
		"cstring":      true,
		"ctime":        true,
		"cassert":      true,
		"cerrno":       true,
		"cctype":       true,
		"cwchar":       true,
		"cwctype":      true,
		"regex":        true,
		"random":       true,
		"bitset":       true,
		"complex":      true,
		"valarray":     true,
		"locale":       true,
		"codecvt":      true,
		"filesystem":   true,
	}

	// Check if it's a standard header
	if stdHeaders[includePath] {
		return false // Standard library is internal
	}

	// System headers (usually in angle brackets) from external libraries
	if isSystem {
		// Check for common external libraries
		externalPrefixes := []string{
			"boost/",
			"Qt",
			"wx/",
			"gtk/",
			"SDL",
			"SFML/",
			"eigen",
			"opencv",
			"curl/",
			"json/",
			"yaml-cpp/",
			"protobuf/",
			"grpc/",
		}

		for _, prefix := range externalPrefixes {
			if strings.HasPrefix(includePath, prefix) || strings.Contains(includePath, prefix) {
				return true
			}
		}

		// POSIX and system headers
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
	}

	// If it's a local header (contains .h/.hpp and no path separator), it's internal
	if (strings.HasSuffix(includePath, ".h") ||
		strings.HasSuffix(includePath, ".hpp") ||
		strings.HasSuffix(includePath, ".hh") ||
		strings.HasSuffix(includePath, ".hxx")) &&
		!strings.Contains(includePath, "/") {
		return false
	}

	// Everything else is considered external
	return true
}
