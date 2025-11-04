# External Dependencies Design

## Problem

When parsing code, we encounter two types of dependencies:

1. **Internal dependencies**: References to symbols within the codebase
   ```javascript
   import { myFunction } from './utils';  // utils.js is in our codebase
   ```

2. **External dependencies**: References to external libraries/modules
   ```javascript
   import lodash from 'lodash';  // lodash is an external package
   ```

For internal dependencies, we can resolve both `source_id` and `target_id` to actual symbols in our database.

For external dependencies, we cannot resolve `target_id` because the external module is not in our codebase.

## Current Approach (Suboptimal)

Currently, we create edges with:
- `source_id`: The importing symbol
- `target_id`: NULL
- `target_module`: The external module name (e.g., "lodash")

**Problem**: An edge without a target is semantically incomplete and makes graph queries awkward.

## Proposed Solution: Virtual Symbols for External Modules

Create virtual symbols to represent external modules, so every edge has both source and target.

### Implementation

1. **Create virtual symbols for external modules**:
   ```go
   // When encountering an external import
   externalSymbol := &models.Symbol{
       SymbolID:  generateExternalSymbolID("lodash"),
       FileID:    "external",  // Special file ID for external modules
       Name:      "lodash",
       Kind:      "external_module",
       Signature: "external module: lodash",
   }
   ```

2. **Create edges with both source and target**:
   ```go
   edge := &models.Edge{
       EdgeID:       generateUUID(),
       SourceID:     sourceSymbolID,
       TargetID:     externalSymbolID,  // Now we have a target!
       EdgeType:     "import",
       SourceFile:   "src/main.js",
       TargetModule: "lodash",
   }
   ```

### Benefits

1. **Complete graph structure**: Every edge has both source and target
2. **Easy queries**: 
   - "Which files import lodash?" → Query edges where target is lodash symbol
   - "What external dependencies does this file have?" → Query edges from file to external symbols
3. **Consistent data model**: No special cases for external dependencies
4. **Dependency analysis**: Can analyze external dependency usage patterns

### Database Schema

Add a new symbol kind for external modules:

```sql
-- symbols table already supports this with the 'kind' field
-- Just need to use kind = 'external_module'

-- Create a special "external" file to hold all external symbols
INSERT INTO files (file_id, repo_id, path, language, size, checksum)
VALUES ('00000000-0000-0000-0000-000000000000', repo_id, '__external__', 'external', 0, '');
```

### Example

For this code:
```javascript
// src/main.js
import lodash from 'lodash';
import { myUtil } from './utils';

function main() {
    return lodash.map([1, 2, 3], x => x * 2);
}
```

We create:

**Symbols**:
1. `main` function (file: src/main.js)
2. `myUtil` function (file: src/utils.js)
3. `lodash` external module (file: __external__)

**Edges**:
1. Import edge: `main` → `lodash` (external)
2. Import edge: `main` → `myUtil` (internal)
3. Call edge: `main` → `lodash.map` (if we can resolve it)

### Migration Path

1. **Phase 1**: Keep current approach (nullable target_id) ✅ DONE
2. **Phase 2**: Add logic to create virtual symbols for external modules
3. **Phase 3**: Migrate existing edges to use virtual symbols
4. **Phase 4**: Make target_id required again

## Alternative: Keep Current Approach

If we decide to keep the current approach (nullable target_id), we should:

1. **Document the semantics clearly**:
   - Edges with `target_id = NULL` represent external dependencies
   - Use `target_module` to identify the external dependency

2. **Update queries to handle NULL targets**:
   ```sql
   -- Find all external dependencies
   SELECT DISTINCT target_module 
   FROM edges 
   WHERE target_id IS NULL AND edge_type = 'import';
   
   -- Find files that import a specific external module
   SELECT DISTINCT source_file 
   FROM edges 
   WHERE target_id IS NULL AND target_module = 'lodash';
   ```

3. **Accept that the graph is incomplete**:
   - External dependencies are tracked but not as graph nodes
   - This is simpler but less powerful for analysis

## Recommendation

**Use virtual symbols for external modules** (Proposed Solution).

This provides:
- Complete graph structure
- Better query capabilities
- More consistent data model
- Foundation for future features (e.g., tracking external module versions, security vulnerabilities)

The implementation complexity is minimal, and the benefits are significant.

---

## Detailed Design

### 1. External Symbol Identification

**Symbol ID Generation**:
```go
// Generate deterministic ID for external modules
func GenerateExternalSymbolID(moduleName string) string {
    return utils.GenerateDeterministicUUID("external:" + moduleName)
}
```

**Symbol Properties**:
- `symbol_id`: Deterministic UUID based on module name
- `file_id`: Special "external" file ID (constant)
- `name`: Module name (e.g., "lodash", "react", "@types/node")
- `kind`: "external_module"
- `signature`: "external module: {name}"
- `docstring`: Optional - could include package description

### 2. External File Management

Create a special virtual file to hold all external symbols:

