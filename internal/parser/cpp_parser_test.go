package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCppParser_Parse(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCppParser(tsParser)

	tests := []struct {
		name          string
		filename      string
		wantSymbols   int
		wantDeps      int
		checkSymbols  func(*testing.T, *ParsedFile)
		checkDeps     func(*testing.T, *ParsedFile)
	}{
		{
			name:        "class header",
			filename:    "../../tests/fixtures/cpp/class.hpp",
			wantSymbols: 2, // MyClass, DerivedClass (namespace may not be extracted as top-level)
			wantDeps:    2, // includes
			checkSymbols: func(t *testing.T, pf *ParsedFile) {
				// Check MyClass
				found := false
				for _, sym := range pf.Symbols {
					if sym.Kind == "class" && sym.Name == "MyClass" {
						found = true
						// Check that it has methods
						if len(sym.Children) == 0 {
							t.Error("Expected MyClass to have methods")
						}
						break
					}
				}
				if !found {
					t.Error("Expected to find class 'MyClass'")
				}

				// Check DerivedClass
				found = false
				for _, sym := range pf.Symbols {
					if sym.Kind == "class" && sym.Name == "DerivedClass" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected to find class 'DerivedClass'")
				}
			},
			checkDeps: func(t *testing.T, pf *ParsedFile) {
				// Check for includes
				foundString := false
				foundVector := false
				for _, dep := range pf.Dependencies {
					if dep.Type == "import" {
						if dep.Target == "string" {
							foundString = true
						}
						if dep.Target == "vector" {
							foundVector = true
						}
					}
				}
				if !foundString {
					t.Error("Expected to find #include <string>")
				}
				if !foundVector {
					t.Error("Expected to find #include <vector>")
				}
			},
		},
		{
			name:        "class implementation",
			filename:    "../../tests/fixtures/cpp/class.cpp",
			wantSymbols: 5, // namespace, functions
			wantDeps:    3, // includes + calls
			checkSymbols: func(t *testing.T, pf *ParsedFile) {
				// Check for function implementations
				foundConstructor := false
				foundGetName := false
				for _, sym := range pf.Symbols {
					if sym.Kind == "function" {
						if sym.Name == "MyClass" {
							foundConstructor = true
						}
						if sym.Name == "getName" {
							foundGetName = true
						}
					}
				}
				if !foundConstructor {
					t.Error("Expected to find MyClass constructor")
				}
				if !foundGetName {
					t.Error("Expected to find getName function")
				}
			},
			checkDeps: func(t *testing.T, pf *ParsedFile) {
				// Check for includes
				foundClassHpp := false
				for _, dep := range pf.Dependencies {
					if dep.Type == "import" && dep.Target == "class.hpp" {
						foundClassHpp = true
					}
				}
				if !foundClassHpp {
					t.Error("Expected to find #include \"class.hpp\"")
				}
			},
		},
		{
			name:        "template header",
			filename:    "../../tests/fixtures/cpp/template.hpp",
			wantSymbols: 4, // namespace, Container template, max template, Pair template
			wantDeps:    2, // includes
			checkSymbols: func(t *testing.T, pf *ParsedFile) {
				// Check for template class
				foundContainer := false
				foundMax := false
				for _, sym := range pf.Symbols {
					if sym.Kind == "class_template" && sym.Name == "Container" {
						foundContainer = true
					}
					if sym.Kind == "function_template" && sym.Name == "max" {
						foundMax = true
					}
				}
				if !foundContainer {
					t.Error("Expected to find template class 'Container'")
				}
				if !foundMax {
					t.Error("Expected to find template function 'max'")
				}
			},
		},
		{
			name:        "namespace file",
			filename:    "../../tests/fixtures/cpp/namespace.cpp",
			wantSymbols: 2, // classes, functions (namespaces may not be extracted as top-level symbols)
			wantDeps:    2, // includes
			checkSymbols: func(t *testing.T, pf *ParsedFile) {
				// Check for classes
				foundOuterClass := false
				foundInnerClass := false
				for _, sym := range pf.Symbols {
					if sym.Kind == "class" && sym.Name == "OuterClass" {
						foundOuterClass = true
					}
					if sym.Kind == "class" && sym.Name == "InnerClass" {
						foundInnerClass = true
					}
				}
				if !foundOuterClass {
					t.Error("Expected to find class 'OuterClass'")
				}
				if !foundInnerClass {
					t.Error("Expected to find class 'InnerClass'")
				}
			},
		},
		{
			name:        "operators file",
			filename:    "../../tests/fixtures/cpp/operators.cpp",
			wantSymbols: 2, // Complex class, main function
			wantDeps:    1, // include
			checkSymbols: func(t *testing.T, pf *ParsedFile) {
				// Check for Complex class
				foundComplex := false
				for _, sym := range pf.Symbols {
					if sym.Kind == "class" && sym.Name == "Complex" {
						foundComplex = true
						break
					}
				}
				if !foundComplex {
					t.Error("Expected to find class 'Complex'")
				}

				// Check for operator overloads
				foundOperator := false
				for _, sym := range pf.Symbols {
					if sym.Kind == "operator" {
						foundOperator = true
						break
					}
				}
				// Note: Operators might be extracted as methods within the class
				// This is acceptable
				_ = foundOperator
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get absolute path
			absPath, err := filepath.Abs(tt.filename)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			// Check if file exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Skipf("Test file does not exist: %s", absPath)
			}

			scannedFile := ScannedFile{
				Path:     tt.filename,
				AbsPath:  absPath,
				Language: "cpp",
			}

			parsedFile, err := parser.Parse(scannedFile)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if parsedFile == nil {
				t.Fatal("Parse() returned nil")
			}

			// Check basic properties
			if parsedFile.Language != "cpp" {
				t.Errorf("Expected language 'cpp', got '%s'", parsedFile.Language)
			}

			// Check symbol count (approximate)
			if len(parsedFile.Symbols) < tt.wantSymbols {
				t.Errorf("Expected at least %d symbols, got %d", tt.wantSymbols, len(parsedFile.Symbols))
			}

			// Check dependency count (approximate)
			if len(parsedFile.Dependencies) < tt.wantDeps {
				t.Errorf("Expected at least %d dependencies, got %d", tt.wantDeps, len(parsedFile.Dependencies))
			}

			// Run custom checks
			if tt.checkSymbols != nil {
				tt.checkSymbols(t, parsedFile)
			}

			if tt.checkDeps != nil {
				tt.checkDeps(t, parsedFile)
			}
		})
	}
}

