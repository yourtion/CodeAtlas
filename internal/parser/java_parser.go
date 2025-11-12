package parser

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// JavaParser parses Java source code using Tree-sitter
type JavaParser struct {
	tsParser *TreeSitterParser
}

// NewJavaParser creates a new Java parser
func NewJavaParser(tsParser *TreeSitterParser) *JavaParser {
	return &JavaParser{
		tsParser: tsParser,
	}
}

// Parse parses a Java file and extracts symbols and dependencies
func (p *JavaParser) Parse(file ScannedFile) (*ParsedFile, error) {
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
	rootNode, parseErr := p.tsParser.Parse(content, "java")

	parsedFile := &ParsedFile{
		Path:     file.Path,
		Language: "java",
		Content:  content,
		RootNode: rootNode,
	}

	// Store file path for package inference
	parsedFile.Path = file.Path

	// If we have no root node at all, return error immediately
	if rootNode == nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to parse Java file: %v", parseErr),
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

	// Extract classes
	if err := p.extractClasses(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract interfaces
	if err := p.extractInterfaces(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract enums
	if err := p.extractEnums(rootNode, parsedFile, content); err != nil {
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
			Message: fmt.Sprintf("syntax error in Java file: %v", parseErr),
			Type:    "parse",
		}
	}

	return parsedFile, nil
}

// Helper functions for node traversal and span conversion

// extractSignature extracts a clean signature from a node
func (p *JavaParser) extractSignature(node *sitter.Node, content []byte) string {
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
func (p *JavaParser) findContainingFunction(node *sitter.Node, parsedFile *ParsedFile) string {
	current := node.Parent()

	for current != nil {
		// Check if this is a method declaration
		if current.Type() == "method_declaration" || current.Type() == "constructor_declaration" {
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
					// Check nested children (methods in nested classes)
					for _, nestedChild := range child.Children {
						if nestedChild.Node == current {
							return nestedChild.Name
						}
					}
				}
			}
		}
		current = current.Parent()
	}

	return ""
}

