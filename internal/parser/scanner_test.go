package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileScanner(t *testing.T) {
	tempDir := t.TempDir()
	filter, err := NewIgnoreFilter(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	scanner := NewFileScanner(tempDir, filter)
	if scanner == nil {
		t.Fatal("NewFileScanner returned nil")
	}

	if scanner.rootPath != tempDir {
		t.Errorf("Expected rootPath %s, got %s", tempDir, scanner.rootPath)
	}

	if scanner.maxSize != 1024*1024 {
		t.Errorf("Expected default maxSize 1MB, got %d", scanner.maxSize)
	}
}

func TestFileScanner_SetMaxSize(t *testing.T) {
	tempDir := t.TempDir()
	scanner := NewFileScanner(tempDir, nil)

	newSize := int64(2048)
	scanner.SetMaxSize(newSize)

	if scanner.maxSize != newSize {
		t.Errorf("Expected maxSize %d, got %d", newSize, scanner.maxSize)
	}
}

func TestFileScanner_SetLanguageFilter(t *testing.T) {
	tempDir := t.TempDir()
	scanner := NewFileScanner(tempDir, nil)

	languages := []string{"Go", "JavaScript"}
	scanner.SetLanguageFilter(languages)

	if len(scanner.languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(scanner.languages))
	}
}

func TestFileScanner_Scan_BasicDiscovery(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"main.go":       "package main",
		"utils.js":      "function test() {}",
		"app.py":        "def main(): pass",
		"README.md":     "# Test",
		"config.json":   "{}",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Create scanner without ignore filter
	scanner := NewFileScanner(tempDir, nil)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find all files
	if len(files) != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), len(files))
	}

	// Verify file paths and languages
	fileMap := make(map[string]ScannedFile)
	for _, file := range files {
		fileMap[file.Path] = file
	}

	expectedLanguages := map[string]string{
		"main.go":     "Go",
		"utils.js":    "JavaScript",
		"app.py":      "Python",
		"README.md":   "Markdown",
		"config.json": "JSON",
	}

	for path, expectedLang := range expectedLanguages {
		file, exists := fileMap[path]
		if !exists {
			t.Errorf("Expected file %s not found", path)
			continue
		}
		if file.Language != expectedLang {
			t.Errorf("File %s: expected language %s, got %s", path, expectedLang, file.Language)
		}
	}
}

func TestFileScanner_Scan_LanguageDetection(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		filename string
		expected string
	}{
		{"test.go", "Go"},
		{"test.js", "JavaScript"},
		{"test.jsx", "JavaScript"},
		{"test.ts", "TypeScript"},
		{"test.tsx", "TypeScript"},
		{"test.py", "Python"},
		{"test.java", "Java"},
		{"test.cpp", "C++"},
		{"test.c", "C"},
		{"test.rs", "Rust"},
		{"test.php", "PHP"},
		{"test.rb", "Ruby"},
		{"test.html", "HTML"},
		{"test.css", "CSS"},
		{"test.md", "Markdown"},
	}

	for _, tc := range testCases {
		fullPath := filepath.Join(tempDir, tc.filename)
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", tc.filename, err)
		}
	}

	scanner := NewFileScanner(tempDir, nil)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	fileMap := make(map[string]ScannedFile)
	for _, file := range files {
		fileMap[file.Path] = file
	}

	for _, tc := range testCases {
		file, exists := fileMap[tc.filename]
		if !exists {
			t.Errorf("File %s not found", tc.filename)
			continue
		}
		if file.Language != tc.expected {
			t.Errorf("File %s: expected language %s, got %s", tc.filename, tc.expected, file.Language)
		}
	}
}

