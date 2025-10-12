package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// TreeSitterParser wraps Tree-sitter parsers for multiple languages
type TreeSitterParser struct {
	goParser     *sitter.Parser
	jsParser     *sitter.Parser
	tsParser     *sitter.Parser
	pythonParser *sitter.Parser
	goLang       *sitter.Language
	jsLang       *sitter.Language
	tsLang       *sitter.Language
	pyLang       *sitter.Language
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
