# CodeAtlas DevContainer 快速参考

> 开箱即用的完整开发环境

## 快速开始

### VS Code
1. 安装 [Dev Containers 扩展](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. 打开项目，点击 "Reopen in Container"
3. 等待构建完成（首次约 3-5 分钟）

### GitHub Codespaces
- 点击 "Code" → "Codespaces" → "Create codespace"

### 命令行
```bash
make devcontainer-up
```

## 包含内容

- **Go 1.25** + 开发工具
- **Node.js 20** + pnpm
- **PostgreSQL 17** + pgvector + AGE
- **预置测试数据**

## 常用命令

```bash
# 启动 API
make run-api

# 启动前端
cd web && pnpm dev

# 运行测试
make test

# 访问数据库
psql -h db -U codeatlas -d codeatlas
```

## 端口

- `8080`: API Server
- `3000`: Frontend
- `5432`: PostgreSQL

## 完整文档

详细使用指南请查看：**[DevContainer 完整指南](../docs/development/devcontainer.md)**

## 故障排除

```bash
# 检查数据库
pg_isready -h db -U codeatlas -d codeatlas

# 重建容器
# Command Palette → Dev Containers: Rebuild Container

# 查看日志
docker-compose -f .devcontainer/docker-compose.yml logs
```
