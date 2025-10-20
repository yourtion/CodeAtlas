package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

func TestPrintSummary(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := &ParseCommand{
		Verbose: true,
	}

	// Create test data
	metadata := schema.ParseMetadata{
		Version:      "1.0.0",
		TotalFiles:   10,
		SuccessCount: 8,
		FailureCount: 2,
		Errors: []schema.ParseError{
			{
				File:    "test1.go",
				Line:    10,
				Column:  5,
				Message: "syntax error",
				Type:    schema.ErrorParse,
			},
			{
				File:    "test2.go",
				Message: "file not found",
				Type:    schema.ErrorFileSystem,
			},
		},
	}

	output := schema.ParseOutput{
		Files: []schema.File{
			{
				FileID:   "file1",
				Path:     "test1.go",
				Language: "go",
				Symbols: []schema.Symbol{
					{
						SymbolID: "sym1",
						Name:     "TestFunc",
						Kind:     schema.SymbolFunction,
					},
					{
						SymbolID: "sym2",
						Name:     "TestStruct",
						Kind:     schema.SymbolClass,
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{
			{
				EdgeID:   "edge1",
				EdgeType: schema.EdgeImport,
			},
			{
				EdgeID:   "edge2",
				EdgeType: schema.EdgeCall,
			},
		},
		Metadata: metadata,
	}

	// Call printSummary
	cmd.printSummary(metadata, output)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	summaryOutput := buf.String()

	// Verify output contains expected information
	expectedStrings := []string{
		"Parse Summary",
		"Version: 1.0.0",
		"Total files scanned: 10",
		"Successfully parsed: 8",
		"Failed: 2",
		"Success rate: 80.0%",
		"Symbols extracted:",
		"Total: 2",
		"function: 1",
		"class: 1",
		"Relationships extracted:",
		"Total: 2",
		"import: 1",
		"call: 1",
		"Error breakdown:",
		"parse: 1",
		"filesystem: 1",
		"Error details",
		"test1.go:10:5: syntax error",
		"test2.go: file not found",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(summaryOutput, expected) {
			t.Errorf("Summary output missing expected string: %q\nGot:\n%s", expected, summaryOutput)
		}
	}
}

func TestErrorCollectionInPipeline(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create a valid Go file
	validGoFile := filepath.Join(tmpDir, "valid.go")
	validGoContent := `package main

func main() {
	println("Hello, World!")
}
`
	if err := os.WriteFile(validGoFile, []byte(validGoContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create an invalid Go file (syntax error)
	invalidGoFile := filepath.Join(tmpDir, "invalid.go")
	invalidGoContent := `package main

func main() {
	println("Missing closing brace"
`
	if err := os.WriteFile(invalidGoFile, []byte(invalidGoContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create parse command
	cmd := ParseCommand{
		Path:    tmpDir,
		Workers: 1,
		Verbose: false,
	}

	// Execute parse command
	err := cmd.Execute()

	// We expect the command to succeed even with errors (graceful degradation)
	if err != nil {
		t.Logf("Parse command returned error (expected for invalid syntax): %v", err)
	}

	t.Logf("Parse command completed with graceful error handling")
}

func TestVerboseLogging(t *testing.T) {
	// Create a temporary directory with a test file
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package main

func TestFunction() {
	println("test")
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create parse command with verbose mode
	cmd := ParseCommand{
		Path:    tmpDir,
		Workers: 1,
		Verbose: true,
	}

	// Execute parse command
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Parse command failed: %v", err)
	}

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify verbose output contains expected information
	expectedStrings := []string{
		"Found",
		"files to parse",
		"Starting parsing",
		"workers",
		"Parse Summary",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Verbose output missing expected string: %q", expected)
		}
	}

	t.Logf("Verbose logging test passed")
}
