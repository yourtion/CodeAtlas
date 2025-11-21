package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKotlinParser_InferPackageFromPath(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	tests := []struct {
		name        string
		filePath    string
		wantPackage string
	}{
		{
			name:        "standard Kotlin structure",
			filePath:    "src/main/kotlin/com/example/MyClass.kt",
			wantPackage: "com.example",
		},
		{
			name:        "test source",
			filePath:    "src/test/kotlin/com/example/test/MyTest.kt",
			wantPackage: "com.example.test",
		},
		{
			name:        "Kotlin in Java folder",
			filePath:    "src/main/java/com/example/MyClass.kt",
			wantPackage: "com.example",
		},
		{
			name:        "simple src structure",
			filePath:    "src/com/example/MyClass.kt",
			wantPackage: "com.example",
		},
		{
			name:        "no package (default package)",
			filePath:    "src/main/kotlin/MyClass.kt",
			wantPackage: "",
		},
		{
			name:        "deep package structure",
			filePath:    "src/main/kotlin/com/example/module/service/MyService.kt",
			wantPackage: "com.example.module.service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.inferPackageFromPath(tt.filePath)
			if got != tt.wantPackage {
				t.Errorf("inferPackageFromPath() = %v, want %v", got, tt.wantPackage)
			}
		})
	}
}

func TestKotlinParser_IsExternalImportWithContext(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	tests := []struct {
		name           string
		importPath     string
		currentPackage string
		wantExternal   bool
	}{
		{
			name:           "kotlin standard library",
			importPath:     "kotlin.collections.List",
			currentPackage: "com.example",
			wantExternal:   false,
		},
		{
			name:           "kotlinx library",
			importPath:     "kotlinx.coroutines.launch",
			currentPackage: "com.example",
			wantExternal:   false,
		},
		{
			name:           "java standard library",
			importPath:     "java.util.List",
			currentPackage: "com.example",
			wantExternal:   false,
		},
		{
			name:           "javax library",
			importPath:     "javax.servlet.http.HttpServlet",
			currentPackage: "com.example",
			wantExternal:   false,
		},
		{
			name:           "same base package",
			importPath:     "com.example.service.UserService",
			currentPackage: "com.example.controller",
			wantExternal:   false,
		},
		{
			name:           "same package",
			importPath:     "com.example.MyClass",
			currentPackage: "com.example",
			wantExternal:   false,
		},
		{
			name:           "different base package",
			importPath:     "org.springframework.boot.SpringApplication",
			currentPackage: "com.example",
			wantExternal:   true,
		},
		{
			name:           "third party library",
			importPath:     "io.ktor.server.engine.embeddedServer",
			currentPackage: "com.example",
			wantExternal:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.isExternalImportWithContext(tt.importPath, tt.currentPackage)
			if got != tt.wantExternal {
				t.Errorf("isExternalImportWithContext() = %v, want %v", got, tt.wantExternal)
			}
		})
	}
}

func TestKotlinParser_ExtractBasePackage(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	tests := []struct {
		name        string
		packageName string
		wantBase    string
	}{
		{
			name:        "two segments",
			packageName: "com.example",
			wantBase:    "com.example",
		},
		{
			name:        "three segments",
			packageName: "com.example.module",
			wantBase:    "com.example",
		},
		{
			name:        "deep package",
			packageName: "com.example.module.service.impl",
			wantBase:    "com.example",
		},
		{
			name:        "single segment",
			packageName: "example",
			wantBase:    "example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.extractBasePackage(tt.packageName)
			if got != tt.wantBase {
				t.Errorf("extractBasePackage() = %v, want %v", got, tt.wantBase)
			}
		})
	}
}

func TestKotlinParser_KotlinJavaInterop(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/kotlin/kotlin_java_interop.kt")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "kotlin_java_interop.kt",
		AbsPath:  absPath,
		Language: "kotlin",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check that class has fully qualified name
	foundClass := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "class" && strings.HasSuffix(symbol.Name, "UserService") {
			foundClass = true
			expectedFQN := "com.example.myapp.service.UserService"
			if symbol.Name != expectedFQN {
				t.Errorf("Expected fully qualified name '%s', got '%s'", expectedFQN, symbol.Name)
			}
			break
		}
	}

	if !foundClass {
		t.Error("UserService class not found")
	}

	// Check import classifications for Kotlin-Java interop
	importChecks := map[string]bool{
		"java.util.ArrayList":                      false, // Should be internal (java.*)
		"java.util.List":                           false, // Should be internal (java.*)
		"com.example.myapp.model.User":             false, // Should be internal (same base package)
		"com.example.myapp.repository.UserRepository": false, // Should be internal (same base package)
		"org.springframework.stereotype.Service":   true,  // Should be external (different base package)
	}

	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			expectedExternal, exists := importChecks[dep.Target]
			if exists {
				if dep.IsExternal != expectedExternal {
					t.Errorf("Import '%s': expected IsExternal=%v, got %v",
						dep.Target, expectedExternal, dep.IsExternal)
				}
			}
		}
	}
}

func TestKotlinParser_FullyQualifiedNames(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/kotlin/simple_class.kt")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// TODO: Create missing test fixture file tests/fixtures/kotlin/simple_class.kt
		t.Skip("Test fixture not found: tests/fixtures/kotlin/simple_class.kt - " +
			"create this fixture file to enable the test")
	}

	file := ScannedFile{
		Path:     "simple_class.kt",
		AbsPath:  absPath,
		Language: "kotlin",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check that classes have fully qualified names
	foundClassWithFQN := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "class" || symbol.Kind == "data_class" {
			// Should have package prefix
			if strings.Contains(symbol.Name, ".") {
				foundClassWithFQN = true
				break
			}
		}
	}

	if !foundClassWithFQN {
		t.Error("No class with fully qualified name found")
	}
}
