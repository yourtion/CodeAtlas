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

// TestParseEndToEnd tests parsing the test repository
func TestParseEndToEnd(t *testing.T) {
	// Build the CLI first
	buildCmd := exec.Command("make", "build-cli")
	buildCmd.Dir = "../.."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	// Create temp output file
	tmpFile, err := os.CreateTemp("", "parse-output-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run parse command on test fixtures
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	cmd := exec.Command("../../bin/cli", "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--verbose")

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

	// Verify metadata
	if result.Metadata.Version == "" {
		t.Error("Expected version in metadata")
	}
	if result.Metadata.TotalFiles == 0 {
		t.Error("Expected total_files > 0 in metadata")
	}
	if result.Metadata.SuccessCount == 0 {
		t.Error("Expected success_count > 0 in metadata")
	}

	// Verify files were parsed
	if len(result.Files) == 0 {
		t.Fatal("Expected at least one file in output")
	}

	// Track which files we found
	foundFiles := make(map[string]bool)
	for _, file := range result.Files {
		foundFiles[filepath.Base(file.Path)] = true
	}

	// Verify expected files are present (not ignored)
	expectedFiles := []string{"main.go", "utils.go", "models.py", "utils.py", "app.js", "api.js"}
	for _, expected := range expectedFiles {
		if !foundFiles[expected] {
			t.Errorf("Expected file %s not found in output", expected)
		}
	}

	// Verify ignored files are NOT present
	ignoredFiles := []string{"ignored_file.go"}
	for _, ignored := range ignoredFiles {
		if foundFiles[ignored] {
			t.Errorf("Ignored file %s should not be in output", ignored)
		}
	}

	// Verify symbols were extracted
	symbolCount := 0
	for _, file := range result.Files {
		symbolCount += len(file.Symbols)
	}
	if symbolCount == 0 {
		t.Error("Expected at least one symbol to be extracted")
	}

	// Verify specific symbols from test files
	t.Run("VerifyGoSymbols", func(t *testing.T) {
		verifyGoSymbols(t, result)
	})

	t.Run("VerifyPythonSymbols", func(t *testing.T) {
		verifyPythonSymbols(t, result)
	})

	t.Run("VerifyJavaScriptSymbols", func(t *testing.T) {
		verifyJavaScriptSymbols(t, result)
	})

	t.Run("VerifyRelationships", func(t *testing.T) {
		verifyRelationships(t, result)
	})
}

func verifyGoSymbols(t *testing.T, result schema.ParseOutput) {
	// Find main.go file
	var mainFile *schema.File
	for i := range result.Files {
		if strings.HasSuffix(result.Files[i].Path, "main.go") {
			mainFile = &result.Files[i]
			break
		}
	}

	if mainFile == nil {
		t.Fatal("main.go not found in output")
	}

	// Verify file metadata
	if mainFile.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", mainFile.Language)
	}
	if mainFile.Checksum == "" {
		t.Error("Expected checksum to be set")
	}
	if mainFile.FileID == "" {
		t.Error("Expected file_id to be set")
	}

	// Check for expected symbols
	symbolNames := make(map[string]schema.Symbol)
	for _, sym := range mainFile.Symbols {
		symbolNames[sym.Name] = sym
	}

	expectedSymbols := []struct {
		name string
		kind schema.SymbolKind
	}{
		{"main", schema.SymbolFunction},
		{"ProcessData", schema.SymbolFunction},
		{"Calculator", schema.SymbolClass},
		{"NewCalculator", schema.SymbolFunction},
	}

	for _, expected := range expectedSymbols {
		sym, found := symbolNames[expected.name]
		if !found {
			t.Errorf("Expected symbol '%s' not found", expected.name)
			continue
		}
		if sym.Kind != expected.kind {
			t.Errorf("Symbol '%s': expected kind '%s', got '%s'", expected.name, expected.kind, sym.Kind)
		}
		if sym.SymbolID == "" {
			t.Errorf("Symbol '%s': expected symbol_id to be set", expected.name)
		}
		if sym.Span.StartLine == 0 {
			t.Errorf("Symbol '%s': expected start_line > 0", expected.name)
		}
	}

	// Verify method symbols (Add, Multiply)
	foundAdd := false
	foundMultiply := false
	for _, sym := range mainFile.Symbols {
		if sym.Name == "Add" {
			foundAdd = true
			if sym.Kind != schema.SymbolFunction {
				t.Errorf("Add: expected kind 'function', got '%s'", sym.Kind)
			}
		}
		if sym.Name == "Multiply" {
			foundMultiply = true
		}
	}
	if !foundAdd {
		t.Error("Expected method 'Add' not found")
	}
	if !foundMultiply {
		t.Error("Expected method 'Multiply' not found")
	}
}

