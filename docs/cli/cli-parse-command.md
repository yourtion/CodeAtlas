# CodeAtlas Parse Command Documentation

## Overview

The `codeatlas parse` command analyzes source code files and outputs structured JSON AST (Abstract Syntax Tree) representations. It supports Go, JavaScript/TypeScript, Python, Kotlin, Java, Swift, Objective-C, C, and C++ languages, leveraging Tree-sitter for accurate parsing.

## Table of Contents

- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [Command-Line Flags](#command-line-flags)
- [Environment Variables](#environment-variables)
- [Output Format](#output-format)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)
- [Performance Tips](#performance-tips)

---

## Installation

Build the CLI tool:

```bash
make build-cli
```

The binary will be available at `bin/cli`.

---

## Basic Usage

### Parse a Repository

```bash
codeatlas parse --path /path/to/repository
```

### Parse a Single File

```bash
codeatlas parse --file /path/to/file.go
```

### Save Output to File

```bash
codeatlas parse --path /path/to/repository --output result.json
```

---

## Command-Line Flags

### Required Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--path`, `-p` | Path to repository or directory to parse | `--path ./myproject` |
| `--file`, `-f` | Path to a single file to parse | `--file main.go` |

**Note**: Either `--path` or `--file` must be specified, but not both.

### Optional Flags

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--output`, `-o` | Output file path (stdout if not specified) | stdout | `--output result.json` |
| `--language`, `-l` | Filter files by language (go, javascript, typescript, python, kotlin, java, swift, objective-c, c, c++) | all | `--language go` |
| `--workers`, `-w` | Number of concurrent workers | CPU count | `--workers 4` |
| `--verbose`, `-v` | Enable verbose logging | false | `--verbose` |
| `--ignore-file` | Path to custom ignore file | none | `--ignore-file .customignore` |
| `--ignore-pattern` | Additional ignore patterns (can be repeated) | none | `--ignore-pattern "*.test.js"` |
| `--no-ignore` | Disable all ignore rules (including .gitignore) | false | `--no-ignore` |
| `--semantic` | Enable LLM-based semantic summaries (requires API key) | false | `--semantic` |
| `--help`, `-h` | Display help information | - | `--help` |

---

## Environment Variables

### LLM Enhancement (Optional)

| Variable | Description | Required | Example |
|----------|-------------|----------|---------|
| `CODEATLAS_LLM_API_KEY` | API key for LLM semantic enhancement | Only if `--semantic` is used | `export CODEATLAS_LLM_API_KEY=sk-...` |
| `CODEATLAS_LLM_API_URL` | Custom LLM API endpoint | No | `export CODEATLAS_LLM_API_URL=https://api.openai.com/v1` |
| `CODEATLAS_LLM_MODEL` | LLM model to use | No | `export CODEATLAS_LLM_MODEL=gpt-4` |

---

## Output Format

The parse command outputs JSON conforming to the CodeAtlas Unified Schema.

### Top-Level Structure

```json
{
  "files": [...],
  "relationships": [...],
  "metadata": {...}
}
```

### Example Output

```json
{
  "files": [
    {
      "file_id": "550e8400-e29b-41d4-a716-446655440000",
      "path": "src/main.go",
      "language": "go",
      "size": 1024,
      "checksum": "a3b2c1d4e5f6...",
      "nodes": [
        {
          "node_id": "660e8400-e29b-41d4-a716-446655440001",
          "file_id": "550e8400-e29b-41d4-a716-446655440000",
          "type": "function_declaration",
          "span": {
            "start_line": 10,
            "end_line": 25,
            "start_byte": 200,
            "end_byte": 450
          },
          "text": "func main() { ... }"
        }
      ],
      "symbols": [
        {
          "symbol_id": "770e8400-e29b-41d4-a716-446655440002",
          "file_id": "550e8400-e29b-41d4-a716-446655440000",
          "name": "main",
          "kind": "function",
          "signature": "func main()",
          "span": {
            "start_line": 10,
            "end_line": 25,
            "start_byte": 200,
            "end_byte": 450
          },
          "docstring": "main is the entry point of the application"
        }
      ]
    }
  ],
  "relationships": [
    {
      "edge_id": "880e8400-e29b-41d4-a716-446655440003",
      "source_id": "770e8400-e29b-41d4-a716-446655440002",
      "target_id": "990e8400-e29b-41d4-a716-446655440004",
      "edge_type": "call",
      "source_file": "src/main.go",
      "target_file": "src/utils.go"
    }
  ],
  "metadata": {
    "version": "1.0.0",
    "timestamp": "2025-10-12T10:30:00Z",
    "total_files": 150,
    "success_count": 145,
    "failure_count": 5,
    "errors": [
      {
        "file": "src/broken.go",
        "line": 42,
        "message": "syntax error: unexpected token",
        "type": "parse"
      }
    ]
  }
}
```

### Schema Reference

#### File Object

| Field | Type | Description |
|-------|------|-------------|
| `file_id` | string | Unique UUID for the file |
| `path` | string | Relative path from repository root |
| `language` | string | Programming language (go, javascript, typescript, python, kotlin, java, swift, objc, c, cpp) |
| `size` | integer | File size in bytes |
| `checksum` | string | SHA256 checksum of file content |
| `nodes` | array | AST nodes extracted from the file |
| `symbols` | array | High-level code entities (functions, classes, etc.) |

#### Symbol Object

| Field | Type | Description |
|-------|------|-------------|
| `symbol_id` | string | Unique UUID for the symbol |
| `file_id` | string | UUID of the containing file |
| `name` | string | Symbol name |
| `kind` | string | Symbol type: function, class, interface, variable, package, module |
| `signature` | string | Full signature (e.g., "func Add(a int, b int) int") |
| `span` | object | Source location (start_line, end_line, start_byte, end_byte) |
| `docstring` | string | Documentation comment (optional) |
| `semantic_summary` | string | LLM-generated summary (optional, requires --semantic) |

#### Relationship Object

| Field | Type | Description |
|-------|------|-------------|
| `edge_id` | string | Unique UUID for the relationship |
| `source_id` | string | Source symbol UUID |
| `target_id` | string | Target symbol UUID |
| `edge_type` | string | Relationship type: import, call, extends, implements, reference |
| `source_file` | string | Source file path |
| `target_file` | string | Target file path (optional) |
| `target_module` | string | External module name (optional, for imports) |

---

## Examples

### Example 1: Parse Go Repository

```bash
codeatlas parse --path ./mygoproject --language go --output go-ast.json
```

**Output**: All Go files parsed, saved to `go-ast.json`

### Example 2: Parse with Custom Ignore Patterns

```bash
codeatlas parse --path ./project \
  --ignore-pattern "*.test.js" \
  --ignore-pattern "*.spec.ts" \
  --ignore-pattern "mock_*.go"
```

**Output**: Excludes test files and mocks from parsing

### Example 3: Parse Single File with Verbose Output

```bash
codeatlas parse --file src/main.go --verbose
```

**Output**:
```
[INFO] Starting parse command
[INFO] Parsing single file: src/main.go
[INFO] Detected language: go
[INFO] Parsing file: src/main.go
[DEBUG] Extracted 5 symbols
[DEBUG] Extracted 3 relationships
[INFO] Parse complete: 1 file, 0 errors
```

### Example 4: Parse with Multiple Workers

```bash
codeatlas parse --path ./large-repo --workers 8 --verbose
```

**Output**: Uses 8 concurrent workers for faster processing

### Example 5: Parse with Semantic Enhancement

```bash
export CODEATLAS_LLM_API_KEY=sk-your-api-key
codeatlas parse --path ./project --semantic --output enhanced.json
```

**Output**: Includes LLM-generated semantic summaries for functions and classes

### Example 6: Parse TypeScript Project

```bash
codeatlas parse --path ./typescript-app --language typescript
```

**Output**: Only TypeScript files (.ts, .tsx) are parsed

### Example 7: Disable All Ignore Rules

```bash
codeatlas parse --path ./project --no-ignore
```

**Output**: Parses all files, including those in .gitignore, node_modules, etc.

### Example 8: Parse Android Project (Kotlin/Java)

```bash
codeatlas parse --path ./android-app --language kotlin --output android-kotlin.json
```

**Output**: Parses only Kotlin files from an Android project

```bash
codeatlas parse --path ./android-app --output android-full.json
```

**Output**: Parses both Kotlin and Java files from an Android project

### Example 9: Parse iOS Project (Swift/Objective-C)

```bash
codeatlas parse --path ./ios-app --language swift --output ios-swift.json
```

**Output**: Parses only Swift files from an iOS project

```bash
codeatlas parse --path ./ios-app --output ios-full.json
```

**Output**: Parses both Swift and Objective-C files from an iOS project

### Example 10: Parse Native C/C++ Project

```bash
codeatlas parse --path ./native-lib --language c++ --output cpp-lib.json
```

**Output**: Parses only C++ files (.cpp, .hpp, etc.)

```bash
codeatlas parse --path ./native-lib --output native-full.json
```

**Output**: Parses both C and C++ files, with automatic header file detection

---

## Troubleshooting

### Common Issues

#### 1. "No files found to parse"

**Cause**: All files are being filtered by ignore rules or no supported files exist.

**Solutions**:
- Check if .gitignore is too restrictive
- Use `--no-ignore` to disable ignore rules
- Use `--verbose` to see which files are being skipped
- Verify the path is correct and contains source files

```bash
# Debug with verbose output
codeatlas parse --path ./project --verbose
```

#### 2. "Syntax error: unexpected token"

**Cause**: File contains syntax errors that Tree-sitter cannot parse.

**Solutions**:
- Fix syntax errors in the source file
- The parser will continue with other files and report errors in metadata
- Check the error summary in the output JSON

```bash
# Parse continues despite errors
codeatlas parse --path ./project --output result.json
# Check result.json metadata.errors for details
```

#### 3. "Permission denied" or "Cannot read file"

**Cause**: Insufficient permissions to read files or directories.

**Solutions**:
- Check file permissions: `ls -la /path/to/file`
- Run with appropriate permissions
- Ensure the path is accessible

```bash
# Check permissions
ls -la ./project

# Fix permissions if needed
chmod -R u+r ./project
```

#### 4. "Out of memory" or Slow Performance

**Cause**: Large repository or too many concurrent workers.

**Solutions**:
- Reduce worker count: `--workers 2`
- Parse specific language only: `--language go`
- Exclude large directories: `--ignore-pattern "vendor/*"`
- Increase system memory or use streaming output

```bash
# Optimize for large repositories
codeatlas parse --path ./huge-repo \
  --workers 2 \
  --ignore-pattern "vendor/*" \
  --ignore-pattern "node_modules/*"
```

#### 5. "LLM API error" (when using --semantic)

**Cause**: API key not set, rate limiting, or network issues.

**Solutions**:
- Verify API key is set: `echo $CODEATLAS_LLM_API_KEY`
- Check API rate limits
- Reduce concurrent requests (parser has built-in rate limiting)
- Parsing continues without semantic summaries on API errors

```bash
# Set API key
export CODEATLAS_LLM_API_KEY=sk-your-key

# Verify it's set
echo $CODEATLAS_LLM_API_KEY

# Run with semantic enhancement
codeatlas parse --path ./project --semantic
```

#### 6. "Invalid JSON output"

**Cause**: Output was interrupted or corrupted.

**Solutions**:
- Always use `--output` flag for large repositories
- Check disk space: `df -h`
- Verify the command completed successfully (exit code 0)

```bash
# Save to file and check exit code
codeatlas parse --path ./project --output result.json
echo $?  # Should print 0 for success
```

### Debug Mode

Enable verbose logging to diagnose issues:

```bash
codeatlas parse --path ./project --verbose 2>&1 | tee parse.log
```

This saves all output to `parse.log` for analysis.

### Getting Help

If you encounter issues not covered here:

1. Check the error message in the output metadata
2. Run with `--verbose` for detailed logs
3. Review the [GitHub Issues](https://github.com/your-org/codeatlas/issues)
4. Open a new issue with:
   - Command used
   - Error message
   - Verbose output
   - Sample code that reproduces the issue

---

## Performance Tips

### 1. Optimize Worker Count

The default worker count is the number of CPU cores. Adjust based on your system:

```bash
# For CPU-bound systems
codeatlas parse --path ./project --workers 4

# For I/O-bound systems (many small files)
codeatlas parse --path ./project --workers 8
```

### 2. Use Language Filters

Parse only the languages you need:

```bash
codeatlas parse --path ./project --language go
```

### 3. Exclude Unnecessary Files

Use ignore patterns to skip large or irrelevant files:

```bash
codeatlas parse --path ./project \
  --ignore-pattern "vendor/*" \
  --ignore-pattern "node_modules/*" \
  --ignore-pattern "*.min.js"
```

### 4. Process Large Repositories in Batches

For very large repositories, consider parsing subdirectories separately:

```bash
codeatlas parse --path ./project/backend --output backend.json
codeatlas parse --path ./project/frontend --output frontend.json
```

### 5. Benchmark Performance

Measure parsing performance:

```bash
time codeatlas parse --path ./project --output result.json
```

**Expected Performance**:
- Small projects (<100 files): <10 seconds
- Medium projects (100-1000 files): <2 minutes
- Large projects (1000+ files): <5 minutes

### 6. Monitor Resource Usage

Use system monitoring tools:

```bash
# Monitor CPU and memory
top -p $(pgrep codeatlas)

# Or use htop for better visualization
htop -p $(pgrep codeatlas)
```

---

## Advanced Usage

### Custom Ignore File Format

Create a custom ignore file (similar to .gitignore):

```
# .customignore
*.test.js
*.spec.ts
mock_*.go
vendor/
node_modules/
```

Use it with:

```bash
codeatlas parse --path ./project --ignore-file .customignore
```

### Combining with Other Tools

#### Pipe to jq for JSON Processing

```bash
codeatlas parse --path ./project | jq '.files[] | select(.language == "go")'
```

#### Count Symbols by Type

```bash
codeatlas parse --path ./project | jq '[.files[].symbols[].kind] | group_by(.) | map({kind: .[0], count: length})'
```

#### Extract All Function Names

```bash
codeatlas parse --path ./project | jq -r '.files[].symbols[] | select(.kind == "function") | .name'
```

### Integration with CI/CD

Add to your CI pipeline to track code structure changes:

```yaml
# .github/workflows/parse.yml
name: Parse Codebase
on: [push]
jobs:
  parse:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Parse code
        run: |
          make build-cli
          ./bin/cli parse --path . --output ast.json
      - name: Upload artifact
        uses: actions/upload-artifact@v2
        with:
          name: ast-output
          path: ast.json
```

---

## Supported Languages

| Language | Extensions | Features |
|----------|-----------|----------|
| Go | .go | Packages, imports, functions, methods, structs, interfaces, types |
| JavaScript | .js, .jsx | ES6 modules, CommonJS, functions, classes, arrow functions |
| TypeScript | .ts, .tsx | All JavaScript features + type annotations |
| Python | .py | Imports, functions, classes, decorators, type hints, docstrings |
| Kotlin | .kt, .kts | Packages, imports, classes, data classes, sealed classes, functions, properties, interfaces |
| Java | .java | Packages, imports, classes, interfaces, enums, annotations, methods, fields |
| Swift | .swift | Imports, classes, structs, enums, protocols, extensions, functions, properties |
| Objective-C | .h, .m, .mm | Imports, interfaces, implementations, protocols, categories, properties, methods |
| C | .c, .h | Includes, functions, structs, unions, enums, typedefs, macros, global variables |
| C++ | .cpp, .cc, .cxx, .hpp, .hh, .hxx | Includes, namespaces, classes, templates, functions, methods, operators, inheritance |

---

## Default Ignore Patterns

The following patterns are ignored by default (unless `--no-ignore` is used):

### Directories
- `.git/`
- `node_modules/`
- `vendor/`
- `__pycache__/`
- `.venv/`
- `venv/`
- `dist/`
- `build/`
- `.next/`
- `.nuxt/`

### File Extensions
- Binary files: `.exe`, `.dll`, `.so`, `.dylib`, `.a`
- Images: `.jpg`, `.jpeg`, `.png`, `.gif`, `.svg`, `.ico`
- Documents: `.pdf`, `.doc`, `.docx`
- Archives: `.zip`, `.tar`, `.gz`, `.rar`
- Media: `.mp3`, `.mp4`, `.avi`, `.mov`

---

## Version Information

Check the CLI version:

```bash
codeatlas parse --version
```

Or:

```bash
codeatlas --version
```

---

## License

MIT License - See [LICENSE](../LICENSE) for details.
