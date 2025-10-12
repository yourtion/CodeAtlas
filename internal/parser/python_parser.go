package parser

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// PythonParser parses Python source code using Tree-sitter
type PythonParser struct {
	tsParser *TreeSitterParser
}

// NewPythonParser creates a new Python parser
func NewPythonParser(tsParser *TreeSitterParser) *PythonParser {
	return &PythonParser{
		tsParser: tsParser,
	}
}

// Parse parses a Python file and extracts symbols and dependencies
func (p *PythonParser) Parse(file ScannedFile) (*ParsedFile, error) {
	// Read file content
	content, err := readFileContent(file.AbsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse with Tree-sitter
	rootNode, parseErr := p.tsParser.Parse(content, "python")

	parsedFile := &ParsedFile{
		Path:     file.Path,
		Language: "python",
		Content:  content,
		RootNode: rootNode,
	}

	// If we have no root node at all, return error immediately
	if rootNode == nil {
		return parsedFile, fmt.Errorf("parse error: %w", parseErr)
	}

	// Extract module docstring
	if err := p.extractModuleDocstring(rootNode, parsedFile, content); err != nil {
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

	// Extract classes
	if err := p.extractClasses(rootNode, parsedFile, content); err != nil {
		// Non-fatal, continue
	}

	// Return parse error if there was one, but with partial results
	if parseErr != nil {
		return parsedFile, fmt.Errorf("parse error: %w", parseErr)
	}

	return parsedFile, nil
}

// extractModuleDocstring extracts the module-level docstring
func (p *PythonParser) extractModuleDocstring(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Look for the first expression_statement with a string
	for i := 0; i < int(rootNode.ChildCount()); i++ {
		child := rootNode.Child(i)
		if child.Type() == "expression_statement" {
			// Check if it contains a string (docstring)
			for j := 0; j < int(child.ChildCount()); j++ {
				strNode := child.Child(j)
				if strNode.Type() == "string" {
					docstring := p.cleanDocstring(strNode.Content(content))
					
					symbol := ParsedSymbol{
						Name:      "__module__",
						Kind:      "module",
						Docstring: docstring,
						Span:      nodeToSpan(strNode),
						Node:      strNode,
					}
					
					parsedFile.Symbols = append(parsedFile.Symbols, symbol)
					return nil
				}
			}
		}
		// Stop at first non-comment, non-docstring node
		if child.Type() != "comment" && child.Type() != "expression_statement" {
			break
		}
	}

	return nil
}

// extractImports extracts import statements
func (p *PythonParser) extractImports(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Simple import: import module
	importQuery := `(import_statement name: (dotted_name) @import.name)`
	
	matches, err := p.tsParser.Query(rootNode, importQuery, "python")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			importName := capture.Node.Content(content)
			
			dependency := ParsedDependency{
				Type:         "import",
				Target:       importName,
				TargetModule: importName,
			}
			
			parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
		}
	}

	// From import: from module import name
	fromImportQuery := `(import_from_statement module_name: (dotted_name) @import.module)`
	
	matches, err = p.tsParser.Query(rootNode, fromImportQuery, "python")
	if err != nil {
		return err
	}

	for _, match := range matches {
		for _, capture := range match.Captures {
			moduleName := capture.Node.Content(content)
			
			dependency := ParsedDependency{
				Type:         "import",
				Target:       moduleName,
				TargetModule: moduleName,
			}
			
			parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
		}
	}

	return nil
}

// extractFunctions extracts function definitions
func (p *PythonParser) extractFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	// Regular function definitions
	funcQuery := `(function_definition name: (identifier) @func.name) @func.def`
	
	matches, err := p.tsParser.Query(rootNode, funcQuery, "python")
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

			signature := p.extractFunctionSignature(funcNode, content)
			docstring := p.extractPythonDocstring(funcNode, content)
			decorators := p.extractDecorators(funcNode, content)
			isAsync := p.isAsyncFunction(funcNode)
			
			kind := "function"
			if isAsync {
				kind = "async_function"
			}
			
			// Add decorator info to signature if present
			fullSignature := signature
			if len(decorators) > 0 {
				fullSignature = strings.Join(decorators, "\n") + "\n" + signature
			}
			
			symbol := ParsedSymbol{
				Name:      funcName,
				Kind:      kind,
				Signature: fullSignature,
				Span:      nodeToSpan(funcNode),
				Docstring: docstring,
				Node:      funcNode,
			}
			
			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
		}
	}

	return nil
}

