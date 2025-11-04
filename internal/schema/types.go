package schema

import "time"

// ParseOutput represents the top-level output structure
type ParseOutput struct {
	Files         []File           `json:"files"`
	Relationships []DependencyEdge `json:"relationships"`
	Metadata      ParseMetadata    `json:"metadata"`
}

// ParseMetadata contains metadata about the parsing operation
type ParseMetadata struct {
	Version      string       `json:"version"`
	Timestamp    time.Time    `json:"timestamp"`
	TotalFiles   int          `json:"total_files"`
	SuccessCount int          `json:"success_count"`
	FailureCount int          `json:"failure_count"`
	Errors       []ParseError `json:"errors,omitempty"`
}

// ParseError represents an error that occurred during parsing
type ParseError struct {
	File    string    `json:"file"`
	Line    int       `json:"line,omitempty"`
	Column  int       `json:"column,omitempty"`
	Message string    `json:"message"`
	Type    ErrorType `json:"type"`
}

// ErrorType represents the category of error
type ErrorType string

const (
	ErrorFileSystem ErrorType = "filesystem"
	ErrorParse      ErrorType = "parse"
	ErrorMapping    ErrorType = "mapping"
	ErrorLLM        ErrorType = "llm"
	ErrorOutput     ErrorType = "output"
)

// File represents a parsed source file
type File struct {
	FileID   string    `json:"file_id"`
	Path     string    `json:"path"`
	Language string    `json:"language"`
	Size     int64     `json:"size"`
	Checksum string    `json:"checksum"`
	Nodes    []ASTNode `json:"nodes"`
	Symbols  []Symbol  `json:"symbols"`
}

// Symbol represents a high-level code entity
type Symbol struct {
	SymbolID        string     `json:"symbol_id"`
	FileID          string     `json:"file_id"`
	Name            string     `json:"name"`
	Kind            SymbolKind `json:"kind"`
	Signature       string     `json:"signature"`
	Span            Span       `json:"span"`
	Docstring       string     `json:"docstring,omitempty"`
	SemanticSummary string     `json:"semantic_summary,omitempty"`
}

// SymbolKind represents the type of symbol
type SymbolKind string

const (
	SymbolFunction  SymbolKind = "function"
	SymbolClass     SymbolKind = "class"
	SymbolInterface SymbolKind = "interface"
	SymbolVariable  SymbolKind = "variable"
	SymbolPackage   SymbolKind = "package"
	SymbolModule    SymbolKind = "module"
	SymbolExternal  SymbolKind = "external_module" // Virtual symbol for external dependencies
)

// Constants for external file management
const (
	ExternalFileID   = "00000000-0000-0000-0000-000000000000"
	ExternalFilePath = "__external__"
)

// ASTNode represents a Tree-sitter AST node
type ASTNode struct {
	NodeID     string            `json:"node_id"`
	FileID     string            `json:"file_id"`
	Type       string            `json:"type"`
	Span       Span              `json:"span"`
	ParentID   string            `json:"parent_id,omitempty"`
	Text       string            `json:"text,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Span represents the location of a code element
type Span struct {
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
	StartByte int `json:"start_byte"`
	EndByte   int `json:"end_byte"`
}

// DependencyEdge represents relationships between symbols
type DependencyEdge struct {
	EdgeID       string   `json:"edge_id"`
	SourceID     string   `json:"source_id"`
	TargetID     string   `json:"target_id,omitempty"` // Optional for external imports
	EdgeType     EdgeType `json:"edge_type"`
	SourceFile   string   `json:"source_file"`
	TargetFile   string   `json:"target_file,omitempty"`
	TargetModule string   `json:"target_module,omitempty"`
}

// EdgeType represents the type of relationship between symbols
type EdgeType string

const (
	EdgeImport     EdgeType = "import"
	EdgeCall       EdgeType = "call"
	EdgeExtends    EdgeType = "extends"
	EdgeImplements EdgeType = "implements"
	EdgeReference  EdgeType = "reference"
)
