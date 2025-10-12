# Error Handling and Logging Implementation

## Overview

This document describes the comprehensive error handling and logging system implemented for the CodeAtlas parse command, fulfilling the requirements of task 16.

## Implementation Summary

### 1. Detailed Error Types

#### DetailedParseError
A new error type was added to the parser package that captures detailed error information:

```go
type DetailedParseError struct {
    File    string  // File path where error occurred
    Line    int     // Line number (if available)
    Column  int     // Column number (if available)
    Message string  // Error message
    Type    string  // Error type: filesystem, parse, mapping
}
```

This error type implements the `error` interface and provides formatted error messages with location information when available.

### 2. Error Collection Throughout Pipeline

#### Parser Level
- **Go Parser**: Enhanced to return `DetailedParseError` for file system errors and parse errors
- **JavaScript/TypeScript Parser**: Enhanced with detailed error reporting
- **Python Parser**: Enhanced with detailed error reporting

All parsers now:
- Return detailed errors for file read failures (filesystem errors)
- Return detailed errors for syntax errors (parse errors)
- Continue processing and return partial results even when errors occur

#### Parser Pool Level
- Collects errors from all workers
- Preserves error information while continuing to process remaining files
- Logs errors through the progress logger interface
- Returns both parsed files and errors for graceful degradation

#### Command Level
- Collects parse errors from the parser pool
- Collects mapping errors from the schema mapper
- Converts `DetailedParseError` instances to `schema.ParseError` for JSON output
- Includes all errors in the output metadata

### 3. Enhanced Verbose Logging

#### Logger Enhancements
The existing logger already supported verbose mode with:
- Info, Warn, Error, and Debug levels
- Debug messages only shown in verbose mode
- Formatted output with timestamps

#### Command-Level Logging
Enhanced the parse command with detailed logging:

```go
// Timing information
logger.Info("Starting parsing with %d workers", cmd.Workers)
startTime := time.Now()
// ... parsing ...
parseTime := time.Since(startTime)
logger.Info("Parsed %d files successfully, %d errors in %v", len(parsedFiles), len(parseErrors), parseTime)
logger.Debug("Average time per file: %v", parseTime/time.Duration(len(files)))

// Schema mapping progress
logger.Debug("Starting schema mapping for %d files", len(parsedFiles))
for i, parsedFile := range parsedFiles {
    logger.Debug("[%d/%d] Mapping file: %s", i+1, len(parsedFiles), parsedFile.Path)
    // ... mapping ...
    logger.Debug("Mapped %d symbols and %d edges from %s", len(schemaFile.Symbols), len(edges), parsedFile.Path)
}

// Error collection
logger.Debug("Total errors collected: %d (%d parse errors, %d mapping errors)", 
    len(allErrors), len(parseErrors), len(mappingErrors))
```

### 4. Enhanced Summary Statistics

The `printSummary` function was significantly enhanced to provide comprehensive statistics:

#### File Statistics
- Total files scanned
- Successfully parsed count
- Failed count
- Success rate percentage

#### Symbol Statistics
- Total symbols extracted
- Breakdown by symbol type (function, class, interface, etc.)

#### Relationship Statistics
- Total relationships extracted
- Breakdown by edge type (import, call, extends, etc.)

#### Error Breakdown
- Count by error type (filesystem, parse, mapping, etc.)
- Detailed error list with file, line, column, and message
- Shows first 10 errors with indication of additional errors

Example output:
```
=== Parse Summary ===
Version: 1.0.0
Timestamp: 2025-10-12T14:00:46+08:00

Files:
  Total files scanned: 10
  Successfully parsed: 8
  Failed: 2
  Success rate: 80.0%

Symbols extracted:
  Total: 45
  function: 25
  class: 12
  interface: 5
  package: 3

Relationships extracted:
  Total: 67
  import: 32
  call: 28
  extends: 5
  implements: 2

Error breakdown:
  parse: 2

Error details (showing first 10):
  - invalid.go:10:5: unexpected token (parse)
  - broken.py: syntax error (parse)
```

