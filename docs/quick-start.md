# 快速开始

> 5 分钟启动 CodeAtlas

## 前置要求

- Docker 和 Docker Compose
- 4GB+ 内存

## 启动方式

### 方式 1: DevContainer（推荐）

**VS Code:**
1. 安装 [Dev Containers 扩展](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. 打开项目 → "Reopen in Container"
3. 等待构建完成（首次 3-5 分钟）

**命令行:**
```bash
make devcontainer-up
```

### 方式 2: Docker Compose

```bash
# 启动服务
docker-compose up -d

# 检查状态
docker-compose ps
```

服务地址：
- API: http://localhost:8080
- 数据库: localhost:5432

### 方式 3: 本地开发

```bash
# 1. 启动数据库
make docker-up

# 2. 构建项目
make build

# 3. 运行 API
make run-api
```

## 第一次使用

### 1. 解析代码

```bash
# 解析整个仓库
./bin/cli parse --path /path/to/repo --output result.json

# 解析单个文件
./bin/cli parse --file main.go
```

### 2. 索引到数据库

```bash
# 索引解析结果
./bin/cli index --input result.json --repo-name myproject
```

### 3. 查询代码

```bash
# 搜索函数
curl "http://localhost:8080/api/v1/search?q=main&type=function"

# 查找调用关系
curl "http://localhost:8080/api/v1/relationships?symbol_id=xxx&type=call"
```

## 常用命令

```bash
# 构建
make build              # 构建所有
make build-api          # 只构建 API
make build-cli          # 只构建 CLI

# 测试
make test-unit          # 单元测试（快速）
make test-all           # 所有测试

# Docker
make docker-up          # 启动数据库
make docker-down        # 停止服务
make docker-clean       # 清理数据

# 开发
make run-api            # 运行 API
cd web && pnpm dev      # 运行前端
```

## 环境变量

创建 `.env` 文件：

```bash
# 数据库
DB_HOST=localhost
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=codeatlas
DB_NAME=codeatlas

# API
API_PORT=8080

# 向量模型（可选）
OPENAI_API_KEY=your-key
OPENAI_MODEL=text-embedding-3-small
```

## 验证安装

```bash
# 检查 API
curl http://localhost:8080/health

# 检查数据库
psql -U codeatlas -d codeatlas -c "SELECT version();"

# 检查 CLI
./bin/cli --version
```

## 下一步

- 查看 [CLI 工具指南](cli.md) 了解更多解析选项
- 查看 [API 指南](api.md) 了解所有 API 端点
- 查看 [配置指南](configuration.md) 自定义配置
- 遇到问题？查看 [故障排除](troubleshooting.md)

## 常见问题

**Q: 端口被占用怎么办？**
```bash
# 修改 docker-compose.yml 中的端口
ports:
  - "8081:8080"  # API
  - "5433:5432"  # 数据库
```

**Q: 数据库连接失败？**
```bash
# 检查数据库是否运行
docker-compose ps

# 查看日志
docker-compose logs postgres
```

**Q: 解析失败？**
```bash
# 使用 verbose 模式查看详细信息
./bin/cli parse --path . --verbose
```