func TestFileScanner_Scan_BinaryFileFiltering(t *testing.T) {
	tempDir := t.TempDir()

	// Create binary files
	binaryFiles := []string{
		"test.exe", "test.dll", "test.so", "test.jpg",
		"test.png", "test.pdf", "test.zip", "test.db",
	}

	for _, filename := range binaryFiles {
		fullPath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(fullPath, []byte("binary content"), 0644); err != nil {
			t.Fatalf("Failed to create binary file %s: %v", filename, err)
		}
	}

	// Create a regular source file
	sourceFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(sourceFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	scanner := NewFileScanner(tempDir, nil)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should only find the source file, not binary files
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	if len(files) > 0 && files[0].Path != "main.go" {
		t.Errorf("Expected main.go, got %s", files[0].Path)
	}
}

func TestFileScanner_Scan_SizeLimitEnforcement(t *testing.T) {
	tempDir := t.TempDir()

	// Create a small file
	smallFile := filepath.Join(tempDir, "small.go")
	if err := os.WriteFile(smallFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create small file: %v", err)
	}

	// Create a large file
	largeContent := make([]byte, 2048)
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	largeFile := filepath.Join(tempDir, "large.go")
	if err := os.WriteFile(largeFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Set max size to 1KB
	scanner := NewFileScanner(tempDir, nil)
	scanner.SetMaxSize(1024)

	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should only find the small file
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	if len(files) > 0 && files[0].Path != "small.go" {
		t.Errorf("Expected small.go, got %s", files[0].Path)
	}
}

func TestFileScanner_Scan_IgnoreFilterIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Create directory structure
	dirs := []string{
		"src",
		"node_modules",
		"vendor",
		".git",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create test files
	testFiles := map[string]string{
		"main.go":                "package main",
		"src/utils.go":           "package src",
		"node_modules/lib.js":    "module.exports = {}",
		"vendor/dep.go":          "package vendor",
		".git/config":            "config",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Create ignore filter with default patterns
	filter, err := NewIgnoreFilter(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	scanner := NewFileScanner(tempDir, filter)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should only find main.go and src/utils.go
	// node_modules, vendor, and .git should be ignored
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
		for _, f := range files {
			t.Logf("Found file: %s", f.Path)
		}
	}

	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file.Path] = true
	}

	expectedFiles := []string{"main.go", filepath.Join("src", "utils.go")}
	for _, expected := range expectedFiles {
		if !fileMap[expected] {
			t.Errorf("Expected file %s not found", expected)
		}
	}

	// Verify ignored files are not present
	ignoredFiles := []string{
		filepath.Join("node_modules", "lib.js"),
		filepath.Join("vendor", "dep.go"),
		filepath.Join(".git", "config"),
	}
	for _, ignored := range ignoredFiles {
		if fileMap[ignored] {
			t.Errorf("File %s should have been ignored", ignored)
		}
	}
}

func TestFileScanner_Scan_CustomIgnorePatterns(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"main.go":       "package main",
		"main_test.go":  "package main",
		"utils.go":      "package main",
		"utils_test.go": "package main",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Create ignore filter with custom pattern to ignore test files
	filter, err := NewIgnoreFilter(nil, []string{"*_test.go"})
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	scanner := NewFileScanner(tempDir, filter)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should only find main.go and utils.go (test files ignored)
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file.Path] = true
	}

	if !fileMap["main.go"] || !fileMap["utils.go"] {
		t.Error("Expected main.go and utils.go to be found")
	}

	if fileMap["main_test.go"] || fileMap["utils_test.go"] {
		t.Error("Test files should have been ignored")
	}
}

func TestFileScanner_Scan_LanguageFilter(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files in different languages
	testFiles := map[string]string{
		"main.go":    "package main",
		"utils.js":   "function test() {}",
		"app.py":     "def main(): pass",
		"README.md":  "# Test",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Create scanner with language filter for Go only
	scanner := NewFileScanner(tempDir, nil)
	scanner.SetLanguageFilter([]string{"Go"})

	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should only find Go files
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	if len(files) > 0 && files[0].Path != "main.go" {
		t.Errorf("Expected main.go, got %s", files[0].Path)
	}
}

func TestFileScanner_Scan_MultipleLanguageFilter(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files in different languages
	testFiles := map[string]string{
		"main.go":    "package main",
		"utils.js":   "function test() {}",
		"app.py":     "def main(): pass",
		"README.md":  "# Test",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Create scanner with language filter for Go and JavaScript
	scanner := NewFileScanner(tempDir, nil)
	scanner.SetLanguageFilter([]string{"Go", "JavaScript"})

	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find Go and JavaScript files
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file.Path] = true
	}

	if !fileMap["main.go"] || !fileMap["utils.js"] {
		t.Error("Expected main.go and utils.js to be found")
	}
}

