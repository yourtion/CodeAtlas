# 配置指南

> CodeAtlas 配置选项完整参考

## 配置方式

### 1. 环境变量（推荐）

创建 `.env` 文件：

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=codeatlas
DB_NAME=codeatlas
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5

# API 服务器
API_PORT=8080
API_HOST=0.0.0.0

# 向量模型
OPENAI_API_KEY=sk-xxx
OPENAI_MODEL=text-embedding-3-small
VECTOR_DIMENSION=1536

# 日志
LOG_LEVEL=info
LOG_FORMAT=json
```

### 2. 配置文件

创建 `configs/config.yaml`：

```yaml
database:
  host: localhost
  port: 5432
  user: codeatlas
  password: codeatlas
  name: codeatlas
  max_connections: 25

api:
  port: 8080
  host: 0.0.0.0
  
vector:
  provider: openai
  model: text-embedding-3-small
  dimension: 1536
```

### 3. 命令行参数

```bash
./bin/api --port 8080 --db-host localhost
```

## 数据库配置

### 连接设置

```bash
# 基本连接
DB_HOST=localhost          # 数据库主机
DB_PORT=5432              # 数据库端口
DB_USER=codeatlas         # 用户名
DB_PASSWORD=codeatlas     # 密码
DB_NAME=codeatlas         # 数据库名

# SSL 连接
DB_SSLMODE=require        # disable, require, verify-ca, verify-full
DB_SSLCERT=/path/to/cert
DB_SSLKEY=/path/to/key
DB_SSLROOTCERT=/path/to/ca
```

### 连接池

```bash
# 连接池大小
DB_MAX_CONNECTIONS=25           # 最大连接数
DB_MAX_IDLE_CONNECTIONS=5       # 最大空闲连接数
DB_CONNECTION_MAX_LIFETIME=1h   # 连接最大生命周期
```

### 向量维度

```bash
# 向量维度（必须与模型匹配）
VECTOR_DIMENSION=1536

# 常用模型和维度
# text-embedding-3-small: 1536
# text-embedding-3-large: 3072
# text-embedding-ada-002: 1536
```

修改向量维度：

```bash
# 方式 1: 使用 Makefile
make alter-vector-dimension VECTOR_DIM=1536

# 方式 2: 手动修改
psql -U codeatlas -d codeatlas -c "
  ALTER TABLE vectors ALTER COLUMN embedding TYPE vector(1536);
"
```

## API 服务器配置

### 基本设置

```bash
# 服务器地址
API_HOST=0.0.0.0          # 监听地址
API_PORT=8080             # 监听端口

# 超时设置
API_READ_TIMEOUT=30s      # 读取超时
API_WRITE_TIMEOUT=30s     # 写入超时
API_IDLE_TIMEOUT=120s     # 空闲超时
```

### 认证

```bash
# API Key 认证
API_KEY=your-secret-key
API_KEY_HEADER=Authorization

# JWT 认证
JWT_SECRET=your-jwt-secret
JWT_EXPIRATION=24h
```

### CORS

```bash
# CORS 设置
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://app.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE
CORS_ALLOWED_HEADERS=Content-Type,Authorization
CORS_MAX_AGE=86400
```

### 速率限制

```bash
# 速率限制
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_REQUESTS_PER_HOUR=1000
```

## 索引器配置

### 批处理

```bash
# 批处理大小
INDEXER_BATCH_SIZE=100          # 每批处理的符号数
INDEXER_MAX_BATCH_SIZE=1000     # 最大批处理大小
```

### 并发

```bash
# 并发设置
INDEXER_WORKERS=4               # 工作线程数
INDEXER_QUEUE_SIZE=1000         # 队列大小
```

### 重试

```bash
# 重试策略
INDEXER_MAX_RETRIES=3           # 最大重试次数
INDEXER_RETRY_DELAY=1s          # 重试延迟
INDEXER_RETRY_BACKOFF=2         # 退避倍数
```

## 向量模型配置

### OpenAI

```bash
# OpenAI 配置
OPENAI_API_KEY=sk-xxx
OPENAI_API_URL=https://api.openai.com/v1
OPENAI_MODEL=text-embedding-3-small
OPENAI_MAX_TOKENS=8191
OPENAI_TIMEOUT=30s
```

### 本地模型

```bash
# 本地 vLLM 服务
VECTOR_PROVIDER=local
VECTOR_API_URL=http://localhost:8000/v1
VECTOR_MODEL=BAAI/bge-large-en-v1.5
VECTOR_DIMENSION=1024
```

### Azure OpenAI

```bash
# Azure OpenAI
AZURE_OPENAI_API_KEY=xxx
AZURE_OPENAI_ENDPOINT=https://xxx.openai.azure.com
AZURE_OPENAI_DEPLOYMENT=text-embedding-ada-002
AZURE_OPENAI_API_VERSION=2023-05-15
```

## 日志配置

### 日志级别

```bash
# 日志级别: debug, info, warn, error
LOG_LEVEL=info

# 开发环境
LOG_LEVEL=debug

