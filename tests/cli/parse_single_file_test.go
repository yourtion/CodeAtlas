//go:build parse_tests

package cli_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// skipIfNoParseCommand skips the test if parse command is not available
func skipIfNoParseCommand(t *testing.T) {
	cmd := exec.Command(cliBinaryPath, "parse", "--help")
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping test: parse command not implemented")
	}
}

// TestParseSingleGoFile tests parsing a single Go file
func TestParseSingleGoFile(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	singleFile := filepath.Join(fixturesPath, "main.go")

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-single-go-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command on single file
	cmd := exec.Command(cliBinaryPath, "parse",
		"--file", singleFile,
		"--output", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	// Read and parse JSON output
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify only one file in output
	if len(result.Files) != 1 {
		t.Errorf("Expected exactly 1 file, got %d", len(result.Files))
	}

	// Verify it's the correct file
	if len(result.Files) > 0 {
		if !strings.Contains(result.Files[0].Path, "main.go") {
			t.Errorf("Expected main.go, got %s", result.Files[0].Path)
		}

		// Verify language detection
		if result.Files[0].Language != "go" {
			t.Errorf("Expected language 'go', got '%s'", result.Files[0].Language)
		}

		// Verify symbols were extracted
		if len(result.Files[0].Symbols) == 0 {
			t.Error("Expected symbols to be extracted from main.go")
		}

		// Verify specific symbols
		symbolNames := make(map[string]bool)
		for _, sym := range result.Files[0].Symbols {
			symbolNames[sym.Name] = true
		}

		expectedSymbols := []string{"main", "ProcessData", "Calculator"}
		for _, expected := range expectedSymbols {
			if !symbolNames[expected] {
				t.Errorf("Expected symbol '%s' not found", expected)
			}
		}
	}

	// Verify metadata
	if result.Metadata.TotalFiles != 1 {
		t.Errorf("Expected total_files = 1, got %d", result.Metadata.TotalFiles)
	}
	if result.Metadata.SuccessCount != 1 {
		t.Errorf("Expected success_count = 1, got %d", result.Metadata.SuccessCount)
	}
}

// TestParseSinglePythonFile tests parsing a single Python file
func TestParseSinglePythonFile(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	singleFile := filepath.Join(fixturesPath, "models.py")

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-single-py-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command on single file
	cmd := exec.Command(cliBinaryPath, "parse",
		"--file", singleFile,
		"--output", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	// Read and parse JSON output
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify only one file in output
	if len(result.Files) != 1 {
		t.Errorf("Expected exactly 1 file, got %d", len(result.Files))
	}

	// Verify it's the correct file
	if len(result.Files) > 0 {
		if !strings.Contains(result.Files[0].Path, "models.py") {
			t.Errorf("Expected models.py, got %s", result.Files[0].Path)
		}

		// Verify language detection
		if result.Files[0].Language != "python" {
			t.Errorf("Expected language 'python', got '%s'", result.Files[0].Language)
		}

		// Verify symbols were extracted
		if len(result.Files[0].Symbols) == 0 {
			t.Error("Expected symbols to be extracted from models.py")
		}

		// Verify specific symbols
		symbolNames := make(map[string]bool)
		for _, sym := range result.Files[0].Symbols {
			symbolNames[sym.Name] = true
		}

		expectedSymbols := []string{"User", "Repository", "process_data"}
		for _, expected := range expectedSymbols {
			if !symbolNames[expected] {
				t.Errorf("Expected symbol '%s' not found", expected)
			}
		}
	}
}

// TestParseSingleJavaScriptFile tests parsing a single JavaScript file
func TestParseSingleJavaScriptFile(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	singleFile := filepath.Join(fixturesPath, "app.js")

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-single-js-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command on single file
	cmd := exec.Command(cliBinaryPath, "parse",
		"--file", singleFile,
		"--output", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	// Read and parse JSON output
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify only one file in output
	if len(result.Files) != 1 {
		t.Errorf("Expected exactly 1 file, got %d", len(result.Files))
	}

	// Verify it's the correct file
	if len(result.Files) > 0 {
		if !strings.Contains(result.Files[0].Path, "app.js") {
			t.Errorf("Expected app.js, got %s", result.Files[0].Path)
		}

		// Verify language detection
		if result.Files[0].Language != "javascript" {
			t.Errorf("Expected language 'javascript', got '%s'", result.Files[0].Language)
		}

		// Verify symbols were extracted
		if len(result.Files[0].Symbols) == 0 {
			t.Error("Expected symbols to be extracted from app.js")
		}

		// Verify specific symbols
		symbolNames := make(map[string]bool)
		for _, sym := range result.Files[0].Symbols {
			symbolNames[sym.Name] = true
		}

		expectedSymbols := []string{"Application", "processInput"}
		for _, expected := range expectedSymbols {
			if !symbolNames[expected] {
				t.Errorf("Expected symbol '%s' not found", expected)
			}
		}
	}
}

// TestParseSingleFileToStdout tests single file parsing to stdout
func TestParseSingleFileToStdout(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	singleFile := filepath.Join(fixturesPath, "main.go")

	// Run parse command without output file (should go to stdout)
	cmd := exec.Command(cliBinaryPath, "parse", "--file", singleFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify output is valid JSON
	var result schema.ParseOutput
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify only one file
	if len(result.Files) != 1 {
		t.Errorf("Expected exactly 1 file, got %d", len(result.Files))
	}
}

// TestParseSingleFileNonExistent tests error handling for non-existent file
func TestParseSingleFileNonExistent(t *testing.T) {
	skipIfBinaryNotExists(t)
	cmd := exec.Command(cliBinaryPath, "parse", "--file", "/nonexistent/file.go")

	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("Expected command to fail for non-existent file")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "does not exist") && !strings.Contains(outputStr, "no such file") {
		t.Errorf("Expected error message about non-existent file, got: %s", outputStr)
	}
}

// TestParseSingleFileUnsupportedLanguage tests handling of unsupported file types
func TestParseSingleFileUnsupportedLanguage(t *testing.T) {
	skipIfBinaryNotExists(t)
	// Create a temporary file with unsupported extension
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("This is a text file, not source code")
	tmpFile.Close()

	// Run parse command
	cmd := exec.Command(cliBinaryPath, "parse", "--file", tmpFile.Name())

	output, err := cmd.CombinedOutput()

	// Should either fail or return empty results
	if err == nil {
		// If it doesn't fail, verify empty results
		var result schema.ParseOutput
		if err := json.Unmarshal(output, &result); err == nil {
			if len(result.Files) > 0 {
				t.Error("Expected no files to be parsed for unsupported language")
			}
		}
	}
}

// TestParseSingleFileSyntaxError tests single file with syntax error
func TestParseSingleFileSyntaxError(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	singleFile := filepath.Join(fixturesPath, "syntax_error.go")

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-error-single-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command
	cmd := exec.Command(cliBinaryPath, "parse",
		"--file", singleFile,
		"--output", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	// May or may not fail - check the output
	t.Logf("Command output: %s", string(output))

	// Read JSON output
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify error was recorded
	if result.Metadata.FailureCount == 0 {
		t.Error("Expected failure_count > 0 for syntax error file")
	}

	// Verify error details
	if len(result.Metadata.Errors) == 0 {
		t.Error("Expected error details in metadata")
	}
}

// TestParseSingleFileAbsolutePath tests parsing with absolute path
func TestParseSingleFileAbsolutePath(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	absolutePath := filepath.Join(fixturesPath, "main.go")

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-abs-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with absolute path
	cmd := exec.Command(cliBinaryPath, "parse",
		"--file", absolutePath,
		"--output", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	// Read and parse JSON output
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify file was parsed
	if len(result.Files) != 1 {
		t.Errorf("Expected exactly 1 file, got %d", len(result.Files))
	}
}

// TestParseSingleFileRelativePath tests parsing with relative path
func TestParseSingleFileRelativePath(t *testing.T) {
	skipIfBinaryNotExists(t)
	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-rel-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with relative path
	cmd := exec.Command(cliBinaryPath, "parse",
		"--file", "../fixtures/test-repo/main.go",
		"--output", tmpFile.Name())

	// Set working directory to tests/cli
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	// Read and parse JSON output
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify file was parsed
	if len(result.Files) != 1 {
		t.Errorf("Expected exactly 1 file, got %d", len(result.Files))
	}
}

// TestParseSingleFileWithVerbose tests verbose output for single file
func TestParseSingleFileWithVerbose(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	singleFile := filepath.Join(fixturesPath, "main.go")

	// Run parse command with verbose
	cmd := exec.Command(cliBinaryPath, "parse",
		"--file", singleFile,
		"--verbose")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v", err)
	}

	outputStr := string(output)

	// Verbose output should contain progress information
	if !strings.Contains(outputStr, "main.go") {
		t.Error("Expected verbose output to mention the file being parsed")
	}
}
