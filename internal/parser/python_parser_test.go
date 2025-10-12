package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonParser_Parse(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewPythonParser(tsParser)

	tests := []struct {
		name          string
		code          string
		wantSymbols   int
		wantFunctions int
		wantClasses   int
		wantImports   int
	}{
		{
			name: "simple function",
			code: `def hello():
    """Say hello"""
    print("Hello")
`,
			wantSymbols:   1,
			wantFunctions: 1,
			wantClasses:   0,
			wantImports:   0,
		},
		{
			name: "function with type hints",
			code: `def add(a: int, b: int) -> int:
    """Add two numbers"""
    return a + b
`,
			wantSymbols:   1,
			wantFunctions: 1,
			wantClasses:   0,
			wantImports:   0,
		},
		{
			name: "async function",
			code: `async def fetch_data():
    """Fetch data asynchronously"""
    return await get_data()
`,
			wantSymbols:   1,
			wantFunctions: 1,
			wantClasses:   0,
			wantImports:   0,
		},
		{
			name: "simple class",
			code: `class Person:
    """A person class"""
    def __init__(self, name):
        self.name = name
    
    def greet(self):
        return f"Hello, {self.name}"
`,
			wantSymbols:   1,
			wantFunctions: 0,
			wantClasses:   1,
			wantImports:   0,
		},
		{
			name: "class with inheritance",
			code: `class Employee(Person):
    """An employee class"""
    def __init__(self, name, employee_id):
        self.name = name
        self.employee_id = employee_id
`,
			wantSymbols:   1,
			wantFunctions: 0,
			wantClasses:   1,
			wantImports:   0,
		},
		{
			name: "imports",
			code: `import os
import sys
from typing import List, Dict
from pathlib import Path
`,
			wantSymbols:   0,
			wantFunctions: 0,
			wantClasses:   0,
			wantImports:   4,
		},
		{
			name: "decorated function",
			code: `@decorator
def decorated_func():
    """A decorated function"""
    pass
`,
			wantSymbols:   1,
			wantFunctions: 1,
			wantClasses:   0,
			wantImports:   0,
		},
		{
			name: "class with decorators",
			code: `@dataclass
class Config:
    """Configuration class"""
    name: str
    value: int
`,
			wantSymbols:   1,
			wantFunctions: 0,
			wantClasses:   1,
			wantImports:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.py")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			file := ScannedFile{
				Path:     "test.py",
				AbsPath:  tmpFile,
				Language: "python",
			}

			result, err := parser.Parse(file)
			if err != nil {
				t.Logf("Parse returned error (may be expected): %v", err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			// Count symbols by type
			functions := 0
			classes := 0
			for _, sym := range result.Symbols {
				switch sym.Kind {
				case "function", "async_function":
					functions++
				case "class":
					classes++
				}
			}

			if len(result.Symbols) != tt.wantSymbols {
				t.Errorf("Got %d symbols, want %d", len(result.Symbols), tt.wantSymbols)
				for i, sym := range result.Symbols {
					t.Logf("Symbol %d: %s (%s)", i, sym.Name, sym.Kind)
				}
			}

			if functions != tt.wantFunctions {
				t.Errorf("Got %d functions, want %d", functions, tt.wantFunctions)
			}

			if classes != tt.wantClasses {
				t.Errorf("Got %d classes, want %d", classes, tt.wantClasses)
			}

			// Count only import dependencies
			imports := 0
			for _, dep := range result.Dependencies {
				if dep.Type == "import" {
					imports++
				}
			}
			
			if imports != tt.wantImports {
				t.Errorf("Got %d imports, want %d", imports, tt.wantImports)
				for i, dep := range result.Dependencies {
					t.Logf("Dependency %d: type=%s, source=%s, target=%s", i, dep.Type, dep.Source, dep.Target)
				}
			}
		})
	}
}

