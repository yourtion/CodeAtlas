package cli_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestCLIVersion tests that the version flag works
func TestCLIVersion(t *testing.T) {
	cmd := exec.Command("../../bin/cli", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run CLI: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "1.0.0") {
		t.Errorf("Expected version 1.0.0, got: %s", outputStr)
	}
}

// TestCLIHelp tests that help displays available commands
func TestCLIHelp(t *testing.T) {
	cmd := exec.Command("../../bin/cli", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run CLI: %v", err)
	}

	outputStr := string(output)
	
	// Check for both parse and upload commands
	if !strings.Contains(outputStr, "parse") {
		t.Error("Expected 'parse' command in help output")
	}
	
	if !strings.Contains(outputStr, "upload") {
		t.Error("Expected 'upload' command in help output")
	}
}

// TestParseCommandHelp tests that parse command help is displayed
func TestParseCommandHelp(t *testing.T) {
	cmd := exec.Command("../../bin/cli", "parse", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run CLI: %v", err)
	}

	outputStr := string(output)
	
	// Check for key flags
	expectedFlags := []string{
		"--path",
		"--output",
		"--file",
		"--language",
		"--workers",
		"--semantic",
		"--verbose",
		"--ignore-file",
		"--ignore-pattern",
		"--no-ignore",
	}
	
	for _, flag := range expectedFlags {
		if !strings.Contains(outputStr, flag) {
			t.Errorf("Expected flag '%s' in help output", flag)
		}
	}
	
	// Check for environment variables documentation
	if !strings.Contains(outputStr, "CODEATLAS_LLM_API_KEY") {
		t.Error("Expected CODEATLAS_LLM_API_KEY in help output")
	}
	
	if !strings.Contains(outputStr, "CODEATLAS_WORKERS") {
		t.Error("Expected CODEATLAS_WORKERS in help output")
	}
	
	if !strings.Contains(outputStr, "CODEATLAS_VERBOSE") {
		t.Error("Expected CODEATLAS_VERBOSE in help output")
	}
}

// TestUploadCommandHelp tests that upload command help is displayed
func TestUploadCommandHelp(t *testing.T) {
	cmd := exec.Command("../../bin/cli", "upload", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run CLI: %v", err)
	}

	outputStr := string(output)
	
	// Check for key flags
	expectedFlags := []string{
		"--path",
		"--server",
		"--name",
	}
	
	for _, flag := range expectedFlags {
		if !strings.Contains(outputStr, flag) {
			t.Errorf("Expected flag '%s' in help output", flag)
		}
	}
	
	// Check for environment variables documentation
	if !strings.Contains(outputStr, "CODEATLAS_SERVER") {
		t.Error("Expected CODEATLAS_SERVER in help output")
	}
}

// TestParseCommandRequiresInput tests that parse command validates input
func TestParseCommandRequiresInput(t *testing.T) {
	cmd := exec.Command("../../bin/cli", "parse")
	output, err := cmd.CombinedOutput()
	
	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected parse command to fail without --path or --file")
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "either --path or --file must be specified") {
		t.Errorf("Expected error message about missing input, got: %s", outputStr)
	}
}

// TestParseCommandSemanticRequiresAPIKey tests that semantic flag requires API key
func TestParseCommandSemanticRequiresAPIKey(t *testing.T) {
	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	tmpFile.WriteString("package main\n\nfunc main() {}\n")
	tmpFile.Close()
	
	// Unset the API key if it exists
	oldKey := os.Getenv("CODEATLAS_LLM_API_KEY")
	os.Unsetenv("CODEATLAS_LLM_API_KEY")
	defer func() {
		if oldKey != "" {
			os.Setenv("CODEATLAS_LLM_API_KEY", oldKey)
		}
	}()
	
	cmd := exec.Command("../../bin/cli", "parse", "--file", tmpFile.Name(), "--semantic")
	output, err := cmd.CombinedOutput()
	
	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected parse command to fail without CODEATLAS_LLM_API_KEY")
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "CODEATLAS_LLM_API_KEY") {
		t.Errorf("Expected error message about missing API key, got: %s", outputStr)
	}
}

// TestConsistentFlagNaming tests that parse and upload use consistent flag names
func TestConsistentFlagNaming(t *testing.T) {
	// Get parse command help
	parseCmd := exec.Command("../../bin/cli", "parse", "--help")
	parseOutput, err := parseCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run parse help: %v", err)
	}
	
	// Get upload command help
	uploadCmd := exec.Command("../../bin/cli", "upload", "--help")
	uploadOutput, err := uploadCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run upload help: %v", err)
	}
	
	parseStr := string(parseOutput)
	uploadStr := string(uploadOutput)
	
	// Both should have --path flag
	if !strings.Contains(parseStr, "--path") {
		t.Error("Parse command missing --path flag")
	}
	if !strings.Contains(uploadStr, "--path") {
		t.Error("Upload command missing --path flag")
	}
	
	// Both should have -p alias
	if !strings.Contains(parseStr, "-p") {
		t.Error("Parse command missing -p alias")
	}
	if !strings.Contains(uploadStr, "-p") {
		t.Error("Upload command missing -p alias")
	}
}
