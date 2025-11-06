# 测试完整指南

> CodeAtlas 的测试策略、工具和最佳实践

## 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [测试类型](#测试类型)
- [运行测试](#运行测试)
- [编写测试](#编写测试)
- [测试覆盖率](#测试覆盖率)
- [CI/CD 集成](#cicd-集成)
- [故障排除](#故障排除)
- [最佳实践](#最佳实践)

## 概述

CodeAtlas 使用全面的测试策略，将**单元测试**和**集成测试**分离，确保快速反馈的同时保持全面的测试覆盖率。

### 测试目标

- **单元测试覆盖率**: 90%+
- **集成测试覆盖率**: 85%+
- **整体覆盖率**: 90%+
- **测试执行速度**: 单元测试 < 10秒，全部测试 < 30秒

## 快速开始

### 日常开发

```bash
# 快速单元测试（无依赖，最快）
make test-unit

# 测试特定包
go test -short ./internal/parser/... -v

# 生成覆盖率报告
make test-coverage-unit
```

### 提交前检查

```bash
# 启动数据库
make docker-up

# 运行所有测试
make test-all

# 生成完整覆盖率报告
make test-coverage-all
open coverage_all.html
```

### 增强输出测试

```bash
# 彩色输出 + 统计信息
make test-unit-pretty          # 单元测试
make test-all-pretty           # 所有测试
make test-integration-pretty   # 集成测试

# CI 友好的 JSON 报告
make test-ci                   # 单元测试
make test-ci-all               # 所有测试
```

## 测试类型

### 单元测试

**特点**：
- ✅ 无外部依赖（无数据库、无 API、无外部服务）
- ✅ 快速执行（通常 < 1 秒/包）
- ✅ CI/CD 默认运行
- ✅ 覆盖率目标：90%+

**运行方式**：
```bash
make test-unit
# 或
go test -short ./...
```

### 集成测试

**特点**：
- 🔧 需要外部依赖（PostgreSQL、vLLM 等）
- 🐢 较慢执行（可能需要几秒钟）
- 🔧 单独运行
- 🔧 覆盖率目标：85%+

**运行方式**：
```bash
# 启动数据库
make docker-up

# 运行集成测试
make test-integration
```

### CLI 集成测试

**特点**：
- 🔧 需要构建 CLI 二进制文件
- 🔧 测试完整的 CLI 工作流
- 🔧 使用 build tags

**运行方式**：
```bash
make test-cli-integration
# 或
make build-cli
go test -tags=parse_tests ./tests/cli/... -v
```

## 运行测试

### 基本命令

```bash
# 单元测试（推荐日常使用）
make test-unit

# 集成测试（需要数据库）
make test-integration

# 所有测试
make test-all

# 特定模块
make test-api          # API 测试
make test-cli          # CLI 测试
make test-models       # 数据库模型测试
```

### 覆盖率报告

```bash
# 单元测试覆盖率
make test-coverage-unit
open coverage_unit.html

# 集成测试覆盖率
make test-coverage-integration
open coverage_integration.html

# 完整覆盖率
make test-coverage-all
open coverage_all.html

# 函数级覆盖率统计
make test-coverage-func
```

### 高级选项

```bash
# 详细输出
go test -v ./...

# 运行特定测试
go test ./internal/parser/... -run TestGoParser

# 带竞态检测
go test -race ./...

# 增加超时
go test -timeout 30s ./...

# 并行运行
go test -parallel 4 ./...
```

## 编写测试

### 测试组织

```
CodeAtlas/
├── cmd/
│   ├── api/
│   │   └── *_test.go          # API 单元测试
│   └── cli/
│       └── *_test.go          # CLI 单元测试
├── internal/
│   ├── parser/
│   │   └── *_test.go          # Parser 单元测试
│   ├── indexer/
│   │   ├── *_test.go          # Indexer 测试
│   │   └── *_integration_test.go  # 集成测试（带 tags）
│   └── ...
├── pkg/
│   └── models/
│       └── *_test.go          # Model 测试（需要数据库）
└── tests/
    ├── api/
    │   └── *_test.go          # API 集成测试
    ├── cli/
    │   └── *_test.go          # CLI 集成测试
    └── models/
        └── *_test.go          # Model 集成测试
```

### 单元测试示例

```go
package parser

import "testing"

func TestGoParser_ExtractFunctions(t *testing.T) {
    // 无外部依赖
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

### 集成测试示例

```go
package models

import (
    "context"
    "testing"
)

func TestSymbolRepository_Create(t *testing.T) {
    // 在 short 模式下跳过
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // 连接数据库
    db, err := NewDB()
    if err != nil {
        t.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()
    
    // 测试数据库操作
    repo := NewSymbolRepository(db)
    symbol := &Symbol{Name: "TestFunc"}
    
    err = repo.Create(context.Background(), symbol)
    if err != nil {
        t.Fatalf("Failed to create symbol: %v", err)
    }
}
```

### 表驱动测试

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
        {
            name:    "special characters",
            input:   "test@123",
            want:    "TEST@123",
            wantErr: false,
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

### 测试辅助函数

```go
// 数据库测试辅助函数
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

// 使用示例
func TestWithDatabase(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    // 测试代码
}
```

### Build Tags

对于需要特定服务的集成测试：

```go
//go:build integration
// +build integration

package indexer

import "testing"

func TestIntegration_OpenAIEmbedder(t *testing.T) {
    // 需要 vLLM 服务
}
```

运行：`go test -tags=integration ./...`

## 测试覆盖率

### 生成覆盖率报告

```bash
# 单元测试覆盖率
make test-coverage-unit

# 集成测试覆盖率
make test-coverage-integration

# 完整覆盖率
make test-coverage-all

# 函数级统计
make test-coverage-func
```

### HTML 报告

覆盖率报告提供交互式代码视图：

- **绿色**：已覆盖的代码（被测试执行）
- **红色**：未覆盖的代码（未被测试执行）
- **灰色**：不可执行代码（注释、声明）

```bash
# 打开 HTML 报告
open coverage_all.html  # macOS
xdg-open coverage_all.html  # Linux
```

### 使用覆盖率脚本

```bash
# 运行所有覆盖率分析
./scripts/coverage.sh all

# 只运行测试
./scripts/coverage.sh run

# 生成 HTML 报告
./scripts/coverage.sh html

# 显示统计信息
./scripts/coverage.sh stats

# 显示低覆盖率文件
./scripts/coverage.sh uncovered

# 显示包级摘要
./scripts/coverage.sh summary
```

### 覆盖率目标

| 包 | 单元覆盖率 | 集成覆盖率 | 综合覆盖率 |
|---------|--------------|---------------------|----------|
| internal/utils | 100% ✅ | N/A | 100% |
| internal/schema | 95.8% ✅ | N/A | 95.8% |
| internal/output | 90.5% ✅ | N/A | 90.5% |
| internal/parser | 89.9% ✅ | N/A | 89.9% |
| internal/indexer | 39.2% 🟡 | 81.6% ✅ | 85%+ |
| pkg/models | 1.2% 🔴 | 85%+ ✅ | 85%+ |
| cmd/cli | 47.9% 🟡 | N/A | 70%+ |
| cmd/api | 0% 🔴 | N/A | 70%+ |

**整体目标**: 90%+ 综合覆盖率

### 覆盖率阈值

项目维护最低覆盖率阈值 **50%**，在以下位置强制执行：

1. **本地开发**：覆盖率脚本在低于阈值时警告
2. **CI/CD 流水线**：GitHub Actions 在覆盖率下降时失败

修改阈值：
- `scripts/coverage.sh`：修改 `COVERAGE_THRESHOLD` 变量
- `.github/workflows/test-coverage.yml`：修改 `THRESHOLD` 变量

## 测试数据库设置

### 使用 Docker Compose

```bash
# 启动测试数据库
make docker-up

# 检查数据库状态
docker-compose ps

# 查看数据库日志
docker-compose logs db

# 停止数据库
make docker-down
```

### 手动数据库设置

```bash
# 创建测试数据库
createdb codeatlas_test

# 设置环境变量
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
export DB_NAME=codeatlas_test

# 运行测试
go test ./pkg/models/... -v
```

### 数据库清理

集成测试应该清理自己的数据：

```go
func TestWithCleanup(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()  // 确保即使测试失败也会清理
    
    // 测试代码
}
```

## CI/CD 集成

### GitHub Actions 配置

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
          go-version: '1.25'
      
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
          go-version: '1.25'
      
      - name: Run integration tests
        run: make test-integration
        env:
          DB_HOST: localhost
          DB_PORT: 5432
          DB_USER: codeatlas
          DB_PASSWORD: codeatlas
          DB_NAME: codeatlas
```

### 查看 CI 覆盖率

工作流运行后：

1. 进入 GitHub 的 **Actions** 标签
2. 选择工作流运行
3. 查看 **Summary** 获取覆盖率统计
4. 下载 **coverage-report** 工件进行详细分析

### Codecov 集成

启用 Codecov 集成：

1. 在 [codecov.io](https://codecov.io) 注册
2. 添加你的仓库
3. 将 `CODECOV_TOKEN` 添加到 GitHub Secrets
4. 覆盖率报告将自动上传

## 故障排除

### 测试失败："database not available"

```bash
# 检查数据库是否运行
docker-compose ps

# 启动数据库
make docker-up

# 检查数据库日志
docker-compose logs db

# 验证连接
psql -h localhost -U codeatlas -d codeatlas
```

### 测试超时

```bash
# 增加测试超时
go test ./... -timeout 30s

# 带详细输出运行测试
go test ./... -v -timeout 30s
```

### 覆盖率报告未生成

```bash
# 清理旧覆盖率文件
make test-coverage-clean

# 重新生成覆盖率
make test-coverage-all

# 检查覆盖率文件是否存在
ls -la coverage*.out
```

### 集成测试在单元测试时运行

检查测试是否有适当的保护：

```go
// 添加到集成测试
if testing.Short() {
    t.Skip("Skipping integration test in short mode")
}
```

### 端口冲突

```bash
# 检查端口占用
lsof -i :5432

# 停止占用端口的进程
kill -9 <PID>

# 或使用不同端口
export DB_PORT=5433
```

## 最佳实践

### 1. 测试隔离
- ✅ 每个测试应该独立
- ✅ 使用 setup/teardown 函数
- ✅ 测试后清理测试数据

### 2. 测试命名
- ✅ 使用描述性测试名称：`TestSymbolRepository_Create`
- ✅ 使用子测试进行变体：`t.Run("with_valid_input", func(t *testing.T) {...})`

### 3. 错误消息
- ✅ 提供清晰的错误消息
- ✅ 包含期望值 vs 实际值
- ✅ 使用 `t.Errorf()` 处理非致命错误，`t.Fatalf()` 处理致命错误

### 4. 测试数据
- ✅ 对复杂测试数据使用 fixtures
- ✅ 为每个测试生成唯一 ID
- ✅ 避免可能冲突的硬编码值

### 5. Mock
- ✅ 在单元测试中 mock 外部依赖
- ✅ 使用接口提高可测试性
- ✅ 考虑使用 `httptest` 测试 HTTP 处理器

### 6. 性能
- ✅ 保持单元测试快速（< 1秒/包）
- ✅ 对慢测试使用 `testing.Short()`
- ✅ 单独运行基准测试：`go test -bench=.`

### 7. 覆盖率
- ✅ 在开发功能时编写测试
- ✅ 测试边界情况和错误路径
- ✅ 在 PR 中审查覆盖率报告
- ✅ 优先测试关键代码路径

### 8. 文档
- ✅ 为复杂测试添加注释
- ✅ 使用示例测试作为文档
- ✅ 保持测试代码清晰易读

## 快速参考

### 常用命令

```bash
# 快速开发循环
make test-unit                    # 单元测试
go test -short ./internal/parser/... -v  # 特定包

# 提交前
make docker-up                    # 启动数据库
make test-all                     # 所有测试
make test-coverage-all            # 覆盖率

# 覆盖率报告
make test-coverage-unit           # 单元测试覆盖率
make test-coverage-integration    # 集成测试覆盖率
make test-coverage-all            # 完整覆盖率
make test-coverage-func           # 函数级统计
```

### 测试类型对比

| 类型 | 命令 | 依赖 | 速度 |
|------|---------|--------------|-------|
| 单元 | `make test-unit` | 无 | ⚡ 快 (~5s) |
| 集成 | `make test-integration` | 数据库 | 🐢 慢 (~15s) |
| CLI | `make test-cli-integration` | 二进制 | 🐢 慢 (~10s) |
| 全部 | `make test-all` | 数据库 | 🐢 慢 (~20s) |

## 相关资源

### 内部文档
- [快速参考](../testing/QUICK_REFERENCE.md)
- [测试模板](../testing/test-template.md)
- [贡献指南](../../CONTRIBUTING.md)

### 外部资源
- [Go Testing 文档](https://golang.org/pkg/testing/)
- [表驱动测试](https://github.com/golang/go/wiki/TableDrivenTests)
- [Test Fixtures](https://github.com/go-testfixtures/testfixtures)
- [Testify](https://github.com/stretchr/testify) - 测试工具包
- [Codecov 文档](https://docs.codecov.io/)

## 总结

- **单元测试**：快速，无依赖，使用 `make test-unit`
- **集成测试**：需要数据库，使用 `make test-integration`
- **覆盖率目标**：90%+ 综合覆盖率
- **始终**在集成测试中添加 `testing.Short()` 检查
- **始终**在集成测试中清理测试数据
- **提交前**运行 `make test-all` 和 `make test-coverage-all`

---

**最后更新**: 2025-11-06  
**维护者**: CodeAtlas Team
