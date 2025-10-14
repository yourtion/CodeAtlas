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

// TestParseIgnoreRules tests that .gitignore rules are respected
func TestParseIgnoreRules(t *testing.T) {
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-ignore-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command
	cmd := exec.Command("../../bin/cli", "parse",
		"--path", fixturesPath,
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

	// Collect all parsed file paths
	parsedFiles := make(map[string]bool)
	for _, file := range result.Files {
		parsedFiles[filepath.Base(file.Path)] = true
	}

	// Verify ignored files are NOT in output
	ignoredFiles := []string{
		"ignored_file.go", // Explicitly ignored in .gitignore
	}

	for _, ignored := range ignoredFiles {
		if parsedFiles[ignored] {
			t.Errorf("File '%s' should be ignored but was parsed", ignored)
		}
	}

	// Verify non-ignored files ARE in output
	expectedFiles := []string{
		"main.go",
		"utils.go",
		"models.py",
		"utils.py",
		"app.js",
		"api.js",
	}

	for _, expected := range expectedFiles {
		if !parsedFiles[expected] {
			t.Errorf("File '%s' should be parsed but was not found", expected)
		}
	}
}

// TestParseNestedGitignore tests that nested .gitignore files are respected
func TestParseNestedGitignore(t *testing.T) {
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-nested-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command
	cmd := exec.Command("../../bin/cli", "parse",
		"--path", fixturesPath,
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

	// Check that local_config.js (ignored by subdir/.gitignore) is not parsed
	foundLocalConfig := false
	for _, file := range result.Files {
		if strings.Contains(file.Path, "local_config.js") {
			foundLocalConfig = true
			break
		}
	}

	if foundLocalConfig {
		t.Error("local_config.js should be ignored by nested .gitignore but was parsed")
	}

	// Check that helper.go (not ignored) IS parsed
	foundHelper := false
	for _, file := range result.Files {
		if strings.Contains(file.Path, "helper.go") {
			foundHelper = true
			break
		}
	}

	if !foundHelper {
		t.Error("helper.go should be parsed but was not found")
	}
}

// TestParseCustomIgnoreFile tests custom ignore file support
func TestParseCustomIgnoreFile(t *testing.T) {
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create custom ignore file
	customIgnore, err := os.CreateTemp("", "custom-ignore-*.txt")
	if err != nil {
		t.Fatalf("Failed to create custom ignore file: %v", err)
	}
	defer os.Remove(customIgnore.Name())

	// Add pattern to ignore all Python files
	customIgnore.WriteString("*.py\n")
	customIgnore.Close()

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-custom-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with custom ignore file
	cmd := exec.Command("../../bin/cli", "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--ignore-file", customIgnore.Name())

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

	// Verify no Python files were parsed
	for _, file := range result.Files {
		if file.Language == "python" {
			t.Errorf("Python file '%s' should be ignored but was parsed", file.Path)
		}
	}

	// Verify Go and JS files were still parsed
	foundGo := false
	foundJS := false
	for _, file := range result.Files {
		if file.Language == "go" {
			foundGo = true
		}
		if file.Language == "javascript" {
			foundJS = true
		}
	}

	if !foundGo {
		t.Error("Expected at least one Go file to be parsed")
	}
	if !foundJS {
		t.Error("Expected at least one JavaScript file to be parsed")
	}
}

// TestParseCustomIgnorePattern tests command-line ignore patterns
func TestParseCustomIgnorePattern(t *testing.T) {
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-pattern-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with ignore pattern for JavaScript files
	cmd := exec.Command("../../bin/cli", "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--ignore-pattern", "*.js")

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

	// Verify no JavaScript files were parsed
	for _, file := range result.Files {
		if file.Language == "javascript" {
			t.Errorf("JavaScript file '%s' should be ignored but was parsed", file.Path)
		}
	}

	// Verify other languages were still parsed
	foundGo := false
	foundPython := false
	for _, file := range result.Files {
		if file.Language == "go" {
			foundGo = true
		}
		if file.Language == "python" {
			foundPython = true
		}
	}

	if !foundGo {
		t.Error("Expected at least one Go file to be parsed")
	}
	if !foundPython {
		t.Error("Expected at least one Python file to be parsed")
	}
}

// TestParseMultipleIgnorePatterns tests multiple ignore patterns
func TestParseMultipleIgnorePatterns(t *testing.T) {
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-multi-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with multiple ignore patterns
	cmd := exec.Command("../../bin/cli", "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--ignore-pattern", "*.js",
		"--ignore-pattern", "*.py")

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
	for _, file := range result.Files {
		if file.Language != "go" {
			t.Errorf("Only Go files should be parsed, found: %s (%s)", file.Path, file.Language)
		}
	}

	if len(result.Files) == 0 {
		t.Error("Expected at least one Go file to be parsed")
	}
}

// TestParseNoIgnore tests that --no-ignore disables all ignore rules
func TestParseNoIgnore(t *testing.T) {
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-noignore-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command with --no-ignore
	cmd := exec.Command("../../bin/cli", "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--no-ignore")

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

	// Verify that ignored_file.go IS parsed (since we disabled ignore rules)
	foundIgnoredFile := false
	for _, file := range result.Files {
		if strings.Contains(file.Path, "ignored_file.go") {
			foundIgnoredFile = true
			break
		}
	}

	if !foundIgnoredFile {
		t.Error("ignored_file.go should be parsed with --no-ignore flag")
	}
}

// TestParseDefaultIgnorePatterns tests that default patterns are applied
func TestParseDefaultIgnorePatterns(t *testing.T) {
	// Create a temporary test directory with files that should be ignored by default
	tmpDir, err := os.MkdirTemp("", "test-default-ignore-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files that should be ignored by default
	testFiles := map[string]string{
		"main.go":              "package main\n\nfunc main() {}\n",
		"node_modules/lib.js":  "// Should be ignored\n",
		"__pycache__/cache.py": "# Should be ignored\n",
		"vendor/dep.go":        "// Should be ignored\n",
		"image.png":            "fake image data",
		"binary.exe":           "fake binary",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-default-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command
	cmd := exec.Command("../../bin/cli", "parse",
		"--path", tmpDir,
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

	// Verify only main.go was parsed
	if len(result.Files) != 1 {
		t.Errorf("Expected exactly 1 file to be parsed, got %d", len(result.Files))
	}

	if len(result.Files) > 0 {
		if !strings.Contains(result.Files[0].Path, "main.go") {
			t.Errorf("Expected main.go to be parsed, got: %s", result.Files[0].Path)
		}
	}

	// Verify ignored files are not in output
	for _, file := range result.Files {
		if strings.Contains(file.Path, "node_modules") {
			t.Error("node_modules should be ignored by default")
		}
		if strings.Contains(file.Path, "__pycache__") {
			t.Error("__pycache__ should be ignored by default")
		}
		if strings.Contains(file.Path, "vendor") {
			t.Error("vendor should be ignored by default")
		}
		if strings.HasSuffix(file.Path, ".png") {
			t.Error("Binary files (.png) should be ignored by default")
		}
		if strings.HasSuffix(file.Path, ".exe") {
			t.Error("Binary files (.exe) should be ignored by default")
		}
	}
}
