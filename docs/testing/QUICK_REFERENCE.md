# Testing Quick Reference

## Common Commands

### Fast Development Cycle
```bash
# Run unit tests (fast, no dependencies)
make test-unit

# Test specific package
go test -short ./internal/parser/... -v

# Watch mode (requires entr or similar)
find . -name "*.go" | entr -c make test-unit
```

### Before Committing
```bash
# Start database
make docker-up

# Run all tests
make test-all

# Generate coverage
make test-coverage-all
```

### Coverage Reports
```bash
# Unit test coverage
make test-coverage-unit
open coverage_unit.html

# Integration test coverage  
make test-coverage-integration
open coverage_integration.html

# Combined coverage
make test-coverage-all
open coverage_all.html

# Show statistics
make test-coverage-func
```

## Test Types

| Type | Command | Dependencies | Speed |
|------|---------|--------------|-------|
| Unit | `make test-unit` | None | ⚡ Fast (~5s) |
| Integration | `make test-integration` | Database | 🐢 Slow (~15s) |
| CLI | `make test-cli-integration` | Binary | 🐢 Slow (~10s) |
| All | `make test-all` | Database | 🐢 Slow (~20s) |

## Package-Specific Tests

```bash
# Parser tests (unit)
go test -short ./internal/parser/... -v

# Indexer tests (integration, needs DB)
go test ./internal/indexer/... -v

# Model tests (integration, needs DB)
go test ./pkg/models/... -v

# CLI tests (unit)
go test -short ./cmd/cli/... -v

# API tests (unit)
go test -short ./internal/api/... -v
```

## Writing Tests

### Unit Test Template
```go
package mypackage

import "testing"

func TestMyFunction(t *testing.T) {
    // Arrange
    input := "test"
    expected := "result"
    
    // Act
    result := MyFunction(input)
    
    // Assert
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

### Integration Test Template
```go
package mypackage

import (
    "context"
    "testing"
)

func TestDatabaseOperation(t *testing.T) {
    // Skip in short mode
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Setup
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Test
    err := db.DoSomething(context.Background())
    if err != nil {
        t.Fatalf("Operation failed: %v", err)
    }
}
```

### Test Helper Template
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
        db.ExecContext(context.Background(), 
            "TRUNCATE TABLE repositories CASCADE")
        db.Close()
    }
    
    return db, cleanup
}
```

## Troubleshooting

### Database Connection Failed
```bash
# Check if database is running
docker-compose ps

# Start database
make docker-up

# Check logs
docker-compose logs db

# Verify connection
psql -h localhost -U codeatlas -d codeatlas
```

### Tests Timeout
```bash
# Increase timeout
go test ./... -timeout 30s -v

# Run with race detector
go test ./... -race -timeout 60s
```

### Coverage Not Generated
```bash
# Clean old files
make test-coverage-clean

# Regenerate
make test-coverage-all

# Check files
ls -la coverage*.out
```

### Integration Tests Run in Unit Mode
```bash
# Verify test has guard
grep -n "testing.Short()" path/to/test.go

# Add guard if missing
if testing.Short() {
    t.Skip("Skipping integration test in short mode")
}
```

## Coverage Goals

| Package | Current | Target | Priority |
|---------|---------|--------|----------|
| internal/utils | 100% ✅ | 100% | - |
| internal/schema | 95.8% ✅ | 95%+ | - |
| internal/output | 90.5% ✅ | 90%+ | - |
| internal/parser | 89.9% ✅ | 90%+ | Low |
| internal/indexer | 39.2% 🟡 | 85%+ | Medium |
| pkg/models | 1.2% 🔴 | 85%+ | High |
| cmd/cli | 47.9% 🟡 | 70%+ | High |
| cmd/api | 0% 🔴 | 70%+ | High |
| internal/api | 0% 🔴 | 70%+ | High |

## CI/CD Integration

### GitHub Actions
```yaml
# .github/workflows/test.yml
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
      - run: make test-unit

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17
        env:
          POSTGRES_PASSWORD: codeatlas
          POSTGRES_USER: codeatlas
          POSTGRES_DB: codeatlas
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: make test-integration
```

## Best Practices

### ✅ DO
- Add `testing.Short()` to integration tests
- Use descriptive test names
- Clean up test data
- Use table-driven tests
- Test error cases
- Keep unit tests fast

### ❌ DON'T
- Mix unit and integration tests
- Leave test data in database
- Use hardcoded IDs
- Skip error checking
- Test implementation details
- Ignore test failures

## Resources

- [Full Testing Guide](./TESTING_GUIDE.md)
- [Test Analysis Report](../../TEST_ANALYSIS.md)
- [Go Testing Docs](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