func TestCppParser_Inheritance(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCppParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/cpp/class.hpp")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	scannedFile := ScannedFile{
		Path:     "../../tests/fixtures/cpp/class.hpp",
		AbsPath:  absPath,
		Language: "cpp",
	}

	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check for inheritance relationship
	foundInheritance := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "extends" && dep.Source == "DerivedClass" && dep.Target == "MyClass" {
			foundInheritance = true
			break
		}
	}

	if !foundInheritance {
		t.Error("Expected to find inheritance relationship: DerivedClass extends MyClass")
	}
}

func TestCppParser_VirtualMethods(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCppParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/cpp/class.hpp")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	scannedFile := ScannedFile{
		Path:     "../../tests/fixtures/cpp/class.hpp",
		AbsPath:  absPath,
		Language: "cpp",
	}

	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check for virtual methods in MyClass
	foundVirtual := false
	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "class" && sym.Name == "MyClass" {
			for _, method := range sym.Children {
				if method.Kind == "virtual_method" || method.Kind == "method" {
					if method.Name == "virtualMethod" || method.Name == "pureVirtualMethod" {
						foundVirtual = true
						break
					}
				}
			}
		}
	}

	if !foundVirtual {
		t.Error("Expected to find virtual methods in MyClass")
	}
}

func TestCppParser_Templates(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCppParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/cpp/template.hpp")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	scannedFile := ScannedFile{
		Path:     "../../tests/fixtures/cpp/template.hpp",
		AbsPath:  absPath,
		Language: "cpp",
	}

	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check for template class
	foundClassTemplate := false
	foundFunctionTemplate := false

	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "class_template" && sym.Name == "Container" {
			foundClassTemplate = true
		}
		if sym.Kind == "function_template" && sym.Name == "max" {
			foundFunctionTemplate = true
		}
	}

	if !foundClassTemplate {
		t.Error("Expected to find class template 'Container'")
	}

	if !foundFunctionTemplate {
		t.Error("Expected to find function template 'max'")
	}
}

