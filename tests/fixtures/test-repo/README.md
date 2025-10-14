# Test Repository Fixtures

This directory contains test fixtures for integration testing of the CodeAtlas parse command.

## Structure

```
test-repo/
├── .gitignore                  # Root gitignore with test patterns
├── main.go                     # Go file with functions, structs, methods
├── utils.go                    # Go file with interfaces, constants
├── models.py                   # Python file with classes, functions, decorators
├── utils.py                    # Python file with decorators, module constants
├── app.js                      # JavaScript file with classes, functions, arrow functions
├── api.js                      # JavaScript file with exports, async functions
├── ignored_file.go             # File that should be ignored by .gitignore
├── syntax_error.go             # Go file with intentional syntax errors
├── syntax_error.py             # Python file with intentional syntax errors
└── subdir/
    ├── .gitignore              # Nested gitignore
    ├── helper.go               # Go file in subdirectory
    └── local_config.js         # File ignored by nested .gitignore
```

## Test Coverage

### Language Support
- **Go**: Functions, methods, structs, interfaces, constants
- **Python**: Classes, functions, decorators, async functions, docstrings
- **JavaScript**: Classes, functions, arrow functions, async functions, exports/imports

### Ignore Rules Testing
- Root `.gitignore` patterns
- Nested `.gitignore` patterns
- Default ignore patterns (node_modules, __pycache__, etc.)
- Custom ignore patterns via CLI flags

### Error Handling Testing
- Files with syntax errors (`syntax_error.go`, `syntax_error.py`)
- Graceful degradation and partial results

### Symbol Extraction Testing
Each file contains various constructs to test:
- Function declarations
- Class/struct definitions
- Method definitions
- Import/export statements
- Decorators (Python)
- Arrow functions (JavaScript)
- Async functions
- Docstrings and comments

## Usage in Tests

These fixtures are used by the integration tests in `tests/cli/`:
- `parse_test.go` - End-to-end parsing tests
- `parse_ignore_test.go` - Ignore rules tests
- `parse_concurrent_test.go` - Concurrent processing tests
- `parse_error_test.go` - Error recovery tests
- `parse_single_file_test.go` - Single file parsing tests
- `parse_language_filter_test.go` - Language filter tests
