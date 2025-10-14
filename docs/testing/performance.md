# Parse Command Performance Guide

## Performance Targets

The `codeatlas parse` command is designed to meet the following performance targets:

- **Throughput**: Parse 1000+ files in under 5 minutes
- **Memory**: Use less than 2GB RAM for 1000 files
- **Scalability**: Linear scaling with worker count up to CPU core count
- **File Scanning**: Scan 1000+ files in under 10 seconds

## Benchmark Results

### Parser Pool Performance

Benchmark results on Apple M2 (8 cores):

| Workers | Time (50 files) | Speedup | Memory/op |
|---------|-----------------|---------|-----------|
| 1       | 179ms           | 1.0x    | 10.1 MB   |
| 2       | 96ms            | 1.9x    | 10.1 MB   |
| 4       | 52ms            | 3.5x    | 10.1 MB   |
| 8       | 40ms            | 4.5x    | 10.1 MB   |

### Large Repository Performance

Test with 1000 files (mixed Go, JavaScript, Python):

- **Parse Time**: ~800ms (1,243 files/sec)
- **Schema Mapping**: ~329ms
- **Total Time**: ~1.13 seconds
- **Success Rate**: 100%
- **Memory Usage**: <1 MB heap after GC

### Individual Parser Performance

| Parser     | Time/op | Memory/op | Allocs/op |
|------------|---------|-----------|-----------|
| Go         | 1.9ms   | 23 KB     | 296       |
| JavaScript | ~2.0ms  | ~25 KB    | ~300      |
| Python     | ~2.1ms  | ~26 KB    | ~310      |

### File Scanner Performance

- **Scan Rate**: 40,000+ files/second
- **Memory**: 449 KB for 100 files
- **Allocations**: 3,752 for 100 files

## Optimization Strategies

### 1. Worker Pool Tuning

The parser automatically optimizes worker count based on file count:

```go
// For small file counts (<10), use 2 workers
// For medium file counts (<50), use half CPU cores
// For large file counts, use all CPU cores (capped at 16)
```

You can override this with the `--workers` flag:

```bash
# Use 4 workers explicitly
codeatlas parse --path /repo --workers 4

# Use all available CPUs
codeatlas parse --path /repo --workers $(nproc)
```

### 2. Memory Optimization

The parser uses several strategies to minimize memory usage:

- **Streaming JSON Output**: Uses `json.Encoder` instead of buffering entire output
- **Garbage Collection**: Explicitly clears large data structures after use
- **File Size Limits**: Skips files larger than 1MB by default
- **Worker Isolation**: Each worker has its own parser instance

### 3. Ignore Patterns

Use ignore patterns to skip unnecessary files:

```bash
# Skip test files
codeatlas parse --path /repo --ignore-pattern "*.test.js" --ignore-pattern "*.spec.ts"

# Skip common directories (automatically done by default)
# - node_modules/
# - vendor/
# - __pycache__/
# - .git/
```

### 4. Language Filtering

Parse only specific languages to reduce processing time:

```bash
# Parse only Go files
codeatlas parse --path /repo --language go

# Parse only JavaScript/TypeScript
codeatlas parse --path /repo --language javascript
```

## Profiling

### CPU Profiling

Profile CPU usage during parsing:

```bash
CPUPROFILE=cpu.prof ./bin/cli parse --path /repo --output output.json
go tool pprof -http=:8080 cpu.prof
```

### Memory Profiling

Profile memory usage:

```bash
MEMPROFILE=mem.prof ./bin/cli parse --path /repo --output output.json
go tool pprof -http=:8080 mem.prof
```

### Using the Profile Script

A convenience script is provided:

```bash
./scripts/profile_parse.sh /path/to/repo 8
```

This will:
1. Build the CLI
2. Run CPU and memory profiling
3. Generate profile reports
4. Display top consumers
5. Save profiles to `profile_results/`

## Performance Tips

### For Large Repositories (1000+ files)

1. **Use all CPU cores**: `--workers $(nproc)`
2. **Write to file**: `--output output.json` (faster than stdout)
3. **Enable verbose mode**: `--verbose` to track progress
4. **Use ignore patterns**: Skip unnecessary files

```bash
codeatlas parse \
  --path /large/repo \
  --output output.json \
  --workers $(nproc) \
  --verbose \
  --ignore-pattern "*.test.*" \
  --ignore-pattern "vendor/**"
```

### For Small Repositories (<100 files)

