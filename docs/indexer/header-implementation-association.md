# Header-Implementation Association

## Overview

The header-implementation association feature automatically pairs header files (.h, .hpp, .hh, .hxx) with their corresponding implementation files (.c, .cpp, .cc, .cxx, .m, .mm) for C, C++, and Objective-C projects. This creates cross-file relationships in the knowledge graph that accurately represent the structure of these languages.

## Architecture

### Components

1. **HeaderImplAssociator** (`internal/indexer/header_impl_associator.go`)
   - Main component responsible for pairing files and matching symbols
   - Integrated into the indexer pipeline as a post-processing step

2. **New Edge Types** (added to `internal/schema/types.go`)
   - `EdgeImplementsHeader`: File-level relationship (implementation → header)
   - `EdgeImplementsDeclaration`: Symbol-level relationship (implementation → declaration)
   - `EdgeCallsDeclaration`: Call-to-declaration relationship (for future use)

3. **Graph Builder Updates** (`internal/indexer/graph_builder.go`)
   - Extended to handle new edge types in AGE graph database

### Processing Flow

```
Parse Files → Write to Database → Associate Headers/Implementations → Build Graph → Generate Embeddings
                                           ↓
                                    1. Find Pairs
                                    2. Match Symbols
                                    3. Create Edges
```

## Features

### File Pairing

The associator automatically identifies header-implementation pairs by:
- Matching file base names (e.g., `test.h` ↔ `test.c`)
- Supporting multiple extensions:
  - C: `.h` ↔ `.c`
  - C++: `.hpp`, `.hh`, `.hxx` ↔ `.cpp`, `.cc`, `.cxx`
  - Objective-C: `.h` ↔ `.m`
  - Objective-C++: `.h` ↔ `.mm`

### Symbol Matching

For each file pair, the associator:
1. Normalizes symbol signatures (handles whitespace differences)
2. Matches symbols by name and signature
3. Checks kind compatibility (e.g., `function_declaration` ↔ `function`)
4. Creates `implements_declaration` edges for matched symbols

### Edge Creation

Two types of edges are created:

1. **File-level edge** (`implements_header`)
   - Links implementation file to header file
   - No source/target symbol IDs (file-level relationship)
   - Example: `test.c` implements `test.h`

2. **Symbol-level edges** (`implements_declaration`)
   - Links implementation symbol to declaration symbol
   - Has both source and target symbol IDs
   - Example: `myFunction` in `test.c` implements `myFunction` in `test.h`

## Usage

The header-implementation association runs automatically as part of the indexing pipeline. No additional configuration is required.

### Example

Given these files:

**test.h**
```c
int myFunction(int x);
void anotherFunction();
```

**test.c**
```c
#include "test.h"

int myFunction(int x) {
    return x * 2;
}

void anotherFunction() {
    // implementation
}
```

The indexer will create:
- 1 file-level edge: `test.c` → `test.h` (implements_header)
- 2 symbol-level edges:
  - `myFunction` in test.c → `myFunction` in test.h (implements_declaration)
  - `anotherFunction` in test.c → `anotherFunction` in test.h (implements_declaration)

## Database Schema

No schema changes were required. The new edge types use the existing `edges` table:

```sql
-- File-level edge
INSERT INTO edges (edge_id, source_id, target_id, edge_type, source_file, target_file)
VALUES ('uuid', '', '', 'implements_header', 'test.c', 'test.h');

-- Symbol-level edge
INSERT INTO edges (edge_id, source_id, target_id, edge_type, source_file, target_file)
VALUES ('uuid', 'impl-symbol-id', 'header-symbol-id', 'implements_declaration', 'test.c', 'test.h');
```

## Testing

### Unit Tests

Located in `internal/indexer/header_impl_associator_test.go`:
- File pairing logic
- Symbol matching
- Signature normalization
- Edge creation

### Integration Tests

Located in `tests/integration/header_impl_integration_test.go`:
- End-to-end indexing with header-implementation files
- Database verification
- Multiple file pairs
- Different languages (C, C++, Objective-C)

Run tests:
```bash
# Unit tests
go test ./internal/indexer -run TestHeaderImplAssociator

# Integration tests (requires database)
go test ./tests/integration -run TestHeaderImplAssociation
```

## Performance

The header-implementation association is designed to be efficient:
- Runs as a post-processing step after all files are parsed
- Uses in-memory maps for O(1) file lookups
- Batch creates edges in the database
- Non-blocking: failures don't stop the indexing pipeline

## Future Enhancements

Potential improvements:
1. **Call Resolution**: Implement `calls_declaration` edges to link function calls to header declarations
2. **Template Matching**: Better support for C++ template instantiations
3. **Cross-Directory Pairing**: Support header/implementation files in different directories
4. **Signature Parsing**: More sophisticated signature comparison using AST parsing
5. **Incremental Updates**: Only re-associate changed file pairs

## Troubleshooting

### No Pairs Found

If no header-implementation pairs are found:
- Verify file extensions are correct
- Check that header and implementation files have matching base names
- Ensure files are in the same directory
- Verify language is set correctly (c, cpp, objc)

### Symbols Not Matching

If symbols aren't being matched:
- Check that function signatures match (ignoring whitespace)
- Verify symbol kinds are compatible
- Look for typos in function names
- Check that symbols are being extracted correctly by the parser

### Performance Issues

If association is slow:
- Check the number of files being processed
- Verify database connection is healthy
- Consider increasing batch size in indexer config
- Monitor memory usage during large indexing operations

## References

- Design Document: `.kiro/specs/mobile-language-parsers/design.md`
- Requirements: `.kiro/specs/mobile-language-parsers/requirements.md`
- Implementation: `internal/indexer/header_impl_associator.go`
- Tests: `internal/indexer/header_impl_associator_test.go`
