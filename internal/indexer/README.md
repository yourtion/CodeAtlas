# Schema Validator

The schema validator provides comprehensive validation for parsed code output structures, ensuring data integrity and referential consistency before indexing into the knowledge graph.

## Features

- **Comprehensive Validation**: Validates all entities (files, symbols, AST nodes, edges) and their relationships
- **Referential Integrity**: Ensures all references between entities are valid
- **Detailed Error Reporting**: Provides context-rich error messages with entity IDs, file paths, and field information
- **Duplicate Detection**: Identifies duplicate IDs across all entity types
- **Type Safety**: Validates enum values and data types according to schema constraints

## Usage

```go
import "github.com/yourtionguo/CodeAtlas/internal/indexer"

// Create a new validator
validator := indexer.NewSchemaValidator()

// Validate parse output
result := validator.Validate(parseOutput)

if result.Valid {
    fmt.Println("Validation passed!")
} else {
    fmt.Printf("Found %d validation errors:\n", len(result.Errors))
    for _, err := range result.Errors {
        fmt.Printf("- %s\n", err.Error())
    }
}
```

## Validation Rules

### Files
- `file_id` is required and must be unique
- `path` is required
- `language` is required
- `size` cannot be negative
- `checksum` is required
- All contained symbols and nodes must reference this file's ID

### Symbols
- `symbol_id` is required and must be unique
- `file_id` is required and must reference an existing file
- `name` is required
- `kind` must be one of: function, class, interface, variable, package, module
- `span` must have valid line and byte ranges

### AST Nodes
- `node_id` is required and must be unique
- `file_id` is required and must reference an existing file
- `type` is required
- `span` must have valid line and byte ranges
- `parent_id` (if present) must reference an existing node

### Dependency Edges
- `edge_id` is required and must be unique
- `source_id` is required and must reference an existing symbol
- `target_id` (if present) must reference an existing symbol
- `edge_type` must be one of: import, call, extends, implements, reference
- `source_file` is required
- Import edges must have either `target_id` or `target_module`
- Call/extends/implements/reference edges must have `target_id`

### Metadata
- `version` is required
- File counts cannot be negative
- `success_count + failure_count` must equal `total_files`

## Error Types

- `required_field`: Missing required field
- `invalid_type`: Wrong data type
- `invalid_value`: Invalid value (e.g., negative size, invalid enum)
- `referential_integrity`: Reference to non-existent entity
- `duplicate_id`: Duplicate entity ID
- `invalid_span`: Invalid line/byte range

## Error Context

Each validation error includes contextual information:
- Entity type and ID
- File path (when applicable)
- Field name
- Invalid value (when applicable)

Example error message:
```
referential_integrity: source_id references non-existent symbol (edge[edge-123], file=/src/main.go, field=source_id)
```

## Testing

The validator includes comprehensive tests covering:
- Valid input scenarios
- All error conditions
- Edge cases and boundary conditions
- Referential integrity checks
- Error message formatting

Run tests with:
```bash
go test ./internal/indexer
```

Current test coverage: 91.6%