package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	tree_sitter_objc "github.com/tree-sitter-grammars/tree-sitter-objc/bindings/go"
)

// TreeSitterParser wraps Tree-sitter parsers for multiple languages
type TreeSitterParser struct {
	// Existing parsers
	goParser     *sitter.Parser
	jsParser     *sitter.Parser
	tsParser     *sitter.Parser
	pythonParser *sitter.Parser

	// Mobile language parsers
	kotlinParser *sitter.Parser
	javaParser   *sitter.Parser
	swiftParser  *sitter.Parser
	objcParser   *sitter.Parser
	cParser      *sitter.Parser
	cppParser    *sitter.Parser

	// Existing languages
	goLang *sitter.Language
	jsLang *sitter.Language
	tsLang *sitter.Language
	pyLang *sitter.Language

	// Mobile languages
	kotlinLang *sitter.Language
	javaLang   *sitter.Language
	swiftLang  *sitter.Language
	objcLang   *sitter.Language
	cLang      *sitter.Language
	cppLang    *sitter.Language
}

// NewTreeSitterParser initializes Tree-sitter parsers for all supported languages
func NewTreeSitterParser() (*TreeSitterParser, error) {
	tsp := &TreeSitterParser{}

	// Initialize Go parser
	tsp.goLang = golang.GetLanguage()
	tsp.goParser = sitter.NewParser()
	tsp.goParser.SetLanguage(tsp.goLang)

	// Initialize JavaScript parser
	tsp.jsLang = javascript.GetLanguage()
	tsp.jsParser = sitter.NewParser()
	tsp.jsParser.SetLanguage(tsp.jsLang)

	// Initialize TypeScript parser
	tsp.tsLang = typescript.GetLanguage()
	tsp.tsParser = sitter.NewParser()
	tsp.tsParser.SetLanguage(tsp.tsLang)

	// Initialize Python parser
	tsp.pyLang = python.GetLanguage()
	tsp.pythonParser = sitter.NewParser()
	tsp.pythonParser.SetLanguage(tsp.pyLang)

	// Initialize Kotlin parser (smacker/go-tree-sitter)
	tsp.kotlinLang = kotlin.GetLanguage()
	tsp.kotlinParser = sitter.NewParser()
	tsp.kotlinParser.SetLanguage(tsp.kotlinLang)

	// Initialize Java parser (smacker/go-tree-sitter)
	tsp.javaLang = java.GetLanguage()
	tsp.javaParser = sitter.NewParser()
	tsp.javaParser.SetLanguage(tsp.javaLang)

	// Initialize Swift parser (smacker/go-tree-sitter)
	tsp.swiftLang = swift.GetLanguage()
	tsp.swiftParser = sitter.NewParser()
	tsp.swiftParser.SetLanguage(tsp.swiftLang)

	// Initialize C parser (smacker/go-tree-sitter)
	tsp.cLang = c.GetLanguage()
	tsp.cParser = sitter.NewParser()
	tsp.cParser.SetLanguage(tsp.cLang)

	// Initialize C++ parser (smacker/go-tree-sitter)
	tsp.cppLang = cpp.GetLanguage()
	tsp.cppParser = sitter.NewParser()
	tsp.cppParser.SetLanguage(tsp.cppLang)

	// Initialize Objective-C parser (tree-sitter-grammars)
	tsp.objcLang = sitter.NewLanguage(tree_sitter_objc.Language())
	tsp.objcParser = sitter.NewParser()
	tsp.objcParser.SetLanguage(tsp.objcLang)

	return tsp, nil
}

// Parse parses content with the specified language and returns the root node
func (p *TreeSitterParser) Parse(content []byte, language string) (*sitter.Node, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("empty content provided")
	}

	var parser *sitter.Parser
	switch language {
	case "go":
		parser = p.goParser
	case "javascript", "js", "jsx":
		parser = p.jsParser
	case "typescript", "ts", "tsx":
		parser = p.tsParser
	case "python", "py":
		parser = p.pythonParser
	case "kotlin", "kt":
		parser = p.kotlinParser
	case "java":
		parser = p.javaParser
	case "swift":
		parser = p.swiftParser
	case "objc", "objective-c":
		parser = p.objcParser
	case "c":
		parser = p.cParser
	case "cpp", "c++", "cc", "cxx":
		parser = p.cppParser
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	if parser == nil {
		return nil, fmt.Errorf("parser for language %s is nil", language)
	}

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content: %w", err)
	}

	if tree == nil {
		return nil, fmt.Errorf("parser returned nil tree")
	}

	rootNode := tree.RootNode()
	if rootNode == nil {
		return nil, fmt.Errorf("tree has no root node")
	}

	// Check for parse errors in the tree
	// We return the node even if it has errors (partial results)
	// but indicate the error in the return value
	if rootNode.HasError() {
		return rootNode, fmt.Errorf("parse tree contains errors")
	}

	return rootNode, nil
}

// Query executes a Tree-sitter query on the given node and returns matches
func (p *TreeSitterParser) Query(node *sitter.Node, queryString string, language string) ([]*sitter.QueryMatch, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	if queryString == "" {
		return nil, fmt.Errorf("query string is empty")
	}

	var lang *sitter.Language
	switch language {
	case "go":
		lang = p.goLang
	case "javascript", "js", "jsx":
		lang = p.jsLang
	case "typescript", "ts", "tsx":
		lang = p.tsLang
	case "python", "py":
		lang = p.pyLang
	case "kotlin", "kt":
		lang = p.kotlinLang
	case "java":
		lang = p.javaLang
	case "swift":
		lang = p.swiftLang
	case "objc", "objective-c":
		lang = p.objcLang
	case "c":
		lang = p.cLang
	case "cpp", "c++", "cc", "cxx":
		lang = p.cppLang
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	query, err := sitter.NewQuery([]byte(queryString), lang)
	if err != nil {
		return nil, fmt.Errorf("failed to create query: %w", err)
	}
	defer query.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	cursor.Exec(query, node)

	var matches []*sitter.QueryMatch
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		matches = append(matches, match)
	}

	return matches, nil
}
