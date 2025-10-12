# Test Coverage Quick Reference

## 🚀 Quick Commands

```bash
# Generate coverage report (most common)
make test-coverage

# View function-level stats
make test-coverage-func

# Advanced analysis
./scripts/coverage.sh all

# Clean coverage files
make test-coverage-clean
```

## 📊 Coverage Targets

| Level | Coverage | Status |
|-------|----------|--------|
| 🔴 Critical | < 50% | Fails CI |
| 🟡 Acceptable | 50-70% | Passes CI |
| 🟢 Good | 70-80% | Recommended |
| ✅ Excellent | > 80% | Target for critical code |

## 🛠 Common Workflows

### 1. Check Current Coverage

```bash
make test-coverage
open coverage.html
```

### 2. Find Low Coverage Areas

```bash
./scripts/coverage.sh uncovered
```

### 3. Before Committing

```bash
make test-coverage-func
# Ensure your changes maintain/improve coverage
```

### 4. Install Pre-commit Hook

```bash
cp scripts/pre-commit-hook.sh .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

## 📁 Generated Files

| File | Description |
|------|-------------|
| `coverage.out` | Raw coverage data |
| `coverage.html` | Interactive HTML report |
| `docs/coverage-badge.svg` | Coverage badge (optional) |

## 🎯 Coverage Script Modes

```bash
./scripts/coverage.sh run        # Run tests only
./scripts/coverage.sh html       # Generate HTML
./scripts/coverage.sh stats      # Show statistics
./scripts/coverage.sh uncovered  # Show low coverage
./scripts/coverage.sh summary    # Package summary
./scripts/coverage.sh all        # Everything (default)
```

## 🔍 Reading Coverage Reports

### HTML Report Colors

- **Green**: Code executed by tests ✅
- **Red**: Code not executed by tests ❌
- **Gray**: Non-executable code (comments, etc.)

### Terminal Output Colors

- **Green** (≥80%): Excellent coverage
- **Yellow** (60-79%): Acceptable coverage
- **Red** (<60%): Needs improvement

## 📝 Writing Tests

### Basic Test Structure

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid", "input", "output", false},
        {"error", "bad", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## 🔄 CI/CD Integration

Coverage is automatically checked on:
- Every push to `main` or `develop`
- Every pull request

View results:
1. GitHub Actions → Workflow run → Summary
2. Download artifacts for detailed reports
3. Codecov dashboard (if configured)

## 💡 Tips

1. **Focus on critical code** - parsers, API handlers, business logic
2. **Test behavior, not implementation** - don't test private details
3. **Use table-driven tests** - easier to add cases
4. **Mock external dependencies** - use interfaces
5. **Test error paths** - not just happy paths

## 📚 Documentation

- [Full Coverage Guide](./testing-coverage.md)
- [Test Templates](./test-template.md)
- [Implementation Summary](./coverage-summary.md)

## 🆘 Troubleshooting

### Coverage file not found
```bash
# Run tests first
make test-coverage
```

### Tests failing
```bash
# Run specific package
go test ./internal/parser/... -v

# Run with short flag (skip slow tests)
go test -short ./...
```

### Low coverage warning
```bash
# Find gaps
./scripts/coverage.sh uncovered

# Add tests for uncovered code
# See test-template.md for examples
```

---

**Quick Help**: Run `make help` to see all available commands
