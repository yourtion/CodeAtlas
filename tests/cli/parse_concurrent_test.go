//go:build parse_tests

package cli_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// skipIfNoParseCommand skips the test if parse command is not available
func skipIfNoParseCommand(t *testing.T) {
	cmd := exec.Command(cliBinaryPath, "parse", "--help")
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping test: parse command not implemented")
	}
}

// TestParseConcurrentProcessing tests parsing with different worker counts
func TestParseConcurrentProcessing(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	workerCounts := []int{1, 2, 4, 8}
	results := make(map[int]*schema.ParseOutput)
	durations := make(map[int]time.Duration)

	for _, workers := range workerCounts {
		t.Run(t.Name()+"_Workers_"+string(rune(workers+'0')), func(t *testing.T) {
			// Create temp output file
			tmpFile, err := os.CreateTemp("", "parse-concurrent-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			// Measure execution time
			start := time.Now()

			// Run parse command with specified worker count
			cmd := exec.Command(cliBinaryPath, "parse",
				"--path", fixturesPath,
				"--output", tmpFile.Name(),
				"--workers", string(rune(workers+'0')))

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Parse command failed with %d workers: %v\nOutput: %s", workers, err, string(output))
			}

			duration := time.Since(start)
			durations[workers] = duration

			// Read and parse JSON output
			data, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			var result schema.ParseOutput
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to parse JSON output: %v", err)
			}

			results[workers] = &result

			t.Logf("Workers: %d, Duration: %v, Files: %d", workers, duration, len(result.Files))
		})
	}

	// Verify all worker counts produced the same results
	t.Run("VerifyConsistentResults", func(t *testing.T) {
		if len(results) < 2 {
			t.Skip("Need at least 2 results to compare")
		}

		// Use 1 worker as baseline
		baseline := results[1]
		if baseline == nil {
			t.Fatal("Baseline result (1 worker) is nil")
		}

		for workers, result := range results {
			if workers == 1 {
				continue
			}

			// Verify same number of files
			if len(result.Files) != len(baseline.Files) {
				t.Errorf("Worker count %d: expected %d files, got %d", workers, len(baseline.Files), len(result.Files))
			}

			// Verify same number of symbols
			baselineSymbols := 0
			for _, file := range baseline.Files {
				baselineSymbols += len(file.Symbols)
			}

			resultSymbols := 0
			for _, file := range result.Files {
				resultSymbols += len(file.Symbols)
			}

			if resultSymbols != baselineSymbols {
				t.Errorf("Worker count %d: expected %d symbols, got %d", workers, baselineSymbols, resultSymbols)
			}

			// Verify same number of relationships
			if len(result.Relationships) != len(baseline.Relationships) {
				t.Errorf("Worker count %d: expected %d relationships, got %d", workers, len(baseline.Relationships), len(result.Relationships))
			}
		}
	})

	// Log performance comparison
	t.Run("LogPerformanceMetrics", func(t *testing.T) {
		if len(durations) < 2 {
			t.Skip("Need at least 2 durations to compare")
		}

		baseline := durations[1]
		t.Logf("Performance comparison (baseline: 1 worker = %v):", baseline)

		for workers := 2; workers <= 8; workers++ {
			if duration, ok := durations[workers]; ok {
				speedup := float64(baseline) / float64(duration)
				t.Logf("  %d workers: %v (%.2fx speedup)", workers, duration, speedup)
			}
		}
	})
}

// TestParseRaceConditions tests for race conditions in concurrent processing
func TestParseRaceConditions(t *testing.T) {
	skipIfBinaryNotExists(t)
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	// Run multiple times to increase chance of detecting race conditions
	iterations := 10
	for i := 0; i < iterations; i++ {
		t.Run(t.Name()+"_Iteration_"+string(rune(i+'0')), func(t *testing.T) {
			// Create temp output file
			tmpFile, err := os.CreateTemp("", "parse-race-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			// Run with maximum workers to stress test
			cmd := exec.Command(cliBinaryPath, "parse",
				"--path", fixturesPath,
				"--output", tmpFile.Name(),
				"--workers", "8")

			// Enable race detector if available
			cmd.Env = append(os.Environ(), "GORACE=halt_on_error=1")

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Parse command failed (iteration %d): %v\nOutput: %s", i, err, string(output))
			}

			// Verify output is valid JSON
			data, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			var result schema.ParseOutput
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to parse JSON output (iteration %d): %v", i, err)
			}

			// Basic sanity checks
			if len(result.Files) == 0 {
				t.Errorf("Iteration %d: expected at least one file", i)
			}
		})
	}
}

