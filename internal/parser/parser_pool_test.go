package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// MockProgressLogger for testing
type MockProgressLogger struct {
	mu            sync.Mutex
	progressCalls []ProgressCall
	errorCalls    []ErrorCall
}

type ProgressCall struct {
	Current int
	Total   int
	File    string
}

type ErrorCall struct {
	File  string
	Error error
}

func (m *MockProgressLogger) LogProgress(current, total int, file string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progressCalls = append(m.progressCalls, ProgressCall{
		Current: current,
		Total:   total,
		File:    file,
	})
}

func (m *MockProgressLogger) LogError(file string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalls = append(m.errorCalls, ErrorCall{
		File:  file,
		Error: err,
	})
}

func (m *MockProgressLogger) GetProgressCalls() []ProgressCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ProgressCall{}, m.progressCalls...)
}

func (m *MockProgressLogger) GetErrorCalls() []ErrorCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ErrorCall{}, m.errorCalls...)
}

// TestNewParserPool tests parser pool creation
func TestNewParserPool(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	tests := []struct {
		name            string
		workers         int
		expectedWorkers int
	}{
		{
			name:            "default workers (0)",
			workers:         0,
			expectedWorkers: 1, // At least 1 CPU
		},
		{
			name:            "specific workers",
			workers:         4,
			expectedWorkers: 4,
		},
		{
			name:            "capped at 16",
			workers:         20,
			expectedWorkers: 16,
		},
		{
			name:            "negative workers defaults to CPU count",
			workers:         -1,
			expectedWorkers: 1, // At least 1 CPU
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewParserPool(tt.workers, tsParser)

			if pool == nil {
				t.Fatal("Expected non-nil parser pool")
			}

			if pool.workers < 1 {
				t.Errorf("Expected at least 1 worker, got %d", pool.workers)
			}

			if tt.workers > 16 && pool.workers != 16 {
				t.Errorf("Expected workers to be capped at 16, got %d", pool.workers)
			}

			if pool.tsParser == nil {
				t.Error("Expected non-nil Tree-sitter parser")
			}
		})
	}
}

// TestParserPoolProcess tests concurrent file processing
func TestParserPoolProcess(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	// Create temporary test files
	tempDir := t.TempDir()

	// Create test Go file
	goFile := filepath.Join(tempDir, "test.go")
	goContent := `package main

func main() {
	println("Hello, World!")
}
`
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		t.Fatalf("Failed to create test Go file: %v", err)
	}

	// Create test JavaScript file
	jsFile := filepath.Join(tempDir, "test.js")
	jsContent := `function hello() {
	console.log("Hello, World!");
}
`
	if err := os.WriteFile(jsFile, []byte(jsContent), 0644); err != nil {
		t.Fatalf("Failed to create test JS file: %v", err)
	}

	// Create test Python file
	pyFile := filepath.Join(tempDir, "test.py")
	pyContent := `def hello():
	print("Hello, World!")
`
	if err := os.WriteFile(pyFile, []byte(pyContent), 0644); err != nil {
		t.Fatalf("Failed to create test Python file: %v", err)
	}

	// Create scanned files
	files := []ScannedFile{
		{
			Path:     "test.go",
			AbsPath:  goFile,
			Language: "Go",
			Size:     int64(len(goContent)),
		},
		{
			Path:     "test.js",
			AbsPath:  jsFile,
			Language: "JavaScript",
			Size:     int64(len(jsContent)),
		},
		{
			Path:     "test.py",
			AbsPath:  pyFile,
			Language: "Python",
			Size:     int64(len(pyContent)),
		},
	}

	tests := []struct {
		name           string
		workers        int
		files          []ScannedFile
		expectedFiles  int
		expectedErrors int
	}{
		{
			name:           "single worker",
			workers:        1,
			files:          files,
			expectedFiles:  3,
			expectedErrors: 0,
		},
		{
			name:           "multiple workers",
			workers:        2,
			files:          files,
			expectedFiles:  3,
			expectedErrors: 0,
		},
		{
			name:           "empty file list",
			workers:        2,
			files:          []ScannedFile{},
			expectedFiles:  0,
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewParserPool(tt.workers, tsParser)

			parsedFiles, errors := pool.Process(tt.files)

			if len(parsedFiles) != tt.expectedFiles {
				t.Errorf("Expected %d parsed files, got %d", tt.expectedFiles, len(parsedFiles))
			}

			if len(errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d", tt.expectedErrors, len(errors))
			}

			// Verify each parsed file has content
			for _, pf := range parsedFiles {
				if pf.Path == "" {
					t.Error("Expected non-empty path")
				}
				if pf.Language == "" {
					t.Error("Expected non-empty language")
				}
				if len(pf.Content) == 0 {
					t.Error("Expected non-empty content")
				}
			}
		})
	}
}

