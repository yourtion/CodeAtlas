# Parse Command Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the `codeatlas parse` command.

## Table of Contents

- [No Files Found](#no-files-found)
- [Syntax Errors](#syntax-errors)
- [Permission Issues](#permission-issues)
- [Performance Problems](#performance-problems)
- [LLM API Errors](#llm-api-errors)
- [Output Issues](#output-issues)
- [Memory Issues](#memory-issues)
- [Ignore Pattern Problems](#ignore-pattern-problems)

---

## No Files Found

### Symptom
```
Error: No files found to parse
```

### Possible Causes
1. All files are being filtered by ignore rules
2. No supported files exist in the specified path
3. Path is incorrect
4. .gitignore is too restrictive

### Solutions

#### Check what's being ignored
```bash
codeatlas parse --path ./project --verbose
```

Look for lines like:
```
[DEBUG] Ignoring file: node_modules/package.json
[DEBUG] Ignoring directory: vendor/
```

#### Disable ignore rules temporarily
```bash
codeatlas parse --path ./project --no-ignore
```

If this works, your ignore rules are too restrictive.

#### Verify the path
```bash
# Check if path exists
ls -la ./project

# Check for source files
find ./project -name "*.go" -o -name "*.js" -o -name "*.py" | head -10
```

#### Check .gitignore
```bash
cat ./project/.gitignore
```

Common overly-restrictive patterns:
- `*` (ignores everything)
- `*.go` (ignores all Go files)
- `/` (ignores root directory)

#### Override specific ignore patterns
```bash
# Parse only Go files, ignoring .gitignore
codeatlas parse --path ./project --language go --no-ignore
```

---

## Syntax Errors

### Symptom
```json
{
  "metadata": {
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

### Understanding Parse Errors

Parse errors are **non-fatal**. The command continues processing other files and reports errors in the output metadata.

### Solutions

#### View error summary
```bash
codeatlas parse --path ./project --output result.json
jq '.metadata.errors' result.json
```

#### Fix syntax errors in source files
```bash
# For Go files
go fmt ./src/broken.go
go vet ./src/broken.go

# For JavaScript/TypeScript
npx eslint ./src/broken.js --fix

# For Python
python -m py_compile ./src/broken.py
```

#### Exclude problematic files temporarily
```bash
codeatlas parse --path ./project --ignore-pattern "broken.go"
```

#### Get detailed error information
```bash
codeatlas parse --path ./project --verbose 2>&1 | grep -A 5 "ERROR"
```

---

## Permission Issues

### Symptom
```
Error: permission denied: /path/to/file.go
Error: cannot read file: /path/to/directory
```

### Solutions

#### Check file permissions
```bash
ls -la /path/to/file.go
```

Look for read permissions (`r--` in the permission string).

#### Fix permissions
```bash
# Make file readable
chmod u+r /path/to/file.go

# Make directory and all files readable
chmod -R u+r /path/to/directory
```

#### Check ownership
```bash
ls -la /path/to/file.go
```

If the file is owned by another user:
```bash
# Change ownership (requires sudo)
sudo chown $USER /path/to/file.go

# Or copy to a location you own
cp /path/to/file.go ~/my-copy.go
codeatlas parse --file ~/my-copy.go
```

#### Run with appropriate permissions
```bash
# If you have sudo access and need to parse system files
sudo codeatlas parse --path /system/path
```

---

## Performance Problems

### Symptom
- Parsing takes too long (>10 minutes for <1000 files)
- High CPU usage
- System becomes unresponsive

### Solutions

#### Reduce worker count
```bash
# Default uses all CPU cores, try reducing
codeatlas parse --path ./project --workers 2
```

#### Exclude large directories
```bash
codeatlas parse --path ./project \
  --ignore-pattern "vendor/*" \
  --ignore-pattern "node_modules/*" \
  --ignore-pattern "dist/*"
```

#### Parse specific language only
```bash
# Only parse Go files
codeatlas parse --path ./project --language go
```

#### Process in batches
```bash
# Parse subdirectories separately
codeatlas parse --path ./project/backend --output backend.json
codeatlas parse --path ./project/frontend --output frontend.json
```

#### Monitor resource usage
```bash
# In one terminal
codeatlas parse --path ./project --verbose

# In another terminal
top -p $(pgrep codeatlas)
```

#### Benchmark different configurations
```bash
# Test with different worker counts
time codeatlas parse --path ./project --workers 1 --output /dev/null
time codeatlas parse --path ./project --workers 2 --output /dev/null
time codeatlas parse --path ./project --workers 4 --output /dev/null
time codeatlas parse --path ./project --workers 8 --output /dev/null
```

---

## LLM API Errors

### Symptom
```
[WARN] LLM API error: rate limit exceeded
[WARN] LLM API error: invalid API key
[ERROR] Failed to enhance symbol: network timeout
```

### Solutions

#### Verify API key is set
```bash
echo $CODEATLAS_LLM_API_KEY
```

If empty:
```bash
export CODEATLAS_LLM_API_KEY=sk-your-api-key-here
```

#### Check API key validity
```bash
# Test with curl
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $CODEATLAS_LLM_API_KEY"
```

#### Handle rate limits
The parser has built-in rate limiting, but you may still hit API limits.

```bash
# Parse without semantic enhancement
codeatlas parse --path ./project

# Or parse smaller batches
codeatlas parse --path ./project/module1 --semantic
codeatlas parse --path ./project/module2 --semantic
```

#### Use custom API endpoint
```bash
export CODEATLAS_LLM_API_URL=https://your-custom-endpoint.com/v1
export CODEATLAS_LLM_MODEL=gpt-3.5-turbo
codeatlas parse --path ./project --semantic
```

#### Check network connectivity
```bash
# Test connection to API
ping api.openai.com

# Test with curl
curl -I https://api.openai.com
```

#### Disable semantic enhancement
```bash
# Parse without LLM enhancement
codeatlas parse --path ./project
```

**Note**: LLM errors are non-fatal. Parsing continues without semantic summaries.

---

## Output Issues

### Symptom
- Invalid JSON output
- Truncated output
- Cannot write to file

### Solutions

#### Check disk space
```bash
df -h
```

If disk is full:
```bash
# Clean up space
rm -rf /tmp/old-files
docker system prune -a
```

#### Verify output path is writable
```bash
# Check directory permissions
ls -la $(dirname /path/to/output.json)

# Test write access
touch /path/to/output.json
```

#### Use absolute path for output
```bash
codeatlas parse --path ./project --output /absolute/path/to/output.json
```

#### Validate JSON output
```bash
codeatlas parse --path ./project --output result.json

# Validate JSON
jq empty result.json
echo $?  # Should print 0 if valid
```

#### Check for incomplete output
```bash
# Check if command completed successfully
codeatlas parse --path ./project --output result.json
echo $?  # Should print 0

# Check file size
ls -lh result.json
```

#### Redirect stderr to file
```bash
codeatlas parse --path ./project --output result.json 2> errors.log
```

---

## Memory Issues

### Symptom
```
Error: out of memory
fatal error: runtime: out of memory
```

### Solutions

#### Reduce worker count
```bash
codeatlas parse --path ./project --workers 1
```

#### Exclude large files
```bash
# Skip files larger than 1MB (default)
codeatlas parse --path ./project --ignore-pattern "*.min.js"
```

#### Parse in smaller batches
```bash
# Parse subdirectories separately
for dir in ./project/*/; do
  codeatlas parse --path "$dir" --output "$(basename $dir).json"
done
```

#### Increase system memory
```bash
# Check current memory
free -h

# Close other applications
# Or increase swap space (Linux)
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

#### Use streaming output
The parser uses streaming JSON output by default, but for very large repositories:

```bash
# Parse and immediately pipe to processing
codeatlas parse --path ./project | jq -c '.files[]' > files.jsonl
```

---

## Ignore Pattern Problems

### Symptom
- Files that should be ignored are being parsed
- Files that should be parsed are being ignored

### Solutions

#### Test ignore patterns
```bash
# See what's being ignored
codeatlas parse --path ./project --verbose 2>&1 | grep "Ignoring"
```

#### Understand pattern syntax
```bash
# Glob patterns (like .gitignore)
*.test.js       # Matches any file ending with .test.js
test/*.js       # Matches .js files in test/ directory
**/test/*.js    # Matches .js files in any test/ directory
!important.js   # Negation (don't ignore this file)
```

#### Debug .gitignore
```bash
# Check what .gitignore is doing
git check-ignore -v ./path/to/file.go
```

#### Override .gitignore
```bash
# Ignore .gitignore completely
codeatlas parse --path ./project --no-ignore

# Add custom patterns on top of .gitignore
codeatlas parse --path ./project --ignore-pattern "*.generated.go"
```

#### Use custom ignore file
```bash
# Create .customignore
cat > .customignore << EOF
*.test.js
*.spec.ts
mock_*.go
vendor/
EOF

# Use it
codeatlas parse --path ./project --ignore-file .customignore
```

#### Check pattern precedence
Patterns are applied in this order:
1. Default patterns (node_modules, .git, etc.)
2. .gitignore files (root and subdirectories)
3. Custom ignore file (--ignore-file)
4. Command-line patterns (--ignore-pattern)

Later patterns override earlier ones.

---

## Getting More Help

### Enable Debug Logging
```bash
codeatlas parse --path ./project --verbose 2>&1 | tee debug.log
```

### Check Version
```bash
codeatlas parse --version
```

### Report Issues

If you've tried the solutions above and still have issues, please report:

1. **Command used**:
   ```bash
   codeatlas parse --path ./project --verbose
   ```

2. **Error message**:
   ```
   [Copy full error message]
   ```

3. **Environment**:
   ```bash
   # OS
   uname -a
   
   # Go version
   go version
   
   # CLI version
   codeatlas parse --version
   ```

4. **Sample code** (if applicable):
   Provide a minimal example that reproduces the issue.

5. **Verbose output**:
   ```bash
   codeatlas parse --path ./project --verbose 2>&1 | tee debug.log
   ```

Submit issues at: [GitHub Issues](https://github.com/your-org/codeatlas/issues)

---

## Quick Diagnostic Commands

```bash
# Full diagnostic
codeatlas parse --path ./project --verbose 2>&1 | tee diagnostic.log

# Check what files would be parsed
find ./project -type f \( -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" \) | wc -l

# Check ignore rules
git check-ignore -v ./project/**/*

# Test with minimal options
codeatlas parse --file ./project/main.go --verbose

# Test with no restrictions
codeatlas parse --path ./project --no-ignore --workers 1 --verbose
```

---

## Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `No files found to parse` | Ignore rules too restrictive | Use `--no-ignore` or adjust patterns |
| `permission denied` | Insufficient file permissions | `chmod u+r` or run with appropriate permissions |
| `syntax error: unexpected token` | Invalid source code | Fix syntax errors in source files |
| `out of memory` | Large repository or too many workers | Reduce `--workers` or parse in batches |
| `LLM API error: rate limit` | Too many API requests | Wait or disable `--semantic` |
| `cannot write to file` | Output path not writable | Check permissions or disk space |
| `invalid JSON output` | Interrupted or corrupted output | Re-run and save to file with `--output` |

---

## Performance Benchmarks

Expected performance on typical hardware:

| Hardware | Repository Size | Time | Workers |
|----------|----------------|------|---------|
| 4-core CPU, 8GB RAM | 100 files | <10s | 4 |
| 4-core CPU, 8GB RAM | 1000 files | <2min | 4 |
| 8-core CPU, 16GB RAM | 1000 files | <1min | 8 |
| 8-core CPU, 16GB RAM | 5000 files | <5min | 8 |

If your performance is significantly worse, check:
- Disk I/O (SSD vs HDD)
- CPU usage (other processes)
- Memory availability
- Network (if using --semantic)

---

## Additional Resources

- [CLI Parse Command Documentation](./cli-parse-command.md)
- [Quick Reference Guide](./parse-command-quick-reference.md)
- [Example Output](./examples/parse-output-example.json)
- [GitHub Issues](https://github.com/your-org/codeatlas/issues)