```go
const (
    ExternalFileID   = "00000000-0000-0000-0000-000000000000"
    ExternalFilePath = "__external__"
)

// Create external file once per repository
func CreateExternalFile(ctx context.Context, repoID string) error {
    file := &models.File{
        FileID:   ExternalFileID,
        RepoID:   repoID,
        Path:     ExternalFilePath,
        Language: "external",
        Size:     0,
        Checksum: "external",
    }
    return fileRepo.Create(ctx, file)
}
```

### 3. Parser Changes

Update parsers to track external dependencies:

```go
// In js_parser.go
func (p *JSParser) extractImports(...) error {
    // ... existing code ...
    
    for _, match := range matches {
        importPath := strings.Trim(capture.Node.Content(content), "\"'`")
        
        // Determine if this is an external or internal import
        isExternal := p.isExternalImport(importPath)
        
        dependency := ParsedDependency{
            Type:         "import",
            Source:       moduleSymbol,
            Target:       importPath,
            TargetModule: importPath,
            IsExternal:   isExternal,  // NEW FIELD
        }
        
        parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
    }
}

// Helper to determine if import is external
func (p *JSParser) isExternalImport(importPath string) bool {
    // Relative paths are internal
    if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
        return true
    }
    // Absolute paths starting with / are internal
    if strings.HasPrefix(importPath, "/") {
        return true
    }
    // Everything else is external (npm packages, etc.)
    return false
}
```

### 4. Schema Mapper Changes

Update mapper to create virtual symbols:

```go
// In mapper.go
type SchemaMapper struct {
    symbolIDMap        map[string]string
    externalSymbols    map[string]*Symbol  // NEW: Track external symbols
}

func (m *SchemaMapper) MapToSchema(parsed *parser.ParsedFile) (*File, []DependencyEdge, error) {
    // ... existing code ...
    
    // Collect external dependencies
    externalModules := m.collectExternalModules(parsed.Dependencies)
    
    // Create virtual symbols for external modules
    for moduleName := range externalModules {
        externalSymbol := m.createExternalSymbol(moduleName)
        m.externalSymbols[moduleName] = &externalSymbol
        m.symbolIDMap[moduleName] = externalSymbol.SymbolID
    }
    
    // Map dependencies (now all will have target_id)
    edges := m.mapDependencies(parsed.Dependencies, fileID, parsed.Path)
    
    return file, edges, nil
}

func (m *SchemaMapper) createExternalSymbol(moduleName string) Symbol {
    symbolID := utils.GenerateDeterministicUUID("external:" + moduleName)
    
    return Symbol{
        SymbolID:  symbolID,
        FileID:    ExternalFileID,
        Name:      moduleName,
        Kind:      SymbolExternal,  // NEW KIND
        Signature: fmt.Sprintf("external module: %s", moduleName),
        Span: Span{
            StartLine: 0,
            EndLine:   0,
            StartByte: 0,
            EndByte:   0,
        },
    }
}

func (m *SchemaMapper) collectExternalModules(deps []parser.ParsedDependency) map[string]bool {
    modules := make(map[string]bool)
    for _, dep := range deps {
        if dep.IsExternal && dep.Type == "import" {
            modules[dep.TargetModule] = true
        }
    }
    return modules
}
```

### 5. Schema Types Update

Add new symbol kind:

```go
// In schema/types.go
const (
    SymbolFunction  SymbolKind = "function"
    SymbolClass     SymbolKind = "class"
    SymbolInterface SymbolKind = "interface"
    SymbolVariable  SymbolKind = "variable"
    SymbolPackage   SymbolKind = "package"
    SymbolModule    SymbolKind = "module"
    SymbolExternal  SymbolKind = "external_module"  // NEW
)

// ParsedDependency in parser package
type ParsedDependency struct {
    Type         string
    Source       string
    Target       string
    TargetModule string
    IsExternal   bool  // NEW FIELD
}
```

### 6. Indexer Changes

Update indexer to handle external symbols:

```go
// In indexer.go
func (idx *Indexer) Index(ctx context.Context, input *schema.ParseOutput) (*IndexResult, error) {
    // ... existing validation and repo creation ...
    
    // Create external file if needed
    if err := idx.ensureExternalFile(ctx); err != nil {
        return nil, err
    }
    
    // Collect all external symbols from all files
    externalSymbols := idx.collectExternalSymbols(input.Files)
    
    // Write external symbols first (so they exist when we write edges)
    if len(externalSymbols) > 0 {
        if err := idx.writeExternalSymbols(ctx, externalSymbols); err != nil {
            return nil, err
        }
    }
    
    // ... continue with normal indexing ...
}

func (idx *Indexer) ensureExternalFile(ctx context.Context) error {
    fileRepo := models.NewFileRepository(idx.db)
    
    // Check if external file exists
    existing, err := fileRepo.GetByID(ctx, schema.ExternalFileID)
    if err != nil {
        return err
    }
    
    if existing == nil {
        // Create external file
        externalFile := &models.File{
            FileID:   schema.ExternalFileID,
            RepoID:   idx.config.RepoID,
            Path:     schema.ExternalFilePath,
            Language: "external",
            Size:     0,
            Checksum: "external",
        }
        return fileRepo.Create(ctx, externalFile)
    }
    
    return nil
}

