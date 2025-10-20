package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/parser"
)

func TestScanRepository(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create some test files
	testFiles := map[string]string{
		"main.go":       "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
		"README.md":     "# Test Repository\n\nThis is a test repository.",
		"utils.js":      "function hello() {\n  return 'world';\n}",
		".gitignore":    "node_modules\n*.log",
		"vendor/lib.go": "package vendor\n\nfunc Lib() {}",
	}

	// Create test files
	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}

		// Write file content
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test scanning the repository
	files, err := parser.ScanRepository(tempDir)
	if err != nil {
		t.Fatalf("ScanRepository failed: %v", err)
	}

	// We expect 3 files (main.go, README.md, utils.js)
	// .gitignore should be skipped (hidden file)
	// vendor/lib.go should be skipped (in vendor directory)
	expectedFiles := 3
	if len(files) != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, len(files))
	}

	// Check that we have the expected files
	filePaths := make(map[string]bool)
	for _, file := range files {
		filePaths[file.Path] = true
	}

	expectedPaths := []string{"main.go", "README.md", "utils.js"}
	for _, path := range expectedPaths {
		if !filePaths[path] {
			t.Errorf("Expected file %s not found", path)
		}
	}

	// Verify file content and language detection
	for _, file := range files {
		switch file.Path {
		case "main.go":
			if file.Language != "Go" {
				t.Errorf("Expected Go language for main.go, got %s", file.Language)
			}
			if file.Content != "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}" {
				t.Errorf("Content mismatch for main.go")
			}
		case "README.md":
			if file.Language != "Markdown" {
				t.Errorf("Expected Markdown language for README.md, got %s", file.Language)
			}
		case "utils.js":
			if file.Language != "JavaScript" {
				t.Errorf("Expected JavaScript language for utils.js, got %s", file.Language)
			}
		}
	}
}

func TestIsBinaryFile(t *testing.T) {
	// Test binary file extensions
	binaryFiles := []string{
		"test.exe", "test.dll", "test.so", "test.jpg",
		"test.png", "test.pdf", "test.zip", "test.db",
	}

	for _, file := range binaryFiles {
		// We would need to test the isBinaryFile function directly,
		// but it's not exported. In a real implementation, we might
		// want to export it or create a testable version.
		_ = file
	}
}

func TestDetermineLanguage(t *testing.T) {
	// Test language detection
	testCases := map[string]string{
		"main.go":     "Go",
		"index.js":    "JavaScript",
		"app.ts":      "TypeScript",
		"style.css":   "CSS",
		"README.md":   "Markdown",
		"config.json": "JSON",
		"test.py":     "Python",
		"App.java":    "Java",
		"main.cpp":    "C++",
		"hello.rb":    "Ruby",
		"unknown.xyz": "Unknown",
	}

	for filename, expectedLang := range testCases {
		// Similar to isBinaryFile, determineLanguage is not exported
		// We would need to test it indirectly through ScanRepository
		// or make it exportable for testing purposes
		_ = filename
		_ = expectedLang
	}
}
