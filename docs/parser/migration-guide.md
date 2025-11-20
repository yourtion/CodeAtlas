# Migration Guide: Mobile Language Support

This guide helps users with existing CodeAtlas projects migrate to the new mobile language parser support.

## Overview

CodeAtlas now supports parsing for mobile development languages:
- **Kotlin** - Android development
- **Java** - Android and enterprise development
- **Swift** - iOS development
- **Objective-C** - Legacy iOS and mixed codebases
- **C** - Native code and system programming
- **C++** - Object-oriented native code

## What's New

### New Features

1. **Mobile Language Parsing**
   - Full symbol extraction for Kotlin, Java, Swift, Objective-C, C, C++
   - Language-specific documentation extraction (KDoc, Javadoc, Swift docs, Doxygen)
   - Dependency and relationship tracking

2. **Header-Implementation Association**
   - Automatic pairing of header and implementation files (C/C++/Objective-C)
   - Cross-file symbol matching
   - Declaration-to-implementation relationships

3. **Enhanced Language Detection**
   - Automatic detection of mobile language files
   - Content-based detection for ambiguous `.h` files
   - Support for multiple file extensions per language

4. **New Edge Types**
   - `implements_header` - File-to-file relationships
   - `implements_declaration` - Symbol-to-symbol relationships
   - `declares_in_header` - Symbol-to-file relationships
   - `calls_declaration` - Call-to-declaration relationships

## Compatibility

### Backward Compatibility

✅ **Fully backward compatible** - No breaking changes to existing functionality:

- Existing Go, JavaScript/TypeScript, and Python parsing unchanged
- Existing database schema unchanged
- Existing API endpoints unchanged
- Existing CLI commands unchanged

### Database Schema

✅ **No schema changes required** - New edge types use existing `edges` table:

```sql
-- Existing schema supports new edge types
CREATE TABLE edges (
    id SERIAL PRIMARY KEY,
    source_id INTEGER NOT NULL,
    target_id INTEGER NOT NULL,
    edge_type VARCHAR(50) NOT NULL,
    repository_id INTEGER NOT NULL,
    -- ... other fields
);
```

New edge types (`implements_header`, `implements_declaration`, etc.) are automatically supported.

## Migration Steps

### Step 1: Update CodeAtlas

Update to the latest version of CodeAtlas:

```bash
# Pull latest changes
git pull origin main

# Rebuild binaries
make build

# Verify version
./bin/cli --version
```

### Step 2: Re-index Projects

Re-index your existing projects to take advantage of new parsers:

```bash
# Re-index a project
./bin/cli index --path /path/to/your/project --api-url http://localhost:8080

# For mobile projects specifically
./bin/cli index --path /path/to/android-project --api-url http://localhost:8080
./bin/cli index --path /path/to/ios-project --api-url http://localhost:8080
```

**Note**: Re-indexing is optional for existing projects. New language support is automatically available.

### Step 3: Verify Parsing

Verify that mobile language files are being parsed:

```bash
# Parse a single file to test
./bin/cli parse --path MyClass.kt --output test.json

# Check output
cat test.json | jq '.symbols'
```

### Step 4: Update Queries (Optional)

If you have custom queries, you may want to update them to use new edge types:

**Before** (generic call relationships):
```sql
SELECT * FROM edges WHERE edge_type = 'call';
```

**After** (distinguish header vs implementation calls):
```sql
-- Calls to header declarations
SELECT * FROM edges WHERE edge_type = 'calls_declaration';

-- All calls (including implementation)
SELECT * FROM edges WHERE edge_type IN ('call', 'calls_declaration');
```

## Language-Specific Migration

### Android Projects (Kotlin/Java)

**Before**: Only Java files parsed (if using older version)

**After**: Both Kotlin and Java files fully supported

**Migration**:
```bash
# Re-index Android project
./bin/cli index --path /path/to/android-project --api-url http://localhost:8080

# Verify Kotlin files parsed
curl http://localhost:8080/api/v1/files?language=kotlin
```

**New Capabilities**:
- Kotlin coroutines and suspend functions
- Data classes and sealed classes
- Extension functions
- Java annotations and generics

### iOS Projects (Swift/Objective-C)

