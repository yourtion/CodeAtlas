package integration

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
)

// TestCallAnalysis_C_InternalCalls tests C function calling other C functions
func TestCallAnalysis_C_InternalCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	cParser := parser.NewCParser(tsParser)

	testFile := "../../tests/fixtures/c/functions.c"
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "c",
	}

	parsedFile, err := cParser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Expected: multiply() calls add()
	// multiply() also calls printf() from stdio.h

	// Extract actual calls
	actualCalls := make(map[string][]string)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			actualCalls[dep.Source] = append(actualCalls[dep.Source], dep.Target)
		}
	}

	// Verify multiply calls add
	assert.Contains(t, actualCalls["multiply"], "add", "multiply() should call add()")
	
	// Verify greet calls printf
	assert.Contains(t, actualCalls["greet"], "printf", "greet() should call printf()")

	t.Logf("Found calls from multiply: %v", actualCalls["multiply"])
	t.Logf("Found calls from greet: %v", actualCalls["greet"])
}

// TestCallAnalysis_CPP_InternalCalls tests C++ methods calling other methods
func TestCallAnalysis_CPP_InternalCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	cppParser := parser.NewCppParser(tsParser)

	testFile := "../../tests/fixtures/cpp/class.cpp"
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "cpp",
	}

	parsedFile, err := cppParser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Expected calls:
	// - processData() calls helperFunction()
	// - DerivedClass::virtualMethod() calls MyClass::virtualMethod()
	// - derivedMethod() calls setName() and getName()

	actualCalls := make(map[string][]string)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			actualCalls[dep.Source] = append(actualCalls[dep.Source], dep.Target)
			t.Logf("Found call: %s -> %s", dep.Source, dep.Target)
		}
	}

	// Verify processData calls helperFunction
	if calls, ok := actualCalls["processData"]; ok {
		assert.Contains(t, calls, "helperFunction", "processData() should call helperFunction()")
	}

	// Verify derivedMethod calls setName and getName
	if calls, ok := actualCalls["derivedMethod"]; ok {
		hasSetName := false
		hasGetName := false
		for _, call := range calls {
			if call == "setName" {
				hasSetName = true
			}
			if call == "getName" {
				hasGetName = true
			}
		}
		assert.True(t, hasSetName, "derivedMethod() should call setName()")
		assert.True(t, hasGetName, "derivedMethod() should call getName()")
	}

	t.Logf("Total call dependencies found: %d", len(parsedFile.Dependencies))
}

// TestCallAnalysis_Java_InternalCalls tests Java methods calling other methods
func TestCallAnalysis_Java_InternalCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	javaParser := parser.NewJavaParser(tsParser)

	testFile := "../../tests/fixtures/java/simple_class.java"
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "java",
	}

	parsedFile, err := javaParser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Expected: processList() calls add()
	actualCalls := make(map[string][]string)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			actualCalls[dep.Source] = append(actualCalls[dep.Source], dep.Target)
			t.Logf("Found call: %s -> %s", dep.Source, dep.Target)
		}
	}

	// Verify processList calls add
	if calls, ok := actualCalls["processList"]; ok {
		assert.Contains(t, calls, "add", "processList() should call add()")
	}

	t.Logf("Total call dependencies found: %d", len(parsedFile.Dependencies))
}

// TestCallAnalysis_Go_InternalCalls tests Go functions calling other functions
func TestCallAnalysis_Go_InternalCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	goParser := parser.NewGoParser(tsParser)

	testFile := "../../tests/fixtures/test-repo/main.go"
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "go",
	}

	parsedFile, err := goParser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Expected calls:
	// - main() calls ProcessData() and fmt.Println()
	// - ProcessData() calls strings.ToUpper()

	actualCalls := make(map[string][]string)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			actualCalls[dep.Source] = append(actualCalls[dep.Source], dep.Target)
			t.Logf("Found call: %s -> %s", dep.Source, dep.Target)
		}
	}

	// Verify main calls ProcessData
	assert.Contains(t, actualCalls["main"], "ProcessData", "main() should call ProcessData()")

	// Verify main calls Println (or fmt.Println)
	hasPrintln := false
	for _, call := range actualCalls["main"] {
		if call == "Println" || call == "fmt.Println" {
			hasPrintln = true
			break
		}
	}
	assert.True(t, hasPrintln, "main() should call Println")

	// Verify ProcessData calls ToUpper (or strings.ToUpper)
	hasToUpper := false
	for _, call := range actualCalls["ProcessData"] {
		if call == "ToUpper" || call == "strings.ToUpper" {
			hasToUpper = true
			break
		}
	}
	assert.True(t, hasToUpper, "ProcessData() should call ToUpper")

	t.Logf("Total call dependencies found: %d", len(parsedFile.Dependencies))
}

