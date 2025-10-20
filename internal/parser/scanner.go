package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo represents information about a file
type FileInfo struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Size     int64  `json:"size"`
}

// ScannedFile represents a discovered file with metadata
type ScannedFile struct {
	Path     string // Relative path from root
	AbsPath  string // Absolute path
	Language string // Detected language
	Size     int64  // File size in bytes
}

// FileScanner scans directories for source files with ignore filter support
type FileScanner struct {
	rootPath  string
	filter    *IgnoreFilter
	maxSize   int64    // Maximum file size in bytes (0 = no limit)
	languages []string // Language filter (empty = all languages)
}

// NewFileScanner creates a new FileScanner
func NewFileScanner(rootPath string, filter *IgnoreFilter) *FileScanner {
	return &FileScanner{
		rootPath: rootPath,
		filter:   filter,
		maxSize:  1024 * 1024, // Default 1MB
	}
}

// SetMaxSize sets the maximum file size to scan
func (s *FileScanner) SetMaxSize(size int64) {
	s.maxSize = size
}

// SetLanguageFilter sets the language filter
func (s *FileScanner) SetLanguageFilter(languages []string) {
	s.languages = languages
}

// Scan walks the directory tree and returns all matching files
func (s *FileScanner) Scan() ([]ScannedFile, error) {
	var files []ScannedFile

	// Ensure root path exists
	if _, err := os.Stat(s.rootPath); err != nil {
		return nil, fmt.Errorf("root path does not exist: %w", err)
	}

	err := filepath.WalkDir(s.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Log but continue with other files
			fmt.Fprintf(os.Stderr, "Warning: error accessing %s: %v\n", path, err)
			return nil
		}

		// Get relative path for ignore filter
		relPath, err := filepath.Rel(s.rootPath, path)
		if err != nil {
			// Log but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to get relative path for %s: %v\n", path, err)
			return nil
		}

		// Apply ignore filter at directory level for efficiency
		if d.IsDir() {
			if s.filter != nil && s.filter.ShouldIgnore(relPath, true) {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply ignore filter for files
		if s.filter != nil && s.filter.ShouldIgnore(relPath, false) {
			return nil
		}

		// Skip binary files
		if isBinaryFile(path) {
			return nil
		}

		// Get file info for size check
		info, err := d.Info()
		if err != nil {
			return nil // Skip files we can't stat
		}

		// Skip files exceeding size limit
		if s.maxSize > 0 && info.Size() > s.maxSize {
			return nil
		}

		// Detect language
		language := determineLanguage(path)

		// Apply language filter if specified
		if len(s.languages) > 0 {
			matched := false
			for _, lang := range s.languages {
				if strings.EqualFold(language, lang) {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		// Skip unknown languages unless no filter is set
		if language == "Unknown" && len(s.languages) == 0 {
			return nil
		}

		// Create scanned file entry
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		scannedFile := ScannedFile{
			Path:     relPath,
			AbsPath:  absPath,
			Language: language,
			Size:     info.Size(),
		}

		files = append(files, scannedFile)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	return files, nil
}

// ScanRepository scans a repository and returns file information
func ScanRepository(repoPath string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			// Skip hidden directories and common build directories
			if strings.HasPrefix(d.Name(), ".") ||
				d.Name() == "node_modules" ||
				d.Name() == "vendor" ||
				d.Name() == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		// Skip binary files and large files
		if isBinaryFile(path) || isLargeFile(path) {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Get relative path from repository root
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		// Determine language based on file extension
		language := determineLanguage(path)

		// Create file info
		fileInfo := FileInfo{
			Path:     relPath,
			Content:  string(content),
			Language: language,
			Size:     int64(len(content)),
		}

		files = append(files, fileInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan repository: %w", err)
	}

	return files, nil
}

// isBinaryFile checks if a file is binary
func isBinaryFile(path string) bool {
	// Common binary file extensions
	binaryExtensions := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".db": true, ".sqlite": true, ".ico": true, ".ttf": true,
		".woff": true, ".woff2": true, ".eot": true, ".otf": true,
	}

	ext := strings.ToLower(filepath.Ext(path))
	return binaryExtensions[ext]
}

// isLargeFile checks if a file is too large to process
func isLargeFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true
	}

	// Skip files larger than 1MB
	return info.Size() > 1024*1024
}

// DetermineLanguage determines the programming language based on file extension (exported)
func DetermineLanguage(path string) string {
	return determineLanguage(path)
}

// determineLanguage determines the programming language based on file extension
func determineLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	languageMap := map[string]string{
		".go":    "Go",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".jsx":   "JavaScript",
		".tsx":   "TypeScript",
		".py":    "Python",
		".java":  "Java",
		".cpp":   "C++",
		".cc":    "C++",
		".cxx":   "C++",
		".c":     "C",
		".h":     "C",
		".hpp":   "C++",
		".rs":    "Rust",
		".php":   "PHP",
		".rb":    "Ruby",
		".swift": "Swift",
		".kt":    "Kotlin",
		".sql":   "SQL",
		".html":  "HTML",
		".css":   "CSS",
		".scss":  "SCSS",
		".sass":  "Sass",
		".md":    "Markdown",
		".txt":   "Text",
		".json":  "JSON",
		".xml":   "XML",
		".yaml":  "YAML",
		".yml":   "YAML",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "Unknown"
}
