package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/parser"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// TestErrorHandlingIntegration tests the complete error handling pipeline
func TestErrorHandlingIntegration(t *testing.T) {
	// Create a temporary directory with various test files
	tmpDir := t.TempDir()

	// Create a valid Go file
	validGoFile := filepath.Join(tmpDir, "valid.go")
	validGoContent := `package main

import "fmt"

// HelloWorld prints a greeting
func HelloWorld() {
	fmt.Println("Hello, World!")
}
`
	if err := os.WriteFile(validGoFile, []byte(validGoContent), 0644); err != nil {
		t.Fatalf("Failed to create valid Go file: %v", err)
	}

	// Create an invalid Go file (syntax error)
	invalidGoFile := filepath.Join(tmpDir, "invalid.go")
	invalidGoContent := `package main

func BrokenFunction() {
	// Missing closing brace
	println("This will cause a parse error"
`
	if err := os.WriteFile(invalidGoFile, []byte(invalidGoContent), 0644); err != nil {
		t.Fatalf("Failed to create invalid Go file: %v", err)
	}

	// Create a valid Python file
	validPyFile := filepath.Join(tmpDir, "valid.py")
	validPyContent := `def hello_world():
    """Print a greeting"""
    print("Hello, World!")
`
	if err := os.WriteFile(validPyFile, []byte(validPyContent), 0644); err != nil {
		t.Fatalf("Failed to create valid Python file: %v", err)
	}

	// Create an invalid Python file (syntax error)
	invalidPyFile := filepath.Join(tmpDir, "invalid.py")
	invalidPyContent := `def broken_function():
    print("Missing closing parenthesis"
`
	if err := os.WriteFile(invalidPyFile, []byte(invalidPyContent), 0644); err != nil {
		t.Fatalf("Failed to create invalid Python file: %v", err)
	}

	// Initialize Tree-sitter parser
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	// Create parser pool
	pool := parser.NewParserPool(2, tsParser)

	// Scan files
	scanner := parser.NewFileScanner(tmpDir, nil)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	t.Logf("Found %d files to parse", len(files))

	// Process files
	parsedFiles, parseErrors := pool.Process(files)

	t.Logf("Parsed %d files with %d errors", len(parsedFiles), len(parseErrors))

	// Verify we got results for all files (even with errors)
	if len(parsedFiles) < 2 {
		t.Errorf("Expected at least 2 parsed files (with partial results), got %d", len(parsedFiles))
	}

	// Verify we got errors for the invalid files
	if len(parseErrors) < 2 {
		t.Errorf("Expected at least 2 parse errors, got %d", len(parseErrors))
	}

	// Verify errors are DetailedParseError
	for i, err := range parseErrors {
		if detailedErr, ok := err.(*parser.DetailedParseError); ok {
			t.Logf("Error %d: %s (type: %s)", i, detailedErr.Error(), detailedErr.Type)
			if detailedErr.Type != "parse" {
				t.Errorf("Expected error type 'parse', got %q", detailedErr.Type)
			}
		} else {
			t.Errorf("Error %d: expected DetailedParseError, got %T", i, err)
		}
	}

	// Map to schema
	mapper := schema.NewSchemaMapper()
	var schemaFiles []schema.File
	var allEdges []schema.DependencyEdge
	var mappingErrors []schema.ParseError

	for _, parsedFile := range parsedFiles {
		schemaFile, edges, err := mapper.MapToSchema(parsedFile)
		if err != nil {
			mappingErrors = append(mappingErrors, schema.ParseError{
				File:    parsedFile.Path,
				Message: err.Error(),
				Type:    schema.ErrorMapping,
			})
			continue
		}

		schemaFiles = append(schemaFiles, *schemaFile)
		allEdges = append(allEdges, edges...)
	}

	t.Logf("Mapped %d files to schema with %d mapping errors", len(schemaFiles), len(mappingErrors))

	// Collect all errors
	var allErrors []schema.ParseError
	for _, err := range parseErrors {
		if detailedErr, ok := err.(*parser.DetailedParseError); ok {
			allErrors = append(allErrors, schema.ParseError{
				File:    detailedErr.File,
				Line:    detailedErr.Line,
				Column:  detailedErr.Column,
				Message: detailedErr.Message,
				Type:    schema.ErrorType(detailedErr.Type),
			})
		} else {
			allErrors = append(allErrors, schema.ParseError{
				Message: err.Error(),
				Type:    schema.ErrorParse,
			})
		}
	}
	allErrors = append(allErrors, mappingErrors...)

	// Create output
	output := schema.ParseOutput{
		Files:         schemaFiles,
		Relationships: allEdges,
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			TotalFiles:   len(files),
			SuccessCount: len(schemaFiles),
			FailureCount: len(allErrors),
			Errors:       allErrors,
		},
	}

	// Verify output structure
	if output.Metadata.TotalFiles != 4 {
		t.Errorf("Expected 4 total files, got %d", output.Metadata.TotalFiles)
	}

	if output.Metadata.SuccessCount < 2 {
		t.Errorf("Expected at least 2 successful parses, got %d", output.Metadata.SuccessCount)
	}

	if output.Metadata.FailureCount < 2 {
		t.Errorf("Expected at least 2 failures, got %d", output.Metadata.FailureCount)
	}

	// Verify we can serialize to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize output to JSON: %v", err)
	}

	t.Logf("Generated JSON output: %d bytes", len(jsonData))

	// Verify JSON contains error information
	var parsedOutput schema.ParseOutput
	if err := json.Unmarshal(jsonData, &parsedOutput); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(parsedOutput.Metadata.Errors) == 0 {
		t.Error("Expected errors in JSON output, got none")
	}

	// Verify error types are preserved
	errorTypes := make(map[schema.ErrorType]int)
	for _, err := range parsedOutput.Metadata.Errors {
		errorTypes[err.Type]++
	}

	t.Logf("Error types: %v", errorTypes)

	if errorTypes[schema.ErrorParse] == 0 {
		t.Error("Expected parse errors in output")
	}
}

