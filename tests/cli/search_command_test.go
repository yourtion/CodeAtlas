package cli

import (
	"os/exec"
	"strings"
	"testing"
)

// TestSearchCommandHelp tests that search command help is displayed
func TestSearchCommandHelp(t *testing.T) {
	skipIfBinaryNotExists(t)
	cmd := exec.Command(cliBinaryPath, "search", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run CLI: %v", err)
	}

	outputStr := string(output)

	// Check for expected flags
	expectedFlags := []string{
		"--query",
		"--repo-id",
		"--language",
		"--kind",
		"--limit",
		"--api-url",
		"--embedding-endpoint",
		"--embedding-model",
	}

	for _, flag := range expectedFlags {
		if !strings.Contains(outputStr, flag) {
			t.Errorf("Expected flag '%s' in help output", flag)
		}
	}

	// Check for description
	if !strings.Contains(outputStr, "semantic search") {
		t.Error("Expected 'semantic search' in help output")
	}
}

// TestSearchCommandMissingQuery tests that search command requires query flag
func TestSearchCommandMissingQuery(t *testing.T) {
	skipIfBinaryNotExists(t)
	cmd := exec.Command(cliBinaryPath, "search")
	output, err := cmd.CombinedOutput()

	// Command should fail without query
	if err == nil {
		t.Error("Expected command to fail without query flag")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "query") {
		t.Error("Expected error message to mention 'query'")
	}
}

// TestSearchCommandMissingAPIURL tests that search command requires API URL
func TestSearchCommandMissingAPIURL(t *testing.T) {
	skipIfBinaryNotExists(t)
	cmd := exec.Command(cliBinaryPath, "search", "--query", "test query")
	output, err := cmd.CombinedOutput()

	// Command should fail without API URL
	if err == nil {
		t.Error("Expected command to fail without API URL")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "API URL") {
		t.Error("Expected error message to mention 'API URL'")
	}
}
