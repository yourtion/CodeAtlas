package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/parser"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// TestParseMobileLanguages tests parsing of mobile language files
func TestParseMobileLanguages(t *testing.T) {
	// Create temporary directory with mobile language files
	tempDir := t.TempDir()

	testFiles := map[string]string{
		"Example.kt": `package com.example

class Example {
    fun greet(name: String) {
        println("Hello, $name")
    }
}
`,
		"Example.java": `package com.example;

public class Example {
    public void greet(String name) {
        System.out.println("Hello, " + name);
    }
}
`,
		"Example.swift": `import Foundation

class Example {
    func greet(name: String) {
        print("Hello, \(name)")
    }
}
`,
		"Example.m": `#import <Foundation/Foundation.h>

@implementation Example

- (void)greet:(NSString *)name {
    NSLog(@"Hello, %@", name);
}

@end
`,
		"example.c": `#include <stdio.h>

void greet(const char *name) {
    printf("Hello, %s\n", name);
}
`,
		"example.cpp": `#include <iostream>
#include <string>

void greet(const std::string& name) {
    std::cout << "Hello, " << name << std::endl;
}
`,
	}

	// Write test files
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Initialize parser
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	// Scan directory
	scanner := parser.NewFileScanner(tempDir, nil)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	if len(files) != 6 {
		t.Fatalf("Expected 6 files, got %d", len(files))
	}

	// Parse files
	pool := parser.NewParserPool(4, tsParser)
	parsedFiles, errors := pool.Process(files)

	if len(errors) != 0 {
		t.Fatalf("Expected no errors, got %d: %v", len(errors), errors)
	}

	if len(parsedFiles) != 6 {
		t.Fatalf("Expected 6 parsed files, got %d", len(parsedFiles))
	}

	// Map to schema
	mapper := schema.NewSchemaMapper()
	var schemaFiles []schema.File
	var allEdges []schema.DependencyEdge

	for _, parsedFile := range parsedFiles {
		schemaFile, edges, err := mapper.MapToSchema(parsedFile)
		if err != nil {
			t.Fatalf("Failed to map file %s: %v", parsedFile.Path, err)
		}
		schemaFiles = append(schemaFiles, *schemaFile)
		allEdges = append(allEdges, edges...)
	}

	// Verify each language was parsed
	languageCount := make(map[string]int)
	for _, file := range schemaFiles {
		languageCount[file.Language]++
	}

	expectedLanguages := []string{"kotlin", "java", "swift", "objc", "c", "cpp"}
	for _, lang := range expectedLanguages {
		if languageCount[lang] != 1 {
			t.Errorf("Expected 1 %s file, got %d", lang, languageCount[lang])
		}
	}

	// Verify symbols were extracted from each file
	for _, file := range schemaFiles {
		if len(file.Symbols) == 0 {
			t.Errorf("Expected symbols to be extracted from %s (%s)", file.Path, file.Language)
		}
	}

	// Verify output can be serialized to JSON
	output := schema.ParseOutput{
		Files:         schemaFiles,
		Relationships: allEdges,
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			TotalFiles:   len(schemaFiles),
			SuccessCount: len(schemaFiles),
			FailureCount: 0,
		},
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal output to JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Expected non-empty JSON output")
	}

	// Verify we can unmarshal it back
	var decoded schema.ParseOutput
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(decoded.Files) != 6 {
		t.Errorf("Expected 6 files after unmarshal, got %d", len(decoded.Files))
	}
}

// TestParseMobileLanguagesWithFilter tests language filtering
func TestParseMobileLanguagesWithFilter(t *testing.T) {
	// Create temporary directory with mobile language files
	tempDir := t.TempDir()

	testFiles := map[string]string{
		"Example.kt":   "package com.example\n\nfun hello() {}\n",
		"Example.java": "package com.example;\n\npublic class Example {}\n",
		"Example.swift": "import Foundation\n\nfunc hello() {}\n",
	}

	// Write test files
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Test filtering for each language
	tests := []struct {
		name           string
		languageFilter string
		expectedCount  int
	}{
		{
			name:           "filter Kotlin",
			languageFilter: "Kotlin",
			expectedCount:  1,
		},
		{
			name:           "filter Java",
			languageFilter: "Java",
			expectedCount:  1,
		},
		{
			name:           "filter Swift",
			languageFilter: "Swift",
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Scan with language filter
			scanner := parser.NewFileScanner(tempDir, nil)
			scanner.SetLanguageFilter([]string{tt.languageFilter})
			files, err := scanner.Scan()
			if err != nil {
				t.Fatalf("Failed to scan directory: %v", err)
			}

			if len(files) != tt.expectedCount {
				t.Errorf("Expected %d file(s), got %d", tt.expectedCount, len(files))
			}

			// Verify the filtered file has the correct language
			if len(files) > 0 && files[0].Language != tt.languageFilter {
				t.Errorf("Expected language %s, got %s", tt.languageFilter, files[0].Language)
			}
		})
	}
}
