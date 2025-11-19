package indexer

import (
	"context"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

func TestHeaderImplAssociator_FindHeaderImplPairs(t *testing.T) {
	tests := []struct {
		name     string
		files    []schema.File
		expected int
	}{
		{
			name: "C header and implementation",
			files: []schema.File{
				{Path: "test.h", Language: "c"},
				{Path: "test.c", Language: "c"},
			},
			expected: 1,
		},
		{
			name: "C++ header and implementation",
			files: []schema.File{
				{Path: "test.hpp", Language: "cpp"},
				{Path: "test.cpp", Language: "cpp"},
			},
			expected: 1,
		},
		{
			name: "Objective-C header and implementation",
			files: []schema.File{
				{Path: "test.h", Language: "objc"},
				{Path: "test.m", Language: "objc"},
			},
			expected: 1,
		},
		{
			name: "No matching implementation",
			files: []schema.File{
				{Path: "test.h", Language: "c"},
			},
			expected: 0,
		},
		{
			name: "Non-header-impl language",
			files: []schema.File{
				{Path: "test.go", Language: "go"},
				{Path: "test.py", Language: "python"},
			},
			expected: 0,
		},
		{
			name: "Multiple pairs",
			files: []schema.File{
				{Path: "foo.h", Language: "c"},
				{Path: "foo.c", Language: "c"},
				{Path: "bar.hpp", Language: "cpp"},
				{Path: "bar.cpp", Language: "cpp"},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			associator := NewHeaderImplAssociator(nil, nil)
			pairs := associator.findHeaderImplPairs(tt.files)
			
			if len(pairs) != tt.expected {
				t.Errorf("expected %d pairs, got %d", tt.expected, len(pairs))
			}
		})
	}
}

func TestHeaderImplAssociator_IsHeaderFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.h", true},
		{"test.hpp", true},
		{"test.hh", true},
		{"test.hxx", true},
		{"test.c", false},
		{"test.cpp", false},
		{"test.m", false},
		{"test.go", false},
	}

	associator := NewHeaderImplAssociator(nil, nil)
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := associator.isHeaderFile(tt.path)
			if result != tt.expected {
				t.Errorf("isHeaderFile(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestHeaderImplAssociator_IsHeaderImplLanguage(t *testing.T) {
	tests := []struct {
		language string
		expected bool
	}{
		{"c", true},
		{"C", true},
		{"cpp", true},
		{"c++", true},
		{"objc", true},
		{"objective-c", true},
		{"go", false},
		{"python", false},
		{"javascript", false},
	}

	associator := NewHeaderImplAssociator(nil, nil)
	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := associator.isHeaderImplLanguage(tt.language)
			if result != tt.expected {
				t.Errorf("isHeaderImplLanguage(%s) = %v, expected %v", tt.language, result, tt.expected)
			}
		})
	}
}

func TestHeaderImplAssociator_MatchSymbols(t *testing.T) {
	tests := []struct {
		name          string
		headerSymbols []schema.Symbol
		implSymbols   []schema.Symbol
		expectedEdges int
	}{
		{
			name: "matching function",
			headerSymbols: []schema.Symbol{
				{
					SymbolID:  "header-func-1",
					Name:      "myFunction",
					Kind:      "function_declaration",
					Signature: "int myFunction(int x)",
				},
			},
			implSymbols: []schema.Symbol{
				{
					SymbolID:  "impl-func-1",
					Name:      "myFunction",
					Kind:      schema.SymbolFunction,
					Signature: "int myFunction(int x)",
				},
			},
			expectedEdges: 1,
		},
		{
			name: "non-matching function",
			headerSymbols: []schema.Symbol{
				{
					SymbolID:  "header-func-1",
					Name:      "myFunction",
					Kind:      "function_declaration",
					Signature: "int myFunction(int x)",
				},
			},
			implSymbols: []schema.Symbol{
				{
					SymbolID:  "impl-func-1",
					Name:      "otherFunction",
					Kind:      schema.SymbolFunction,
					Signature: "int otherFunction(int x)",
				},
			},
			expectedEdges: 0,
		},
		{
			name: "multiple matching functions",
			headerSymbols: []schema.Symbol{
				{
					SymbolID:  "header-func-1",
					Name:      "func1",
					Kind:      "function_declaration",
					Signature: "void func1()",
				},
				{
					SymbolID:  "header-func-2",
					Name:      "func2",
					Kind:      "function_declaration",
					Signature: "int func2(int x)",
				},
			},
			implSymbols: []schema.Symbol{
				{
					SymbolID:  "impl-func-1",
					Name:      "func1",
					Kind:      schema.SymbolFunction,
					Signature: "void func1()",
				},
				{
					SymbolID:  "impl-func-2",
					Name:      "func2",
					Kind:      schema.SymbolFunction,
					Signature: "int func2(int x)",
				},
			},
			expectedEdges: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			associator := NewHeaderImplAssociator(nil, nil)
			
			headerFile := &schema.File{
				Path:    "test.h",
				Symbols: tt.headerSymbols,
			}
			implFile := &schema.File{
				Path:    "test.c",
				Symbols: tt.implSymbols,
			}
			
			edges := associator.matchSymbols(headerFile, implFile)
			
			if len(edges) != tt.expectedEdges {
				t.Errorf("expected %d edges, got %d", tt.expectedEdges, len(edges))
			}
			
			// Verify edge types
			for _, edge := range edges {
				if edge.EdgeType != schema.EdgeImplementsDeclaration {
					t.Errorf("expected edge type %s, got %s", schema.EdgeImplementsDeclaration, edge.EdgeType)
				}
			}
		})
	}
}

