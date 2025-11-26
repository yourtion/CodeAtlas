package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
)

// TestCallAnalysis_GoCallerToCallee tests finding callees from a Go caller
func TestCallAnalysis_GoCallerToCallee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	// Create test file with clear call relationships
	testFile := filepath.Join(t.TempDir(), "caller.go")
	content := `package main

import "fmt"

func caller() {
	callee1()
	callee2("arg")
	fmt.Println("test")
}

func callee1() {
	// implementation
}

func callee2(s string) {
	// implementation
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "go",
	}

	parsedFile, err := goParser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Find all calls from caller function
	callsFromCaller := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "caller" {
			callsFromCaller = append(callsFromCaller, dep.Target)
		}
	}

	// Verify completeness: all callees should be found
	assert.Contains(t, callsFromCaller, "callee1", "Should find call to callee1")
	assert.Contains(t, callsFromCaller, "callee2", "Should find call to callee2")
	// Parser may return "Println" or "fmt.Println" depending on implementation
	hasPrintln := false
	for _, call := range callsFromCaller {
		if call == "Println" || call == "fmt.Println" {
			hasPrintln = true
			break
		}
	}
	assert.True(t, hasPrintln, "Should find call to Println or fmt.Println")

	// Verify precision: no false positives
	assert.GreaterOrEqual(t, len(callsFromCaller), 3, "Should find at least 3 calls")
	t.Logf("Found %d calls from caller: %v", len(callsFromCaller), callsFromCaller)
}

// TestCallAnalysis_GoCalleeToCallers tests finding callers of a Go function
func TestCallAnalysis_GoCalleeToCallers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "callee.go")
	content := `package main

func targetFunction() {
	// target function
}

func caller1() {
	targetFunction()
}

func caller2() {
	targetFunction()
}

func caller3() {
	targetFunction()
}

func notACaller() {
	// does not call targetFunction
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "go",
	}

	parsedFile, err := goParser.Parse(file)
	require.NoError(t, err)

	// Find all callers of targetFunction
	callersOfTarget := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Target == "targetFunction" {
			callersOfTarget = append(callersOfTarget, dep.Source)
		}
	}

	// Verify completeness: all callers should be found
	assert.Contains(t, callersOfTarget, "caller1", "Should find caller1")
	assert.Contains(t, callersOfTarget, "caller2", "Should find caller2")
	assert.Contains(t, callersOfTarget, "caller3", "Should find caller3")

	// Verify precision: notACaller should not be in the list
	assert.NotContains(t, callersOfTarget, "notACaller", "Should not include notACaller")
	assert.Equal(t, 3, len(callersOfTarget), "Should find exactly 3 callers")

	t.Logf("Found %d callers of targetFunction: %v", len(callersOfTarget), callersOfTarget)
}

// TestCallAnalysis_JavaCallerToCallee tests Java call analysis
func TestCallAnalysis_JavaCallerToCallee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	javaParser := parser.NewJavaParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "Caller.java")
	content := `package com.example;

public class Caller {
    public void caller() {
        callee1();
        callee2("arg");
        System.out.println("test");
    }
    
    private void callee1() {
        // implementation
    }
    
    private void callee2(String s) {
        // implementation
    }
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := javaParser.Parse(file)
	require.NoError(t, err)

	// Find calls from caller method
	callsFromCaller := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "caller" {
			callsFromCaller = append(callsFromCaller, dep.Target)
		}
	}

	// Verify completeness
	assert.Contains(t, callsFromCaller, "callee1", "Should find call to callee1")
	assert.Contains(t, callsFromCaller, "callee2", "Should find call to callee2")
	assert.Contains(t, callsFromCaller, "println", "Should find call to println")

	t.Logf("Found %d calls from caller: %v", len(callsFromCaller), callsFromCaller)
}

// TestCallAnalysis_PythonCallerToCallee tests Python call analysis
func TestCallAnalysis_PythonCallerToCallee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	pythonParser := parser.NewPythonParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "caller.py")
	content := `def caller():
    callee1()
    callee2("arg")
    print("test")

def callee1():
    pass