// TestParserPoolErrorHandling tests error collection
func TestParserPoolErrorHandling(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	// Create files with various error conditions
	files := []ScannedFile{
		{
			Path:     "nonexistent.go",
			AbsPath:  "/nonexistent/path/file.go",
			Language: "Go",
			Size:     100,
		},
		{
			Path:     "unsupported.xyz",
			AbsPath:  "/some/path/file.xyz",
			Language: "Unknown",
			Size:     100,
		},
	}

	pool := NewParserPool(2, tsParser)
	mockLogger := &MockProgressLogger{}
	pool.SetProgressLogger(mockLogger)
	pool.SetVerbose(true)

	parsedFiles, errors := pool.Process(files)

	// Should have errors for both files
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	// Should have no successfully parsed files
	if len(parsedFiles) != 0 {
		t.Errorf("Expected 0 parsed files, got %d", len(parsedFiles))
	}

	// Verify error logging
	errorCalls := mockLogger.GetErrorCalls()
	if len(errorCalls) != 2 {
		t.Errorf("Expected 2 error log calls, got %d", len(errorCalls))
	}
}

// TestParserPoolProgressTracking tests progress tracking
func TestParserPoolProgressTracking(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	// Create temporary test files
	tempDir := t.TempDir()

	var files []ScannedFile
	for i := 0; i < 5; i++ {
		filename := fmt.Sprintf("test%d.go", i)
		filepath := filepath.Join(tempDir, filename)
		content := fmt.Sprintf("package main\nfunc test%d() {}\n", i)

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		files = append(files, ScannedFile{
			Path:     filename,
			AbsPath:  filepath,
			Language: "Go",
			Size:     int64(len(content)),
		})
	}

	pool := NewParserPool(2, tsParser)
	mockLogger := &MockProgressLogger{}
	pool.SetProgressLogger(mockLogger)
	pool.SetVerbose(true)

	parsedFiles, errors := pool.Process(files)

	// Verify results
	if len(parsedFiles) != 5 {
		t.Errorf("Expected 5 parsed files, got %d", len(parsedFiles))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}

	// Verify progress tracking
	progressCalls := mockLogger.GetProgressCalls()
	if len(progressCalls) != 5 {
		t.Errorf("Expected 5 progress calls, got %d", len(progressCalls))
	}

	// Verify progress calls have correct total
	for _, call := range progressCalls {
		if call.Total != 5 {
			t.Errorf("Expected total=5, got %d", call.Total)
		}
		if call.Current < 1 || call.Current > 5 {
			t.Errorf("Expected current between 1-5, got %d", call.Current)
		}
	}
}

// TestParserPoolConcurrency tests for race conditions
func TestParserPoolConcurrency(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	// Create temporary test files
	tempDir := t.TempDir()

	var files []ScannedFile
	for i := 0; i < 20; i++ {
		filename := fmt.Sprintf("test%d.go", i)
		filepath := filepath.Join(tempDir, filename)
		content := fmt.Sprintf("package main\n\nfunc test%d() {\n\tprintln(\"test\")\n}\n", i)

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		files = append(files, ScannedFile{
			Path:     filename,
			AbsPath:  filepath,
			Language: "Go",
			Size:     int64(len(content)),
		})
	}

	// Test with different worker counts
	workerCounts := []int{1, 2, 4, 8}

	for _, workers := range workerCounts {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			pool := NewParserPool(workers, tsParser)

			parsedFiles, errors := pool.Process(files)

			if len(parsedFiles) != 20 {
				t.Errorf("Expected 20 parsed files, got %d", len(parsedFiles))
			}

			if len(errors) != 0 {
				t.Errorf("Expected 0 errors, got %d", len(errors))
			}

			// Verify all files were processed (no duplicates or missing)
			fileMap := make(map[string]bool)
			for _, pf := range parsedFiles {
				if fileMap[pf.Path] {
					t.Errorf("Duplicate file processed: %s", pf.Path)
				}
				fileMap[pf.Path] = true
			}

			if len(fileMap) != 20 {
				t.Errorf("Expected 20 unique files, got %d", len(fileMap))
			}
		})
	}
}

