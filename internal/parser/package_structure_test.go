package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJavaParser_RealProjectStructure tests Java parser with realistic directory structure
func TestJavaParser_RealProjectStructure(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	tests := []struct {
		name            string
		relativePath    string
		expectedPackage string
		expectedClass   string
	}{
		{
			name:            "User model",
			relativePath:    "../../tests/fixtures/java/src/main/java/com/example/myapp/model/User.java",
			expectedPackage: "com.example.myapp.model",
			expectedClass:   "com.example.myapp.model.User",
		},
		{
			name:            "UserRepository",
			relativePath:    "../../tests/fixtures/java/src/main/java/com/example/myapp/repository/UserRepository.java",
			expectedPackage: "com.example.myapp.repository",
			expectedClass:   "com.example.myapp.repository.UserRepository",
		},
		{
			name:            "UserService",
			relativePath:    "../../tests/fixtures/java/src/main/java/com/example/myapp/service/UserService.java",
			expectedPackage: "com.example.myapp.service",
			expectedClass:   "com.example.myapp.service.UserService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := filepath.Abs(tt.relativePath)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("Test fixture not found: %s", absPath)
			}

			file := ScannedFile{
				Path:     tt.relativePath,
				AbsPath:  absPath,
				Language: "java",
			}

			parsedFile, err := parser.Parse(file)
			if err != nil {
				t.Fatalf("Parse() failed: %v", err)
			}

			// Verify package
			foundPackage := false
			for _, symbol := range parsedFile.Symbols {
				if symbol.Kind == "package" {
					foundPackage = true
					if symbol.Name != tt.expectedPackage {
						t.Errorf("Expected package '%s', got '%s'", tt.expectedPackage, symbol.Name)
					}
					break
				}
			}

			if !foundPackage {
				t.Error("Package symbol not found")
			}

			// Verify class with FQN
			foundClass := false
			for _, symbol := range parsedFile.Symbols {
				if symbol.Kind == "class" {
					foundClass = true
					if symbol.Name != tt.expectedClass {
						t.Errorf("Expected class FQN '%s', got '%s'", tt.expectedClass, symbol.Name)
					}
					break
				}
			}

			if !foundClass {
				t.Error("Class symbol not found")
			}
		})
	}
}

// TestJavaParser_InternalDependencies tests that internal project dependencies are correctly identified
func TestJavaParser_InternalDependencies(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewJavaParser(tsParser)

	// Parse UserService which imports User and UserRepository
	absPath, err := filepath.Abs("../../tests/fixtures/java/src/main/java/com/example/myapp/service/UserService.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "../../tests/fixtures/java/src/main/java/com/example/myapp/service/UserService.java",
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check import classifications
	importChecks := map[string]struct {
		shouldBeExternal bool
		reason           string
	}{
		"java.util.List": {
			shouldBeExternal: false,
			reason:           "Java standard library",
		},
		"com.example.myapp.model.User": {
			shouldBeExternal: false,
			reason:           "Same base package (com.example)",
		},
		"com.example.myapp.repository.UserRepository": {
			shouldBeExternal: false,
			reason:           "Same base package (com.example)",
		},
	}

	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			check, exists := importChecks[dep.Target]
			if exists {
				if dep.IsExternal != check.shouldBeExternal {
					t.Errorf("Import '%s': expected IsExternal=%v (%s), got %v",
						dep.Target, check.shouldBeExternal, check.reason, dep.IsExternal)
				}
			}
		}
	}
}

// TestKotlinParser_RealProjectStructure tests Kotlin parser with realistic directory structure
func TestKotlinParser_RealProjectStructure(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	tests := []struct {
		name            string
		relativePath    string
		expectedPackage string
		expectedClass   string
	}{
		{
			name:            "User model",
			relativePath:    "../../tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/model/User.kt",
			expectedPackage: "com.example.myapp.model",
			expectedClass:   "com.example.myapp.model.User",
		},
		{
			name:            "UserRepository",
			relativePath:    "../../tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/repository/UserRepository.kt",
			expectedPackage: "com.example.myapp.repository",
			expectedClass:   "com.example.myapp.repository.UserRepository",
		},
		{
			name:            "UserService",
			relativePath:    "../../tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt",
			expectedPackage: "com.example.myapp.service",
			expectedClass:   "com.example.myapp.service.UserService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := filepath.Abs(tt.relativePath)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("Test fixture not found: %s", absPath)
			}

			file := ScannedFile{
				Path:     tt.relativePath,
				AbsPath:  absPath,
				Language: "kotlin",
			}

			parsedFile, err := parser.Parse(file)
			if err != nil {
				t.Fatalf("Parse() failed: %v", err)
			}

			// Verify package
			foundPackage := false
			for _, symbol := range parsedFile.Symbols {
				if symbol.Kind == "package" {
					foundPackage = true
					if symbol.Name != tt.expectedPackage {
						t.Errorf("Expected package '%s', got '%s'", tt.expectedPackage, symbol.Name)
					}
					break
				}
			}

			if !foundPackage {
				t.Error("Package symbol not found")
			}

			// Verify class with FQN
			foundClass := false
			for _, symbol := range parsedFile.Symbols {
				if symbol.Kind == "class" || symbol.Kind == "data_class" {
					foundClass = true
					if symbol.Name != tt.expectedClass {
						t.Errorf("Expected class FQN '%s', got '%s'", tt.expectedClass, symbol.Name)
					}
					break
				}
			}

			if !foundClass {
				t.Error("Class symbol not found")
			}
		})
	}
}

