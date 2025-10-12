package parser

import (
	"strings"
	"testing"
)

func TestDetailedParseError(t *testing.T) {
	tests := []struct {
		name     string
		err      *DetailedParseError
		expected string
	}{
		{
			name: "error with line and column",
			err: &DetailedParseError{
				File:    "test.go",
				Line:    10,
				Column:  5,
				Message: "unexpected token",
				Type:    "parse",
			},
			expected: "test.go:10:5: unexpected token",
		},
		{
			name: "error without line and column",
			err: &DetailedParseError{
				File:    "test.go",
				Message: "file not found",
				Type:    "filesystem",
			},
			expected: "test.go: file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseErrorHandling(t *testing.T) {
	// Create a temporary file with invalid Go syntax
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	// Test with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		scannedFile := ScannedFile{
			Path:     "nonexistent.go",
			AbsPath:  "/tmp/nonexistent.go",
			Language: "Go",
			Size:     0,
		}

		_, err := goParser.Parse(scannedFile)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}

		detailedErr, ok := err.(*DetailedParseError)
		if !ok {
			t.Errorf("Expected DetailedParseError, got %T", err)
		} else {
			if detailedErr.Type != "filesystem" {
				t.Errorf("Expected error type 'filesystem', got %q", detailedErr.Type)
			}
			if !strings.Contains(detailedErr.Message, "failed to read file") {
				t.Errorf("Expected error message to contain 'failed to read file', got %q", detailedErr.Message)
			}
		}
	})
}

func TestParserPoolErrorCollection(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	pool := NewParserPool(2, tsParser)

	// Create test files with one invalid file
	files := []ScannedFile{
		{
			Path:     "nonexistent1.go",
			AbsPath:  "/tmp/nonexistent1.go",
			Language: "Go",
			Size:     0,
		},
		{
			Path:     "nonexistent2.go",
			AbsPath:  "/tmp/nonexistent2.go",
			Language: "Go",
			Size:     0,
		},
	}

	parsedFiles, errors := pool.Process(files)

	// We should get errors for both files
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	// We might still get partial results
	t.Logf("Parsed %d files with %d errors", len(parsedFiles), len(errors))

	// Verify errors are DetailedParseError
	for i, err := range errors {
		if _, ok := err.(*DetailedParseError); !ok {
			t.Errorf("Error %d: expected DetailedParseError, got %T", i, err)
		}
	}
}
