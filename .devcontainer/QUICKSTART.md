# DevContainer 快速参考

## 🚀 一键启动

### VS Code
```
Cmd/Ctrl + Shift + P → "Dev Containers: Reopen in Container"
```

### 命令行
```bash
make devcontainer-up
```

## 📋 常用命令

### 构建和运行
```bash
make build              # 构建所有二进制文件
make run-api            # 启动 API 服务器 (端口 8080)
cd web && pnpm dev      # 启动前端 (端口 3000)
```

### 测试
```bash
make test               # 运行所有测试
make test-coverage      # 生成覆盖率报告
./scripts/test_devcontainer.sh  # 验证环境
```

### 数据库
```bash
psql -h db -U codeatlas -d codeatlas  # 连接数据库

# 查看测试数据
SELECT * FROM repositories;
SELECT * FROM files;
SELECT * FROM symbols;
```

## 🔌 端口

| 端口 | 服务 |
|------|------|
| 8080 | API Server |
| 3000 | Frontend Dev Server |
| 5432 | PostgreSQL |

## 🗄️ 数据库连接

```
Host: db
Port: 5432
Database: codeatlas
User: codeatlas
Password: codeatlas
```

## 📦 预置数据

- 3 个示例仓库
- 多个代码文件（Go, Svelte）
- 符号和依赖关系
- Mock 向量嵌入

## 🐛 调试

### API Server
按 `F5` 或使用 "Debug API Server" 配置

### 查看日志
```bash
make devcontainer-logs
```

## 🔧 故障排除

### 数据库未就绪
```bash
pg_isready -h db -U codeatlas -d codeatlas
```

### 重建容器
```bash
make devcontainer-clean
make devcontainer-build
make devcontainer-up
```

### 查看容器状态
```bash
docker-compose -f .devcontainer/docker-compose.yml ps
```

## 📚 更多信息

- [完整指南](README.md)
- [项目文档](../docs/devcontainer-guide.md)
- [贡献指南](../CONTRIBUTING.md)

## 💡 提示

- 首次构建需要 3-5 分钟
- 使用命名卷缓存依赖，加快重建速度
- 数据在容器重启后保持
- 所有 VS Code 扩展已预配置
