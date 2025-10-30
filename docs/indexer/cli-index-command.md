# CLI Index Command

The `index` command indexes parsed code into the knowledge graph, enabling semantic search, graph traversal, and relational queries.

## Table of Contents

- [Overview](#overview)
- [Basic Usage](#basic-usage)
- [Command-Line Flags](#command-line-flags)
- [Examples](#examples)
- [Workflows](#workflows)
- [Performance Tips](#performance-tips)
- [Troubleshooting](#troubleshooting)

## Overview

The index command:

1. Parses code (if `--path` is provided) or loads existing parse output
2. Validates the parsed output against the unified schema
3. Sends the data to the API server via HTTP
4. Displays progress and results

## Basic Usage

### Index from Repository Path

```bash
# Parse and index in one command
./bin/cli index --path /path/to/repo --api-url http://localhost:8080
```

### Index from Parse Output File

```bash
# First parse
./bin/cli parse --path /path/to/repo --output parsed.json

# Then index
./bin/cli index --input parsed.json --api-url http://localhost:8080
```

### Index with Repository Metadata

```bash
./bin/cli index \
  --path /path/to/repo \
  --repo-name "my-project" \
  --repo-url "https://github.com/user/my-project" \
  --branch "main" \
  --api-url http://localhost:8080
```

## Command-Line Flags

### Required Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--path`, `-p` | Repository path to parse and index | `--path ./myproject` |
| `--input`, `-i` | Path to existing parse output JSON | `--input parsed.json` |

**Note**: Either `--path` or `--input` must be specified, but not both.

### API Configuration

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--api-url` | API server URL | `http://localhost:8080` | `--api-url http://api.example.com` |
| `--api-token` | Authentication token | `` | `--api-token token123` |
| `--timeout` | Request timeout | `5m` | `--timeout 10m` |

### Repository Metadata

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--repo-id` | Repository UUID | auto-generated | `--repo-id 550e8400-e29b-41d4-a716-446655440000` |
| `--repo-name`, `-n` | Repository name | directory name | `--repo-name my-project` |
| `--repo-url`, `-u` | Repository URL | `` | `--repo-url https://github.com/user/repo` |
| `--branch`, `-b` | Git branch | `main` | `--branch develop` |
| `--commit-hash` | Git commit hash | auto-detected | `--commit-hash abc123` |

### Indexing Options

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--incremental` | Only process changed files | `false` | `--incremental` |
| `--skip-vectors` | Skip embedding generation | `false` | `--skip-vectors` |
| `--batch-size` | Batch size for processing | `100` | `--batch-size 200` |
| `--embedding-model` | Override embedding model | `` | `--embedding-model text-embedding-3-small` |

### Parser Options (when using --path)

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--language`, `-l` | Filter by language | all | `--language go` |
| `--workers`, `-w` | Number of parser workers | CPU count | `--workers 4` |
| `--verbose`, `-v` | Enable verbose logging | `false` | `--verbose` |

### Output Options

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--quiet`, `-q` | Suppress progress output | `false` | `--quiet` |
| `--json` | Output results as JSON | `false` | `--json` |

## Examples

### Example 1: Basic Indexing

```bash
./bin/cli index --path ./myproject --api-url http://localhost:8080
```

**Output**:
```
Parsing repository...
Parsed 150 files, 1250 symbols, 3400 edges

Indexing to knowledge graph...
Repository ID: 550e8400-e29b-41d4-a716-446655440000
Files processed: 150
Symbols created: 1250
Edges created: 3400
Vectors created: 1250
Duration: 45.2s

✓ Indexing completed successfully
```

### Example 2: Incremental Update

```bash
# Initial index
./bin/cli index --path ./myproject --api-url http://localhost:8080

# Make changes to code
# ...

# Re-index only changed files
./bin/cli index --path ./myproject --incremental --api-url http://localhost:8080
```

**Output**:
```
Parsing repository...
Detected 5 changed files (145 unchanged, skipped)
Parsed 5 files, 42 symbols, 87 edges

Indexing to knowledge graph...
Files processed: 5
Symbols created: 42
Edges created: 87
Vectors created: 42
Duration: 3.1s

✓ Incremental indexing completed successfully
```

### Example 3: Fast Indexing Without Embeddings

```bash
./bin/cli index \
  --path ./large-project \
  --skip-vectors \
  --batch-size 200 \
  --workers 8 \
  --api-url http://localhost:8080
```

**Output**:
```
Parsing repository...
Parsed 5000 files, 45000 symbols, 120000 edges

Indexing to knowledge graph...
Files processed: 5000
Symbols created: 45000
Edges created: 120000
Vectors created: 0 (skipped)
Duration: 3m 15s

✓ Indexing completed successfully
```

### Example 4: Index with Authentication

```bash
export API_TOKEN=my-secret-token

./bin/cli index \
  --path ./myproject \
  --api-url https://api.example.com \
  --api-token $API_TOKEN
```

Or:

```bash
./bin/cli index \
  --path ./myproject \
  --api-url https://api.example.com \
  --api-token my-secret-token
```

### Example 5: Index from Existing Parse Output

```bash
# Parse once
./bin/cli parse --path ./myproject --output parsed.json

# Index multiple times (e.g., to different servers)
./bin/cli index --input parsed.json --api-url http://dev.example.com
./bin/cli index --input parsed.json --api-url http://staging.example.com
```

### Example 6: Index with Full Metadata

```bash
./bin/cli index \
  --path ./myproject \
  --repo-name "CodeAtlas" \
  --repo-url "https://github.com/yourtionguo/CodeAtlas" \
  --branch "main" \
  --commit-hash "$(git rev-parse HEAD)" \
  --api-url http://localhost:8080
```

### Example 7: JSON Output for Automation

```bash
./bin/cli index \
  --path ./myproject \
  --api-url http://localhost:8080 \
  --json > index-result.json
```

**Output** (index-result.json):
```json
{
  "repo_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "success",
  "files_processed": 150,
  "symbols_created": 1250,
  "edges_created": 3400,
  "vectors_created": 1250,
  "errors": [],
  "duration": "45.2s"
}
```

### Example 8: Index Specific Language

```bash
./bin/cli index \
  --path ./polyglot-project \
  --language go \
  --api-url http://localhost:8080
```

## Workflows

### Workflow 1: Initial Repository Setup

```bash
# 1. Start API server
make run-api

# 2. Index repository
./bin/cli index \
  --path /path/to/repo \
  --repo-name "my-project" \
  --repo-url "https://github.com/user/my-project" \
  --api-url http://localhost:8080

# 3. Verify indexing
curl http://localhost:8080/api/v1/repositories
```

### Workflow 2: Continuous Integration

```bash
#!/bin/bash
# ci-index.sh

# Parse code
./bin/cli parse --path . --output parsed.json

# Index to staging
./bin/cli index \
  --input parsed.json \
  --repo-name "$CI_PROJECT_NAME" \
  --repo-url "$CI_PROJECT_URL" \
  --branch "$CI_COMMIT_BRANCH" \
  --commit-hash "$CI_COMMIT_SHA" \
  --api-url "$STAGING_API_URL" \
  --api-token "$STAGING_API_TOKEN"

# Index to production (on main branch)
if [ "$CI_COMMIT_BRANCH" = "main" ]; then
  ./bin/cli index \
    --input parsed.json \
    --repo-name "$CI_PROJECT_NAME" \
    --repo-url "$CI_PROJECT_URL" \
    --branch "$CI_COMMIT_BRANCH" \
    --commit-hash "$CI_COMMIT_SHA" \
    --api-url "$PROD_API_URL" \
    --api-token "$PROD_API_TOKEN"
fi
```

### Workflow 3: Incremental Updates

```bash
#!/bin/bash
# update-index.sh

# Pull latest changes
git pull

# Re-index only changed files
./bin/cli index \
  --path . \
  --incremental \
  --api-url http://localhost:8080

echo "Index updated successfully"
```

### Workflow 4: Two-Phase Indexing

```bash
# Phase 1: Fast indexing without embeddings
./bin/cli index \
  --path ./large-project \
  --skip-vectors \
  --api-url http://localhost:8080

# Phase 2: Generate embeddings later
# (This would require a separate command or API call)
curl -X POST http://localhost:8080/api/v1/repositories/$REPO_ID/generate-embeddings
```

## Performance Tips

### 1. Use Incremental Indexing

For repositories that are frequently updated:

```bash
./bin/cli index --path . --incremental --api-url http://localhost:8080
```

**Benefits**:
- Only processes changed files
- 10-100x faster for small changes
- Reduces API server load

### 2. Skip Embeddings Initially

For large repositories, skip embeddings during initial indexing:

```bash
./bin/cli index --path . --skip-vectors --api-url http://localhost:8080
```

**Benefits**:
- 5-10x faster indexing
- Reduces API costs (if using external embedding service)
- Can generate embeddings later

### 3. Optimize Batch Size

Adjust batch size based on repository size:

```bash
# Small repositories (< 100 files)
./bin/cli index --path . --batch-size 50 --api-url http://localhost:8080

# Large repositories (> 1000 files)
./bin/cli index --path . --batch-size 200 --api-url http://localhost:8080
```

### 4. Increase Parser Workers

For CPU-bound parsing:

```bash
./bin/cli index --path . --workers 8 --api-url http://localhost:8080
```

### 5. Parse Once, Index Multiple Times

If indexing to multiple environments:

```bash
# Parse once
./bin/cli parse --path . --output parsed.json

# Index to multiple servers
./bin/cli index --input parsed.json --api-url http://dev.example.com
./bin/cli index --input parsed.json --api-url http://staging.example.com
./bin/cli index --input parsed.json --api-url http://prod.example.com
```

## Troubleshooting

### Issue 1: Connection Refused

**Error**:
```
Error: failed to connect to API server: connection refused
```

**Solutions**:
1. Check if API server is running:
   ```bash
   curl http://localhost:8080/health
   ```

2. Start API server:
   ```bash
   make run-api
   ```

3. Verify API URL:
   ```bash
   ./bin/cli index --path . --api-url http://localhost:8080
   ```

### Issue 2: Authentication Failed

**Error**:
```
Error: authentication failed: invalid token
```

**Solutions**:
1. Check if authentication is enabled on server:
   ```bash
   echo $ENABLE_AUTH
   ```

2. Provide valid token:
   ```bash
   ./bin/cli index --path . --api-token your-token --api-url http://localhost:8080
   ```

3. Verify token is in server's AUTH_TOKENS:
   ```bash
   echo $AUTH_TOKENS
   ```

### Issue 3: Timeout

**Error**:
```
Error: request timeout after 5m0s
```

**Solutions**:
1. Increase timeout:
   ```bash
   ./bin/cli index --path . --timeout 10m --api-url http://localhost:8080
   ```

2. Use smaller batch size:
   ```bash
   ./bin/cli index --path . --batch-size 50 --api-url http://localhost:8080
   ```

3. Skip embeddings:
   ```bash
   ./bin/cli index --path . --skip-vectors --api-url http://localhost:8080
   ```

### Issue 4: Validation Errors

**Error**:
```
Error: validation failed: invalid parse output
```

**Solutions**:
1. Verify parse output is valid:
   ```bash
   ./bin/cli parse --path . --output parsed.json
   cat parsed.json | jq .
   ```

2. Check for parsing errors:
   ```bash
   ./bin/cli parse --path . --verbose
   ```

3. Re-parse with latest CLI version:
   ```bash
   make build-cli
   ./bin/cli parse --path . --output parsed.json
   ```

### Issue 5: Partial Failures

**Output**:
```
Files processed: 145/150
Errors: 5
```

**Solutions**:
1. Check error details in output
2. Fix syntax errors in source files
3. Re-run with `--verbose` for more details:
   ```bash
   ./bin/cli index --path . --verbose --api-url http://localhost:8080
   ```

### Issue 6: Out of Memory

**Error**:
```
Error: signal: killed (out of memory)
```

**Solutions**:
1. Reduce batch size:
   ```bash
   ./bin/cli index --path . --batch-size 25 --api-url http://localhost:8080
   ```

2. Reduce parser workers:
   ```bash
   ./bin/cli index --path . --workers 2 --api-url http://localhost:8080
   ```

3. Index subdirectories separately:
   ```bash
   ./bin/cli index --path ./backend --api-url http://localhost:8080
   ./bin/cli index --path ./frontend --api-url http://localhost:8080
   ```

## Environment Variables

The index command respects these environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `CODEATLAS_API_URL` | Default API URL | `export CODEATLAS_API_URL=http://localhost:8080` |
| `CODEATLAS_API_TOKEN` | Default API token | `export CODEATLAS_API_TOKEN=token123` |
| `CODEATLAS_BATCH_SIZE` | Default batch size | `export CODEATLAS_BATCH_SIZE=200` |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | API connection error |
| 4 | Authentication error |
| 5 | Validation error |
| 6 | Timeout |

## Next Steps

- **[CLI Search Command](./cli-search-command.md)** - Search indexed code
- **[API Reference](./api-reference.md)** - API endpoint documentation
- **[Configuration](./configuration.md)** - Configuration options
- **[Troubleshooting](./troubleshooting.md)** - Detailed troubleshooting guide