### 5. Graceful Degradation

The system ensures graceful degradation at multiple levels:

#### File System Level
- Scanner continues even if some files cannot be accessed
- Logs warnings for inaccessible files but continues scanning

#### Parse Level
- Parsers return partial results even when syntax errors occur
- Tree-sitter can extract valid nodes before the error location
- Errors are collected but don't stop processing of other files

#### Worker Pool Level
- Individual worker failures don't affect other workers
- Errors are collected and reported but processing continues
- All files are attempted even if some fail

#### Command Level
- Parse command succeeds even with errors
- Outputs valid JSON with partial results
- Includes error information in metadata
- Returns success exit code if any files were parsed successfully

## Testing

### Unit Tests

1. **DetailedParseError Tests** (`internal/parser/error_test.go`)
   - Tests error formatting with and without line/column information
   - Tests error type preservation

2. **Parse Error Handling Tests** (`internal/parser/error_test.go`)
   - Tests file system errors (non-existent files)
   - Tests error type classification
   - Tests parser pool error collection

3. **Command Error Tests** (`cmd/cli/parse_command_error_test.go`)
   - Tests summary output formatting
   - Tests error collection in pipeline
   - Tests verbose logging output

### Integration Tests

1. **Error Handling Integration** (`tests/cli/error_handling_test.go`)
   - Tests complete pipeline with mixed valid/invalid files
   - Tests error collection and reporting
   - Tests JSON serialization with errors
   - Verifies error types are preserved

2. **Graceful Degradation** (`tests/cli/error_handling_test.go`)
   - Tests that parsing continues with multiple errors
   - Verifies valid files are parsed successfully
   - Verifies partial results are returned

## Requirements Coverage

### Requirement 6.1: File Read Errors
✅ Implemented: Files that cannot be read generate filesystem errors and processing continues

### Requirement 6.2: Parse Errors
✅ Implemented: Syntax errors are logged with location information and processing continues

### Requirement 6.3: Output Errors
✅ Implemented: Output file write errors return non-zero exit code

### Requirement 6.4: Verbose Mode
✅ Implemented: Detailed progress and statistics with `--verbose` flag

### Requirement 6.5: Summary Output
✅ Implemented: Comprehensive summary with file counts, symbol counts, relationship counts, and error breakdown

## Error Types

The system uses the following error types defined in `schema.ErrorType`:

- `ErrorFileSystem`: File access errors (read, stat, etc.)
- `ErrorParse`: Syntax errors during parsing
- `ErrorMapping`: Errors during AST to schema transformation
- `ErrorLLM`: Errors during semantic enhancement (future)
- `ErrorOutput`: Errors writing output files

## Usage Examples

### Basic Usage with Error Handling
```bash
# Parse with verbose output to see detailed error information
codeatlas parse --path /path/to/repo --verbose

# Parse and save output even with errors
codeatlas parse --path /path/to/repo --output output.json
```

### Interpreting Error Output

The JSON output includes an `errors` array in the metadata:

```json
{
  "metadata": {
    "version": "1.0.0",
    "total_files": 10,
    "success_count": 8,
    "failure_count": 2,
    "errors": [
      {
        "file": "broken.go",
        "line": 42,
        "column": 5,
        "message": "unexpected token",
        "type": "parse"
      }
    ]
  }
}
```

## Performance Impact

The error handling implementation has minimal performance impact:

- Error collection uses efficient data structures (slices)
- Detailed error creation only occurs when errors happen
- Verbose logging is conditional and only active when enabled
- Summary statistics are computed only once at the end

## Future Enhancements

Potential improvements for error handling:

1. **Error Recovery**: Implement more sophisticated error recovery in parsers
2. **Error Severity Levels**: Distinguish between warnings and errors
3. **Error Filtering**: Allow users to filter errors by type or severity
4. **Error Export**: Export errors to separate file for analysis
5. **Error Metrics**: Track error patterns across multiple runs
