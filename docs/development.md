# 开发指南

> CodeAtlas 开发环境设置和最佳实践

## 快速开始

### 前置要求

- Go 1.25+
- Docker 和 Docker Compose
- Node.js 20+ 和 pnpm
- Git
- 4GB+ 内存

### 开发环境设置

#### 方式 1: DevContainer（推荐）

最简单的方式，开箱即用。

**VS Code:**
1. 安装 [Dev Containers 扩展](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. 打开项目
3. 点击 "Reopen in Container"
4. 等待构建完成

**特性**：
- ✅ Go 1.25 + 所有工具（gopls, delve, golangci-lint）
- ✅ Node.js 20 + pnpm
- ✅ PostgreSQL 17（带 pgvector 和 AGE）
- ✅ 预置测试数据
- ✅ VS Code 扩展自动安装

**命令行启动**：
```bash
make devcontainer-up
```

#### 方式 2: 本地开发

```bash
# 1. 克隆仓库
git clone https://github.com/yourtionguo/CodeAtlas.git
cd CodeAtlas

# 2. 安装依赖
make install

# 3. 启动数据库
make docker-up

# 4. 初始化数据库
make db-init

# 5. 构建项目
make build

# 6. 运行测试
make test-unit
```

### 项目结构

```
CodeAtlas/
├── cmd/                    # 应用入口
│   ├── api/               # API 服务器
│   └── cli/               # CLI 工具
├── internal/              # 私有代码
│   ├── api/               # API 实现
│   ├── parser/            # 解析器
│   ├── graph/             # 图服务
│   ├── retrieval/         # 检索服务
│   └── qa/                # QA 引擎
├── pkg/                   # 公共库
│   ├── models/            # 数据模型
│   └── utils/             # 工具函数
├── web/                   # 前端
│   ├── src/               # Svelte 源码
│   └── public/            # 静态资源
├── tests/                 # 测试
│   ├── fixtures/          # 测试数据
│   └── integration/       # 集成测试
├── docs/                  # 文档
├── scripts/               # 脚本
└── deployments/           # 部署配置
```

## 开发工作流

### 日常开发

```bash
# 1. 创建功能分支
git checkout -b feature/my-feature

# 2. 开发和测试
make test-unit              # 快速单元测试
make test-all               # 完整测试

# 3. 代码检查
make lint                   # 代码检查
make fmt                    # 代码格式化

# 4. 提交代码
git add .
git commit -m "feat: add new feature"
git push origin feature/my-feature
```

### 运行服务

```bash
# 后端 API
make run-api                # http://localhost:8080

# 前端（另一个终端）
cd web
pnpm install
pnpm dev                    # http://localhost:3000

# CLI 工具
make run-cli -- parse --path .
```

### 调试

#### VS Code 调试配置

已包含在 `.vscode/launch.json`：

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug API",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/api",
      "env": {
        "DB_HOST": "localhost",
        "DB_PORT": "5432"
      }
    },
    {
      "name": "Debug CLI",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/cli",
      "args": ["parse", "--path", "."]
    }
  ]
}
```

#### 命令行调试

```bash
# 使用 delve
dlv debug ./cmd/api -- --port 8080

# 设置断点
(dlv) break main.main
(dlv) continue
```

## 测试

### 测试策略

CodeAtlas 使用分层测试策略：
- **单元测试**：无外部依赖，快速（< 10 秒）
- **集成测试**：需要数据库，完整（< 30 秒）

### 运行测试

```bash
# 单元测试（最快，日常开发）
make test-unit

# 集成测试（需要数据库）
make docker-up
make test-integration

# 所有测试
make test-all

# 特定包
go test -short ./internal/parser/... -v

# 特定测试
go test -short ./internal/parser -run TestGoParser_Functions -v
```

### 测试覆盖率

```bash
# 生成覆盖率报告
make test-coverage-all

# 查看 HTML 报告
go tool cover -html=coverage_all.out

# 查看函数级统计
make test-coverage-func
```

**覆盖率目标**：
- 单元测试：90%+
- 集成测试：85%+
- 整体：90%+

### 编写测试

#### 单元测试模板

```go
package parser

import (
    "testing"
)

