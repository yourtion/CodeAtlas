package indexer

import (
	"fmt"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// ExampleSchemaValidator demonstrates how to use the schema validator
func ExampleSchemaValidator() {
	// Create a new validator
	validator := NewSchemaValidator()

	// Create sample parse output
	parseOutput := &schema.ParseOutput{
		Metadata: schema.ParseMetadata{
			Version:      "1.0.0",
			Timestamp:    time.Now(),
			TotalFiles:   1,
			SuccessCount: 1,
			FailureCount: 0,
		},
		Files: []schema.File{
			{
				FileID:   "file-1",
				Path:     "/example/main.go",
				Language: "go",
				Size:     256,
				Checksum: "abc123def456",
				Symbols: []schema.Symbol{
					{
						SymbolID:  "symbol-1",
						FileID:    "file-1",
						Name:      "main",
						Kind:      schema.SymbolFunction,
						Signature: "func main()",
						Span: schema.Span{
							StartLine: 5,
							EndLine:   10,
							StartByte: 100,
							EndByte:   200,
						},
						Docstring: "Main function entry point",
					},
				},
				Nodes: []schema.ASTNode{
					{
						NodeID: "node-1",
						FileID: "file-1",
						Type:   "function_declaration",
						Span: schema.Span{
							StartLine: 5,
							EndLine:   10,
							StartByte: 100,
							EndByte:   200,
						},
						Text: "func main() { ... }",
					},
				},
			},
		},
		Relationships: []schema.DependencyEdge{
			{
				EdgeID:     "edge-1",
				SourceID:   "symbol-1",
				TargetID:   "symbol-1", // Self-reference for example
				EdgeType:   schema.EdgeCall,
				SourceFile: "/example/main.go",
				TargetFile: "/example/main.go",
			},
		},
	}

	// Validate the parse output
	result := validator.Validate(parseOutput)

	if result.Valid {
		fmt.Println("Validation passed!")
	} else {
		fmt.Printf("Validation failed with %d errors:\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("- %s\n", err.Error())
		}
	}

	// Output: Validation passed!
}

// ExampleSchemaValidator_invalidInput demonstrates validation with invalid input
func ExampleSchemaValidator_invalidInput() {
	validator := NewSchemaValidator()

	// Create invalid parse output (missing required fields)
	parseOutput := &schema.ParseOutput{
		Metadata: schema.ParseMetadata{
			// Missing version
			TotalFiles:   1,
			SuccessCount: 2, // Invalid: more success than total
			FailureCount: 0,
		},
		Files: []schema.File{
			{
				// Missing FileID, Path, Language, Checksum
				Size: -1, // Invalid negative size
			},
		},
	}

	result := validator.Validate(parseOutput)

	fmt.Printf("Valid: %v\n", result.Valid)
	fmt.Printf("Error count: %d\n", len(result.Errors))
	
	// Show first few errors as examples
	for i, err := range result.Errors {
		if i >= 3 { // Limit output for example
			break
		}
		fmt.Printf("Error %d: %s\n", i+1, err.Type)
	}

	// Output:
	// Valid: false
	// Error count: 7
	// Error 1: required_field
	// Error 2: invalid_value
	// Error 3: required_field
}