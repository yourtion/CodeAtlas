# 测试指南

> CodeAtlas 测试策略和最佳实践

## 测试类型

### 单元测试

快速测试，不依赖数据库：

```bash
make test              # 运行所有单元测试
make test-coverage     # 生成覆盖率报告
```

### 集成测试

需要数据库的完整测试：

```bash
make test-integration  # 运行所有集成测试
```

### CLI 测试

命令行工具测试：

```bash
make test-cli          # 测试 CLI 命令
```

## 测试环境设置

### 数据库要求

- PostgreSQL 17+ with pgvector extension
- 通过环境变量配置连接

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
```

### 测试数据库管理

每个测试创建唯一的测试数据库（如 `codeatlas_test_abc123`）并在完成后自动清理。

手动清理：

```bash
make clean-test-dbs
```

## 运行测试

### 快速验证（跳过集成测试）

```bash
go test -v -short ./...
```

### 完整测试套件

```bash
make verify            # 完整验证
make verify-tests      # 测试环境验证
```

### 特定测试

```bash
go test -v ./internal/indexer -run TestIndexValidInput
go test -v ./tests/integration -run TestEndToEndIndexing
```

### 带覆盖率

```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 测试覆盖率目标

- **最低要求**: 90% 覆盖率
- **新代码**: 必须包含测试
- **关键路径**: 100% 覆盖率

### 查看覆盖率

```bash
make test-coverage
make test-coverage-report  # 生成 HTML 报告
```

## 测试工具

### 增强测试运行器

彩色输出和统计信息：

```bash
make test-pretty
```

### CI 友好测试

生成 JSON 报告：

```bash
make test-ci
```

### 完整验证

验证整个测试环境：

```bash
make verify-tests
```

验证步骤：
1. 检查数据库连接
2. 清理旧测试数据库
3. 构建 CLI 二进制
4. 运行单元测试
5. 运行集成测试
6. 运行 CLI 测试
7. 最终清理

## 编写测试

### 单元测试示例

```go
func TestSymbolCreation(t *testing.T) {
    tests := []struct {
        name    string
        input   *schema.Symbol
        wantErr bool
    }{
        {
            name: "valid symbol",
            input: &schema.Symbol{
                Name: "testFunc",
                Kind: schema.SymbolFunction,
            },
            wantErr: false,
        },
        {
            name: "missing name",
            input: &schema.Symbol{
                Kind: schema.SymbolFunction,
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateSymbol(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateSymbol() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 集成测试示例

```go
func TestEndToEndIndexing(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // 设置测试数据库
    db, cleanup := setupTestDB(t)
    defer cleanup()

    // 运行测试
    indexer := NewIndexer(db, config)
    result, err := indexer.Index(ctx, parseOutput)
    
    // 验证结果
    assert.NoError(t, err)
    assert.Equal(t, "success", result.Status)
    assert.Greater(t, result.FilesProcessed, 0)
}
```

## 性能测试

### 大规模索引

```bash
go test -v ./tests/integration -run TestLargeScaleIndexing
```

### 并发测试

```bash
go test -v ./tests/integration -run TestConcurrentIndexing
```

### 性能分析

```bash
bash scripts/profile_parse.sh tests/fixtures/test-repo 4
go tool pprof -http=:8080 profile_results/cpu.prof
```

## CI/CD 集成

### GitHub Actions 示例

```yaml
- name: Start PostgreSQL
  run: docker-compose up -d

- name: Run Tests
  run: make test-integration

- name: Upload Coverage
  uses: codecov/codecov-action@v3
  with:
    files: ./coverage.out
```

## 测试最佳实践

1. **使用表驱动测试** - 更容易添加测试用例
2. **测试边界条件** - 空输入、大输入、无效输入
3. **Mock 外部依赖** - 数据库、HTTP 客户端
4. **使用子测试** - `t.Run()` 组织测试
5. **清理资源** - 使用 `defer` 确保清理
6. **测试错误路径** - 不只测试成功场景
7. **保持测试独立** - 测试之间不应有依赖
8. **使用有意义的名称** - 清晰描述测试内容

## 故障排除

### 数据库连接错误

```bash
# 检查 PostgreSQL 是否运行
pg_isready

