# 配置完整指南

> CodeAtlas 的所有配置选项和最佳实践

## 目录

- [概述](#概述)
- [快速配置](#快速配置)
- [数据库配置](#数据库配置)
- [API 服务器配置](#api-服务器配置)
- [索引器配置](#索引器配置)
- [向量模型配置](#向量模型配置)
- [安全配置](#安全配置)
- [配置示例](#配置示例)

## 概述

CodeAtlas 使用环境变量进行配置，提供合理的默认值，开箱即用。

### 配置方式

1. **环境变量**（推荐）
2. **.env 文件**
3. **命令行参数**（部分支持）

### 配置优先级

命令行参数 > 环境变量 > .env 文件 > 默认值

## 快速配置

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

## 数据库配置

### 连接配置

| 变量 | 类型 | 默认值 | 说明 |
|----------|------|---------|-------------|
| `DB_HOST` | string | `localhost` | PostgreSQL 服务器主机名 |
| `DB_PORT` | int | `5432` | PostgreSQL 服务器端口 |
| `DB_USER` | string | `codeatlas` | 数据库用户名 |
| `DB_PASSWORD` | string | `codeatlas` | 数据库密码 |
| `DB_NAME` | string | `codeatlas` | 数据库名称 |
| `DB_SSLMODE` | string | `disable` | SSL 模式 (`disable`, `require`, `verify-ca`, `verify-full`) |

### 连接池配置

| 变量 | 类型 | 默认值 | 说明 |
|----------|------|---------|-------------|
| `DB_MAX_OPEN_CONNS` | int | `25` | 最大打开连接数 |
| `DB_MAX_IDLE_CONNS` | int | `5` | 最大空闲连接数 |
| `DB_CONN_MAX_LIFETIME` | duration | `5m` | 连接最大生命周期 |
| `DB_CONN_MAX_IDLE_TIME` | duration | `5m` | 连接最大空闲时间 |

### 配置示例

#### 开发环境

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
export DB_NAME=codeatlas
export DB_SSLMODE=disable
export DB_MAX_OPEN_CONNS=10
export DB_MAX_IDLE_CONNS=2
```

#### 生产环境

```bash
export DB_HOST=db.production.example.com
export DB_PORT=5432
export DB_USER=codeatlas_prod
export DB_PASSWORD=<secure-password>
export DB_NAME=codeatlas_prod
export DB_SSLMODE=require
export DB_MAX_OPEN_CONNS=50
export DB_MAX_IDLE_CONNS=10
export DB_CONN_MAX_LIFETIME=10m
```

#### 高吞吐量场景

```bash
export DB_MAX_OPEN_CONNS=100
export DB_MAX_IDLE_CONNS=20
export DB_CONN_MAX_LIFETIME=15m
```

#### 低资源环境

```bash
export DB_MAX_OPEN_CONNS=5
export DB_MAX_IDLE_CONNS=2
export DB_CONN_MAX_LIFETIME=3m
```

## API 服务器配置

### 基本配置

| 变量 | 类型 | 默认值 | 说明 |
|----------|------|---------|-------------|
| `API_HOST` | string | `0.0.0.0` | 服务器绑定地址 |
| `API_PORT` | int | `8080` | 服务器端口 |
| `API_TIMEOUT` | duration | `30s` | 请求超时 |

### 认证配置

| 变量 | 类型 | 默认值 | 说明 |
|----------|------|---------|-------------|
| `ENABLE_AUTH` | bool | `false` | 启用认证中间件 |
| `AUTH_TOKENS` | string | `` | 逗号分隔的有效令牌列表 |

### CORS 配置

| 变量 | 类型 | 默认值 | 说明 |
|----------|------|---------|-------------|
| `CORS_ORIGINS` | string | `*` | 逗号分隔的允许的 CORS 源列表 |

### 配置示例

#### 开发环境

```bash
export API_HOST=0.0.0.0
export API_PORT=8080
export ENABLE_AUTH=false
export CORS_ORIGINS=*
```

#### 生产环境

```bash
export API_HOST=0.0.0.0
export API_PORT=8080
export ENABLE_AUTH=true
export AUTH_TOKENS=<secure-token-1>,<secure-token-2>
export CORS_ORIGINS=https://app.example.com,https://admin.example.com
export API_TIMEOUT=60s
```

#### 启用认证

```bash
export ENABLE_AUTH=true
export AUTH_TOKENS="token1,token2,token3"
```

客户端请求时需要包含令牌：

```bash
curl -H "Authorization: Bearer token1" \
  http://localhost:8080/api/v1/repositories
```

#### 限制 CORS 源

```bash
export CORS_ORIGINS="http://localhost:3000,https://app.example.com"
```

## 索引器配置

### 基本配置

| 变量 | 类型 | 默认值 | 说明 |
|----------|------|---------|-------------|
| `INDEXER_BATCH_SIZE` | int | `100` | 每批处理的实体数量 |
| `INDEXER_WORKER_COUNT` | int | `4` | 并行处理的工作线程数 |
| `INDEXER_SKIP_VECTORS` | bool | `false` | 跳过向量嵌入生成 |
| `INDEXER_INCREMENTAL` | bool | `false` | 启用增量索引（只处理变更文件） |
| `INDEXER_USE_TRANSACTIONS` | bool | `true` | 使用数据库事务进行原子操作 |
| `INDEXER_GRAPH_NAME` | string | `code_graph` | Apache AGE 图名称 |

### 配置示例

#### 大型代码库（高性能）

```bash
export INDEXER_BATCH_SIZE=200
export INDEXER_WORKER_COUNT=8
export INDEXER_USE_TRANSACTIONS=true
```

#### 快速索引（跳过向量）

```bash
export INDEXER_SKIP_VECTORS=true
export INDEXER_BATCH_SIZE=200
export INDEXER_WORKER_COUNT=8
```

#### 增量更新

```bash
export INDEXER_INCREMENTAL=true
export INDEXER_BATCH_SIZE=50
```

#### 低资源环境

```bash
export INDEXER_BATCH_SIZE=25
export INDEXER_WORKER_COUNT=2
export INDEXER_SKIP_VECTORS=true
```

#### 禁用事务（提高性能）

```bash
export INDEXER_USE_TRANSACTIONS=false
export INDEXER_BATCH_SIZE=500
```

**注意**：禁用事务可能导致部分失败时数据不一致。

## 向量模型配置

### 基本配置

| 变量 | 类型 | 默认值 | 说明 |
|----------|------|---------|-------------|
| `EMBEDDING_BACKEND` | string | `openai` | 后端类型 (`openai` 或 `local`) |
| `EMBEDDING_API_ENDPOINT` | string | `http://localhost:1234/v1/embeddings` | API 端点 URL |
| `EMBEDDING_API_KEY` | string | `` | API 认证密钥（本地可选） |
| `EMBEDDING_MODEL` | string | `text-embedding-qwen3-embedding-0.6b` | 模型名称 |
| `EMBEDDING_DIMENSIONS` | int | `768` | 预期嵌入维度 |

### 高级配置

| 变量 | 类型 | 默认值 | 说明 |
|----------|------|---------|-------------|
| `EMBEDDING_BATCH_SIZE` | int | `50` | 每次 API 调用嵌入的文本数量 |
| `EMBEDDING_MAX_REQUESTS_PER_SECOND` | int | `10` | API 请求速率限制 |
| `EMBEDDING_MAX_RETRIES` | int | `3` | 失败请求的最大重试次数 |
| `EMBEDDING_BASE_RETRY_DELAY` | duration | `100ms` | 初始重试延迟（指数退避） |
| `EMBEDDING_MAX_RETRY_DELAY` | duration | `5s` | 最大重试延迟 |
| `EMBEDDING_TIMEOUT` | duration | `30s` | HTTP 请求超时 |

### 支持的模型

| 模型 | 维度 | 后端 |
|-------|------------|---------|
| `text-embedding-qwen3-embedding-0.6b` | 768 | Local/OpenAI-compatible |
| `nomic-embed-text` | 768 | Local/OpenAI-compatible |
| `text-embedding-3-small` | 1536 | OpenAI |
| `text-embedding-3-large` | 3072 | OpenAI |

### 配置示例

#### OpenAI API

```bash
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=https://api.openai.com/v1/embeddings
export EMBEDDING_API_KEY=sk-...
export EMBEDDING_MODEL=text-embedding-3-small
export EMBEDDING_DIMENSIONS=1536
export EMBEDDING_BATCH_SIZE=100
export EMBEDDING_MAX_REQUESTS_PER_SECOND=50
```

#### 本地模型（LM Studio / vLLM）

```bash
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
export EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
export EMBEDDING_DIMENSIONS=768
export EMBEDDING_BATCH_SIZE=50
```

#### Azure OpenAI

```bash
export EMBEDDING_BACKEND=openai
export EMBEDDING_API_ENDPOINT=https://your-resource.openai.azure.com/openai/deployments/your-deployment/embeddings?api-version=2023-05-15
export EMBEDDING_API_KEY=your-azure-key
export EMBEDDING_MODEL=text-embedding-ada-002
export EMBEDDING_DIMENSIONS=1536
```

#### 速率限制配置

避免触发 API 速率限制：

```bash
export EMBEDDING_MAX_REQUESTS_PER_SECOND=5
export EMBEDDING_BATCH_SIZE=25
export EMBEDDING_MAX_RETRIES=5
export EMBEDDING_BASE_RETRY_DELAY=200ms
export EMBEDDING_MAX_RETRY_DELAY=10s
```

### 向量维度配置

**重要**：向量维度必须与模型匹配，并在初始化数据库前设置。

#### 新数据库

```bash
# 在 .env 中设置维度
echo "EMBEDDING_DIMENSIONS=1536" >> .env

# 初始化数据库
make docker-up
make init-db
```

#### 已有数据库

修改向量维度（需要重建向量表）：

```bash
make alter-vector-dimension VECTOR_DIM=1536
```

**警告**：这将删除所有现有向量数据！

## 安全配置

### 认证

#### 启用 API 认证

```bash
export ENABLE_AUTH=true
export AUTH_TOKENS="secure-token-1,secure-token-2,secure-token-3"
```

#### 生成安全令牌

```bash
# 使用 openssl 生成随机令牌
openssl rand -hex 32

# 或使用 uuidgen
uuidgen
```

### SSL/TLS

#### 数据库 SSL

```bash
export DB_SSLMODE=require  # 要求 SSL
# 或
export DB_SSLMODE=verify-full  # 验证证书
```

#### API HTTPS

在生产环境中，使用反向代理（nginx/Caddy）处理 HTTPS：

```nginx
server {
    listen 443 ssl;
    server_name api.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### CORS 安全

#### 开发环境

```bash
export CORS_ORIGINS=*  # 允许所有源
```

#### 生产环境

```bash
export CORS_ORIGINS=https://app.example.com,https://admin.example.com
```

### 密码安全

#### 更改默认密码

```bash
# 生成强密码
openssl rand -base64 32

# 设置数据库密码
export DB_PASSWORD="your-strong-password-here"
```

#### 使用密钥管理

生产环境建议使用密钥管理服务：

- **HashiCorp Vault**
- **AWS Secrets Manager**
- **Azure Key Vault**
- **Google Secret Manager**

## 配置示例

### 开发环境

```bash
# .env.development
# 数据库（使用 Docker Compose 默认值）
DB_HOST=localhost
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=codeatlas
DB_NAME=codeatlas
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=2

# API 服务器
API_HOST=0.0.0.0
API_PORT=8080
ENABLE_AUTH=false
CORS_ORIGINS=*

# 索引器（快速，无向量）
INDEXER_BATCH_SIZE=50
INDEXER_WORKER_COUNT=4
INDEXER_SKIP_VECTORS=true

# 向量模型（如果需要）
EMBEDDING_BACKEND=openai
EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
EMBEDDING_DIMENSIONS=768
```

### 生产环境

```bash
# .env.production
# 数据库（生产 PostgreSQL）
DB_HOST=db.production.example.com
DB_PORT=5432
DB_USER=codeatlas_prod
DB_PASSWORD=<secure-password>
DB_NAME=codeatlas_prod
DB_SSLMODE=require
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=10m

# API 服务器（带认证）
API_HOST=0.0.0.0
API_PORT=8080
ENABLE_AUTH=true
AUTH_TOKENS=<secure-token-1>,<secure-token-2>
CORS_ORIGINS=https://app.example.com,https://admin.example.com
API_TIMEOUT=60s

# 索引器（高性能）
INDEXER_BATCH_SIZE=200
INDEXER_WORKER_COUNT=8
INDEXER_USE_TRANSACTIONS=true
INDEXER_INCREMENTAL=true

# 向量模型（OpenAI）
EMBEDDING_BACKEND=openai
EMBEDDING_API_ENDPOINT=https://api.openai.com/v1/embeddings
EMBEDDING_API_KEY=<openai-api-key>
EMBEDDING_MODEL=text-embedding-3-small
EMBEDDING_DIMENSIONS=1536
EMBEDDING_BATCH_SIZE=100
EMBEDDING_MAX_REQUESTS_PER_SECOND=50
```

### 高吞吐量索引

```bash
# .env.high-throughput
# 数据库（高连接池）
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=20
DB_CONN_MAX_LIFETIME=15m

# 索引器（最大并行）
INDEXER_BATCH_SIZE=500
INDEXER_WORKER_COUNT=16
INDEXER_SKIP_VECTORS=true  # 稍后生成向量
INDEXER_USE_TRANSACTIONS=false  # 提高性能

# 增量处理
INDEXER_INCREMENTAL=true
```

### 资源受限环境

```bash
# .env.low-resource
# 数据库（最小池）
DB_MAX_OPEN_CONNS=5
DB_MAX_IDLE_CONNS=2
DB_CONN_MAX_LIFETIME=3m

# 索引器（低资源使用）
INDEXER_BATCH_SIZE=25
INDEXER_WORKER_COUNT=2
INDEXER_SKIP_VECTORS=true

# 向量模型（如果需要，使用本地模型）
EMBEDDING_BACKEND=openai
EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
EMBEDDING_BATCH_SIZE=10
EMBEDDING_MAX_REQUESTS_PER_SECOND=2
```

## 配置验证

### 验证错误

配置系统在启动时验证所有设置。常见验证错误：

#### 数据库验证错误

- `database host cannot be empty` - 设置 `DB_HOST`
- `database port must be between 1 and 65535` - 检查 `DB_PORT`
- `database max idle connections cannot exceed max open connections` - 调整 `DB_MAX_IDLE_CONNS`

#### API 验证错误

- `API port must be between 1 and 65535` - 检查 `API_PORT`
- `authentication is enabled but no auth tokens are configured` - 当 `ENABLE_AUTH=true` 时设置 `AUTH_TOKENS`

#### 索引器验证错误

- `indexer batch size must be at least 1` - 检查 `INDEXER_BATCH_SIZE`
- `indexer worker count must be at least 1` - 检查 `INDEXER_WORKER_COUNT`
- `indexer graph name cannot be empty` - 设置 `INDEXER_GRAPH_NAME`

#### 向量模型验证错误

- `embedder backend must be 'openai' or 'local'` - 检查 `EMBEDDING_BACKEND`
- `embedder API endpoint cannot be empty` - 设置 `EMBEDDING_API_ENDPOINT`
- `embedder dimensions must be at least 1` - 检查 `EMBEDDING_DIMENSIONS`

### 测试配置

```bash
# 测试数据库连接
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1;"

# 测试 API 服务器
curl http://$API_HOST:$API_PORT/health

# 测试向量模型 API
curl $EMBEDDING_API_ENDPOINT \
  -H "Authorization: Bearer $EMBEDDING_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"input": "test", "model": "'$EMBEDDING_MODEL'"}'
```

## 最佳实践

### 1. 使用环境特定配置

为不同环境维护单独的 `.env` 文件：

```bash
.env.development
.env.staging
.env.production
```

加载特定环境：

```bash
# 开发
export $(cat .env.development | xargs)
make run-api

# 生产
export $(cat .env.production | xargs)
make run-api
```

### 2. 保护敏感值

```bash
# 永远不要提交密码或 API 密钥到版本控制
echo ".env*" >> .gitignore
echo "*.key" >> .gitignore

# 设置限制性权限
chmod 600 .env.production
```

### 3. 从保守开始

从默认值开始，根据实际使用情况调整：

```bash
# 开始时使用默认值
make run-api

# 监控资源使用
# 根据需要调整
```

### 4. 监控资源使用

```bash
# 监控数据库连接
SELECT count(*) FROM pg_stat_activity;

# 监控 API 性能
# 使用 Prometheus/Grafana

# 监控向量 API 使用
# 检查 API 提供商的仪表板
```

### 5. 记录覆盖

保留非默认配置值及其原因的记录：

```bash
# config-notes.md
## 生产配置覆盖

- `DB_MAX_OPEN_CONNS=50` - 增加以处理高流量（2025-11-06）
- `INDEXER_BATCH_SIZE=200` - 优化大型仓库索引（2025-11-06）
- `EMBEDDING_MAX_REQUESTS_PER_SECOND=50` - 匹配 OpenAI 速率限制（2025-11-06）
```

## 故障排除

### 连接池耗尽

如果看到 "too many connections" 错误：

```bash
export DB_MAX_OPEN_CONNS=50
export DB_MAX_IDLE_CONNS=10
```

### 索引慢

提高索引性能：

```bash
export INDEXER_BATCH_SIZE=200
export INDEXER_WORKER_COUNT=8
export INDEXER_SKIP_VECTORS=true
```

### 向量 API 速率限制

如果触发速率限制：

```bash
export EMBEDDING_MAX_REQUESTS_PER_SECOND=5
export EMBEDDING_BATCH_SIZE=25
export EMBEDDING_MAX_RETRIES=5
```

### 内存问题

减少内存使用：

```bash
export INDEXER_BATCH_SIZE=50
export INDEXER_WORKER_COUNT=2
export DB_MAX_OPEN_CONNS=10
```

## 相关文档

- [快速开始指南](../getting-started/quick-start.md)
- [部署指南](../deployment/README.md)
- [API 文档](../user-guide/api/README.md)
- [故障排除](../troubleshooting/README.md)

---

**最后更新**: 2025-11-06  
**维护者**: CodeAtlas Team
