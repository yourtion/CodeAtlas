package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCParser_Parse_FunctionsHeader(t *testing.T) {
	// Create parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCParser(tsParser)

	// Read test file
	testFile := "../../tests/fixtures/c/functions.h"
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "c",
	}

	// Parse file
	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Verify basic properties
	if parsedFile.Language != "c" {
		t.Errorf("Expected language 'c', got '%s'", parsedFile.Language)
	}

	// Check for function declarations
	foundAdd := false
	foundMultiply := false
	foundGreet := false

	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "function_declaration" {
			switch symbol.Name {
			case "add":
				foundAdd = true
				if symbol.Docstring == "" {
					t.Error("Expected docstring for add function")
				}
			case "multiply":
				foundMultiply = true
			case "greet":
				foundGreet = true
			}
		}
	}

	if !foundAdd {
		t.Error("Expected to find 'add' function declaration")
	}
	if !foundMultiply {
		t.Error("Expected to find 'multiply' function declaration")
	}
	if !foundGreet {
		t.Error("Expected to find 'greet' function declaration")
	}

	// Check for includes
	foundStdio := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" && dep.Target == "stdio.h" {
			foundStdio = true
			if dep.IsExternal {
				t.Error("stdio.h should be internal (standard library)")
			}
		}
	}

	if !foundStdio {
		t.Error("Expected to find stdio.h include")
	}
}

func TestCParser_Parse_FunctionsImplementation(t *testing.T) {
	// Create parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCParser(tsParser)

	// Read test file
	testFile := "../../tests/fixtures/c/functions.c"
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "c",
	}

	// Parse file
	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Check for function definitions
	foundAdd := false
	foundMultiply := false
	foundGreet := false
	foundHelper := false

	for _, symbol := range parsedFile.Symbols {
		switch symbol.Name {
		case "add":
			foundAdd = true
			if symbol.Kind != "function" {
				t.Errorf("Expected 'function' kind for add, got '%s'", symbol.Kind)
			}
		case "multiply":
			foundMultiply = true
		case "greet":
			foundGreet = true
		case "helper_function":
			foundHelper = true
			if symbol.Kind != "static_function" {
				t.Errorf("Expected 'static_function' kind for helper_function, got '%s'", symbol.Kind)
			}
		}
	}

	if !foundAdd {
		t.Error("Expected to find 'add' function definition")
	}
	if !foundMultiply {
		t.Error("Expected to find 'multiply' function definition")
	}
	if !foundGreet {
		t.Error("Expected to find 'greet' function definition")
	}
	if !foundHelper {
		t.Error("Expected to find 'helper_function' static function")
	}

	// Check for includes
	foundFunctionsH := false
	foundStringH := false

	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			if dep.Target == "functions.h" {
				foundFunctionsH = true
				if dep.IsExternal {
					t.Error("functions.h should be internal (local header)")
				}
			}
			if dep.Target == "string.h" {
				foundStringH = true
			}
		}
	}

	if !foundFunctionsH {
		t.Error("Expected to find functions.h include")
	}
	if !foundStringH {
		t.Error("Expected to find string.h include")
	}

	// Check for call relationships
	foundAddCall := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "multiply" && dep.Target == "add" {
			foundAddCall = true
		}
	}

	if !foundAddCall {
		t.Error("Expected to find call from multiply to add")
	}
}

func TestCParser_Parse_Structs(t *testing.T) {
	// Create parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCParser(tsParser)

	// Read test file
	testFile := "../../tests/fixtures/c/structs.c"
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "c",
	}

	// Parse file
	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Check for struct declarations
	foundPoint := false
	foundPerson := false

	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "struct" {
			switch symbol.Name {
			case "Point":
				foundPoint = true
				// Check for fields
				if len(symbol.Children) < 2 {
					t.Errorf("Expected Point struct to have at least 2 fields, got %d", len(symbol.Children))
				}
			case "Person":
				foundPerson = true
				if len(symbol.Children) < 3 {
					t.Errorf("Expected Person struct to have at least 3 fields, got %d", len(symbol.Children))
				}
			}
		}
	}

	if !foundPoint {
		t.Error("Expected to find Point struct")
	}
	if !foundPerson {
		t.Error("Expected to find Person struct")
	}

	// Check for union
	foundData := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "union" && symbol.Name == "Data" {
			foundData = true
			if len(symbol.Children) < 3 {
				t.Errorf("Expected Data union to have at least 3 fields, got %d", len(symbol.Children))
			}
		}
	}

	if !foundData {
		t.Error("Expected to find Data union")
	}

	// Check for enum
	foundDay := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "enum" && symbol.Name == "Day" {
			foundDay = true
			if len(symbol.Children) < 7 {
				t.Errorf("Expected Day enum to have 7 constants, got %d", len(symbol.Children))
			}
		}
	}

	if !foundDay {
		t.Error("Expected to find Day enum")
	}

	// Check for typedefs
	foundTypedefs := 0
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "typedef" {
			foundTypedefs++
		}
	}

	if foundTypedefs < 2 {
		t.Errorf("Expected at least 2 typedefs, got %d", foundTypedefs)
	}
}