// TestParseWorkerCountValidation tests worker count validation
func TestParseWorkerCountValidation(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	testCases := []struct {
		name        string
		workers     string
		shouldFail  bool
		description string
	}{
		{"ZeroWorkers", "0", true, "Zero workers should fail"},
		{"NegativeWorkers", "-1", true, "Negative workers should fail"},
		{"OneWorker", "1", false, "One worker should succeed"},
		{"FourWorkers", "4", false, "Four workers should succeed"},
		{"SixteenWorkers", "16", false, "Sixteen workers should succeed"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "parse-workers-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			cmd := exec.Command(cliBinaryPath, "parse",
				"--path", fixturesPath,
				"--output", tmpFile.Name(),
				"--workers", tc.workers)

			output, err := cmd.CombinedOutput()

			if tc.shouldFail {
				if err == nil {
					t.Errorf("%s: expected command to fail but it succeeded", tc.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: expected command to succeed but it failed: %v\nOutput: %s", tc.description, err, string(output))
				}
			}
		})
	}
}

// TestParseDefaultWorkerCount tests that default worker count is reasonable
func TestParseDefaultWorkerCount(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "parse-default-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run without specifying workers (should use default)
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name(),
		"--verbose")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify output is valid
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(result.Files) == 0 {
		t.Error("Expected at least one file to be parsed")
	}
}

// TestParseLargeRepository tests performance with a larger repository
func TestParseLargeRepository(t *testing.T) {
	skipIfBinaryNotExists(t)
	if testing.Short() {
		t.Skip("Skipping large repository test in short mode")
	}

	// Create a temporary directory with many files
	tmpDir, err := os.MkdirTemp("", "test-large-repo-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create 100 Go files
	fileCount := 100
	for i := 0; i < fileCount; i++ {
		filename := filepath.Join(tmpDir, "file_"+string(rune(i/10+'0'))+string(rune(i%10+'0'))+".go")
		content := `package main

import "fmt"

// Function` + string(rune(i/10+'0')) + string(rune(i%10+'0')) + ` does something
func Function` + string(rune(i/10+'0')) + string(rune(i%10+'0')) + `() {
	fmt.Println("Hello from function ` + string(rune(i/10+'0')) + string(rune(i%10+'0')) + `")
}

// Struct` + string(rune(i/10+'0')) + string(rune(i%10+'0')) + ` represents something
type Struct` + string(rune(i/10+'0')) + string(rune(i%10+'0')) + ` struct {
	Value int
}
`
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test with different worker counts
	workerCounts := []int{1, 4, 8}
	for _, workers := range workerCounts {
		t.Run("Workers_"+string(rune(workers+'0')), func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "parse-large-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			start := time.Now()

			cmd := exec.Command(cliBinaryPath, "parse",
				"--path", tmpDir,
				"--output", tmpFile.Name(),
				"--workers", string(rune(workers+'0')))

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
			}

			duration := time.Since(start)

			// Read and verify output
			data, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			var result schema.ParseOutput
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to parse JSON output: %v", err)
			}

			// Verify all files were parsed
			if len(result.Files) != fileCount {
				t.Errorf("Expected %d files, got %d", fileCount, len(result.Files))
			}

			// Verify symbols were extracted
			totalSymbols := 0
			for _, file := range result.Files {
				totalSymbols += len(file.Symbols)
			}

			// Each file should have at least 2 symbols (function + struct)
			expectedMinSymbols := fileCount * 2
			if totalSymbols < expectedMinSymbols {
				t.Errorf("Expected at least %d symbols, got %d", expectedMinSymbols, totalSymbols)
			}

			t.Logf("Workers: %d, Duration: %v, Files: %d, Symbols: %d", workers, duration, len(result.Files), totalSymbols)

			// Performance target: should complete in reasonable time
			maxDuration := 30 * time.Second
			if duration > maxDuration {
				t.Errorf("Parsing took too long: %v (max: %v)", duration, maxDuration)
			}
		})
	}
}

// TestParseEnvironmentVariableWorkers tests CODEATLAS_WORKERS env var
func TestParseEnvironmentVariableWorkers(t *testing.T) {
	skipIfBinaryNotExists(t)
	fixturesPath, err := filepath.Abs("../fixtures/test-repo")
	if err != nil {
		t.Fatalf("Failed to get fixtures path: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "parse-env-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Set environment variable
	cmd := exec.Command(cliBinaryPath, "parse",
		"--path", fixturesPath,
		"--output", tmpFile.Name())

	cmd.Env = append(os.Environ(), "CODEATLAS_WORKERS=4")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify output is valid
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result schema.ParseOutput
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(result.Files) == 0 {
		t.Error("Expected at least one file to be parsed")
	}
}