// TestKotlinParser_InternalDependencies tests that internal project dependencies are correctly identified
func TestKotlinParser_InternalDependencies(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewKotlinParser(tsParser)

	// Parse UserService which imports User and UserRepository
	absPath, err := filepath.Abs("../../tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test fixture not found")
	}

	file := ScannedFile{
		Path:     "../../tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt",
		AbsPath:  absPath,
		Language: "kotlin",
	}

	parsedFile, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check import classifications
	importChecks := map[string]struct {
		shouldBeExternal bool
		reason           string
	}{
		"com.example.myapp.model.User": {
			shouldBeExternal: false,
			reason:           "Same base package (com.example)",
		},
		"com.example.myapp.repository.UserRepository": {
			shouldBeExternal: false,
			reason:           "Same base package (com.example)",
		},
	}

	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			check, exists := importChecks[dep.Target]
			if exists {
				if dep.IsExternal != check.shouldBeExternal {
					t.Errorf("Import '%s': expected IsExternal=%v (%s), got %v",
						dep.Target, check.shouldBeExternal, check.reason, dep.IsExternal)
				}
			}
		}
	}
}

// TestCrossLanguageStructure tests that both parsers handle the same package structure consistently
func TestCrossLanguageStructure(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	javaParser := NewJavaParser(tsParser)
	kotlinParser := NewKotlinParser(tsParser)

	// Parse Java UserService
	javaPath, err := filepath.Abs("../../tests/fixtures/java/src/main/java/com/example/myapp/service/UserService.java")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(javaPath); os.IsNotExist(err) {
		t.Skip("Java test fixture not found")
	}

	javaFile := ScannedFile{
		Path:     "../../tests/fixtures/java/src/main/java/com/example/myapp/service/UserService.java",
		AbsPath:  javaPath,
		Language: "java",
	}

	javaParsed, err := javaParser.Parse(javaFile)
	if err != nil {
		t.Fatalf("Java Parse() failed: %v", err)
	}

	// Parse Kotlin UserService
	kotlinPath, err := filepath.Abs("../../tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(kotlinPath); os.IsNotExist(err) {
		t.Skip("Kotlin test fixture not found")
	}

	kotlinFile := ScannedFile{
		Path:     "../../tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt",
		AbsPath:  kotlinPath,
		Language: "kotlin",
	}

	kotlinParsed, err := kotlinParser.Parse(kotlinFile)
	if err != nil {
		t.Fatalf("Kotlin Parse() failed: %v", err)
	}

	// Both should have the same package
	var javaPackage, kotlinPackage string
	for _, symbol := range javaParsed.Symbols {
		if symbol.Kind == "package" {
			javaPackage = symbol.Name
			break
		}
	}
	for _, symbol := range kotlinParsed.Symbols {
		if symbol.Kind == "package" {
			kotlinPackage = symbol.Name
			break
		}
	}

	if javaPackage != kotlinPackage {
		t.Errorf("Package mismatch: Java='%s', Kotlin='%s'", javaPackage, kotlinPackage)
	}

	expectedPackage := "com.example.myapp.service"
	if javaPackage != expectedPackage {
		t.Errorf("Expected package '%s', Java got '%s'", expectedPackage, javaPackage)
	}
	if kotlinPackage != expectedPackage {
		t.Errorf("Expected package '%s', Kotlin got '%s'", expectedPackage, kotlinPackage)
	}

	// Both should have UserService class with same FQN
	var javaClass, kotlinClass string
	for _, symbol := range javaParsed.Symbols {
		if symbol.Kind == "class" && strings.HasSuffix(symbol.Name, "UserService") {
			javaClass = symbol.Name
			break
		}
	}
	for _, symbol := range kotlinParsed.Symbols {
		if symbol.Kind == "class" && strings.HasSuffix(symbol.Name, "UserService") {
			kotlinClass = symbol.Name
			break
		}
	}

	expectedFQN := "com.example.myapp.service.UserService"
	if javaClass != expectedFQN {
		t.Errorf("Expected Java class FQN '%s', got '%s'", expectedFQN, javaClass)
	}
	if kotlinClass != expectedFQN {
		t.Errorf("Expected Kotlin class FQN '%s', got '%s'", expectedFQN, kotlinClass)
	}

	// Both should classify imports the same way
	// Check that both recognize com.example.myapp.* as internal
	javaInternalCount := 0
	kotlinInternalCount := 0

	for _, dep := range javaParsed.Dependencies {
		if dep.Type == "import" && strings.HasPrefix(dep.Target, "com.example.myapp.") && !dep.IsExternal {
			javaInternalCount++
		}
	}

	for _, dep := range kotlinParsed.Dependencies {
		if dep.Type == "import" && strings.HasPrefix(dep.Target, "com.example.myapp.") && !dep.IsExternal {
			kotlinInternalCount++
		}
	}

	if javaInternalCount == 0 {
		t.Error("Java parser should find internal imports from com.example.myapp.*")
	}
	if kotlinInternalCount == 0 {
		t.Error("Kotlin parser should find internal imports from com.example.myapp.*")
	}
}
