# CodeAtlas Testing Guide

## Overview

CodeAtlas uses a comprehensive testing strategy that separates **unit tests** from **integration tests** to ensure fast feedback during development while maintaining thorough test coverage.

## Test Types

### Unit Tests
- **No external dependencies** (no database, no API server, no external services)
- **Fast execution** (typically < 1 second per package)
- **Run by default** in CI/CD pipelines
- **Coverage target**: 90%+

### Integration Tests
- **Require external dependencies** (PostgreSQL database, vLLM service, etc.)
- **Slower execution** (may take several seconds)
- **Run separately** from unit tests
- **Coverage target**: 85%+

## Running Tests

### Quick Start

```bash
# Run unit tests only (fast, no dependencies)
make test-unit

# Run all tests (requires database)
make docker-up
make test-all

# Generate coverage report
make test-coverage-all
```

### Unit Tests

Unit tests run without any external dependencies:

```bash
# Run all unit tests
make test-unit
# or
go test -short ./...

# Run unit tests for specific package
go test -short ./internal/parser/... -v

# Run with coverage
make test-coverage-unit
```

### Integration Tests

Integration tests require a running PostgreSQL database:

```bash
# Start database
make docker-up

# Run integration tests
make test-integration

# Run specific integration test
go test ./pkg/models/... -v -run TestSymbolRepository_Create
```

### CLI Integration Tests

CLI integration tests require the CLI binary to be built:

```bash
# Build CLI and run integration tests
make test-cli-integration

# Or manually
make build-cli
go test -tags=parse_tests ./tests/cli/... -v
```

### All Tests

Run both unit and integration tests:

```bash
# Ensure database is running
make docker-up

# Run all tests
make test-all

# Generate combined coverage report
make test-coverage-all
```

## Test Organization

### Directory Structure

```
CodeAtlas/
├── cmd/
│   ├── api/
│   │   └── *_test.go          # API server unit tests
│   └── cli/
│       └── *_test.go          # CLI unit tests
├── internal/
│   ├── parser/
│   │   └── *_test.go          # Parser unit tests (no DB)
│   ├── indexer/
│   │   ├── *_test.go          # Indexer tests (DB required)
│   │   └── *_integration_test.go  # Integration tests (tagged)
│   └── ...
├── pkg/
│   └── models/
│       └── *_test.go          # Model tests (DB required)
└── tests/
    ├── api/
    │   └── *_test.go          # API integration tests
    ├── cli/
    │   └── *_test.go          # CLI integration tests
    └── models/
        └── *_test.go          # Model integration tests
```

### Test Naming Conventions

- `*_test.go` - Standard test files
- `*_integration_test.go` - Integration tests (with build tags)
- `*_example_test.go` - Example tests (documentation)
- `*_bench_test.go` - Benchmark tests

### Build Tags

Integration tests that require specific services use build tags:

```go
//go:build integration
// +build integration

package indexer

import "testing"

func TestIntegration_OpenAIEmbedder(t *testing.T) {
    // Test requires vLLM service
}
```

Run with: `go test -tags=integration ./...`

### Short Mode

Tests that require external dependencies check `testing.Short()`:

```go
func TestDatabaseOperation(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    db, err := models.NewDB()
    // ... test code
}
```

Run unit tests only: `go test -short ./...`

## Writing Tests

### Unit Test Example

```go
package parser

import "testing"

func TestGoParser_ExtractFunctions(t *testing.T) {
    // No external dependencies
    parser := NewGoParser()
    
    code := `package main
    func Hello() string {
        return "hello"
    }`
    
    result, err := parser.Parse(code)
    if err != nil {
        t.Fatalf("Parse failed: %v", err)
    }
    
    if len(result.Functions) != 1 {
        t.Errorf("Expected 1 function, got %d", len(result.Functions))
    }
}
```

### Integration Test Example

```go
package models

import (
    "context"
    "testing"
)

func TestSymbolRepository_Create(t *testing.T) {
    // Skip in short mode
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Connect to database
    db, err := NewDB()
    if err != nil {
        t.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()
    
    // Test database operations
    repo := NewSymbolRepository(db)
    symbol := &Symbol{Name: "TestFunc"}
    
    err = repo.Create(context.Background(), symbol)
    if err != nil {
        t.Fatalf("Failed to create symbol: %v", err)
    }
}
```

### Test Helpers

Use setup/teardown helpers for integration tests:

