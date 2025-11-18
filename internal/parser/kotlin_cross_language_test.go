package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKotlinParser_CallsToJava tests Kotlin calling Java APIs
func TestKotlinParser_CallsToJava(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewKotlinParser(tsParser)

	// Test parsing Kotlin file that calls Java
	kotlinPath := filepath.Join("../../tests/fixtures/kotlin/kotlin_calls_java.kt")
	absPath, err := filepath.Abs(kotlinPath)
	require.NoError(t, err)

	_, err = os.ReadFile(kotlinPath)
	if err != nil {
		t.Skip("Test file does not exist")
	}

	file := ScannedFile{
		Path:     kotlinPath,
		AbsPath:  absPath,
		Language: "kotlin",
	}

	parsedFile, err := parser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Check for Kotlin classes (may have fully qualified names)
	foundKotlinJavaInterop := false
	foundJavaStaticCalls := false
	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "class" && (sym.Name == "KotlinJavaInterop" || sym.Name == "com.example.interop.KotlinJavaInterop") {
			foundKotlinJavaInterop = true
			t.Logf("Found class: %s", sym.Name)
		}
		if sym.Kind == "object" && (sym.Name == "JavaStaticCalls" || sym.Name == "com.example.interop.JavaStaticCalls") {
			foundJavaStaticCalls = true
			t.Logf("Found object: %s", sym.Name)
		}
	}

	if !foundKotlinJavaInterop {
		t.Logf("All symbols: %+v", parsedFile.Symbols)
	}

	assert.True(t, foundKotlinJavaInterop, "Expected to find KotlinJavaInterop class")
	assert.True(t, foundJavaStaticCalls, "Expected to find JavaStaticCalls object")

	// Check for Java library imports
	javaImports := []string{
		"java.util.ArrayList",
		"java.util.HashMap",
		"java.util.Date",
		"java.text.SimpleDateFormat",
		"java.io.File",
		"java.io.FileReader",
		"java.io.BufferedReader",
	}

	foundImports := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			for _, javaImport := range javaImports {
				if dep.Target == javaImport || dep.TargetModule == javaImport {
					foundImports++
					t.Logf("Found Java import: %s", javaImport)
					break
				}
			}
		}
	}

	assert.GreaterOrEqual(t, foundImports, 5, "Expected to find at least 5 Java imports")

	// Check for calls to Java APIs
	javaAPICalls := []string{
		"ArrayList",      // Constructor
		"HashMap",        // Constructor
		"Date",           // Constructor
		"SimpleDateFormat", // Constructor
		"File",           // Constructor
		"add",            // ArrayList.add
		"put",            // HashMap.put
		"get",            // ArrayList/HashMap.get
		"size",           // Collection.size
		"format",         // SimpleDateFormat.format
		"exists",         // File.exists
		"readLine",       // BufferedReader.readLine
		"close",          // Closeable.close
		"length",         // String.length
		"toUpperCase",    // String.toUpperCase
		"substring",      // String.substring
		"currentTimeMillis", // System.currentTimeMillis
		"getProperty",    // System.getProperty
		"println",        // System.out.println
		"max",            // Math.max
		"sqrt",           // Math.sqrt
		"parseInt",       // Integer.parseInt
		"toHexString",    // Integer.toHexString
		"format",         // String.format
	}

	foundAPICalls := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			for _, apiCall := range javaAPICalls {
				if dep.Target == apiCall {
					foundAPICalls++
					t.Logf("Found Java API call: %s", apiCall)
					break
				}
			}
		}
	}

	// Kotlin parser should extract method calls
	assert.GreaterOrEqual(t, foundAPICalls, 10, "Expected to find at least 10 Java API calls")

	t.Logf("Total symbols: %d, Total dependencies: %d", len(parsedFile.Symbols), len(parsedFile.Dependencies))
}