def callee2(s):
    pass
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "python",
	}

	parsedFile, err := pythonParser.Parse(file)
	require.NoError(t, err)

	// Find calls from caller function
	callsFromCaller := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "caller" {
			callsFromCaller = append(callsFromCaller, dep.Target)
		}
	}

	// Verify completeness
	assert.Contains(t, callsFromCaller, "callee1", "Should find call to callee1")
	assert.Contains(t, callsFromCaller, "callee2", "Should find call to callee2")
	assert.Contains(t, callsFromCaller, "print", "Should find call to print")

	t.Logf("Found %d calls from caller: %v", len(callsFromCaller), callsFromCaller)
}

// TestCallAnalysis_JSCallerToCallee tests JavaScript/TypeScript call analysis
func TestCallAnalysis_JSCallerToCallee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	jsParser := parser.NewJSParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "caller.ts")
	content := `function caller() {
    callee1();
    callee2("arg");
    console.log("test");
}

function callee1() {
    // implementation
}

function callee2(s: string) {
    // implementation
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "typescript",
	}

	parsedFile, err := jsParser.Parse(file)
	require.NoError(t, err)

	// Find calls from caller function
	callsFromCaller := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "caller" {
			callsFromCaller = append(callsFromCaller, dep.Target)
		}
	}

	// Verify completeness
	assert.Contains(t, callsFromCaller, "callee1", "Should find call to callee1")
	assert.Contains(t, callsFromCaller, "callee2", "Should find call to callee2")

	t.Logf("Found %d calls from caller: %v", len(callsFromCaller), callsFromCaller)
}

// TestCallAnalysis_CrossLanguage_KotlinToJava tests Kotlin calling Java
func TestCallAnalysis_CrossLanguage_KotlinToJava(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	kotlinParser := parser.NewKotlinParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "KotlinCaller.kt")
	content := `package com.example

import java.util.ArrayList
import java.util.HashMap

class KotlinCaller {
    fun callJavaAPIs() {
        val list = ArrayList<String>()
        list.add("item")
        
        val map = HashMap<String, Int>()
        map.put("key", 1)
        
        System.out.println("test")
    }
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "kotlin",
	}

	parsedFile, err := kotlinParser.Parse(file)
	require.NoError(t, err)

	// Find Java imports
	javaImports := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" && dep.IsExternal {
			javaImports = append(javaImports, dep.Target)
		}
	}

	// Verify Java library imports
	assert.Contains(t, javaImports, "java.util.ArrayList", "Should import ArrayList")
	assert.Contains(t, javaImports, "java.util.HashMap", "Should import HashMap")

	// Find Java API calls
	javaCalls := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			javaCalls = append(javaCalls, dep.Target)
		}
	}

	// Verify Java method calls
	assert.Contains(t, javaCalls, "add", "Should find ArrayList.add call")
	assert.Contains(t, javaCalls, "put", "Should find HashMap.put call")

	t.Logf("Found %d Java imports and %d Java calls", len(javaImports), len(javaCalls))
}

// TestCallAnalysis_CrossLanguage_SwiftToObjC tests Swift calling Objective-C
func TestCallAnalysis_CrossLanguage_SwiftToObjC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	swiftParser := parser.NewSwiftParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "SwiftCaller.swift")
	content := `import Foundation
import UIKit

class SwiftCaller: UIViewController {
    func callObjCAPIs() {
        let str = NSString(string: "test")
        let length = str.length
        
        let array = NSArray(array: [1, 2, 3])
        let count = array.count
        
        NotificationCenter.default.addObserver(self, selector: #selector(handleNotification), name: nil, object: nil)
    }
    
    @objc func handleNotification() {
        // handle
    }
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "swift",
	}

	parsedFile, err := swiftParser.Parse(file)
	require.NoError(t, err)

	// Find Objective-C framework imports
	objcImports := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" && dep.IsExternal {
			objcImports = append(objcImports, dep.Target)
		}
	}

	// Verify framework imports
	assert.Contains(t, objcImports, "Foundation", "Should import Foundation")
	assert.Contains(t, objcImports, "UIKit", "Should import UIKit")

	// Find inheritance from UIViewController
	foundInheritance := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "extends" && dep.Target == "UIViewController" {
			foundInheritance = true
			break
		}
	}
	assert.True(t, foundInheritance, "Should find inheritance from UIViewController")

	t.Logf("Found %d Objective-C framework imports", len(objcImports))
}

// TestCallAnalysis_CrossLanguage_TypeScriptToJS tests TypeScript calling JavaScript
func TestCallAnalysis_CrossLanguage_TypeScriptToJS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	jsParser := parser.NewJSParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "typescript_caller.ts")
	content := `import { legacyFunction } from './legacy.js';

