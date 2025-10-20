package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IgnoreRule represents a single ignore pattern rule
type IgnoreRule struct {
	pattern  string
	isNegate bool
	isDir    bool
}

// IgnoreFilter manages ignore patterns from .gitignore files and custom patterns
type IgnoreFilter struct {
	rules []IgnoreRule
}

// defaultIgnorePatterns returns the default patterns that should always be ignored
func defaultIgnorePatterns() []string {
	return []string{
		".git/",
		"node_modules/",
		"vendor/",
		"__pycache__/",
		"*.pyc",
		"*.exe",
		"*.dll",
		"*.so",
		"*.jpg",
		"*.jpeg",
		"*.png",
		"*.gif",
		"*.bmp",
		"*.pdf",
		"*.zip",
		"*.tar",
		"*.gz",
		"*.rar",
		"*.7z",
		"*.bin",
		"*.dat",
		"*.o",
		"*.a",
		"*.lib",
		"*.dylib",
	}
}

// NewIgnoreFilter creates a new IgnoreFilter with patterns from .gitignore files and custom patterns
// gitignorePaths: paths to .gitignore files to parse
// customPatterns: additional patterns to apply
func NewIgnoreFilter(gitignorePaths []string, customPatterns []string) (*IgnoreFilter, error) {
	filter := &IgnoreFilter{
		rules: make([]IgnoreRule, 0),
	}

	// Add default ignore patterns first
	for _, pattern := range defaultIgnorePatterns() {
		filter.addPattern(pattern)
	}

	// Parse .gitignore files
	for _, path := range gitignorePaths {
		if err := filter.parseGitignoreFile(path); err != nil {
			return nil, fmt.Errorf("failed to parse .gitignore file %s: %w", path, err)
		}
	}

	// Add custom patterns
	for _, pattern := range customPatterns {
		filter.addPattern(pattern)
	}

	return filter, nil
}

// parseGitignoreFile reads and parses a .gitignore file
func (f *IgnoreFilter) parseGitignoreFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		f.addPattern(line)
	}

	return scanner.Err()
}

// addPattern adds a pattern to the filter rules
func (f *IgnoreFilter) addPattern(pattern string) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return
	}

	rule := IgnoreRule{
		pattern:  pattern,
		isNegate: false,
		isDir:    false,
	}

	// Check for negation pattern
	if strings.HasPrefix(pattern, "!") {
		rule.isNegate = true
		rule.pattern = strings.TrimPrefix(pattern, "!")
	}

	// Check if pattern is directory-specific
	if strings.HasSuffix(rule.pattern, "/") {
		rule.isDir = true
		rule.pattern = strings.TrimSuffix(rule.pattern, "/")
	}

	f.rules = append(f.rules, rule)
}

// ShouldIgnore checks if a path should be ignored based on the filter rules
// path: the file or directory path to check (relative path)
// isDir: whether the path is a directory
func (f *IgnoreFilter) ShouldIgnore(path string, isDir bool) bool {
	// Normalize path separators
	path = filepath.ToSlash(path)

	ignored := false

	// Apply rules in order (later rules can override earlier ones)
	for _, rule := range f.rules {
		// If rule is directory-specific and path is not a directory, skip
		if rule.isDir && !isDir {
			// But check if any parent directory matches
			if !f.matchesPattern(path, rule.pattern, true) {
				continue
			}
		}

		if f.matchesPattern(path, rule.pattern, isDir) {
			if rule.isNegate {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

// matchesPattern checks if a path matches a gitignore pattern
func (f *IgnoreFilter) matchesPattern(path, pattern string, isDir bool) bool {
	// Normalize separators
	path = filepath.ToSlash(path)
	pattern = filepath.ToSlash(pattern)

	// Handle ** (match any number of directories)
	if strings.Contains(pattern, "**") {
		return f.matchDoubleStarPattern(path, pattern)
	}

	// Handle absolute patterns (starting with /)
	if strings.HasPrefix(pattern, "/") {
		pattern = strings.TrimPrefix(pattern, "/")
		matched, _ := filepath.Match(pattern, path)
		return matched
	}

	// Try matching against the full path
	matched, _ := filepath.Match(pattern, path)
	if matched {
		return true
	}

	// Try matching against the basename
	basename := filepath.Base(path)
	matched, _ = filepath.Match(pattern, basename)
	if matched {
		return true
	}

	// Try matching against any path component
	parts := strings.Split(path, "/")
	for i := range parts {
		subpath := strings.Join(parts[i:], "/")
		matched, _ := filepath.Match(pattern, subpath)
		if matched {
			return true
		}
	}

	// Check if any parent directory matches (for directory patterns)
	if strings.Contains(path, "/") {
		dir := filepath.Dir(path)
		for dir != "." && dir != "/" {
			dirBase := filepath.Base(dir)
			matched, _ := filepath.Match(pattern, dirBase)
			if matched {
				return true
			}

			// Also check full directory path
			matched, _ = filepath.Match(pattern, dir)
			if matched {
				return true
			}

			dir = filepath.Dir(dir)
		}
	}

	return false
}

// matchDoubleStarPattern handles patterns with ** (match any number of directories)
func (f *IgnoreFilter) matchDoubleStarPattern(path, pattern string) bool {
	// Split pattern by **
	parts := strings.Split(pattern, "**")

	if len(parts) == 1 {
		// No ** in pattern, use regular matching
		matched, _ := filepath.Match(pattern, path)
		return matched
	}

	// For patterns like **/temp/**, we need to check if "temp" appears as a directory component
	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[len(parts)-1], "/")

	// If pattern is just "**", it matches everything
	if len(parts) == 2 && prefix == "" && suffix == "" {
		return true
	}

	pathParts := strings.Split(path, "/")

	// Check if path starts with the prefix (if any)
	if prefix != "" {
		if !strings.HasPrefix(path, prefix) {
			return false
		}
	}

	// For middle parts (between **), check if they appear in the path
	// This is the key part - for **/temp/**, the middle part is "/temp/"
	for i := 1; i < len(parts)-1; i++ {
		middle := strings.Trim(parts[i], "/")
		if middle == "" {
			continue
		}

		// Check if this middle part appears as a path component
		found := false
		for _, part := range pathParts {
			if part == middle {
				found = true
				break
			}
			// Also try pattern matching for wildcards in middle part
			matched, _ := filepath.Match(middle, part)
			if matched {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check if path ends with the suffix (if any)
	if suffix != "" {
		// For patterns like **/temp/**, suffix would be empty after the last **
		// We need to check if any directory component matches
		if strings.Contains(suffix, "/") {
			// Complex suffix with path components
			if !strings.Contains(path, suffix) {
				return false
			}
		} else {
			// Simple suffix - check if it matches any path component or the end
			found := false
			for _, part := range pathParts {
				if part == suffix {
					found = true
					break
				}
				matched, _ := filepath.Match(suffix, part)
				if matched {
					found = true
					break
				}
			}
			if !found && !strings.HasSuffix(path, suffix) {
				matched, _ := filepath.Match(suffix, filepath.Base(path))
				if !matched {
					return false
				}
			}
		}
	}

	return true
}
