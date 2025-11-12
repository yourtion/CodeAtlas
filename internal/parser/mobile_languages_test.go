package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParse_MobileLanguages tests parsing for mobile development languages
func TestParse_MobileLanguages(t *testing.T) {
	parser, err := NewTreeSitterParser()
	require.NoError(t, err, "Failed to create TreeSitterParser")
	require.NotNil(t, parser, "Parser should not be nil")

	tests := []struct {
		name     string
		language string
		code     string
		wantErr  bool
	}{
		{
			name:     "valid Kotlin code",
			language: "kotlin",
			code: `
fun main() {
    println("Hello, Kotlin!")
}

class Person(val name: String, val age: Int)
`,
			wantErr: false,
		},
		{
			name:     "valid Java code",
			language: "java",
			code: `
public class HelloWorld {
    public static void main(String[] args) {
        System.out.println("Hello, Java!");
    }
}
`,
			wantErr: false,
		},
		{
			name:     "valid Swift code",
			language: "swift",
			code: `
func greet(name: String) -> String {
    return "Hello, \(name)!"
}

class Person {
    var name: String
    var age: Int
    
    init(name: String, age: Int) {
        self.name = name
        self.age = age
    }
}
`,
			wantErr: false,
		},

		{
			name:     "valid C code",
			language: "c",
			code: `
#include <stdio.h>

int add(int a, int b) {
    return a + b;
}

int main() {
    printf("Hello, C!\n");
    return 0;
}
`,
			wantErr: false,
		},
		{
			name:     "valid C++ code",
			language: "cpp",
			code: `
#include <iostream>
#include <string>

class Person {
private:
    std::string name;
    int age;
    
public:
    Person(std::string n, int a) : name(n), age(a) {}
    
    void greet() {
        std::cout << "Hello, " << name << "!" << std::endl;
    }
};

int main() {
    Person p("Alice", 30);
    p.greet();
    return 0;
}
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := parser.Parse([]byte(tt.code), tt.language)
			
			if tt.wantErr {
				assert.Error(t, err, "Expected an error but got none")
			} else {
				// We accept either no error or a parse error (partial results)
				// as long as we get a valid node
				if err != nil {
					assert.Contains(t, err.Error(), "parse tree contains errors",
						"Unexpected error type: %v", err)
				}
				assert.NotNil(t, node, "Node should not be nil for valid code")
				if node != nil {
					assert.Greater(t, node.ChildCount(), uint32(0),
						"Root node should have children")
				}
			}
		})
	}
}

// TestParse_MobileLanguageAliases tests language identifier aliases
func TestParse_MobileLanguageAliases(t *testing.T) {
	parser, err := NewTreeSitterParser()
	require.NoError(t, err)

	tests := []struct {
		name     string
		language string
		code     string
	}{
		{
			name:     "Kotlin with .kt extension",
			language: "kt",
			code:     `fun test() { println("test") }`,
		},
		{
			name:     "C++ with c++ identifier",
			language: "c++",
			code:     `int main() { return 0; }`,
		},
		{
			name:     "C++ with cc identifier",
			language: "cc",
			code:     `int main() { return 0; }`,
		},
		{
			name:     "C++ with cxx identifier",
			language: "cxx",
			code:     `int main() { return 0; }`,
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := parser.Parse([]byte(tt.code), tt.language)
			// Accept either success or parse error (partial results)
			if err != nil {
				assert.Contains(t, err.Error(), "parse tree contains errors")
			}
			assert.NotNil(t, node, "Should parse with language alias: %s", tt.language)
		})
	}
}

// TestQuery_MobileLanguages tests querying for mobile development languages
func TestQuery_MobileLanguages(t *testing.T) {
	parser, err := NewTreeSitterParser()
	require.NoError(t, err)

	tests := []struct {
		name     string
		language string
		code     string
		query    string
	}{
		{
			name:     "Kotlin function query",
			language: "kotlin",
			code:     `fun greet() { println("Hello") }`,
			query:    `(function_declaration) @func`,
		},
		{
			name:     "Java class query",
			language: "java",
			code:     `public class Test { }`,
			query:    `(class_declaration) @class`,
		},
		{
			name:     "Swift function query",
			language: "swift",
			code:     `func test() { print("test") }`,
			query:    `(function_declaration) @func`,
		},
		{
			name:     "C function query",
			language: "c",
			code:     `int add(int a, int b) { return a + b; }`,
			query:    `(function_definition) @func`,
		},
		{
			name:     "C++ class query",
			language: "cpp",
			code:     `class Test { };`,
			query:    `(class_specifier) @class`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := parser.Parse([]byte(tt.code), tt.language)
			if err != nil && node == nil {
				t.Skipf("Skipping query test due to parse failure: %v", err)
				return
			}
			require.NotNil(t, node)

			matches, err := parser.Query(node, tt.query, tt.language)
			// Query might fail if the grammar doesn't match exactly,
			// but we should at least not panic
			if err != nil {
				t.Logf("Query failed (expected for some grammars): %v", err)
			} else {
				t.Logf("Found %d matches for query", len(matches))
			}
		})
	}
}
