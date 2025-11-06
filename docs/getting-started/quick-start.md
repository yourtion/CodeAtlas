# 快速开始指南

> 5 分钟内启动 CodeAtlas 并开始使用

## 前置要求

- Docker 和 Docker Compose
- Go 1.25+ （本地开发）
- 4GB+ 内存
- 20GB+ 磁盘空间

## 三种启动方式

### 方式 1: DevContainer（推荐）⭐

最简单的方式，开箱即用的完整开发环境。

**VS Code:**
1. 安装 [Dev Containers 扩展](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. 打开项目，点击 "Reopen in Container"
3. 等待容器构建完成（首次约 3-5 分钟）
4. 开始开发！

**GitHub Codespaces:**
- 点击 "Code" → "Codespaces" → "Create codespace"

**命令行:**
```bash
make devcontainer-up
```

详细文档：[DevContainer 开发指南](../development/devcontainer.md)

### 方式 2: Docker Compose

适合快速测试和演示。

```bash
# 1. 启动所有服务
docker-compose up -d

# 2. 检查服务状态
docker-compose ps

# 3. 查看日志
docker-compose logs -f api
```

服务地址：
- API: http://localhost:8080
- 数据库: localhost:5432

### 方式 3: 本地开发

适合需要完全控制的开发场景。

```bash
# 1. 启动数据库
make docker-up

# 2. 运行 API 服务
make run-api

# 3. 运行前端（另一个终端）
cd web
pnpm install
pnpm dev
```

## 第一次使用

### 1. 验证安装

```bash
# 检查 API 健康状态
curl http://localhost:8080/health

# 预期输出
# {"status":"ok","message":"CodeAtlas API server is running"}
```

### 2. 解析代码仓库

```bash
# 构建 CLI 工具
make build-cli

# 解析本地仓库
./bin/cli parse --path /path/to/your/repo --output result.json

# 查看解析结果
cat result.json | jq '.summary'
```

### 3. 索引到知识图谱

```bash
# 索引解析结果
./bin/cli index \
  --path /path/to/your/repo \
  --name "my-project" \
  --api-url http://localhost:8080
```

### 4. 查询代码

```bash
# 列出所有仓库
curl http://localhost:8080/api/v1/repositories

# 搜索代码（需要先配置向量模型）
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "authentication function",
    "limit": 10
  }'
```

## 常用命令

### 开发命令

```bash
# 构建
make build              # 构建所有二进制文件
make build-api          # 只构建 API
make build-cli          # 只构建 CLI

# 运行
make run-api            # 启动 API 服务器
make run-cli            # 运行 CLI 工具

# 测试
make test               # 运行所有测试
make test-unit          # 只运行单元测试
make test-coverage      # 生成覆盖率报告

# Docker
make docker-up          # 启动数据库
make docker-down        # 停止所有服务
```

### CLI 命令

```bash
# 解析代码
codeatlas parse --path /path/to/repo

# 解析单个文件
codeatlas parse --file main.go

# 解析特定语言
codeatlas parse --path /path/to/repo --language go

# 索引代码
codeatlas index --path /path/to/repo --name "project-name"

# 查看帮助
codeatlas --help
codeatlas parse --help
```

## 配置

### 最小配置

使用默认配置即可开始：

```bash
# 复制示例配置
cp .env.example .env

# 使用默认值启动
make run-api
```

### 常用配置

```bash
# 数据库
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
export DB_NAME=codeatlas

# API 服务
export API_PORT=8080

# 索引器
export INDEXER_BATCH_SIZE=100
export INDEXER_WORKER_COUNT=4

# 向量模型（可选）
export EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
export EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
export EMBEDDING_DIMENSIONS=1024
```

完整配置文档：[配置指南](../configuration/README.md)

## 故障排除

### API 无法启动

```bash
# 检查数据库是否运行
docker-compose ps

# 查看 API 日志
docker-compose logs api

# 检查端口占用
lsof -i :8080
```

### 数据库连接失败

```bash
# 测试数据库连接
psql -h localhost -U codeatlas -d codeatlas

# 重启数据库
make docker-down
make docker-up
```

### CLI 解析失败

```bash
# 启用详细日志
codeatlas parse --path /path/to/repo --verbose

# 检查文件权限
ls -la /path/to/repo
```

更多问题：[故障排除指南](../troubleshooting/README.md)

## 下一步

### 学习更多

- [CLI 工具详细文档](../user-guide/cli/README.md)
- [API 使用指南](../user-guide/api/README.md)
- [配置参考](../configuration/README.md)
- [开发指南](../development/README.md)

### 部署到生产

- [Docker 部署](../deployment/docker.md)
- [Systemd 部署](../deployment/systemd.md)
- [生产环境最佳实践](../deployment/production.md)

### 参与贡献

- [贡献指南](../../CONTRIBUTING.md)
- [开发环境设置](../development/devcontainer.md)
- [测试指南](../development/testing.md)

## 获取帮助

- 📖 [完整文档](../README.md)
- 🐛 [报告问题](https://github.com/yourtionguo/CodeAtlas/issues)
- 💬 [讨论区](https://github.com/yourtionguo/CodeAtlas/discussions)
- 📧 联系维护者

## 示例项目

查看 `examples/` 目录获取完整的使用示例：

- `examples/simple-go-project/` - 简单的 Go 项目示例
- `examples/multi-language/` - 多语言项目示例
- `examples/large-codebase/` - 大型代码库处理示例
