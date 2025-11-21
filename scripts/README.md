# Scripts 目录说明

本目录包含 CodeAtlas 项目的辅助脚本和工具。

## 脚本分类

### 🔨 构建和初始化

#### `init_db.go`
数据库初始化工具

**用途：**
- 创建数据库模式（表、扩展、索引）
- 验证数据库连接
- 显示数据库统计信息

**使用方式：**
```bash
# 通过 Makefile（推荐）
make init-db                # 基本初始化
make init-db-stats         # 初始化并显示统计
make init-db-with-index    # 初始化并创建向量索引

# 直接运行
go run scripts/init_db.go
go run scripts/init_db.go -stats
go run scripts/init_db.go -create-vector-index -vector-index-lists 100
```

**参数：**
- `-max-retries`: 最大重试次数（默认 10）
- `-retry-delay`: 重试延迟秒数（默认 2）
- `-create-vector-index`: 创建向量相似度索引
- `-vector-index-lists`: IVFFlat 索引的列表数（默认 100）
- `-stats`: 显示数据库统计信息

#### `alter_vector_dimension.go`
向量维度管理工具

**用途：**
- 更改向量表的维度
- 支持不同的嵌入模型

**使用方式：**
```bash
# 通过 Makefile（推荐）
make alter-vector-dimension VECTOR_DIM=1536
make alter-vector-dimension-force VECTOR_DIM=768

# 直接运行
go run scripts/alter_vector_dimension.go -dimension 1536
go run scripts/alter_vector_dimension.go -dimension 768 -force
EMBEDDING_DIMENSIONS=1536 go run scripts/alter_vector_dimension.go
```

**参数：**
- `-dimension`: 新的向量维度（必需）
- `-force`: 强制更改（清空 vectors 表）
- `-dry-run`: 显示将执行的操作但不实际执行

**常用维度：**
- 768: nomic-embed-text
- 1024: text-embedding-qwen3-embedding-0.6b
- 1536: text-embedding-3-small (OpenAI)
- 3072: text-embedding-3-large (OpenAI)

### 🧪 测试相关

#### `test_runner.sh`
增强的测试运行器

**用途：**
- 彩色输出
- 突出显示失败的测试
- 显示测试统计信息
- 显示通过率

**使用方式：**
```bash
# 通过 Makefile（推荐）
make test-pretty

# 直接运行
bash scripts/test_runner.sh go test ./... -v
```

#### `test_ci.sh`
CI 友好的测试运行器

**用途：**
- 生成 JSON 格式的测试报告
- 提取失败的测试信息
- 适合 CI/CD 管道

**使用方式：**
```bash
# 通过 Makefile（推荐）
make test-ci

# 直接运行
bash scripts/test_ci.sh go test ./... -v
```

**输出：**
- 控制台：格式化的测试摘要
- 文件：`test_report_YYYYMMDD_HHMMSS.json`

#### `verify_test_setup.sh`
完整的测试环境验证

**用途：**
- 验证数据库连接
- 清理旧的测试数据库
- 运行单元测试
- 运行集成测试
- 验证 CLI 测试
- 生成验证报告

**使用方式：**
```bash
# 通过 Makefile（推荐）
make verify-tests

# 直接运行
bash scripts/verify_test_setup.sh
```

**验证步骤：**
1. 检查数据库连接
2. 清理现有测试数据库
3. 构建 CLI 二进制
4. 运行单元测试
5. 运行集成测试
6. 运行 CLI 测试
7. 最终清理

#### `cleanup_test_databases.sh`
清理测试数据库

**用途：**
- 删除所有 `codeatlas_test_*` 数据库
- 释放测试资源

**使用方式：**
```bash
# 通过 Makefile（推荐）
make clean-test-dbs

# 直接运行
bash scripts/cleanup_test_databases.sh
```

**环境变量：**
- `DB_HOST`: 数据库主机（默认 localhost）
- `DB_PORT`: 数据库端口（默认 5432）
- `DB_USER`: 数据库用户（默认 codeatlas）
- `DB_PASSWORD`: 数据库密码（默认 codeatlas）

#### `coverage_report.sh`
生成覆盖率 HTML 报告

**用途：**
- 从现有的 `.out` 文件生成 HTML 报告

**使用方式：**
```bash
# 通过 Makefile（推荐）
make test-coverage-report

# 直接运行
bash scripts/coverage_report.sh
```

### 🐳 DevContainer 相关

#### `init_devcontainer.sh`
DevContainer 初始化脚本