# 验证凭据
psql -h localhost -U codeatlas -d postgres
```

### 测试超时

```bash
# 增加超时时间
go test -v -timeout 60s ./tests/integration
```

### 权限错误

```sql
-- 确保用户有权限
ALTER USER codeatlas CREATEDB;
ALTER USER codeatlas WITH SUPERUSER;
```

### 清理失败的测试数据库

```bash
# 手动清理
make clean-test-dbs

# 或直接连接
psql -h localhost -U codeatlas -d postgres
DROP DATABASE codeatlas_test_xxx;
```

## 测试脚本

### 可用脚本

- `scripts/test_runner.sh` - 增强测试运行器
- `scripts/test_ci.sh` - CI 友好测试
- `scripts/verify_test_setup.sh` - 完整验证
- `scripts/cleanup_test_databases.sh` - 清理测试数据库
- `scripts/coverage_report.sh` - 覆盖率报告

### 使用示例

```bash
# 彩色输出测试
bash scripts/test_runner.sh go test ./... -v

# 生成 CI 报告
bash scripts/test_ci.sh go test ./... -v

# 完整验证
bash scripts/verify_test_setup.sh
```

## 调用分析测试

### 概述

调用分析测试验证解析器能够准确识别代码中的调用关系，包括：
- 从调用方找到被调用方（caller → callee）
- 从被调用方找到调用方（callee → caller）
- 跨语言调用（如 Kotlin → Java, Swift → Objective-C）

### 运行调用分析测试

```bash
# 运行所有调用分析测试
go test -v ./tests/integration -run TestCallAnalysis

# 运行特定语言测试
go test -v ./tests/integration -run TestCallAnalysis_Go
go test -v ./tests/integration -run TestCallAnalysis_Java
go test -v ./tests/integration -run TestCallAnalysis_Python

# 运行跨语言测试
go test -v ./tests/integration -run TestCallAnalysis_CrossLanguage

# 运行精度和召回率测试
go test -v ./tests/integration -run TestCallAnalysisMetrics
```

### 测试覆盖的语言

**单语言测试：**
- Go, Java, Python, JavaScript/TypeScript
- C, C++, Objective-C, Swift
- Kotlin

**跨语言测试：**
- Kotlin → Java (JVM 互操作)
- Swift → Objective-C (iOS/macOS 互操作)
- TypeScript → JavaScript (模块导入)
- C++ → C (extern "C")
- Objective-C++ → Objective-C (混合代码)

### 测试指标

**精度（Precision）和召回率（Recall）目标：**
- 单语言：≥90% 精度，≥90% 召回率
- 跨语言：≥75% 精度，≥65% 召回率

**测试内容：**
1. **完整性**：确保找到所有真实的调用关系
2. **精确性**：确保没有误报（如注释中的函数名）
3. **特殊场景**：递归调用、间接调用、接口调用、嵌套调用

### 测试结果

查看详细测试结果：
```bash
cat tests/integration/CALL_ANALYSIS_TEST_RESULTS.md
```

### 编写调用分析测试

```go
func TestCallAnalysis_MyLanguage(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    tsParser, err := parser.NewTreeSitterParser()
    require.NoError(t, err)

    myParser := parser.NewMyLanguageParser(tsParser)

    // 创建测试文件
    testFile := filepath.Join(t.TempDir(), "test.ext")
    content := `// 测试代码`
    err = os.WriteFile(testFile, []byte(content), 0644)
    require.NoError(t, err)

    // 解析文件
    parsedFile, err := myParser.Parse(file)
    require.NoError(t, err)

    // 验证调用关系
    callsFromCaller := []string{}
    for _, dep := range parsedFile.Dependencies {
        if dep.Type == "call" && dep.Source == "caller" {
            callsFromCaller = append(callsFromCaller, dep.Target)
        }
    }

    // 断言
    assert.Contains(t, callsFromCaller, "expectedCallee")
}
```

## 参考资料

- [Go 测试文档](https://golang.org/pkg/testing/)
- [表驱动测试](https://github.com/golang/go/wiki/TableDrivenTests)
- [集成测试 README](../../tests/integration/README.md)
- [调用分析测试结果](../../tests/integration/CALL_ANALYSIS_TEST_RESULTS.md)