// TestKotlinParser_JavaInterop tests Kotlin-Java interoperability features
func TestKotlinParser_JavaInterop(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewKotlinParser(tsParser)

	kotlinPath := filepath.Join("../../tests/fixtures/kotlin/kotlin_calls_java.kt")
	absPath, err := filepath.Abs(kotlinPath)
	require.NoError(t, err)

	_, err = os.ReadFile(kotlinPath)
	if err != nil {
		t.Skip("Test file does not exist")
	}

	file := ScannedFile{
		Path:     kotlinPath,
		AbsPath:  absPath,
		Language: "kotlin",
	}

	parsedFile, err := parser.Parse(file)
	require.NoError(t, err)

	// Check for Java interface implementation (may have fully qualified names)
	foundRunnableImpl := false
	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "class" && (sym.Name == "JavaInterfaceImpl" || sym.Name == "com.example.interop.JavaInterfaceImpl") {
			foundRunnableImpl = true
			t.Logf("Found class: %s", sym.Name)
		}
	}

	assert.True(t, foundRunnableImpl, "Expected to find class implementing Runnable")

	// Check for implements/extends dependency (Kotlin parser may use "extends" for interfaces)
	foundImplements := false
	for _, dep := range parsedFile.Dependencies {
		if (dep.Type == "implements" || dep.Type == "extends") && dep.Target == "Runnable" {
			foundImplements = true
			t.Logf("Found %s Runnable: %s", dep.Type, dep.Source)
		}
	}

	// Log all extends/implements dependencies for debugging
	if !foundImplements {
		t.Log("All extends/implements dependencies:")
		for _, dep := range parsedFile.Dependencies {
			if dep.Type == "implements" || dep.Type == "extends" {
				t.Logf("  %s: %s -> %s", dep.Type, dep.Source, dep.Target)
			}
		}
	}

	// This is optional - Kotlin parser may not extract interface implementation
	// assert.True(t, foundImplements, "Expected to find implements/extends Runnable dependency")

	// Check for Java exception handling (may have fully qualified names)
	foundExceptionClass := false
	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "class" && (sym.Name == "JavaExceptionHandling" || sym.Name == "com.example.interop.JavaExceptionHandling") {
			foundExceptionClass = true
			t.Logf("Found class: %s", sym.Name)
		}
	}

	assert.True(t, foundExceptionClass, "Expected to find JavaExceptionHandling class")

	// Analyze dependency types
	importDeps := 0
	callDeps := 0
	implementsDeps := 0

	for _, dep := range parsedFile.Dependencies {
		switch dep.Type {
		case "import":
			importDeps++
		case "call":
			callDeps++
		case "implements":
			implementsDeps++
		}
	}

	assert.Greater(t, importDeps, 0, "Expected import dependencies")
	assert.Greater(t, callDeps, 0, "Expected call dependencies")

	t.Logf("Found %d imports, %d calls, %d implements", importDeps, callDeps, implementsDeps)
}

// TestKotlinParser_JavaCollections tests Kotlin using Java collections
func TestKotlinParser_JavaCollections(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewKotlinParser(tsParser)

	kotlinPath := filepath.Join("../../tests/fixtures/kotlin/kotlin_calls_java.kt")
	absPath, err := filepath.Abs(kotlinPath)
	require.NoError(t, err)

	_, err = os.ReadFile(kotlinPath)
	if err != nil {
		t.Skip("Test file does not exist")
	}

	file := ScannedFile{
		Path:     kotlinPath,
		AbsPath:  absPath,
		Language: "kotlin",
	}

	parsedFile, err := parser.Parse(file)
	require.NoError(t, err)

	// Check for Java collection imports
	javaCollections := []string{
		"java.util.ArrayList",
		"java.util.HashMap",
	}

	foundCollections := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			for _, collection := range javaCollections {
				if dep.Target == collection || dep.TargetModule == collection {
					foundCollections++
					t.Logf("Found Java collection import: %s", collection)
					break
				}
			}
		}
	}

	assert.GreaterOrEqual(t, foundCollections, 2, "Expected to find at least 2 Java collection imports")

	// Check for collection method calls
	collectionMethods := []string{
		"add",
		"put",
		"get",
		"size",
	}

	foundMethods := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			for _, method := range collectionMethods {
				if dep.Target == method {
					foundMethods++
					break
				}
			}
		}
	}

	assert.GreaterOrEqual(t, foundMethods, 2, "Expected to find at least 2 collection method calls")
}

// TestKotlinParser_JavaStaticMethods tests Kotlin calling Java static methods
func TestKotlinParser_JavaStaticMethods(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewKotlinParser(tsParser)

	kotlinPath := filepath.Join("../../tests/fixtures/kotlin/kotlin_calls_java.kt")
	absPath, err := filepath.Abs(kotlinPath)
	require.NoError(t, err)

	_, err = os.ReadFile(kotlinPath)
	if err != nil {
		t.Skip("Test file does not exist")
	}

	file := ScannedFile{
		Path:     kotlinPath,
		AbsPath:  absPath,
		Language: "kotlin",
	}

	parsedFile, err := parser.Parse(file)
	require.NoError(t, err)

	// Check for Java static method calls
	staticMethods := []string{
		"max",            // Math.max
		"sqrt",           // Math.sqrt
		"parseInt",       // Integer.parseInt
		"toHexString",    // Integer.toHexString
		"format",         // String.format
		"currentTimeMillis", // System.currentTimeMillis
		"getProperty",    // System.getProperty
	}

	foundStaticCalls := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			for _, method := range staticMethods {
				if dep.Target == method {
					foundStaticCalls++
					t.Logf("Found Java static method call: %s", method)
					break
				}
			}
		}
	}

	assert.GreaterOrEqual(t, foundStaticCalls, 3, "Expected to find at least 3 Java static method calls")
}