func verifyPythonSymbols(t *testing.T, result schema.ParseOutput) {
	// Find models.py file
	var modelsFile *schema.File
	for i := range result.Files {
		if strings.HasSuffix(result.Files[i].Path, "models.py") {
			modelsFile = &result.Files[i]
			break
		}
	}

	if modelsFile == nil {
		t.Fatal("models.py not found in output")
	}

	if modelsFile.Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", modelsFile.Language)
	}

	// Check for expected symbols
	symbolNames := make(map[string]schema.Symbol)
	for _, sym := range modelsFile.Symbols {
		symbolNames[sym.Name] = sym
	}

	expectedSymbols := []struct {
		name string
		kind schema.SymbolKind
	}{
		{"User", schema.SymbolClass},
		{"Repository", schema.SymbolClass},
		{"process_data", schema.SymbolFunction},
		{"fetch_remote_data", schema.SymbolFunction},
	}

	for _, expected := range expectedSymbols {
		sym, found := symbolNames[expected.name]
		if !found {
			t.Errorf("Expected symbol '%s' not found", expected.name)
			continue
		}
		if sym.Kind != expected.kind {
			t.Errorf("Symbol '%s': expected kind '%s', got '%s'", expected.name, expected.kind, sym.Kind)
		}
		// Check for docstrings
		if expected.kind == schema.SymbolClass && sym.Docstring == "" {
			t.Errorf("Symbol '%s': expected docstring to be extracted", expected.name)
		}
	}

	// Verify methods were extracted
	foundInit := false
	foundToDict := false
	for _, sym := range modelsFile.Symbols {
		if sym.Name == "__init__" {
			foundInit = true
		}
		if sym.Name == "to_dict" {
			foundToDict = true
		}
	}
	if !foundInit {
		t.Error("Expected method '__init__' not found")
	}
	if !foundToDict {
		t.Error("Expected method 'to_dict' not found")
	}
}

func verifyJavaScriptSymbols(t *testing.T, result schema.ParseOutput) {
	// Find app.js file
	var appFile *schema.File
	for i := range result.Files {
		if strings.HasSuffix(result.Files[i].Path, "app.js") {
			appFile = &result.Files[i]
			break
		}
	}

	if appFile == nil {
		t.Fatal("app.js not found in output")
	}

	if appFile.Language != "javascript" {
		t.Errorf("Expected language 'javascript', got '%s'", appFile.Language)
	}

	// Check for expected symbols
	symbolNames := make(map[string]schema.Symbol)
	for _, sym := range appFile.Symbols {
		symbolNames[sym.Name] = sym
	}

	expectedSymbols := []struct {
		name string
		kind schema.SymbolKind
	}{
		{"Application", schema.SymbolClass},
		{"processInput", schema.SymbolFunction},
		{"transformData", schema.SymbolVariable},
		{"loadConfig", schema.SymbolVariable},
	}

	for _, expected := range expectedSymbols {
		sym, found := symbolNames[expected.name]
		if !found {
			t.Errorf("Expected symbol '%s' not found", expected.name)
			continue
		}
		if sym.Kind != expected.kind {
			t.Errorf("Symbol '%s': expected kind '%s', got '%s'", expected.name, expected.kind, sym.Kind)
		}
	}

	// Verify methods were extracted
	foundInitialize := false
	foundGetStatus := false
	for _, sym := range appFile.Symbols {
		if sym.Name == "initialize" {
			foundInitialize = true
		}
		if sym.Name == "getStatus" {
			foundGetStatus = true
		}
	}
	if !foundInitialize {
		t.Error("Expected method 'initialize' not found")
	}
	if !foundGetStatus {
		t.Error("Expected method 'getStatus' not found")
	}
}

func verifyRelationships(t *testing.T, result schema.ParseOutput) {
	if len(result.Relationships) == 0 {
		t.Error("Expected at least one relationship to be extracted")
		return
	}

	// Verify relationship structure
	for i, rel := range result.Relationships {
		if rel.EdgeID == "" {
			t.Errorf("Relationship %d: expected edge_id to be set", i)
		}
		if rel.SourceID == "" {
			t.Errorf("Relationship %d: expected source_id to be set", i)
		}
		if rel.TargetID == "" && rel.TargetModule == "" {
			t.Errorf("Relationship %d: expected either target_id or target_module to be set", i)
		}
		if rel.EdgeType == "" {
			t.Errorf("Relationship %d: expected edge_type to be set", i)
		}
		if rel.SourceFile == "" {
			t.Errorf("Relationship %d: expected source_file to be set", i)
		}
	}

	// Count relationship types
	relationshipTypes := make(map[schema.EdgeType]int)
	for _, rel := range result.Relationships {
		relationshipTypes[rel.EdgeType]++
	}

	// We should have at least import relationships
	if relationshipTypes[schema.EdgeImport] == 0 {
		t.Error("Expected at least one import relationship")
	}
}

// TestParseOutputToStdout tests that parse can output to stdout
func TestParseOutputToStdout(t *testing.T) {
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	cmd := exec.Command("../../bin/cli", "parse", "--path", fixturesPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify output is valid JSON
	var result schema.ParseOutput
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(result.Files) == 0 {
		t.Error("Expected at least one file in output")
	}
}

// TestParseNonExistentPath tests error handling for non-existent paths
func TestParseNonExistentPath(t *testing.T) {
	cmd := exec.Command("../../bin/cli", "parse", "--path", "/nonexistent/path")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("Expected parse command to fail for non-existent path")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "does not exist") && !strings.Contains(outputStr, "no such file") {
		t.Errorf("Expected error message about non-existent path, got: %s", outputStr)
	}
}

// TestParseVerboseOutput tests verbose logging
func TestParseVerboseOutput(t *testing.T) {
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	cmd := exec.Command("../../bin/cli", "parse", "--path", fixturesPath, "--verbose")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v", err)
	}

	outputStr := string(output)

	// Verbose output should contain progress information
	verboseIndicators := []string{
		"Scanning",
		"Processing",
		"files",
	}

	foundIndicator := false
	for _, indicator := range verboseIndicators {
		if strings.Contains(outputStr, indicator) {
			foundIndicator = true
			break
		}
	}

	if !foundIndicator {
		t.Error("Expected verbose output to contain progress information")
	}
}
