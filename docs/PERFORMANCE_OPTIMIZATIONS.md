# Performance Optimizations

This document describes the performance optimizations implemented in the CodeAtlas knowledge graph indexer.

## Overview

The indexer has been optimized for high-throughput batch operations, efficient memory usage, and scalable concurrent processing. These optimizations enable the system to handle large codebases (10,000+ files) efficiently.

## Key Optimizations

### 1. Connection Pooling

**Location:** `pkg/models/database.go`

The database connection pool is configured with optimal settings:

- **Max Open Connections:** 25 (configurable via `DB_MAX_OPEN_CONNS`)
- **Max Idle Connections:** 5 (configurable via `DB_MAX_IDLE_CONNS`)
- **Connection Max Lifetime:** 5 minutes (configurable via `DB_CONN_MAX_LIFETIME`)
- **Connection Max Idle Time:** 5 minutes (configurable via `DB_CONN_MAX_IDLE_TIME`)

**Features:**
- Automatic connection reuse
- Connection lifetime management
- Pool statistics monitoring via `GetPoolStats()`
- Logging of pool metrics

**Configuration:**
```bash
export DB_MAX_OPEN_CONNS=25
export DB_MAX_IDLE_CONNS=5
export DB_CONN_MAX_LIFETIME=5m
export DB_CONN_MAX_IDLE_TIME=5m
```

### 2. Batch Processing with Adaptive Sizing

**Location:** `internal/indexer/batch_optimizer.go`

The batch optimizer dynamically adjusts batch sizes based on observed latencies:

- **Initial Batch Size:** 10-100 items
- **Target Latency:** 500ms per batch
- **Adaptive Algorithm:** 
  - Increases batch size by 20% when latency < target/2
  - Decreases batch size by 20% when latency > target*2

**Benefits:**
- Optimal throughput for varying data sizes
- Automatic adaptation to system load
- Reduced database round trips

**Usage:**
```go
optimizer := NewBatchOptimizer(DefaultBatchOptimizerConfig())
batchSize := optimizer.GetBatchSize()
// Process batch...
optimizer.RecordLatency(latency)
```

### 3. Streaming for Large AST Trees

**Location:** `internal/indexer/streaming.go`

The stream processor handles large data sets with memory management:

- **Memory Limits:** Configurable max memory usage (default: 512 MB)
- **Backpressure:** Automatic throttling when memory threshold reached
- **Goroutine Limits:** Configurable max concurrent goroutines
- **Batch Processing:** Processes data in chunks to avoid loading entire trees

**Features:**
- Memory usage estimation per entity
- Automatic garbage collection when memory pressure is high
- Context cancellation support
- Real-time memory statistics

**Configuration:**
```go
config := &StreamConfig{
    MaxMemoryMB:   512,
    MaxGoroutines: runtime.NumCPU() * 2,
    BatchSize:     100,
}
processor := NewStreamProcessor(config)
```

### 4. Worker Pool for Parallel Processing

**Location:** `internal/indexer/pool.go`

The worker pool manages concurrent task execution:

- **Configurable Workers:** Default 4 workers (based on CPU cores)
- **Task Queue:** Buffered channel for smooth task distribution
- **Error Collection:** Aggregates errors from all workers
- **Context Support:** Graceful cancellation of all workers

**Benefits:**
- Efficient CPU utilization
- Controlled concurrency
- Error handling without blocking

**Usage:**
```go
pool := NewWorkerPool(ctx, workerCount)
pool.Submit(func(ctx context.Context) error {
    // Task logic
    return nil
})
pool.Wait() // Wait for all tasks to complete
```

### 5. Database Query Optimizations

**Location:** `docker/initdb/02_performance_indexes.sql`

Comprehensive indexing strategy for optimal query performance:

#### Composite Indexes
- `idx_files_repo_language` - File lookups by repository and language
- `idx_symbols_file_kind` - Symbol lookups by file and kind
- `idx_edges_source_type_target` - Edge traversal optimization

#### Partial Indexes
- `idx_symbols_with_docstring` - Symbols with documentation (for embedding)
- `idx_edges_with_target` - Internal edges only
- `idx_edges_external` - External references only

#### Covering Indexes
- `idx_symbols_name_covering` - Includes commonly accessed fields
- `idx_files_repo_covering` - Reduces table lookups

#### Expression Indexes
- `idx_symbols_name_lower` - Case-insensitive symbol search
- `idx_files_path_lower` - Case-insensitive path search

#### Vector Indexes
- IVFFlat index for pgvector similarity search
- Configurable lists parameter based on data size

**Statistics Targets:**
- Increased to 1000 for frequently queried columns
- Improves query planner accuracy

