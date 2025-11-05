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

// TestParseErrorRecovery tests that parsing continues after encountering errors
func TestParseErrorRecovery(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-error-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command (test-repo contains syntax_error.go and syntax_error.py)
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--verbose")

	output, err := cmd.CombinedOutput()
	// Command should succeed even with syntax errors
	if err != nil {
		t.Logf("Command output: %s", string(output))
		// Don't fail immediately - check if it's a legitimate error
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

	// Verify metadata shows failures
	if result.Metadata.FailureCount == 0 {
		t.Error("Expected failure_count > 0 due to syntax errors")
	}

	// Verify some files were successfully parsed
	if result.Metadata.SuccessCount == 0 {
		t.Error("Expected success_count > 0 for valid files")
	}

	// Verify total = success + failure
	expectedTotal := result.Metadata.SuccessCount + result.Metadata.FailureCount
	if result.Metadata.TotalFiles != expectedTotal {
		t.Errorf("Total files (%d) should equal success (%d) + failure (%d)",
			result.Metadata.TotalFiles, result.Metadata.SuccessCount, result.Metadata.FailureCount)
	}

	// Verify error details are included
	if len(result.Metadata.Errors) == 0 {
		t.Error("Expected error details in metadata")
	}

	// Check that errors reference the problematic files
	foundSyntaxError := false
	for _, parseErr := range result.Metadata.Errors {
		if strings.Contains(parseErr.File, "syntax_error") {
			foundSyntaxError = true
			if parseErr.Message == "" {
				t.Error("Expected error message to be set")
			}
			if parseErr.Type == "" {
				t.Error("Expected error type to be set")
			}
		}
	}

	if !foundSyntaxError {
		t.Error("Expected error for syntax_error files")
	}

	// Verify valid files were still parsed successfully
	foundValidFile := false
	for _, file := range result.Files {
		if strings.Contains(file.Path, "main.go") || strings.Contains(file.Path, "models.py") {
			foundValidFile = true
			if len(file.Symbols) == 0 {
				t.Errorf("Valid file %s should have symbols extracted", file.Path)
			}
		}
	}

	if !foundValidFile {
		t.Error("Expected at least one valid file to be parsed successfully")
	}
}

// TestParseSyntaxErrors tests handling of files with syntax errors
func TestParseSyntaxErrors(t *testing.T) {
	skipIfBinaryNotExists(t)
	// Create a temporary directory with syntax error files
	tmpDir, err := os.MkdirTemp("", "test-syntax-errors-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files with syntax errors
	testFiles := map[string]string{
		"valid.go": `package main

func ValidFunction() string {
	return "valid"
}
`,
		"broken.go": `package main

func BrokenFunction() {
	if true {
		x := 10
	// Missing closing braces
`,
		"valid.py": `def valid_function():
    """A valid function"""
    return "valid"
`,
		"broken.py": `def broken_function():
    """A broken function"""
    if True:
        x = 10
    # Missing proper indentation and closing
`,
	}

	for filename, content := range testFiles {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-syntax-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", tmpDir,
		"--output", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	// Should not fail completely
	if err != nil {
		t.Logf("Command output: %s", string(output))
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

	// Verify valid files were parsed
	foundValidGo := false
	foundValidPy := false
	for _, file := range result.Files {
		if strings.Contains(file.Path, "valid.go") {
			foundValidGo = true
			if len(file.Symbols) == 0 {
				t.Error("valid.go should have symbols extracted")
			}
		}
		if strings.Contains(file.Path, "valid.py") {
			foundValidPy = true
			if len(file.Symbols) == 0 {
				t.Error("valid.py should have symbols extracted")
			}
		}
	}

	if !foundValidGo {
		t.Error("Expected valid.go to be parsed successfully")
	}
	if !foundValidPy {
		t.Error("Expected valid.py to be parsed successfully")
	}

	// Verify errors were recorded
	if result.Metadata.FailureCount == 0 {
		t.Error("Expected failures for broken files")
	}

	// Verify at least some successes
	if result.Metadata.SuccessCount == 0 {
		t.Error("Expected successes for valid files")
	}
}

// TestParsePartialResults tests that partial results are returned on errors
func TestParsePartialResults(t *testing.T) {
	skipIfBinaryNotExists(t)
	// Create a temporary directory with mixed valid/invalid files
	tmpDir, err := os.MkdirTemp("", "test-partial-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple valid files and one broken file
	validFiles := []string{"file1.go", "file2.go", "file3.go"}
	for i, filename := range validFiles {
		content := `package main

func Function` + string(rune(i+'0')) + `() {
	// Valid function
}
`
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create one broken file
	brokenContent := `package main

func BrokenFunction() {
	// Missing closing brace
`
	brokenPath := filepath.Join(tmpDir, "broken.go")
	if err := os.WriteFile(brokenPath, []byte(brokenContent), 0644); err != nil {
		t.Fatalf("Failed to create broken file: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-partial-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", tmpDir,
		"--output", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Command output: %s", string(output))
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

	// Verify we got partial results (3 valid files)
	if len(result.Files) != 3 {
		t.Errorf("Expected 3 valid files to be parsed, got %d", len(result.Files))
	}

	// Verify metadata reflects the partial success
	if result.Metadata.SuccessCount != 3 {
		t.Errorf("Expected success_count = 3, got %d", result.Metadata.SuccessCount)
	}

	if result.Metadata.FailureCount != 1 {
		t.Errorf("Expected failure_count = 1, got %d", result.Metadata.FailureCount)
	}
}

// TestParseFileReadErrors tests handling of file read errors
func TestParseFileReadErrors(t *testing.T) {
	skipIfBinaryNotExists(t)
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-read-errors-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid file
	validPath := filepath.Join(tmpDir, "valid.go")
	if err := os.WriteFile(validPath, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("Failed to create valid file: %v", err)
	}

	// Create a file with no read permissions (Unix-like systems only)
	if os.Getenv("GOOS") != "windows" {
		noReadPath := filepath.Join(tmpDir, "noread.go")
		if err := os.WriteFile(noReadPath, []byte("package main\n"), 0000); err != nil {
			t.Fatalf("Failed to create no-read file: %v", err)
		}
		defer os.Chmod(noReadPath, 0644) // Cleanup
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-read-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", tmpDir,
		"--output", tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Command output: %s", string(output))
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

	// Verify at least the valid file was parsed
	if result.Metadata.SuccessCount == 0 {
		t.Error("Expected at least one successful parse")
	}

	// On Unix-like systems, verify the unreadable file caused an error
	if os.Getenv("GOOS") != "windows" {
		if result.Metadata.FailureCount == 0 {
			t.Error("Expected failure for unreadable file")
		}
	}
}

// TestParseGracefulDegradation tests graceful degradation on various errors
func TestParseGracefulDegradation(t *testing.T) {
	skipIfBinaryNotExists(t)
	testCases := []struct {
		name        string
		setupFunc   func(string) error
		expectFiles int
		expectError bool
	}{
		{
			name: "EmptyDirectory",
			setupFunc: func(dir string) error {
				// Empty directory - no files to create
				return nil
			},
			expectFiles: 0,
			expectError: false,
		},
		{
			name: "OnlyInvalidFiles",
			setupFunc: func(dir string) error {
				// Create only files with syntax errors
				return os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package main\nfunc broken() {"), 0644)
			},
			expectFiles: 0,
			expectError: false, // Should not fail, just report errors
		},
		{
			name: "MixedValidInvalid",
			setupFunc: func(dir string) error {
				// Create mix of valid and invalid files
				if err := os.WriteFile(filepath.Join(dir, "valid.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package main\nfunc broken() {"), 0644)
			},
			expectFiles: 1,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "test-degradation-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup test scenario
			if err := tc.setupFunc(tmpDir); err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create temp output file
			tmpFile, err := os.CreateTemp("", "parse-degrade-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			// Run parse command
			cmd := exec.Command(cliBinaryPath, "parse",
				"--path", tmpDir,
				"--output", tmpFile.Name())

			output, err := cmd.CombinedOutput()

			if tc.expectError && err == nil {
				t.Error("Expected command to fail but it succeeded")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected command to succeed but it failed: %v\nOutput: %s", err, string(output))
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

			// Verify expected number of files
			if len(result.Files) != tc.expectFiles {
				t.Errorf("Expected %d files, got %d", tc.expectFiles, len(result.Files))
			}

			// Verify metadata is present
			if result.Metadata.Version == "" {
				t.Error("Expected version in metadata")
			}
		})
	}
}

// TestParseErrorSummary tests that error summary is properly formatted
func TestParseErrorSummary(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-summary-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with verbose to see error summary
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--verbose")

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Verbose output should contain summary information
	summaryIndicators := []string{
		"total",
		"success",
		"fail",
	}

	for _, indicator := range summaryIndicators {
		if !strings.Contains(strings.ToLower(outputStr), indicator) {
			t.Errorf("Expected summary to contain '%s'", indicator)
		}
	}

	// Read JSON output
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify error structure
	for i, parseErr := range result.Metadata.Errors {
		if parseErr.File == "" {
			t.Errorf("Error %d: expected file to be set", i)
		}
		if parseErr.Message == "" {
			t.Errorf("Error %d: expected message to be set", i)
		}
		if parseErr.Type == "" {
			t.Errorf("Error %d: expected type to be set", i)
		}
	}
}
