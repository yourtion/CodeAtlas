# DevContainer 开发环境指南

## 概述

CodeAtlas 提供了完整的 DevContainer 配置，让你可以在几分钟内启动一个包含所有依赖和测试数据的开发环境。

## 特性

### 🚀 开箱即用
- Go 1.25 开发环境（包含 gopls、delve、golangci-lint）
- Node.js 20 + pnpm（用于前端开发）
- PostgreSQL 17（带 pgvector 和 AGE 扩展）
- 预置测试数据（3个示例仓库，多个代码文件）

### 🔧 VS Code 集成
- 自动安装推荐扩展
- 预配置的调试器设置
- 代码格式化和 lint 自动运行
- PostgreSQL 数据库客户端

### 📦 持久化存储
- Go modules 缓存
- pnpm store 缓存
- PostgreSQL 数据持久化

## 快速开始

### 方式 1: VS Code（推荐）

1. 安装 [Dev Containers 扩展](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

2. 打开项目，点击左下角的远程连接按钮，选择 "Reopen in Container"
   
   或使用命令面板：`Dev Containers: Reopen in Container`

3. 等待容器构建（首次约 3-5 分钟）

4. 容器启动后，自动执行：
   - 安装 Go 依赖
   - 安装前端依赖
   - 初始化数据库
   - 构建项目

### 方式 2: GitHub Codespaces

1. 在 GitHub 仓库页面点击 "Code" → "Codespaces"
2. 点击 "Create codespace on main"
3. 等待环境初始化完成

### 方式 3: 命令行（不使用 VS Code）

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

### 启动 API 服务器

```bash
make run-api
```

API 服务器将在 `http://localhost:8080` 启动

### 启动前端开发服务器

```bash
cd web
pnpm dev
```

前端将在 `http://localhost:3000` 启动

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定测试
make test-api      # API 测试
make test-cli      # CLI 测试
make test-models   # 数据库模型测试

# 生成测试覆盖率报告
make test-coverage
```

### 使用 CLI 工具

```bash
# 上传代码仓库
make run-cli upload -p /path/to/repo -s http://localhost:8080

# 查询代码
make run-cli query -q "如何实现用户认证" -s http://localhost:8080
```

## 数据库访问

### 连接信息

- **Host**: `db`
- **Port**: `5432`
- **Database**: `codeatlas`
- **Username**: `codeatlas`
- **Password**: `codeatlas`

### 使用 psql

```bash
psql -h db -U codeatlas -d codeatlas
```

### 使用 VS Code PostgreSQL 扩展

1. 点击左侧的 PostgreSQL 图标
2. 添加新连接，使用上述连接信息
3. 浏览表结构和数据

### 查看测试数据

```sql
-- 查看所有仓库
SELECT * FROM repositories;

-- 查看文件
SELECT id, path, language FROM files;

-- 查看符号
SELECT s.name, s.kind, f.path 
FROM symbols s 
JOIN files f ON s.file_id = f.id;

-- 查看依赖关系
SELECT 
    sf.path as source,
    tf.path as target,
    d.dependency_type
FROM dependencies d
JOIN files sf ON d.source_file_id = sf.id
JOIN files tf ON d.target_file_id = tf.id;
```

## 预置测试数据

DevContainer 包含以下测试数据：

### 仓库
1. **sample-go-api**: Go REST API 项目
2. **sample-frontend**: Svelte 前端应用
3. **sample-microservice**: 微服务架构示例

### 代码文件
- Go 源文件（main.go, models, handlers）
- Svelte 组件（App.svelte, UserList.svelte）
- 包含真实的代码内容和结构

### 符号和依赖
- 函数、结构体、方法定义
- 文件间的导入依赖关系
- Mock 向量嵌入数据

## 调试

### 调试 Go 代码

VS Code 已预配置调试器，按 F5 即可启动调试。

或手动配置 `.vscode/launch.json`:

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
        "DB_HOST": "db"
      }
    }
  ]
}
```

### 调试前端

```bash
cd web
pnpm dev
```

在浏览器中使用开发者工具调试。

## 常见问题

### 数据库连接失败

检查数据库是否就绪：

```bash
pg_isready -h db -U codeatlas -d codeatlas
```

查看数据库日志：

```bash
make devcontainer-logs
```

### 容器构建失败

清理并重建：

```bash
make devcontainer-clean
make devcontainer-build
```

### 端口冲突

如果端口被占用，修改 `.devcontainer/docker-compose.yml` 中的端口映射：

```yaml
ports:
  - "8081:8080"  # 将 API 端口改为 8081
```

### 性能问题

DevContainer 使用命名卷来缓存依赖，提升性能：
- `go-modules`: Go 模块缓存
- `pnpm-store`: pnpm 包缓存
- `postgres-data`: 数据库数据

如需清理缓存：

```bash
make devcontainer-clean
```

## 自定义配置

### 添加 VS Code 扩展

编辑 `.devcontainer/devcontainer.json`:

```json
"extensions": [
  "golang.go",
  "your.extension-id"
]
```

### 修改测试数据

编辑 `scripts/seed_data.sql`，然后重建容器。

### 添加环境变量

编辑 `.devcontainer/docker-compose.yml`:

```yaml
environment:
  - DB_HOST=db
  - YOUR_VAR=value
```

## 测试环境验证

运行测试脚本验证环境：

```bash
./scripts/test_devcontainer.sh
```

该脚本会检查：
- Go 和工具链安装
- Node.js 和 pnpm
- 数据库连接和数据
- 项目构建

## 性能优化建议

1. **使用 WSL2**（Windows 用户）：比 Docker Desktop 性能更好
2. **分配足够资源**：建议至少 4GB 内存，2 CPU 核心
3. **使用 SSD**：显著提升容器启动和构建速度
4. **保持容器运行**：避免频繁重启容器

## 与生产环境的差异

DevContainer 针对开发优化，与生产环境的主要差异：

| 特性 | DevContainer | 生产环境 |
|------|-------------|---------|
| 数据库 | 单容器 | 独立服务/集群 |
| 数据持久化 | Docker 卷 | 持久化存储 |
| 日志 | 标准输出 | 日志聚合系统 |
| 监控 | 无 | Prometheus/Grafana |
| 安全 | 开发密码 | 密钥管理系统 |

## 更多资源

- [VS Code Dev Containers 文档](https://code.visualstudio.com/docs/devcontainers/containers)
- [GitHub Codespaces 文档](https://docs.github.com/en/codespaces)
- [Docker Compose 文档](https://docs.docker.com/compose/)
- [CodeAtlas 主文档](../README.md)
