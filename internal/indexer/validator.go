package indexer

import (
	"fmt"
	"strings"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// ValidationError represents a validation error with context
type ValidationError struct {
	Type      ValidationErrorType `json:"type"`
	Message   string              `json:"message"`
	EntityID  string              `json:"entity_id,omitempty"`
	EntityType string             `json:"entity_type,omitempty"`
	FilePath  string              `json:"file_path,omitempty"`
	Field     string              `json:"field,omitempty"`
	Value     interface{}         `json:"value,omitempty"`
}

// ValidationErrorType represents the category of validation error
type ValidationErrorType string

const (
	ErrRequired           ValidationErrorType = "required_field"
	ErrInvalidType        ValidationErrorType = "invalid_type"
	ErrInvalidValue       ValidationErrorType = "invalid_value"
	ErrReferentialIntegrity ValidationErrorType = "referential_integrity"
	ErrDuplicateID        ValidationErrorType = "duplicate_id"
	ErrInvalidSpan        ValidationErrorType = "invalid_span"
)

func (e *ValidationError) Error() string {
	var parts []string
	
	if e.EntityType != "" {
		if e.EntityID != "" {
			parts = append(parts, fmt.Sprintf("%s[%s]", e.EntityType, e.EntityID))
		} else {
			parts = append(parts, e.EntityType)
		}
	}
	
	if e.FilePath != "" {
		parts = append(parts, fmt.Sprintf("file=%s", e.FilePath))
	}
	
	if e.Field != "" {
		parts = append(parts, fmt.Sprintf("field=%s", e.Field))
	}
	
	context := ""
	if len(parts) > 0 {
		context = fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
	}
	
	return fmt.Sprintf("%s: %s%s", e.Type, e.Message, context)
}

// ValidationResult contains the results of validation
type ValidationResult struct {
	Valid  bool               `json:"valid"`
	Errors []*ValidationError `json:"errors,omitempty"`
}

// AddError adds a validation error to the result
func (r *ValidationResult) AddError(err *ValidationError) {
	r.Valid = false
	r.Errors = append(r.Errors, err)
}

// HasErrors returns true if there are validation errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ErrorCount returns the number of validation errors
func (r *ValidationResult) ErrorCount() int {
	return len(r.Errors)
}

// Validator interface for validating parsed output
type Validator interface {
	// Validate checks the parsed output against schema constraints
	Validate(output *schema.ParseOutput) *ValidationResult
	
	// ValidateFile validates a single file entity
	ValidateFile(file *schema.File) *ValidationResult
	
	// ValidateSymbol validates a single symbol entity
	ValidateSymbol(symbol *schema.Symbol) *ValidationResult
	
	// ValidateASTNode validates a single AST node entity
	ValidateASTNode(node *schema.ASTNode) *ValidationResult
	
	// ValidateEdge validates a single edge entity
	ValidateEdge(edge *schema.DependencyEdge) *ValidationResult
}

// SchemaValidator implements the Validator interface
type SchemaValidator struct {
	// Track IDs to detect duplicates and validate references
	fileIDs   map[string]bool
	symbolIDs map[string]bool
	nodeIDs   map[string]bool
	edgeIDs   map[string]bool
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		fileIDs:   make(map[string]bool),
		symbolIDs: make(map[string]bool),
		nodeIDs:   make(map[string]bool),
		edgeIDs:   make(map[string]bool),
	}
}

// Validate validates the entire ParseOutput structure
func (v *SchemaValidator) Validate(output *schema.ParseOutput) *ValidationResult {
	result := &ValidationResult{Valid: true}
	
	if output == nil {
		result.AddError(&ValidationError{
			Type:    ErrRequired,
			Message: "ParseOutput cannot be nil",
		})
		return result
	}
	
	// Reset ID tracking for this validation
	v.fileIDs = make(map[string]bool)
	v.symbolIDs = make(map[string]bool)
	v.nodeIDs = make(map[string]bool)
	v.edgeIDs = make(map[string]bool)
	
	// Validate metadata
	v.validateMetadata(&output.Metadata, result)
	
	// Validate files and collect IDs
	for _, file := range output.Files {
		fileResult := v.ValidateFile(&file)
		result.Errors = append(result.Errors, fileResult.Errors...)
		if fileResult.HasErrors() {
			result.Valid = false
		}
	}
	
	// Validate relationships and check referential integrity
	for _, edge := range output.Relationships {
		edgeResult := v.ValidateEdge(&edge)
		result.Errors = append(result.Errors, edgeResult.Errors...)
		if edgeResult.HasErrors() {
			result.Valid = false
		}
	}
	
	return result
}

