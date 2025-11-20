# 故障排除

> 常见问题和解决方案

## 数据库问题

### 连接失败

**症状**：
```
Error: failed to connect to database: connection refused
```

**解决方案**：

```bash
# 1. 检查数据库是否运行
docker-compose ps postgres

# 2. 检查端口是否开放
lsof -i :5432

# 3. 测试连接
psql -h localhost -p 5432 -U codeatlas -d codeatlas

# 4. 检查环境变量
echo $DB_HOST $DB_PORT $DB_USER
```

### 向量维度不匹配

**症状**：
```
Error: vector dimension mismatch: expected 1536, got 768
```

**解决方案**：

```bash
# 检查当前维度
psql -U codeatlas -d codeatlas -c "
  SELECT atttypmod FROM pg_attribute 
  WHERE attrelid = 'vectors'::regclass AND attname = 'embedding';
"

# 修改维度（会清空数据）
make alter-vector-dimension VECTOR_DIM=1536

# 或手动修改
psql -U codeatlas -d codeatlas -c "
  TRUNCATE TABLE vectors;
  ALTER TABLE vectors ALTER COLUMN embedding TYPE vector(1536);
"
```

### 扩展未安装

**症状**：
```
Error: extension "vector" does not exist
```

**解决方案**：

```bash
# 安装 pgvector
psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS vector;"

# 安装 AGE
psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS age;"

# 或重新初始化数据库
make docker-clean
make docker-up
```

## API 问题

### 端口被占用

**症状**：
```
Error: bind: address already in use
```

**解决方案**：

```bash
# 查找占用端口的进程
lsof -i :8080

# 杀死进程
kill -9 <PID>

# 或使用不同端口
export API_PORT=8081
make run-api
```

### 认证失败

**症状**：
```
401 Unauthorized
```

**解决方案**：

```bash
# 检查 API key
echo $API_KEY

# 设置 API key
export API_KEY=your-secret-key

# 测试认证
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/search?q=main
```

### CORS 错误

**症状**：
```
Access to fetch at 'http://localhost:8080' from origin 'http://localhost:3000' 
has been blocked by CORS policy
```

**解决方案**：

```bash
# 配置 CORS
export CORS_ALLOWED_ORIGINS=http://localhost:3000
make run-api

# 或在 docker-compose.yml 中配置
environment:
  - CORS_ALLOWED_ORIGINS=http://localhost:3000,https://app.example.com
```

## CLI 问题

### 解析失败

**症状**：
```
Error: failed to parse file: syntax error
```

**解决方案**：

```bash
# 使用 verbose 模式查看详细错误
./bin/cli parse --file problematic.go --verbose

# 检查文件编码
file problematic.go

# 检查语法错误
go build problematic.go  # 对于 Go 文件
```

### 内存不足

**症状**：
```
Error: runtime: out of memory
```

**解决方案**：

```bash
# 减少并发数
./bin/cli parse --path . --workers 2

# 分批处理
./bin/cli parse --path ./src --output src.json
./bin/cli parse --path ./lib --output lib.json

# 排除大文件
./bin/cli parse --path . \
  --ignore-pattern "vendor/*" \
  --ignore-pattern "node_modules/*"
```

### 找不到文件

**症状**：
```
Error: no files found to parse
```

**解决方案**：

```bash
# 检查路径
ls -la /path/to/repo

# 使用 verbose 查看被忽略的文件
./bin/cli parse --path . --verbose

# 禁用忽略规则
./bin/cli parse --path . --no-ignore

# 检查语言过滤
./bin/cli parse --path . --language go --verbose
```

## 索引问题

### 索引失败

**症状**：
```
Error: failed to index symbols: database error
```

**解决方案**：

```bash
# 检查数据库连接
psql -U codeatlas -d codeatlas -c "SELECT 1;"

# 检查 JSON 格式
jq . result.json

# 使用小批量
./bin/cli index -i result.json -r myproject --batch-size 50

# 查看详细日志
./bin/cli index -i result.json -r myproject --verbose
```

### 重复符号

**症状**：
```
Error: duplicate key value violates unique constraint
```

**解决方案**：

```bash
# 删除旧索引
psql -U codeatlas -d codeatlas -c "
  DELETE FROM symbols WHERE repo_id = 'xxx';
"

# 或删除整个仓库
curl -X DELETE http://localhost:8080/api/v1/repositories/xxx

# 重新索引
./bin/cli index -i result.json -r myproject
```

## 向量模型问题

### API Key 无效

**症状**：
```
Error: invalid API key
```

**解决方案**：

```bash
# 检查 API key
echo $OPENAI_API_KEY

# 测试 API key
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  https://api.openai.com/v1/models

# 设置新的 API key
export OPENAI_API_KEY=sk-xxx
```

### 速率限制

**症状**：
```
Error: rate limit exceeded
```

**解决方案**：

```bash
# 减少并发
export INDEXER_WORKERS=2

# 增加重试延迟
export INDEXER_RETRY_DELAY=5s

# 使用本地模型
export VECTOR_PROVIDER=local
export VECTOR_API_URL=http://localhost:8000/v1
```

### 模型不可用

**症状**：
```
Error: model not found
```

**解决方案**：

```bash
# 检查可用模型
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  https://api.openai.com/v1/models

# 使用正确的模型名
export OPENAI_MODEL=text-embedding-3-small

# 检查向量维度
export VECTOR_DIMENSION=1536
```

