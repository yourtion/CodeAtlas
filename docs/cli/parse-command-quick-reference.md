# Parse Command Quick Reference

## Common Commands

### Basic Parsing

```bash
# Parse entire repository
codeatlas parse --path ./myproject

# Parse and save to file
codeatlas parse --path ./myproject --output ast.json

# Parse single file
codeatlas parse --file src/main.go
```

### Language-Specific Parsing

```bash
# Parse only Go files
codeatlas parse --path ./project --language go

# Parse only JavaScript/TypeScript
codeatlas parse --path ./project --language javascript

# Parse only Python files
codeatlas parse --path ./project --language python
```

### Performance Optimization

```bash
# Use 8 concurrent workers
codeatlas parse --path ./project --workers 8

# Exclude test files
codeatlas parse --path ./project --ignore-pattern "*.test.js" --ignore-pattern "*.spec.ts"

# Exclude vendor and node_modules
codeatlas parse --path ./project --ignore-pattern "vendor/*" --ignore-pattern "node_modules/*"
```

### Debugging

```bash
# Enable verbose logging
codeatlas parse --path ./project --verbose

# Save logs to file
codeatlas parse --path ./project --verbose 2>&1 | tee parse.log

# Parse without any ignore rules
codeatlas parse --path ./project --no-ignore
```

### Advanced Usage

```bash
# Use custom ignore file
codeatlas parse --path ./project --ignore-file .customignore

# Enable semantic summaries (requires API key)
export CODEATLAS_LLM_API_KEY=sk-your-key
codeatlas parse --path ./project --semantic

# Combine multiple options
codeatlas parse --path ./project \
  --language go \
  --workers 4 \
  --ignore-pattern "*.test.go" \
  --output go-ast.json \
  --verbose
```

## Output Processing with jq

```bash
# Extract all function names
codeatlas parse --path ./project | jq -r '.files[].symbols[] | select(.kind == "function") | .name'

# Count symbols by type
codeatlas parse --path ./project | jq '[.files[].symbols[].kind] | group_by(.) | map({kind: .[0], count: length})'

# Get files with errors
codeatlas parse --path ./project | jq '.metadata.errors'

# Filter Go files only
codeatlas parse --path ./project | jq '.files[] | select(.language == "go")'

# Get all import relationships
codeatlas parse --path ./project | jq '.relationships[] | select(.edge_type == "import")'
```

## Troubleshooting Quick Fixes

```bash
# No files found?
codeatlas parse --path ./project --no-ignore --verbose

# Syntax errors?
codeatlas parse --path ./project --output result.json
# Check result.json metadata.errors

# Permission denied?
chmod -R u+r ./project
codeatlas parse --path ./project

# Out of memory?
codeatlas parse --path ./project --workers 2 --ignore-pattern "vendor/*"

# LLM API errors?
export CODEATLAS_LLM_API_KEY=sk-your-key
codeatlas parse --path ./project --semantic --verbose
```

## Environment Variables

```bash
# LLM API configuration
export CODEATLAS_LLM_API_KEY=sk-your-api-key
export CODEATLAS_LLM_API_URL=https://api.openai.com/v1
export CODEATLAS_LLM_MODEL=gpt-4

# Use in parse command
codeatlas parse --path ./project --semantic
```

## Exit Codes

- `0`: Success
- `1`: General error (file not found, permission denied, etc.)
- `2`: Parse errors (some files failed to parse, but command completed)

Check exit code:
```bash
codeatlas parse --path ./project
echo $?
```

## Performance Benchmarks

Expected performance on typical hardware (4-core CPU, 8GB RAM):

| Repository Size | Files | Time | Workers |
|----------------|-------|------|---------|
| Small | <100 | <10s | 4 |
| Medium | 100-1000 | <2min | 4-8 |
| Large | 1000+ | <5min | 8 |

Optimize for your system:
```bash
# Benchmark with different worker counts
time codeatlas parse --path ./project --workers 2
time codeatlas parse --path ./project --workers 4
time codeatlas parse --path ./project --workers 8
```

## Default Ignore Patterns

Automatically ignored (unless `--no-ignore` is used):

**Directories:**
- `.git/`, `node_modules/`, `vendor/`, `__pycache__/`
- `.venv/`, `venv/`, `dist/`, `build/`

**Extensions:**
- Binary: `.exe`, `.dll`, `.so`, `.dylib`
- Images: `.jpg`, `.png`, `.gif`, `.svg`
- Archives: `.zip`, `.tar`, `.gz`

## Help

```bash
# Show help
codeatlas parse --help

# Show version
codeatlas parse --version
```

## Full Documentation

For complete documentation, see [CLI Parse Command Documentation](./cli-parse-command.md).