// ValidateFile validates a file entity
func (v *SchemaValidator) ValidateFile(file *schema.File) *ValidationResult {
	result := &ValidationResult{Valid: true}
	
	if file == nil {
		result.AddError(&ValidationError{
			Type:    ErrRequired,
			Message: "File cannot be nil",
		})
		return result
	}
	
	// Validate required fields
	if file.FileID == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "file_id is required",
			EntityType: "file",
			FilePath:   file.Path,
			Field:      "file_id",
		})
	} else {
		// Check for duplicate file IDs
		if v.fileIDs[file.FileID] {
			result.AddError(&ValidationError{
				Type:       ErrDuplicateID,
				Message:    "duplicate file_id",
				EntityType: "file",
				EntityID:   file.FileID,
				FilePath:   file.Path,
				Field:      "file_id",
				Value:      file.FileID,
			})
		} else {
			v.fileIDs[file.FileID] = true
		}
	}
	
	if file.Path == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "path is required",
			EntityType: "file",
			EntityID:   file.FileID,
			Field:      "path",
		})
	}
	
	if file.Language == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "language is required",
			EntityType: "file",
			EntityID:   file.FileID,
			FilePath:   file.Path,
			Field:      "language",
		})
	}
	
	if file.Size < 0 {
		result.AddError(&ValidationError{
			Type:       ErrInvalidValue,
			Message:    "size cannot be negative",
			EntityType: "file",
			EntityID:   file.FileID,
			FilePath:   file.Path,
			Field:      "size",
			Value:      file.Size,
		})
	}
	
	if file.Checksum == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "checksum is required",
			EntityType: "file",
			EntityID:   file.FileID,
			FilePath:   file.Path,
			Field:      "checksum",
		})
	}
	
	// Validate symbols in this file
	for _, symbol := range file.Symbols {
		// Ensure symbol belongs to this file
		if symbol.FileID != file.FileID {
			result.AddError(&ValidationError{
				Type:       ErrReferentialIntegrity,
				Message:    "symbol file_id does not match parent file",
				EntityType: "symbol",
				EntityID:   symbol.SymbolID,
				FilePath:   file.Path,
				Field:      "file_id",
				Value:      symbol.FileID,
			})
		}
		
		symbolResult := v.ValidateSymbol(&symbol)
		result.Errors = append(result.Errors, symbolResult.Errors...)
		if symbolResult.HasErrors() {
			result.Valid = false
		}
	}
	
	// Validate AST nodes in this file
	for _, node := range file.Nodes {
		// Ensure node belongs to this file
		if node.FileID != file.FileID {
			result.AddError(&ValidationError{
				Type:       ErrReferentialIntegrity,
				Message:    "AST node file_id does not match parent file",
				EntityType: "ast_node",
				EntityID:   node.NodeID,
				FilePath:   file.Path,
				Field:      "file_id",
				Value:      node.FileID,
			})
		}
		
		nodeResult := v.ValidateASTNode(&node)
		result.Errors = append(result.Errors, nodeResult.Errors...)
		if nodeResult.HasErrors() {
			result.Valid = false
		}
	}
	
	return result
}

// ValidateSymbol validates a symbol entity
func (v *SchemaValidator) ValidateSymbol(symbol *schema.Symbol) *ValidationResult {
	result := &ValidationResult{Valid: true}
	
	if symbol == nil {
		result.AddError(&ValidationError{
			Type:    ErrRequired,
			Message: "Symbol cannot be nil",
		})
		return result
	}
	
	// Validate required fields
	if symbol.SymbolID == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "symbol_id is required",
			EntityType: "symbol",
			Field:      "symbol_id",
		})
	} else {
		// Check for duplicate symbol IDs
		if v.symbolIDs[symbol.SymbolID] {
			result.AddError(&ValidationError{
				Type:       ErrDuplicateID,
				Message:    "duplicate symbol_id",
				EntityType: "symbol",
				EntityID:   symbol.SymbolID,
				Field:      "symbol_id",
				Value:      symbol.SymbolID,
			})
		} else {
			v.symbolIDs[symbol.SymbolID] = true
		}
	}
	
	if symbol.FileID == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "file_id is required",
			EntityType: "symbol",
			EntityID:   symbol.SymbolID,
			Field:      "file_id",
		})
	}
	
	if symbol.Name == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "name is required",
			EntityType: "symbol",
			EntityID:   symbol.SymbolID,
			Field:      "name",
		})
	}
	
	// Validate symbol kind
	if symbol.Kind == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "kind is required",
			EntityType: "symbol",
			EntityID:   symbol.SymbolID,
			Field:      "kind",
		})
	} else {
		validKinds := map[schema.SymbolKind]bool{
			schema.SymbolFunction:  true,
			schema.SymbolClass:     true,
			schema.SymbolInterface: true,
			schema.SymbolVariable:  true,
			schema.SymbolPackage:   true,
			schema.SymbolModule:    true,
		}
		
		if !validKinds[symbol.Kind] {
			result.AddError(&ValidationError{
				Type:       ErrInvalidValue,
				Message:    "invalid symbol kind",
				EntityType: "symbol",
				EntityID:   symbol.SymbolID,
				Field:      "kind",
				Value:      symbol.Kind,
			})
		}
	}
	
	// Validate span
	spanResult := v.validateSpan(&symbol.Span, "symbol", symbol.SymbolID)
	result.Errors = append(result.Errors, spanResult.Errors...)
	if spanResult.HasErrors() {
		result.Valid = false
	}
	
	return result
}

