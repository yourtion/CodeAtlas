# CodeAtlas 文档中心

> 欢迎来到 CodeAtlas 文档中心，这里是所有文档的导航入口

## 📚 文档导航

### 🚀 新手入门

从这里开始，快速上手 CodeAtlas。

- **[快速开始指南](getting-started/quick-start.md)** - 5 分钟快速上手
  - 三种启动方式（DevContainer、Docker、本地）
  - 第一次使用教程
  - 常用命令参考

### 📖 用户指南

日常使用 CodeAtlas 的完整指南。

#### CLI 工具

- **[CLI 工具完整指南](user-guide/cli/README.md)** - 命令行工具使用手册
  - Parse 命令 - 代码解析
  - Index 命令 - 代码索引
  - 环境变量配置
  - 性能优化和故障排除

#### API 服务

- **[API 完整指南](user-guide/api/README.md)** - HTTP API 使用手册
  - 端点参考（仓库、索引、搜索、关系查询）
  - 认证和错误处理
  - 集成示例（JavaScript、Python、cURL）
  - 搜索和关系查询最佳实践

### ⚙️ 配置

配置 CodeAtlas 以满足你的需求。

- **[配置完整指南](configuration/README.md)** - 所有配置选项说明
  - 数据库配置（连接、连接池）
  - API 服务器配置（认证、CORS）
  - 索引器配置（批处理、并发）
  - 向量模型配置（OpenAI、本地模型）
  - 安全配置（SSL、认证、密码）
  - 多环境配置示例

### 🚢 部署

将 CodeAtlas 部署到生产环境。

- **[部署完整指南](deployment/README.md)** - 生产环境部署手册
  - Docker 部署（推荐用于中小规模）
  - Systemd 部署（推荐用于大规模生产）
  - 数据库迁移
  - 生产环境最佳实践
  - 监控、备份和故障排除

### 💻 开发

参与 CodeAtlas 开发的指南。

- **[DevContainer 开发指南](development/devcontainer.md)** - 开箱即用的开发环境
  - VS Code 和 GitHub Codespaces 支持
  - 预置测试数据
  - 调试和性能优化
  - 自定义配置

- **[测试完整指南](development/testing.md)** - 测试和覆盖率
  - 单元测试和集成测试
  - 测试覆盖率工具
  - CI/CD 集成
  - 最佳实践和故障排除

### 📋 参考

技术参考文档。

- **[数据库 Schema](reference/schema.md)** - 数据库结构说明
- **[HTTP 请求示例](../example.http)** - 可直接使用的 API 请求示例

### 🔧 故障排除

遇到问题？这里有解决方案。

- **[CLI 故障排除](cli/parse-troubleshooting.md)** - CLI 工具常见问题
- **[索引器故障排除](indexer/troubleshooting.md)** - 索引器常见问题

## 🔍 快速查找

### 按任务查找

#### 我想开始使用 CodeAtlas
→ [快速开始指南](getting-started/quick-start.md)

#### 我想解析代码
→ [CLI 工具指南 - Parse 命令](user-guide/cli/README.md#parse-命令)

#### 我想索引代码到知识图谱
→ [CLI 工具指南 - Index 命令](user-guide/cli/README.md#index-命令)

#### 我想搜索代码
→ [API 指南 - 语义搜索](user-guide/api/README.md#语义搜索)

#### 我想查询代码关系
→ [API 指南 - 关系查询](user-guide/api/README.md#关系查询)

#### 我想配置 CodeAtlas
→ [配置指南](configuration/README.md)

#### 我想部署到生产环境
→ [部署指南](deployment/README.md)

#### 我想设置开发环境
→ [DevContainer 开发指南](development/devcontainer.md)

#### 我想运行测试
→ [测试指南](development/testing.md)

### 按角色查找

#### 开发者
- [快速开始](getting-started/quick-start.md)
- [DevContainer 开发指南](development/devcontainer.md)
- [测试指南](development/testing.md)
- [CLI 工具指南](user-guide/cli/README.md)

#### 运维人员
- [部署指南](deployment/README.md)
- [配置指南](configuration/README.md)
- [API 指南](user-guide/api/README.md)

#### 用户
- [快速开始](getting-started/quick-start.md)
- [CLI 工具指南](user-guide/cli/README.md)
- [API 指南](user-guide/api/README.md)

## 📊 文档结构

```
docs/
├── README.md                       # 本文件 - 文档导航中心
├── getting-started/                # 新手入门
│   └── quick-start.md             # 快速开始指南
├── user-guide/                     # 用户指南
│   ├── cli/                       # CLI 工具
│   │   └── README.md              # CLI 完整指南
│   └── api/                       # API 服务
│       └── README.md              # API 完整指南
├── configuration/                  # 配置指南
│   └── README.md                  # 配置完整指南
├── deployment/                     # 部署指南
│   └── README.md                  # 部署完整指南
├── development/                    # 开发指南
│   ├── devcontainer.md            # DevContainer 指南
│   └── testing.md                 # 测试指南
├── reference/                      # 技术参考
│   └── schema.md                  # 数据库 Schema
├── cli/                           # CLI 详细文档（保留）
├── indexer/                       # 索引器详细文档（保留）
├── testing/                       # 测试详细文档（保留）
└── examples/                      # 示例文件
```

## 🆘 获取帮助

### 文档问题

如果你在文档中发现错误或有改进建议：

1. 查看 [GitHub Issues](https://github.com/yourtionguo/CodeAtlas/issues)
2. 创建新 Issue 并标记为 `documentation`
3. 或直接提交 Pull Request

### 技术问题

如果你遇到技术问题：

1. 先查看相关文档的"故障排除"部分
2. 搜索 [GitHub Issues](https://github.com/yourtionguo/CodeAtlas/issues)
3. 如果没有找到解决方案，创建新 Issue

### 功能请求

如果你有功能建议：

1. 查看 [GitHub Discussions](https://github.com/yourtionguo/CodeAtlas/discussions)
2. 在 "Ideas" 分类下创建新讨论
3. 或创建 Issue 并标记为 `enhancement`

## 🤝 贡献

想要改进文档？

1. 阅读 [贡献指南](../CONTRIBUTING.md)
2. Fork 仓库
3. 创建分支
4. 提交 Pull Request

### 文档编写规范

- 使用清晰、简洁的语言
- 提供实际可运行的示例
- 包含故障排除部分
- 保持格式一致
- 更新相关链接

## 📝 文档版本

- **当前版本**: 1.0.0
- **最后更新**: 2025-11-06
- **维护者**: CodeAtlas Team

## 📄 许可证

文档采用 [MIT License](../LICENSE)

---

**提示**：使用 `Ctrl+F` (Windows/Linux) 或 `Cmd+F` (macOS) 在本页面快速搜索关键词。