### 6. Bulk Insert Optimizations

**Location:** `pkg/models/database.go`

Database session optimizations for bulk operations:

```go
db.OptimizeForBulkInserts(ctx)
// Perform bulk inserts...
db.ResetOptimizations(ctx)
```

**Settings:**
- `work_mem = 256MB` - Better sort performance
- `maintenance_work_mem = 512MB` - Faster index creation
- `synchronous_commit = off` - Higher throughput (use with caution)

**Post-Insert Maintenance:**
- `ANALYZE` - Updates table statistics
- `VACUUM` - Reclaims storage and updates statistics

## Performance Metrics

### Monitoring

The indexer provides real-time performance statistics:

```go
stats := indexer.GetPerformanceStats()
// Returns:
// - Memory usage and pressure
// - Goroutine usage and pressure
// - Current batch size and average latency
// - Connection pool statistics
```

### Expected Performance

Based on standard hardware (4-core CPU, 16GB RAM, SSD):

- **Throughput:** 100+ files/second
- **Memory Usage:** < 100MB per 1000 files
- **Batch Latency:** < 1 second per 100 symbols
- **Concurrent Workers:** 4-8 optimal

### Scaling Guidelines

#### Small Codebases (< 1,000 files)
```bash
INDEXER_BATCH_SIZE=50
INDEXER_WORKER_COUNT=2
DB_MAX_OPEN_CONNS=10
```

#### Medium Codebases (1,000 - 10,000 files)
```bash
INDEXER_BATCH_SIZE=100
INDEXER_WORKER_COUNT=4
DB_MAX_OPEN_CONNS=25
```

#### Large Codebases (> 10,000 files)
```bash
INDEXER_BATCH_SIZE=200
INDEXER_WORKER_COUNT=8
DB_MAX_OPEN_CONNS=50
EMBEDDING_BATCH_SIZE=100
```

## Best Practices

### 1. Incremental Indexing
Use incremental mode for re-indexing to skip unchanged files:
```bash
cli index --incremental --path /repo
```

### 2. Skip Vectors for Testing
Disable embedding generation during development:
```bash
cli index --skip-vectors --path /repo
```

### 3. Monitor Memory Usage
Check memory statistics during indexing:
```go
memStats := streamProcessor.GetMemoryStats()
log.Printf("Memory pressure: %.1f%%", memStats.MemoryPressure())
```

### 4. Tune Batch Sizes
Adjust batch sizes based on your data characteristics:
- Larger batches for simple symbols
- Smaller batches for complex AST trees
- Monitor average latency and adjust

### 5. Database Maintenance
Run periodic maintenance:
```sql
VACUUM ANALYZE repositories;
VACUUM ANALYZE files;
VACUUM ANALYZE symbols;
```

### 6. Vector Index Creation
Create vector indexes after initial data load:
```sql
-- For 100k vectors
CREATE INDEX idx_vectors_embedding ON vectors 
  USING ivfflat (embedding vector_cosine_ops) WITH (lists = 316);

-- For 1M vectors
CREATE INDEX idx_vectors_embedding ON vectors 
  USING ivfflat (embedding vector_cosine_ops) WITH (lists = 1000);
```

## Troubleshooting

### High Memory Usage
- Reduce `INDEXER_BATCH_SIZE`
- Reduce `INDEXER_WORKER_COUNT`
- Lower `MaxMemoryMB` in stream config

### Slow Indexing
- Increase `INDEXER_BATCH_SIZE`
- Increase `INDEXER_WORKER_COUNT`
- Increase `DB_MAX_OPEN_CONNS`
- Check database indexes are created

### Connection Pool Exhaustion
- Increase `DB_MAX_OPEN_CONNS`
- Reduce `INDEXER_WORKER_COUNT`
- Check for connection leaks

### Database Deadlocks
- Reduce concurrent workers
- Use transactions for related operations
- Check for circular dependencies in edges

## Future Optimizations

Potential areas for further optimization:

1. **Prepared Statement Caching** - Reuse prepared statements across batches
2. **COPY Protocol** - Use PostgreSQL COPY for even faster bulk inserts
3. **Parallel Embedding Generation** - Distribute embedding work across multiple API endpoints
4. **Incremental Graph Updates** - Update only changed graph nodes/edges
5. **Compression** - Compress large text fields (docstrings, summaries)
6. **Partitioning** - Partition large tables by repository or date

## References

- [PostgreSQL Connection Pooling](https://www.postgresql.org/docs/current/runtime-config-connection.html)
- [pgvector Performance Tuning](https://github.com/pgvector/pgvector#performance)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Database Indexing Best Practices](https://www.postgresql.org/docs/current/indexes.html)