// ValidateASTNode validates an AST node entity
func (v *SchemaValidator) ValidateASTNode(node *schema.ASTNode) *ValidationResult {
	result := &ValidationResult{Valid: true}
	
	if node == nil {
		result.AddError(&ValidationError{
			Type:    ErrRequired,
			Message: "ASTNode cannot be nil",
		})
		return result
	}
	
	// Validate required fields
	if node.NodeID == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "node_id is required",
			EntityType: "ast_node",
			Field:      "node_id",
		})
	} else {
		// Check for duplicate node IDs
		if v.nodeIDs[node.NodeID] {
			result.AddError(&ValidationError{
				Type:       ErrDuplicateID,
				Message:    "duplicate node_id",
				EntityType: "ast_node",
				EntityID:   node.NodeID,
				Field:      "node_id",
				Value:      node.NodeID,
			})
		} else {
			v.nodeIDs[node.NodeID] = true
		}
	}
	
	if node.FileID == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "file_id is required",
			EntityType: "ast_node",
			EntityID:   node.NodeID,
			Field:      "file_id",
		})
	}
	
	if node.Type == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "type is required",
			EntityType: "ast_node",
			EntityID:   node.NodeID,
			Field:      "type",
		})
	}
	
	// Validate span
	spanResult := v.validateSpan(&node.Span, "ast_node", node.NodeID)
	result.Errors = append(result.Errors, spanResult.Errors...)
	if spanResult.HasErrors() {
		result.Valid = false
	}
	
	// Validate parent reference if present
	if node.ParentID != "" && !v.nodeIDs[node.ParentID] {
		result.AddError(&ValidationError{
			Type:       ErrReferentialIntegrity,
			Message:    "parent_id references non-existent node",
			EntityType: "ast_node",
			EntityID:   node.NodeID,
			Field:      "parent_id",
			Value:      node.ParentID,
		})
	}
	
	return result
}

// ValidateEdge validates a dependency edge entity
func (v *SchemaValidator) ValidateEdge(edge *schema.DependencyEdge) *ValidationResult {
	result := &ValidationResult{Valid: true}
	
	if edge == nil {
		result.AddError(&ValidationError{
			Type:    ErrRequired,
			Message: "DependencyEdge cannot be nil",
		})
		return result
	}
	
	// Validate required fields
	if edge.EdgeID == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "edge_id is required",
			EntityType: "edge",
			Field:      "edge_id",
		})
	} else {
		// Check for duplicate edge IDs
		if v.edgeIDs[edge.EdgeID] {
			result.AddError(&ValidationError{
				Type:       ErrDuplicateID,
				Message:    "duplicate edge_id",
				EntityType: "edge",
				EntityID:   edge.EdgeID,
				Field:      "edge_id",
				Value:      edge.EdgeID,
			})
		} else {
			v.edgeIDs[edge.EdgeID] = true
		}
	}
	
	if edge.SourceID == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "source_id is required",
			EntityType: "edge",
			EntityID:   edge.EdgeID,
			Field:      "source_id",
		})
	} else {
		// Check referential integrity - source must exist
		if !v.symbolIDs[edge.SourceID] {
			result.AddError(&ValidationError{
				Type:       ErrReferentialIntegrity,
				Message:    "source_id references non-existent symbol",
				EntityType: "edge",
				EntityID:   edge.EdgeID,
				Field:      "source_id",
				Value:      edge.SourceID,
			})
		}
	}
	
	// Target ID is optional for some edge types (e.g., external imports)
	if edge.TargetID != "" && !v.symbolIDs[edge.TargetID] {
		result.AddError(&ValidationError{
			Type:       ErrReferentialIntegrity,
			Message:    "target_id references non-existent symbol",
			EntityType: "edge",
			EntityID:   edge.EdgeID,
			Field:      "target_id",
			Value:      edge.TargetID,
		})
	}
	
	// Validate edge type
	if edge.EdgeType == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "edge_type is required",
			EntityType: "edge",
			EntityID:   edge.EdgeID,
			Field:      "edge_type",
		})
	} else {
		validTypes := map[schema.EdgeType]bool{
			schema.EdgeImport:     true,
			schema.EdgeCall:       true,
			schema.EdgeExtends:    true,
			schema.EdgeImplements: true,
			schema.EdgeReference:  true,
		}
		
		if !validTypes[edge.EdgeType] {
			result.AddError(&ValidationError{
				Type:       ErrInvalidValue,
				Message:    "invalid edge type",
				EntityType: "edge",
				EntityID:   edge.EdgeID,
				Field:      "edge_type",
				Value:      edge.EdgeType,
			})
		}
	}
	
	if edge.SourceFile == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "source_file is required",
			EntityType: "edge",
			EntityID:   edge.EdgeID,
			Field:      "source_file",
		})
	}
	
	// For certain edge types, validate additional constraints
	switch edge.EdgeType {
	case schema.EdgeImport:
		// Import edges should have either target_id or target_module
		if edge.TargetID == "" && edge.TargetModule == "" {
			result.AddError(&ValidationError{
				Type:       ErrInvalidValue,
				Message:    "import edge must have either target_id or target_module",
				EntityType: "edge",
				EntityID:   edge.EdgeID,
				Field:      "target_id/target_module",
			})
		}
	case schema.EdgeCall, schema.EdgeExtends, schema.EdgeImplements, schema.EdgeReference:
		// These edge types should have a target_id
		if edge.TargetID == "" {
			result.AddError(&ValidationError{
				Type:       ErrInvalidValue,
				Message:    fmt.Sprintf("%s edge must have target_id", edge.EdgeType),
				EntityType: "edge",
				EntityID:   edge.EdgeID,
				Field:      "target_id",
			})
		}
	}
	
	return result
}

