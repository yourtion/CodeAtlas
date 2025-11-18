package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSwiftParser_CallsToObjC tests Swift calling Objective-C APIs
func TestSwiftParser_CallsToObjC(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewSwiftParser(tsParser)

	// Test parsing Swift file that calls Objective-C
	swiftPath := filepath.Join("../../tests/fixtures/swift/swift_calls_objc.swift")
	absPath, err := filepath.Abs(swiftPath)
	require.NoError(t, err)

	_, err = os.ReadFile(swiftPath)
	if err != nil {
		t.Skip("Test file does not exist")
	}

	file := ScannedFile{
		Path:     swiftPath,
		AbsPath:  absPath,
		Language: "swift",
	}

	parsedFile, err := parser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, parsedFile)

	// Check for Swift classes
	foundSwiftViewController := false
	foundBridgedClass := false
	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "class" && sym.Name == "SwiftViewController" {
			foundSwiftViewController = true
		}
		if sym.Kind == "class" && sym.Name == "BridgedClass" {
			foundBridgedClass = true
		}
	}

	assert.True(t, foundSwiftViewController, "Expected to find SwiftViewController class")
	assert.True(t, foundBridgedClass, "Expected to find BridgedClass")

	// Check for Objective-C framework imports
	objcFrameworks := []string{
		"Foundation",
		"UIKit",
	}

	foundFrameworks := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "import" {
			for _, framework := range objcFrameworks {
				if dep.Target == framework {
					foundFrameworks++
					assert.True(t, dep.IsExternal, "Framework %s should be external", framework)
					t.Logf("Found Objective-C framework import: %s", framework)
					break
				}
			}
		}
	}

	assert.GreaterOrEqual(t, foundFrameworks, 2, "Expected to find at least 2 Objective-C framework imports")

	// Check for calls to Objective-C APIs
	// Note: Swift parser extracts method calls, which include ObjC API calls
	objcAPICalls := []string{
		"length",        // NSString.length
		"uppercased",    // NSString.uppercased
		"count",         // NSArray.count
		"firstObject",   // NSArray.firstObject
		"addObserver",   // NotificationCenter.addObserver
		"set",           // UserDefaults.set
		"synchronize",   // UserDefaults.synchronize
		"fileExists",    // FileManager.fileExists
	}

	foundAPICalls := 0
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "call" {
			for _, apiCall := range objcAPICalls {
				if dep.Target == apiCall {
					foundAPICalls++
					t.Logf("Found Objective-C API call: %s", apiCall)
					break
				}
			}
		}
	}

	// Swift parser should extract at least some method calls
	assert.GreaterOrEqual(t, foundAPICalls, 3, "Expected to find at least 3 Objective-C API calls")

	t.Logf("Total symbols: %d, Total dependencies: %d", len(parsedFile.Symbols), len(parsedFile.Dependencies))
}

// TestSwiftParser_ObjCInterop tests Swift-ObjC interoperability features
func TestSwiftParser_ObjCInterop(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewSwiftParser(tsParser)

	swiftPath := filepath.Join("../../tests/fixtures/swift/swift_calls_objc.swift")
	absPath, err := filepath.Abs(swiftPath)
	require.NoError(t, err)

	_, err = os.ReadFile(swiftPath)
	if err != nil {
		t.Skip("Test file does not exist")
	}

	file := ScannedFile{
		Path:     swiftPath,
		AbsPath:  absPath,
		Language: "swift",
	}

	parsedFile, err := parser.Parse(file)
	require.NoError(t, err)

	// Check for @objc protocol
	foundObjCProtocol := false
	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "protocol" && sym.Name == "CustomDelegate" {
			foundObjCProtocol = true
			// Check if protocol has methods
			assert.GreaterOrEqual(t, len(sym.Children), 1, "Protocol should have at least 1 method")
		}
	}

	assert.True(t, foundObjCProtocol, "Expected to find @objc protocol")

	// Check for @objc class (BridgedClass)
	foundObjCClass := false
	for _, sym := range parsedFile.Symbols {
		if sym.Kind == "class" && sym.Name == "BridgedClass" {
			foundObjCClass = true
			// Check if class has @objc methods
			hasObjCMethod := false
			for _, child := range sym.Children {
				if child.Kind == "method" && child.Name == "objcAccessibleMethod" {
					hasObjCMethod = true
					break
				}
			}
			assert.True(t, hasObjCMethod, "BridgedClass should have @objc method")
		}
	}

	assert.True(t, foundObjCClass, "Expected to find @objc class")

	// Analyze dependency types
	importDeps := 0
	callDeps := 0
	conformsDeps := 0

	for _, dep := range parsedFile.Dependencies {
		switch dep.Type {
		case "import":
			importDeps++
		case "call":
			callDeps++
		case "conforms":
			conformsDeps++
		}
	}

	assert.Greater(t, importDeps, 0, "Expected import dependencies")
	assert.Greater(t, callDeps, 0, "Expected call dependencies")

	t.Logf("Found %d imports, %d calls, %d conforms", importDeps, callDeps, conformsDeps)
}

// TestSwiftParser_NSObjectSubclass tests Swift classes inheriting from NSObject
func TestSwiftParser_NSObjectSubclass(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewSwiftParser(tsParser)

	swiftPath := filepath.Join("../../tests/fixtures/swift/swift_calls_objc.swift")
	absPath, err := filepath.Abs(swiftPath)
	require.NoError(t, err)

	_, err = os.ReadFile(swiftPath)
	if err != nil {
		t.Skip("Test file does not exist")
	}

	file := ScannedFile{
		Path:     swiftPath,
		AbsPath:  absPath,
		Language: "swift",
	}

	parsedFile, err := parser.Parse(file)
	require.NoError(t, err)

	// Check for NSObject subclass
	foundNSObjectSubclass := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "extends" && dep.Target == "NSObject" {
			foundNSObjectSubclass = true
			t.Logf("Found NSObject subclass: %s", dep.Source)
		}
	}

	assert.True(t, foundNSObjectSubclass, "Expected to find class extending NSObject")

	// Check for UIViewController subclass
	foundUIViewControllerSubclass := false
	for _, dep := range parsedFile.Dependencies {
		if dep.Type == "extends" && dep.Target == "UIViewController" {
			foundUIViewControllerSubclass = true
			t.Logf("Found UIViewController subclass: %s", dep.Source)
		}
	}

	assert.True(t, foundUIViewControllerSubclass, "Expected to find class extending UIViewController")
}
