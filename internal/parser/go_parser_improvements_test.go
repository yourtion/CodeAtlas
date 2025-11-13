package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoParser_InferModulePath(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewGoParser(tsParser)

	tests := []struct {
		name       string
		filePath   string
		wantModule string
	}{
		{
			name:       "github.com path",
			filePath:   "/home/user/go/src/github.com/user/project/pkg/service/user.go",
			wantModule: "github.com/user/project",
		},
		{
			name:       "gitlab.com path",
			filePath:   "/home/user/go/src/gitlab.com/company/project/internal/handler.go",
			wantModule: "gitlab.com/company/project",
		},
		{
			name:       "no recognizable pattern",
			filePath:   "/home/user/myproject/service.go",
			wantModule: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.inferModulePath(tt.filePath)
			if got != tt.wantModule {
				t.Errorf("inferModulePath() = %v, want %v", got, tt.wantModule)
			}
		})
	}
}

func TestGoParser_IsExternalImport(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewGoParser(tsParser)

	tests := []struct {
		name         string
		importPath   string
		filePath     string
		wantExternal bool
	}{
		{
			name:         "standard library - no dots",
			importPath:   "fmt",
			filePath:     "/any/path/file.go",
			wantExternal: false,
		},
		{
			name:         "standard library - with path",
			importPath:   "net/http",
			filePath:     "/any/path/file.go",
			wantExternal: false,
		},
		{
			name:         "internal import - same module",
			importPath:   "github.com/user/project/pkg/service",
			filePath:     "/home/user/go/src/github.com/user/project/cmd/main.go",
			wantExternal: false,
		},
		{
			name:         "external import - different module",
			importPath:   "github.com/other/library",
			filePath:     "/home/user/go/src/github.com/user/project/cmd/main.go",
			wantExternal: true,
		},
		{
			name:         "external import - third party",
			importPath:   "go.uber.org/zap",
			filePath:     "/home/user/go/src/github.com/user/project/cmd/main.go",
			wantExternal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.isExternalImport(tt.importPath, tt.filePath)
			if got != tt.wantExternal {
				t.Errorf("isExternalImport() = %v, want %v", got, tt.wantExternal)
			}
		})
	}
}

func TestGoParser_FindModulePathFromGoMod(t *testing.T) {
	// Create a temporary directory structure with go.mod
	tmpDir, err := os.MkdirTemp("", "goparser-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod file
	goModContent := `module github.com/yourtionguo/CodeAtlas

go 1.21

require (
	github.com/smacker/go-tree-sitter v0.0.0-20230720070738-0d0a9f78d8f8
)
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "internal", "parser")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a test file in subdirectory
	testFile := filepath.Join(subDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package parser"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewGoParser(tsParser)

	// Test finding module path from subdirectory
	modulePath := parser.findModulePathFromGoMod(testFile)
	expectedModule := "github.com/yourtionguo/CodeAtlas"

	if modulePath != expectedModule {
		t.Errorf("findModulePathFromGoMod() = %v, want %v", modulePath, expectedModule)
	}
}

func TestGoParser_RealProjectImports(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewGoParser(tsParser)

	// Test with actual go_parser.go file
	absPath, err := filepath.Abs("go_parser.go")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("go_parser.go not found")
	}

	file := ScannedFile{
		Path:     "go_parser.go",
		AbsPath:  absPath,
		Language: "go",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check that imports are classified
	hasStdlibImport := false
	hasInternalImport := false

	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			// Standard library imports should be internal
			if dep.Target == "fmt" || dep.Target == "os" || dep.Target == "strings" {
				if dep.IsExternal {
					t.Errorf("Standard library import '%s' should be internal", dep.Target)
				}
				hasStdlibImport = true
			}

			// Imports from same module should be internal
			if dep.Target == "github.com/yourtionguo/CodeAtlas/internal/parser" {
				if dep.IsExternal {
					t.Errorf("Internal module import '%s' should be internal", dep.Target)
				}
				hasInternalImport = true
			}
		}
	}

	if !hasStdlibImport {
		t.Log("Note: No standard library imports found (this is okay)")
	}
	
	if !hasInternalImport {
		t.Log("Note: No internal module imports found (this is okay)")
	}
}