# 生产环境
LOG_LEVEL=warn
```

### 日志格式

```bash
# 日志格式: json, text
LOG_FORMAT=json           # 生产环境推荐
LOG_FORMAT=text           # 开发环境推荐
```

### 日志输出

```bash
# 日志输出
LOG_OUTPUT=stdout         # stdout, stderr, file
LOG_FILE=/var/log/codeatlas/api.log
LOG_MAX_SIZE=100          # MB
LOG_MAX_BACKUPS=10
LOG_MAX_AGE=30            # 天
```

## 性能配置

### 缓存

```bash
# 缓存设置
CACHE_ENABLED=true
CACHE_TTL=1h
CACHE_MAX_SIZE=1000       # 最大缓存条目数
```

### 查询优化

```bash
# 查询优化
QUERY_TIMEOUT=30s
QUERY_MAX_RESULTS=1000
VECTOR_SEARCH_LIMIT=100
```

## 安全配置

### TLS/SSL

```bash
# TLS 配置
TLS_ENABLED=true
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem
TLS_MIN_VERSION=1.2
```

### 密码策略

```bash
# 密码要求
PASSWORD_MIN_LENGTH=8
PASSWORD_REQUIRE_UPPERCASE=true
PASSWORD_REQUIRE_LOWERCASE=true
PASSWORD_REQUIRE_NUMBERS=true
PASSWORD_REQUIRE_SPECIAL=true
```

## 多环境配置

### 开发环境

```bash
# .env.development
DB_HOST=localhost
DB_PORT=5432
LOG_LEVEL=debug
LOG_FORMAT=text
API_PORT=8080
CACHE_ENABLED=false
```

### 测试环境

```bash
# .env.test
DB_HOST=localhost
DB_PORT=5433
DB_NAME=codeatlas_test
LOG_LEVEL=warn
API_PORT=8081
```

### 生产环境

```bash
# .env.production
DB_HOST=db.example.com
DB_PORT=5432
DB_SSLMODE=require
LOG_LEVEL=warn
LOG_FORMAT=json
API_PORT=8080
CACHE_ENABLED=true
TLS_ENABLED=true
```

## 配置验证

### 检查配置

```bash
# 验证数据库连接
make test-db-connection

# 验证 API 配置
./bin/api --validate-config

# 验证向量模型
make test-vector-model
```

### 配置示例

完整的生产环境配置示例：

```bash
# .env.production

# 数据库
DB_HOST=postgres.example.com
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=${DB_PASSWORD}  # 从密钥管理系统获取
DB_NAME=codeatlas
DB_SSLMODE=require
DB_MAX_CONNECTIONS=50
DB_MAX_IDLE_CONNECTIONS=10

# API
API_HOST=0.0.0.0
API_PORT=8080
API_READ_TIMEOUT=30s
API_WRITE_TIMEOUT=30s

# 认证
API_KEY=${API_KEY}
JWT_SECRET=${JWT_SECRET}

# CORS
CORS_ALLOWED_ORIGINS=https://app.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE

# 速率限制
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60

# 向量模型
OPENAI_API_KEY=${OPENAI_API_KEY}
OPENAI_MODEL=text-embedding-3-small
VECTOR_DIMENSION=1536

# 日志
LOG_LEVEL=warn
LOG_FORMAT=json
LOG_OUTPUT=file
LOG_FILE=/var/log/codeatlas/api.log

# 性能
CACHE_ENABLED=true
CACHE_TTL=1h
QUERY_TIMEOUT=30s

# 安全
TLS_ENABLED=true
TLS_CERT_FILE=/etc/ssl/certs/codeatlas.crt
TLS_KEY_FILE=/etc/ssl/private/codeatlas.key
```

## 故障排除

### 数据库连接失败

```bash
# 检查连接
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME

# 检查 SSL 配置
psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=require"
```

### 向量维度不匹配

```bash
# 检查当前维度
psql -U codeatlas -d codeatlas -c "
  SELECT atttypmod FROM pg_attribute 
  WHERE attrelid = 'vectors'::regclass AND attname = 'embedding';
"

# 修改维度
make alter-vector-dimension VECTOR_DIM=1536
```

### API 无法访问

```bash
# 检查端口占用
lsof -i :8080

# 检查防火墙
sudo ufw status

# 检查日志
tail -f /var/log/codeatlas/api.log
```

## 最佳实践

1. **使用环境变量**：敏感信息不要硬编码
2. **分离环境配置**：开发、测试、生产使用不同配置
3. **启用 SSL**：生产环境必须使用 SSL
4. **配置日志**：生产环境使用 JSON 格式便于解析
5. **调整连接池**：根据负载调整数据库连接池大小
6. **启用缓存**：生产环境启用缓存提高性能
7. **监控配置**：定期检查配置是否合理

## 下一步

- 查看 [部署指南](deployment.md) 了解生产环境部署
- 查看 [快速开始](quick-start.md) 了解基本使用
- 查看 [故障排除](troubleshooting.md) 解决常见问题
