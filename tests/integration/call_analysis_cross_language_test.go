package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
)

// TestCallAnalysis_CCallerToCallee tests C call analysis
func TestCallAnalysis_CCallerToCallee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	cParser := parser.NewCParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "caller.c")
	content := `#include <stdio.h>

void callee1() {
    // implementation
}

void callee2(int x) {
    // implementation
}

void caller() {
    callee1();
    callee2(42);
    printf("test\n");
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "c",
	}

	parsedFile, err := cParser.Parse(file)
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
	assert.Contains(t, callsFromCaller, "printf", "Should find call to printf")

	t.Logf("Found %d calls from caller: %v", len(callsFromCaller), callsFromCaller)
}

// TestCallAnalysis_CPPCallerToCallee tests C++ call analysis
func TestCallAnalysis_CPPCallerToCallee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	cppParser := parser.NewCppParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "caller.cpp")
	content := `#include <iostream>

class MyClass {
public:
    void method1() {
        // implementation
    }
    
    void method2(int x) {
        // implementation
    }
    
    void caller() {
        method1();
        method2(42);
        std::cout << "test" << std::endl;
    }
};
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "cpp",
	}

	parsedFile, err := cppParser.Parse(file)
	require.NoError(t, err)

	// Find calls from caller method
	callsFromCaller := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "caller" {
			callsFromCaller = append(callsFromCaller, dep.Target)
		}
	}

	// Verify completeness
	assert.Contains(t, callsFromCaller, "method1", "Should find call to method1")
	assert.Contains(t, callsFromCaller, "method2", "Should find call to method2")

	t.Logf("Found %d calls from caller: %v", len(callsFromCaller), callsFromCaller)
}

// TestCallAnalysis_ObjCCallerToCallee tests Objective-C call analysis
func TestCallAnalysis_ObjCCallerToCallee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	objcParser := parser.NewObjCParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "Caller.m")
	content := `#import <Foundation/Foundation.h>

@interface Caller : NSObject
- (void)caller;
- (void)callee1;
- (void)callee2:(NSString *)arg;
@end

@implementation Caller

- (void)caller {
    [self callee1];
    [self callee2:@"test"];
    NSLog(@"test");
}

- (void)callee1 {
    // implementation
}

- (void)callee2:(NSString *)arg {
    // implementation
}

@end
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "objc",
	}

	parsedFile, err := objcParser.Parse(file)
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
	assert.Contains(t, callsFromCaller, "callee2:", "Should find call to callee2:")

	t.Logf("Found %d calls from caller: %v", len(callsFromCaller), callsFromCaller)
}

// TestCallAnalysis_CrossLanguage_CPPToC tests C++ calling C functions
func TestCallAnalysis_CrossLanguage_CPPToC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	cppParser := parser.NewCppParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "cpp_calls_c.cpp")
	content := `extern "C" {
    #include <stdio.h>
    #include <stdlib.h>
    
    void c_function();
}

class CPPCaller {
public:
    void callCFunctions() {
        // Call C standard library functions
        printf("test\n");
        malloc(100);
        free(nullptr);
        
        // Call custom C function
        c_function();
    }
};
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "cpp",
	}

	parsedFile, err := cppParser.Parse(file)
	require.NoError(t, err)

	// Find C function calls
	cCalls := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "callCFunctions" {
			cCalls = append(cCalls, dep.Target)
		}
	}

	// Verify C function calls
	assert.Contains(t, cCalls, "printf", "Should find call to printf")
	assert.Contains(t, cCalls, "malloc", "Should find call to malloc")
	assert.Contains(t, cCalls, "c_function", "Should find call to c_function")

	t.Logf("Found %d C function calls: %v", len(cCalls), cCalls)
}

// TestCallAnalysis_CrossLanguage_ObjCPPToObjC tests Objective-C++ calling Objective-C
func TestCallAnalysis_CrossLanguage_ObjCPPToObjC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	objcppParser := parser.NewObjCppParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "ObjCPPCaller.mm")
	content := `#import <Foundation/Foundation.h>
#include <iostream>

@interface ObjCClass : NSObject
- (void)objcMethod;
@end

@implementation ObjCClass
- (void)objcMethod {
    NSLog(@"Objective-C method");
}
@end

class CPPClass {
public:
    void callObjC() {
        ObjCClass *obj = [[ObjCClass alloc] init];
        [obj objcMethod];
        
        NSString *str = @"test";
        NSLog(@"%@", str);
        
        std::cout << "C++ code" << std::endl;
    }
};
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(testFile)
	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "objcpp",
	}

	parsedFile, err := objcppParser.Parse(file)
	require.NoError(t, err)

	// Find Objective-C method calls
	objcCalls := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			objcCalls = append(objcCalls, dep.Target)
		}
	}

	// Verify Objective-C calls are detected
	assert.Contains(t, objcCalls, "objcMethod", "Should find call to objcMethod")
	assert.Contains(t, objcCalls, "alloc", "Should find call to alloc")
	assert.Contains(t, objcCalls, "init", "Should find call to init")

	t.Logf("Found %d Objective-C calls: %v", len(objcCalls), objcCalls)
}

// TestCallAnalysis_MultiFile_CallerCallee tests call analysis across multiple files
func TestCallAnalysis_MultiFile_CallerCallee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	tmpDir := t.TempDir()

	// Create caller file
	callerFile := filepath.Join(tmpDir, "caller.go")
	callerContent := `package main