**用途：**
- 等待数据库就绪
- 检查数据库初始化状态
- 构建项目
- 显示快速开始指南

**使用方式：**
- 自动在 DevContainer 启动时运行
- 不需要手动执行

#### `test_devcontainer.sh`
DevContainer 环境测试

**用途：**
- 验证 Go 安装
- 验证 Node.js 和 pnpm
- 验证数据库连接
- 验证项目构建

**使用方式：**
```bash
# 在 DevContainer 中运行
bash scripts/test_devcontainer.sh
```

### 🔧 开发工具

#### `test_schema.sh`
数据库模式测试

**用途：**
- 验证数据库扩展
- 验证表结构
- 验证 AGE 图谱
- 显示表统计

**使用方式：**
```bash
bash scripts/test_schema.sh
```

#### `profile_parse.sh`
解析性能分析

**用途：**
- CPU 性能分析
- 内存分析
- 生成性能报告

**使用方式：**
```bash
bash scripts/profile_parse.sh <repo-path> [workers]

# 示例
bash scripts/profile_parse.sh tests/fixtures/test-repo 4
```

**输出：**
- `profile_results/cpu.prof`: CPU 分析文件
- `profile_results/mem.prof`: 内存分析文件
- `profile_results/output.json`: 解析结果

**查看分析：**
```bash
go tool pprof -http=:8080 profile_results/cpu.prof
go tool pprof -http=:8080 profile_results/mem.prof
```

#### `pre-commit-hook.sh`
Git 预提交钩子

**用途：**
- 自动格式化代码
- 运行 go vet
- 运行测试
- 可选：检查覆盖率

**安装：**
```bash
ln -s ../../scripts/pre-commit-hook.sh .git/hooks/pre-commit
```

**功能：**
1. 检查是否有 Go 文件修改
2. 运行 `gofmt` 格式化
3. 运行 `go vet` 检查
4. 运行单元测试

### 📄 SQL 脚本

#### `alter_vector_dimension.sql`
向量维度更改 SQL 模板

**用途：**
- 手动更改向量维度的 SQL 参考

#### `seed_data.sql`
测试数据种子

**用途：**
- 为开发和测试提供示例数据

## 脚本依赖关系

```
Makefile
├── init_db.go
├── alter_vector_dimension.go
├── test_runner.sh
├── test_ci.sh
├── verify_test_setup.sh
│   ├── cleanup_test_databases.sh
│   └── test_runner.sh
├── cleanup_test_databases.sh
└── coverage_report.sh

DevContainer
├── init_devcontainer.sh
└── test_devcontainer.sh
```

## 最佳实践

### 1. 使用 Makefile 命令
优先使用 Makefile 命令而不是直接运行脚本：

```bash
# 推荐
make test-pretty

# 不推荐
bash scripts/test_runner.sh go test ./... -v
```

### 2. 定期清理测试数据库
```bash
# 每周或在测试失败后
make clean-test-dbs
```

### 3. 使用 verify-tests 进行完整验证
```bash
# 在重大更改后
make verify-tests
```

### 4. 性能分析
```bash
# 优化解析性能时
bash scripts/profile_parse.sh path/to/large/repo
```

### 5. 安装预提交钩子
```bash
# 一次性设置
ln -s ../../scripts/pre-commit-hook.sh .git/hooks/pre-commit
```

## 环境变量

所有脚本使用以下环境变量（带默认值）：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_HOST` | localhost | 数据库主机 |
| `DB_PORT` | 5432 | 数据库端口 |
| `DB_USER` | codeatlas | 数据库用户 |
| `DB_PASSWORD` | codeatlas | 数据库密码 |
| `DB_NAME` | codeatlas | 数据库名称 |
| `DB_SSLMODE` | disable | SSL 模式 |
| `EMBEDDING_DIMENSIONS` | - | 向量维度 |

## 故障排除

### 脚本权限错误
```bash
chmod +x scripts/*.sh
```

### 数据库连接失败
```bash
# 检查数据库是否运行
docker-compose ps

# 启动数据库
make docker-db
```

### 测试数据库清理失败
```bash
# 手动连接并清理
psql -h localhost -U codeatlas -d postgres
DROP DATABASE codeatlas_test_xxx;
```

## 参考资料

- [Makefile 使用指南](../docs/development/makefile-guide.md)
- [测试指南](../docs/development/testing.md)
- [DevContainer 指南](../docs/development/devcontainer.md)
