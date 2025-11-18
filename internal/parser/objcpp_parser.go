package parser

import (
	"fmt"
	"strings"
)

// ObjCppParser parses Objective-C++ source code (.mm files)
// It uses a hybrid approach, trying both C++ and Objective-C parsers
type ObjCppParser struct {
	cppParser  *CppParser
	objcParser *ObjCParser
}

// NewObjCppParser creates a new Objective-C++ parser
func NewObjCppParser(tsParser *TreeSitterParser) *ObjCppParser {
	return &ObjCppParser{
		cppParser:  NewCppParser(tsParser),
		objcParser: NewObjCParser(tsParser),
	}
}

// Parse parses an Objective-C++ file (.mm)
// It attempts to use the C++ parser first (for better C++ syntax support)
// and falls back to Objective-C parser if needed
func (p *ObjCppParser) Parse(file ScannedFile) (*ParsedFile, error) {
	// Try C++ parser first (better for C++ syntax)
	parsedFile, err := p.cppParser.Parse(file)
	
	// If C++ parser succeeds without major errors, use it
	// Even with parse errors, we may have extracted useful symbols
	if err == nil {
		// Mark as Objective-C++ for clarity
		if parsedFile != nil {
			parsedFile.Language = "objcpp"
		}
		return parsedFile, nil
	}
	
	// If we got partial results from C++ parser, use them
	if parsedFile != nil && (len(parsedFile.Symbols) > 0 || len(parsedFile.Dependencies) > 0) {
		parsedFile.Language = "objcpp"
		// Return with partial results, but keep the error
		return parsedFile, err
	}
	
	// Fall back to Objective-C parser
	parsedFile, err = p.objcParser.Parse(file)
	if parsedFile != nil {
		parsedFile.Language = "objcpp"
		// If we got partial results, return them even with error
		if len(parsedFile.Symbols) > 0 || len(parsedFile.Dependencies) > 0 {
			return parsedFile, err
		}
	}
	
	return parsedFile, err
}

// isParseError checks if an error is a parse error
func isParseError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	return strings.Contains(errStr, "parse") || 
	       strings.Contains(errStr, "syntax")
}

// ParseWithBothParsers attempts to parse with both parsers and merge results
// This is an advanced approach that combines Objective-C and C++ parsing
func (p *ObjCppParser) ParseWithBothParsers(file ScannedFile) (*ParsedFile, error) {
	// Parse with C++ parser
	cppParsed, cppErr := p.cppParser.Parse(file)
	
	// Parse with Objective-C parser
	objcParsed, objcErr := p.objcParser.Parse(file)
	
	// If both failed, return the C++ error (more likely to be accurate)
	if cppErr != nil && objcErr != nil {
		return cppParsed, cppErr
	}
	
	// If only one succeeded, return that one
	if cppErr != nil {
		objcParsed.Language = "objcpp"
		return objcParsed, objcErr
	}
	if objcErr != nil {
		cppParsed.Language = "objcpp"
		return cppParsed, cppErr
	}
	
	// Both succeeded - merge results
	merged := p.mergeResults(cppParsed, objcParsed)
	merged.Language = "objcpp"
	
	return merged, nil
}

// mergeResults merges symbols and dependencies from both parsers
func (p *ObjCppParser) mergeResults(cppResult, objcResult *ParsedFile) *ParsedFile {
	merged := &ParsedFile{
		Path:     cppResult.Path,
		Language: "objcpp",
		Content:  cppResult.Content,
		RootNode: cppResult.RootNode,
	}
	
	// Merge symbols (avoid duplicates by name)
	symbolMap := make(map[string]ParsedSymbol)
	
	// Add C++ symbols
	for _, sym := range cppResult.Symbols {
		key := fmt.Sprintf("%s:%s", sym.Kind, sym.Name)
		symbolMap[key] = sym
	}
	
	// Add Objective-C symbols (may override C++ if same name)
	for _, sym := range objcResult.Symbols {
		key := fmt.Sprintf("%s:%s", sym.Kind, sym.Name)
		// Prefer Objective-C symbols for classes/methods
		if sym.Kind == "class" || sym.Kind == "implementation" || 
		   sym.Kind == "protocol" || sym.Kind == "category" {
			symbolMap[key] = sym
		} else if _, exists := symbolMap[key]; !exists {
			symbolMap[key] = sym
		}
	}
	
	// Convert map back to slice
	for _, sym := range symbolMap {
		merged.Symbols = append(merged.Symbols, sym)
	}
	
	// Merge dependencies (avoid duplicates)
	depMap := make(map[string]ParsedDependency)
	
	for _, dep := range cppResult.Dependencies {
		key := fmt.Sprintf("%s:%s:%s", dep.Type, dep.Source, dep.Target)
		depMap[key] = dep
	}
	
	for _, dep := range objcResult.Dependencies {
		key := fmt.Sprintf("%s:%s:%s", dep.Type, dep.Source, dep.Target)
		if _, exists := depMap[key]; !exists {
			depMap[key] = dep
		}
	}
	
	// Convert map back to slice
	for _, dep := range depMap {
		merged.Dependencies = append(merged.Dependencies, dep)
	}
	
	return merged
}
