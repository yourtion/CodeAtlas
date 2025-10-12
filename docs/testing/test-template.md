# Test Template Guide

This document provides templates and examples for writing tests in CodeAtlas.

## Basic Test Template

```go
package mypackage

import (
    "testing"
)

func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:    "valid case",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        {
            name:    "error case",
            input:   invalidInput,
            want:    zeroValue,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("FunctionName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Parser Test Template

```go
package parser

import (
    "os"
    "path/filepath"
    "testing"
)

func TestParser_Parse(t *testing.T) {
    parser := NewParser()

    tests := []struct {
        name        string
        code        string
        wantSymbols int
        wantError   bool
    }{
        {
            name: "simple function",
            code: `func hello() {
    fmt.Println("Hello")
}`,
            wantSymbols: 1,
            wantError:   false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create temp file
            tmpDir := t.TempDir()
            tmpFile := filepath.Join(tmpDir, "test.go")
            if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
                t.Fatalf("Failed to write temp file: %v", err)
            }

            file := ScannedFile{
                Path:     "test.go",
                AbsPath:  tmpFile,
                Language: "go",
            }

            result, err := parser.Parse(file)
            
            if (err != nil) != tt.wantError {
                t.Errorf("Parse() error = %v, wantError %v", err, tt.wantError)
                return
            }

            if len(result.Symbols) != tt.wantSymbols {
                t.Errorf("Got %d symbols, want %d", len(result.Symbols), tt.wantSymbols)
            }
        })
    }
}
```

## API Handler Test Template

```go
package api

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestHandler(t *testing.T) {
    gin.SetMode(gin.TestMode)

    tests := []struct {
        name           string
        method         string
        path           string
        body           interface{}
        wantStatusCode int
        wantBody       map[string]interface{}
    }{
        {
            name:           "successful request",
            method:         "POST",
            path:           "/api/v1/resource",
            body:           map[string]string{"key": "value"},
            wantStatusCode: http.StatusOK,
            wantBody:       map[string]interface{}{"status": "success"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup router
            router := gin.New()
            router.POST("/api/v1/resource", Handler)

            // Create request
            bodyBytes, _ := json.Marshal(tt.body)
            req := httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(bodyBytes))
            req.Header.Set("Content-Type", "application/json")
            
            // Record response
            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)

            // Check status code
            if w.Code != tt.wantStatusCode {
                t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatusCode)
            }

            // Check response body
            var got map[string]interface{}
            if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
                t.Fatalf("Failed to unmarshal response: %v", err)
            }

            if !reflect.DeepEqual(got, tt.wantBody) {
                t.Errorf("Response body = %v, want %v", got, tt.wantBody)
            }
        })
    }
}
```

## Database Test Template

```go
package models

import (
    "database/sql"
    "testing"

    _ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("postgres", "postgres://test:test@localhost/test?sslmode=disable")
    if err != nil {
        t.Fatalf("Failed to connect to test database: %v", err)
    }

    // Create test schema
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS test_table (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL
    )`)
    if err != nil {
        t.Fatalf("Failed to create test table: %v", err)
    }

    return db
}

func teardownTestDB(t *testing.T, db *sql.DB) {
    _, err := db.Exec("DROP TABLE IF EXISTS test_table")
    if err != nil {
        t.Errorf("Failed to drop test table: %v", err)
    }
    db.Close()
}

func TestDatabaseOperation(t *testing.T) {
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {
            name:    "insert valid data",
            input:   "test",
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := InsertData(db, tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("InsertData() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Benchmark Test Template

```go
package mypackage

import (
    "testing"
)

func BenchmarkFunction(b *testing.B) {
    // Setup
    input := prepareInput()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        Function(input)
    }
}

func BenchmarkFunctionParallel(b *testing.B) {
    input := prepareInput()

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            Function(input)
        }
    })
}
```

## Mock Interface Template

```go
package mypackage

import (
    "testing"
)

// MockInterface implements Interface for testing
type MockInterface struct {
    MethodFunc func(input string) (string, error)
}

func (m *MockInterface) Method(input string) (string, error) {
    if m.MethodFunc != nil {
        return m.MethodFunc(input)
    }
    return "", nil
}

func TestWithMock(t *testing.T) {
    mock := &MockInterface{
        MethodFunc: func(input string) (string, error) {
            return "mocked", nil
        },
    }

    result, err := FunctionUsingInterface(mock, "test")
    
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
    
    if result != "expected" {
        t.Errorf("Got %v, want expected", result)
    }
}
```

## Test Helpers

```go
package testutil

import (
    "os"
    "path/filepath"
    "testing"
)

// CreateTempFile creates a temporary file with content
func CreateTempFile(t *testing.T, content string) string {
    tmpDir := t.TempDir()
    tmpFile := filepath.Join(tmpDir, "test.txt")
    
    if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    
    return tmpFile
}

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, got, want interface{}) {
    t.Helper()
    
    if !reflect.DeepEqual(got, want) {
        t.Errorf("Got %v, want %v", got, want)
    }
}

// AssertError checks if error matches expectation
func AssertError(t *testing.T, err error, wantErr bool) {
    t.Helper()
    
    if (err != nil) != wantErr {
        t.Errorf("Error = %v, wantErr %v", err, wantErr)
    }
}
```

## Best Practices

1. **Use table-driven tests** for multiple test cases
2. **Use t.Helper()** in helper functions for better error reporting
3. **Use t.TempDir()** for temporary files (auto-cleanup)
4. **Use t.Parallel()** for independent tests
5. **Use subtests** with t.Run() for better organization
6. **Mock external dependencies** using interfaces
7. **Test error cases** as thoroughly as success cases
8. **Use meaningful test names** that describe what's being tested
9. **Keep tests focused** - one concept per test
10. **Clean up resources** with defer or t.Cleanup()

## Running Tests

```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/parser/...

# Run specific test
go test -run TestFunctionName

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Run short tests only
go test -short ./...
```

## Coverage Tips

- Aim for **80%+ coverage** on critical code
- Focus on **business logic** and **complex algorithms**
- Don't obsess over **100% coverage** - test what matters
- Use coverage reports to **find gaps**, not as a goal
- **Test behavior**, not implementation details