// extractClasses extracts class definitions
func (p *PythonParser) extractClasses(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
	classQuery := `(class_definition name: (identifier) @class.name) @class.def`
	
	matches, err := p.tsParser.Query(rootNode, classQuery, "python")
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
			// Skip nested classes
			if p.isInsideClass(classNode) {
				continue
			}

			methods := p.extractClassMethods(classNode, content)
			baseClasses := p.extractBaseClasses(classNode, content)
			decorators := p.extractDecorators(classNode, content)
			docstring := p.extractPythonDocstring(classNode, content)
			
			signature := fmt.Sprintf("class %s", className)
			if len(baseClasses) > 0 {
				signature = fmt.Sprintf("class %s(%s)", className, strings.Join(baseClasses, ", "))
			}
			
			// Add decorator info to signature if present
			if len(decorators) > 0 {
				signature = strings.Join(decorators, "\n") + "\n" + signature
			}
			
			symbol := ParsedSymbol{
				Name:      className,
				Kind:      "class",
				Signature: signature,
				Span:      nodeToSpan(classNode),
				Docstring: docstring,
				Node:      classNode,
				Children:  methods,
			}
			
			parsedFile.Symbols = append(parsedFile.Symbols, symbol)
			
			// Add inheritance dependencies
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

	return nil
}

// extractClassMethods extracts methods from a class
func (p *PythonParser) extractClassMethods(classNode *sitter.Node, content []byte) []ParsedSymbol {
	var methods []ParsedSymbol
	
	// Find the class body (block)
	classBody := findChildByType(classNode, "block")
	if classBody == nil {
		return methods
	}
	
	// Look for function definitions in the class body
	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)
		
		if child.Type() == "function_definition" {
			methodName := ""
			
			// Find the method name
			for j := 0; j < int(child.ChildCount()); j++ {
				nameNode := child.Child(j)
				if nameNode.Type() == "identifier" {
					methodName = nameNode.Content(content)
					break
				}
			}
			
			if methodName != "" {
				signature := p.extractFunctionSignature(child, content)
				docstring := p.extractPythonDocstring(child, content)
				decorators := p.extractDecorators(child, content)
				isAsync := p.isAsyncFunction(child)
				isStatic := p.hasDecorator(decorators, "staticmethod")
				isClassMethod := p.hasDecorator(decorators, "classmethod")
				
				kind := "method"
				if isAsync {
					kind = "async_method"
				}
				if isStatic {
					kind = "static_method"
				}
				if isClassMethod {
					kind = "class_method"
				}
				
				// Add decorator info to signature if present
				fullSignature := signature
				if len(decorators) > 0 {
					fullSignature = strings.Join(decorators, "\n") + "\n" + signature
				}
				
				method := ParsedSymbol{
					Name:      methodName,
					Kind:      kind,
					Signature: fullSignature,
					Span:      nodeToSpan(child),
					Docstring: docstring,
					Node:      child,
				}
				
				methods = append(methods, method)
			}
		} else if child.Type() == "decorated_definition" {
			// Handle decorated methods
			funcDef := findChildByType(child, "function_definition")
			if funcDef != nil {
				methodName := ""
				
				// Find the method name
				for j := 0; j < int(funcDef.ChildCount()); j++ {
					nameNode := funcDef.Child(j)
					if nameNode.Type() == "identifier" {
						methodName = nameNode.Content(content)
						break
					}
				}
				
				if methodName != "" {
					signature := p.extractFunctionSignature(funcDef, content)
					docstring := p.extractPythonDocstring(funcDef, content)
					decorators := p.extractDecorators(child, content)
					isAsync := p.isAsyncFunction(funcDef)
					isStatic := p.hasDecorator(decorators, "staticmethod")
					isClassMethod := p.hasDecorator(decorators, "classmethod")
					
					kind := "method"
					if isAsync {
						kind = "async_method"
					}
					if isStatic {
						kind = "static_method"
					}
					if isClassMethod {
						kind = "class_method"
					}
					
					// Add decorator info to signature if present
					fullSignature := signature
					if len(decorators) > 0 {
						fullSignature = strings.Join(decorators, "\n") + "\n" + signature
					}
					
					method := ParsedSymbol{
						Name:      methodName,
						Kind:      kind,
						Signature: fullSignature,
						Span:      nodeToSpan(child),
						Docstring: docstring,
						Node:      child,
					}
					
					methods = append(methods, method)
				}
			}
		}
	}
	
	return methods
}