// validateMetadata validates the parse metadata
func (v *SchemaValidator) validateMetadata(metadata *schema.ParseMetadata, result *ValidationResult) {
	if metadata == nil {
		result.AddError(&ValidationError{
			Type:    ErrRequired,
			Message: "metadata is required",
		})
		return
	}
	
	if metadata.Version == "" {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "version is required",
			EntityType: "metadata",
			Field:      "version",
		})
	}
	
	if metadata.TotalFiles < 0 {
		result.AddError(&ValidationError{
			Type:       ErrInvalidValue,
			Message:    "total_files cannot be negative",
			EntityType: "metadata",
			Field:      "total_files",
			Value:      metadata.TotalFiles,
		})
	}
	
	if metadata.SuccessCount < 0 {
		result.AddError(&ValidationError{
			Type:       ErrInvalidValue,
			Message:    "success_count cannot be negative",
			EntityType: "metadata",
			Field:      "success_count",
			Value:      metadata.SuccessCount,
		})
	}
	
	if metadata.FailureCount < 0 {
		result.AddError(&ValidationError{
			Type:       ErrInvalidValue,
			Message:    "failure_count cannot be negative",
			EntityType: "metadata",
			Field:      "failure_count",
			Value:      metadata.FailureCount,
		})
	}
	
	if metadata.SuccessCount+metadata.FailureCount != metadata.TotalFiles {
		result.AddError(&ValidationError{
			Type:       ErrInvalidValue,
			Message:    "success_count + failure_count must equal total_files",
			EntityType: "metadata",
			Field:      "success_count/failure_count/total_files",
		})
	}
}

// validateSpan validates a span structure
func (v *SchemaValidator) validateSpan(span *schema.Span, entityType, entityID string) *ValidationResult {
	result := &ValidationResult{Valid: true}
	
	if span == nil {
		result.AddError(&ValidationError{
			Type:       ErrRequired,
			Message:    "span is required",
			EntityType: entityType,
			EntityID:   entityID,
			Field:      "span",
		})
		return result
	}
	
	if span.StartLine < 1 {
		result.AddError(&ValidationError{
			Type:       ErrInvalidValue,
			Message:    "start_line must be >= 1",
			EntityType: entityType,
			EntityID:   entityID,
			Field:      "span.start_line",
			Value:      span.StartLine,
		})
	}
	
	if span.EndLine < span.StartLine {
		result.AddError(&ValidationError{
			Type:       ErrInvalidSpan,
			Message:    "end_line must be >= start_line",
			EntityType: entityType,
			EntityID:   entityID,
			Field:      "span.end_line",
			Value:      span.EndLine,
		})
	}
	
	if span.StartByte < 0 {
		result.AddError(&ValidationError{
			Type:       ErrInvalidValue,
			Message:    "start_byte cannot be negative",
			EntityType: entityType,
			EntityID:   entityID,
			Field:      "span.start_byte",
			Value:      span.StartByte,
		})
	}
	
	if span.EndByte < span.StartByte {
		result.AddError(&ValidationError{
			Type:       ErrInvalidSpan,
			Message:    "end_byte must be >= start_byte",
			EntityType: entityType,
			EntityID:   entityID,
			Field:      "span.end_byte",
			Value:      span.EndByte,
		})
	}
	
	return result
}