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

// determineLanguage determines the programming language based on file extension
func determineLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	languageMap := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".jsx":  "JavaScript",
		".tsx":  "TypeScript",
		".py":   "Python",
		".java": "Java",
		".cpp":  "C++",
		".cc":   "C++",
		".cxx":  "C++",
		".c":    "C",
		".h":    "C",
		".hpp":  "C++",
		".rs":   "Rust",
		".php":  "PHP",
		".rb":   "Ruby",
		".swift": "Swift",
		".kt":   "Kotlin",
		".sql":  "SQL",
		".html": "HTML",
		".css":  "CSS",
		".scss": "SCSS",
		".sass": "Sass",
		".md":   "Markdown",
		".txt":  "Text",
		".json": "JSON",
		".xml":  "XML",
		".yaml": "YAML",
		".yml":  "YAML",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}
	
	return "Unknown"
}