// extractBaseClasses extracts the base classes from a class definition
func (p *PythonParser) extractBaseClasses(classNode *sitter.Node, content []byte) []string {
	var baseClasses []string
	
	// Find argument_list (base classes)
	argList := findChildByType(classNode, "argument_list")
	if argList == nil {
		return baseClasses
	}
	
	// Extract each base class
	for i := 0; i < int(argList.ChildCount()); i++ {
		child := argList.Child(i)
		if child.Type() == "identifier" || child.Type() == "attribute" {
			baseClasses = append(baseClasses, child.Content(content))
		}
	}
	
	return baseClasses
}

// extractDecorators extracts decorators from a function or class
func (p *PythonParser) extractDecorators(node *sitter.Node, content []byte) []string {
	var decorators []string
	
	// Check if parent is decorated_definition
	parent := node.Parent()
	if parent != nil && parent.Type() == "decorated_definition" {
		// Look for decorator nodes
		for i := 0; i < int(parent.ChildCount()); i++ {
			child := parent.Child(i)
			if child.Type() == "decorator" {
				decorator := strings.TrimSpace(child.Content(content))
				decorators = append(decorators, decorator)
			}
		}
	}
	
	// Also check if the node itself is decorated_definition
	if node.Type() == "decorated_definition" {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "decorator" {
				decorator := strings.TrimSpace(child.Content(content))
				decorators = append(decorators, decorator)
			}
		}
	}
	
	return decorators
}

// extractFunctionSignature extracts the function signature with type hints
func (p *PythonParser) extractFunctionSignature(funcNode *sitter.Node, content []byte) string {
	// Get the first line of the function (def line)
	funcText := funcNode.Content(content)
	lines := strings.Split(funcText, "\n")
	
	// Find the def line (might span multiple lines)
	signature := ""
	for i, line := range lines {
		signature += line
		// Check if we've reached the colon
		if strings.Contains(line, ":") {
			break
		}
		// If not the last line and no colon yet, add space for continuation
		if i < len(lines)-1 {
			signature += " "
		}
	}
	
	return strings.TrimSpace(signature)
}

// extractPythonDocstring extracts the docstring from a function or class
func (p *PythonParser) extractPythonDocstring(node *sitter.Node, content []byte) string {
	// Find the block (body) of the function/class
	block := findChildByType(node, "block")
	if block == nil {
		return ""
	}
	
	// Look for the first expression_statement with a string
	for i := 0; i < int(block.ChildCount()); i++ {
		child := block.Child(i)
		if child.Type() == "expression_statement" {
			// Check if it contains a string (docstring)
			for j := 0; j < int(child.ChildCount()); j++ {
				strNode := child.Child(j)
				if strNode.Type() == "string" {
					return p.cleanDocstring(strNode.Content(content))
				}
			}
		}
		// Stop at first non-docstring statement
		if child.Type() != "expression_statement" {
			break
		}
	}
	
	return ""
}

// cleanDocstring removes quotes and cleans up docstring formatting
func (p *PythonParser) cleanDocstring(docstring string) string {
	// Remove triple quotes
	docstring = strings.TrimPrefix(docstring, `"""`)
	docstring = strings.TrimSuffix(docstring, `"""`)
	docstring = strings.TrimPrefix(docstring, `'''`)
	docstring = strings.TrimSuffix(docstring, `'''`)
	
	// Remove single quotes
	docstring = strings.TrimPrefix(docstring, `"`)
	docstring = strings.TrimSuffix(docstring, `"`)
	docstring = strings.TrimPrefix(docstring, `'`)
	docstring = strings.TrimSuffix(docstring, `'`)
	
	// Clean up whitespace
	docstring = strings.TrimSpace(docstring)
	
	return docstring
}

// isAsyncFunction checks if a function is async
func (p *PythonParser) isAsyncFunction(funcNode *sitter.Node) bool {
	// Check if the function has 'async' keyword
	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child.Type() == "async" {
			return true
		}
	}
	return false
}

// isInsideClass checks if a node is inside a class definition
func (p *PythonParser) isInsideClass(node *sitter.Node) bool {
	current := node.Parent()
	for current != nil {
		if current.Type() == "class_definition" {
			return true
		}
		current = current.Parent()
	}
	return false
}

// hasDecorator checks if a decorator list contains a specific decorator
func (p *PythonParser) hasDecorator(decorators []string, name string) bool {
	for _, dec := range decorators {
		if strings.Contains(dec, "@"+name) {
			return true
		}
	}
	return false
}
