//go:build parse_tests

package cli_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

// TestParseLanguageFilterGo tests filtering for Go files only
func TestParseLanguageFilterGo(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-lang-go-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with Go language filter
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--language", "go")

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

	// Verify only Go files were parsed
	if len(result.Files) == 0 {
		t.Fatal("Expected at least one Go file to be parsed")
	}

	for _, file := range result.Files {
		if file.Language != "go" {
			t.Errorf("Expected only Go files, found: %s (language: %s)", file.Path, file.Language)
		}
	}

	// Verify we got the expected Go files
	expectedGoFiles := 0
	for _, file := range result.Files {
		if file.Language == "go" {
			expectedGoFiles++
		}
	}

	if expectedGoFiles == 0 {
		t.Error("Expected at least one Go file")
	}
}

// TestParseLanguageFilterPython tests filtering for Python files only
func TestParseLanguageFilterPython(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-lang-py-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with Python language filter
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--language", "python")

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

	// Verify only Python files were parsed
	if len(result.Files) == 0 {
		t.Fatal("Expected at least one Python file to be parsed")
	}

	for _, file := range result.Files {
		if file.Language != "python" {
			t.Errorf("Expected only Python files, found: %s (language: %s)", file.Path, file.Language)
		}
	}
}

// TestParseLanguageFilterJavaScript tests filtering for JavaScript files only
func TestParseLanguageFilterJavaScript(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-lang-js-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with JavaScript language filter
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--language", "javascript")

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

	// Verify only JavaScript files were parsed
	if len(result.Files) == 0 {
		t.Fatal("Expected at least one JavaScript file to be parsed")
	}

	for _, file := range result.Files {
		if file.Language != "javascript" {
			t.Errorf("Expected only JavaScript files, found: %s (language: %s)", file.Path, file.Language)
		}
	}
}

// TestParseLanguageFilterCaseInsensitive tests case-insensitive language filter
func TestParseLanguageFilterCaseInsensitive(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	testCases := []struct {
		name     string
		language string
		expected string
	}{
		{"UpperCase", "GO", "go"},
		{"MixedCase", "Python", "python"},
		{"LowerCase", "javascript", "javascript"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "parse-case-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			cmd := exec.Command(cliBinaryPath, "parse",
				"--path", fixturesPath,
				"--output", tmpFile.Name(),
				"--language", tc.language)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
			}

			data, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			var result schema.ParseOutput
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to parse JSON output: %v", err)
			}

			// Verify correct language files were parsed
			for _, file := range result.Files {
				if file.Language != tc.expected {
					t.Errorf("Expected language '%s', found: %s", tc.expected, file.Language)
				}
			}
		})
	}
}

// TestParseLanguageFilterInvalid tests handling of invalid language filter
func TestParseLanguageFilterInvalid(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "parse-invalid-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with invalid language
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--language", "rust") // Unsupported language

	output, err := cmd.CombinedOutput()

	// Should either fail or return empty results
	if err == nil {
		// If it doesn't fail, verify empty or minimal results
		data, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		var result schema.ParseOutput
		if err := json.Unmarshal(data, &result); err == nil {
			if len(result.Files) > 0 {
				t.Error("Expected no files for unsupported language filter")
			}
		}
	} else {
		// If it fails, that's also acceptable
		t.Logf("Command failed as expected for invalid language: %s", string(output))
	}
}

// TestParseLanguageFilterWithIgnore tests language filter combined with ignore rules
func TestParseLanguageFilterWithIgnore(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "parse-filter-ignore-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with language filter and ignore pattern
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--language", "go",
		"--ignore-pattern", "utils.go")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify only Go files (excluding utils.go) were parsed
	for _, file := range result.Files {
		if file.Language != "go" {
			t.Errorf("Expected only Go files, found: %s", file.Language)
		}
		if filepath.Base(file.Path) == "utils.go" {
			t.Error("utils.go should be ignored")
		}
	}

	// Verify we still got some Go files
	if len(result.Files) == 0 {
		t.Error("Expected at least one Go file (main.go)")
	}
}

// TestParseLanguageFilterMultipleExtensions tests language with multiple extensions
func TestParseLanguageFilterMultipleExtensions(t *testing.T) {
	skipIfBinaryNotExists(t)
	// Create a temporary directory with JavaScript and TypeScript files
	tmpDir, err := os.MkdirTemp("", "test-multi-ext-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := map[string]string{
		"app.js":  "function test() { return 'js'; }",
		"app.jsx": "function Component() { return <div>JSX</div>; }",
		"app.ts":  "function test(): string { return 'ts'; }",
		"app.tsx": "function Component(): JSX.Element { return <div>TSX</div>; }",
		"main.go": "package main\n\nfunc main() {}",
	}

	for filename, content := range testFiles {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tmpFile, err := os.CreateTemp("", "parse-multi-ext-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with JavaScript filter (should include .js and .jsx)
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", tmpDir,
		"--output", tmpFile.Name(),
		"--language", "javascript")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify only JavaScript files were parsed
	jsCount := 0
	for _, file := range result.Files {
		if file.Language == "javascript" {
			jsCount++
		} else {
			t.Errorf("Expected only JavaScript files, found: %s (language: %s)", file.Path, file.Language)
		}
	}

	// Should have parsed .js and .jsx files (2 files)
	if jsCount < 2 {
		t.Errorf("Expected at least 2 JavaScript files (.js and .jsx), got %d", jsCount)
	}
}

// TestParseLanguageFilterEmptyResult tests language filter with no matching files
func TestParseLanguageFilterEmptyResult(t *testing.T) {
	skipIfBinaryNotExists(t)
	// Create a temporary directory with only Go files
	tmpDir, err := os.MkdirTemp("", "test-empty-filter-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create only Go files
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc main() {}"), 0644); err != nil {
		t.Fatalf("Failed to create Go file: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "parse-empty-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with Python filter (no Python files exist)
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", tmpDir,
		"--output", tmpFile.Name(),
		"--language", "python")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Command output: %s", string(output))
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify no files were parsed
	if len(result.Files) != 0 {
		t.Errorf("Expected 0 files for non-matching language filter, got %d", len(result.Files))
	}

	// Verify metadata reflects this
	if result.Metadata.TotalFiles != 0 {
		t.Errorf("Expected total_files = 0, got %d", result.Metadata.TotalFiles)
	}
}

// TestParseLanguageFilterWithWorkers tests language filter with concurrent processing
func TestParseLanguageFilterWithWorkers(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "parse-filter-workers-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with language filter and multiple workers
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--language", "go",
		"--workers", "4")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify only Go files were parsed
	for _, file := range result.Files {
		if file.Language != "go" {
			t.Errorf("Expected only Go files, found: %s", file.Language)
		}
	}

	if len(result.Files) == 0 {
		t.Error("Expected at least one Go file")
	}
}

// TestParseLanguageFilterVerbose tests verbose output with language filter
func TestParseLanguageFilterVerbose(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--language", "go",
		"--verbose")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v", err)
	}

	outputStr := string(output)

	// Verbose output should mention the language filter
	// (implementation-dependent, but good to check)
	t.Logf("Verbose output: %s", outputStr)

	// Verify output is valid JSON
	var result schema.ParseOutput
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
}