func TestCParser_Parse_Macros(t *testing.T) {
	// Create parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCParser(tsParser)

	// Read test file
	testFile := "../../tests/fixtures/c/macros.h"
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "c",
	}

	// Parse file
	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Check for simple macros
	foundMaxBuffer := false
	foundMinValue := false

	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "macro" {
			switch symbol.Name {
			case "MAX_BUFFER_SIZE":
				foundMaxBuffer = true
			case "MIN_VALUE":
				foundMinValue = true
			}
		}
	}

	if !foundMaxBuffer {
		t.Error("Expected to find MAX_BUFFER_SIZE macro")
	}
	if !foundMinValue {
		t.Error("Expected to find MIN_VALUE macro")
	}

	// Check for function-like macros
	foundMax := false
	foundMin := false
	foundSquare := false

	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "function_macro" {
			switch symbol.Name {
			case "MAX":
				foundMax = true
			case "MIN":
				foundMin = true
			case "SQUARE":
				foundSquare = true
			}
		}
	}

	if !foundMax {
		t.Error("Expected to find MAX function macro")
	}
	if !foundMin {
		t.Error("Expected to find MIN function macro")
	}
	if !foundSquare {
		t.Error("Expected to find SQUARE function macro")
	}

	// Check for extern variable
	foundGlobalCounter := false
	for _, symbol := range parsedFile.Symbols {
		if symbol.Name == "global_counter" {
			foundGlobalCounter = true
			if symbol.Kind != "extern_variable" {
				t.Errorf("Expected global_counter to be extern_variable, got %s", symbol.Kind)
			}
		}
	}

	if !foundGlobalCounter {
		// Debug: print all symbols
		t.Log("All symbols found:")
		for _, symbol := range parsedFile.Symbols {
			t.Logf("  - %s (%s)", symbol.Name, symbol.Kind)
		}
		t.Error("Expected to find global_counter extern variable")
	}
}

func TestCParser_HeaderImplementationPairing(t *testing.T) {
	// Create parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCParser(tsParser)

	// Parse header file
	headerFile := "../../tests/fixtures/c/functions.h"
	headerAbsPath, err := filepath.Abs(headerFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	headerScanned := ScannedFile{
		Path:     headerFile,
		AbsPath:  headerAbsPath,
		Language: "c",
	}

	headerParsed, err := parser.Parse(headerScanned)
	if err != nil {
		t.Fatalf("Failed to parse header file: %v", err)
	}

	// Parse implementation file
	implFile := "../../tests/fixtures/c/functions.c"
	implAbsPath, err := filepath.Abs(implFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	implScanned := ScannedFile{
		Path:     implFile,
		AbsPath:  implAbsPath,
		Language: "c",
	}

	implParsed, err := parser.Parse(implScanned)
	if err != nil {
		t.Fatalf("Failed to parse implementation file: %v", err)
	}

	// Match declarations to implementations
	parser.matchDeclarationToImplementation(headerParsed, implParsed)

	// Check for implements_declaration edges
	foundImplementsDecl := false
	for _, dep := range implParsed.Dependencies {
		if dep.Type == "implements_declaration" {
			foundImplementsDecl = true
			break
		}
	}

	if !foundImplementsDecl {
		t.Error("Expected to find implements_declaration edges")
	}

	// Check for implements_header edge
	foundImplementsHeader := false
	for _, dep := range implParsed.Dependencies {
		if dep.Type == "implements_header" {
			foundImplementsHeader = true
			break
		}
	}

	if !foundImplementsHeader {
		t.Error("Expected to find implements_header edge")
	}
}

func TestCParser_ErrorHandling(t *testing.T) {
	// Create parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCParser(tsParser)

	// Test with non-existent file
	scannedFile := ScannedFile{
		Path:     "nonexistent.c",
		AbsPath:  "/nonexistent/nonexistent.c",
		Language: "c",
	}

	_, err = parser.Parse(scannedFile)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with file containing only comments
	tmpFile, err := os.CreateTemp("", "comment_*.c")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write a comment to the file
	if _, err := tmpFile.WriteString("// This is a comment\n"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	absPath, _ := filepath.Abs(tmpFile.Name())
	commentScanned := ScannedFile{
		Path:     tmpFile.Name(),
		AbsPath:  absPath,
		Language: "c",
	}

	parsedFile, err := parser.Parse(commentScanned)
	if err != nil {
		t.Errorf("Should handle comment-only file gracefully: %v", err)
	}

	if parsedFile == nil {
		t.Error("Expected parsedFile to be non-nil for comment-only file")
	}
}

func TestCParser_DoxygenComments(t *testing.T) {
	// Create parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCParser(tsParser)

	// Read test file with Doxygen comments
	testFile := "../../tests/fixtures/c/functions.h"
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "c",
	}

	// Parse file
	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Check that functions have docstrings
	for _, symbol := range parsedFile.Symbols {
		if symbol.Kind == "function_declaration" {
			if symbol.Docstring == "" {
				t.Errorf("Expected docstring for function '%s'", symbol.Name)
			}
		}
	}
}