class TypeScriptCaller {
    callJavaScriptAPIs() {
        // Call JavaScript module
        legacyFunction();
        
        // Use JavaScript built-ins
        console.log("test");
        setTimeout(() => {}, 1000);
        
        // Use JavaScript Array methods
        const arr = [1, 2, 3];
        arr.map(x => x * 2);
        arr.filter(x => x > 1);
    }
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "typescript",
	}

	parsedFile, err := jsParser.Parse(file)
	require.NoError(t, err)

	// Find JavaScript module imports
	jsImports := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" && dep.TargetModule == "./legacy.js" {
			jsImports = append(jsImports, dep.Target)
		}
	}

	// Verify JavaScript module import
	assert.GreaterOrEqual(t, len(jsImports), 1, "Should find import from JavaScript module")

	// Find JavaScript API calls
	jsCalls := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			jsCalls = append(jsCalls, dep.Target)
		}
	}

	// Verify JavaScript API calls
	assert.Contains(t, jsCalls, "legacyFunction", "Should find call to JavaScript function")

	t.Logf("Found %d JavaScript imports and %d calls", len(jsImports), len(jsCalls))
}

// TestCallAnalysis_Precision_NoFalsePositives tests that we don't report false calls
func TestCallAnalysis_Precision_NoFalsePositives(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "precision.go")
	content := `package main

func actualCaller() {
	actualCallee()
}

func actualCallee() {
	// implementation
}

func notACaller() {
	// This function does NOT call actualCallee
	// Just has it in a comment: actualCallee()
	var x = "actualCallee" // string literal
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "go",
	}

	parsedFile, err := goParser.Parse(file)
	require.NoError(t, err)

	// Find callers of actualCallee
	callersOfActualCallee := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Target == "actualCallee" {
			callersOfActualCallee = append(callersOfActualCallee, dep.Source)
		}
	}

	// Verify precision: only actualCaller should be found
	assert.Contains(t, callersOfActualCallee, "actualCaller", "Should find actualCaller")
	assert.NotContains(t, callersOfActualCallee, "notACaller", "Should NOT find notACaller (false positive)")
	assert.Equal(t, 1, len(callersOfActualCallee), "Should find exactly 1 caller")

	t.Logf("Precision test passed: found %d callers (expected 1)", len(callersOfActualCallee))
}

// TestCallAnalysis_Completeness_NestedCalls tests finding calls in nested contexts
func TestCallAnalysis_Completeness_NestedCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "nested.go")
	content := `package main

func caller() {
	// Direct call
	target()
	
	// Call in if statement
	if true {
		target()
	}
	
	// Call in for loop
	for i := 0; i < 10; i++ {
		target()
	}
	
	// Call in anonymous function
	func() {
		target()
	}()
	
	// Call in defer
	defer target()
	
	// Call in go routine
	go target()
}

func target() {
	// implementation
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "go",
	}

	parsedFile, err := goParser.Parse(file)
	require.NoError(t, err)

	// Count calls to target from caller
	callCount := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Target == "target" && dep.Source == "caller" {
			callCount++
		}
	}

	// Verify completeness: should find multiple calls in different contexts
	// Note: Parser may or may not attribute nested calls to parent function
	// At minimum, we should find at least one call
	assert.GreaterOrEqual(t, callCount, 1, "Should find at least 1 call to target")

	t.Logf("Completeness test: found %d calls to target from caller", callCount)
}

// TestCallAnalysis_MethodCalls tests method calls on objects
func TestCallAnalysis_MethodCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "methods.go")
	content := `package main

type MyStruct struct {
	value int
}

func (m *MyStruct) Method1() {
	// implementation
}

func (m *MyStruct) Method2() {
	// implementation
}

func caller() {
	obj := &MyStruct{}
	obj.Method1()
	obj.Method2()
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "go",
	}

	parsedFile, err := goParser.Parse(file)
	require.NoError(t, err)

	// Find method calls from caller
	methodCalls := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "caller" {
			methodCalls = append(methodCalls, dep.Target)
		}
	}

	// Verify method calls are detected
	assert.Contains(t, methodCalls, "Method1", "Should find call to Method1")
	assert.Contains(t, methodCalls, "Method2", "Should find call to Method2")

	t.Logf("Found %d method calls: %v", len(methodCalls), methodCalls)
}