## 性能问题

### 查询慢

**解决方案**：

```bash
# 1. 检查索引
psql -U codeatlas -d codeatlas -c "
  SELECT schemaname, tablename, indexname 
  FROM pg_indexes 
  WHERE tablename IN ('symbols', 'vectors');
"

# 2. 分析查询
psql -U codeatlas -d codeatlas -c "
  EXPLAIN ANALYZE 
  SELECT * FROM symbols WHERE name = 'main';
"

# 3. 启用缓存
export CACHE_ENABLED=true
export CACHE_TTL=1h

# 4. 限制结果数量
curl "http://localhost:8080/api/v1/search?q=main&limit=10"
```

### 内存占用高

**解决方案**：

```bash
# 1. 减少连接池
export DB_MAX_CONNECTIONS=10
export DB_MAX_IDLE_CONNECTIONS=2

# 2. 减少批处理大小
export INDEXER_BATCH_SIZE=50

# 3. 清理缓存
psql -U codeatlas -d codeatlas -c "VACUUM FULL;"

# 4. 重启服务
docker-compose restart api
```

### 磁盘空间不足

**解决方案**：

```bash
# 1. 检查磁盘使用
df -h
du -sh docker/pgdata

# 2. 清理旧数据
psql -U codeatlas -d codeatlas -c "
  DELETE FROM symbols WHERE created_at < NOW() - INTERVAL '30 days';
"

# 3. 清理 Docker
docker system prune -a

# 4. 压缩数据库
psql -U codeatlas -d codeatlas -c "VACUUM FULL;"
```

## DevContainer 问题

### 构建失败

**症状**：
```
Error: failed to build devcontainer
```

**解决方案**：

```bash
# 清理 Docker 缓存
docker system prune -a

# 重新构建
make devcontainer-rebuild

# 检查 Docker 资源
docker info
```

### 扩展未安装

**解决方案**：

```bash
# 重新打开容器
# VS Code: Command Palette → "Dev Containers: Rebuild Container"

# 或手动安装扩展
code --install-extension golang.go
```

## 测试问题

### 测试失败

**症状**：
```
FAIL: TestSymbolRepository_Create
```

**解决方案**：

```bash
# 1. 启动测试数据库
make docker-up

# 2. 运行单个测试
go test -v ./pkg/models -run TestSymbolRepository_Create

# 3. 清理测试数据
psql -U codeatlas -d codeatlas -c "
  DELETE FROM symbols WHERE name LIKE 'test_%';
"

# 4. 重新运行所有测试
make test-all
```

### 集成测试超时

**解决方案**：

```bash
# 增加超时
go test ./... -timeout 60s

# 只运行单元测试
make test-unit

# 跳过慢测试
go test -short ./...
```

## 日志和调试

### 启用详细日志

```bash
# API 服务
export LOG_LEVEL=debug
make run-api

# CLI 工具
./bin/cli parse --path . --verbose

# 数据库查询日志
psql -U codeatlas -d codeatlas -c "
  ALTER DATABASE codeatlas SET log_statement = 'all';
"
```

### 查看日志

```bash
# Docker 日志
docker-compose logs -f api
docker-compose logs -f postgres

# 系统日志
tail -f /var/log/codeatlas/api.log

# 查询慢查询
psql -U codeatlas -d codeatlas -c "
  SELECT query, calls, total_time, mean_time
  FROM pg_stat_statements
  ORDER BY total_time DESC
  LIMIT 10;
"
```

## 获取帮助

### 收集诊断信息

```bash
# 系统信息
uname -a
docker --version
go version

# 服务状态
docker-compose ps
curl http://localhost:8080/health

# 数据库状态
psql -U codeatlas -d codeatlas -c "
  SELECT version();
  SELECT * FROM pg_extension;
"

# 配置信息
env | grep -E 'DB_|API_|OPENAI_'
```

### 提交 Issue

包含以下信息：

1. **问题描述**：详细描述问题
2. **重现步骤**：如何重现问题
3. **期望行为**：期望的结果
4. **实际行为**：实际的结果
5. **环境信息**：操作系统、Docker 版本、Go 版本
6. **日志**：相关的错误日志
7. **配置**：相关的配置（隐藏敏感信息）

### 社区支持

- [GitHub Issues](https://github.com/yourtionguo/CodeAtlas/issues)
- [GitHub Discussions](https://github.com/yourtionguo/CodeAtlas/discussions)
- [文档](https://github.com/yourtionguo/CodeAtlas/tree/main/docs)

## 常见错误码

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| DB001 | 数据库连接失败 | 检查数据库配置和网络 |
| DB002 | 向量维度不匹配 | 修改向量维度配置 |
| API001 | 认证失败 | 检查 API key |
| API002 | 速率限制 | 减少请求频率 |
| PARSE001 | 解析失败 | 检查文件语法 |
| PARSE002 | 内存不足 | 减少并发或分批处理 |
| INDEX001 | 索引失败 | 检查数据库连接 |
| VECTOR001 | 向量模型错误 | 检查 API key 和模型名 |

## 预防措施

1. **定期备份**：定期备份数据库
2. **监控资源**：监控 CPU、内存、磁盘使用
3. **日志轮转**：配置日志轮转避免磁盘满
4. **更新依赖**：定期更新依赖包
5. **测试配置**：在测试环境验证配置
6. **文档化**：记录自定义配置和问题解决方案