// TestGracefulDegradation verifies that parsing continues even with errors
func TestGracefulDegradation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files with various issues
	files := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name: "good1.go",
			content: `package main
func Good1() {}`,
			valid: true,
		},
		{
			name: "bad1.go",
			content: `package main
func Bad1() {`,
			valid: false,
		},
		{
			name: "good2.go",
			content: `package main
func Good2() {}`,
			valid: true,
		},
		{
			name: "bad2.go",
			content: `package main
func Bad2() {
	// Missing brace`,
			valid: false,
		},
		{
			name: "good3.go",
			content: `package main
func Good3() {}`,
			valid: true,
		},
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(path, []byte(f.content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f.name, err)
		}
	}

	// Parse all files
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	pool := parser.NewParserPool(2, tsParser)
	scanner := parser.NewFileScanner(tmpDir, nil)
	scannedFiles, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	parsedFiles, parseErrors := pool.Process(scannedFiles)

	// Verify we got results for all files (graceful degradation)
	t.Logf("Parsed %d files with %d errors", len(parsedFiles), len(parseErrors))

	// We should have parsed all files (even with errors)
	if len(parsedFiles) < 3 {
		t.Errorf("Expected at least 3 parsed files (valid ones), got %d", len(parsedFiles))
	}

	// We should have errors for the invalid files
	if len(parseErrors) < 2 {
		t.Errorf("Expected at least 2 errors, got %d", len(parseErrors))
	}

	// Verify that valid files were parsed successfully
	validCount := 0
	for _, pf := range parsedFiles {
		if len(pf.Symbols) > 0 {
			validCount++
		}
	}

	t.Logf("Successfully parsed %d files with symbols", validCount)

	if validCount < 3 {
		t.Errorf("Expected at least 3 files with symbols, got %d", validCount)
	}
}