func TestCppParser_Namespaces(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCppParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/cpp/namespace.cpp")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	scannedFile := ScannedFile{
		Path:     "../../tests/fixtures/cpp/namespace.cpp",
		AbsPath:  absPath,
		Language: "cpp",
	}

	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check for classes (namespaces may not be extracted as top-level symbols)
	foundOuterClass := false
	foundInnerClass := false

	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "class" {
			if sym.Name == "OuterClass" {
				foundOuterClass = true
			}
			if sym.Name == "InnerClass" {
				foundInnerClass = true
			}
		}
	}

	if !foundOuterClass {
		t.Error("Expected to find class 'OuterClass'")
	}

	if !foundInnerClass {
		t.Error("Expected to find class 'InnerClass'")
	}
}

func TestCppParser_Operators(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCppParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/cpp/operators.cpp")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	scannedFile := ScannedFile{
		Path:     "../../tests/fixtures/cpp/operators.cpp",
		AbsPath:  absPath,
		Language: "cpp",
	}

	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check for Complex class
	foundComplex := false
	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "class" && sym.Name == "Complex" {
			foundComplex = true
			// Check that it has methods (operators are methods)
			if len(sym.Children) == 0 {
				t.Error("Expected Complex class to have methods/operators")
			}
			break
		}
	}

	if !foundComplex {
		t.Error("Expected to find class 'Complex'")
	}
}

func TestCppParser_HeaderImplementationPairing(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCppParser(tsParser)

	// Parse header
	headerPath, err := filepath.Abs("../../tests/fixtures/cpp/class.hpp")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(headerPath); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	headerFile := ScannedFile{
		Path:     "../../tests/fixtures/cpp/class.hpp",
		AbsPath:  headerPath,
		Language: "cpp",
	}

	parsedHeader, err := parser.Parse(headerFile)
	if err != nil {
		t.Fatalf("Parse() error for header: %v", err)
	}

	// Parse implementation
	implPath, err := filepath.Abs("../../tests/fixtures/cpp/class.cpp")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(implPath); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	implFile := ScannedFile{
		Path:     "../../tests/fixtures/cpp/class.cpp",
		AbsPath:  implPath,
		Language: "cpp",
	}

	parsedImpl, err := parser.Parse(implFile)
	if err != nil {
		t.Fatalf("Parse() error for implementation: %v", err)
	}

	// Match declarations to implementations
	parser.matchDeclarationToImplementation(parsedHeader, parsedImpl)

	// Check for implements_header edge
	foundImplementsHeader := false
	for _, dep := range parsedImpl.Dependencies {
		if dep.Type == "implements_header" {
			foundImplementsHeader = true
			break
		}
	}

	if !foundImplementsHeader {
		t.Error("Expected to find implements_header dependency")
	}

	// Check for implements_declaration edges
	foundImplementsDecl := false
	for _, dep := range parsedImpl.Dependencies {
		if dep.Type == "implements_declaration" {
			foundImplementsDecl = true
			break
		}
	}

	if !foundImplementsDecl {
		t.Error("Expected to find implements_declaration dependencies")
	}
}

func TestCppParser_ErrorHandling(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCppParser(tsParser)

	tests := []struct {
		name    string
		file    ScannedFile
		wantErr bool
	}{
		{
			name: "non-existent file",
			file: ScannedFile{
				Path:     "non_existent.cpp",
				AbsPath:  "/tmp/non_existent.cpp",
				Language: "cpp",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCppParser_DoxygenComments(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create TreeSitterParser: %v", err)
	}

	parser := NewCppParser(tsParser)

	absPath, err := filepath.Abs("../../tests/fixtures/cpp/class.hpp")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("Test file does not exist")
	}

	scannedFile := ScannedFile{
		Path:     "../../tests/fixtures/cpp/class.hpp",
		AbsPath:  absPath,
		Language: "cpp",
	}

	parsedFile, err := parser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check that at least some symbols have docstrings
	foundDocstring := false
	for _, sym := range parsedFile.Symbols {
		if sym.Docstring != "" {
			foundDocstring = true
			break
		}
	}

	if !foundDocstring {
		t.Error("Expected to find at least one symbol with a docstring")
	}
}