func TestGoParser_Functions(t *testing.T) {
    // 跳过集成测试
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    tests := []struct {
        name    string
        input   string
        want    int
        wantErr bool
    }{
        {
            name:  "simple function",
            input: "func main() {}",
            want:  1,
        },
        {
            name:    "invalid syntax",
            input:   "func {",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewGoParser()
            result, err := parser.Parse(tt.input)

            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                }
                return
            }

            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            if len(result.Symbols) != tt.want {
                t.Errorf("got %d symbols, want %d", len(result.Symbols), tt.want)
            }
        })
    }
}
```

#### 集成测试模板

```go
package integration

import (
    "testing"
    "github.com/yourtionguo/CodeAtlas/pkg/models"
)

func TestSymbolRepository_Create(t *testing.T) {
    // 跳过单元测试模式
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // 设置测试数据库
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    repo := models.NewSymbolRepository(db)

    symbol := &models.Symbol{
        Name: "testFunction",
        Kind: "function",
    }

    // 测试创建
    err := repo.Create(symbol)
    if err != nil {
        t.Fatalf("failed to create symbol: %v", err)
    }

    // 验证
    found, err := repo.GetByID(symbol.ID)
    if err != nil {
        t.Fatalf("failed to get symbol: %v", err)
    }

    if found.Name != symbol.Name {
        t.Errorf("got name %s, want %s", found.Name, symbol.Name)
    }

    // 清理
    err = repo.Delete(symbol.ID)
    if err != nil {
        t.Fatalf("failed to delete symbol: %v", err)
    }
}
```

### 测试最佳实践

1. **测试隔离**：每个测试独立，不依赖其他测试
2. **清理数据**：测试后清理测试数据
3. **使用子测试**：使用 `t.Run()` 组织测试
4. **描述性命名**：`TestPackage_Function_Scenario`
5. **表驱动测试**：使用测试表处理多个场景
6. **Mock 外部依赖**：单元测试中 mock 数据库、API 等

## 代码规范

### Go 代码规范

遵循标准 Go 规范：

```bash
# 格式化代码
make fmt
# 或
gofmt -w .

# 代码检查
make lint
# 或
golangci-lint run
```

**关键规范**：
- 使用 `gofmt` 格式化
- 导出的函数和类型必须有文档注释
- 错误处理：返回 error 作为最后一个返回值
- 使用 `context.Context` 处理取消和超时
- 接口优先，便于测试和模块化

**示例**：

```go
// ParseFile 解析单个文件并返回 AST
// 如果文件不存在或解析失败，返回错误
func ParseFile(ctx context.Context, path string) (*AST, error) {
    // 检查上下文
    if err := ctx.Err(); err != nil {
        return nil, err
    }

    // 读取文件
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }

    // 解析
    ast, err := parse(content)
    if err != nil {
        return nil, fmt.Errorf("failed to parse: %w", err)
    }

    return ast, nil
}
```

### 前端代码规范

```bash
# 格式化
cd web
pnpm format

# 检查
pnpm lint
```

**关键规范**：
- 使用 TypeScript，避免 `any`
- 组件文件使用 PascalCase
- 工具函数使用 camelCase
- 使用 Prettier 格式化

### 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/)：

```bash
# 格式
<type>(<scope>): <subject>

# 类型
feat:     新功能
fix:      修复 bug
docs:     文档更新
style:    代码格式（不影响功能）
refactor: 重构
test:     测试
chore:    构建/工具

# 示例
feat(parser): add Swift parser support
fix(api): handle nil pointer in search endpoint
docs: update deployment guide
test(parser): add tests for Go parser
```

## 数据库开发

### 迁移

```bash
# 创建迁移
cat > deployments/migrations/003_add_new_table.sql << EOF
-- Up
CREATE TABLE new_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);

-- Down
DROP TABLE new_table;
EOF

# 应用迁移
make db-migrate

# 回滚
make db-rollback
```

### 查询优化

```sql
-- 使用 EXPLAIN ANALYZE
EXPLAIN ANALYZE
SELECT * FROM symbols WHERE name = 'main';

-- 创建索引
CREATE INDEX idx_symbols_name ON symbols(name);

