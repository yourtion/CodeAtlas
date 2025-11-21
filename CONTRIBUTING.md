# 贡献指南

感谢你对 CodeAtlas 的关注！我们欢迎各种形式的贡献。

## 开发环境设置

### 推荐方式：使用 DevContainer

我们强烈推荐使用 DevContainer 进行开发，它提供：
- 统一的开发环境
- 预配置的工具和扩展
- 预置的测试数据
- 开箱即用的体验

详细说明请参考：[DevContainer 开发环境指南](docs/devcontainer-guide.md)

**快速开始：**
1. 安装 [VS Code](https://code.visualstudio.com/) 和 [Dev Containers 扩展](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. 克隆仓库：`git clone https://github.com/yourtionguo/CodeAtlas.git`
3. 在 VS Code 中打开项目
4. 点击 "Reopen in Container"
5. 等待容器构建完成

### 传统方式

如果你不想使用 DevContainer，请确保安装：
- Go 1.25+
- Node.js 20+
- PostgreSQL 17
- Docker & Docker Compose

## 开发流程

### 1. 创建分支

```bash
git checkout -b feature/your-feature-name
# 或
git checkout -b fix/your-bug-fix
```

### 2. 编写代码

遵循项目的代码规范：
- Go 代码使用 `gofmt` 格式化
- 运行 `golangci-lint` 检查
- 添加必要的测试
- 保持测试覆盖率在 90% 以上

### 3. 运行测试

```bash
# 快速单元测试（无需数据库）
make test

# 集成测试（需要数据库）
make db                  # 启动数据库
make test-integration    # 运行所有测试

# 生成覆盖率报告
make test-coverage

# 完整验证（清理 + 测试）
make verify
```

### 4. 提交代码

提交信息格式：
```
<type>(<scope>): <subject>

<body>

<footer>
```

类型（type）：
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

示例：
```bash
git commit -m "feat(parser): add support for Rust language"
git commit -m "fix(api): resolve database connection timeout issue"
git commit -m "docs(devcontainer): update setup instructions"
```

### 5. 推送并创建 PR

```bash
git push origin feature/your-feature-name
```

然后在 GitHub 上创建 Pull Request。

## 代码规范

### Go 代码

- 遵循 [Effective Go](https://golang.org/doc/effective_go.html)
- 使用 `gofmt` 格式化代码
- 运行 `golangci-lint` 检查
- 导出的函数和类型必须有文档注释
- 错误处理要明确，不要忽略错误

### 测试

- 每个包都应该有对应的测试文件
- 测试文件命名：`*_test.go`
- 测试函数命名：`TestXxx`
- 使用表驱动测试（table-driven tests）
- 保持测试覆盖率在 90% 以上

示例：
```go
func TestUserValidation(t *testing.T) {
    tests := []struct {
        name    string
        user    User
        wantErr bool
    }{
        {"valid user", User{Name: "Alice", Email: "alice@example.com"}, false},
        {"empty name", User{Name: "", Email: "alice@example.com"}, true},
        {"invalid email", User{Name: "Alice", Email: "invalid"}, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.user.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 前端代码

- 使用 Prettier 格式化
- 遵循 Svelte 最佳实践
- 组件应该小而专注
- 使用 TypeScript 类型注解

## 文档

- 更新相关文档（如果适用）
- API 变更需要更新 API 文档
- 新功能需要添加使用示例
- 重大变更需要更新 README

## Pull Request 检查清单

提交 PR 前，请确保：

- [ ] 代码已格式化（`gofmt`, `prettier`）
- [ ] 通过所有测试（`make test`）
- [ ] 测试覆盖率达标（`make test-coverage`）
- [ ] 通过 lint 检查（`golangci-lint`）
- [ ] 添加了必要的测试
- [ ] 更新了相关文档
- [ ] 提交信息符合规范
- [ ] PR 描述清晰，说明了变更内容和原因

## 报告问题

发现 bug 或有功能建议？请创建 Issue：

1. 搜索现有 Issue，避免重复
2. 使用 Issue 模板
3. 提供详细信息：
   - 问题描述
   - 复现步骤
   - 期望行为
   - 实际行为
   - 环境信息（OS、Go 版本等）
   - 相关日志或截图

## 获取帮助

- 查看 [文档](docs/)
- 查看 [FAQ](docs/FAQ.md)（如果有）
- 在 Issue 中提问
- 加入讨论（Discussions）

## 行为准则

- 尊重他人
- 保持专业
- 接受建设性批评
- 关注对项目最有利的事情

## 许可证

通过贡献代码，你同意你的贡献将在与项目相同的许可证下发布。

---

再次感谢你的贡献！🎉
