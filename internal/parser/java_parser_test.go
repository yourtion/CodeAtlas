package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJavaParser_Parse(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "simple class",
			filename: "simple_class.java",
			wantErr:  false,
		},
		{
			name:     "interface",
			filename: "interface.java",
			wantErr:  false,
		},
		{
			name:     "enum",
			filename: "enum.java",
			wantErr:  false,
		},
		{
			name:     "annotations",
			filename: "annotations.java",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get absolute path to test fixture
			absPath, err := filepath.Abs(filepath.Join("../../tests/fixtures/java", tt.filename))
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			// Check if file exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("Test fixture not found: %s", absPath)
			}

			file := ScannedFile{
				Path:     tt.filename,
				AbsPath:  absPath,
				Language: "java",
			}

			parsedFile, err := parser.Parse(file)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if parsedFile == nil {
				t.Error("Parse() returned nil parsedFile")
				return
			}

			// Basic validation
			if parsedFile.Language != "java" {
				t.Errorf("Expected language 'java', got '%s'", parsedFile.Language)
			}

			if len(parsedFile.Content) == 0 {
				t.Error("Parsed file has empty content")
			}
		})
	}
}

func TestJavaParser_ExtractPackage(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/simple_class.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "simple_class.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for package symbol
	foundPackage := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "package" {
			foundPackage = true
			if symbol.Name != "com.example.test" {
				t.Errorf("Expected package 'com.example.test', got '%s'", symbol.Name)
			}
			break
		}
	}

	if !foundPackage {
		t.Error("Package symbol not found")
	}
}

func TestJavaParser_ExtractImports(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/simple_class.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "simple_class.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for import dependencies
	importCount := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			importCount++
			
			// Check that java.* imports are marked as internal
			if dep.Target == "java.util.List" || dep.Target == "java.util.ArrayList" {
				if dep.IsExternal {
					t.Errorf("Import '%s' should be internal (java.* package)", dep.Target)
				}
			}
		}
	}

	if importCount == 0 {
		t.Error("No import dependencies found")
	}
}

func TestJavaParser_ExtractClasses(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/simple_class.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "simple_class.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for class symbol
	foundClass := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "class" && strings.HasSuffix(symbol.Name, "SimpleClass") {
			foundClass = true

			// Check for Javadoc
			if symbol.Docstring == "" {
				t.Error("Class should have Javadoc")
			}

			// Check for fields
			foundField := false
			for _, child := range symbol.Children {
				if child.Kind == "field" && (child.Name == "name" || child.Name == "age") {
					foundField = true
					break
				}
			}
			if !foundField {
				t.Error("Class should have fields")
			}

			// Check for methods
			foundMethod := false
			for _, child := range symbol.Children {
				if child.Kind == "method" && (child.Name == "getName" || child.Name == "setName") {
					foundMethod = true
					break
				}
			}
			if !foundMethod {
				t.Error("Class should have methods")
			}

			// Check for constructor
			foundConstructor := false
			for _, child := range symbol.Children {
				if child.Kind == "constructor" && child.Name == "SimpleClass" {
					foundConstructor = true
					break
				}
			}
			if !foundConstructor {
				t.Error("Class should have constructor")
			}

			break
		}
	}

	if !foundClass {
		t.Error("SimpleClass not found")
	}
}

func TestJavaParser_ExtractInterfaces(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/interface.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "interface.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for interface symbols
	foundDrawable := false
	foundAdvancedDrawable := false

	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "interface" {
			if strings.HasSuffix(symbol.Name, "Drawable") && !strings.Contains(symbol.Name, "Advanced") {
				foundDrawable = true

				// Check for methods
				if len(symbol.Children) == 0 {
					t.Error("Drawable interface should have methods")
				}

				// Check for specific methods
				foundDraw := false
				for _, child := range symbol.Children {
					if child.Kind == "method" && child.Name == "draw" {
						foundDraw = true
						break
					}
				}
				if !foundDraw {
					t.Error("Drawable interface should have draw() method")
				}
			}

			if strings.HasSuffix(symbol.Name, "AdvancedDrawable") {
				foundAdvancedDrawable = true
			}
		}
	}

	if !foundDrawable {
		t.Error("Drawable interface not found")
	}

	if !foundAdvancedDrawable {
		t.Error("AdvancedDrawable interface not found")
	}

	// Check for interface extension dependency
	foundExtends := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "extends" && strings.HasSuffix(dep.Source, "AdvancedDrawable") && dep.Target == "Drawable" {
			foundExtends = true
			break
		}
	}

	if !foundExtends {
		t.Error("Interface extension dependency not found")
	}
}

