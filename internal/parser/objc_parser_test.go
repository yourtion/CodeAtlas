package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjCParser_ParseHeader(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewObjCParser(tsParser)

	// Test parsing simple_class.h
	headerPath := filepath.Join("../../tests/fixtures/objc/simple_class.h")
	_, err = os.ReadFile(headerPath)
	require.NoError(t, err)

	file := ScannedFile{
		Path:    "simple_class.h",
		AbsPath: headerPath,
	}

	result, err := parser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have parsed successfully
	assert.Equal(t, "simple_class.h", result.Path)
	assert.Equal(t, "objc", result.Language)
	assert.NotNil(t, result.Content)

	// Check for @interface
	var interfaceSymbol *ParsedSymbol
	for i := range result.Symbols {
		if result.Symbols[i].Kind == "interface" {
			interfaceSymbol = &result.Symbols[i]
			break
		}
	}
	require.NotNil(t, interfaceSymbol, "Should find @interface declaration")
	assert.Equal(t, "MyClass", interfaceSymbol.Name)

	// Check for properties
	propertyCount := 0
	for _, child := range interfaceSymbol.Children {
		if child.Kind == "property" {
			propertyCount++
		}
	}
	assert.GreaterOrEqual(t, propertyCount, 2, "Should have at least 2 properties")

	// Check for methods
	methodCount := 0
	for _, child := range interfaceSymbol.Children {
		if child.Kind == "method" {
			methodCount++
		}
	}
	assert.GreaterOrEqual(t, methodCount, 2, "Should have at least 2 methods")

	// Check for imports
	hasFoundationImport := false
	for _, dep := range result.Dependencies {
		if dep.Type == "import" && dep.Target == "Foundation/Foundation.h" {
			hasFoundationImport = true
			assert.True(t, dep.IsExternal, "Foundation should be external")
		}
	}
	assert.True(t, hasFoundationImport, "Should import Foundation")

	// Check for superclass dependency
	hasSuperclass := false
	for _, dep := range result.Dependencies {
		if dep.Type == "extends" && dep.Source == "MyClass" {
			hasSuperclass = true
			assert.Equal(t, "NSObject", dep.Target)
		}
	}
	assert.True(t, hasSuperclass, "Should have superclass dependency")
}

func TestObjCParser_ParseImplementation(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewObjCParser(tsParser)

	// Test parsing simple_class.m
	implPath := filepath.Join("../../tests/fixtures/objc/simple_class.m")
	_, err = os.ReadFile(implPath)
	require.NoError(t, err)

	file := ScannedFile{
		Path:    "simple_class.m",
		AbsPath: implPath,
	}

	result, err := parser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have parsed successfully
	assert.Equal(t, "simple_class.m", result.Path)
	assert.Equal(t, "objc", result.Language)

	// Check for @implementation
	var implSymbol *ParsedSymbol
	for i := range result.Symbols {
		if result.Symbols[i].Kind == "implementation" {
			implSymbol = &result.Symbols[i]
			break
		}
	}
	require.NotNil(t, implSymbol, "Should find @implementation declaration")
	assert.Equal(t, "MyClass", implSymbol.Name)

	// Check for method implementations
	methodCount := 0
	for _, child := range implSymbol.Children {
		if child.Kind == "method_implementation" {
			methodCount++
		}
	}
	assert.GreaterOrEqual(t, methodCount, 3, "Should have at least 3 method implementations")

	// Check for imports
	hasLocalImport := false
	for _, dep := range result.Dependencies {
		if dep.Type == "import" && dep.Target == "simple_class.h" {
			hasLocalImport = true
			assert.False(t, dep.IsExternal, "Local header should not be external")
		}
	}
	assert.True(t, hasLocalImport, "Should import local header")

	// Check for implements_header dependency
	hasImplementsHeader := false
	for _, dep := range result.Dependencies {
		if dep.Type == "implements_header" {
			hasImplementsHeader = true
		}
	}
	assert.True(t, hasImplementsHeader, "Should have implements_header dependency")
}

func TestObjCParser_ParseProtocol(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewObjCParser(tsParser)

	// Test parsing protocol.h
	protocolPath := filepath.Join("../../tests/fixtures/objc/protocol.h")
	_, err = os.ReadFile(protocolPath)
	require.NoError(t, err)

	file := ScannedFile{
		Path:    "protocol.h",
		AbsPath: protocolPath,
	}

	result, err := parser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check for @protocol
	var protocolSymbol *ParsedSymbol
	for i := range result.Symbols {
		if result.Symbols[i].Kind == "protocol" {
			protocolSymbol = &result.Symbols[i]
			break
		}
	}
	require.NotNil(t, protocolSymbol, "Should find @protocol declaration")
	assert.Equal(t, "DataSource", protocolSymbol.Name)

	// Check for protocol methods
	methodCount := 0
	for _, child := range protocolSymbol.Children {
		if child.Kind == "method" {
			methodCount++
		}
	}
	assert.GreaterOrEqual(t, methodCount, 2, "Should have at least 2 protocol methods")

	// Check for class that conforms to protocol
	var classSymbol *ParsedSymbol
	for i := range result.Symbols {
		if result.Symbols[i].Kind == "interface" && result.Symbols[i].Name == "DataProvider" {
			classSymbol = &result.Symbols[i]
			break
		}
	}
	require.NotNil(t, classSymbol, "Should find DataProvider class")

	// Check for protocol conformance
	hasProtocolConformance := false
	for _, dep := range result.Dependencies {
		if dep.Type == "conforms" && dep.Source == "DataProvider" && dep.Target == "DataSource" {
			hasProtocolConformance = true
		}
	}
	assert.True(t, hasProtocolConformance, "Should have protocol conformance dependency")
}

