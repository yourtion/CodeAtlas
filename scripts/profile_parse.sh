#!/bin/bash

# Script to profile the parse command performance
# Usage: ./scripts/profile_parse.sh <repo-path> [workers]

set -e

REPO_PATH="${1:-tests/fixtures/test-repo}"
WORKERS="${2:-$(nproc)}"
OUTPUT_DIR="profile_results"

echo "=== CodeAtlas Parse Performance Profiling ==="
echo "Repository: $REPO_PATH"
echo "Workers: $WORKERS"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build the CLI
echo "Building CLI..."
make build-cli

# Run with CPU profiling
echo ""
echo "Running CPU profiling..."
CPUPROFILE="$OUTPUT_DIR/cpu.prof" ./bin/cli parse \
    --path "$REPO_PATH" \
    --workers "$WORKERS" \
    --output "$OUTPUT_DIR/output.json" \
    --verbose

# Run with memory profiling
echo ""
echo "Running memory profiling..."
MEMPROFILE="$OUTPUT_DIR/mem.prof" ./bin/cli parse \
    --path "$REPO_PATH" \
    --workers "$WORKERS" \
    --output "$OUTPUT_DIR/output_mem.json" \
    --verbose

# Analyze profiles
echo ""
echo "=== Profile Analysis ==="

if command -v go &> /dev/null; then
    echo ""
    echo "Top 10 CPU consumers:"
    go tool pprof -top -nodecount=10 "$OUTPUT_DIR/cpu.prof" 2>/dev/null || echo "CPU profile analysis failed"
    
    echo ""
    echo "Top 10 memory allocators:"
    go tool pprof -top -nodecount=10 "$OUTPUT_DIR/mem.prof" 2>/dev/null || echo "Memory profile analysis failed"
    
    echo ""
    echo "Profile files saved to $OUTPUT_DIR/"
    echo "View CPU profile: go tool pprof -http=:8080 $OUTPUT_DIR/cpu.prof"
    echo "View memory profile: go tool pprof -http=:8080 $OUTPUT_DIR/mem.prof"
fi

# Parse output statistics
if [ -f "$OUTPUT_DIR/output.json" ]; then
    echo ""
    echo "=== Parse Statistics ==="
    
    # Extract statistics using jq if available
    if command -v jq &> /dev/null; then
        echo "Total files: $(jq '.metadata.total_files' "$OUTPUT_DIR/output.json")"
        echo "Success count: $(jq '.metadata.success_count' "$OUTPUT_DIR/output.json")"
        echo "Failure count: $(jq '.metadata.failure_count' "$OUTPUT_DIR/output.json")"
        echo "Total symbols: $(jq '[.files[].symbols | length] | add' "$OUTPUT_DIR/output.json")"
        echo "Total relationships: $(jq '.relationships | length' "$OUTPUT_DIR/output.json")"
    else
        echo "Install jq for detailed statistics"
    fi
fi

echo ""
echo "Profiling complete!"