func TestHeaderImplAssociator_SignaturesMatch(t *testing.T) {
	tests := []struct {
		name         string
		headerSymbol schema.Symbol
		implSymbol   schema.Symbol
		expected     bool
	}{
		{
			name: "exact match",
			headerSymbol: schema.Symbol{
				Name:      "myFunction",
				Kind:      "function_declaration",
				Signature: "int myFunction(int x)",
			},
			implSymbol: schema.Symbol{
				Name:      "myFunction",
				Kind:      schema.SymbolFunction,
				Signature: "int myFunction(int x)",
			},
			expected: true,
		},
		{
			name: "different names",
			headerSymbol: schema.Symbol{
				Name:      "func1",
				Kind:      "function_declaration",
				Signature: "int func1(int x)",
			},
			implSymbol: schema.Symbol{
				Name:      "func2",
				Kind:      schema.SymbolFunction,
				Signature: "int func2(int x)",
			},
			expected: false,
		},
		{
			name: "whitespace differences",
			headerSymbol: schema.Symbol{
				Name:      "myFunction",
				Kind:      "function_declaration",
				Signature: "int  myFunction( int  x )",
			},
			implSymbol: schema.Symbol{
				Name:      "myFunction",
				Kind:      schema.SymbolFunction,
				Signature: "int myFunction(int x)",
			},
			expected: true,
		},
	}

	associator := NewHeaderImplAssociator(nil, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := associator.signaturesMatch(tt.headerSymbol, tt.implSymbol)
			if result != tt.expected {
				t.Errorf("signaturesMatch() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHeaderImplAssociator_Integration(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test would require a real database connection
	// For now, we'll just test the logic without database operations
	t.Run("end-to-end without database", func(t *testing.T) {
		files := []schema.File{
			{
				FileID:   "file-1",
				Path:     "test.h",
				Language: "c",
				Symbols: []schema.Symbol{
					{
						SymbolID:  "sym-1",
						Name:      "myFunction",
						Kind:      "function_declaration",
						Signature: "int myFunction(int x)",
					},
				},
			},
			{
				FileID:   "file-2",
				Path:     "test.c",
				Language: "c",
				Symbols: []schema.Symbol{
					{
						SymbolID:  "sym-2",
						Name:      "myFunction",
						Kind:      schema.SymbolFunction,
						Signature: "int myFunction(int x)",
					},
				},
			},
		}

		associator := NewHeaderImplAssociator(nil, nil)
		
		// Test pair finding
		pairs := associator.findHeaderImplPairs(files)
		if len(pairs) != 1 {
			t.Fatalf("expected 1 pair, got %d", len(pairs))
		}
		
		// Test symbol matching
		edges, err := associator.matchSymbolsAndCreateEdges(context.Background(), pairs[0], files)
		if err != nil {
			t.Fatalf("matchSymbolsAndCreateEdges failed: %v", err)
		}
		
		// Should have 1 file-level edge + 1 symbol-level edge
		if len(edges) != 2 {
			t.Errorf("expected 2 edges, got %d", len(edges))
		}
		
		// Verify edge types
		hasFileEdge := false
		hasSymbolEdge := false
		for _, edge := range edges {
			if edge.EdgeType == schema.EdgeImplementsHeader {
				hasFileEdge = true
			}
			if edge.EdgeType == schema.EdgeImplementsDeclaration {
				hasSymbolEdge = true
			}
		}
		
		if !hasFileEdge {
			t.Error("missing file-level implements_header edge")
		}
		if !hasSymbolEdge {
			t.Error("missing symbol-level implements_declaration edge")
		}
	})
}