func (idx *Indexer) collectExternalSymbols(files []schema.File) []schema.Symbol {
    seen := make(map[string]bool)
    var externals []schema.Symbol
    
    for _, file := range files {
        for _, symbol := range file.Symbols {
            if symbol.Kind == schema.SymbolExternal {
                if !seen[symbol.SymbolID] {
                    seen[symbol.SymbolID] = true
                    externals = append(externals, symbol)
                }
            }
        }
    }
    
    return externals
}
```

### 7. Validator Updates

Update validator to accept external symbols:

```go
// In validator.go
func (v *SchemaValidator) ValidateSymbol(symbol *schema.Symbol) *ValidationResult {
    // ... existing validation ...
    
    validKinds := map[schema.SymbolKind]bool{
        schema.SymbolFunction:  true,
        schema.SymbolClass:     true,
        schema.SymbolInterface: true,
        schema.SymbolVariable:  true,
        schema.SymbolPackage:   true,
        schema.SymbolModule:    true,
        schema.SymbolExternal:  true,  // NEW
    }
    
    // ... rest of validation ...
}

func (v *SchemaValidator) ValidateEdge(edge *schema.DependencyEdge) *ValidationResult {
    // ... existing validation ...
    
    // Now target_id is ALWAYS required (no more NULL targets)
    if edge.TargetID == "" {
        result.AddError(&ValidationError{
            Type:       ErrRequired,
            Message:    "target_id is required",
            EntityType: "edge",
            EntityID:   edge.EdgeID,
            Field:      "target_id",
        })
    }
    
    // ... rest of validation ...
}
```

### 8. Query Examples

With virtual symbols, queries become simpler:

```sql
-- Find all external dependencies in a repository
SELECT DISTINCT s.name, s.signature
FROM symbols s
WHERE s.kind = 'external_module' AND s.file_id = '00000000-0000-0000-0000-000000000000';

-- Find all files that import lodash
SELECT DISTINCT e.source_file, s1.name as source_symbol
FROM edges e
JOIN symbols s1 ON e.source_id = s1.symbol_id
JOIN symbols s2 ON e.target_id = s2.symbol_id
WHERE s2.name = 'lodash' AND s2.kind = 'external_module';

-- Count usage of each external dependency
SELECT s.name, COUNT(*) as usage_count
FROM edges e
JOIN symbols s ON e.target_id = s.symbol_id
WHERE s.kind = 'external_module'
GROUP BY s.name
ORDER BY usage_count DESC;

-- Find files with most external dependencies
SELECT e.source_file, COUNT(DISTINCT e.target_id) as dep_count
FROM edges e
JOIN symbols s ON e.target_id = s.symbol_id
WHERE s.kind = 'external_module'
GROUP BY e.source_file
ORDER BY dep_count DESC;
```

### 9. Benefits Summary

1. **Complete Graph**: Every edge has both source and target
2. **Consistent Model**: No special cases for external dependencies
3. **Better Queries**: Can treat external dependencies as first-class nodes
4. **Dependency Analysis**: Easy to analyze external dependency usage
5. **Future Features**: Foundation for:
   - Tracking package versions
   - Security vulnerability scanning
   - License compliance checking
   - Dependency update recommendations

### 10. Migration Strategy

For existing data:

```sql
-- Step 1: Create external file for each repository
INSERT INTO files (file_id, repo_id, path, language, size, checksum)
SELECT '00000000-0000-0000-0000-000000000000', repo_id, '__external__', 'external', 0, 'external'
FROM repositories
WHERE NOT EXISTS (
    SELECT 1 FROM files 
    WHERE file_id = '00000000-0000-0000-0000-000000000000' 
    AND repo_id = repositories.repo_id
);

-- Step 2: Create symbols for external modules
INSERT INTO symbols (symbol_id, file_id, name, kind, signature, start_line, end_line, start_byte, end_byte)
SELECT 
    md5('external:' || target_module)::uuid,
    '00000000-0000-0000-0000-000000000000',
    target_module,
    'external_module',
    'external module: ' || target_module,
    0, 0, 0, 0
FROM (
    SELECT DISTINCT target_module
    FROM edges
    WHERE target_id IS NULL AND target_module IS NOT NULL
) AS external_modules;

-- Step 3: Update edges to point to external symbols
UPDATE edges e
SET target_id = (
    SELECT symbol_id 
    FROM symbols s 
    WHERE s.name = e.target_module 
    AND s.kind = 'external_module'
)
WHERE e.target_id IS NULL AND e.target_module IS NOT NULL;

-- Step 4: Verify no NULL target_ids remain
SELECT COUNT(*) FROM edges WHERE target_id IS NULL;
-- Should return 0
```

### 11. Testing Strategy

1. **Unit Tests**:
   - Test external symbol creation
   - Test external import detection
   - Test mapper with external dependencies

2. **Integration Tests**:
   - Index a project with external dependencies
   - Verify external symbols are created
   - Verify edges point to external symbols
   - Query external dependencies

3. **Edge Cases**:
   - Scoped packages (@types/node)
   - Relative vs absolute imports
   - Mixed internal/external imports
   - Duplicate external dependencies
