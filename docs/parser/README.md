# Parser Documentation

CodeAtlas uses Tree-sitter based parsers to analyze source code and extract symbols, dependencies, and relationships. This documentation covers all supported languages and parsing capabilities.

## Supported Languages

| Language | Extensions | Status | Special Features |
|----------|-----------|--------|------------------|
| **Go** | `.go` | ✅ Stable | Package management, interfaces, goroutines |
| **JavaScript** | `.js`, `.jsx` | ✅ Stable | ES6+, JSX support |
| **TypeScript** | `.ts`, `.tsx` | ✅ Stable | Type definitions, decorators |
| **Python** | `.py` | ✅ Stable | Decorators, async/await |
| **Kotlin** | `.kt`, `.kts` | ✅ Stable | Data classes, coroutines, extensions |
| **Java** | `.java` | ✅ Stable | Annotations, generics, inner classes |
| **Swift** | `.swift` | ✅ Stable | Protocols, extensions, property observers |
| **Objective-C** | `.h`, `.m` | ✅ Stable | Header-implementation pairing, categories |
| **C** | `.h`, `.c` | ✅ Stable | Header-implementation pairing, preprocessor |
| **C++** | `.hpp`, `.cpp`, `.cc`, `.cxx` | ✅ Stable | Templates, namespaces, multiple inheritance |

## Language-Specific Documentation

- [Kotlin Parser](kotlin-parser.md) - Android development language support
- [Java Parser](java-parser.md) - Enterprise and Android language support
- [Swift Parser](swift-parser.md) - iOS development language support
- [Objective-C Parser](objc-parser.md) - Legacy iOS and mixed codebases
- [C Parser](c-parser.md) - Native code and system programming
- [C++ Parser](cpp-parser.md) - Object-oriented native code

## Key Concepts

### Symbols

Symbols are code elements extracted from source files:

- **Functions/Methods**: Callable code units with signatures
- **Classes/Structs**: Type definitions with members
- **Interfaces/Protocols**: Contract definitions
- **Properties/Fields**: Data members
- **Enums**: Enumeration types
- **Packages/Modules**: Code organization units

### Dependencies

Dependencies represent relationships between code elements:

- **Import/Include**: File-level dependencies
- **Call**: Function/method invocation relationships
- **Extends/Implements**: Inheritance and interface implementation
- **Conforms**: Protocol conformance (Swift/Objective-C)
- **Implements_Declaration**: Header-implementation relationships (C/C++/Objective-C)

### Header-Implementation Association

For C, C++, and Objective-C, CodeAtlas intelligently pairs header files with implementation files and creates cross-file relationships. See [Header-Implementation Association](header-implementation-association.md) for details.

## Parser Architecture

All parsers follow a consistent architecture:

```
File Scanner → Language Detector → Parser Router → Language Parser → Symbol Extraction → Dependency Extraction → Knowledge Graph
```

### Common Features

1. **Error Resilience**: Parsers return partial results even on syntax errors
2. **Documentation Extraction**: Language-specific doc comments (KDoc, Javadoc, Swift docs, Doxygen)
3. **Incremental Parsing**: Efficient updates using Tree-sitter's incremental parsing
4. **Parallel Processing**: Multiple files parsed concurrently

## Usage Examples

### Parsing a Single File

```bash
# Parse a Kotlin file
./bin/cli parse --path MyClass.kt --output result.json

# Parse a Swift file
./bin/cli parse --path ViewController.swift --output result.json

# Parse C++ with header
./bin/cli parse --path MyClass.cpp --output result.json
```

### Parsing a Project

```bash
# Parse entire Android project
./bin/cli parse --path /path/to/android-project --output android-result.json

# Parse iOS project
./bin/cli parse --path /path/to/ios-project --output ios-result.json

# Parse mixed C/C++ project
./bin/cli parse --path /path/to/native-project --output native-result.json
```

### Language Detection

CodeAtlas automatically detects languages based on file extensions:

```bash
# Automatically detects Kotlin
./bin/cli parse --path src/

# For .h files, content-based detection determines C/C++/Objective-C
./bin/cli parse --path include/
```

## Performance

Parser performance varies by language complexity:

| Language | Files/sec | Notes |
|----------|-----------|-------|
| Go | ~100 | Fast, simple syntax |
| JavaScript/TypeScript | ~80 | Complex AST |
| Python | ~90 | Indentation-sensitive |
| Kotlin | ~70 | Rich language features |
| Java | ~75 | Verbose syntax |
| Swift | ~65 | Complex type system |
| Objective-C | ~60 | Header pairing overhead |
| C | ~85 | Simple syntax |
| C++ | ~55 | Templates, complex syntax |

See [Performance Testing](../testing/mobile-parsers-performance.md) for detailed benchmarks.

## Migration Guide

If you have existing projects indexed with CodeAtlas, the mobile language parsers are automatically available. Simply re-index your projects:

```bash
# Re-index with new parser support
./bin/cli index --path /path/to/your/project --api-url http://localhost:8080
```

No schema changes are required - new edge types are automatically supported.

## Troubleshooting

### Common Issues

**Issue**: Parser fails on valid code
- **Solution**: Check Tree-sitter grammar version, report issue with code sample

**Issue**: Header-implementation pairing not working
- **Solution**: Ensure header and implementation files have matching base names

**Issue**: Slow parsing on large projects
- **Solution**: Use `--workers` flag to increase parallelism

See [Parse Troubleshooting](../cli/parse-troubleshooting.md) for more details.

## API Reference

For programmatic access to parsers, see:
- [Parser API Reference](../api/README.md)
- [Indexer API Reference](../indexer/api-reference.md)

## Contributing

To add support for a new language:

1. Add Tree-sitter grammar dependency
2. Create language-specific parser in `internal/parser/`
3. Implement symbol and dependency extraction
4. Add language detection rules
5. Write comprehensive tests
6. Update documentation

See [Development Guide](../development/testing.md) for details.
