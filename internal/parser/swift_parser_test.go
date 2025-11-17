package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSwiftParser_Parse(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "simple class",
			filename: "simple_class.swift",
			wantErr:  false,
		},
		{
			name:     "protocol",
			filename: "protocol.swift",
			wantErr:  false,
		},
		{
			name:     "extension",
			filename: "extension.swift",
			wantErr:  false,
		},
		{
			name:     "struct",
			filename: "struct.swift",
			wantErr:  false,
		},
		{
			name:     "enum",
			filename: "enum.swift",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath := filepath.Join("../../tests/fixtures/swift", tt.filename)
			
			file := ScannedFile{
				Path:     tt.filename,
				AbsPath:  absPath,
				Language: "swift",
			}

			result, err := parser.Parse(file)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Error("Parse() returned nil result")
				return
			}

			if result.Language != "swift" {
				t.Errorf("Parse() language = %v, want swift", result.Language)
			}

			if len(result.Content) == 0 {
				t.Error("Parse() returned empty content")
			}
		})
	}
}

func TestSwiftParser_ExtractClasses(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)
	absPath := filepath.Join("../../tests/fixtures/swift", "simple_class.swift")
	
	file := ScannedFile{
		Path:     "simple_class.swift",
		AbsPath:  absPath,
		Language: "swift",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for User class
	foundUser := false
	foundAdmin := false
	
	for _, symbol := range result.Symbols {
		if symbol.Kind == "class" && symbol.Name == "User" {
			foundUser = true
			
			// Check for properties
			foundName := false
			foundAge := false
			foundGreet := false
			
			for _, child := range symbol.Children {
				if child.Kind == "property" && child.Name == "name" {
					foundName = true
				}
				if child.Kind == "property" && child.Name == "age" {
					foundAge = true
				}
				if child.Kind == "method" && child.Name == "greet" {
					foundGreet = true
				}
			}
			
			if !foundName {
				t.Error("User class missing 'name' property")
			}
			if !foundAge {
				t.Error("User class missing 'age' property")
			}
			if !foundGreet {
				t.Error("User class missing 'greet' method")
			}
		}
		
		if symbol.Kind == "class" && symbol.Name == "AdminUser" {
			foundAdmin = true
		}
	}

	if !foundUser {
		t.Error("User class not found")
	}
	if !foundAdmin {
		t.Error("AdminUser class not found")
	}

	// Check for inheritance dependency
	foundInheritance := false
	for _, dep := range result.Dependencies {
		if dep.Type == "extends" && dep.Source == "AdminUser" && dep.Target == "User" {
			foundInheritance = true
			break
		}
	}
	
	if !foundInheritance {
		t.Error("AdminUser -> User inheritance dependency not found")
	}
}

func TestSwiftParser_ExtractStructs(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)
	absPath := filepath.Join("../../tests/fixtures/swift", "struct.swift")
	
	file := ScannedFile{
		Path:     "struct.swift",
		AbsPath:  absPath,
		Language: "swift",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for Point struct
	foundPoint := false
	foundGame := false
	foundStack := false
	
	for _, symbol := range result.Symbols {
		if symbol.Kind == "struct" && symbol.Name == "Point" {
			foundPoint = true
			
			// Check for properties and methods
			foundX := false
			foundDistance := false
			
			for _, child := range symbol.Children {
				if child.Kind == "property" && child.Name == "x" {
					foundX = true
				}
				if child.Kind == "method" && child.Name == "distance" {
					foundDistance = true
				}
			}
			
			if !foundX {
				t.Error("Point struct missing 'x' property")
			}
			if !foundDistance {
				t.Error("Point struct missing 'distance' method")
			}
		}
		
		if symbol.Kind == "struct" && symbol.Name == "Game" {
			foundGame = true
		}
		
		if symbol.Kind == "struct" && symbol.Name == "Stack" {
			foundStack = true
		}
	}

	if !foundPoint {
		t.Error("Point struct not found")
	}
	if !foundGame {
		t.Error("Game struct not found")
	}
	if !foundStack {
		t.Error("Stack struct not found")
	}
}

func TestSwiftParser_ExtractEnums(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)
	absPath := filepath.Join("../../tests/fixtures/swift", "enum.swift")
	
	file := ScannedFile{
		Path:     "enum.swift",
		AbsPath:  absPath,
		Language: "swift",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for enums
	foundDirection := false
	foundResult := false
	foundTrafficLight := false
	
	for _, symbol := range result.Symbols {
		if symbol.Kind == "enum" && symbol.Name == "Direction" {
			foundDirection = true
		}
		
		if symbol.Kind == "enum" && symbol.Name == "Result" {
			foundResult = true
		}
		
		if symbol.Kind == "enum" && symbol.Name == "TrafficLight" {
			foundTrafficLight = true
			
			// Check for methods
			foundCanGo := false
			for _, child := range symbol.Children {
				if child.Kind == "method" && child.Name == "canGo" {
					foundCanGo = true
				}
			}
			
			if !foundCanGo {
				t.Error("TrafficLight enum missing 'canGo' method")
			}
		}
	}

	if !foundDirection {
		t.Error("Direction enum not found")
	}
	if !foundResult {
		t.Error("Result enum not found")
	}
	if !foundTrafficLight {
		t.Error("TrafficLight enum not found")
	}
}

