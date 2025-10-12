package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoParser_Parse(t *testing.T) {
	// Initialize Tree-sitter parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	tests := []struct {
		name          string
		code          string
		wantSymbols   int
		wantFunctions int
		wantStructs   int
		wantError     bool
	}{
		{
			name: "simple function",
			code: `package main

func Hello() string {
	return "hello"
}`,
			wantSymbols:   2, // package + function
			wantFunctions: 1,
			wantStructs:   0,
			wantError:     false,
		},
		{
			name: "function with parameters and return type",
			code: `package main

func Add(a int, b int) int {
	return a + b
}`,
			wantSymbols:   2,
			wantFunctions: 1,
			wantStructs:   0,
			wantError:     false,
		},
		{
			name: "struct definition",
			code: `package main

type Person struct {
	Name string
	Age  int
}
`,
			wantSymbols:   2, // package + struct
			wantFunctions: 0,
			wantStructs:   1,
			wantError:     false,
		},
		{
			name: "interface definition",
			code: `package main

type Reader interface {
	Read(p []byte) (n int, err error)
}
`,
			wantSymbols:   2, // package + interface
			wantFunctions: 0,
			wantStructs:   0,
			wantError:     false,
		},
		{
			name: "method with receiver",
			code: `package main

type Person struct {
	Name string
}

func (p *Person) GetName() string {
	return p.Name
}`,
			wantSymbols:   3, // package + struct + method
			wantFunctions: 0,
			wantStructs:   1,
			wantError:     false,
		},
		{
			name: "syntax error",
			code: `package main

func broken( {
	return
}`,
			wantSymbols: 2, // package + broken function (partial parse)
			wantFunctions: 1, // The broken function is still extracted
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			scannedFile := ScannedFile{
				Path:     "test.go",
				AbsPath:  tmpFile,
				Language: "go",
				Size:     int64(len(tt.code)),
			}

			parsedFile, err := goParser.Parse(scannedFile)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				// Even with errors, we should get partial results
				if parsedFile == nil {
					t.Errorf("Expected partial results even with error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if parsedFile == nil {
				return
			}

			if len(parsedFile.Symbols) != tt.wantSymbols {
				t.Errorf("Expected %d symbols, got %d", tt.wantSymbols, len(parsedFile.Symbols))
			}

			// Count specific symbol types
			funcCount := 0
			structCount := 0
			for _, sym := range parsedFile.Symbols {
				if sym.Kind == "function" {
					funcCount++
				}
				if sym.Kind == "struct" {
					structCount++
				}
			}

			if funcCount != tt.wantFunctions {
				t.Errorf("Expected %d functions, got %d", tt.wantFunctions, funcCount)
			}

			if structCount != tt.wantStructs {
				t.Errorf("Expected %d structs, got %d", tt.wantStructs, structCount)
			}
		})
	}
}

func TestGoParser_ExtractFunctions(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	code := `package main

// Hello returns a greeting message
func Hello(name string) string {
	return "Hello, " + name
}

// Add adds two numbers
func Add(a, b int) int {
	return a + b
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.go",
		AbsPath:  tmpFile,
		Language: "go",
		Size:     int64(len(code)),
	}

	parsedFile, err := goParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 2 functions
	funcCount := 0
	var helloFunc, addFunc *ParsedSymbol
	
	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "function" {
			funcCount++
			if sym.Name == "Hello" {
				helloFunc = sym
			} else if sym.Name == "Add" {
				addFunc = sym
			}
		}
	}

	if funcCount != 2 {
		t.Errorf("Expected 2 functions, got %d", funcCount)
	}

	// Check Hello function
	if helloFunc == nil {
		t.Fatal("Hello function not found")
	}
	if helloFunc.Name != "Hello" {
		t.Errorf("Expected function name 'Hello', got '%s'", helloFunc.Name)
	}
	if helloFunc.Docstring == "" {
		t.Errorf("Expected docstring for Hello function")
	}

	// Check Add function
	if addFunc == nil {
		t.Fatal("Add function not found")
	}
	if addFunc.Name != "Add" {
		t.Errorf("Expected function name 'Add', got '%s'", addFunc.Name)
	}
}

func TestGoParser_ExtractStructs(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	code := `package main

// Person represents a person
type Person struct {
	Name string
	Age  int
	Email string
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.go",
		AbsPath:  tmpFile,
		Language: "go",
		Size:     int64(len(code)),
	}

	parsedFile, err := goParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find the struct
	var personStruct *ParsedSymbol
	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "struct" && sym.Name == "Person" {
			personStruct = sym
			break
		}
	}

	if personStruct == nil {
		t.Fatal("Person struct not found")
	}

	if personStruct.Name != "Person" {
		t.Errorf("Expected struct name 'Person', got '%s'", personStruct.Name)
	}

	if personStruct.Docstring == "" {
		t.Errorf("Expected docstring for Person struct")
	}

	// Check fields
	if len(personStruct.Children) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(personStruct.Children))
	}

	expectedFields := map[string]bool{
		"Name":  false,
		"Age":   false,
		"Email": false,
	}

	for _, field := range personStruct.Children {
		if field.Kind != "field" {
			t.Errorf("Expected field kind, got '%s'", field.Kind)
		}
		if _, exists := expectedFields[field.Name]; exists {
			expectedFields[field.Name] = true
		}
	}

	for fieldName, found := range expectedFields {
		if !found {
			t.Errorf("Field '%s' not found", fieldName)
		}
	}
}

