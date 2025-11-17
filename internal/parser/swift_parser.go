package parser

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// SwiftParser parses Swift source code using Tree-sitter
type SwiftParser struct {
	tsParser *TreeSitterParser
}

// NewSwiftParser creates a new Swift parser
func NewSwiftParser(tsParser *TreeSitterParser) *SwiftParser {
	return &SwiftParser{
		tsParser: tsParser,
	}
}

// Parse parses a Swift file and extracts symbols and dependencies
func (p *SwiftParser) Parse(file ScannedFile) (*ParsedFile, error) {
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
	rootNode, parseErr := p.tsParser.Parse(content, "swift")

	parsedFile := &ParsedFile{
		Path:     file.Path,
		Language: "swift",
		Content:  content,
		RootNode: rootNode,
	}

	// If we have no root node at all, return error immediately
	if rootNode == nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to parse Swift file: %v", parseErr),
			Type:    "parse",
		}
	}

	// Extract imports
	if err := p.extractImports(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract classes
	if err := p.extractClasses(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract structs
	if err := p.extractStructs(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract enums
	if err := p.extractEnums(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract protocols
	if err := p.extractProtocols(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract extensions
	if err := p.extractExtensions(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract functions
	if err := p.extractFunctions(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract properties
	if err := p.extractProperties(rootNode, parsedFile, content); err != nil {
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
			Message: fmt.Sprintf("syntax error in Swift file: %v", parseErr),
			Type:    "parse",
		}
	}

	return parsedFile, nil
}

// Helper functions for node traversal and span conversion

// extractSignature extracts a clean signature from a node
func (p *SwiftParser) extractSignature(node *sitter.Node, content []byte) string {
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
func (p *SwiftParser) findContainingFunction(node *sitter.Node, parsedFile *ParsedFile) string {
	current := node.Parent()

	for current != nil {
		// Check if this is a function declaration
		if current.Type() == "function_declaration" {
			// Find the matching symbol in our parsed symbols
			for _, symbol := range parsedFile.Symbols {
				if symbol.Node == current {
					return symbol.Name
				}
				// Also check children (methods in classes/structs)
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

// isInsideType checks if a node is inside a class, struct, enum, or protocol definition
func (p *SwiftParser) isInsideType(node *sitter.Node) bool {
	current := node.Parent()
	for current != nil {
		nodeType := current.Type()
		if nodeType == "class_declaration" || 
		   nodeType == "struct_declaration" || 
		   nodeType == "enum_declaration" || 
		   nodeType == "protocol_declaration" {
			return true
		}
		current = current.Parent()
	}
	return false
}

// extractImports extracts import statements with framework classification
func (p *SwiftParser) extractImports(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for import declarations
	query := `(import_declaration) @import.decl`

	matches, err := p.tsParser.Query(rootNode, query, "swift")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			importText := capture.Node.Content(content)
			
			// Parse import statement: "import Foundation" or "import UIKit.UIView"
			importText = strings.TrimPrefix(importText, "import")
			importText = strings.TrimSpace(importText)
			
			// Extract the module name (first part before any dot)
			parts := strings.Fields(importText)
			if len(parts) > 0 {
				importPath := parts[0]
				
				dependency := ParsedDependency{
					Type:         "import",
					Source:       "", // Swift doesn't have packages like Java/Kotlin
					Target:       importPath,
					TargetModule: importPath,
					IsExternal:   p.isExternalImport(importPath),
				}

				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractClasses extracts class declarations
func (p *SwiftParser) extractClasses(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for class declarations
	classQuery := `(class_declaration 
		name: (type_identifier) @class.name) @class.def`

	matches, err := p.tsParser.Query(rootNode, classQuery, "swift")
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
			// Check if this is actually a class (not a struct/enum)
			isClass := false
			for i := 0; i < int(classNode.ChildCount()); i++ {
				child := classNode.Child(i)
				if child.Type() == "class" {
					isClass = true
					break
				}
			}
			
			if !isClass {
				continue
			}
			
			// Extract class members
			properties := p.extractTypeProperties(classNode, content)
			methods := p.extractTypeMethods(classNode, content)
			
			// Extract protocol conformance
			protocols := p.extractProtocolConformance(classNode, content)
			
			// Extract superclass
			superclass := p.extractSuperclass(classNode, content)
			
			// Extract Swift documentation
			docstring := p.extractSwiftDoc(classNode, content)
			
			// Build signature
			signature := p.extractSignature(classNode, content)

			symbol := ParsedSymbol{
				Name:      className,
				Kind:      "class",
				Signature: signature,
				Span:      nodeToSpan(classNode),
				Docstring: docstring,
				Node:      classNode,
				Children:  append(properties, methods...),
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)

			// Add superclass dependency
			if superclass != "" {
				dependency := ParsedDependency{
					Type:   "extends",
					Source: className,
					Target: superclass,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}

			// Add protocol conformance dependencies
			for _, protocol := range protocols {
				dependency := ParsedDependency{
					Type:   "conforms",
					Source: className,
					Target: protocol,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractStructs extracts struct declarations
func (p *SwiftParser) extractStructs(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for struct declarations (structs use class_declaration node type)
	structQuery := `(class_declaration 
		name: (type_identifier) @struct.name) @struct.def`

	matches, err := p.tsParser.Query(rootNode, structQuery, "swift")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var structNode *sitter.Node
		var structName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // struct.name
				structName = capture.Node.Content(content)
			} else if capture.Index == 1 { // struct.def
				structNode = capture.Node
			}
		}

		if structNode != nil && structName != "" {
			// Check if this is actually a struct (not a class)
			isStruct := false
			for i := 0; i < int(structNode.ChildCount()); i++ {
				child := structNode.Child(i)
				if child.Type() == "struct" {
					isStruct = true
					break
				}
			}
			
			if !isStruct {
				continue
			}
			// Extract struct members
			properties := p.extractTypeProperties(structNode, content)
			methods := p.extractTypeMethods(structNode, content)
			
			// Extract protocol conformance
			protocols := p.extractProtocolConformance(structNode, content)
			
			// Extract Swift documentation
			docstring := p.extractSwiftDoc(structNode, content)
			
			// Build signature
			signature := p.extractSignature(structNode, content)

			symbol := ParsedSymbol{
				Name:      structName,
				Kind:      "struct",
				Signature: signature,
				Span:      nodeToSpan(structNode),
				Docstring: docstring,
				Node:      structNode,
				Children:  append(properties, methods...),
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)

			// Add protocol conformance dependencies
			for _, protocol := range protocols {
				dependency := ParsedDependency{
					Type:   "conforms",
					Source: structName,
					Target: protocol,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractEnums extracts enum declarations with associated values
func (p *SwiftParser) extractEnums(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for enum declarations (enums use class_declaration node type)
	enumQuery := `(class_declaration 
		name: (type_identifier) @enum.name) @enum.def`

	matches, err := p.tsParser.Query(rootNode, enumQuery, "swift")
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
			// Check if this is actually an enum (not a class/struct)
			isEnum := false
			for i := 0; i < int(enumNode.ChildCount()); i++ {
				child := enumNode.Child(i)
				if child.Type() == "enum" {
					isEnum = true
					break
				}
			}
			
			if !isEnum {
				continue
			}
			// Extract enum cases
			cases := p.extractEnumCases(enumNode, content)
			
			// Extract methods (enums can have methods in Swift)
			methods := p.extractTypeMethods(enumNode, content)
			
			// Extract protocol conformance
			protocols := p.extractProtocolConformance(enumNode, content)
			
			// Extract Swift documentation
			docstring := p.extractSwiftDoc(enumNode, content)
			
			// Build signature
			signature := p.extractSignature(enumNode, content)

			symbol := ParsedSymbol{
				Name:      enumName,
				Kind:      "enum",
				Signature: signature,
				Span:      nodeToSpan(enumNode),
				Docstring: docstring,
				Node:      enumNode,
				Children:  append(cases, methods...),
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)

			// Add protocol conformance dependencies
			for _, protocol := range protocols {
				dependency := ParsedDependency{
					Type:   "conforms",
					Source: enumName,
					Target: protocol,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractProtocols extracts protocol declarations
func (p *SwiftParser) extractProtocols(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for protocol declarations
	protocolQuery := `(protocol_declaration 
		name: (type_identifier) @protocol.name) @protocol.def`

	matches, err := p.tsParser.Query(rootNode, protocolQuery, "swift")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var protocolNode *sitter.Node
		var protocolName string

		for _, capture := range match.Captures {
			if capture.Index == 0 { // protocol.name
				protocolName = capture.Node.Content(content)
			} else if capture.Index == 1 { // protocol.def
				protocolNode = capture.Node
			}
		}

		if protocolNode != nil && protocolName != "" {
			// Extract protocol requirements (methods and properties)
			methods := p.extractProtocolMethods(protocolNode, content)
			properties := p.extractProtocolProperties(protocolNode, content)
			
			// Extract inherited protocols
			inheritedProtocols := p.extractInheritedProtocols(protocolNode, content)
			
			// Extract Swift documentation
			docstring := p.extractSwiftDoc(protocolNode, content)
			
			// Build signature
			signature := p.extractSignature(protocolNode, content)

			symbol := ParsedSymbol{
				Name:      protocolName,
				Kind:      "protocol",
				Signature: signature,
				Span:      nodeToSpan(protocolNode),
				Docstring: docstring,
				Node:      protocolNode,
				Children:  append(properties, methods...),
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)

			// Add protocol inheritance dependencies
			for _, inherited := range inheritedProtocols {
				dependency := ParsedDependency{
					Type:   "extends",
					Source: protocolName,
					Target: inherited,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractExtensions extracts extension declarations
func (p *SwiftParser) extractExtensions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for extension declarations (extensions use class_declaration node type)
	extensionQuery := `(class_declaration) @extension.def`

	matches, err := p.tsParser.Query(rootNode, extensionQuery, "swift")
	if err != nil {
		return err
	}

	for _, match := range matches {
		var extensionNode *sitter.Node

		for _, capture := range match.Captures {
			extensionNode = capture.Node
		}

		if extensionNode != nil {
			// Check if this is actually an extension (not a class/struct/enum)
			isExtension := false
			var extendedType string
			
			for i := 0; i < int(extensionNode.ChildCount()); i++ {
				child := extensionNode.Child(i)
				if child.Type() == "extension" {
					isExtension = true
				} else if child.Type() == "user_type" {
					// Extract the type being extended
					for j := 0; j < int(child.ChildCount()); j++ {
						typeIdent := child.Child(j)
						if typeIdent.Type() == "type_identifier" {
							extendedType = typeIdent.Content(content)
							break
						}
					}
				}
			}
			
			if !isExtension || extendedType == "" {
				continue
			}
			// Extract extension members
			properties := p.extractTypeProperties(extensionNode, content)
			methods := p.extractTypeMethods(extensionNode, content)
			
			// Extract protocol conformance added by extension
			protocols := p.extractProtocolConformance(extensionNode, content)
			
			// Extract Swift documentation
			docstring := p.extractSwiftDoc(extensionNode, content)
			
			// Build signature
			signature := p.extractSignature(extensionNode, content)

			// Create a unique name for the extension
			extensionName := fmt.Sprintf("extension_%s", extendedType)

			symbol := ParsedSymbol{
				Name:      extensionName,
				Kind:      "extension",
				Signature: signature,
				Span:      nodeToSpan(extensionNode),
				Docstring: docstring,
				Node:      extensionNode,
				Children:  append(properties, methods...),
			}

			parsedFile.Symbols = append(parsedFile.Symbols, symbol)

			// Add extension-to-type dependency
			dependency := ParsedDependency{
				Type:   "extends",
				Source: extensionName,
				Target: extendedType,
			}
			parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)

			// Add protocol conformance dependencies
			for _, protocol := range protocols {
				dependency := ParsedDependency{
					Type:   "conforms",
					Source: extensionName,
					Target: protocol,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractFunctions extracts function declarations (top-level functions)
func (p *SwiftParser) extractFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for function declarations
	funcQuery := `(function_declaration 
		name: (simple_identifier) @func.name) @func.def`

	matches, err := p.tsParser.Query(rootNode, funcQuery, "swift")
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
			// Skip if this is a method (inside a type)
			if p.isInsideType(funcNode) {
				continue
			}

			// Extract Swift documentation
			docstring := p.extractSwiftDoc(funcNode, content)
			
			// Build signature
			signature := p.extractSignature(funcNode, content)

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

// extractProperties extracts top-level property declarations
func (p *SwiftParser) extractProperties(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for property declarations
	propertyQuery := `(property_declaration 
		(pattern 
			(simple_identifier) @prop.name)) @prop.def`

	matches, err := p.tsParser.Query(rootNode, propertyQuery, "swift")
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
			// Skip if this is inside a type
			if p.isInsideType(propNode) {
				continue
			}

			// Check for property observers (willSet, didSet)
			hasObservers := p.hasPropertyObservers(propNode, content)
			kind := "property"
			if hasObservers {
				kind = "property_observer"
			}

			// Extract Swift documentation
			docstring := p.extractSwiftDoc(propNode, content)
			
			// Build signature
			signature := p.extractSignature(propNode, content)

			symbol := ParsedSymbol{
				Name:      propName,
				Kind:      kind,
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

// extractCallRelationships extracts function calls and method calls
func (p *SwiftParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for call expressions (function calls)
	callQuery := `(call_expression 
		(simple_identifier) @call.target)`

	matches, err := p.tsParser.Query(rootNode, callQuery, "swift")
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

	// Query for navigation expressions (method calls with dot notation)
	// This handles optional chaining as well (e.g., object?.method())
	navQuery := `(navigation_expression 
		(navigation_suffix 
			(simple_identifier) @call.method))`

	matches, err = p.tsParser.Query(rootNode, navQuery, "swift")
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

	// Protocol conformance and extension relationships are already handled
	// in extractClasses, extractStructs, extractEnums, and extractExtensions

	return nil
}


// Helper methods for symbol extraction

// extractTypeProperties extracts properties from a class, struct, enum, or extension
func (p *SwiftParser) extractTypeProperties(typeNode *sitter.Node, content []byte) []ParsedSymbol {
	var properties []ParsedSymbol

	// Find the class_body or struct_body
	var body *sitter.Node
	for i := 0; i < int(typeNode.ChildCount()); i++ {
		child := typeNode.Child(i)
		if child.Type() == "class_body" || child.Type() == "struct_body" || 
		   child.Type() == "enum_class_body" || child.Type() == "protocol_body" {
			body = child
			break
		}
	}

	if body == nil {
		return properties
	}

	// Iterate through body children
	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		
		if child.Type() == "property_declaration" {
			propName := ""
			
			// Find the pattern with simple_identifier
			pattern := findChildByType(child, "pattern")
			if pattern != nil {
				identifier := findChildByType(pattern, "simple_identifier")
				if identifier != nil {
					propName = identifier.Content(content)
				}
			}

			if propName != "" {
				// Check for property observers
				hasObservers := p.hasPropertyObservers(child, content)
				kind := "property"
				if hasObservers {
					kind = "property_observer"
				}

				signature := p.extractSignature(child, content)
				docstring := p.extractSwiftDoc(child, content)

				property := ParsedSymbol{
					Name:      propName,
					Kind:      kind,
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

// extractTypeMethods extracts methods from a class, struct, enum, or extension
func (p *SwiftParser) extractTypeMethods(typeNode *sitter.Node, content []byte) []ParsedSymbol {
	var methods []ParsedSymbol

	// Find the class_body or struct_body
	var body *sitter.Node
	for i := 0; i < int(typeNode.ChildCount()); i++ {
		child := typeNode.Child(i)
		if child.Type() == "class_body" || child.Type() == "struct_body" || 
		   child.Type() == "enum_class_body" || child.Type() == "protocol_body" {
			body = child
			break
		}
	}

	if body == nil {
		return methods
	}

	// Iterate through body children
	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		
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
				docstring := p.extractSwiftDoc(child, content)

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

// extractProtocolConformance extracts protocols that a type conforms to
func (p *SwiftParser) extractProtocolConformance(typeNode *sitter.Node, content []byte) []string {
	var protocols []string

	// Find inheritance_specifier nodes
	for i := 0; i < int(typeNode.ChildCount()); i++ {
		child := typeNode.Child(i)
		if child.Type() == "inheritance_specifier" {
			// Extract user_type which contains type_identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				userType := child.Child(j)
				if userType.Type() == "user_type" {
					for k := 0; k < int(userType.ChildCount()); k++ {
						typeIdent := userType.Child(k)
						if typeIdent.Type() == "type_identifier" {
							protocols = append(protocols, typeIdent.Content(content))
						}
					}
				} else if userType.Type() == "type_identifier" {
					protocols = append(protocols, userType.Content(content))
				}
			}
		}
	}

	return protocols
}

// extractSuperclass extracts the superclass from a class declaration
func (p *SwiftParser) extractSuperclass(classNode *sitter.Node, content []byte) string {
	// In Swift, the first inheritance_specifier is typically the superclass
	// Subsequent ones are protocols
	
	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child.Type() == "inheritance_specifier" {
			// Extract user_type which contains type_identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				userType := child.Child(j)
				if userType.Type() == "user_type" {
					for k := 0; k < int(userType.ChildCount()); k++ {
						typeIdent := userType.Child(k)
						if typeIdent.Type() == "type_identifier" {
							// Return the first one as superclass
							return typeIdent.Content(content)
						}
					}
				} else if userType.Type() == "type_identifier" {
					return userType.Content(content)
				}
			}
			// Only return the first inheritance_specifier as superclass
			break
		}
	}

	return ""
}

// extractEnumCases extracts cases from an enum declaration
func (p *SwiftParser) extractEnumCases(enumNode *sitter.Node, content []byte) []ParsedSymbol {
	var cases []ParsedSymbol

	// Find enum cases
	for i := 0; i < int(enumNode.ChildCount()); i++ {
		child := enumNode.Child(i)
		
		if child.Type() == "enum_class_body" {
			// Iterate through body
			for j := 0; j < int(child.ChildCount()); j++ {
				bodyChild := child.Child(j)
				
				if bodyChild.Type() == "enum_entry" {
					caseName := ""
					
					// Find the simple_identifier
					for k := 0; k < int(bodyChild.ChildCount()); k++ {
						nameNode := bodyChild.Child(k)
						if nameNode.Type() == "simple_identifier" {
							caseName = nameNode.Content(content)
							break
						}
					}

					if caseName != "" {
						signature := bodyChild.Content(content)

						caseSymbol := ParsedSymbol{
							Name:      caseName,
							Kind:      "enum_case",
							Signature: signature,
							Span:      nodeToSpan(bodyChild),
							Node:      bodyChild,
						}

						cases = append(cases, caseSymbol)
					}
				}
			}
		}
	}

	return cases
}

// extractProtocolMethods extracts method requirements from a protocol
func (p *SwiftParser) extractProtocolMethods(protocolNode *sitter.Node, content []byte) []ParsedSymbol {
	var methods []ParsedSymbol

	// Find protocol body
	for i := 0; i < int(protocolNode.ChildCount()); i++ {
		child := protocolNode.Child(i)
		
		if child.Type() == "protocol_body" {
			// Iterate through body
			for j := 0; j < int(child.ChildCount()); j++ {
				bodyChild := child.Child(j)
				
				if bodyChild.Type() == "protocol_function_declaration" || 
				   bodyChild.Type() == "function_declaration" {
					methodName := ""
					
					// Find the simple_identifier
					for k := 0; k < int(bodyChild.ChildCount()); k++ {
						nameNode := bodyChild.Child(k)
						if nameNode.Type() == "simple_identifier" {
							methodName = nameNode.Content(content)
							break
						}
					}

					if methodName != "" {
						signature := p.extractSignature(bodyChild, content)
						docstring := p.extractSwiftDoc(bodyChild, content)

						method := ParsedSymbol{
							Name:      methodName,
							Kind:      "method",
							Signature: signature,
							Span:      nodeToSpan(bodyChild),
							Docstring: docstring,
							Node:      bodyChild,
						}

						methods = append(methods, method)
					}
				}
			}
		}
	}

	return methods
}

// extractProtocolProperties extracts property requirements from a protocol
func (p *SwiftParser) extractProtocolProperties(protocolNode *sitter.Node, content []byte) []ParsedSymbol {
	var properties []ParsedSymbol

	// Find protocol body
	for i := 0; i < int(protocolNode.ChildCount()); i++ {
		child := protocolNode.Child(i)
		
		if child.Type() == "protocol_body" {
			// Iterate through body
			for j := 0; j < int(child.ChildCount()); j++ {
				bodyChild := child.Child(j)
				
				if bodyChild.Type() == "protocol_property_declaration" || 
				   bodyChild.Type() == "property_declaration" {
					propName := ""
					
					// Find the pattern with simple_identifier
					pattern := findChildByType(bodyChild, "pattern")
					if pattern != nil {
						identifier := findChildByType(pattern, "simple_identifier")
						if identifier != nil {
							propName = identifier.Content(content)
						}
					}

					if propName != "" {
						signature := p.extractSignature(bodyChild, content)
						docstring := p.extractSwiftDoc(bodyChild, content)

						property := ParsedSymbol{
							Name:      propName,
							Kind:      "property",
							Signature: signature,
							Span:      nodeToSpan(bodyChild),
							Docstring: docstring,
							Node:      bodyChild,
						}

						properties = append(properties, property)
					}
				}
			}
		}
	}

	return properties
}

// extractInheritedProtocols extracts protocols that a protocol inherits from
func (p *SwiftParser) extractInheritedProtocols(protocolNode *sitter.Node, content []byte) []string {
	var protocols []string

	// Find inheritance_specifier nodes
	for i := 0; i < int(protocolNode.ChildCount()); i++ {
		child := protocolNode.Child(i)
		if child.Type() == "inheritance_specifier" {
			// Extract user_type which contains type_identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				userType := child.Child(j)
				if userType.Type() == "user_type" {
					for k := 0; k < int(userType.ChildCount()); k++ {
						typeIdent := userType.Child(k)
						if typeIdent.Type() == "type_identifier" {
							protocols = append(protocols, typeIdent.Content(content))
						}
					}
				} else if userType.Type() == "type_identifier" {
					protocols = append(protocols, userType.Content(content))
				}
			}
		}
	}

	return protocols
}

// hasPropertyObservers checks if a property has willSet or didSet observers
func (p *SwiftParser) hasPropertyObservers(propNode *sitter.Node, content []byte) bool {
	propText := propNode.Content(content)
	return strings.Contains(propText, "willSet") || strings.Contains(propText, "didSet")
}

// extractSwiftDoc extracts Swift documentation comments
func (p *SwiftParser) extractSwiftDoc(node *sitter.Node, content []byte) string {
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

			// Handle Swift documentation comments (/// or /** ... */)
			if strings.HasPrefix(commentText, "///") {
				commentText = strings.TrimPrefix(commentText, "///")
				commentText = strings.TrimSpace(commentText)
			} else if strings.HasPrefix(commentText, "/**") {
				commentText = strings.TrimPrefix(commentText, "/**")
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

// isExternalImport determines if an import refers to an external framework
func (p *SwiftParser) isExternalImport(importPath string) bool {
	// Common iOS/macOS frameworks are considered external
	externalFrameworks := []string{
		"Foundation",
		"UIKit",
		"SwiftUI",
		"Combine",
		"CoreData",
		"CoreGraphics",
		"CoreLocation",
		"MapKit",
		"AVFoundation",
		"WebKit",
		"AppKit", // macOS
		"Cocoa",  // macOS
	}

	for _, framework := range externalFrameworks {
		if importPath == framework || strings.HasPrefix(importPath, framework+".") {
			return true
		}
	}

	// Everything else is considered internal (project modules)
	return false
}
