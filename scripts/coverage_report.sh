#!/bin/bash
# Generate HTML coverage reports from existing coverage data

set -e

echo "Generating HTML coverage reports..."

GENERATED=0

if [ -f coverage_unit.out ]; then
    go tool cover -html=coverage_unit.out -o coverage_unit.html
    echo "✓ Unit coverage report: coverage_unit.html"
    ((GENERATED++))
fi

if [ -f coverage_integration.out ]; then
    go tool cover -html=coverage_integration.out -o coverage_integration.html
    echo "✓ Integration coverage report: coverage_integration.html"
    ((GENERATED++))
fi

if [ -f coverage_all.out ]; then
    go tool cover -html=coverage_all.out -o coverage_all.html
    echo "✓ Total coverage report: coverage_all.html"
    ((GENERATED++))
fi

if [ $GENERATED -eq 0 ]; then
    echo "No coverage data found. Run 'make test-coverage' first."
    exit 1
fi

echo ""
echo "Generated $GENERATED report(s)"