1. **Use fewer workers**: `--workers 2` or `--workers 4`
2. **Skip verbose mode**: Reduces overhead
3. **Output to stdout**: For piping to other tools

```bash
codeatlas parse --path /small/repo --workers 2
```

### For CI/CD Pipelines

1. **Set explicit worker count**: Avoid auto-detection issues
2. **Use file output**: More reliable than stdout
3. **Enable error reporting**: `--verbose` for debugging

```bash
codeatlas parse \
  --path . \
  --output parse-results.json \
  --workers 4 \
  --verbose
```

## Troubleshooting Performance Issues

### Slow Parsing

**Symptom**: Parsing takes longer than expected

**Solutions**:
1. Check worker count: `--workers` should match CPU cores
2. Verify ignore patterns are working: Use `--verbose`
3. Check for large files: Files >1MB are skipped by default
4. Profile with `CPUPROFILE` to identify bottlenecks

### High Memory Usage

**Symptom**: Process uses excessive memory

**Solutions**:
1. Reduce worker count: `--workers 4` instead of 8
2. Use file output: `--output file.json` instead of stdout
3. Check file sizes: Large files consume more memory
4. Profile with `MEMPROFILE` to identify leaks

### Poor Scaling

**Symptom**: More workers don't improve performance

**Solutions**:
1. Check CPU utilization: May be I/O bound
2. Verify file count: Small file counts don't benefit from many workers
3. Check disk speed: Slow disk can bottleneck scanning
4. Use SSD if available: Significantly faster file I/O

## Running Performance Tests

### Unit Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./internal/parser

# Run specific benchmark
go test -bench=BenchmarkParserPool -benchmem ./internal/parser

# Run with longer duration
go test -bench=. -benchmem -benchtime=10s ./internal/parser
```

### Integration Tests

```bash
# Run performance validation tests
go test -v -run=TestParsePerformance ./tests/cli

# Run memory usage test
go test -v -run=TestParseMemoryUsage ./tests/cli

# Run worker scaling test
go test -v -run=TestWorkerScaling ./tests/cli

# Run all performance tests
go test -v -run=Performance ./tests/cli
```

### Performance Regression Testing

Add to CI/CD pipeline:

```bash
# Run benchmarks and save results
go test -bench=. -benchmem ./internal/parser > bench-new.txt

# Compare with baseline (requires benchstat)
benchstat bench-baseline.txt bench-new.txt
```

## Performance Metrics

### Key Metrics to Monitor

1. **Files per Second**: Throughput metric
   - Target: >200 files/sec for mixed languages
   - Measure: Total files / parse time

2. **Memory per File**: Memory efficiency
   - Target: <500 KB per file
   - Measure: Total memory / file count

3. **Worker Efficiency**: Scaling effectiveness
   - Target: >3x speedup with 8 workers vs 1
   - Measure: Time(1 worker) / Time(N workers)

4. **Success Rate**: Reliability metric
   - Target: >95% for typical codebases
   - Measure: Successful parses / total files

### Collecting Metrics

Use verbose mode to see detailed metrics:

```bash
codeatlas parse --path /repo --verbose
```

Output includes:
- Total files scanned
- Parse time and rate
- Success/failure counts
- Symbol and relationship counts
- Error breakdown by type

## Environment Variables

Performance-related environment variables:

```bash
# Set default worker count
export CODEATLAS_WORKERS=8

# Enable verbose logging
export CODEATLAS_VERBOSE=true

# Enable CPU profiling
export CPUPROFILE=cpu.prof

# Enable memory profiling
export MEMPROFILE=mem.prof
```

## Hardware Recommendations

### Minimum Requirements

- **CPU**: 2 cores
- **RAM**: 2 GB
- **Disk**: HDD with 100 MB/s read speed

### Recommended Configuration

- **CPU**: 4+ cores
- **RAM**: 4+ GB
- **Disk**: SSD with 500+ MB/s read speed

### Optimal Configuration

- **CPU**: 8+ cores
- **RAM**: 8+ GB
- **Disk**: NVMe SSD with 2000+ MB/s read speed

## Future Optimizations

Planned improvements:

1. **Incremental Parsing**: Cache results, only re-parse changed files
2. **Parallel Schema Mapping**: Map to schema concurrently with parsing
3. **Compressed Output**: Optional gzip compression for large outputs
4. **Batch Processing**: Process multiple repositories in one invocation
5. **Distributed Parsing**: Split work across multiple machines