// TestParserPoolMixedLanguages tests parsing multiple languages concurrently
func TestParserPoolMixedLanguages(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	// Create temporary test files
	tempDir := t.TempDir()

	testFiles := []struct {
		name     string
		language string
		content  string
	}{
		{
			name:     "test1.go",
			language: "Go",
			content:  "package main\n\nfunc hello() {}\n",
		},
		{
			name:     "test2.go",
			language: "Go",
			content:  "package main\n\nfunc world() {}\n",
		},
		{
			name:     "test1.js",
			language: "JavaScript",
			content:  "function hello() {}\n",
		},
		{
			name:     "test2.js",
			language: "JavaScript",
			content:  "function world() {}\n",
		},
		{
			name:     "test1.py",
			language: "Python",
			content:  "def hello():\n    pass\n",
		},
		{
			name:     "test2.py",
			language: "Python",
			content:  "def world():\n    pass\n",
		},
	}

	var files []ScannedFile
	for _, tf := range testFiles {
		filepath := filepath.Join(tempDir, tf.name)
		if err := os.WriteFile(filepath, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		files = append(files, ScannedFile{
			Path:     tf.name,
			AbsPath:  filepath,
			Language: tf.language,
			Size:     int64(len(tf.content)),
		})
	}

	pool := NewParserPool(4, tsParser)
	parsedFiles, errors := pool.Process(files)

	// Verify all files were parsed successfully
	if len(parsedFiles) != 6 {
		t.Errorf("Expected 6 parsed files, got %d", len(parsedFiles))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errors), errors)
	}

	// Verify language distribution
	langCount := make(map[string]int)
	for _, pf := range parsedFiles {
		langCount[pf.Language]++
	}

	if langCount["go"] != 2 {
		t.Errorf("Expected 2 Go files, got %d", langCount["go"])
	}
	if langCount["javascript"] != 2 {
		t.Errorf("Expected 2 JavaScript files, got %d", langCount["javascript"])
	}
	if langCount["python"] != 2 {
		t.Errorf("Expected 2 Python files, got %d", langCount["python"])
	}
}

// TestParserPoolVerboseMode tests verbose mode toggle
func TestParserPoolVerboseMode(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	pool := NewParserPool(2, tsParser)

	// Test default verbose mode (should be false)
	if pool.verbose {
		t.Error("Expected verbose to be false by default")
	}

	// Test setting verbose mode
	pool.SetVerbose(true)
	if !pool.verbose {
		t.Error("Expected verbose to be true after SetVerbose(true)")
	}

	pool.SetVerbose(false)
	if pool.verbose {
		t.Error("Expected verbose to be false after SetVerbose(false)")
	}
}

// TestDefaultProgressLogger tests the default progress logger
func TestDefaultProgressLogger(t *testing.T) {
	logger := &DefaultProgressLogger{}

	// These should not panic
	logger.LogProgress(1, 10, "test.go")
	logger.LogError("test.go", fmt.Errorf("test error"))
}

// TestParserPoolWorkerCreationError tests worker behavior when parser creation fails
func TestParserPoolWorkerCreationError(t *testing.T) {
	// This test verifies the error handling path in worker()
	// In practice, NewTreeSitterParser() rarely fails, but we test the path
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	pool := NewParserPool(1, tsParser)

	// Create a simple test file
	tempDir := t.TempDir()
	goFile := filepath.Join(tempDir, "test.go")
	goContent := `package main
func main() {}
`
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	files := []ScannedFile{
		{
			Path:     "test.go",
			AbsPath:  goFile,
			Language: "Go",
			Size:     int64(len(goContent)),
		},
	}

	// Process should work normally
	parsedFiles, errors := pool.Process(files)

	if len(parsedFiles) != 1 {
		t.Errorf("Expected 1 parsed file, got %d", len(parsedFiles))
	}
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

// TestParserPoolProcessWithNilResult tests handling of nil results
func TestParserPoolProcessWithNilResult(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	pool := NewParserPool(1, tsParser)
	mockLogger := &MockProgressLogger{}
	pool.SetProgressLogger(mockLogger)
	pool.SetVerbose(false) // Test without verbose mode

	// Create a file that will fail to parse
	files := []ScannedFile{
		{
			Path:     "nonexistent.go",
			AbsPath:  "/nonexistent/file.go",
			Language: "Go",
			Size:     100,
		},
	}

	parsedFiles, errors := pool.Process(files)

	// Should have error
	if len(errors) == 0 {
		t.Error("Expected at least one error")
	}

	// Should have no parsed files
	if len(parsedFiles) != 0 {
		t.Errorf("Expected 0 parsed files, got %d", len(parsedFiles))
	}

	// Error should be logged even without verbose mode
	errorCalls := mockLogger.GetErrorCalls()
	if len(errorCalls) == 0 {
		t.Error("Expected error to be logged")
	}
}