func TestJavaParser_ExtractEnums(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/enum.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "enum.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for enum symbol
	foundEnum := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "enum" && strings.HasSuffix(symbol.Name, "DayOfWeek") {
			foundEnum = true

			// Check for enum constants
			if len(symbol.Children) == 0 {
				t.Error("Enum should have constants")
			}

			// Check for specific constants
			foundMonday := false
			foundSunday := false
			for _, child := range symbol.Children {
				if child.Kind == "enum_constant" {
					if child.Name == "MONDAY" {
						foundMonday = true
					}
					if child.Name == "SUNDAY" {
						foundSunday = true
					}
				}
			}

			if !foundMonday {
				t.Error("MONDAY constant not found")
			}

			if !foundSunday {
				t.Error("SUNDAY constant not found")
			}

			break
		}
	}

	if !foundEnum {
		t.Error("DayOfWeek enum not found")
	}
}

func TestJavaParser_ExtractAnnotations(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/annotations.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "annotations.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for annotation definition
	foundAnnotation := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "annotation" && symbol.Name == "TestAnnotation" {
			foundAnnotation = true
			break
		}
	}

	if !foundAnnotation {
		t.Error("TestAnnotation definition not found")
	}

	// Check for annotation usage dependencies
	foundAnnotationUsage := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "annotated_with" {
			foundAnnotationUsage = true
			break
		}
	}

	if !foundAnnotationUsage {
		t.Error("Annotation usage dependencies not found")
	}
}

func TestJavaParser_ExtractCallRelationships(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/simple_class.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "simple_class.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for call dependencies
	foundCall := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			foundCall = true
			break
		}
	}

	if !foundCall {
		t.Error("No call dependencies found")
	}
}

func TestJavaParser_ExtractJavadoc(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/simple_class.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "simple_class.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check that class has Javadoc
	foundClassWithJavadoc := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "class" && strings.HasSuffix(symbol.Name, "SimpleClass") {
			if symbol.Docstring != "" {
				foundClassWithJavadoc = true
			}
			break
		}
	}

	if !foundClassWithJavadoc {
		t.Error("Class should have Javadoc documentation")
	}
}

func TestJavaParser_ErrorHandling(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	t.Run("non-existent file", func(t *testing.T) {
		file := ScannedFile{
			Path:     "nonexistent.java",
			AbsPath:  "/nonexistent/path/nonexistent.java",
			Language: "java",
		}

		_, err := parser.Parse(file)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}

		detailedErr, ok := err.(*DetailedParseError)
		if !ok {
			t.Error("Expected DetailedParseError")
		} else if detailedErr.Type != "filesystem" {
			t.Errorf("Expected error type 'filesystem', got '%s'", detailedErr.Type)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		// Create a temporary empty file
		tmpFile, err := os.CreateTemp("", "empty_*.java")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		file := ScannedFile{
			Path:     "empty.java",
			AbsPath:  tmpFile.Name(),
			Language: "java",
		}

		parsedFile, err := parser.Parse(file)
		// Empty file should parse but may have errors
		if parsedFile == nil {
			t.Error("Expected parsedFile even for empty file")
		}
	})
}

func TestJavaParser_Generics(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/simple_class.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "simple_class.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// The simple_class.java file uses List<String> and ArrayList<>
	// Just verify that the file parses successfully with generics
	if parsedFile == nil {
		t.Error("Failed to parse file with generics")
	}
}

func TestJavaParser_InferPackageFromPath(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	tests := []struct {
		name         string
		filePath     string
		wantPackage  string
	}{
		{
			name:        "standard Maven structure",
			filePath:    "src/main/java/com/example/MyClass.java",
			wantPackage: "com.example",
		},
		{
			name:        "test source",
			filePath:    "src/test/java/com/example/test/MyTest.java",
			wantPackage: "com.example.test",
		},
		{
			name:        "simple src structure",
			filePath:    "src/com/example/MyClass.java",
			wantPackage: "com.example",
		},
		{
			name:        "no package (default package)",
			filePath:    "src/main/java/MyClass.java",
			wantPackage: "",
		},
		{
			name:        "deep package structure",
			filePath:    "src/main/java/com/example/module/service/MyService.java",
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

func TestJavaParser_IsExternalImportWithContext(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	tests := []struct {
		name           string
		importPath     string
		currentPackage string
		wantExternal   bool
	}{
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
			importPath:     "org.apache.commons.lang3.StringUtils",
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

func TestJavaParser_FullyQualifiedNames(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/simple_class.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "simple_class.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check that class has fully qualified name
	foundClass := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "class" {
			// Should be com.example.test.SimpleClass, not just SimpleClass
			if strings.Contains(symbol.Name, ".") {
				foundClass = true
				expectedFQN := "com.example.test.SimpleClass"
				if symbol.Name != expectedFQN {
					t.Errorf("Expected fully qualified name '%s', got '%s'", expectedFQN, symbol.Name)
				}
			}
			break
		}
	}

	if !foundClass {
		t.Error("Class with fully qualified name not found")
	}
}

func TestJavaParser_ExtractBasePackage(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

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

func TestJavaParser_ProjectInternalDependencies(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/java/project_structure_example.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "project_structure_example.java",
		AbsPath:  absPath,
		Language: "java",
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

	// Check import classifications
	importChecks := map[string]bool{
		"java.util.List":                           false, // Should be internal (java.*)
		"java.util.ArrayList":                      false, // Should be internal (java.*)
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