func TestFileScanner_Scan_NonExistentPath(t *testing.T) {
	scanner := NewFileScanner("/nonexistent/path", nil)
	_, err := scanner.Scan()
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestFileScanner_Scan_NestedDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create nested directory structure
	nestedFiles := map[string]string{
		"main.go":              "package main",
		"pkg/utils/helper.go":  "package utils",
		"cmd/app/main.go":      "package main",
		"internal/core/api.go": "package core",
	}

	for path, content := range nestedFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	scanner := NewFileScanner(tempDir, nil)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find all nested files
	if len(files) != len(nestedFiles) {
		t.Errorf("Expected %d files, got %d", len(nestedFiles), len(files))
	}

	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file.Path] = true
	}

	for expectedPath := range nestedFiles {
		if !fileMap[expectedPath] {
			t.Errorf("Expected file %s not found", expectedPath)
		}
	}
}

func TestFileScanner_Scan_AbsolutePath(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewFileScanner(tempDir, nil)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	// Verify absolute path is set
	if files[0].AbsPath == "" {
		t.Error("AbsPath should not be empty")
	}

	// Verify relative path is correct
	if files[0].Path != "main.go" {
		t.Errorf("Expected relative path 'main.go', got %s", files[0].Path)
	}
}

func TestFileScanner_Scan_FileSize(t *testing.T) {
	tempDir := t.TempDir()

	content := "package main\n\nfunc main() {}"
	testFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewFileScanner(tempDir, nil)
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	expectedSize := int64(len(content))
	if files[0].Size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, files[0].Size)
	}
}

func TestScanRepository(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	goFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatalf("Failed to create Go file: %v", err)
	}

	jsFile := filepath.Join(tempDir, "app.js")
	if err := os.WriteFile(jsFile, []byte("console.log('hello');"), 0644); err != nil {
		t.Fatalf("Failed to create JS file: %v", err)
	}

	// Create a subdirectory with a file
	subDir := filepath.Join(tempDir, "src")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	pyFile := filepath.Join(subDir, "test.py")
	if err := os.WriteFile(pyFile, []byte("print('hello')"), 0644); err != nil {
		t.Fatalf("Failed to create Python file: %v", err)
	}

	// Create a hidden file (should be skipped)
	hiddenFile := filepath.Join(tempDir, ".hidden")
	if err := os.WriteFile(hiddenFile, []byte("hidden"), 0644); err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}

	// Create a binary file (should be skipped)
	binaryFile := filepath.Join(tempDir, "image.png")
	if err := os.WriteFile(binaryFile, []byte("binary data"), 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	files, err := ScanRepository(tempDir)
	if err != nil {
		t.Fatalf("ScanRepository failed: %v", err)
	}

	// Should have 3 files (main.go, app.js, test.py)
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	// Verify file contents
	foundGo := false
	foundJS := false
	foundPy := false
	for _, file := range files {
		if file.Path == "main.go" {
			foundGo = true
			if file.Language != "Go" {
				t.Errorf("Expected language 'Go', got '%s'", file.Language)
			}
		}
		if file.Path == "app.js" {
			foundJS = true
			if file.Language != "JavaScript" {
				t.Errorf("Expected language 'JavaScript', got '%s'", file.Language)
			}
		}
		if filepath.Base(file.Path) == "test.py" {
			foundPy = true
			if file.Language != "Python" {
				t.Errorf("Expected language 'Python', got '%s'", file.Language)
			}
		}
	}

	if !foundGo {
		t.Error("main.go not found in results")
	}
	if !foundJS {
		t.Error("app.js not found in results")
	}
	if !foundPy {
		t.Error("test.py not found in results")
	}
}

