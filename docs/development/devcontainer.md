# DevContainer 开发环境完整指南

> 使用 DevContainer 获得开箱即用的完整开发环境

## 概述

CodeAtlas 提供了完整的 DevContainer 配置，让你可以在几分钟内启动一个包含所有依赖和测试数据的开发环境，无需手动安装任何工具。

## 特性

### 🚀 开箱即用
- **Go 1.25** 开发环境（包含 gopls、delve、golangci-lint）
- **Node.js 20 + pnpm**（用于前端开发）
- **PostgreSQL 17**（带 pgvector 和 AGE 扩展）
- **预置测试数据**（3个示例仓库，多个代码文件）

### 🔧 VS Code 集成
- 自动安装推荐扩展
- 预配置的调试器设置
- 代码格式化和 lint 自动运行
- PostgreSQL 数据库客户端

### 📦 持久化存储
- Go modules 缓存
- pnpm store 缓存
- PostgreSQL 数据持久化

### ⚡ 性能优化
- 使用命名卷缓存依赖
- 多阶段构建减少镜像大小
- 并行初始化加速启动

## 快速开始

### 方式 1: VS Code（推荐）

1. **安装扩展**
   
   安装 [Dev Containers 扩展](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

2. **打开容器**
   
   打开项目，点击左下角的远程连接按钮，选择 "Reopen in Container"
   
   或使用命令面板：`Dev Containers: Reopen in Container`

3. **等待初始化**
   
   首次构建约 3-5 分钟，包括：
   - 构建开发容器镜像
   - 安装 Go 依赖
   - 安装前端依赖
   - 初始化数据库
   - 加载测试数据

4. **开始开发**
   
   容器启动后，所有工具和服务都已就绪！

### 方式 2: GitHub Codespaces

1. 在 GitHub 仓库页面点击 "Code" → "Codespaces"
2. 点击 "Create codespace on main"
3. 等待环境初始化完成（约 5-7 分钟）
4. 开始开发！

### 方式 3: 命令行

```bash
# 构建并启动 devcontainer
make devcontainer-build
make devcontainer-up

# 进入开发容器
docker exec -it codeatlas-dev-1 bash

# 在容器内运行测试
./scripts/test_devcontainer.sh
```

## 开发工作流

### 启动服务

#### API 服务器

```bash
# 在容器内
make run-api

# 或直接运行
cd cmd/api
go run main.go
```

API 服务器将在 `http://localhost:8080` 启动

#### 前端开发服务器

```bash
cd web
pnpm dev
```

前端将在 `http://localhost:3000` 启动

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定测试套件
make test-api          # API 测试
make test-cli          # CLI 测试
make test-models       # 数据库模型测试
make test-integration  # 集成测试

# 生成测试覆盖率报告
make test-coverage-all
open coverage_all.html
```

### 使用 CLI 工具

```bash
# 构建 CLI
make build-cli

# 解析代码
./bin/cli parse --path /path/to/repo --output result.json

# 索引代码
./bin/cli index \
  --path /path/to/repo \
  --name "my-project" \
  --api-url http://localhost:8080
```

## 数据库访问

### 连接信息

| 参数 | 值 |
|------|-----|
| Host | `db` |
| Port | `5432` |
| Database | `codeatlas` |
| Username | `codeatlas` |
| Password | `codeatlas` |

### 使用 psql

```bash
# 连接数据库
psql -h db -U codeatlas -d codeatlas

# 常用查询
\dt                    # 列出所有表
\d repositories        # 查看表结构
SELECT * FROM repositories;  # 查询数据
```

### 使用 VS Code PostgreSQL 扩展

1. 点击左侧的 PostgreSQL 图标
2. 添加新连接，使用上述连接信息
3. 浏览表结构和数据
4. 执行 SQL 查询

### 查看测试数据

```sql
-- 查看所有仓库
SELECT repo_id, name, branch, created_at 
FROM repositories 
ORDER BY created_at;

-- 查看文件统计
SELECT 
    language,
    COUNT(*) as file_count,
    SUM(size) as total_size
FROM files
GROUP BY language;

-- 查看符号分布
SELECT 
    kind,
    COUNT(*) as count
FROM symbols
GROUP BY kind
ORDER BY count DESC;

-- 查看依赖关系
SELECT 
    sf.path as source_file,
    tf.path as target_file,
    e.edge_type
FROM edges e
JOIN symbols ss ON e.source_symbol_id = ss.symbol_id
JOIN symbols ts ON e.target_symbol_id = ts.symbol_id
JOIN files sf ON ss.file_id = sf.file_id
JOIN files tf ON ts.file_id = tf.file_id;
```

## 预置测试数据

DevContainer 包含以下测试数据，可以立即用于开发和测试：

### 仓库

| 名称 | 语言 | 描述 |
|------|------|------|
| sample-go-api | Go | REST API 项目示例 |
| sample-frontend | JavaScript | Svelte 前端应用 |
| sample-microservice | Go | 微服务架构示例 |

### 代码文件

- **Go 文件**
  - `main.go` - API 主入口
  - `models/user.go` - User 模型
  - `handlers/user_handler.go` - 用户处理器

- **Svelte 文件**
  - `src/App.svelte` - 主组件
  - `src/components/UserList.svelte` - 用户列表组件

### 符号和关系

- 3 个函数（main, healthCheck, getUsers）
- 2 个结构体（User, UserHandler）
- 2 个方法（Validate, GetUser）
- 多个导入和调用关系

## 调试

### 调试 Go 代码

VS Code 已预配置调试器，使用方法：

1. 在代码中设置断点
2. 按 `F5` 或点击 "Run and Debug"
3. 选择 "Debug API" 或 "Debug CLI"
4. 开始调试

手动配置示例（`.vscode/launch.json`）：

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
        "DB_HOST": "db",
        "DB_PORT": "5432",
        "DB_USER": "codeatlas",
        "DB_PASSWORD": "codeatlas",
        "DB_NAME": "codeatlas"
      }
    },
    {
      "name": "Debug CLI Parse",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/cli",
      "args": [
        "parse",
        "--path", "/workspace/examples/sample-project",
        "--verbose"
      ]
    }
  ]
}
```

### 调试前端

```bash
cd web
pnpm dev
```

在浏览器中打开 http://localhost:3000，使用浏览器开发者工具调试。

### 调试测试

```bash
# 运行单个测试并启用调试
go test -v ./internal/parser/... -run TestGoParser

# 使用 VS Code 调试测试
# 在测试函数上右键 → "Debug Test"
```

## 常见问题

### 数据库连接失败

**症状**：API 启动时报错 "failed to connect to database"

**解决方案**：

```bash
# 1. 检查数据库是否就绪
pg_isready -h db -U codeatlas -d codeatlas

# 2. 查看数据库日志
make devcontainer-logs

# 3. 重启数据库容器
docker-compose -f .devcontainer/docker-compose.yml restart db

# 4. 等待数据库完全启动（约 10 秒）
sleep 10
pg_isready -h db -U codeatlas -d codeatlas
```

### 容器构建失败

**症状**：容器构建过程中出错

**解决方案**：

```bash
# 1. 清理所有容器和卷
make devcontainer-clean

# 2. 重新构建
make devcontainer-build

# 3. 如果仍然失败，检查 Docker 资源
docker system df
docker system prune  # 清理未使用的资源
```

### 端口冲突

**症状**：端口已被占用

**解决方案**：

修改 `.devcontainer/docker-compose.yml` 中的端口映射：

```yaml
services:
  app:
    ports:
      - "8081:8080"  # 将 API 端口改为 8081
  db:
    ports:
      - "5433:5432"  # 将数据库端口改为 5433
```

### 性能问题

**症状**：容器运行缓慢

**解决方案**：

1. **增加 Docker 资源**
   - Docker Desktop → Settings → Resources
   - 建议：至少 4GB 内存，2 CPU 核心

2. **使用 WSL2**（Windows 用户）
   - WSL2 比 Hyper-V 性能更好
   - Docker Desktop → Settings → General → Use WSL2

3. **使用 SSD**
   - 将项目和 Docker 数据存储在 SSD 上

4. **清理缓存**
   ```bash
   # 清理 Docker 缓存
   docker system prune -a
   
   # 清理 Go 缓存
   go clean -cache -modcache
   ```

### 扩展未自动安装

**症状**：VS Code 扩展没有自动安装

**解决方案**：

1. 打开命令面板（Cmd/Ctrl + Shift + P）
2. 运行 "Dev Containers: Rebuild Container"
3. 或手动安装扩展：
   - Go (golang.go)
   - Svelte (svelte.svelte-vscode)
   - PostgreSQL (cweijan.vscode-postgresql-client2)

## 自定义配置

### 添加 VS Code 扩展

编辑 `.devcontainer/devcontainer.json`：

```json
{
  "extensions": [
    "golang.go",
    "svelte.svelte-vscode",
    "your.extension-id"  // 添加你的扩展
  ]
}
```

### 修改测试数据

编辑 `scripts/seed_data.sql`：

```sql
-- 添加你的测试数据
INSERT INTO repositories (repo_id, name, url, branch)
VALUES (
  gen_random_uuid(),
  'my-test-repo',
  'https://github.com/user/repo',
  'main'
);
```

然后重建容器：

```bash
make devcontainer-clean
make devcontainer-build
```

### 添加环境变量

编辑 `.devcontainer/docker-compose.yml`：

```yaml
services:
  app:
    environment:
      - DB_HOST=db
      - YOUR_CUSTOM_VAR=value
      - ANOTHER_VAR=another_value
```

### 修改初始化脚本

编辑 `scripts/init_devcontainer.sh`：

```bash
#!/bin/bash

# 添加你的初始化逻辑
echo "Running custom initialization..."

# 安装额外的工具
go install github.com/your/tool@latest

# 设置别名
echo "alias ll='ls -la'" >> ~/.bashrc
```

## 环境验证

运行测试脚本验证环境配置：

```bash
./scripts/test_devcontainer.sh
```

该脚本会检查：

- ✅ Go 安装和版本
- ✅ Go 工具链（gopls, delve, golangci-lint）
- ✅ Node.js 和 pnpm
- ✅ 数据库连接
- ✅ 数据库扩展（pgvector, AGE）
- ✅ 测试数据完整性
- ✅ 项目构建
- ✅ 二进制文件生成

## 与生产环境的差异

DevContainer 针对开发优化，与生产环境的主要差异：

| 特性 | DevContainer | 生产环境 |
|------|-------------|---------|
| 数据库 | 单容器 PostgreSQL | 独立数据库服务/集群 |
| 数据持久化 | Docker 卷 | 持久化存储（EBS/PD） |
| 日志 | 标准输出 | 日志聚合系统 |
| 监控 | 无 | Prometheus/Grafana |
| 安全 | 开发密码 | 密钥管理系统 |
| 性能 | 开发优化 | 生产优化 |
| 备份 | 无 | 自动备份 |
| 高可用 | 单实例 | 多实例/集群 |

## 性能优化建议

### 1. 使用命名卷

DevContainer 已配置命名卷来缓存依赖：

```yaml
volumes:
  go-modules:      # Go 模块缓存
  pnpm-store:      # pnpm 包缓存
  postgres-data:   # 数据库数据
```

这些卷在容器重建时保留，显著提升启动速度。

### 2. 并行构建

在 `.devcontainer/Dockerfile` 中使用并行构建：

```dockerfile
# 并行安装 Go 工具
RUN go install golang.org/x/tools/gopls@latest & \
    go install github.com/go-delve/delve/cmd/dlv@latest & \
    wait
```

### 3. 分层缓存

Dockerfile 使用分层缓存，频繁变化的层放在后面：

```dockerfile
# 1. 基础镜像（很少变化）
FROM golang:1.25

# 2. 系统依赖（偶尔变化）
RUN apt-get update && apt-get install -y ...

# 3. Go 依赖（经常变化）
COPY go.mod go.sum ./
RUN go mod download

# 4. 源代码（最常变化）
COPY . .
```

### 4. 资源限制

在 `docker-compose.yml` 中设置合理的资源限制：

```yaml
services:
  app:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

## CI/CD 集成

GitHub Actions workflow 已配置（`.github/workflows/devcontainer-test.yml`）：

```yaml
name: DevContainer Test

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build DevContainer
        run: make devcontainer-build
      
      - name: Test DevContainer
        run: |
          make devcontainer-up
          docker exec codeatlas-dev-1 ./scripts/test_devcontainer.sh
```

## 相关文档

- [快速开始指南](../getting-started/quick-start.md)
- [测试指南](./testing.md)
- [贡献指南](../../CONTRIBUTING.md)
- [VS Code Dev Containers 文档](https://code.visualstudio.com/docs/devcontainers/containers)
- [GitHub Codespaces 文档](https://docs.github.com/en/codespaces)

## 获取帮助

- 📖 查看 [故障排除指南](../troubleshooting/README.md)
- 🐛 [报告问题](https://github.com/yourtionguo/CodeAtlas/issues)
- 💬 [讨论区](https://github.com/yourtionguo/CodeAtlas/discussions)