// extractPackage extracts the package declaration
func (p *JavaParser) extractPackage(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for package declaration
	query := `(package_declaration (scoped_identifier) @package.name)`

	matches, err := p.tsParser.Query(rootNode, query, "java")
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
	// Java convention: src/main/java/com/example/MyClass.java -> com.example
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
// Java convention: src/main/java/com/example/MyClass.java -> com.example
func (p *JavaParser) inferPackageFromPath(filePath string) string {
	// Normalize path separators
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	
	// Common Java source roots
	sourceRoots := []string{
		"src/main/java/",
		"src/test/java/",
		"src/",
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
func (p *JavaParser) extractImports(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Find the package symbol to use as the source for imports
	var packageSymbol string
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "package" {
			packageSymbol = symbol.Name
			break
		}
	}

	// Query for import statements
	query := `(import_declaration (scoped_identifier) @import.path)`

	matches, err := p.tsParser.Query(rootNode, query, "java")
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

	// Also handle wildcard imports
	wildcardQuery := `(import_declaration (asterisk) @import.wildcard)`
	matches, err = p.tsParser.Query(rootNode, wildcardQuery, "java")
	if err == nil {
		for _, match := range matches {
			// Get the parent import_declaration to extract the full path
			for _, capture := range match.Captures {
				importDecl := capture.Node.Parent()
				if importDecl != nil && importDecl.Type() == "import_declaration" {
					importPath := strings.TrimSpace(importDecl.Content(content))
					// Remove "import " and ";"
					importPath = strings.TrimPrefix(importPath, "import")
					importPath = strings.TrimSpace(importPath)
					importPath = strings.TrimSuffix(importPath, ";")
					importPath = strings.TrimSpace(importPath)

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
		}
	}

	return nil
}

// isExternalImport determines if an import path refers to an external module
func (p *JavaParser) isExternalImport(importPath string) bool {
	// Java standard library and javax are considered internal
	if strings.HasPrefix(importPath, "java.") || strings.HasPrefix(importPath, "javax.") {
		return false
	}
	
	// Everything else is external (third-party libraries)
	return true
}

// isExternalImportWithContext determines if an import is external based on current package
func (p *JavaParser) isExternalImportWithContext(importPath string, currentPackage string) bool {
	// Java standard library and javax are considered internal
	if strings.HasPrefix(importPath, "java.") || strings.HasPrefix(importPath, "javax.") {
		return false
	}
	
	// If import is from the same package or subpackage, it's internal to the project
	if currentPackage != "" {
		// Extract base package (first 2-3 segments, e.g., com.example from com.example.module)
		basePackage := p.extractBasePackage(currentPackage)
		importBasePackage := p.extractBasePackage(importPath)
		
		// If they share the same base package, consider it internal to the project
		if basePackage != "" && importBasePackage != "" && basePackage == importBasePackage {
			return false
		}
	}
	
	// Everything else is external (third-party libraries)
	return true
}

// extractBasePackage extracts the base package (first 2-3 segments)
// e.g., com.example.module.service -> com.example
func (p *JavaParser) extractBasePackage(packageName string) string {
	parts := strings.Split(packageName, ".")
	
	// For typical Java packages like com.example.*, org.project.*, etc.
	// we consider the first 2 segments as the base package
	if len(parts) >= 2 {
		return strings.Join(parts[:2], ".")
	}
	
	return packageName
}

// extractClasses extracts class declarations with fields, methods, and constructors
func (p *JavaParser) extractClasses(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for class declarations
	classQuery := `(class_declaration 
		name: (identifier) @class.name) @class.def`

	matches, err := p.tsParser.Query(rootNode, classQuery, "java")
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
			
			// Extract class members
			fields := p.extractClassFields(classNode, content)
			methods := p.extractClassMethods(classNode, content)
			constructors := p.extractConstructors(classNode, content)
			
			// Extract inheritance
			superClass := p.extractSuperClass(classNode, content)
			interfaces := p.extractImplementedInterfaces(classNode, content)
			
			// Extract Javadoc
			docstring := p.extractJavadoc(classNode, content)
			
			// Build signature
			signature := p.extractSignature(classNode, content)

			// Combine all children
			children := append(fields, methods...)
			children = append(children, constructors...)

			symbol := ParsedSymbol{
				Name:      fullyQualifiedName, // Use fully qualified name
				Kind:      "class",
				Signature: signature,
				Span:      nodeToSpan(classNode),
				Docstring: docstring,
				Node:      classNode,
				Children:  children,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)

			// Add inheritance dependencies
			if superClass != "" {
				dependency := ParsedDependency{
					Type:   "extends",
					Source: className,
					Target: superClass,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}

			// Add interface implementation dependencies
			for _, iface := range interfaces {
				dependency := ParsedDependency{
					Type:   "implements",
					Source: className,
					Target: iface,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractClassFields extracts fields from a class
func (p *JavaParser) extractClassFields(classNode *sitter.Node, content []byte) []ParsedSymbol {
	var fields []ParsedSymbol

	// Find class body
	classBody := findChildByType(classNode, "class_body")
	if classBody == nil {
		return fields
	}

	// Iterate through class body children
	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "field_declaration" {
			// Extract field declarators
			for j := 0; j < int(child.ChildCount()); j++ {
				declarator := child.Child(j)
				if declarator.Type() == "variable_declarator" {
					fieldName := ""
					
					// Find the identifier
					for k := 0; k < int(declarator.ChildCount()); k++ {
						nameNode := declarator.Child(k)
						if nameNode.Type() == "identifier" {
							fieldName = nameNode.Content(content)
							break
						}
					}

					if fieldName != "" {
						signature := p.extractSignature(child, content)
						docstring := p.extractJavadoc(child, content)

						field := ParsedSymbol{
							Name:      fieldName,
							Kind:      "field",
							Signature: signature,
							Span:      nodeToSpan(child),
							Docstring: docstring,
							Node:      child,
						}

						fields = append(fields, field)
					}
				}
			}
		}
	}

	return fields
}

// extractClassMethods extracts methods from a class
func (p *JavaParser) extractClassMethods(classNode *sitter.Node, content []byte) []ParsedSymbol {
	var methods []ParsedSymbol

	// Find class body
	classBody := findChildByType(classNode, "class_body")
	if classBody == nil {
		return methods
	}

	// Iterate through class body children
	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "method_declaration" {
			methodName := ""

			// Find the identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				nameNode := child.Child(j)
				if nameNode.Type() == "identifier" {
					methodName = nameNode.Content(content)
					break
				}
			}

			if methodName != "" {
				signature := p.extractSignature(child, content)
				docstring := p.extractJavadoc(child, content)

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

// extractConstructors extracts constructors from a class
func (p *JavaParser) extractConstructors(classNode *sitter.Node, content []byte) []ParsedSymbol {
	var constructors []ParsedSymbol

	// Find class body
	classBody := findChildByType(classNode, "class_body")
	if classBody == nil {
		return constructors
	}

	// Iterate through class body children
	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "constructor_declaration" {
			constructorName := ""

			// Find the identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				nameNode := child.Child(j)
				if nameNode.Type() == "identifier" {
					constructorName = nameNode.Content(content)
					break
				}
			}

			if constructorName != "" {
				signature := p.extractSignature(child, content)
				docstring := p.extractJavadoc(child, content)

				constructor := ParsedSymbol{
					Name:      constructorName,
					Kind:      "constructor",
					Signature: signature,
					Span:      nodeToSpan(child),
					Docstring: docstring,
					Node:      child,
				}

				constructors = append(constructors, constructor)
			}
		}
	}

	return constructors
}

// extractSuperClass extracts the parent class
func (p *JavaParser) extractSuperClass(classNode *sitter.Node, content []byte) string {
	// Find superclass
	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child.Type() == "superclass" {
			// Find the type_identifier
			typeIdent := findChildByType(child, "type_identifier")
			if typeIdent != nil {
				return typeIdent.Content(content)
			}
		}
	}

	return ""
}

// extractImplementedInterfaces extracts implemented interfaces
func (p *JavaParser) extractImplementedInterfaces(classNode *sitter.Node, content []byte) []string {
	var interfaces []string

	// Find super_interfaces
	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child.Type() == "super_interfaces" {
			// Find type_list
			typeList := findChildByType(child, "type_list")
			if typeList != nil {
				// Extract each type_identifier
				for j := 0; j < int(typeList.ChildCount()); j++ {
					typeNode := typeList.Child(j)
					if typeNode.Type() == "type_identifier" {
						interfaces = append(interfaces, typeNode.Content(content))
					}
				}
			}
		}
	}

	return interfaces
}

// extractInterfaces extracts interface declarations
func (p *JavaParser) extractInterfaces(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for interface declarations
	interfaceQuery := `(interface_declaration 
		name: (identifier) @interface.name) @interface.def`

	matches, err := p.tsParser.Query(rootNode, interfaceQuery, "java")
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
			
			// Extract extended interfaces
			extendedInterfaces := p.extractExtendedInterfaces(interfaceNode, content)
			
			// Extract Javadoc
			docstring := p.extractJavadoc(interfaceNode, content)
			
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

			// Add interface extension dependencies
			for _, extInterface := range extendedInterfaces {
				dependency := ParsedDependency{
					Type:   "extends",
					Source: interfaceName,
					Target: extInterface,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractInterfaceMethods extracts methods from an interface
func (p *JavaParser) extractInterfaceMethods(interfaceNode *sitter.Node, content []byte) []ParsedSymbol {
	var methods []ParsedSymbol

	// Find interface body
	interfaceBody := findChildByType(interfaceNode, "interface_body")
	if interfaceBody == nil {
		return methods
	}

	// Iterate through interface body children
	for i := 0; i < int(interfaceBody.ChildCount()); i++ {
		child := interfaceBody.Child(i)

		if child.Type() == "method_declaration" {
			methodName := ""

			// Find the identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				nameNode := child.Child(j)
				if nameNode.Type() == "identifier" {
					methodName = nameNode.Content(content)
					break
				}
			}

			if methodName != "" {
				signature := p.extractSignature(child, content)
				docstring := p.extractJavadoc(child, content)

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

// extractExtendedInterfaces extracts interfaces that this interface extends
func (p *JavaParser) extractExtendedInterfaces(interfaceNode *sitter.Node, content []byte) []string {
	var interfaces []string

	// Find extends_interfaces
	for i := 0; i < int(interfaceNode.ChildCount()); i++ {
		child := interfaceNode.Child(i)
		if child.Type() == "extends_interfaces" {
			// Find type_list
			typeList := findChildByType(child, "type_list")
			if typeList != nil {
				// Extract each type_identifier
				for j := 0; j < int(typeList.ChildCount()); j++ {
					typeNode := typeList.Child(j)
					if typeNode.Type() == "type_identifier" {
						interfaces = append(interfaces, typeNode.Content(content))
					}
				}
			}
		}
	}

	return interfaces
}

// extractEnums extracts enum declarations with constants
func (p *JavaParser) extractEnums(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for enum declarations
	enumQuery := `(enum_declaration 
		name: (identifier) @enum.name) @enum.def`

	matches, err := p.tsParser.Query(rootNode, enumQuery, "java")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var enumNode *sitter.Node
		var enumName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // enum.name
				enumName = capture.Node.Content(content)
			} else if capture.Index == 1 { // enum.def
				enumNode = capture.Node
			}
		}

		if enumNode != nil && enumName != "" {
			// Get package name for fully qualified name
			var packageName string
			for _, symbol := range parsedFile.Symbols {
				if symbol.Kind == "package" {
					packageName = symbol.Name
					break
				}
			}
			
			// Build fully qualified name
			fullyQualifiedName := enumName
			if packageName != "" {
				fullyQualifiedName = packageName + "." + enumName
			}
			
			// Extract enum constants
			constants := p.extractEnumConstants(enumNode, content)
			
			// Extract Javadoc
			docstring := p.extractJavadoc(enumNode, content)
			
			// Build signature
			signature := p.extractSignature(enumNode, content)

			symbol := ParsedSymbol{
				Name:      fullyQualifiedName, // Use fully qualified name
				Kind:      "enum",
				Signature: signature,
				Span:      nodeToSpan(enumNode),
				Docstring: docstring,
				Node:      enumNode,
				Children:  constants,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractEnumConstants extracts constants from an enum
func (p *JavaParser) extractEnumConstants(enumNode *sitter.Node, content []byte) []ParsedSymbol {
	var constants []ParsedSymbol

	// Find enum body
	enumBody := findChildByType(enumNode, "enum_body")
	if enumBody == nil {
		return constants
	}

	// Iterate through enum body children
	for i := 0; i < int(enumBody.ChildCount()); i++ {
		child := enumBody.Child(i)

		if child.Type() == "enum_constant" {
			constantName := ""

			// Find the identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				nameNode := child.Child(j)
				if nameNode.Type() == "identifier" {
					constantName = nameNode.Content(content)
					break
				}
			}

			if constantName != "" {
				signature := child.Content(content)
				docstring := p.extractJavadoc(child, content)

				constant := ParsedSymbol{
					Name:      constantName,
					Kind:      "enum_constant",
					Signature: signature,
					Span:      nodeToSpan(child),
					Docstring: docstring,
					Node:      child,
				}

				constants = append(constants, constant)
			}
		}
	}

	return constants
}

// extractAnnotations extracts annotation definitions and usage
func (p *JavaParser) extractAnnotations(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for annotation type declarations
	annotationQuery := `(annotation_type_declaration 
		name: (identifier) @annotation.name) @annotation.def`

	matches, err := p.tsParser.Query(rootNode, annotationQuery, "java")
	if err != nil {
		return err
	}

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
			// Extract Javadoc
			docstring := p.extractJavadoc(annotationNode, content)
			
			// Build signature
			signature := p.extractSignature(annotationNode, content)

			symbol := ParsedSymbol{
				Name:      annotationName,
				Kind:      "annotation",
				Signature: signature,
				Span:      nodeToSpan(annotationNode),
				Docstring: docstring,
				Node:      annotationNode,
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	// Also track annotation usage (marker annotations)
	markerQuery := `(marker_annotation 
		name: (identifier) @annotation.usage)`

	matches, err = p.tsParser.Query(rootNode, markerQuery, "java")
	if err == nil {
		seenAnnotations := make(map[string]bool)
		
		for _, match := range matches {
			for _, capture := range match.Captures {
				annotationName := capture.Node.Content(content)
				
				// Only add unique annotations
				if seenAnnotations[annotationName] {
					continue
				}
				seenAnnotations[annotationName] = true

				// Find what this annotation is applied to
				parent := capture.Node.Parent()
				if parent != nil && parent.Type() == "marker_annotation" {
					parent = parent.Parent()
				}
				
				// Create annotation usage dependency
				if parent != nil {
					targetName := p.findAnnotationTarget(parent, content)
					if targetName != "" {
						dependency := ParsedDependency{
							Type:   "annotated_with",
							Source: targetName,
							Target: annotationName,
						}
						parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
					}
				}
			}
		}
	}

	return nil
}

// findAnnotationTarget finds the name of the element that an annotation is applied to
func (p *JavaParser) findAnnotationTarget(node *sitter.Node, content []byte) string {
	// Look for class, method, field, etc.
	for node != nil {
		switch node.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration":
			// Find the identifier
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "identifier" {
					return child.Content(content)
				}
			}
		case "method_declaration", "constructor_declaration":
			// Find the identifier
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "identifier" {
					return child.Content(content)
				}
			}
		case "field_declaration":
			// Find the variable_declarator
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "variable_declarator" {
					for j := 0; j < int(child.ChildCount()); j++ {
						nameNode := child.Child(j)
						if nameNode.Type() == "identifier" {
							return nameNode.Content(content)
						}
					}
				}
			}
		}
		node = node.Parent()
	}
	return ""
}

// extractJavadoc extracts Javadoc comments
func (p *JavaParser) extractJavadoc(node *sitter.Node, content []byte) string {
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
		if sibling.Type() == "block_comment" {
			commentText := sibling.Content(content)

			// Handle Javadoc comments (/** ... */)
			if strings.HasPrefix(commentText, "/**") {
				commentText = strings.TrimPrefix(commentText, "/**")
				commentText = strings.TrimSuffix(commentText, "*/")
				commentText = strings.TrimSpace(commentText)

				// Clean up Javadoc formatting
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
			}

			comments = append([]string{commentText}, comments...)
		} else if sibling.Type() == "line_comment" {
			commentText := sibling.Content(content)
			// Handle single-line comments
			commentText = strings.TrimPrefix(commentText, "//")
			commentText = strings.TrimSpace(commentText)
			comments = append([]string{commentText}, comments...)
		} else if sibling.Type() != "block_comment" && sibling.Type() != "line_comment" {
			break
		}
	}

	return strings.Join(comments, "\n")
}

// extractCallRelationships extracts method calls and inheritance relationships
func (p *JavaParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for method invocations
	callQuery := `(method_invocation 
		name: (identifier) @call.target)`

	matches, err := p.tsParser.Query(rootNode, callQuery, "java")
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

	// Also query for object creation (constructor calls)
	constructorQuery := `(object_creation_expression 
		type: (type_identifier) @call.constructor)`

	matches, err = p.tsParser.Query(rootNode, constructorQuery, "java")
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

	// Extract inheritance relationships (already handled in extractClasses and extractInterfaces)
	// The dependencies are added there with type "extends" and "implements"

	// Extract annotation usage relationships (already handled in extractAnnotations)
	// The dependencies are added there with type "annotated_with"

	return nil
}

// handleGenerics extracts type information from generic type parameters
// This is a helper function for future use in handling generics
func (p *JavaParser) handleGenerics(node *sitter.Node, content []byte) []string {
	var typeParams []string

	// Find type_parameters node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_parameters" {
			// Extract each type_parameter
			for j := 0; j < int(child.ChildCount()); j++ {
				typeParam := child.Child(j)
				if typeParam.Type() == "type_parameter" {
					// Find the type_identifier
					for k := 0; k < int(typeParam.ChildCount()); k++ {
						typeIdent := typeParam.Child(k)
						if typeIdent.Type() == "type_identifier" {
							typeParams = append(typeParams, typeIdent.Content(content))
						}
					}
				}
			}
		}
	}

	return typeParams
}

// extractGenericTypeArguments extracts type arguments from generic types
// This helps track relationships through generics like List<String>
func (p *JavaParser) extractGenericTypeArguments(node *sitter.Node, content []byte) []string {
	var typeArgs []string

	// Find type_arguments node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_arguments" {
			// Extract each type argument
			for j := 0; j < int(child.ChildCount()); j++ {
				typeArg := child.Child(j)
				if typeArg.Type() == "type_identifier" {
					typeArgs = append(typeArgs, typeArg.Content(content))
				} else if typeArg.Type() == "generic_type" {
					// Recursively extract nested generics
					nestedArgs := p.extractGenericTypeArguments(typeArg, content)
					typeArgs = append(typeArgs, nestedArgs...)
				}
			}
		}
	}

	return typeArgs
}