func TestScanRepository_SkipDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create node_modules directory (should be skipped)
	nodeModules := filepath.Join(tempDir, "node_modules")
	if err := os.Mkdir(nodeModules, 0755); err != nil {
		t.Fatalf("Failed to create node_modules: %v", err)
	}
	nodeFile := filepath.Join(nodeModules, "package.js")
	if err := os.WriteFile(nodeFile, []byte("module.exports = {}"), 0644); err != nil {
		t.Fatalf("Failed to create file in node_modules: %v", err)
	}

	// Create vendor directory (should be skipped)
	vendor := filepath.Join(tempDir, "vendor")
	if err := os.Mkdir(vendor, 0755); err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}
	vendorFile := filepath.Join(vendor, "lib.go")
	if err := os.WriteFile(vendorFile, []byte("package lib"), 0644); err != nil {
		t.Fatalf("Failed to create file in vendor: %v", err)
	}

	// Create .git directory (should be skipped)
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git: %v", err)
	}
	gitFile := filepath.Join(gitDir, "config")
	if err := os.WriteFile(gitFile, []byte("config"), 0644); err != nil {
		t.Fatalf("Failed to create file in .git: %v", err)
	}

	// Create a regular file
	mainFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	files, err := ScanRepository(tempDir)
	if err != nil {
		t.Fatalf("ScanRepository failed: %v", err)
	}

	// Should only have main.go
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	if len(files) > 0 && files[0].Path != "main.go" {
		t.Errorf("Expected main.go, got %s", files[0].Path)
	}
}

func TestScanRepository_NonExistentPath(t *testing.T) {
	_, err := ScanRepository("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestIsLargeFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a small file
	smallFile := filepath.Join(tempDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte("small content"), 0644); err != nil {
		t.Fatalf("Failed to create small file: %v", err)
	}

	// Create a large file (> 1MB)
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := make([]byte, 2*1024*1024) // 2MB
	if err := os.WriteFile(largeFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Test small file
	if isLargeFile(smallFile) {
		t.Error("Small file should not be considered large")
	}

	// Test large file
	if !isLargeFile(largeFile) {
		t.Error("Large file should be considered large")
	}

	// Test non-existent file
	if !isLargeFile("/nonexistent/file.txt") {
		t.Error("Non-existent file should be considered large (error case)")
	}
}

func TestScanRepository_LargeFileSkipping(t *testing.T) {
	tempDir := t.TempDir()

	// Create a normal file
	normalFile := filepath.Join(tempDir, "normal.go")
	if err := os.WriteFile(normalFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}

	// Create a large file (> 1MB)
	largeFile := filepath.Join(tempDir, "large.go")
	largeContent := make([]byte, 2*1024*1024) // 2MB
	if err := os.WriteFile(largeFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	files, err := ScanRepository(tempDir)
	if err != nil {
		t.Fatalf("ScanRepository failed: %v", err)
	}

	// Should only have normal.go (large.go should be skipped)
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	if len(files) > 0 && files[0].Path != "normal.go" {
		t.Errorf("Expected normal.go, got %s", files[0].Path)
	}
}

func TestScanRepository_ReadError(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make the file unreadable (this may not work on all systems)
	if err := os.Chmod(testFile, 0000); err != nil {
		t.Skipf("Cannot change file permissions: %v", err)
	}
	defer os.Chmod(testFile, 0644) // Restore permissions for cleanup

	_, err := ScanRepository(tempDir)
	// On some systems this may succeed, on others it may fail
	// We just verify the function handles it gracefully
	if err != nil {
		// Error is expected and acceptable
		t.Logf("Got expected error: %v", err)
	}
}
