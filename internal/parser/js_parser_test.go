package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSParser_Parse(t *testing.T) {
	// Initialize Tree-sitter parser
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	tests := []struct {
		name          string
		code          string
		language      string
		wantSymbols   int
		wantFunctions int
		wantClasses   int
		wantError     bool
	}{
		{
			name: "simple function",
			code: `function hello() {
	return "hello";
}`,
			language:      "javascript",
			wantSymbols:   2, // module + function
			wantFunctions: 1,
			wantClasses:   0,
			wantError:     false,
		},
		{
			name: "arrow function",
			code: `const add = (a, b) => {
	return a + b;
};`,
			language:      "javascript",
			wantSymbols:   2, // module + arrow function
			wantFunctions: 1,
			wantClasses:   0,
			wantError:     false,
		},
		{
			name: "class definition",
			code: `class Person {
	constructor(name) {
		this.name = name;
	}
	
	getName() {
		return this.name;
	}
}`,
			language:      "javascript",
			wantSymbols:   2, // module + class
			wantFunctions: 0,
			wantClasses:   1,
			wantError:     false,
		},
		{
			name: "async function",
			code: `async function fetchData() {
	const response = await fetch('/api/data');
	return response.json();
}`,
			language:      "javascript",
			wantSymbols:   2, // module + async function
			wantFunctions: 1,
			wantClasses:   0,
			wantError:     false,
		},
		{
			name: "syntax error",
			code: `function broken( {
	return
}`,
			language:    "javascript",
			wantSymbols: 1, // Still extracts module even with syntax errors
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			ext := ".js"
			if tt.language == "typescript" {
				ext = ".ts"
			}
			tmpFile := filepath.Join(tmpDir, "test"+ext)
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			scannedFile := ScannedFile{
				Path:     "test" + ext,
				AbsPath:  tmpFile,
				Language: tt.language,
				Size:     int64(len(tt.code)),
			}

			parsedFile, err := jsParser.Parse(scannedFile)

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
				for i, sym := range parsedFile.Symbols {
					t.Logf("Symbol %d: name=%s, kind=%s", i, sym.Name, sym.Kind)
				}
			}

			// Count specific symbol types
			funcCount := 0
			classCount := 0
			for _, sym := range parsedFile.Symbols {
				if sym.Kind == "function" || sym.Kind == "async_function" || sym.Kind == "arrow_function" || sym.Kind == "async_arrow_function" {
					funcCount++
				}
				if sym.Kind == "class" {
					classCount++
				}
			}

			if funcCount != tt.wantFunctions {
				t.Errorf("Expected %d functions, got %d", tt.wantFunctions, funcCount)
			}

			if classCount != tt.wantClasses {
				t.Errorf("Expected %d classes, got %d", tt.wantClasses, classCount)
			}
		})
	}
}

