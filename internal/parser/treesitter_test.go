package parser

import (
	"testing"
)

func TestNewTreeSitterParser(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	if parser == nil {
		t.Fatal("Parser is nil")
	}

	if parser.goParser == nil {
		t.Error("Go parser not initialized")
	}

	if parser.jsParser == nil {
		t.Error("JavaScript parser not initialized")
	}

	if parser.tsParser == nil {
		t.Error("TypeScript parser not initialized")
	}

	if parser.pythonParser == nil {
		t.Error("Python parser not initialized")
	}
}

func TestParse_Go(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "valid Go code",
			content: `package main

func main() {
	println("Hello, World!")
}`,
			expectError: false,
		},
		{
			name: "Go with struct and main",
			content: `package main

type Person struct {
	Name string
	Age  int
}

func main() {}`,
			expectError: false,
		},
		{
			name:        "invalid Go syntax",
			content:     `package main\nfunc {`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := parser.Parse([]byte(tt.content), "go")

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if node == nil {
					t.Error("Node is nil")
				}
			}
		})
	}
}

func TestParse_JavaScript(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name        string
		content     string
		language    string
		expectError bool
	}{
		{
			name: "valid JavaScript",
			content: `function greet(name) {
	console.log("Hello, " + name);
}`,
			language:    "javascript",
			expectError: false,
		},
		{
			name:        "arrow function",
			content:     `const add = (a, b) => a + b;`,
			language:    "js",
			expectError: false,
		},
		{
			name: "class definition",
			content: `class Person {
	constructor(name) {
		this.name = name;
	}
}`,
			language:    "jsx",
			expectError: false,
		},
		{
			name:        "invalid JavaScript syntax",
			content:     `function {`,
			language:    "javascript",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := parser.Parse([]byte(tt.content), tt.language)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if node == nil {
					t.Error("Node is nil")
				}
			}
		})
	}
}

func TestParse_TypeScript(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name        string
		content     string
		language    string
		expectError bool
	}{
		{
			name: "valid TypeScript",
			content: `function greet(name: string): void {
	console.log("Hello, " + name);
}`,
			language:    "typescript",
			expectError: false,
		},
		{
			name: "interface definition",
			content: `interface Person {
	name: string;
	age: number;
}`,
			language:    "ts",
			expectError: false,
		},
		{
			name: "generic function",
			content: `function identity<T>(arg: T): T {
	return arg;
}`,
			language:    "tsx",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := parser.Parse([]byte(tt.content), tt.language)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if node == nil {
					t.Error("Node is nil")
				}
			}
		})
	}
}

func TestParse_Python(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name        string
		content     string
		language    string
		expectError bool
	}{
		{
			name: "valid Python",
			content: `def greet(name):
    print(f"Hello, {name}")`,
			language:    "python",
			expectError: false,
		},
		{
			name: "class definition",
			content: `class Person:
    def __init__(self, name):
        self.name = name`,
			language:    "py",
			expectError: false,
		},
		{
			name: "async function",
			content: `async def fetch_data():
    return await get_data()`,
			language:    "python",
			expectError: false,
		},
		{
			name:        "invalid Python syntax",
			content:     `def ():`,
			language:    "python",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := parser.Parse([]byte(tt.content), tt.language)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if node == nil {
					t.Error("Node is nil")
				}
			}
		})
	}
}

func TestParse_UnsupportedLanguage(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	_, err = parser.Parse([]byte("some content"), "rust")
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

func TestParse_EmptyContent(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	_, err = parser.Parse([]byte(""), "go")
	if err == nil {
		t.Error("Expected error for empty content")
	}
}

func TestQuery_Go(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	content := `package main

func add(a, b int) int {
	return a + b
}

func multiply(x, y int) int {
	return x * y
}`

	node, err := parser.Parse([]byte(content), "go")
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	// Query for function declarations
	queryString := `(function_declaration name: (identifier) @func.name)`
	matches, err := parser.Query(node, queryString, "go")
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 function matches, got %d", len(matches))
	}
}

func TestQuery_JavaScript(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	content := `function greet(name) {
	console.log("Hello, " + name);
}

const add = (a, b) => a + b;`

	node, err := parser.Parse([]byte(content), "javascript")
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	// Query for function declarations
	queryString := `(function_declaration name: (identifier) @func.name)`
	matches, err := parser.Query(node, queryString, "javascript")
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("Expected 1 function match, got %d", len(matches))
	}
}

func TestQuery_Python(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	content := `def greet(name):
    print(f"Hello, {name}")

def add(a, b):
    return a + b`

	node, err := parser.Parse([]byte(content), "python")
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	// Query for function definitions
	queryString := `(function_definition name: (identifier) @func.name)`
	matches, err := parser.Query(node, queryString, "python")
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 function matches, got %d", len(matches))
	}
}

func TestQuery_InvalidQuery(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	content := `package main

func main() {}`

	node, err := parser.Parse([]byte(content), "go")
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	// Invalid query syntax
	_, err = parser.Query(node, "(invalid query", "go")
	if err == nil {
		t.Error("Expected error for invalid query")
	}
}

func TestQuery_NilNode(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	_, err = parser.Query(nil, "(function_declaration)", "go")
	if err == nil {
		t.Error("Expected error for nil node")
	}
}

func TestQuery_EmptyQuery(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	content := `package main`
	node, err := parser.Parse([]byte(content), "go")
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	_, err = parser.Query(node, "", "go")
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestQuery_UnsupportedLanguage(t *testing.T) {
	parser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	content := `package main`
	node, err := parser.Parse([]byte(content), "go")
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	_, err = parser.Query(node, "(function_declaration)", "rust")
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}
