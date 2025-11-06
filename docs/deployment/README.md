# 部署完整指南

> CodeAtlas 生产环境部署手册

## 目录

- [概述](#概述)
- [部署选项](#部署选项)
- [Docker 部署](#docker-部署)
- [Systemd 部署](#systemd-部署)
- [数据库迁移](#数据库迁移)
- [生产环境最佳实践](#生产环境最佳实践)

## 概述

CodeAtlas 支持两种主要部署方式：

1. **Docker 部署**：容器化部署（推荐用于开发和中小规模生产）
2. **Systemd 部署**：原生二进制部署（推荐用于大规模生产）

### 前置要求

#### 通用要求

- PostgreSQL 17+ 带扩展：
  - pgvector（语义搜索）
  - Apache AGE（图查询）
- Go 1.25+（从源码构建）
- 网络访问嵌入 API（OpenAI 或本地服务器）

#### Docker 部署要求

- Docker 20.10+
- Docker Compose 2.0+
- 4GB+ 内存
- 20GB+ 磁盘空间

#### Systemd 部署要求

- Linux 系统带 systemd
- Root 访问权限
- PostgreSQL 已安装并运行
- 8GB+ 内存（推荐）
- 50GB+ 磁盘空间

## 部署选项

### 快速对比

| 特性 | Docker | Systemd |
|------|--------|---------|
| 设置难度 | 简单 | 中等 |
| 性能 | 良好 | 优秀 |
| 资源使用 | 中等 | 低 |
| 隔离性 | 优秀 | 良好 |
| 适用场景 | 开发、测试、中小规模 | 大规模生产 |

## Docker 部署

Docker 部署是最简单的入门方式，包含预配置所有必需扩展的 PostgreSQL。

### 快速开始

```bash
# 1. 进入部署目录
cd deployments

# 2. 复制环境配置
cp .env.example .env

# 3. 编辑配置
nano .env

# 4. 运行部署脚本
./scripts/deploy.sh docker
```

### 手动 Docker 部署

```bash
# 1. 构建镜像
docker-compose -f deployments/docker-compose.prod.yml build

# 2. 启动服务
docker-compose -f deployments/docker-compose.prod.yml up -d

# 3. 检查状态
docker-compose -f deployments/docker-compose.prod.yml ps

# 4. 查看日志
docker-compose -f deployments/docker-compose.prod.yml logs -f api
```

### Docker 管理命令

```bash
# 停止服务
docker-compose -f deployments/docker-compose.prod.yml down

# 重启服务
docker-compose -f deployments/docker-compose.prod.yml restart

# 查看 API 日志
docker-compose -f deployments/docker-compose.prod.yml logs -f api

# 查看数据库日志
docker-compose -f deployments/docker-compose.prod.yml logs -f db

# 在 API 容器中执行命令
docker-compose -f deployments/docker-compose.prod.yml exec api /bin/sh

# 访问数据库
docker-compose -f deployments/docker-compose.prod.yml exec db psql -U codeatlas -d codeatlas

# 备份数据库
docker-compose -f deployments/docker-compose.prod.yml exec db \
  pg_dump -U codeatlas codeatlas > backup.sql

# 恢复数据库
docker-compose -f deployments/docker-compose.prod.yml exec -T db \
  psql -U codeatlas codeatlas < backup.sql
```

### Docker Compose 配置

`deployments/docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  db:
    image: codeatlas-postgres:17
    build:
      context: ../docker
      dockerfile: Dockerfile
    environment:
      POSTGRES_USER: ${DB_USER:-codeatlas}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-codeatlas}
      POSTGRES_DB: ${DB_NAME:-codeatlas}
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "${DB_PORT:-5432}:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U codeatlas"]
      interval: 10s
      timeout: 5s
      retries: 5

  api:
    image: codeatlas-api:latest
    build:
      context: ..
      dockerfile: deployments/Dockerfile.api
    environment:
      DB_HOST: db
      DB_PORT: 5432
      DB_USER: ${DB_USER:-codeatlas}
      DB_PASSWORD: ${DB_PASSWORD:-codeatlas}
      DB_NAME: ${DB_NAME:-codeatlas}
      API_PORT: ${API_PORT:-8080}
      ENABLE_AUTH: ${ENABLE_AUTH:-false}
      AUTH_TOKENS: ${AUTH_TOKENS:-}
      CORS_ORIGINS: ${CORS_ORIGINS:-*}
    ports:
      - "${API_PORT:-8080}:8080"
    depends_on:
      db:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres-data:
```

## Systemd 部署

Systemd 部署将 CodeAtlas 作为原生 Linux 服务运行，为生产环境提供更好的性能和资源管理。

### 安装步骤

```bash
# 1. 以 root 身份运行部署脚本
sudo ./scripts/deploy.sh systemd

# 2. 更新环境配置
sudo nano /etc/codeatlas/api.env

# 3. 重启服务
sudo systemctl restart codeatlas-api
```

### 手动 Systemd 部署

```bash
# 1. 创建 codeatlas 用户
sudo useradd -r -s /bin/false -d /opt/codeatlas codeatlas

# 2. 创建目录
sudo mkdir -p /opt/codeatlas/bin
sudo mkdir -p /etc/codeatlas
sudo mkdir -p /var/log/codeatlas

# 3. 构建二进制文件
cd /path/to/codeatlas
go build -o /opt/codeatlas/bin/codeatlas-api cmd/api/main.go

# 4. 复制配置
sudo cp deployments/systemd/api.env.example /etc/codeatlas/api.env
sudo nano /etc/codeatlas/api.env

# 5. 安装 systemd 服务
sudo cp deployments/systemd/codeatlas-api.service /etc/systemd/system/
sudo systemctl daemon-reload

# 6. 设置权限
sudo chown -R codeatlas:codeatlas /opt/codeatlas
sudo chown -R codeatlas:codeatlas /var/log/codeatlas
sudo chmod 600 /etc/codeatlas/api.env
sudo chmod 755 /opt/codeatlas/bin/codeatlas-api

# 7. 启用并启动服务
sudo systemctl enable codeatlas-api
sudo systemctl start codeatlas-api
```

### Systemd 管理命令

```bash
# 检查服务状态
sudo systemctl status codeatlas-api

# 启动服务
sudo systemctl start codeatlas-api

# 停止服务
sudo systemctl stop codeatlas-api

# 重启服务
sudo systemctl restart codeatlas-api

# 查看日志
sudo journalctl -u codeatlas-api -f

# 查看最近日志
sudo journalctl -u codeatlas-api -n 100

# 开机启动
sudo systemctl enable codeatlas-api

# 禁用开机启动
sudo systemctl disable codeatlas-api
```

### Systemd 服务配置

`deployments/systemd/codeatlas-api.service`:

```ini
[Unit]
Description=CodeAtlas API Server
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=codeatlas
Group=codeatlas
WorkingDirectory=/opt/codeatlas
EnvironmentFile=/etc/codeatlas/api.env
ExecStart=/opt/codeatlas/bin/codeatlas-api
Restart=on-failure
RestartSec=5s

# 安全加固
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/codeatlas

# 资源限制
LimitNOFILE=65536
LimitNPROC=4096

# 日志
StandardOutput=journal
StandardError=journal
SyslogIdentifier=codeatlas-api

[Install]
WantedBy=multi-user.target
```

## 数据库迁移

数据库迁移通过 `migrations/` 目录中的 SQL 脚本管理。

### 迁移文件

- `01_init_schema.sql`: 初始数据库 schema，包含所有表和扩展
- `02_performance_indexes.sql`: 向量相似度搜索的性能索引

### 运行迁移

#### Docker 环境

```bash
# 运行所有迁移
./scripts/migrate.sh docker all

# 运行特定迁移
./scripts/migrate.sh docker 01_init_schema
```

#### 直接数据库连接

```bash
# 设置环境变量
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=your_password
export DB_NAME=codeatlas

# 运行所有迁移
./scripts/migrate.sh direct all

# 运行特定迁移
./scripts/migrate.sh direct 01_init_schema
```

#### 手动迁移

```bash
# Docker
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -f /docker-entrypoint-initdb.d/01_init_schema.sql

# 直接
psql -h localhost -U codeatlas -d codeatlas -f deployments/migrations/01_init_schema.sql
```

### 创建新迁移

1. 在 `deployments/migrations/` 中创建新 SQL 文件，格式：`XX_description.sql`
2. 在末尾包含迁移跟踪：

```sql
INSERT INTO schema_migrations (version, description)
VALUES ('XX_description', 'Description of migration')
ON CONFLICT (version) DO NOTHING;
```

3. 在开发数据库上测试迁移
4. 运行迁移脚本应用

## 生产环境最佳实践

### 安全配置

#### 1. 更改默认密码

```bash
# 生成强密码
openssl rand -base64 32

# 设置数据库密码
export DB_PASSWORD="your-strong-password-here"
```

#### 2. 启用 SSL

```bash
# 数据库 SSL
export DB_SSLMODE=require

# API HTTPS（使用反向代理）
# 见下文 nginx 配置
```

#### 3. 限制 CORS

```bash
# 开发环境
export CORS_ORIGINS=*

# 生产环境
export CORS_ORIGINS=https://app.example.com,https://admin.example.com
```

#### 4. 启用认证

```bash
export ENABLE_AUTH=true
export AUTH_TOKENS="$(openssl rand -hex 32),$(openssl rand -hex 32)"
```

### 反向代理配置

#### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name api.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket 支持（如果需要）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # 超时
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name api.example.com;
    return 301 https://$server_name$request_uri;
}
```

#### Caddy

```caddyfile
api.example.com {
    reverse_proxy localhost:8080
    
    # 自动 HTTPS
    tls your-email@example.com
    
    # 日志
    log {
        output file /var/log/caddy/api.log
    }
}
```

### 监控和维护

#### 健康检查

```bash
# 检查 API 健康
curl http://localhost:8080/health

# 检查数据库连接
docker-compose -f deployments/docker-compose.prod.yml exec db \
  pg_isready -U codeatlas -d codeatlas
```

#### 性能监控

```bash
# 查看 API 指标（如果启用）
curl http://localhost:8080/metrics

# 数据库统计
psql -U codeatlas -d codeatlas -c \
  "SELECT * FROM pg_stat_database WHERE datname = 'codeatlas';"

# 活动连接
psql -U codeatlas -d codeatlas -c \
  "SELECT count(*) FROM pg_stat_activity;"
```

#### 备份和恢复

**数据库备份**：

```bash
# Docker
docker-compose -f deployments/docker-compose.prod.yml exec db \
  pg_dump -U codeatlas codeatlas > backup_$(date +%Y%m%d_%H%M%S).sql

# 直接
pg_dump -h localhost -U codeatlas codeatlas > backup_$(date +%Y%m%d_%H%M%S).sql
```

**数据库恢复**：

```bash
# Docker
docker-compose -f deployments/docker-compose.prod.yml exec -T db \
  psql -U codeatlas codeatlas < backup.sql

# 直接
psql -h localhost -U codeatlas codeatlas < backup.sql
```

**自动备份**：

添加到 crontab 进行每日备份：

```bash
# 每天凌晨 2 点备份
0 2 * * * /path/to/backup_script.sh
```

#### 日志轮转

**Docker 日志**：

在 `docker-compose.prod.yml` 中配置：

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "5"
```

**Systemd 日志**：

```bash
# 配置 journald
sudo nano /etc/systemd/journald.conf

# 设置限制
SystemMaxUse=1G
SystemMaxFileSize=100M
```

### 故障排除

#### API 服务器无法启动

```bash
# 检查日志
docker-compose -f deployments/docker-compose.prod.yml logs api
# 或
sudo journalctl -u codeatlas-api -n 100

# 检查数据库连接
psql -h localhost -U codeatlas -d codeatlas -c "SELECT 1;"

# 验证环境变量
docker-compose -f deployments/docker-compose.prod.yml exec api env | grep DB_
```

#### 数据库连接错误

```bash
# 检查 PostgreSQL 是否运行
docker-compose -f deployments/docker-compose.prod.yml ps db
# 或
sudo systemctl status postgresql

# 检查网络连接
docker-compose -f deployments/docker-compose.prod.yml exec api ping db

# 验证凭据
psql -h localhost -U codeatlas -d codeatlas -c "SELECT current_user;"
```

#### 扩展未找到

```bash
# 检查扩展
psql -U codeatlas -d codeatlas -c "SELECT * FROM pg_extension;"

# 安装 pgvector
psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS vector;"

# 安装 AGE
psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS age;"
```

#### 性能问题

```bash
# 检查数据库统计
psql -U codeatlas -d codeatlas -c "SELECT * FROM pg_stat_user_tables;"

# 分析表
psql -U codeatlas -d codeatlas -c "ANALYZE;"

# 检查慢查询
psql -U codeatlas -d codeatlas -c \
  "SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;"
```

### 升级

#### Docker 升级

```bash
# 1. 拉取最新变更
git pull origin main

# 2. 重建镜像
docker-compose -f deployments/docker-compose.prod.yml build

# 3. 停止服务
docker-compose -f deployments/docker-compose.prod.yml down

# 4. 运行迁移
./scripts/migrate.sh docker all

# 5. 启动服务
docker-compose -f deployments/docker-compose.prod.yml up -d
```

#### Systemd 升级

```bash
# 1. 拉取最新变更
git pull origin main

# 2. 构建新二进制文件
go build -o /tmp/codeatlas-api cmd/api/main.go

# 3. 停止服务
sudo systemctl stop codeatlas-api

# 4. 替换二进制文件
sudo mv /tmp/codeatlas-api /opt/codeatlas/bin/codeatlas-api
sudo chmod 755 /opt/codeatlas/bin/codeatlas-api

# 5. 运行迁移
./scripts/migrate.sh direct all

# 6. 启动服务
sudo systemctl start codeatlas-api
```

### 生产检查清单

- [ ] 更改默认数据库密码
- [ ] 为数据库连接启用 SSL
- [ ] 配置特定源的 CORS
- [ ] 启用 API 认证
- [ ] 设置自动备份
- [ ] 配置日志轮转
- [ ] 设置监控和告警
- [ ] 审查和调整资源限制
- [ ] 测试灾难恢复程序
- [ ] 记录自定义配置
- [ ] 设置防火墙规则
- [ ] 配置反向代理（nginx/traefik）
- [ ] 使用有效证书启用 HTTPS
- [ ] 审查安全加固设置

## 相关文档

- [快速开始指南](../getting-started/quick-start.md)
- [配置指南](../configuration/README.md)
- [API 文档](../user-guide/api/README.md)
- [故障排除](../troubleshooting/README.md)

---

**最后更新**: 2025-11-06  
**维护者**: CodeAtlas Team