-- 检查索引使用
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
ORDER BY idx_scan;
```

## 性能分析

### CPU 分析

```bash
# 生成 CPU profile
go test -cpuprofile=cpu.prof -bench=.

# 分析
go tool pprof cpu.prof
(pprof) top10
(pprof) web
```

### 内存分析

```bash
# 生成内存 profile
go test -memprofile=mem.prof -bench=.

# 分析
go tool pprof mem.prof
(pprof) top10
(pprof) list FunctionName
```

### 基准测试

```go
func BenchmarkGoParser_Parse(b *testing.B) {
    parser := NewGoParser()
    code := `package main
func main() {
    println("hello")
}`

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := parser.Parse(code)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

运行基准测试：

```bash
# 运行基准测试
go test -bench=. ./internal/parser/

# 比较性能
go test -bench=. -benchmem ./internal/parser/
```

## 常用命令

### Makefile 命令

```bash
# 构建
make build              # 构建所有
make build-api          # 构建 API
make build-cli          # 构建 CLI

# 运行
make run-api            # 运行 API
make run-cli            # 运行 CLI

# 测试
make test-unit          # 单元测试
make test-integration   # 集成测试
make test-all           # 所有测试
make test-coverage-all  # 覆盖率

# 代码质量
make fmt                # 格式化
make lint               # 检查
make vet                # 静态分析

# Docker
make docker-up          # 启动数据库
make docker-down        # 停止服务
make docker-clean       # 清理数据

# 数据库
make db-init            # 初始化
make db-migrate         # 迁移
make db-seed            # 填充测试数据

# 清理
make clean              # 清理构建产物
```

### Git 工作流

```bash
# 更新主分支
git checkout main
git pull origin main

# 创建功能分支
git checkout -b feature/my-feature

# 开发...

# 提交
git add .
git commit -m "feat: add new feature"

# 推送
git push origin feature/my-feature

# 创建 Pull Request
# 在 GitHub 上创建 PR

# 合并后清理
git checkout main
git pull origin main
git branch -d feature/my-feature
```

## 故障排除

### 构建失败

```bash
# 清理缓存
go clean -cache -modcache -testcache

# 重新下载依赖
rm go.sum
go mod tidy
go mod download
```

### 测试失败

```bash
# 清理测试数据
make docker-clean
make docker-up

# 重新运行
make test-all
```

### 数据库问题

```bash
# 重置数据库
make docker-down
make docker-clean
make docker-up
make db-init
```

## 贡献指南

### 提交 Pull Request

1. Fork 仓库
2. 创建功能分支
3. 编写代码和测试
4. 确保测试通过
5. 提交 PR

**PR 检查清单**：
- [ ] 代码遵循规范
- [ ] 添加了测试
- [ ] 测试通过（`make test-all`）
- [ ] 更新了文档
- [ ] 提交信息符合规范

### Code Review

**审查重点**：
- 代码质量和可读性
- 测试覆盖率
- 性能影响
- 安全问题
- 文档完整性

## 开发工具

### 推荐 VS Code 扩展

- Go (golang.go)
- Svelte (svelte.svelte-vscode)
- PostgreSQL (ckolkman.vscode-postgres)
- GitLens (eamodio.gitlens)
- Error Lens (usernamehw.errorlens)

### 推荐命令行工具

```bash
# Go 工具
go install golang.org/x/tools/gopls@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 数据库工具
brew install postgresql@17
brew install pgcli

# 其他工具
brew install jq          # JSON 处理
brew install httpie      # HTTP 客户端
brew install k9s         # Kubernetes 管理
```

## 资源

### 内部文档
- [快速开始](quick-start.md)
- [CLI 工具](cli.md)
- [API 服务](api.md)
- [架构设计](architecture.md)

### 外部资源
- [Go 文档](https://golang.org/doc/)
- [Svelte 文档](https://svelte.dev/docs)
- [PostgreSQL 文档](https://www.postgresql.org/docs/)
- [Tree-sitter 文档](https://tree-sitter.github.io/tree-sitter/)

## 下一步

- 查看 [架构设计](architecture.md) 了解系统设计
- 查看 [贡献指南](../CONTRIBUTING.md) 了解贡献流程
- 加入 [GitHub Discussions](https://github.com/yourtionguo/CodeAtlas/discussions)