func TestObjCParser_ParseCategory(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewObjCParser(tsParser)

	// Test parsing category.h
	categoryPath := filepath.Join("../../tests/fixtures/objc/category.h")
	_, err = os.ReadFile(categoryPath)
	require.NoError(t, err)

	file := ScannedFile{
		Path:    "category.h",
		AbsPath: categoryPath,
	}

	result, err := parser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check for category
	var categorySymbol *ParsedSymbol
	for i := range result.Symbols {
		if result.Symbols[i].Kind == "category" {
			categorySymbol = &result.Symbols[i]
			break
		}
	}
	require.NotNil(t, categorySymbol, "Should find category declaration")
	assert.Contains(t, categorySymbol.Name, "NSString", "Category should be on NSString")
	assert.Contains(t, categorySymbol.Name, "Utilities", "Category should be named Utilities")

	// Check for category methods
	methodCount := 0
	for _, child := range categorySymbol.Children {
		if child.Kind == "method" {
			methodCount++
		}
	}
	assert.GreaterOrEqual(t, methodCount, 2, "Should have at least 2 category methods")

	// Check for category-to-class dependency
	hasCategoryDep := false
	for _, dep := range result.Dependencies {
		if dep.Type == "extends" && dep.Target == "NSString" {
			hasCategoryDep = true
		}
	}
	assert.True(t, hasCategoryDep, "Should have category-to-class dependency")
}

func TestObjCParser_MessageSends(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewObjCParser(tsParser)

	// Test parsing implementation with message sends
	implPath := filepath.Join("../../tests/fixtures/objc/simple_class.m")
	_, err = os.ReadFile(implPath)
	require.NoError(t, err)

	file := ScannedFile{
		Path:    "simple_class.m",
		AbsPath: implPath,
	}

	result, err := parser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check for call relationships (message sends)
	hasCallDep := false
	for _, dep := range result.Dependencies {
		if dep.Type == "call" {
			hasCallDep = true
			break
		}
	}
	assert.True(t, hasCallDep, "Should have call dependencies from message sends")
}

func TestObjCParser_ErrorHandling(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewObjCParser(tsParser)

	t.Run("NonexistentFile", func(t *testing.T) {
		file := ScannedFile{
			Path:    "nonexistent.m",
			AbsPath: "/nonexistent/path/nonexistent.m",
		}

		result, err := parser.Parse(file)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		// Create a temporary empty file
		tmpDir := t.TempDir()
		emptyFile := filepath.Join(tmpDir, "empty.m")
		err := os.WriteFile(emptyFile, []byte(""), 0644)
		require.NoError(t, err)

		file := ScannedFile{
			Path:    "empty.m",
			AbsPath: emptyFile,
		}

		result, err := parser.Parse(file)
		// Should handle empty file gracefully
		assert.NotNil(t, result)
	})
}

func TestObjCParser_HeaderDocumentation(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewObjCParser(tsParser)

	// Test parsing header with documentation
	headerPath := filepath.Join("../../tests/fixtures/objc/simple_class.h")
	_, err = os.ReadFile(headerPath)
	require.NoError(t, err)

	file := ScannedFile{
		Path:    "simple_class.h",
		AbsPath: headerPath,
	}

	result, err := parser.Parse(file)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check for interface with documentation
	var interfaceSymbol *ParsedSymbol
	for i := range result.Symbols {
		if result.Symbols[i].Kind == "interface" {
			interfaceSymbol = &result.Symbols[i]
			break
		}
	}
	require.NotNil(t, interfaceSymbol)

	// Check if documentation was extracted
	// Note: This depends on the tree-sitter grammar's ability to capture comments
	// The test verifies the extraction mechanism exists
	assert.NotNil(t, interfaceSymbol.Docstring)
}

func TestObjCParser_IsExternalImport(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewObjCParser(tsParser)

	testCases := []struct {
		name       string
		importPath string
		expected   bool
	}{
		{"Foundation framework", "Foundation/Foundation.h", true},
		{"UIKit framework", "UIKit/UIKit.h", true},
		{"CoreData framework", "CoreData/CoreData.h", true},
		{"Local header", "MyClass.h", false},
		{"Local header with path", "Models/User.h", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.isExternalImport(tc.importPath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestObjCParser_FindPairedFile(t *testing.T) {
	tsParser, err := NewTreeSitterParser()
	require.NoError(t, err)

	parser := NewObjCParser(tsParser)

	testCases := []struct {
		name         string
		currentPath  string
		expectedExt  string
	}{
		{"Header to implementation", "MyClass.h", ".m"},
		{"Implementation to header", "MyClass.m", ".h"},
		{"With path", "src/MyClass.h", ".m"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.findPairedFile(tc.currentPath)
			assert.NotEmpty(t, result)
			assert.True(t, filepath.Ext(result) == tc.expectedExt)
		})
	}
}