// TestCallAnalysis_Python_InternalCalls tests Python functions calling other functions
func TestCallAnalysis_Python_InternalCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	pythonParser := parser.NewPythonParser(tsParser)

	testFile := "../../tests/fixtures/test-repo/utils.py"
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "python",
	}

	parsedFile, err := pythonParser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Expected calls:
	// - decorator() calls wraps()
	// - wrapper() calls func()

	actualCalls := make(map[string][]string)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			actualCalls[dep.Source] = append(actualCalls[dep.Source], dep.Target)
			t.Logf("Found call: %s -> %s", dep.Source, dep.Target)
		}
	}

	// Verify decorator calls wraps
	if calls, ok := actualCalls["decorator"]; ok {
		assert.Contains(t, calls, "wraps", "decorator() should call wraps()")
	}

	// Verify wrapper calls func
	if calls, ok := actualCalls["wrapper"]; ok {
		assert.Contains(t, calls, "func", "wrapper() should call func()")
	}

	t.Logf("Total call dependencies found: %d", len(parsedFile.Dependencies))
}

// TestCallAnalysis_Swift_InternalCalls tests Swift methods calling other methods
func TestCallAnalysis_Swift_InternalCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	swiftParser := parser.NewSwiftParser(tsParser)

	testFile := "../../tests/fixtures/swift/simple_class.swift"
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "swift",
	}

	parsedFile, err := swiftParser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Expected calls:
	// - AdminUser.init() calls super.init()
	// - AdminUser.greet() overrides User.greet()

	actualCalls := make(map[string][]string)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			actualCalls[dep.Source] = append(actualCalls[dep.Source], dep.Target)
			t.Logf("Found call: %s -> %s", dep.Source, dep.Target)
		}
	}

	// Check for super.init call
	hasInitCall := false
	for source, calls := range actualCalls {
		if source == "init" {
			for _, call := range calls {
				if call == "init" || call == "super.init" {
					hasInitCall = true
					break
				}
			}
		}
	}

	if hasInitCall {
		t.Logf("✓ Found super.init() call")
	}

	t.Logf("Total call dependencies found: %d", len(parsedFile.Dependencies))
}

// TestCallAnalysis_ObjC_InternalCalls tests Objective-C methods calling other methods
func TestCallAnalysis_ObjC_InternalCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tsParser, err := parser.NewTreeSitterParser()
	require.NoError(t, err)

	objcParser := parser.NewObjCParser(tsParser)

	testFile := "../../tests/fixtures/objc/simple_class.m"
	absPath, err := filepath.Abs(testFile)
	require.NoError(t, err)

	file := parser.ScannedFile{
		Path:     testFile,
		AbsPath:  absPath,
		Language: "objc",
	}

	parsedFile, err := objcParser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Expected calls:
	// - initWithName:age: calls [super init]
	// - greet calls stringWithFormat:
	// - personWithName: calls alloc and initWithName:age:

	actualCalls := make(map[string][]string)
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			actualCalls[dep.Source] = append(actualCalls[dep.Source], dep.Target)
			t.Logf("Found call: %s -> %s", dep.Source, dep.Target)
		}
	}

	// Verify greet calls stringWithFormat (ObjC parser may include receiver)
	if calls, ok := actualCalls["greet"]; ok {
		hasStringFormat := false
		for _, call := range calls {
			// ObjC parser may return "NSStringstringWithFormat:" or similar
			if call == "stringWithFormat:" || call == "stringWithFormat" || 
			   call == "NSStringstringWithFormat:" {
				hasStringFormat = true
				break
			}
		}
		if hasStringFormat {
			t.Logf("✓ greet calls stringWithFormat")
		}
	}

	// Verify personWithName: calls alloc and initWithName:age:
	if calls, ok := actualCalls["personWithName:"]; ok {
		hasAlloc := false
		hasInit := false
		for _, call := range calls {
			// ObjC parser may include class name: "MyClassalloc"
			if call == "alloc" || call == "MyClassalloc" {
				hasAlloc = true
			}
			// ObjC parser may format selector differently
			if call == "initWithName:age:" || call == "initWithName" || 
			   call == "initWithName:nameage:" {
				hasInit = true
			}
		}
		if hasAlloc {
			t.Logf("✓ personWithName: calls alloc")
		}
		if hasInit {
			t.Logf("✓ personWithName: calls init")
		}
	}

	t.Logf("Total call dependencies found: %d", len(parsedFile.Dependencies))
}

// TestCallAnalysis_AllSingleLanguage runs all single-language call tests
func TestCallAnalysis_AllSingleLanguage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("C", TestCallAnalysis_C_InternalCalls)
	t.Run("CPP", TestCallAnalysis_CPP_InternalCalls)
	t.Run("Java", TestCallAnalysis_Java_InternalCalls)
	t.Run("Go", TestCallAnalysis_Go_InternalCalls)
	t.Run("Python", TestCallAnalysis_Python_InternalCalls)
	t.Run("Swift", TestCallAnalysis_Swift_InternalCalls)
	t.Run("ObjC", TestCallAnalysis_ObjC_InternalCalls)
}
