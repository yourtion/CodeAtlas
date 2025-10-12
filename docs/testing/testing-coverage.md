# Test Coverage Guide

This document describes the test coverage functionality in CodeAtlas and how to use it effectively.

## Overview

CodeAtlas includes comprehensive test coverage tools to help maintain code quality and ensure all functionality is properly tested. The coverage system provides:

- **Automated coverage reports** with HTML visualization
- **Package-level statistics** to identify areas needing more tests
- **CI/CD integration** with GitHub Actions
- **Coverage thresholds** to maintain minimum quality standards
- **Detailed analysis** of uncovered code

## Quick Start

### Run Tests with Coverage

```bash
# Generate coverage report and open HTML
make test-coverage

# Show function-level coverage statistics
make test-coverage-func

# Generate HTML report from existing coverage data
make test-coverage-report

# Clean coverage files
make test-coverage-clean
```

### Using the Coverage Script

The `scripts/coverage.sh` script provides advanced coverage analysis:

```bash
# Run all coverage analysis
./scripts/coverage.sh all

# Run tests only
./scripts/coverage.sh run

# Generate HTML report
./scripts/coverage.sh html

# Show coverage statistics
./scripts/coverage.sh stats

# Show files with low coverage
./scripts/coverage.sh uncovered

# Show package-level summary
./scripts/coverage.sh summary
```

## Coverage Reports

### HTML Report

The HTML report (`coverage.html`) provides an interactive view of your code coverage:

- **Green**: Well-covered code (executed by tests)
- **Red**: Uncovered code (not executed by tests)
- **Gray**: Non-executable code (comments, declarations)

Open the report in your browser:

```bash
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Terminal Output

The coverage script provides color-coded terminal output:

- **Green** (≥80%): Excellent coverage
- **Yellow** (60-79%): Acceptable coverage
- **Red** (<60%): Needs improvement

## Coverage Threshold

The project maintains a minimum coverage threshold of **50%**. This threshold is enforced in:

1. **Local development**: The coverage script warns when below threshold
2. **CI/CD pipeline**: GitHub Actions fails if coverage drops below threshold

To modify the threshold, update:

- `scripts/coverage.sh`: Change `COVERAGE_THRESHOLD` variable
- `.github/workflows/test-coverage.yml`: Change `THRESHOLD` variable

## Best Practices

### Writing Tests

1. **Test alongside code**: Write tests as you develop features
2. **Follow naming conventions**: Use `_test.go` suffix for test files
3. **Use table-driven tests**: For testing multiple scenarios
4. **Test edge cases**: Include error conditions and boundary values
5. **Mock external dependencies**: Use interfaces for testability

### Example Test Structure

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "TEST",
            wantErr: false,
        },
        {
            name:    "empty input",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Improving Coverage

1. **Identify gaps**: Use `./scripts/coverage.sh uncovered` to find low-coverage files
2. **Prioritize critical code**: Focus on business logic and complex algorithms
3. **Add integration tests**: Test component interactions in `tests/` directory
4. **Test error paths**: Ensure error handling is covered
5. **Review regularly**: Check coverage reports in pull requests

## CI/CD Integration

### GitHub Actions

The project includes a GitHub Actions workflow (`.github/workflows/test-coverage.yml`) that:

1. Runs on every push and pull request
2. Executes all tests with coverage
3. Generates coverage reports
4. Checks coverage threshold
5. Uploads reports to Codecov
6. Stores artifacts for 30 days

### Viewing CI Coverage

After a workflow run:

1. Go to the **Actions** tab in GitHub
2. Select the workflow run
3. View the **Summary** for coverage statistics
4. Download **coverage-report** artifact for detailed analysis

### Codecov Integration

To enable Codecov integration:

1. Sign up at [codecov.io](https://codecov.io)
2. Add your repository
3. Add `CODECOV_TOKEN` to GitHub Secrets
4. Coverage reports will be automatically uploaded

## Package Organization

Tests are organized in two locations:

### Unit Tests (alongside source)

Located in the same package as the code:

```
internal/parser/
├── python_parser.go
├── python_parser_test.go
├── js_parser.go
└── js_parser_test.go
```

### Integration Tests (separate directory)

Located in the `tests/` directory:

```
tests/
├── api/
│   └── server_test.go
├── cli/
│   └── scanner_test.go
└── models/
    └── database_test.go
```

## Coverage Metrics

### Understanding Coverage Types

- **Statement Coverage**: Percentage of statements executed
- **Branch Coverage**: Percentage of conditional branches taken
- **Function Coverage**: Percentage of functions called

Go's coverage tool measures **statement coverage** by default.

### Target Coverage Levels

- **Critical code** (parsers, API handlers): ≥80%
- **Business logic**: ≥70%
- **Utilities**: ≥60%
- **Overall project**: ≥50%

## Troubleshooting

### Coverage File Not Found

```bash
# Error: coverage.out not found
# Solution: Run tests first
make test-coverage
```

### Low Coverage Warning

```bash
# Warning: Coverage below threshold
# Solution: Add more tests or adjust threshold
./scripts/coverage.sh uncovered  # Find gaps
```

### Tests Failing in CI

```bash
# Check database connection
# Ensure environment variables are set
# Review GitHub Actions logs
```

## Advanced Usage

### Coverage for Specific Packages

```bash
# Test specific package with coverage
go test ./internal/parser/... -coverprofile=parser.out
go tool cover -html=parser.out -o parser.html
```

### Exclude Files from Coverage

Add build tags to exclude files:

```go
//go:build !test
// +build !test

package mypackage
```

### Benchmark with Coverage

```bash
# Run benchmarks with coverage
go test -bench=. -coverprofile=bench.out ./...
```

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Go Coverage Tool](https://blog.golang.org/cover)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Codecov Documentation](https://docs.codecov.io/)

## Contributing

When contributing to CodeAtlas:

1. Write tests for new features
2. Maintain or improve coverage
3. Run `make test-coverage` before submitting PR
4. Review coverage report for your changes
5. Address any coverage warnings

---

For questions or issues, please open a GitHub issue or contact the maintainers.