func TestPythonParser_ExtractFunctions(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewPythonParser(tsParser)

	code := `def simple_func():
    """A simple function"""
    pass

def func_with_params(a, b, c):
    """Function with parameters"""
    return a + b + c

def func_with_types(name: str, age: int) -> str:
    """Function with type hints"""
    return f"{name} is {age} years old"

async def async_func():
    """An async function"""
    await something()

@decorator
def decorated_func():
    """A decorated function"""
    pass

@decorator1
@decorator2
def multi_decorated():
    """Multiple decorators"""
    pass
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	file := ScannedFile{
		Path:     "test.py",
		AbsPath:  tmpFile,
		Language: "python",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Logf("Parse returned error (may be expected): %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check we found all functions
	expectedFuncs := map[string]string{
		"simple_func":      "function",
		"func_with_params": "function",
		"func_with_types":  "function",
		"async_func":       "async_function",
		"decorated_func":   "function",
		"multi_decorated":  "function",
	}

	foundFuncs := make(map[string]string)
	for _, sym := range result.Symbols {
		if sym.Kind == "function" || sym.Kind == "async_function" {
			foundFuncs[sym.Name] = sym.Kind
		}
	}

	for name, expectedKind := range expectedFuncs {
		kind, found := foundFuncs[name]
		if !found {
			t.Errorf("Function %s not found", name)
		} else if kind != expectedKind {
			t.Errorf("Function %s has kind %s, want %s", name, kind, expectedKind)
		}
	}

	// Check docstrings
	for _, sym := range result.Symbols {
		if sym.Name == "simple_func" {
			if sym.Docstring != "A simple function" {
				t.Errorf("simple_func docstring = %q, want %q", sym.Docstring, "A simple function")
			}
		}
	}

	// Check signatures contain type hints
	for _, sym := range result.Symbols {
		if sym.Name == "func_with_types" {
			if !containsString(sym.Signature, "str") || !containsString(sym.Signature, "int") {
				t.Errorf("func_with_types signature missing type hints: %s", sym.Signature)
			}
		}
	}

	// Check decorators in signature
	for _, sym := range result.Symbols {
		if sym.Name == "decorated_func" {
			if !containsString(sym.Signature, "@decorator") {
				t.Errorf("decorated_func signature missing decorator: %s", sym.Signature)
			}
		}
	}
}

func TestPythonParser_ExtractClasses(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewPythonParser(tsParser)

	code := `class SimpleClass:
    """A simple class"""
    pass

class Person:
    """A person class"""
    def __init__(self, name):
        self.name = name
    
    def greet(self):
        """Greet someone"""
        return f"Hello, {self.name}"
    
    @staticmethod
    def static_method():
        """A static method"""
        pass
    
    @classmethod
    def class_method(cls):
        """A class method"""
        pass
    
    async def async_method(self):
        """An async method"""
        await something()

class Employee(Person):
    """An employee inherits from Person"""
    def __init__(self, name, employee_id):
        super().__init__(name)
        self.employee_id = employee_id

class Manager(Employee, Leader):
    """Multiple inheritance"""
    pass

@dataclass
class Config:
    """A decorated class"""
    name: str
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	file := ScannedFile{
		Path:     "test.py",
		AbsPath:  tmpFile,
		Language: "python",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Logf("Parse returned error (may be expected): %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check we found all classes
	expectedClasses := []string{"SimpleClass", "Person", "Employee", "Manager", "Config"}
	foundClasses := make(map[string]bool)
	
	for _, sym := range result.Symbols {
		if sym.Kind == "class" {
			foundClasses[sym.Name] = true
		}
	}

	for _, name := range expectedClasses {
		if !foundClasses[name] {
			t.Errorf("Class %s not found", name)
		}
	}

	// Check Person class has methods
	for _, sym := range result.Symbols {
		if sym.Name == "Person" {
			if len(sym.Children) == 0 {
				t.Error("Person class should have methods")
			}
			
			// Check for specific methods
			methodNames := make(map[string]string)
			for _, child := range sym.Children {
				methodNames[child.Name] = child.Kind
			}
			
			expectedMethods := map[string]string{
				"__init__":      "method",
				"greet":         "method",
				"static_method": "static_method",
				"class_method":  "class_method",
				"async_method":  "async_method",
			}
			
			for name, expectedKind := range expectedMethods {
				kind, found := methodNames[name]
				if !found {
					t.Errorf("Method %s not found in Person class", name)
				} else if kind != expectedKind {
					t.Errorf("Method %s has kind %s, want %s", name, kind, expectedKind)
				}
			}
		}
	}

	// Check inheritance
	inheritanceFound := false
	for _, dep := range result.Dependencies {
		if dep.Type == "extends" && dep.Source == "Employee" && dep.Target == "Person" {
			inheritanceFound = true
			break
		}
	}
	if !inheritanceFound {
		t.Error("Employee -> Person inheritance not found")
	}

	// Check class signatures
	for _, sym := range result.Symbols {
		if sym.Name == "Employee" {
			if !containsString(sym.Signature, "Person") {
				t.Errorf("Employee signature should mention Person: %s", sym.Signature)
			}
		}
		if sym.Name == "Manager" {
			if !containsString(sym.Signature, "Employee") || !containsString(sym.Signature, "Leader") {
				t.Errorf("Manager signature should mention both base classes: %s", sym.Signature)
			}
		}
	}

	// Check docstrings
	for _, sym := range result.Symbols {
		if sym.Name == "Person" {
			if sym.Docstring != "A person class" {
				t.Errorf("Person docstring = %q, want %q", sym.Docstring, "A person class")
			}
		}
	}

	// Check decorator in signature
	for _, sym := range result.Symbols {
		if sym.Name == "Config" {
			if !containsString(sym.Signature, "@dataclass") {
				t.Errorf("Config signature missing decorator: %s", sym.Signature)
			}
		}
	}
}

func TestPythonParser_ExtractImports(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewPythonParser(tsParser)

	code := `import os
import sys
import json

from typing import List, Dict, Optional
from pathlib import Path
from .local import something
from ..parent import other
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	file := ScannedFile{
		Path:     "test.py",
		AbsPath:  tmpFile,
		Language: "python",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Logf("Parse returned error (may be expected): %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check imports
	expectedImports := []string{"os", "sys", "json", "typing", "pathlib"}
	foundImports := make(map[string]bool)
	
	for _, dep := range result.Dependencies {
		if dep.Type == "import" {
			foundImports[dep.TargetModule] = true
		}
	}

	for _, imp := range expectedImports {
		if !foundImports[imp] {
			t.Errorf("Import %s not found", imp)
		}
	}

	if len(result.Dependencies) < 5 {
		t.Errorf("Expected at least 5 imports, got %d", len(result.Dependencies))
	}
}

func TestPythonParser_ExtractDocstrings(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewPythonParser(tsParser)

	code := `"""
This is a module docstring.
It can span multiple lines.
"""

def func_with_docstring():
    """This is a function docstring"""
    pass

def func_with_multiline_docstring():
    """
    This is a multiline docstring.
    
    It has multiple paragraphs.
    """
    pass

class ClassWithDocstring:
    """This is a class docstring"""
    
    def method_with_docstring(self):
        """This is a method docstring"""
        pass
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	file := ScannedFile{
		Path:     "test.py",
		AbsPath:  tmpFile,
		Language: "python",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Logf("Parse returned error (may be expected): %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check module docstring
	moduleDocFound := false
	for _, sym := range result.Symbols {
		if sym.Kind == "module" {
			moduleDocFound = true
			if !containsString(sym.Docstring, "module docstring") {
				t.Errorf("Module docstring incorrect: %s", sym.Docstring)
			}
		}
	}
	if !moduleDocFound {
		t.Error("Module docstring not found")
	}

	// Check function docstrings
	for _, sym := range result.Symbols {
		if sym.Name == "func_with_docstring" {
			if sym.Docstring != "This is a function docstring" {
				t.Errorf("func_with_docstring docstring = %q", sym.Docstring)
			}
		}
		if sym.Name == "func_with_multiline_docstring" {
			if !containsString(sym.Docstring, "multiline docstring") {
				t.Errorf("func_with_multiline_docstring docstring = %q", sym.Docstring)
			}
		}
	}

	// Check class and method docstrings
	for _, sym := range result.Symbols {
		if sym.Name == "ClassWithDocstring" {
			if sym.Docstring != "This is a class docstring" {
				t.Errorf("ClassWithDocstring docstring = %q", sym.Docstring)
			}
			
			// Check method docstring
			for _, method := range sym.Children {
				if method.Name == "method_with_docstring" {
					if method.Docstring != "This is a method docstring" {
						t.Errorf("method_with_docstring docstring = %q", method.Docstring)
					}
				}
			}
		}
	}
}

func TestPythonParser_ExtractCallRelationships(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewPythonParser(tsParser)

	code := `def helper():
    return "helper"

def caller():
    result = helper()
    print(result)

class Service:
    def process(self):
        self.validate()
        helper()
    
    def validate(self):
        return True

async def async_caller():
    result = await helper()
    print(result)
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	file := ScannedFile{
		Path:     "test.py",
		AbsPath:  tmpFile,
		Language: "python",
	}

	result, err := parser.Parse(file)
	if err != nil {
		t.Logf("Parse returned error (may be expected): %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check for call dependencies
	callDeps := []ParsedDependency{}
	for _, dep := range result.Dependencies {
		if dep.Type == "call" {
			callDeps = append(callDeps, dep)
		}
	}

	if len(callDeps) == 0 {
		t.Error("Expected call dependencies but found none")
	}

	// Check for specific call relationships
	foundCalls := make(map[string]map[string]bool)
	for _, dep := range callDeps {
		if dep.Type == "call" {
			if foundCalls[dep.Source] == nil {
				foundCalls[dep.Source] = make(map[string]bool)
			}
			foundCalls[dep.Source][dep.Target] = true
		}
	}

	// Verify some expected calls exist
	expectedCallers := []string{"caller", "process", "async_caller"}
	for _, caller := range expectedCallers {
		if foundCalls[caller] == nil {
			t.Errorf("No calls found from %s", caller)
		}
	}

	// Check for specific call from caller to helper
	if foundCalls["caller"] != nil && !foundCalls["caller"]["helper"] {
		t.Error("Expected call from caller to helper not found")
	}

	// Check for method call from process to validate
	if foundCalls["process"] != nil && !foundCalls["process"]["self.validate"] {
		t.Error("Expected call from process to self.validate not found")
	}
}

func TestPythonParser_ErrorHandling(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	parser := NewPythonParser(tsParser)

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "syntax error - incomplete function",
			code: `def incomplete_func(
    pass
`,
			wantErr: true,
		},
		{
			name: "syntax error - missing colon",
			code: `def func()
    pass
`,
			wantErr: true,
		},
		{
			name: "valid code",
			code: `def valid_func():
    pass
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.py")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			file := ScannedFile{
				Path:     "test.py",
				AbsPath:  tmpFile,
				Language: "python",
			}

			result, err := parser.Parse(file)
			
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Should still return partial results even on error
			if result == nil {
				t.Error("Expected non-nil result even on error")
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		len(s) > len(substr)+1 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