import "example.com/test/callee"

func caller() {
	callee.ExportedFunction()
}
`
	err = os.WriteFile(callerFile, []byte(callerContent), 0644)
	require.NoError(t, err)

	// Create callee file
	calleeFile := filepath.Join(tmpDir, "callee", "callee.go")
	err = os.MkdirAll(filepath.Dir(calleeFile), 0755)
	require.NoError(t, err)

	calleeContent := `package callee

func ExportedFunction() {
	// implementation
}
`
	err = os.WriteFile(calleeFile, []byte(calleeContent), 0644)
	require.NoError(t, err)

	// Parse caller file
	absCallerPath, _ := filepath.Abs(callerFile)
	callerScanned := parser.ScannedFile{
		Path:     callerFile,
		AbsPath:  absCallerPath,
		Language: "go",
	}

	parsedCaller, err := goParser.Parse(callerScanned)
	require.NoError(t, err)

	// Find calls from caller
	callsFromCaller := []string{}
	for _, dep := range parsedCaller.Dependencies {
		if dep.Type == "call" && dep.Source == "caller" {
			callsFromCaller = append(callsFromCaller, dep.Target)
		}
	}

	// Verify cross-file call is detected
	assert.Contains(t, callsFromCaller, "ExportedFunction", "Should find call to ExportedFunction from another package")

	// Parse callee file
	absCalleePath, _ := filepath.Abs(calleeFile)
	calleeScanned := parser.ScannedFile{
		Path:     calleeFile,
		AbsPath:  absCalleePath,
		Language: "go",
	}

	parsedCallee, err := goParser.Parse(calleeScanned)
	require.NoError(t, err)

	// Verify ExportedFunction is defined
	foundExportedFunction := false
	for _, sym := range parsedCallee.Symbols {
		if sym.Name == "ExportedFunction" && sym.Kind == "function" {
			foundExportedFunction = true
			break
		}
	}
	assert.True(t, foundExportedFunction, "Should find ExportedFunction definition in callee file")

	t.Logf("Multi-file test: found %d calls from caller", len(callsFromCaller))
}

// TestCallAnalysis_RecursiveCalls tests detection of recursive function calls
func TestCallAnalysis_RecursiveCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "recursive.go")
	content := `package main

func factorial(n int) int {
    if n <= 1 {
        return 1
    }
    return n * factorial(n - 1)
}

func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
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

	// Find recursive calls in factorial
	factorialRecursiveCalls := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "factorial" && dep.Target == "factorial" {
			factorialRecursiveCalls++
		}
	}

	// Find recursive calls in fibonacci
	fibonacciRecursiveCalls := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "fibonacci" && dep.Target == "fibonacci" {
			fibonacciRecursiveCalls++
		}
	}

	// Verify recursive calls are detected
	assert.GreaterOrEqual(t, factorialRecursiveCalls, 1, "Should find recursive call in factorial")
	assert.GreaterOrEqual(t, fibonacciRecursiveCalls, 1, "Should find recursive calls in fibonacci")

	t.Logf("Found %d recursive calls in factorial, %d in fibonacci", 
		factorialRecursiveCalls, fibonacciRecursiveCalls)
}

// TestCallAnalysis_IndirectCalls tests detection of indirect calls (function pointers, callbacks)
func TestCallAnalysis_IndirectCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "indirect.go")
	content := `package main

func callback() {
    // implementation
}

func caller() {
    // Direct call
    callback()
    
    // Indirect call via variable
    fn := callback
    fn()
    
    // Anonymous function
    func() {
        callback()
    }()
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

	// Find calls to callback
	callsToCallback := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Target == "callback" {
			callsToCallback++
		}
	}

	// Should find at least the direct call
	assert.GreaterOrEqual(t, callsToCallback, 1, "Should find at least direct call to callback")

	t.Logf("Found %d calls to callback (direct and indirect)", callsToCallback)
}

// TestCallAnalysis_InterfaceCalls tests calls through interfaces
func TestCallAnalysis_InterfaceCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)
	

	goParser := parser.NewGoParser(tsParser)

	testFile := filepath.Join(t.TempDir(), "interface.go")
	content := `package main

type MyInterface interface {
    Method1()
    Method2(int) string
}

type MyImpl struct{}

func (m *MyImpl) Method1() {
    // implementation
}

func (m *MyImpl) Method2(x int) string {
    return "result"
}

func caller(obj MyInterface) {
    obj.Method1()
    result := obj.Method2(42)
    _ = result
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

	// Find interface method calls
	interfaceCalls := []string{}
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" && dep.Source == "caller" {
			interfaceCalls = append(interfaceCalls, dep.Target)
		}
	}

	// Verify interface method calls are detected
	assert.Contains(t, interfaceCalls, "Method1", "Should find call to Method1")
	assert.Contains(t, interfaceCalls, "Method2", "Should find call to Method2")

	t.Logf("Found %d interface method calls: %v", len(interfaceCalls), interfaceCalls)
}