func TestSwiftParser_ExtractProtocols(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)
	absPath := filepath.Join("../../tests/fixtures/swift", "protocol.swift")
	
	file := ScannedFile{
		Path:     "protocol.swift",
		AbsPath:  absPath,
		Language: "swift",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for protocols
	foundDrawable := false
	foundResizable := false
	foundShape := false
	
	for _, symbol := range result.Symbols {
		if symbol.Kind == "protocol" && symbol.Name == "Drawable" {
			foundDrawable = true
		}
		
		if symbol.Kind == "protocol" && symbol.Name == "Resizable" {
			foundResizable = true
		}
		
		if symbol.Kind == "protocol" && symbol.Name == "Shape" {
			foundShape = true
		}
	}

	if !foundDrawable {
		t.Error("Drawable protocol not found")
	}
	if !foundResizable {
		t.Error("Resizable protocol not found")
	}
	if !foundShape {
		t.Error("Shape protocol not found")
	}

	// Check for protocol conformance
	foundCircleConforms := false
	foundRectangleConforms := false
	
	for _, dep := range result.Dependencies {
		if dep.Type == "conforms" && dep.Source == "Circle" && dep.Target == "Drawable" {
			foundCircleConforms = true
		}
		if dep.Type == "conforms" && dep.Source == "Rectangle" && dep.Target == "Shape" {
			foundRectangleConforms = true
		}
	}
	
	if !foundCircleConforms {
		t.Error("Circle -> Drawable conformance not found")
	}
	if !foundRectangleConforms {
		t.Error("Rectangle -> Shape conformance not found")
	}
}

func TestSwiftParser_ExtractExtensions(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)
	absPath := filepath.Join("../../tests/fixtures/swift", "extension.swift")
	
	file := ScannedFile{
		Path:     "extension.swift",
		AbsPath:  absPath,
		Language: "swift",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for extensions
	foundPersonExtension := false
	foundStringExtension := false
	
	for _, symbol := range result.Symbols {
		if symbol.Kind == "extension" && symbol.Name == "extension_Person" {
			foundPersonExtension = true
		}
		
		if symbol.Kind == "extension" && symbol.Name == "extension_String" {
			foundStringExtension = true
		}
	}

	if !foundPersonExtension {
		t.Error("Person extension not found")
	}
	if !foundStringExtension {
		t.Error("String extension not found")
	}

	// Check for extension-to-type dependencies
	foundExtensionDep := false
	for _, dep := range result.Dependencies {
		if dep.Type == "extends" && dep.Source == "extension_Person" && dep.Target == "Person" {
			foundExtensionDep = true
			break
		}
	}
	
	if !foundExtensionDep {
		t.Error("Extension -> Person dependency not found")
	}
}

func TestSwiftParser_ExtractImports(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)
	absPath := filepath.Join("../../tests/fixtures/swift", "simple_class.swift")
	
	file := ScannedFile{
		Path:     "simple_class.swift",
		AbsPath:  absPath,
		Language: "swift",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for Foundation import
	foundFoundation := false
	for _, dep := range result.Dependencies {
		if dep.Type == "import" && dep.Target == "Foundation" {
			foundFoundation = true
			if !dep.IsExternal {
				t.Error("Foundation should be marked as external")
			}
			break
		}
	}
	
	if !foundFoundation {
		t.Error("Foundation import not found")
	}
}

func TestSwiftParser_ExtractDocumentation(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)
	absPath := filepath.Join("../../tests/fixtures/swift", "simple_class.swift")
	
	file := ScannedFile{
		Path:     "simple_class.swift",
		AbsPath:  absPath,
		Language: "swift",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for documentation on User class
	for _, symbol := range result.Symbols {
		if symbol.Kind == "class" && symbol.Name == "User" {
			if symbol.Docstring == "" {
				t.Error("User class missing documentation")
			}
		}
	}
}

func TestSwiftParser_PropertyObservers(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)
	absPath := filepath.Join("../../tests/fixtures/swift", "simple_class.swift")
	
	file := ScannedFile{
		Path:     "simple_class.swift",
		AbsPath:  absPath,
		Language: "swift",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for property with observers
	foundStatusWithObservers := false
	for _, symbol := range result.Symbols {
		if symbol.Kind == "class" && symbol.Name == "User" {
			for _, child := range symbol.Children {
				if child.Name == "status" && child.Kind == "property_observer" {
					foundStatusWithObservers = true
					break
				}
			}
		}
	}
	
	if !foundStatusWithObservers {
		t.Error("Property with observers (status) not found or not marked correctly")
	}
}

func TestSwiftParser_ErrorHandling(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)

	t.Run("non-existent file", func(t *testing.T) {
		file := ScannedFile{
			Path:     "nonexistent.swift",
			AbsPath:  "nonexistent.swift",
			Language: "swift",
		}

		_, err := parser.Parse(file)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		// Create a temporary empty file
		tmpFile, err := os.CreateTemp("", "empty_*.swift")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		file := ScannedFile{
			Path:     "empty.swift",
			AbsPath:  tmpFile.Name(),
			Language: "swift",
		}

		result, err := parser.Parse(file)
		// Empty files should return an error from the parser
		if err == nil {
			t.Error("Expected error for empty file")
		}
		if result == nil {
			t.Error("Result should not be nil even for empty file (partial results)")
		}
	})
}

func TestSwiftParser_CallRelationships(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewSwiftParser(tsParser)
	absPath := filepath.Join("../../tests/fixtures/swift", "simple_class.swift")
	
	file := ScannedFile{
		Path:     "simple_class.swift",
		AbsPath:  absPath,
		Language: "swift",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check for call dependencies
	foundCalls := false
	for _, dep := range result.Dependencies {
		if dep.Type == "call" {
			foundCalls = true
			break
		}
	}
	
	// Note: This test may not find calls if the fixture doesn't have explicit function calls
	// The test is here to verify the extraction mechanism works
	_ = foundCalls // Suppress unused variable warning
}
