package indexer

import (
	"testing"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

func TestSchemaValidator_Validate(t *testing.T) {
	tests := []struct {
		name         string
		input        *schema.ParseOutput
		expectValid  bool
		expectErrors []ValidationErrorType
	}{
		{
			name:         "nil input",
			input:        nil,
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired},
		},
		{
			name: "valid parse output",
			input: &schema.ParseOutput{
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
						Path:     "/test/file.go",
						Language: "go",
						Size:     100,
						Checksum: "abc123",
						Symbols: []schema.Symbol{
							{
								SymbolID:  "symbol-1",
								FileID:    "file-1",
								Name:      "TestFunc",
								Kind:      schema.SymbolFunction,
								Signature: "func TestFunc()",
								Span: schema.Span{
									StartLine: 1,
									EndLine:   5,
									StartByte: 0,
									EndByte:   50,
								},
							},
						},
						Nodes: []schema.ASTNode{
							{
								NodeID: "node-1",
								FileID: "file-1",
								Type:   "function_declaration",
								Span: schema.Span{
									StartLine: 1,
									EndLine:   5,
									StartByte: 0,
									EndByte:   50,
								},
							},
						},
					},
				},
				Relationships: []schema.DependencyEdge{
					{
						EdgeID:     "edge-1",
						SourceID:   "symbol-1",
						TargetID:   "symbol-1", // self-reference for test
						EdgeType:   schema.EdgeCall,
						SourceFile: "/test/file.go",
						TargetFile: "/test/file.go",
					},
				},
			},
			expectValid:  true,
			expectErrors: nil,
		},
		{
			name: "missing required metadata fields",
			input: &schema.ParseOutput{
				Metadata: schema.ParseMetadata{
					// Missing version
					TotalFiles:   1,
					SuccessCount: 2, // Invalid: more success than total
					FailureCount: 0,
				},
				Files: []schema.File{},
			},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired, ErrInvalidValue},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSchemaValidator()
			result := validator.Validate(tt.input)

			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectValid, result.Valid)
			}

			if tt.expectErrors == nil && len(result.Errors) > 0 {
				t.Errorf("Expected no errors, got %d errors: %v", len(result.Errors), result.Errors)
			}

			if tt.expectErrors != nil {
				if len(result.Errors) != len(tt.expectErrors) {
					t.Errorf("Expected %d errors, got %d errors", len(tt.expectErrors), len(result.Errors))
				}

				errorTypes := make(map[ValidationErrorType]bool)
				for _, err := range result.Errors {
					errorTypes[err.Type] = true
				}

				for _, expectedType := range tt.expectErrors {
					if !errorTypes[expectedType] {
						t.Errorf("Expected error type %s not found in results", expectedType)
					}
				}
			}
		})
	}
}