```go
func setupTestDB(t *testing.T) (*models.DB, func()) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    db, err := models.NewDB()
    if err != nil {
        t.Skipf("Database not available: %v", err)
    }
    
    cleanup := func() {
        db.ExecContext(context.Background(), "TRUNCATE TABLE repositories CASCADE")
        db.Close()
    }
    
    return db, cleanup
}

func TestWithDatabase(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Test code here
}
```

## Coverage Reports

### Generate Coverage Reports

```bash
# Unit test coverage
make test-coverage-unit
# Opens: coverage_unit.html

# Integration test coverage
make test-coverage-integration
# Opens: coverage_integration.html

# Combined coverage
make test-coverage-all
# Opens: coverage_all.html
```

### View Coverage Statistics

```bash
# Show coverage by function
make test-coverage-func

# Show coverage for specific package
go test -short ./internal/parser/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Coverage Goals

| Package | Unit Coverage | Integration Coverage | Combined |
|---------|--------------|---------------------|----------|
| internal/utils | 100% ✅ | N/A | 100% |
| internal/schema | 95.8% ✅ | N/A | 95.8% |
| internal/output | 90.5% ✅ | N/A | 90.5% |
| internal/parser | 89.9% ✅ | N/A | 89.9% |
| internal/indexer | 39.2% 🟡 | 81.6% ✅ | 85%+ |
| pkg/models | 1.2% 🔴 | 85%+ ✅ | 85%+ |
| cmd/cli | 47.9% 🟡 | N/A | 70%+ |
| cmd/api | 0% 🔴 | N/A | 70%+ |

**Overall Target**: 90%+ combined coverage

## Test Database Setup

### Using Docker Compose

```bash
# Start test database
make docker-up

# Check database status
docker-compose ps

# View database logs
docker-compose logs db

# Stop database
make docker-down
```

### Manual Database Setup

```bash
# Create test database
createdb codeatlas_test

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
export DB_NAME=codeatlas_test

# Run tests
go test ./pkg/models/... -v
```

### Database Cleanup

Integration tests should clean up after themselves:

```go
func TestWithCleanup(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()  // Ensures cleanup even if test fails
    
    // Test code
}
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run unit tests
        run: make test-unit
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage_unit.out

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17
        env:
          POSTGRES_PASSWORD: codeatlas
          POSTGRES_USER: codeatlas
          POSTGRES_DB: codeatlas
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run integration tests
        run: make test-integration
        env:
          DB_HOST: localhost
          DB_PORT: 5432
          DB_USER: codeatlas
          DB_PASSWORD: codeatlas
          DB_NAME: codeatlas
```

## Troubleshooting

### Tests Fail with "database not available"

```bash
# Check if database is running
docker-compose ps

# Start database
make docker-up

# Check database logs
docker-compose logs db

# Verify connection
psql -h localhost -U codeatlas -d codeatlas
```

### Tests Timeout

```bash
# Increase test timeout
go test ./... -timeout 30s

# Run tests with verbose output
go test ./... -v -timeout 30s
```

### Coverage Report Not Generated

```bash
# Clean old coverage files
make test-coverage-clean

# Regenerate coverage
make test-coverage-all

# Check if coverage file exists
ls -la coverage*.out
```

### Integration Tests Run During Unit Tests

Check that tests have proper guards:

```go
// Add this to integration tests
if testing.Short() {
    t.Skip("Skipping integration test in short mode")
}
```

## Best Practices

### 1. Test Isolation
- Each test should be independent
- Use setup/teardown functions
- Clean up test data after each test

### 2. Test Naming
- Use descriptive test names: `TestSymbolRepository_Create`
- Use subtests for variations: `t.Run("with_valid_input", func(t *testing.T) {...})`

### 3. Error Messages
- Provide clear error messages
- Include expected vs actual values
- Use `t.Errorf()` for non-fatal errors, `t.Fatalf()` for fatal errors

### 4. Test Data
- Use fixtures for complex test data
- Generate unique IDs for each test
- Avoid hardcoded values that may conflict

### 5. Mocking
- Mock external dependencies in unit tests
- Use interfaces for testability
- Consider using `httptest` for HTTP handlers

### 6. Performance
- Keep unit tests fast (< 1s per package)
- Use `testing.Short()` for slow tests
- Run benchmarks separately: `go test -bench=.`

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Test Fixtures](https://github.com/go-testfixtures/testfixtures)
- [Testify](https://github.com/stretchr/testify) - Testing toolkit

## Summary

- **Unit tests**: Fast, no dependencies, run with `make test-unit`
- **Integration tests**: Require database, run with `make test-integration`
- **Coverage target**: 90%+ combined
- **Always** add `testing.Short()` checks to integration tests
- **Always** clean up test data in integration tests
