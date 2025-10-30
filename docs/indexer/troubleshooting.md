# Troubleshooting Guide

Common issues and solutions for the Knowledge Graph Indexer.

## Table of Contents

- [Connection Issues](#connection-issues)
- [Authentication Issues](#authentication-issues)
- [Indexing Errors](#indexing-errors)
- [Performance Issues](#performance-issues)
- [Database Issues](#database-issues)
- [Embedding Issues](#embedding-issues)
- [Graph Issues](#graph-issues)
- [Diagnostic Commands](#diagnostic-commands)

## Connection Issues

### API Server Not Responding

**Symptoms**:
```
Error: failed to connect to API server: connection refused
```

**Diagnosis**:
```bash
# Check if server is running
curl http://localhost:8080/health

# Check server logs
docker logs codeatlas-api

# Check if port is in use
lsof -i :8080
```

**Solutions**:

1. **Start the API server**:
   ```bash
   make run-api
   ```

2. **Check server configuration**:
   ```bash
   echo $API_PORT
   echo $API_HOST
   ```

3. **Verify firewall rules**:
   ```bash
   # macOS
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
   
   # Linux
   sudo ufw status
   ```

4. **Check Docker network** (if using Docker):
   ```bash
   docker network ls
   docker network inspect codeatlas_default
   ```

### Timeout Errors

**Symptoms**:
```
Error: request timeout after 5m0s
```

**Diagnosis**:
```bash
# Check server load
top

# Check database connections
psql -U codeatlas -d codeatlas -c "SELECT count(*) FROM pg_stat_activity;"

# Check network latency
ping api.example.com
```

**Solutions**:

1. **Increase timeout**:
   ```bash
   ./bin/cli index --path . --timeout 10m --api-url http://localhost:8080
   ```

2. **Reduce batch size**:
   ```bash
   ./bin/cli index --path . --batch-size 50 --api-url http://localhost:8080
   ```

3. **Skip embeddings**:
   ```bash
   ./bin/cli index --path . --skip-vectors --api-url http://localhost:8080
   ```

4. **Check server resources**:
   ```bash
   # CPU usage
   top
   
   # Memory usage
   free -h
   
   # Disk I/O
   iostat -x 1
   ```

## Authentication Issues

### Invalid Token

**Symptoms**:
```
Error: authentication failed: invalid token
HTTP 401 Unauthorized
```

**Diagnosis**:
```bash
# Check if auth is enabled
echo $ENABLE_AUTH

# Check configured tokens
echo $AUTH_TOKENS

# Test with curl
curl -H "Authorization: Bearer your-token" http://localhost:8080/api/v1/repositories
```

**Solutions**:

1. **Provide valid token**:
   ```bash
   ./bin/cli index --path . --api-token your-token --api-url http://localhost:8080
   ```

2. **Check server configuration**:
   ```bash
   # Server should have matching token
   export AUTH_TOKENS="token1,token2,token3"
   make run-api
   ```

3. **Disable authentication** (development only):
   ```bash
   export ENABLE_AUTH=false
   make run-api
   ```

### Missing Token

**Symptoms**:
```
Error: authentication required but no token provided
```

**Solutions**:

1. **Set environment variable**:
   ```bash
   export CODEATLAS_API_TOKEN=your-token
   ./bin/cli index --path . --api-url http://localhost:8080
   ```

2. **Use command-line flag**:
   ```bash
   ./bin/cli index --path . --api-token your-token --api-url http://localhost:8080
   ```

## Indexing Errors

### Validation Errors

**Symptoms**:
```
Error: validation failed: invalid parse output
Error: missing required field 'files'
Error: invalid symbol reference in edge
```

**Diagnosis**:
```bash
# Validate parse output
./bin/cli parse --path . --output parsed.json
cat parsed.json | jq .

# Check for parsing errors
./bin/cli parse --path . --verbose 2>&1 | grep -i error
```

**Solutions**:

1. **Re-parse with latest CLI**:
   ```bash
   make build-cli
   ./bin/cli parse --path . --output parsed.json
   ```

2. **Check parse output structure**:
   ```bash
   cat parsed.json | jq '.files | length'
   cat parsed.json | jq '.relationships | length'
   cat parsed.json | jq '.metadata'
   ```

3. **Fix source code syntax errors**:
   ```bash
   # Check for syntax errors in source files
   ./bin/cli parse --path . --verbose
   ```

### Partial Failures

**Symptoms**:
```
Files processed: 145/150
Symbols created: 1200/1250
Errors: 5
```

**Diagnosis**:
```bash
# Check error details
./bin/cli index --path . --verbose --api-url http://localhost:8080

# Check server logs
docker logs codeatlas-api | grep -i error
```

**Solutions**:

1. **Review error messages** in output
2. **Fix problematic files** and re-index
3. **Use incremental indexing** to retry failed files:
   ```bash
   ./bin/cli index --path . --incremental --api-url http://localhost:8080
   ```

### Duplicate Key Errors

**Symptoms**:
```
Error: duplicate key value violates unique constraint
Error: repository already exists
```

**Solutions**:

1. **Use existing repo_id**:
   ```bash
   # Get existing repo_id
   curl http://localhost:8080/api/v1/repositories
   
   # Index with existing ID
   ./bin/cli index --path . --repo-id <existing-uuid> --api-url http://localhost:8080
   ```

2. **Use incremental indexing**:
   ```bash
   ./bin/cli index --path . --incremental --api-url http://localhost:8080
   ```

3. **Delete and re-index** (development only):
   ```bash
   # Delete repository
   curl -X DELETE http://localhost:8080/api/v1/repositories/<repo-id>
   
   # Re-index
   ./bin/cli index --path . --api-url http://localhost:8080
   ```

## Performance Issues

### Slow Indexing

**Symptoms**:
- Indexing takes longer than expected
- High CPU or memory usage
- Database connection pool exhaustion

**Diagnosis**:
```bash
# Monitor system resources
top
htop

# Check database connections
psql -U codeatlas -d codeatlas -c "SELECT count(*), state FROM pg_stat_activity GROUP BY state;"

# Check database query performance
psql -U codeatlas -d codeatlas -c "SELECT query, calls, total_time, mean_time FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;"

# Profile indexing
time ./bin/cli index --path . --api-url http://localhost:8080
```

**Solutions**:

1. **Increase batch size**:
   ```bash
   ./bin/cli index --path . --batch-size 200 --api-url http://localhost:8080
   ```

2. **Increase worker count**:
   ```bash
   ./bin/cli index --path . --workers 8 --api-url http://localhost:8080
   ```

3. **Skip embeddings initially**:
   ```bash
   ./bin/cli index --path . --skip-vectors --api-url http://localhost:8080
   ```

4. **Increase database connection pool**:
   ```bash
   export DB_MAX_OPEN_CONNS=50
   export DB_MAX_IDLE_CONNS=10
   make run-api
   ```

5. **Optimize database**:
   ```sql
   -- Analyze tables
   ANALYZE files;
   ANALYZE symbols;
   ANALYZE edges;
   ANALYZE vectors;
   
   -- Vacuum tables
   VACUUM ANALYZE;
   ```

### High Memory Usage

**Symptoms**:
```
Error: signal: killed (out of memory)
```

**Diagnosis**:
```bash
# Monitor memory usage
free -h
watch -n 1 free -h

# Check process memory
ps aux | grep codeatlas
```

**Solutions**:

1. **Reduce batch size**:
   ```bash
   ./bin/cli index --path . --batch-size 25 --api-url http://localhost:8080
   ```

2. **Reduce worker count**:
   ```bash
   ./bin/cli index --path . --workers 2 --api-url http://localhost:8080
   ```

3. **Index subdirectories separately**:
   ```bash
   for dir in */; do
     ./bin/cli index --path "$dir" --api-url http://localhost:8080
   done
   ```

4. **Increase system memory** or use swap:
   ```bash
   # Add swap (Linux)
   sudo fallocate -l 4G /swapfile
   sudo chmod 600 /swapfile
   sudo mkswap /swapfile
   sudo swapon /swapfile
   ```

## Database Issues

### Connection Pool Exhausted

**Symptoms**:
```
Error: database connection pool exhausted
Error: too many connections
```

**Diagnosis**:
```bash
# Check active connections
psql -U codeatlas -d codeatlas -c "SELECT count(*) FROM pg_stat_activity;"

# Check max connections
psql -U codeatlas -d codeatlas -c "SHOW max_connections;"

# Check connection pool settings
echo $DB_MAX_OPEN_CONNS
echo $DB_MAX_IDLE_CONNS
```

**Solutions**:

1. **Increase connection pool**:
   ```bash
   export DB_MAX_OPEN_CONNS=50
   export DB_MAX_IDLE_CONNS=10
   make run-api
   ```

2. **Increase PostgreSQL max_connections**:
   ```bash
   # Edit postgresql.conf
   max_connections = 200
   
   # Restart PostgreSQL
   docker restart codeatlas-postgres
   ```

3. **Reduce concurrent workers**:
   ```bash
   export INDEXER_WORKER_COUNT=2
   make run-api
   ```

### Database Lock Timeout

**Symptoms**:
```
Error: database lock timeout
Error: deadlock detected
```

**Solutions**:

1. **Disable transactions** (not recommended for production):
   ```bash
   export INDEXER_USE_TRANSACTIONS=false
   make run-api
   ```

2. **Reduce batch size**:
   ```bash
   export INDEXER_BATCH_SIZE=50
   make run-api
   ```

3. **Check for long-running queries**:
   ```sql
   SELECT pid, now() - pg_stat_activity.query_start AS duration, query
   FROM pg_stat_activity
   WHERE state = 'active'
   ORDER BY duration DESC;
   ```

### Missing Extensions

**Symptoms**:
```
Error: extension "pgvector" does not exist
Error: extension "age" does not exist
```

**Solutions**:

1. **Install pgvector**:
   ```sql
   CREATE EXTENSION IF NOT EXISTS vector;
   ```

2. **Install AGE**:
   ```sql
   CREATE EXTENSION IF NOT EXISTS age;
   LOAD 'age';
   SET search_path = ag_catalog, "$user", public;
   ```

3. **Use Docker image** with extensions pre-installed:
   ```bash
   make docker-up
   ```

4. **Check extension installation**:
   ```sql
   SELECT * FROM pg_extension WHERE extname IN ('vector', 'age');
   ```

## Embedding Issues

### Embedding API Errors

**Symptoms**:
```
Error: embedding API request failed
Error: rate limit exceeded
Error: invalid API key
```

**Diagnosis**:
```bash
# Check API configuration
echo $EMBEDDING_API_ENDPOINT
echo $EMBEDDING_API_KEY
echo $EMBEDDING_MODEL

# Test API directly
curl -X POST $EMBEDDING_API_ENDPOINT \
  -H "Authorization: Bearer $EMBEDDING_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"input": "test", "model": "text-embedding-3-small"}'
```

**Solutions**:

1. **Verify API key**:
   ```bash
   export EMBEDDING_API_KEY=sk-your-key
   make run-api
   ```

2. **Reduce request rate**:
   ```bash
   export EMBEDDING_MAX_REQUESTS_PER_SECOND=5
   export EMBEDDING_BATCH_SIZE=25
   make run-api
   ```

3. **Use local model**:
   ```bash
   export EMBEDDING_BACKEND=openai
   export EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
   export EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
   make run-api
   ```

4. **Skip embeddings**:
   ```bash
   ./bin/cli index --path . --skip-vectors --api-url http://localhost:8080
   ```

### Dimension Mismatch

**Symptoms**:
```
Error: embedding dimension mismatch
Error: expected 768 dimensions, got 1536
```

**Solutions**:

1. **Update configuration**:
   ```bash
   export EMBEDDING_DIMENSIONS=1536
   make run-api
   ```

2. **Recreate vectors table**:
   ```sql
   DROP TABLE IF EXISTS vectors;
   CREATE TABLE vectors (
     vector_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
     entity_id UUID NOT NULL,
     entity_type VARCHAR(50) NOT NULL,
     embedding vector(1536),  -- Update dimension
     content TEXT NOT NULL,
     model VARCHAR(100) NOT NULL,
     chunk_index INT DEFAULT 0,
     created_at TIMESTAMP DEFAULT NOW()
   );
   ```

## Graph Issues

### Graph Not Created

**Symptoms**:
```
Error: graph "code_graph" does not exist
Error: AGE extension not loaded
```

**Solutions**:

1. **Load AGE extension**:
   ```sql
   LOAD 'age';
   SET search_path = ag_catalog, "$user", public;
   ```

2. **Create graph**:
   ```sql
   SELECT create_graph('code_graph');
   ```

3. **Verify graph exists**:
   ```sql
   SELECT * FROM ag_catalog.ag_graph WHERE name = 'code_graph';
   ```

### Cypher Query Errors

**Symptoms**:
```
Error: invalid Cypher query
Error: node not found
```

**Solutions**:

1. **Check graph schema**:
   ```sql
   SELECT * FROM cypher('code_graph', $$
     MATCH (n) RETURN labels(n), count(n)
   $$) as (label agtype, count agtype);
   ```

2. **Verify nodes exist**:
   ```sql
   SELECT * FROM cypher('code_graph', $$
     MATCH (n:Function) RETURN n LIMIT 10
   $$) as (node agtype);
   ```

3. **Check edge types**:
   ```sql
   SELECT * FROM cypher('code_graph', $$
     MATCH ()-[r]->() RETURN type(r), count(r)
   $$) as (type agtype, count agtype);
   ```

## Diagnostic Commands

### Check System Health

```bash
# API server health
curl http://localhost:8080/health

# Database connectivity
psql -U codeatlas -d codeatlas -c "SELECT 1;"

# Check extensions
psql -U codeatlas -d codeatlas -c "SELECT * FROM pg_extension;"

# Check tables
psql -U codeatlas -d codeatlas -c "\dt"

# Check table sizes
psql -U codeatlas -d codeatlas -c "
  SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
  FROM pg_tables
  WHERE schemaname = 'public'
  ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
"
```

### Check Indexing Status

```bash
# List repositories
curl http://localhost:8080/api/v1/repositories

# Get repository details
curl http://localhost:8080/api/v1/repositories/<repo-id>

# Count entities
psql -U codeatlas -d codeatlas -c "
  SELECT 
    'files' as entity, count(*) as count FROM files
  UNION ALL
  SELECT 'symbols', count(*) FROM symbols
  UNION ALL
  SELECT 'edges', count(*) FROM edges
  UNION ALL
  SELECT 'vectors', count(*) FROM vectors;
"
```

### Check Performance

```bash
# Database query statistics
psql -U codeatlas -d codeatlas -c "
  SELECT 
    query,
    calls,
    total_time,
    mean_time,
    max_time
  FROM pg_stat_statements
  ORDER BY total_time DESC
  LIMIT 10;
"

# Index usage
psql -U codeatlas -d codeatlas -c "
  SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
  FROM pg_stat_user_indexes
  ORDER BY idx_scan DESC;
"

# Connection pool status
psql -U codeatlas -d codeatlas -c "
  SELECT 
    state,
    count(*) as connections
  FROM pg_stat_activity
  GROUP BY state;
"
```

### Enable Debug Logging

```bash
# API server
export LOG_LEVEL=debug
make run-api

# CLI
./bin/cli index --path . --verbose --api-url http://localhost:8080

# Database
psql -U codeatlas -d codeatlas -c "ALTER SYSTEM SET log_statement = 'all';"
psql -U codeatlas -d codeatlas -c "SELECT pg_reload_conf();"
```

## Getting Help

If you can't resolve the issue:

1. **Check logs**:
   ```bash
   # API server logs
   docker logs codeatlas-api
   
   # Database logs
   docker logs codeatlas-postgres
   
   # CLI verbose output
   ./bin/cli index --path . --verbose --api-url http://localhost:8080 2>&1 | tee debug.log
   ```

2. **Gather diagnostic information**:
   ```bash
   # System info
   uname -a
   docker --version
   psql --version
   
   # Configuration
   env | grep -E '(DB_|API_|INDEXER_|EMBEDDING_)'
   
   # Resource usage
   top -b -n 1 | head -20
   free -h
   df -h
   ```

3. **Create minimal reproduction**:
   ```bash
   # Create small test repository
   mkdir test-repo
   cd test-repo
   echo 'package main\nfunc main() {}' > main.go
   
   # Try indexing
   ./bin/cli index --path . --verbose --api-url http://localhost:8080
   ```

4. **Open GitHub issue** with:
   - Error message
   - Steps to reproduce
   - Diagnostic output
   - Configuration (redact sensitive values)

## Next Steps

- **[Configuration Guide](./configuration.md)** - Tuning options
- **[Performance Tuning](./performance-tuning.md)** - Optimization strategies
- **[API Reference](./api-reference.md)** - API documentation
- **[CLI Documentation](./cli-index-command.md)** - CLI usage