func TestSchemaValidator_ValidateFile(t *testing.T) {
	tests := []struct {
		name         string
		input        *schema.File
		expectValid  bool
		expectErrors []ValidationErrorType
	}{
		{
			name:         "nil file",
			input:        nil,
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired},
		},
		{
			name: "valid file",
			input: &schema.File{
				FileID:   "file-1",
				Path:     "/test/file.go",
				Language: "go",
				Size:     100,
				Checksum: "abc123",
				Symbols:  []schema.Symbol{},
				Nodes:    []schema.ASTNode{},
			},
			expectValid:  true,
			expectErrors: nil,
		},
		{
			name: "missing required fields",
			input: &schema.File{
				// Missing FileID, Path, Language, Checksum
				Size: -1, // Invalid negative size
			},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired, ErrRequired, ErrRequired, ErrRequired, ErrInvalidValue},
		},
		{
			name: "duplicate file IDs",
			input: &schema.File{
				FileID:   "file-1",
				Path:     "/test/file.go",
				Language: "go",
				Size:     100,
				Checksum: "abc123",
			},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrDuplicateID},
		},
		{
			name: "symbol with mismatched file_id",
			input: &schema.File{
				FileID:   "file-1",
				Path:     "/test/file.go",
				Language: "go",
				Size:     100,
				Checksum: "abc123",
				Symbols: []schema.Symbol{
					{
						SymbolID: "symbol-1",
						FileID:   "different-file", // Mismatch
						Name:     "TestFunc",
						Kind:     schema.SymbolFunction,
						Span: schema.Span{
							StartLine: 1,
							EndLine:   5,
							StartByte: 0,
							EndByte:   50,
						},
					},
				},
			},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrReferentialIntegrity},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSchemaValidator()

			// For duplicate ID test, add the file first
			if tt.name == "duplicate file IDs" {
				validator.fileIDs["file-1"] = true
			}

			result := validator.ValidateFile(tt.input)

			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectValid, result.Valid)
			}

			if tt.expectErrors == nil && len(result.Errors) > 0 {
				t.Errorf("Expected no errors, got %d errors: %v", len(result.Errors), result.Errors)
			}

			if tt.expectErrors != nil {
				errorTypes := make(map[ValidationErrorType]bool)
				for _, err := range result.Errors {
					errorTypes[err.Type] = true
				}

				for _, expectedType := range tt.expectErrors {
					if !errorTypes[expectedType] {
						t.Errorf("Expected error type %s not found in results", expectedType)
					}
				}
			}
		})
	}
}

func TestSchemaValidator_ValidateSymbol(t *testing.T) {
	tests := []struct {
		name         string
		input        *schema.Symbol
		expectValid  bool
		expectErrors []ValidationErrorType
	}{
		{
			name:         "nil symbol",
			input:        nil,
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired},
		},
		{
			name: "valid symbol",
			input: &schema.Symbol{
				SymbolID:  "symbol-1",
				FileID:    "file-1",
				Name:      "TestFunc",
				Kind:      schema.SymbolFunction,
				Signature: "func TestFunc()",
				Span: schema.Span{
					StartLine: 1,
					EndLine:   5,
					StartByte: 0,
					EndByte:   50,
				},
			},
			expectValid:  true,
			expectErrors: nil,
		},
		{
			name: "missing required fields",
			input: &schema.Symbol{
				// Missing SymbolID, FileID, Name, Kind
				Span: schema.Span{
					StartLine: 0,  // Invalid: should be >= 1
					EndLine:   -1, // Invalid: should be >= start_line
					StartByte: -1, // Invalid: should be >= 0
					EndByte:   -2, // Invalid: should be >= start_byte
				},
			},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired, ErrRequired, ErrRequired, ErrRequired, ErrInvalidValue, ErrInvalidSpan, ErrInvalidValue, ErrInvalidSpan},
		},
		{
			name: "invalid symbol kind",
			input: &schema.Symbol{
				SymbolID: "symbol-1",
				FileID:   "file-1",
				Name:     "TestFunc",
				Kind:     "invalid_kind",
				Span: schema.Span{
					StartLine: 1,
					EndLine:   5,
					StartByte: 0,
					EndByte:   50,
				},
			},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrInvalidValue},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSchemaValidator()
			result := validator.ValidateSymbol(tt.input)

			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectValid, result.Valid)
			}

			if tt.expectErrors == nil && len(result.Errors) > 0 {
				t.Errorf("Expected no errors, got %d errors: %v", len(result.Errors), result.Errors)
			}

			if tt.expectErrors != nil {
				errorTypes := make(map[ValidationErrorType]bool)
				for _, err := range result.Errors {
					errorTypes[err.Type] = true
				}

				for _, expectedType := range tt.expectErrors {
					if !errorTypes[expectedType] {
						t.Errorf("Expected error type %s not found in results", expectedType)
					}
				}
			}
		})
	}
}

