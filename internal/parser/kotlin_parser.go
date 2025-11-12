package parser

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// KotlinParser parses Kotlin source code using Tree-sitter
type KotlinParser struct {
	tsParser *TreeSitterParser
}

// NewKotlinParser creates a new Kotlin parser
func NewKotlinParser(tsParser *TreeSitterParser) *KotlinParser {
	return &KotlinParser{
		tsParser: tsParser,
	}
}

// Parse parses a Kotlin file and extracts symbols and dependencies
func (p *KotlinParser) Parse(file ScannedFile) (*ParsedFile, error) {
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
	rootNode, parseErr := p.tsParser.Parse(content, "kotlin")

	parsedFile := &ParsedFile{
		Path:     file.Path,
		Language: "kotlin",
		Content:  content,
		RootNode: rootNode,
	}

	// Store file path for package inference
	parsedFile.Path = file.Path

	// If we have no root node at all, return error immediately
	if rootNode == nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to parse Kotlin file: %v", parseErr),
			Type:    "parse",
		}
	}

	// Extract package declaration
	if err := p.extractPackage(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract imports
	if err := p.extractImports(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract classes (including data classes, sealed classes, objects)
	if err := p.extractClasses(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract functions (top-level and extension functions)
	if err := p.extractFunctions(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract interfaces
	if err := p.extractInterfaces(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract properties (top-level)
	if err := p.extractProperties(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract annotations
	if err := p.extractAnnotations(rootNode, parsedFile, content); err != nil {
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
			Message: fmt.Sprintf("syntax error in Kotlin file: %v", parseErr),
			Type:    "parse",
		}
	}

	return parsedFile, nil
}

// Helper functions for node traversal and span conversion

// nodeToSpan is already defined in go_parser.go, but we'll use it here
// It converts a Tree-sitter node to a ParsedSpan

// findChildByType is already defined in go_parser.go
// It finds a child node by type

// extractSignature extracts a clean signature from a node
func (p *KotlinParser) extractSignature(node *sitter.Node, content []byte) string {
	// Get the first line or up to opening brace
	nodeText := node.Content(content)
	lines := strings.Split(nodeText, "\n")
	
	signature := ""
	for _, line := range lines {
		signature += line
		// Stop at opening brace or if we have enough
		if strings.Contains(line, "{") {
			// Remove the brace and everything after
			if idx := strings.Index(signature, "{"); idx != -1 {
				signature = signature[:idx]
			}
			break
		}
		signature += " "
	}
	
	return strings.TrimSpace(signature)
}

// findContainingFunction finds the name of the function/method containing a node
func (p *KotlinParser) findContainingFunction(node *sitter.Node, parsedFile *ParsedFile) string {
	current := node.Parent()

	for current != nil {
		// Check if this is a function declaration
		if current.Type() == "function_declaration" {
			// Find the matching symbol in our parsed symbols
			for _, symbol := range parsedFile.Symbols {
				if symbol.Node == current {
					return symbol.Name
				}
				// Also check children (methods in classes)
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

// extractPackage extracts the package declaration
func (p *KotlinParser) extractPackage(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for package declaration
	query := `(package_header (identifier) @package.name)`

	matches, err := p.tsParser.Query(rootNode, query, "kotlin")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			packageName := capture.Node.Content(content)

			symbol := ParsedSymbol{
				Name: packageName,
				Kind: "package",
				Span: nodeToSpan(capture.Node),
				Node: capture.Node,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
			return nil
		}
	}

	// If no package declaration found, try to infer from file path
	// Kotlin follows same convention as Java: src/main/kotlin/com/example/MyClass.kt -> com.example
	inferredPackage := p.inferPackageFromPath(parsedFile.Path)
	if inferredPackage != "" {
		symbol := ParsedSymbol{
			Name: inferredPackage,
			Kind: "package",
		}
		parsedFile.Symbols = append(parsedFile.Symbols, symbol)
	}

	return nil
}

// inferPackageFromPath infers the package name from the file path
// Kotlin follows same convention as Java
func (p *KotlinParser) inferPackageFromPath(filePath string) string {
	// Normalize path separators
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	
	// Common Kotlin/Java source roots
	sourceRoots := []string{
		"src/main/kotlin/",
		"src/test/kotlin/",
		"src/main/java/",  // Kotlin can be in java folders too
		"src/test/java/",
		"src/",
		"kotlin/",
		"java/",
	}
	
	for _, root := range sourceRoots {
		if idx := strings.Index(filePath, root); idx != -1 {
			// Extract path after source root
			packagePath := filePath[idx+len(root):]
			
			// Remove filename
			if lastSlash := strings.LastIndex(packagePath, "/"); lastSlash != -1 {
				packagePath = packagePath[:lastSlash]
			} else {
				// No subdirectory, default package
				return ""
			}
			
			// Convert path to package name (replace / with .)
			packageName := strings.ReplaceAll(packagePath, "/", ".")
			return packageName
		}
	}
	
	return ""
}

// extractImports extracts import statements with external/internal classification
func (p *KotlinParser) extractImports(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Find the package symbol to use as the source for imports
	var packageSymbol string
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "package" {
			packageSymbol = symbol.Name
			break
		}
	}

	// Query for import statements
	query := `(import_header (identifier) @import.path)`

	matches, err := p.tsParser.Query(rootNode, query, "kotlin")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			importPath := capture.Node.Content(content)

			dependency := ParsedDependency{
				Type:         "import",
				Source:       packageSymbol,
				Target:       importPath,
				TargetModule: importPath,
				IsExternal:   p.isExternalImportWithContext(importPath, packageSymbol),
			}

			parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
		}
	}

	return nil
}

// isExternalImport determines if an import path refers to an external module
func (p *KotlinParser) isExternalImport(importPath string) bool {
	// Kotlin standard library and kotlinx are considered internal
	if strings.HasPrefix(importPath, "kotlin.") || strings.HasPrefix(importPath, "kotlinx.") {
		return false
	}
	
	// Everything else is external (third-party libraries)
	return true
}

// isExternalImportWithContext determines if an import is external based on current package
// This handles both Kotlin and Java interop
func (p *KotlinParser) isExternalImportWithContext(importPath string, currentPackage string) bool {
	// Kotlin standard library and kotlinx are considered internal
	if strings.HasPrefix(importPath, "kotlin.") || strings.HasPrefix(importPath, "kotlinx.") {
		return false
	}
	
	// Java standard library (for Kotlin-Java interop)
	if strings.HasPrefix(importPath, "java.") || strings.HasPrefix(importPath, "javax.") {
		return false
	}
	
	// If import is from the same package or subpackage, it's internal to the project
	if currentPackage != "" {
		// Extract base package (first 2-3 segments, e.g., com.example from com.example.module)
		basePackage := p.extractBasePackage(currentPackage)
		importBasePackage := p.extractBasePackage(importPath)
		
		// If they share the same base package, consider it internal to the project
		// This works for both Kotlin-Kotlin and Kotlin-Java interop
		if basePackage != "" && importBasePackage != "" && basePackage == importBasePackage {
			return false
		}
	}
	
	// Everything else is external (third-party libraries)
	return true
}

// extractBasePackage extracts the base package (first 2-3 segments)
// e.g., com.example.module.service -> com.example
func (p *KotlinParser) extractBasePackage(packageName string) string {
	parts := strings.Split(packageName, ".")
	
	// For typical packages like com.example.*, org.project.*, etc.
	// we consider the first 2 segments as the base package
	if len(parts) >= 2 {
		return strings.Join(parts[:2], ".")
	}
	
	return packageName
}

// extractClasses extracts class, data class, sealed class, and object declarations
func (p *KotlinParser) extractClasses(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for class declarations (including data classes, sealed classes)
	classQuery := `(class_declaration 
		(type_identifier) @class.name) @class.def`

	matches, err := p.tsParser.Query(rootNode, classQuery, "kotlin")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var classNode *sitter.Node
		var className string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // class.name
				className = capture.Node.Content(content)
			} else if capture.Index == 1 { // class.def
				classNode = capture.Node
			}
		}

		if classNode != nil && className != "" {
			// Get package name for fully qualified name
			var packageName string
			for _, symbol := range parsedFile.Symbols {
				if symbol.Kind == "package" {
					packageName = symbol.Name
					break
				}
			}
			
			// Build fully qualified name
			fullyQualifiedName := className
			if packageName != "" {
				fullyQualifiedName = packageName + "." + className
			}
			
			// Determine class kind (class, data class, sealed class)
			kind := p.determineClassKind(classNode, content)
			
			// Extract class members
			properties := p.extractClassProperties(classNode, content)
			methods := p.extractClassMethods(classNode, content)
			
			// Extract inheritance
			superTypes := p.extractSuperTypes(classNode, content)
			
			// Extract KDoc
			docstring := p.extractKDoc(classNode, content)
			
			// Build signature
			signature := p.extractSignature(classNode, content)

			symbol := ParsedSymbol{
				Name:      fullyQualifiedName, // Use fully qualified name
				Kind:      kind,
				Signature: signature,
				Span:      nodeToSpan(classNode),
				Docstring: docstring,
				Node:      classNode,
				Children:  append(properties, methods...),
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)

			// Add inheritance dependencies
			for _, superType := range superTypes {
				dependency := ParsedDependency{
					Type:   "extends",
					Source: fullyQualifiedName, // Use fully qualified name
					Target: superType,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	// Query for object declarations
	objectQuery := `(object_declaration 
		(type_identifier) @object.name) @object.def`

	matches, err = p.tsParser.Query(rootNode, objectQuery, "kotlin")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var objectNode *sitter.Node
		var objectName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // object.name
				objectName = capture.Node.Content(content)
			} else if capture.Index == 1 { // object.def
				objectNode = capture.Node
			}
		}

		if objectNode != nil && objectName != "" {
			// Get package name for fully qualified name
			var packageName string
			for _, symbol := range parsedFile.Symbols {
				if symbol.Kind == "package" {
					packageName = symbol.Name
					break
				}
			}
			
			// Build fully qualified name
			fullyQualifiedName := objectName
			if packageName != "" {
				fullyQualifiedName = packageName + "." + objectName
			}
			
			// Extract object members
			properties := p.extractClassProperties(objectNode, content)
			methods := p.extractClassMethods(objectNode, content)
			
			// Extract KDoc
			docstring := p.extractKDoc(objectNode, content)
			
			// Build signature
			signature := p.extractSignature(objectNode, content)

			symbol := ParsedSymbol{
				Name:      fullyQualifiedName, // Use fully qualified name
				Kind:      "object",
				Signature: signature,
				Span:      nodeToSpan(objectNode),
				Docstring: docstring,
				Node:      objectNode,
				Children:  append(properties, methods...),
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// determineClassKind determines if a class is a regular class, data class, or sealed class
func (p *KotlinParser) determineClassKind(classNode *sitter.Node, content []byte) string {
	// Check for modifiers
	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child.Type() == "modifiers" {
			modifiersText := child.Content(content)
			if strings.Contains(modifiersText, "data") {
				return "data_class"
			}
			if strings.Contains(modifiersText, "sealed") {
				return "sealed_class"
			}
		}
	}
	return "class"
}

// extractClassProperties extracts properties from a class
func (p *KotlinParser) extractClassProperties(classNode *sitter.Node, content []byte) []ParsedSymbol {
	var properties []ParsedSymbol

	// Find class body
	classBody := findChildByType(classNode, "class_body")
	if classBody == nil {
		return properties
	}

	// Iterate through class body children
	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "property_declaration" {
			propertyName := ""

			// Find the variable_declaration
			varDecl := findChildByType(child, "variable_declaration")
			if varDecl != nil {
				// Find the simple_identifier
				for j := 0; j < int(varDecl.ChildCount()); j++ {
					nameNode := varDecl.Child(j)
					if nameNode.Type() == "simple_identifier" {
						propertyName = nameNode.Content(content)
						break
					}
				}
			}

			if propertyName != "" {
				signature := p.extractSignature(child, content)
				docstring := p.extractKDoc(child, content)

				property := ParsedSymbol{
					Name:      propertyName,
					Kind:      "property",
					Signature: signature,
					Span:      nodeToSpan(child),
					Docstring: docstring,
					Node:      child,
				}

				properties = append(properties, property)
			}
		}
	}

	return properties
}

// extractClassMethods extracts methods from a class
func (p *KotlinParser) extractClassMethods(classNode *sitter.Node, content []byte) []ParsedSymbol {
	var methods []ParsedSymbol

	// Find class body
	classBody := findChildByType(classNode, "class_body")
	if classBody == nil {
		return methods
	}

	// Iterate through class body children
	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "function_declaration" {
			methodName := ""

			// Find the simple_identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				nameNode := child.Child(j)
				if nameNode.Type() == "simple_identifier" {
					methodName = nameNode.Content(content)
					break
				}
			}

			if methodName != "" {
				signature := p.extractSignature(child, content)
				docstring := p.extractKDoc(child, content)
				
				// Check if it's a suspend function
				kind := "method"
				if p.isSuspendFunction(child, content) {
					kind = "suspend_method"
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

// extractSuperTypes extracts parent classes and implemented interfaces
func (p *KotlinParser) extractSuperTypes(classNode *sitter.Node, content []byte) []string {
	var superTypes []string

	// Find delegation_specifiers (parent classes and interfaces)
	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child.Type() == "delegation_specifiers" {
			// Extract each delegation specifier
			for j := 0; j < int(child.ChildCount()); j++ {
				specifier := child.Child(j)
				if specifier.Type() == "delegation_specifier" || specifier.Type() == "user_type" {
					// Find the type identifier
					typeIdent := findChildByType(specifier, "type_identifier")
					if typeIdent != nil {
						superTypes = append(superTypes, typeIdent.Content(content))
					} else {
						// Sometimes it's directly a user_type
						if specifier.Type() == "user_type" {
							typeIdent = findChildByType(specifier, "type_identifier")
							if typeIdent != nil {
								superTypes = append(superTypes, typeIdent.Content(content))
							}
						}
					}
				}
			}
		}
	}

	return superTypes
}

// extractFunctions extracts top-level and extension functions
func (p *KotlinParser) extractFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for function declarations
	funcQuery := `(function_declaration 
		(simple_identifier) @func.name) @func.def`

	matches, err := p.tsParser.Query(rootNode, funcQuery, "kotlin")
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
			// Skip if this is a method (inside a class)
			if p.isInsideClass(funcNode) {
				continue
			}

			// Determine if it's an extension function
			kind := "function"
			if p.isExtensionFunction(funcNode, content) {
				kind = "extension_function"
			}
			if p.isSuspendFunction(funcNode, content) {
				kind = "suspend_function"
			}

			signature := p.extractSignature(funcNode, content)
			docstring := p.extractKDoc(funcNode, content)

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

// isExtensionFunction checks if a function is an extension function
func (p *KotlinParser) isExtensionFunction(funcNode *sitter.Node, content []byte) bool {
	// Extension functions have a receiver type before the function name
	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child.Type() == "function_value_parameters" {
			// Check if there's a receiver type before parameters
			// In Kotlin tree-sitter, extension functions have a receiver_type
			return false // Will be detected by checking for receiver_type node
		}
	}
	
	// Check for receiver_type in the function signature
	funcText := funcNode.Content(content)
	// Extension functions have format: fun Type.functionName()
	return strings.Contains(funcText, "fun ") && strings.Contains(strings.Split(funcText, "(")[0], ".")
}

// isSuspendFunction checks if a function is a suspend function
func (p *KotlinParser) isSuspendFunction(funcNode *sitter.Node, content []byte) bool {
	// Check for suspend modifier
	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child.Type() == "modifiers" {
			modifiersText := child.Content(content)
			if strings.Contains(modifiersText, "suspend") {
				return true
			}
		}
	}
	return false
}

// isInsideClass checks if a node is inside a class definition
func (p *KotlinParser) isInsideClass(node *sitter.Node) bool {
	current := node.Parent()
	for current != nil {
		if current.Type() == "class_declaration" || current.Type() == "object_declaration" {
			return true
		}
		current = current.Parent()
	}
	return false
}

// extractInterfaces extracts interface declarations
func (p *KotlinParser) extractInterfaces(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for interface declarations
	interfaceQuery := `(class_declaration 
		(modifiers)? 
		(type_identifier) @interface.name) @interface.def`

	matches, err := p.tsParser.Query(rootNode, interfaceQuery, "kotlin")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var interfaceNode *sitter.Node
		var interfaceName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // interface.name
				interfaceName = capture.Node.Content(content)
			} else if capture.Index == 1 { // interface.def
				interfaceNode = capture.Node
			}
		}

		if interfaceNode != nil && interfaceName != "" {
			// Check if it's actually an interface
			isInterface := false
			for i := 0; i < int(interfaceNode.ChildCount()); i++ {
				child := interfaceNode.Child(i)
				if child.Type() == "interface" || (child.Type() == "modifiers" && strings.Contains(child.Content(content), "interface")) {
					isInterface = true
					break
				}
				// Check if the node text starts with "interface"
				if strings.HasPrefix(strings.TrimSpace(interfaceNode.Content(content)), "interface") {
					isInterface = true
					break
				}
			}

			if !isInterface {
				continue
			}

			// Get package name for fully qualified name
			var packageName string
			for _, symbol := range parsedFile.Symbols {
				if symbol.Kind == "package" {
					packageName = symbol.Name
					break
				}
			}
			
			// Build fully qualified name
			fullyQualifiedName := interfaceName
			if packageName != "" {
				fullyQualifiedName = packageName + "." + interfaceName
			}
			
			// Extract interface methods
			methods := p.extractInterfaceMethods(interfaceNode, content)
			
			// Extract KDoc
			docstring := p.extractKDoc(interfaceNode, content)
			
			// Build signature
			signature := p.extractSignature(interfaceNode, content)

			symbol := ParsedSymbol{
				Name:      fullyQualifiedName, // Use fully qualified name
				Kind:      "interface",
				Signature: signature,
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

// extractInterfaceMethods extracts methods from an interface
func (p *KotlinParser) extractInterfaceMethods(interfaceNode *sitter.Node, content []byte) []ParsedSymbol {
	var methods []ParsedSymbol

	// Find class body (interfaces also use class_body)
	classBody := findChildByType(interfaceNode, "class_body")
	if classBody == nil {
		return methods
	}

	// Iterate through class body children
	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "function_declaration" {
			methodName := ""

			// Find the simple_identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				nameNode := child.Child(j)
				if nameNode.Type() == "simple_identifier" {
					methodName = nameNode.Content(content)
					break
				}
			}

			if methodName != "" {
				signature := p.extractSignature(child, content)
				docstring := p.extractKDoc(child, content)

				method := ParsedSymbol{
					Name:      methodName,
					Kind:      "method",
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

// extractProperties extracts top-level properties
func (p *KotlinParser) extractProperties(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for property declarations
	propertyQuery := `(property_declaration 
		(variable_declaration 
			(simple_identifier) @prop.name)) @prop.def`

	matches, err := p.tsParser.Query(rootNode, propertyQuery, "kotlin")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var propNode *sitter.Node
		var propName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // prop.name
				propName = capture.Node.Content(content)
			} else if capture.Index == 1 { // prop.def
				propNode = capture.Node
			}
		}

		if propNode != nil && propName != "" {
			// Skip if this is inside a class
			if p.isInsideClass(propNode) {
				continue
			}

			signature := p.extractSignature(propNode, content)
			docstring := p.extractKDoc(propNode, content)

			symbol := ParsedSymbol{
				Name:      propName,
				Kind:      "property",
				Signature: signature,
				Span:      nodeToSpan(propNode),
				Docstring: docstring,
				Node:      propNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractAnnotations extracts annotation usage
func (p *KotlinParser) extractAnnotations(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for annotations
	annotationQuery := `(annotation 
		(user_type 
			(type_identifier) @annotation.name)) @annotation.def`

	matches, err := p.tsParser.Query(rootNode, annotationQuery, "kotlin")
	if err != nil {
		return err
	}

	// Track unique annotations
	seenAnnotations := make(map[string]bool)

	for _, match := range matches {
		var annotationNode *sitter.Node
		var annotationName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // annotation.name
				annotationName = capture.Node.Content(content)
			} else if capture.Index == 1 { // annotation.def
				annotationNode = capture.Node
			}
		}

		if annotationNode != nil && annotationName != "" {
			// Only add unique annotations
			if seenAnnotations[annotationName] {
				continue
			}
			seenAnnotations[annotationName] = true

			signature := annotationNode.Content(content)

			symbol := ParsedSymbol{
				Name:      annotationName,
				Kind:      "annotation",
				Signature: signature,
				Span:      nodeToSpan(annotationNode),
				Node:      annotationNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractKDoc extracts KDoc documentation comments
func (p *KotlinParser) extractKDoc(node *sitter.Node, content []byte) string {
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
		if sibling.Type() == "comment" || sibling.Type() == "multiline_comment" {
			commentText := sibling.Content(content)

			// Handle KDoc comments (/** ... */)
			if strings.HasPrefix(commentText, "/**") {
				commentText = strings.TrimPrefix(commentText, "/**")
				commentText = strings.TrimSuffix(commentText, "*/")
				commentText = strings.TrimSpace(commentText)

				// Clean up KDoc formatting
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
		} else if sibling.Type() != "comment" && sibling.Type() != "multiline_comment" {
			break
		}
	}

	return strings.Join(comments, "\n")
}

// extractCallRelationships extracts function calls and inheritance relationships
func (p *KotlinParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for call expressions
	callQuery := `(call_expression 
		(simple_identifier) @call.target)`

	matches, err := p.tsParser.Query(rootNode, callQuery, "kotlin")
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

	// Also query for navigation expressions (method calls on objects)
	navQuery := `(navigation_expression 
		(navigation_suffix 
			(simple_identifier) @call.target))`

	matches, err = p.tsParser.Query(rootNode, navQuery, "kotlin")
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

	// Extract interface implementation relationships
	// These are already handled in extractClasses via extractSuperTypes
	// The dependencies are added there with type "extends"

	return nil
}