func TestJSParser_ExtractFunctions(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `/**
 * Returns a greeting message
 */
function hello(name) {
	return "Hello, " + name;
}

/**
 * Adds two numbers
 */
function add(a, b) {
	return a + b;
}

// Function expression
const multiply = function(a, b) {
	return a * b;
};
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.js",
		AbsPath:  tmpFile,
		Language: "javascript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 3 functions
	funcCount := 0
	var helloFunc, addFunc, multiplyFunc *ParsedSymbol

	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "function" {
			funcCount++
			if sym.Name == "hello" {
				helloFunc = sym
			} else if sym.Name == "add" {
				addFunc = sym
			} else if sym.Name == "multiply" {
				multiplyFunc = sym
			}
		}
	}

	if funcCount != 3 {
		t.Errorf("Expected 3 functions, got %d", funcCount)
	}

	// Check hello function
	if helloFunc == nil {
		t.Fatal("hello function not found")
	}
	if helloFunc.Name != "hello" {
		t.Errorf("Expected function name 'hello', got '%s'", helloFunc.Name)
	}
	if helloFunc.Docstring == "" {
		t.Errorf("Expected docstring for hello function")
	}

	// Check add function
	if addFunc == nil {
		t.Fatal("add function not found")
	}
	if addFunc.Name != "add" {
		t.Errorf("Expected function name 'add', got '%s'", addFunc.Name)
	}

	// Check multiply function
	if multiplyFunc == nil {
		t.Fatal("multiply function not found")
	}
	if multiplyFunc.Name != "multiply" {
		t.Errorf("Expected function name 'multiply', got '%s'", multiplyFunc.Name)
	}
}

func TestJSParser_ExtractArrowFunctions(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `/**
 * Arrow function with block body
 */
const add = (a, b) => {
	return a + b;
};

// Arrow function with expression body
const multiply = (a, b) => a * b;

// Async arrow function
const fetchData = async () => {
	const response = await fetch('/api/data');
	return response.json();
};
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.js",
		AbsPath:  tmpFile,
		Language: "javascript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Count arrow functions
	arrowCount := 0
	asyncArrowCount := 0
	var addFunc, multiplyFunc, fetchDataFunc *ParsedSymbol

	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "arrow_function" {
			arrowCount++
			if sym.Name == "add" {
				addFunc = sym
			} else if sym.Name == "multiply" {
				multiplyFunc = sym
			}
		} else if sym.Kind == "async_arrow_function" {
			asyncArrowCount++
			if sym.Name == "fetchData" {
				fetchDataFunc = sym
			}
		}
	}

	if arrowCount != 2 {
		t.Errorf("Expected 2 arrow functions, got %d", arrowCount)
	}

	if asyncArrowCount != 1 {
		t.Errorf("Expected 1 async arrow function, got %d", asyncArrowCount)
	}

	// Check add function
	if addFunc == nil {
		t.Fatal("add arrow function not found")
	}
	if addFunc.Docstring == "" {
		t.Errorf("Expected docstring for add function")
	}

	// Check multiply function
	if multiplyFunc == nil {
		t.Fatal("multiply arrow function not found")
	}

	// Check fetchData function
	if fetchDataFunc == nil {
		t.Fatal("fetchData async arrow function not found")
	}
}