func TestSchemaValidator_ValidateASTNode(t *testing.T) {
	tests := []struct {
		name         string
		input        *schema.ASTNode
		expectValid  bool
		expectErrors []ValidationErrorType
	}{
		{
			name:         "nil node",
			input:        nil,
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired},
		},
		{
			name: "valid node",
			input: &schema.ASTNode{
				NodeID: "node-1",
				FileID: "file-1",
				Type:   "function_declaration",
				Span: schema.Span{
					StartLine: 1,
					EndLine:   5,
					StartByte: 0,
					EndByte:   50,
				},
			},
			expectValid:  true,
			expectErrors: nil,
		},
		{
			name:  "missing required fields",
			input: &schema.ASTNode{
				// Missing NodeID, FileID, Type
			},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired, ErrRequired, ErrRequired, ErrRequired},
		},
		{
			name: "invalid parent reference",
			input: &schema.ASTNode{
				NodeID:   "node-1",
				FileID:   "file-1",
				Type:     "function_declaration",
				ParentID: "non-existent-parent",
				Span: schema.Span{
					StartLine: 1,
					EndLine:   5,
					StartByte: 0,
					EndByte:   50,
				},
			},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrReferentialIntegrity},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSchemaValidator()
			result := validator.ValidateASTNode(tt.input)

			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectValid, result.Valid)
			}

			if tt.expectErrors == nil && len(result.Errors) > 0 {
				t.Errorf("Expected no errors, got %d errors: %v", len(result.Errors), result.Errors)
			}

			if tt.expectErrors != nil {
				errorTypes := make(map[ValidationErrorType]bool)
				for _, err := range result.Errors {
					errorTypes[err.Type] = true
				}

				for _, expectedType := range tt.expectErrors {
					if !errorTypes[expectedType] {
						t.Errorf("Expected error type %s not found in results", expectedType)
					}
				}
			}
		})
	}
}

