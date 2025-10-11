package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewIgnoreFilter_DefaultPatterns(t *testing.T) {
	filter, err := NewIgnoreFilter(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	// Test default ignore patterns
	testCases := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{".git/config", false, true},
		{".git", true, true},
		{"node_modules/package.json", false, true},
		{"node_modules", true, true},
		{"vendor/lib.go", false, true},
		{"__pycache__/cache.pyc", false, true},
		{"file.pyc", false, true},
		{"file.exe", false, true},
		{"file.dll", false, true},
		{"file.so", false, true},
		{"image.jpg", false, true},
		{"image.png", false, true},
		{"doc.pdf", false, true},
		{"main.go", false, false},
		{"src/app.js", false, false},
	}

	for _, tc := range testCases {
		result := filter.ShouldIgnore(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("ShouldIgnore(%q, %v) = %v, expected %v", tc.path, tc.isDir, result, tc.expected)
		}
	}
}

func TestNewIgnoreFilter_GitignoreFile(t *testing.T) {
	// Create temporary directory with .gitignore
	tmpDir := t.TempDir()
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	
	gitignoreContent := `# Comment
*.log
build/
!important.log
temp*.txt
`
	err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	filter, err := NewIgnoreFilter([]string{gitignorePath}, nil)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"debug.log", false, true},
		{"app/error.log", false, true},
		{"build/output.txt", false, true},
		{"build", true, true},
		{"important.log", false, false}, // Negation pattern
		{"temp1.txt", false, true},
		{"temp_file.txt", false, true},
		{"main.go", false, false},
	}

	for _, tc := range testCases {
		result := filter.ShouldIgnore(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("ShouldIgnore(%q, %v) = %v, expected %v", tc.path, tc.isDir, result, tc.expected)
		}
	}
}

func TestNewIgnoreFilter_CustomPatterns(t *testing.T) {
	customPatterns := []string{
		"*.tmp",
		"cache/",
		"test_*",
	}

	filter, err := NewIgnoreFilter(nil, customPatterns)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"file.tmp", false, true},
		{"cache/data.txt", false, true},
		{"cache", true, true},
		{"test_file.go", false, true},
		{"main.go", false, false},
	}

	for _, tc := range testCases {
		result := filter.ShouldIgnore(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("ShouldIgnore(%q, %v) = %v, expected %v", tc.path, tc.isDir, result, tc.expected)
		}
	}
}

func TestNewIgnoreFilter_NegationPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	
	gitignoreContent := `*.log
!important.log
!critical/*.log
`
	err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	filter, err := NewIgnoreFilter([]string{gitignorePath}, nil)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"debug.log", false, true},
		{"important.log", false, false}, // Negated
		{"critical/error.log", false, false}, // Negated
		{"other/error.log", false, true},
	}

	for _, tc := range testCases {
		result := filter.ShouldIgnore(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("ShouldIgnore(%q, %v) = %v, expected %v", tc.path, tc.isDir, result, tc.expected)
		}
	}
}

func TestNewIgnoreFilter_DirectorySpecificRules(t *testing.T) {
	tmpDir := t.TempDir()
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	
	gitignoreContent := `logs/
*.log
`
	err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	filter, err := NewIgnoreFilter([]string{gitignorePath}, nil)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"logs", true, true},
		{"logs/debug.log", false, true},
		{"app/logs", true, true},
		{"debug.log", false, true},
		{"src/debug.log", false, true},
	}

	for _, tc := range testCases {
		result := filter.ShouldIgnore(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("ShouldIgnore(%q, %v) = %v, expected %v", tc.path, tc.isDir, result, tc.expected)
		}
	}
}

func TestNewIgnoreFilter_NestedGitignore(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	
	// Root .gitignore
	rootGitignore := filepath.Join(tmpDir, ".gitignore")
	rootContent := `*.log
build/
`
	err := os.WriteFile(rootGitignore, []byte(rootContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create root .gitignore: %v", err)
	}

	// Nested .gitignore in subdir
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	
	subGitignore := filepath.Join(subDir, ".gitignore")
	subContent := `!important.log
*.tmp
`
	err = os.WriteFile(subGitignore, []byte(subContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create sub .gitignore: %v", err)
	}

	filter, err := NewIgnoreFilter([]string{rootGitignore, subGitignore}, nil)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"debug.log", false, true},
		{"build/output.txt", false, true},
		{"subdir/important.log", false, false}, // Negated in subdir
		{"subdir/file.tmp", false, true},
		{"file.tmp", false, true}, // Also ignored (*.tmp pattern applies globally)
	}

	for _, tc := range testCases {
		result := filter.ShouldIgnore(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("ShouldIgnore(%q, %v) = %v, expected %v", tc.path, tc.isDir, result, tc.expected)
		}
	}
}

func TestNewIgnoreFilter_Precedence(t *testing.T) {
	tmpDir := t.TempDir()
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	
	gitignoreContent := `*.log
!important.log
`
	err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Custom patterns should be applied after .gitignore
	customPatterns := []string{"important.log"}

	filter, err := NewIgnoreFilter([]string{gitignorePath}, customPatterns)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	// Custom pattern should override negation from .gitignore
	result := filter.ShouldIgnore("important.log", false)
	if !result {
		t.Errorf("Expected important.log to be ignored by custom pattern, but it wasn't")
	}
}

func TestNewIgnoreFilter_EmptyPatterns(t *testing.T) {
	filter, err := NewIgnoreFilter([]string{}, []string{})
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	// Should still have default patterns
	if !filter.ShouldIgnore(".git", true) {
		t.Error("Expected .git to be ignored by default patterns")
	}
}

func TestNewIgnoreFilter_InvalidGitignoreFile(t *testing.T) {
	_, err := NewIgnoreFilter([]string{"/nonexistent/path/.gitignore"}, nil)
	if err == nil {
		t.Error("Expected error for nonexistent .gitignore file")
	}
}

func TestShouldIgnore_WildcardPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	
	gitignoreContent := `*.log
test_*.go
**/temp/**
`
	err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	filter, err := NewIgnoreFilter([]string{gitignorePath}, nil)
	if err != nil {
		t.Fatalf("Failed to create ignore filter: %v", err)
	}

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"app.log", false, true},
		{"src/debug.log", false, true},
		{"test_main.go", false, true},
		{"src/test_helper.go", false, true},
		{"temp/file.txt", false, true},
		{"src/temp/data.json", false, true},
		{"main.go", false, false},
	}

	for _, tc := range testCases {
		result := filter.ShouldIgnore(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("ShouldIgnore(%q, %v) = %v, expected %v", tc.path, tc.isDir, result, tc.expected)
		}
	}
}