func TestJSParser_ExtractClasses(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `/**
 * Represents a person
 */
class Person {
	constructor(name, age) {
		this.name = name;
		this.age = age;
	}

	/**
	 * Gets the person's name
	 */
	getName() {
		return this.name;
	}

	/**
	 * Sets the person's name
	 */
	setName(name) {
		this.name = name;
	}

	static create(name, age) {
		return new Person(name, age);
	}
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.js",
		AbsPath:  tmpFile,
		Language: "javascript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find the class
	var personClass *ParsedSymbol
	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "class" && sym.Name == "Person" {
			personClass = sym
			break
		}
	}

	if personClass == nil {
		t.Fatal("Person class not found")
	}

	if personClass.Name != "Person" {
		t.Errorf("Expected class name 'Person', got '%s'", personClass.Name)
	}

	if personClass.Docstring == "" {
		t.Errorf("Expected docstring for Person class")
	}

	// Check methods (constructor, getName, setName, create)
	if len(personClass.Children) < 3 {
		t.Errorf("Expected at least 3 methods, got %d", len(personClass.Children))
	}

	// Count method types
	methodCount := 0
	staticMethodCount := 0
	for _, child := range personClass.Children {
		if child.Kind == "method" {
			methodCount++
		} else if child.Kind == "static_method" {
			staticMethodCount++
		}
	}

	if methodCount < 2 {
		t.Errorf("Expected at least 2 regular methods, got %d", methodCount)
	}

	if staticMethodCount != 1 {
		t.Errorf("Expected 1 static method, got %d", staticMethodCount)
	}
}

func TestJSParser_ExtractES6Imports(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `import React from 'react';
import { useState, useEffect } from 'react';
import * as utils from './utils';
import './styles.css';

function App() {
	return <div>Hello</div>;
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.js",
		AbsPath:  tmpFile,
		Language: "javascript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check imports
	if len(parsedFile.Dependencies) != 4 {
		t.Errorf("Expected 4 imports, got %d", len(parsedFile.Dependencies))
	}

	expectedImports := map[string]bool{
		"react":        false,
		"./utils":      false,
		"./styles.css": false,
	}

	for _, dep := range parsedFile.Dependencies {
		if dep.Type != "import" {
			t.Errorf("Expected dependency type 'import', got '%s'", dep.Type)
		}
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

func TestJSParser_ExtractCommonJSRequire(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `const express = require('express');
const path = require('path');
const { readFile } = require('fs');

const app = express();
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.js",
		AbsPath:  tmpFile,
		Language: "javascript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check requires
	if len(parsedFile.Dependencies) != 3 {
		t.Errorf("Expected 3 requires, got %d", len(parsedFile.Dependencies))
	}

	expectedRequires := map[string]bool{
		"express": false,
		"path":    false,
		"fs":      false,
	}

	for _, dep := range parsedFile.Dependencies {
		if dep.Type != "import" {
			t.Errorf("Expected dependency type 'import', got '%s'", dep.Type)
		}
		if _, exists := expectedRequires[dep.Target]; exists {
			expectedRequires[dep.Target] = true
		}
	}

	for requirePath, found := range expectedRequires {
		if !found {
			t.Errorf("Require '%s' not found", requirePath)
		}
	}
}

func TestJSParser_TypeScriptAnnotations(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `interface User {
	name: string;
	age: number;
}

function greet(user: User): string {
	return "Hello, " + user.name;
}

const add = (a: number, b: number): number => {
	return a + b;
};

class Person {
	private name: string;
	
	constructor(name: string) {
		this.name = name;
	}
	
	getName(): string {
		return this.name;
	}
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.ts")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.ts",
		AbsPath:  tmpFile,
		Language: "typescript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have function, arrow function, and class
	funcCount := 0
	arrowCount := 0
	classCount := 0

	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "function" {
			funcCount++
		} else if sym.Kind == "arrow_function" {
			arrowCount++
		} else if sym.Kind == "class" {
			classCount++
		}
	}

	if funcCount != 1 {
		t.Errorf("Expected 1 function, got %d", funcCount)
	}

	if arrowCount != 1 {
		t.Errorf("Expected 1 arrow function, got %d", arrowCount)
	}

	if classCount != 1 {
		t.Errorf("Expected 1 class, got %d", classCount)
	}
}

func TestJSParser_AsyncFunctions(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `async function fetchUser(id) {
	const response = await fetch('/api/users/' + id);
	return response.json();
}

const getData = async () => {
	const data = await fetchUser(1);
	return data;
};

class API {
	async getUsers() {
		const response = await fetch('/api/users');
		return response.json();
	}
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.js",
		AbsPath:  tmpFile,
		Language: "javascript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Count async functions
	asyncFuncCount := 0
	asyncArrowCount := 0
	asyncMethodCount := 0

	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "async_function" {
			asyncFuncCount++
		} else if sym.Kind == "async_arrow_function" {
			asyncArrowCount++
		} else if sym.Kind == "class" {
			for _, child := range sym.Children {
				if child.Kind == "async_method" {
					asyncMethodCount++
				}
			}
		}
	}

	if asyncFuncCount != 1 {
		t.Errorf("Expected 1 async function, got %d", asyncFuncCount)
	}

	if asyncArrowCount != 1 {
		t.Errorf("Expected 1 async arrow function, got %d", asyncArrowCount)
	}

	if asyncMethodCount != 1 {
		t.Errorf("Expected 1 async method, got %d", asyncMethodCount)
	}
}

func TestJSParser_ClassInheritance(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `class Animal {
	constructor(name) {
		this.name = name;
	}
	
	speak() {
		console.log(this.name + ' makes a sound');
	}
}

class Dog extends Animal {
	constructor(name, breed) {
		super(name);
		this.breed = breed;
	}
	
	speak() {
		console.log(this.name + ' barks');
	}
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.js",
		AbsPath:  tmpFile,
		Language: "javascript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 2 classes
	classCount := 0
	var dogClass *ParsedSymbol

	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "class" {
			classCount++
			if sym.Name == "Dog" {
				dogClass = sym
			}
		}
	}

	if classCount != 2 {
		t.Errorf("Expected 2 classes, got %d", classCount)
	}

	// Check Dog class extends Animal
	if dogClass == nil {
		t.Fatal("Dog class not found")
	}

	// Check for extends dependency
	extendsFound := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "extends" && dep.Source == "Dog" && dep.Target == "Animal" {
			extendsFound = true
			break
		}
	}

	if !extendsFound {
		t.Errorf("Expected extends dependency from Dog to Animal")
	}
}

func TestJSParser_ErrorHandling(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "missing closing brace",
			code: `function broken() {
	return "test"
`,
			wantErr: true,
		},
		{
			name: "invalid syntax",
			code: `function ( {
}`,
			wantErr: true,
		},
		{
			name: "incomplete class",
			code: `class Person {
	constructor(name) {
		this.name = name
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.js")
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			scannedFile := ScannedFile{
				Path:     "test.js",
				AbsPath:  tmpFile,
				Language: "javascript",
				Size:     int64(len(tt.code)),
			}

			parsedFile, err := jsParser.Parse(scannedFile)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}

			// Even with errors, we should get partial results
			if parsedFile == nil {
				t.Errorf("Expected partial results even with error")
			}
		})
	}
}

func TestJSParser_ExtractCallRelationships(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `function helper() {
	return "helper";
}

function caller() {
	const result = helper();
	console.log(result);
}

class Service {
	process() {
		this.validate();
		helper();
	}
	
	validate() {
		return true;
	}
}

const arrowCaller = () => {
	helper();
	console.log("done");
};
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.js",
		AbsPath:  tmpFile,
		Language: "javascript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
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
	expectedCallers := []string{"caller", "process", "arrowCaller"}
	for _, caller := range expectedCallers {
		if foundCalls[caller] == nil {
			t.Errorf("No calls found from %s", caller)
		}
	}
}

func TestJSParser_ComplexExample(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("Failed to create Tree-sitter parser: %v", err)
	}

	jsParser := NewJSParser(tsParser)

	code := `import React, { useState } from 'react';
import './App.css';

/**
 * Main application component
 */
class App extends React.Component {
	constructor(props) {
		super(props);
		this.state = { count: 0 };
	}

	/**
	 * Increments the counter
	 */
	increment() {
		this.setState({ count: this.state.count + 1 });
	}

	render() {
		return <div>{this.state.count}</div>;
	}
}

/**
 * Counter hook component
 */
const Counter = () => {
	const [count, setCount] = useState(0);
	
	const increment = () => {
		setCount(count + 1);
	};
	
	return <button onClick={increment}>{count}</button>;
};

/**
 * Fetches user data
 */
async function fetchUser(id) {
	const response = await fetch('/api/users/' + id);
	return response.json();
}

export { App, Counter, fetchUser };
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scannedFile := ScannedFile{
		Path:     "test.js",
		AbsPath:  tmpFile,
		Language: "javascript",
		Size:     int64(len(code)),
	}

	parsedFile, err := jsParser.Parse(scannedFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Count symbol types
	counts := make(map[string]int)
	for _, sym := range parsedFile.Symbols {
		counts[sym.Kind]++
	}

	// Expected: 1 class, arrow functions (Counter and increment inside), 1 async function (fetchUser), 1 export
	if counts["class"] != 1 {
		t.Errorf("Expected 1 class, got %d", counts["class"])
	}

	// We extract both Counter and the nested increment arrow function
	if counts["arrow_function"] < 1 {
		t.Errorf("Expected at least 1 arrow function, got %d", counts["arrow_function"])
	}

	if counts["async_function"] != 1 {
		t.Errorf("Expected 1 async function, got %d", counts["async_function"])
	}

	// Check imports
	if len(parsedFile.Dependencies) < 2 {
		t.Errorf("Expected at least 2 imports, got %d", len(parsedFile.Dependencies))
	}

	// Verify class has methods
	var appClass *ParsedSymbol
	for i := range parsedFile.Symbols {
		sym := &parsedFile.Symbols[i]
		if sym.Kind == "class" && sym.Name == "App" {
			appClass = sym
			break
		}
	}

	if appClass == nil {
		t.Fatal("App class not found")
	}

	if len(appClass.Children) < 2 {
		t.Errorf("Expected at least 2 methods in App class, got %d", len(appClass.Children))
	}

	// Check for extends dependency (App extends React.Component)
	extendsFound := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "extends" && dep.Source == "App" && strings.Contains(dep.Target, "React") {
			extendsFound = true
			break
		}
	}

	if !extendsFound {
		t.Errorf("Expected extends dependency from App to React.Component")
	}
}