func TestGoParser_ExtractInterfaces(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	code := `package main

// Reader is an interface for reading
type Reader interface {
	Read(p []byte) (n int, err error)
	Close() error
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.go",
		AbsPath:  tmpFile,
		Language: "go",
		Size:     int64(len(code)),
	}

	parsedFile, err := goParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find the interface
	var readerInterface *ParsedSymbol
	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "interface" && sym.Name == "Reader" {
			readerInterface = sym
			break
		}
	}

	if readerInterface == nil {
		t.Fatal("Reader interface not found")
	}

	if readerInterface.Name != "Reader" {
		t.Errorf("Expected interface name 'Reader', got '%s'", readerInterface.Name)
	}

	// Check methods
	if len(readerInterface.Children) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(readerInterface.Children))
	}

	expectedMethods := map[string]bool{
		"Read":  false,
		"Close": false,
	}

	for _, method := range readerInterface.Children {
		if method.Kind != "method" {
			t.Errorf("Expected method kind, got '%s'", method.Kind)
		}
		if _, exists := expectedMethods[method.Name]; exists {
			expectedMethods[method.Name] = true
		}
	}

	for methodName, found := range expectedMethods {
		if !found {
			t.Errorf("Method '%s' not found", methodName)
		}
	}
}

func TestGoParser_ExtractMethods(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	code := `package main

type Person struct {
	Name string
}

// GetName returns the person's name
func (p *Person) GetName() string {
	return p.Name
}

// SetName sets the person's name
func (p *Person) SetName(name string) {
	p.Name = name
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.go",
		AbsPath:  tmpFile,
		Language: "go",
		Size:     int64(len(code)),
	}

	parsedFile, err := goParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Count methods
	methodCount := 0
	var getNameMethod, setNameMethod *ParsedSymbol
	
	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "method" {
			methodCount++
			if sym.Name == "GetName" {
				getNameMethod = sym
			} else if sym.Name == "SetName" {
				setNameMethod = sym
			}
		}
	}

	if methodCount != 2 {
		t.Errorf("Expected 2 methods, got %d", methodCount)
	}

	// Check GetName method
	if getNameMethod == nil {
		t.Fatal("GetName method not found")
	}
	if getNameMethod.Name != "GetName" {
		t.Errorf("Expected method name 'GetName', got '%s'", getNameMethod.Name)
	}
	if getNameMethod.Docstring == "" {
		t.Errorf("Expected docstring for GetName method")
	}

	// Check SetName method
	if setNameMethod == nil {
		t.Fatal("SetName method not found")
	}
	if setNameMethod.Name != "SetName" {
		t.Errorf("Expected method name 'SetName', got '%s'", setNameMethod.Name)
	}
}

func TestGoParser_ExtractImports(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	code := `package main

import (
	"fmt"
	"strings"
	"github.com/example/pkg"
)

func main() {
	fmt.Println("Hello")
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.go",
		AbsPath:  tmpFile,
		Language: "go",
		Size:     int64(len(code)),
	}

	parsedFile, err := goParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check imports (filter only import type dependencies)
	importDeps := []ParsedDependency{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			importDeps = append(importDeps, dep)
		}
	}

	if len(importDeps) != 3 {
		t.Errorf("Expected 3 imports, got %d", len(importDeps))
	}

	expectedImports := map[string]bool{
		"fmt":                   false,
		"strings":               false,
		"github.com/example/pkg": false,
	}

	for _, dep := range importDeps {
		if _, exists := expectedImports[dep.Target]; exists {
			expectedImports[dep.Target] = true
		}
	}

	for importPath, found := range expectedImports {
		if !found {
			t.Errorf("Import '%s' not found", importPath)
		}
	}
}

func TestGoParser_ErrorHandling(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "missing closing brace",
			code: `package main

func broken() {
	return
`,
			wantErr: true,
		},
		{
			name: "invalid syntax",
			code: `package main

func ( {
}`,
			wantErr: true,
		},
		{
			name: "incomplete struct",
			code: `package main

type Person struct {
	Name string
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			scannedFile := ScannedFile{
				Path:     "test.go",
				AbsPath:  tmpFile,
				Language: "go",
				Size:     int64(len(tt.code)),
			}

			parsedFile, err := goParser.Parse(scannedFile)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}

			// Even with errors, we should get partial results
			if parsedFile == nil {
				t.Errorf("Expected partial results even with error")
			}

			// Should at least have the package declaration
			if parsedFile != nil && len(parsedFile.Symbols) == 0 {
				t.Errorf("Expected at least package symbol in partial results")
			}
		})
	}
}

