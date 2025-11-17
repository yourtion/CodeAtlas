package parser

import (
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ObjCParser parses Objective-C source code using Tree-sitter
type ObjCParser struct {
	tsParser *TreeSitterParser
}

// NewObjCParser creates a new Objective-C parser
func NewObjCParser(tsParser *TreeSitterParser) *ObjCParser {
	return &ObjCParser{
		tsParser: tsParser,
	}
}

// Parse parses an Objective-C file and extracts symbols and dependencies
func (p *ObjCParser) Parse(file ScannedFile) (*ParsedFile, error) {
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
	rootNode, parseErr := p.tsParser.Parse(content, "objc")

	parsedFile := &ParsedFile{
		Path:     file.Path,
		Language: "objc",
		Content:  content,
		RootNode: rootNode,
	}

	// If we have no root node at all, return error immediately
	if rootNode == nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("failed to parse Objective-C file: %v", parseErr),
			Type:    "parse",
		}
	}

	// Determine if this is a header or implementation file
	isHeader := strings.HasSuffix(file.Path, ".h")

	// Extract imports
	if err := p.extractImports(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	if isHeader {
		// Extract from header file
		if err := p.extractInterfaces(rootNode, parsedFile, content); err != nil {
			// Non-fatal, continue
		}
	} else {
		// Extract from implementation file
		if err := p.extractImplementations(rootNode, parsedFile, content); err != nil {
			// Non-fatal, continue
		}
	}

	// Extract protocols (can be in both header and implementation)
	if err := p.extractProtocols(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract categories (can be in both header and implementation)
	if err := p.extractCategories(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Extract call relationships (message sends)
	if err := p.extractCallRelationships(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Return parse error if there was one, but with partial results
	if parseErr != nil {
		return parsedFile, &DetailedParseError{
			File:    file.Path,
			Message: fmt.Sprintf("syntax error in Objective-C file: %v", parseErr),
			Type:    "parse",
		}
	}

	return parsedFile, nil
}

// Helper functions for node traversal and span conversion

// extractSignature extracts a clean signature from a node
func (p *ObjCParser) extractSignature(node *sitter.Node, content []byte) string {
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

// findContainingMethod finds the name of the method containing a node
func (p *ObjCParser) findContainingMethod(node *sitter.Node, parsedFile *ParsedFile) string {
	current := node.Parent()

	for current != nil {
		// Check if this is a method declaration or definition
		if current.Type() == "method_declaration" || current.Type() == "method_definition" {
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

// findPairedFile locates the corresponding .h or .m file
func (p *ObjCParser) findPairedFile(currentPath string) string {
	dir := filepath.Dir(currentPath)
	baseName := strings.TrimSuffix(filepath.Base(currentPath), filepath.Ext(currentPath))

	if strings.HasSuffix(currentPath, ".h") {
		// Look for implementation file
		extensions := []string{".m", ".mm"}
		for _, ext := range extensions {
			implPath := filepath.Join(dir, baseName+ext)
			// Note: In a real implementation, we would check if the file exists
			// For now, we just return the expected path
			return implPath
		}
	} else {
		// Look for header file
		headerPath := filepath.Join(dir, baseName+".h")
		return headerPath
	}

	return ""
}

// extractImports extracts #import and #include statements
func (p *ObjCParser) extractImports(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// In Objective-C tree-sitter grammar, #import is represented as preproc_include
	includeQuery := `(preproc_include) @include.decl`

	matches, err := p.tsParser.Query(rootNode, includeQuery, "objc")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			includeNode := capture.Node
			
			// Look for system_lib_string (<...>) or string_literal ("...")
			var importPath string
			for i := 0; i < int(includeNode.ChildCount()); i++ {
				child := includeNode.Child(i)
				if child.Type() == "system_lib_string" {
					// System import like <Foundation/Foundation.h>
					text := child.Content(content)
					importPath = strings.Trim(text, "<>")
				} else if child.Type() == "string_literal" {
					// Local import like "MyClass.h"
					text := child.Content(content)
					importPath = strings.Trim(text, "\"")
				}
			}

			if importPath != "" {
				dependency := ParsedDependency{
					Type:         "import",
					Source:       "",
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

// extractInterfaces extracts @interface declarations from headers
func (p *ObjCParser) extractInterfaces(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for class interface declarations
	interfaceQuery := `(class_interface) @interface.def`

	matches, err := p.tsParser.Query(rootNode, interfaceQuery, "objc")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			interfaceNode := capture.Node
			
			// Check if this is a category (has parentheses) - skip if so
			hasParentheses := false
			for i := 0; i < int(interfaceNode.ChildCount()); i++ {
				child := interfaceNode.Child(i)
				if child.Type() == "(" {
					hasParentheses = true
					break
				}
			}
			
			// Skip categories - they're handled by extractCategories
			if hasParentheses {
				continue
			}
			
			// Find the class name (first identifier after @interface)
			var interfaceName string
			for i := 0; i < int(interfaceNode.ChildCount()); i++ {
				child := interfaceNode.Child(i)
				if child.Type() == "identifier" {
					interfaceName = child.Content(content)
					break
				}
			}

			if interfaceName != "" {
			// Extract properties
			properties := p.extractProperties(interfaceNode, content)

			// Extract methods
			methods := p.extractMethods(interfaceNode, content)

			// Extract superclass
			superclass := p.extractSuperclass(interfaceNode, content)

			// Extract protocol conformance
			protocols := p.extractProtocolConformance(interfaceNode, content)

			// Extract header documentation
			docstring := p.extractHeaderDoc(interfaceNode, content)

			// Build signature
			signature := p.extractSignature(interfaceNode, content)

			symbol := ParsedSymbol{
				Name:      interfaceName,
				Kind:      "interface",
				Signature: signature,
				Span:      nodeToSpan(interfaceNode),
				Docstring: docstring,
				Node:      interfaceNode,
				Children:  append(properties, methods...),
			}

				parsedFile.Symbols = append(parsedFile.Symbols, symbol)

				// Add superclass dependency
				if superclass != "" {
					dependency := ParsedDependency{
						Type:   "extends",
						Source: interfaceName,
						Target: superclass,
					}
					parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
				}

				// Add protocol conformance dependencies
				for _, protocol := range protocols {
					dependency := ParsedDependency{
						Type:   "conforms",
						Source: interfaceName,
						Target: protocol,
					}
					parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
				}
			}
		}
	}

	return nil
}

// extractImplementations extracts @implementation declarations from .m files
func (p *ObjCParser) extractImplementations(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for class implementation declarations
	implQuery := `(class_implementation) @impl.def`

	matches, err := p.tsParser.Query(rootNode, implQuery, "objc")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			implNode := capture.Node
			
			// Find the class name (first identifier after @implementation)
			var implName string
			for i := 0; i < int(implNode.ChildCount()); i++ {
				child := implNode.Child(i)
				if child.Type() == "identifier" {
					implName = child.Content(content)
					break
				}
			}

			if implName != "" {
				// Extract methods from implementation_definition nodes
				var methods []ParsedSymbol
				for i := 0; i < int(implNode.ChildCount()); i++ {
					child := implNode.Child(i)
					if child.Type() == "implementation_definition" {
						// Look for method_definition inside
						for j := 0; j < int(child.ChildCount()); j++ {
							methodNode := child.Child(j)
							if methodNode.Type() == "method_definition" {
								// Extract method name
								var selectorParts []string
								
								for k := 0; k < int(methodNode.ChildCount()); k++ {
									node := methodNode.Child(k)
									if node.Type() == "identifier" {
										selectorParts = append(selectorParts, node.Content(content))
									} else if node.Type() == "method_parameter" {
										// Add colon for parameter
										if len(selectorParts) > 0 {
											selectorParts[len(selectorParts)-1] += ":"
										}
									}
								}
								
								var methodName string
								if len(selectorParts) > 0 {
									methodName = selectorParts[0]
									// For multi-part selectors, join them
									if len(selectorParts) > 1 {
										methodName = ""
										for _, part := range selectorParts {
											methodName += part
										}
									}
								}
								
								if methodName != "" {
									signature := p.extractSignature(methodNode, content)
									docstring := p.extractHeaderDoc(methodNode, content)

									method := ParsedSymbol{
										Name:      methodName,
										Kind:      "method_implementation",
										Signature: signature,
										Span:      nodeToSpan(methodNode),
										Docstring: docstring,
										Node:      methodNode,
									}

									methods = append(methods, method)
								}
							}
						}
					}
				}

				// Extract header documentation
				docstring := p.extractHeaderDoc(implNode, content)

				// Build signature
				signature := p.extractSignature(implNode, content)

				symbol := ParsedSymbol{
					Name:      implName,
					Kind:      "implementation",
					Signature: signature,
					Span:      nodeToSpan(implNode),
					Docstring: docstring,
					Node:      implNode,
					Children:  methods,
				}

				parsedFile.Symbols = append(parsedFile.Symbols, symbol)

				// Create dependency to the interface (header file)
				pairedHeader := p.findPairedFile(parsedFile.Path)
				if pairedHeader != "" {
					dependency := ParsedDependency{
						Type:   "implements_header",
						Source: implName,
						Target: implName, // Same class name in header
					}
					parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
				}
			}
		}
	}

	return nil
}

// extractProtocols extracts @protocol declarations
func (p *ObjCParser) extractProtocols(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for protocol declarations
	protocolQuery := `(protocol_declaration) @protocol.def`

	matches, err := p.tsParser.Query(rootNode, protocolQuery, "objc")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			protocolNode := capture.Node
			
			// Find the protocol name (first identifier after @protocol)
			var protocolName string
			for i := 0; i < int(protocolNode.ChildCount()); i++ {
				child := protocolNode.Child(i)
				if child.Type() == "identifier" {
					protocolName = child.Content(content)
					break
				}
			}

			if protocolName != "" {
				// Extract methods from qualified_protocol_interface_declaration nodes
				var methods []ParsedSymbol
				var properties []ParsedSymbol
				
				for i := 0; i < int(protocolNode.ChildCount()); i++ {
					child := protocolNode.Child(i)
					if child.Type() == "qualified_protocol_interface_declaration" {
						// Extract methods from this section
						sectionMethods := p.extractMethods(child, content)
						methods = append(methods, sectionMethods...)
						
						// Extract properties from this section
						sectionProperties := p.extractProperties(child, content)
						properties = append(properties, sectionProperties...)
					}
				}

				// Extract header documentation
				docstring := p.extractHeaderDoc(protocolNode, content)

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
			}
		}
	}

	return nil
}

// extractProperties extracts @property declarations with attributes
func (p *ObjCParser) extractProperties(parentNode *sitter.Node, content []byte) []ParsedSymbol {
	var properties []ParsedSymbol

	// Iterate through children to find property declarations
	for i := 0; i < int(parentNode.ChildCount()); i++ {
		child := parentNode.Child(i)

		if child.Type() == "property_declaration" {
			propName := ""

			// Look for struct_declaration which contains the property name
			for j := 0; j < int(child.ChildCount()); j++ {
				structDecl := child.Child(j)
				if structDecl.Type() == "struct_declaration" {
					// Find struct_declarator which contains the identifier
					for k := 0; k < int(structDecl.ChildCount()); k++ {
						declarator := structDecl.Child(k)
						if declarator.Type() == "struct_declarator" {
							// Look for identifier or pointer_declarator
							if declarator.ChildCount() > 0 {
								firstChild := declarator.Child(0)
								if firstChild.Type() == "identifier" {
									propName = firstChild.Content(content)
								} else if firstChild.Type() == "pointer_declarator" {
									// For pointer properties like *name
									for l := 0; l < int(firstChild.ChildCount()); l++ {
										idNode := firstChild.Child(l)
										if idNode.Type() == "identifier" {
											propName = idNode.Content(content)
											break
										}
									}
								}
							}
							if propName != "" {
								break
							}
						}
					}
				}
			}

			if propName != "" {
				signature := child.Content(content)
				docstring := p.extractHeaderDoc(child, content)

				property := ParsedSymbol{
					Name:      propName,
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

// extractMethods extracts method declarations and implementations
func (p *ObjCParser) extractMethods(parentNode *sitter.Node, content []byte) []ParsedSymbol {
	var methods []ParsedSymbol

	// Iterate through children to find method declarations/definitions
	for i := 0; i < int(parentNode.ChildCount()); i++ {
		child := parentNode.Child(i)

		if child.Type() == "method_declaration" || child.Type() == "method_definition" {
			methodName := ""

			// Extract method name from identifiers
			// For simple methods like "greet", there's just one identifier
			// For methods with parameters like "initWithName:age:", we need to build the selector
			var selectorParts []string
			
			for j := 0; j < int(child.ChildCount()); j++ {
				node := child.Child(j)
				if node.Type() == "identifier" {
					selectorParts = append(selectorParts, node.Content(content))
				} else if node.Type() == "method_parameter" {
					// Add colon for parameter
					if len(selectorParts) > 0 {
						selectorParts[len(selectorParts)-1] += ":"
					}
				}
			}
			
			if len(selectorParts) > 0 {
				methodName = selectorParts[0]
				// For multi-part selectors, join them
				if len(selectorParts) > 1 {
					methodName = ""
					for _, part := range selectorParts {
						methodName += part
					}
				}
			}

			if methodName != "" {
				signature := p.extractSignature(child, content)
				docstring := p.extractHeaderDoc(child, content)

				kind := "method"
				if child.Type() == "method_definition" {
					kind = "method_implementation"
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

// extractMethodSelector extracts the method selector (name) from a method node
func (p *ObjCParser) extractMethodSelector(methodNode *sitter.Node, content []byte) string {
	// Find method_selector node
	for i := 0; i < int(methodNode.ChildCount()); i++ {
		child := methodNode.Child(i)
		if child.Type() == "method_selector" {
			// Extract the full selector
			return child.Content(content)
		}
	}

	return ""
}

// extractCategories extracts category declarations
func (p *ObjCParser) extractCategories(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Categories are represented as class_interface with parentheses
	// We need to check if a class_interface has a category name in parentheses
	categoryQuery := `(class_interface) @category.def`

	matches, err := p.tsParser.Query(rootNode, categoryQuery, "objc")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			categoryNode := capture.Node
			
			// Check if this is a category (has parentheses with category name)
			var className string
			var categoryName string
			var hasParentheses bool
			
			for i := 0; i < int(categoryNode.ChildCount()); i++ {
				child := categoryNode.Child(i)
				if child.Type() == "identifier" && className == "" {
					className = child.Content(content)
				} else if child.Type() == "(" {
					hasParentheses = true
				} else if child.Type() == "identifier" && hasParentheses && categoryName == "" {
					categoryName = child.Content(content)
				}
			}

			// Only process if this is actually a category (has category name)
			if className != "" && categoryName != "" && hasParentheses {
			// Extract methods
			methods := p.extractMethods(categoryNode, content)

			// Extract properties
			properties := p.extractProperties(categoryNode, content)

			// Extract header documentation
			docstring := p.extractHeaderDoc(categoryNode, content)

			// Build signature
			signature := p.extractSignature(categoryNode, content)

			// Create a unique name for the category
			fullName := fmt.Sprintf("%s(%s)", className, categoryName)

			symbol := ParsedSymbol{
				Name:      fullName,
				Kind:      "category",
				Signature: signature,
				Span:      nodeToSpan(categoryNode),
				Docstring: docstring,
				Node:      categoryNode,
				Children:  append(properties, methods...),
			}

				parsedFile.Symbols = append(parsedFile.Symbols, symbol)

				// Add category-to-class dependency
				dependency := ParsedDependency{
					Type:   "extends",
					Source: fullName,
					Target: className,
				}
				parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
			}
		}
	}

	return nil
}

// extractSuperclass extracts the superclass from a class interface
func (p *ObjCParser) extractSuperclass(interfaceNode *sitter.Node, content []byte) string {
	// Look for the pattern: @interface ClassName : SuperClass
	// The superclass is the identifier after the colon
	foundColon := false
	for i := 0; i < int(interfaceNode.ChildCount()); i++ {
		child := interfaceNode.Child(i)
		if child.Type() == ":" {
			foundColon = true
		} else if foundColon && child.Type() == "identifier" {
			return child.Content(content)
		}
	}

	return ""
}

// extractProtocolConformance extracts protocols that a class conforms to
func (p *ObjCParser) extractProtocolConformance(interfaceNode *sitter.Node, content []byte) []string {
	var protocols []string

	// Find protocol_qualifiers or parameterized_arguments node
	for i := 0; i < int(interfaceNode.ChildCount()); i++ {
		child := interfaceNode.Child(i)
		if child.Type() == "protocol_qualifiers" || child.Type() == "parameterized_arguments" {
			// Extract each protocol identifier
			for j := 0; j < int(child.ChildCount()); j++ {
				protocolNode := child.Child(j)
				if protocolNode.Type() == "identifier" {
					protocols = append(protocols, protocolNode.Content(content))
				} else if protocolNode.Type() == "type_name" {
					// Look for type_identifier inside type_name
					for k := 0; k < int(protocolNode.ChildCount()); k++ {
						typeIdNode := protocolNode.Child(k)
						if typeIdNode.Type() == "type_identifier" {
							protocols = append(protocols, typeIdNode.Content(content))
						}
					}
				}
			}
		}
	}

	return protocols
}

// extractCallRelationships extracts message sends (method calls)
func (p *ObjCParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Query for all message expressions
	callQuery := `(message_expression) @call.expr`

	matches, err := p.tsParser.Query(rootNode, callQuery, "objc")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			messageNode := capture.Node
			
			// Extract the method selector from the message expression
			// For simple messages like [obj method], the second identifier is the method
			// For messages with parameters like [obj method:arg], we need to build the selector
			var selectorParts []string
			
			for i := 0; i < int(messageNode.ChildCount()); i++ {
				child := messageNode.Child(i)
				if child.Type() == "identifier" && i > 0 { // Skip first identifier (receiver)
					selectorParts = append(selectorParts, child.Content(content))
				} else if child.Type() == ":" {
					// Add colon to the last selector part
					if len(selectorParts) > 0 {
						selectorParts[len(selectorParts)-1] += ":"
					}
				}
			}
			
			if len(selectorParts) > 0 {
				// Build the full selector
				callTarget := ""
				for _, part := range selectorParts {
					callTarget += part
				}
				
				// Find the containing method for this call
				caller := p.findContainingMethod(messageNode, parsedFile)
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
	}

	return nil
}

// extractHeaderDoc extracts header documentation comments
func (p *ObjCParser) extractHeaderDoc(node *sitter.Node, content []byte) string {
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

			// Handle documentation comments (/** ... */ or ///)
			if strings.HasPrefix(commentText, "/**") {
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
			} else if strings.HasPrefix(commentText, "///") {
				commentText = strings.TrimPrefix(commentText, "///")
				commentText = strings.TrimSpace(commentText)
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

// isExternalImport determines if an import refers to an external framework
func (p *ObjCParser) isExternalImport(importPath string) bool {
	// Common iOS/macOS frameworks are considered external
	externalFrameworks := []string{
		"Foundation",
		"UIKit",
		"CoreData",
		"CoreGraphics",
		"CoreLocation",
		"MapKit",
		"AVFoundation",
		"WebKit",
		"AppKit",    // macOS
		"Cocoa",     // macOS
		"QuartzCore",
		"CoreAnimation",
		"CoreText",
		"Security",
		"SystemConfiguration",
	}

	for _, framework := range externalFrameworks {
		if strings.HasPrefix(importPath, framework+"/") || importPath == framework+".h" {
			return true
		}
	}

	// System headers (angle brackets) are external
	// Local headers (quotes) are internal
	// This is a heuristic - in practice, we'd check the actual import syntax

	return false
}

// matchInterfaceToImplementation matches @interface with @implementation
// This should be called after parsing both header and implementation files
func (p *ObjCParser) matchInterfaceToImplementation(headerFile, implFile *ParsedFile) {
	// Find @interface in header
	for _, headerSymbol := range headerFile.Symbols {
		if headerSymbol.Kind == "interface" {
			// Find matching @implementation in .m file
			for _, implSymbol := range implFile.Symbols {
				if implSymbol.Kind == "implementation" && implSymbol.Name == headerSymbol.Name {
					// Match methods between interface and implementation
					p.matchObjCMethods(headerSymbol, implSymbol, headerFile, implFile)

					// Create implements_declaration edges for the class itself
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

// matchObjCMethods matches method declarations with implementations
func (p *ObjCParser) matchObjCMethods(headerSymbol, implSymbol ParsedSymbol, headerFile, implFile *ParsedFile) {
	// Match methods from interface (header) to implementation
	for _, headerMethod := range headerSymbol.Children {
		if headerMethod.Kind == "method" {
			// Find matching method in implementation
			for _, implMethod := range implSymbol.Children {
				if implMethod.Kind == "method_implementation" && implMethod.Name == headerMethod.Name {
					// Create implements_declaration edge for this method
					dependency := ParsedDependency{
						Type:   "implements_declaration",
						Source: implMethod.Name,
						Target: headerMethod.Name,
					}
					implFile.Dependencies = append(implFile.Dependencies, dependency)
				}
			}
		}
	}
}

// MatchSignature compares two method signatures to determine if they match
// This is used for matching method declarations with implementations
func (p *ObjCParser) MatchSignature(sig1, sig2 string) bool {
	// Normalize signatures by removing whitespace and comparing
	normalize := func(s string) string {
		s = strings.TrimSpace(s)
		// Remove extra whitespace
		s = strings.Join(strings.Fields(s), " ")
		return s
	}

	return normalize(sig1) == normalize(sig2)
}