func TestSchemaValidator_ValidateEdge(t *testing.T) {
	tests := []struct {
		name         string
		input        *schema.DependencyEdge
		setupSymbols []string // Symbol IDs to pre-register
		expectValid  bool
		expectErrors []ValidationErrorType
	}{
		{
			name:         "nil edge",
			input:        nil,
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired},
		},
		{
			name: "valid call edge",
			input: &schema.DependencyEdge{
				EdgeID:     "edge-1",
				SourceID:   "symbol-1",
				TargetID:   "symbol-2",
				EdgeType:   schema.EdgeCall,
				SourceFile: "/test/file.go",
				TargetFile: "/test/file2.go",
			},
			setupSymbols: []string{"symbol-1", "symbol-2"},
			expectValid:  true,
			expectErrors: nil,
		},
		{
			name: "valid import edge with target_module",
			input: &schema.DependencyEdge{
				EdgeID:       "edge-1",
				SourceID:     "symbol-1",
				EdgeType:     schema.EdgeImport,
				SourceFile:   "/test/file.go",
				TargetModule: "external.module",
			},
			setupSymbols: []string{"symbol-1"},
			expectValid:  true,
			expectErrors: nil,
		},
		{
			name:  "missing required fields",
			input: &schema.DependencyEdge{
				// Missing EdgeID, SourceID, EdgeType, SourceFile
			},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrRequired, ErrRequired, ErrRequired, ErrRequired},
		},
		{
			name: "invalid edge type",
			input: &schema.DependencyEdge{
				EdgeID:     "edge-1",
				SourceID:   "symbol-1",
				TargetID:   "symbol-2",
				EdgeType:   "invalid_type",
				SourceFile: "/test/file.go",
			},
			setupSymbols: []string{"symbol-1", "symbol-2"},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrInvalidValue},
		},
		{
			name: "referential integrity violation - source",
			input: &schema.DependencyEdge{
				EdgeID:     "edge-1",
				SourceID:   "non-existent-source",
				TargetID:   "symbol-2",
				EdgeType:   schema.EdgeCall,
				SourceFile: "/test/file.go",
			},
			setupSymbols: []string{"symbol-2"},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrReferentialIntegrity},
		},
		{
			name: "referential integrity violation - target",
			input: &schema.DependencyEdge{
				EdgeID:     "edge-1",
				SourceID:   "symbol-1",
				TargetID:   "non-existent-target",
				EdgeType:   schema.EdgeCall,
				SourceFile: "/test/file.go",
			},
			setupSymbols: []string{"symbol-1"},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrReferentialIntegrity},
		},
		{
			name: "import edge without target_id or target_module",
			input: &schema.DependencyEdge{
				EdgeID:     "edge-1",
				SourceID:   "symbol-1",
				EdgeType:   schema.EdgeImport,
				SourceFile: "/test/file.go",
				// Missing both TargetID and TargetModule
			},
			setupSymbols: []string{"symbol-1"},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrInvalidValue},
		},
		{
			name: "call edge without target_id",
			input: &schema.DependencyEdge{
				EdgeID:     "edge-1",
				SourceID:   "symbol-1",
				EdgeType:   schema.EdgeCall,
				SourceFile: "/test/file.go",
				// Missing TargetID
			},
			setupSymbols: []string{"symbol-1"},
			expectValid:  false,
			expectErrors: []ValidationErrorType{ErrInvalidValue},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSchemaValidator()

			// Setup symbols for referential integrity checks
			for _, symbolID := range tt.setupSymbols {
				validator.symbolIDs[symbolID] = true
			}

			result := validator.ValidateEdge(tt.input)

			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectValid, result.Valid)
			}

			if tt.expectErrors == nil && len(result.Errors) > 0 {
				t.Errorf("Expected no errors, got %d errors: %v", len(result.Errors), result.Errors)
			}

			if tt.expectErrors != nil {
				errorTypes := make(map[ValidationErrorType]bool)
				for _, err := range result.Errors {
					errorTypes[err.Type] = true
				}

				for _, expectedType := range tt.expectErrors {
					if !errorTypes[expectedType] {
						t.Errorf("Expected error type %s not found in results", expectedType)
					}
				}
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		error    *ValidationError
		expected string
	}{
		{
			name: "basic error",
			error: &ValidationError{
				Type:    ErrRequired,
				Message: "field is required",
			},
			expected: "required_field: field is required",
		},
		{
			name: "error with entity context",
			error: &ValidationError{
				Type:       ErrInvalidValue,
				Message:    "invalid value",
				EntityType: "symbol",
				EntityID:   "symbol-1",
				FilePath:   "/test/file.go",
				Field:      "kind",
				Value:      "invalid_kind",
			},
			expected: "invalid_value: invalid value (symbol[symbol-1], file=/test/file.go, field=kind)",
		},
		{
			name: "error with partial context",
			error: &ValidationError{
				Type:       ErrReferentialIntegrity,
				Message:    "reference not found",
				EntityType: "edge",
				Field:      "target_id",
			},
			expected: "referential_integrity: reference not found (edge, field=target_id)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			if result != tt.expected {
				t.Errorf("Expected error string %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidationResult_Methods(t *testing.T) {
	result := &ValidationResult{Valid: true}

	// Test initial state
	if result.HasErrors() {
		t.Error("Expected no errors initially")
	}
	if result.ErrorCount() != 0 {
		t.Errorf("Expected error count 0, got %d", result.ErrorCount())
	}

	// Add an error
	err := &ValidationError{
		Type:    ErrRequired,
		Message: "test error",
	}
	result.AddError(err)

	// Test after adding error
	if result.Valid {
		t.Error("Expected Valid to be false after adding error")
	}
	if !result.HasErrors() {
		t.Error("Expected HasErrors to be true after adding error")
	}
	if result.ErrorCount() != 1 {
		t.Errorf("Expected error count 1, got %d", result.ErrorCount())
	}
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error in slice, got %d", len(result.Errors))
	}
	if result.Errors[0] != err {
		t.Error("Expected added error to be in slice")
	}
}