func TestGoParser_ExtractCallRelationships(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	code := `package main

import "fmt"

func helper() string {
	return "helper"
}

func caller() {
	result := helper()
	fmt.Println(result)
}

type Service struct{}

func (s *Service) Process() {
	s.Validate()
}

func (s *Service) Validate() bool {
	return true
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.go",
		AbsPath:  tmpFile,
		Language: "go",
		Size:     int64(len(code)),
	}

	parsedFile, err := goParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check for call dependencies
	callDeps := []ParsedDependency{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			callDeps = append(callDeps, dep)
		}
	}

	if len(callDeps) == 0 {
		t.Error("Expected call dependencies but found none")
	}

	// Check for specific call relationships
	expectedCalls := map[string][]string{
		"caller":  {"helper", "fmt.Println"},
		"Process": {"s.Validate"},
	}

	foundCalls := make(map[string]map[string]bool)
	for _, dep := range callDeps {
		if dep.Type == "call" {
			if foundCalls[dep.Source] == nil {
				foundCalls[dep.Source] = make(map[string]bool)
			}
			foundCalls[dep.Source][dep.Target] = true
		}
	}

	for caller, expectedTargets := range expectedCalls {
		if foundCalls[caller] == nil {
			t.Errorf("No calls found from %s", caller)
			continue
		}
		for _, target := range expectedTargets {
			if !foundCalls[caller][target] {
				t.Errorf("Expected call from %s to %s not found", caller, target)
			}
		}
	}
}

func TestGoParser_ComplexExample(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	goParser := NewGoParser(tsParser)

	code := `package main

import (
	"fmt"
	"strings"
)

// User represents a user in the system
type User struct {
	ID    int
	Name  string
	Email string
}

// UserService provides user-related operations
type UserService interface {
	GetUser(id int) (*User, error)
	CreateUser(user *User) error
}

// NewUser creates a new user
func NewUser(name, email string) *User {
	return &User{
		Name:  name,
		Email: email,
	}
}

// GetName returns the user's name
func (u *User) GetName() string {
	return u.Name
}

// SetName sets the user's name
func (u *User) SetName(name string) {
	u.Name = strings.TrimSpace(name)
}

func main() {
	user := NewUser("John", "john@example.com")
	fmt.Println(user.GetName())
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.go",
		AbsPath:  tmpFile,
		Language: "go",
		Size:     int64(len(code)),
	}

	parsedFile, err := goParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Count symbol types
	counts := make(map[string]int)
	for _, sym := range parsedFile.Symbols {
		counts[sym.Kind]++
	}

	// Expected: 1 package, 1 struct, 1 interface, 3 functions, 2 methods
	expected := map[string]int{
		"package":   1,
		"struct":    1,
		"interface": 1,
		"function":  2, // NewUser, main
		"method":    2, // GetName, SetName
	}

	for kind, expectedCount := range expected {
		if counts[kind] != expectedCount {
			t.Errorf("Expected %d %s symbols, got %d", expectedCount, kind, counts[kind])
		}
	}

	// Check imports (filter only import type dependencies)
	importDeps := []ParsedDependency{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			importDeps = append(importDeps, dep)
		}
	}

	if len(importDeps) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(importDeps))
	}

	// Verify struct has fields
	var userStruct *ParsedSymbol
	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "struct" && sym.Name == "User" {
			userStruct = sym
			break
		}
	}

	if userStruct == nil {
		t.Fatal("User struct not found")
	}

	if len(userStruct.Children) != 3 {
		t.Errorf("Expected 3 fields in User struct, got %d", len(userStruct.Children))
	}

	// Verify interface has methods
	var userServiceInterface *ParsedSymbol
	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "interface" && sym.Name == "UserService" {
			userServiceInterface = sym
			break
		}
	}

	if userServiceInterface == nil {
		t.Fatal("UserService interface not found")
	}

	if len(userServiceInterface.Children) != 2 {
		t.Errorf("Expected 2 methods in UserService interface, got %d", len(userServiceInterface.Children))
	}
}