**Before**: No iOS language support

**After**: Full Swift and Objective-C support with header-implementation pairing

**Migration**:
```bash
# Re-index iOS project
./bin/cli index --path /path/to/ios-project --api-url http://localhost:8080

# Verify Swift files parsed
curl http://localhost:8080/api/v1/files?language=swift

# Verify Objective-C files parsed
curl http://localhost:8080/api/v1/files?language=objc
```

**New Capabilities**:
- Swift protocols and extensions
- SwiftUI views
- Objective-C categories and protocols
- Automatic .h/.m file pairing

### Native Code Projects (C/C++)

**Before**: No C/C++ support

**After**: Full C/C++ support with header-implementation pairing

**Migration**:
```bash
# Re-index native project
./bin/cli index --path /path/to/native-project --api-url http://localhost:8080

# Verify C files parsed
curl http://localhost:8080/api/v1/files?language=c

# Verify C++ files parsed
curl http://localhost:8080/api/v1/files?language=cpp
```

**New Capabilities**:
- C++ templates and namespaces
- Multiple inheritance
- Operator overloading
- Automatic header-implementation pairing

## Common Migration Scenarios

### Scenario 1: Mixed Language Project

**Example**: Android project with Kotlin, Java, and native C++ code

**Migration**:
```bash
# Single command indexes all languages
./bin/cli index --path /path/to/mixed-project --api-url http://localhost:8080
```

**Result**: All languages parsed automatically, relationships tracked across language boundaries.

### Scenario 2: Legacy iOS Project

**Example**: iOS project with Objective-C and Swift files

**Migration**:
```bash
# Index entire project
./bin/cli index --path /path/to/ios-project --api-url http://localhost:8080
```

**Result**: 
- Objective-C .h/.m files automatically paired
- Swift-Objective-C interop tracked
- Protocol conformance relationships captured

### Scenario 3: Native Library

**Example**: C++ library with headers and implementations

**Migration**:
```bash
# Index library
./bin/cli index --path /path/to/cpp-library --api-url http://localhost:8080
```

**Result**:
- Header files paired with implementations
- Template instantiations tracked
- Namespace relationships captured

## Troubleshooting

### Issue: Files Not Being Parsed

**Symptoms**: Mobile language files not appearing in results

**Solutions**:

1. **Check file extensions**:
   ```bash
   # Verify file has correct extension
   ls -la *.kt *.java *.swift *.m
   ```

2. **Check language detection**:
   ```bash
   # Test single file
   ./bin/cli parse --path MyClass.kt --output test.json
   cat test.json | jq '.language'
   ```

3. **Check for syntax errors**:
   ```bash
   # Parser returns partial results on errors
   cat test.json | jq '.errors'
   ```

### Issue: Header-Implementation Not Paired

**Symptoms**: C/C++/Objective-C files not showing relationships

**Solutions**:

1. **Check file naming**:
   - Header: `MyClass.h`
   - Implementation: `MyClass.c`, `MyClass.cpp`, or `MyClass.m`
   - Base names must match

2. **Check file locations**:
   - Files should be in same directory
   - Or follow common patterns (e.g., `include/` and `src/`)

3. **Verify pairing**:
   ```bash
   # Check for implements_header edges
   curl http://localhost:8080/api/v1/edges?type=implements_header
   ```

### Issue: Slow Parsing

**Symptoms**: Parsing takes longer than expected

**Solutions**:

1. **Use parallel processing**:
   ```bash
   # Increase worker count
   ./bin/cli parse --path /path/to/project --workers 8
   ```

2. **Check file sizes**:
   ```bash
   # Large files take longer
   find . -name "*.cpp" -size +1M
   ```

3. **Monitor performance**:
   ```bash
   # Enable performance logging
   export LOG_LEVEL=debug
   ./bin/cli parse --path /path/to/project
   ```

### Issue: Missing Dependencies

**Symptoms**: Import/include relationships not captured

**Solutions**:

1. **Check import statements**:
   ```bash
   # Verify imports in parsed output
   cat test.json | jq '.dependencies[] | select(.type=="import")'
   ```

2. **Check file paths**:
   - Relative imports must be resolvable
   - System imports are marked as external

3. **Verify classification**:
   ```bash
   # Check internal vs external classification
   cat test.json | jq '.dependencies[] | select(.external==true)'
   ```

## Performance Considerations

### Parsing Speed

Expected parsing speeds for mobile languages:

| Language | Files/sec | Notes |
|----------|-----------|-------|
| Kotlin | ~70 | Rich language features |
| Java | ~75 | Verbose syntax |
| Swift | ~65 | Complex type system |
| Objective-C | ~60 | Header pairing overhead |
| C | ~85 | Simple syntax |
| C++ | ~55 | Templates, complex syntax |

### Memory Usage

Expected memory usage:

| Language | MB per 1000 LOC | Notes |
|----------|----------------|-------|
| Kotlin | ~5 | Moderate |
| Java | ~6 | Verbose |
| Swift | ~7 | Complex AST |
| Objective-C | ~8 | Header pairing |
| C | ~5 | Simple |
| C++ | ~10 | Templates |

### Optimization Tips

1. **Increase parallelism**:
   ```bash
   ./bin/cli parse --workers 8
   ```

2. **Filter by language**:
   ```bash
   ./bin/cli parse --languages kotlin,java
   ```

3. **Exclude test files**:
   ```bash
   ./bin/cli parse --exclude "**/test/**"
   ```

## API Changes

### New Query Parameters

**List files by language**:
```bash
# Get all Kotlin files
curl http://localhost:8080/api/v1/files?language=kotlin

# Get all Swift files
curl http://localhost:8080/api/v1/files?language=swift
```

**Filter edges by type**:
```bash
# Get header-implementation relationships
curl http://localhost:8080/api/v1/edges?type=implements_header

# Get declaration-implementation relationships
curl http://localhost:8080/api/v1/edges?type=implements_declaration
```

### New Response Fields

**ParsedFile** response includes language:
```json
{
  "path": "MyClass.kt",
  "language": "kotlin",
  "symbols": [...],
  "dependencies": [...]
}
```

**Edge** response includes new types:
```json
{
  "source_id": 123,
  "target_id": 456,
  "edge_type": "implements_header",
  "repository_id": 1
}
```

## Best Practices

### 1. Re-index After Updates

Always re-index projects after updating CodeAtlas:

```bash
./bin/cli index --path /path/to/project --api-url http://localhost:8080
```

### 2. Use Language Filters

For large projects, filter by language to speed up queries:

```bash
# Parse only mobile languages
./bin/cli parse --languages kotlin,java,swift,objc
```

### 3. Monitor Performance

Track parsing performance over time:

```bash
# Enable performance metrics
export ENABLE_METRICS=true
./bin/cli parse --path /path/to/project
```

### 4. Validate Results

Always validate parsing results:

```bash
# Check for errors
cat result.json | jq '.errors'

# Verify symbol count
cat result.json | jq '.symbols | length'

# Check dependency count
cat result.json | jq '.dependencies | length'
```

## Getting Help

### Documentation

- [Parser Overview](README.md)
- [Kotlin Parser](kotlin-parser.md)
- [Java Parser](java-parser.md)
- [Swift Parser](swift-parser.md)
- [Objective-C Parser](objc-parser.md)
- [C Parser](c-parser.md)
- [C++ Parser](cpp-parser.md)
- [Header-Implementation Association](header-implementation-association.md)

### Support

- **GitHub Issues**: Report bugs or request features
- **Documentation**: Check docs/ directory for detailed guides
- **Examples**: See tests/fixtures/ for example code

## Rollback

If you encounter issues, you can rollback to the previous version:

```bash
# Checkout previous version
git checkout <previous-version-tag>

# Rebuild
make build

# Existing data remains unchanged
```

**Note**: No data migration is required, so rollback is safe.

## Summary

✅ **No breaking changes** - Fully backward compatible

✅ **No schema changes** - Existing database works as-is

✅ **Automatic detection** - New languages detected automatically

✅ **Optional re-indexing** - Re-index to take advantage of new features

✅ **Enhanced capabilities** - Mobile language support with header-implementation pairing

The migration is straightforward: update CodeAtlas, optionally re-index your projects, and start using mobile language support immediately.
