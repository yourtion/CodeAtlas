# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

**CodeAtlas** 是一个智能知识图谱平台，用于探索、检索和理解代码库。它结合了 RAG（检索增强生成）、代码知识图谱和语义检索技术，帮助开发者、架构师和运维人员快速理解和导航大型代码库。

### 核心功能
- **代码/文档语义检索** - 支持自然语言查询代码实现、调用关系和业务逻辑
- **代码知识图谱** - 基于静态分析和语义解析构建全局代码关系图，支持复杂路径和依赖查询
- **多语言支持** - 支持 Go, JavaScript/TypeScript, Python, Kotlin, Java, Swift, Objective-C, C, C++
- **跨语言调用分析** - 自动检测跨语言调用关系（如 Kotlin→Java, Swift→Objective-C, C++→C）
- **增量更新** - 通过 CLI 轻量同步更新，支持基于 checksum 的增量索引

## 技术栈

| 模块 | 技术栈/工具 | 说明 |
|------|-------------|------|
| **后端服务** | Go 1.25+ (Gin) | 高性能 API 服务 |
| **解析引擎** | Go + Tree-sitter | 精确语法解析，支持 9+ 语言 |
| **向量存储** | PostgreSQL + pgvector | 语义检索 |
| **图谱存储** | PostgreSQL 关系表 | 依赖关系、路径查询 |
| **前端界面** | Svelte + Rsbuild | 轻量现代前端框架 |
| **容器化** | Docker + Docker Compose | 本地和生产环境一致 |
| **CLI 工具** | Go (urfave/cli/v2) | 跨平台代码分析和上传工具 |

## 核心架构

### 索引管道 (Indexing Pipeline)

```
扫描文件 → Tree-sitter解析 → 提取符号/关系 → 验证 → 写入数据库 → 构建图 → 生成向量
    ↓           ↓               ↓            ↓        ↓         ↓         ↓
 ScannedFile → ParsedFile → ParseOutput → Validator → Writer → GraphBuilder → Embedder
```

**关键组件**：
- `internal/parser/` - 语言解析器（每种语言一个 *_parser.go 文件）
- `internal/schema/` - 内部数据表示（ParseOutput, File, Symbol, DependencyEdge）
- `internal/indexer/` - 索引管道编排（验证 → 写入 → 图 → 向量）
- `internal/api/` - REST API 服务（Gin）
- `pkg/models/` - 数据库模型和仓库

### 多语言解析器架构

所有解析器实现统一接口：
```go
type Parser interface {
    Parse(file ScannedFile) (*ParsedFile, error)
    Language() string
}
```

**语言解析器** (`internal/parser/`)：
- `go_parser.go` - Go 语言解析器
- `js_parser.go` - JavaScript/TypeScript 解析器
- `python_parser.go` - Python 解析器
- `java_parser.go` - Java 解析器
- `kotlin_parser.go` - Kotlin 解析器
- `swift_parser.go` - Swift 解析器
- `c_parser.go` / `cpp_parser.go` - C/C++ 解析器
- `objc_parser.go` - Objective-C 解析器

**跨语言调用检测**：
- Kotlin → Java (JVM 互操作)
- Swift → Objective-C (iOS/macOS 互操作)
- C++ → C (extern "C")
- TypeScript → JavaScript (模块导入)

### CLI 工具命令

```bash
codeatlas parse  --path <repo> [--output <file>]    # 解析代码库
codeatlas index  --path <repo> --server <url>       # 索引到服务器
codeatlas upload --path <repo> --server <url>       # 上传仓库
codeatlas search --query <text> --server <url>      # 搜索代码
```

### API 端点

```
GET    /health                                 # 健康检查
POST   /api/v1/index                          # 索引仓库
GET    /api/v1/repositories                   # 列出仓库
GET    /api/v1/repositories/:id               # 获取仓库
POST   /api/v1/search                         # 搜索符号
GET    /api/v1/symbols/:id/callers            # 获取调用者
GET    /api/v1/symbols/:id/callees            # 获取被调用方
```

## 常用开发命令

### 快速开始

```bash
# 方式1: DevContainer（推荐）
# VS Code: 点击 "Reopen in Container"

# 方式2: Docker Compose
docker-compose up -d

# 方式3: 本地开发
make db                    # 启动数据库
make db-init              # 初始化数据库
make run-api              # 启动 API 服务
cd web && pnpm dev        # 启动前端
```

### 构建和运行

```bash
make build               # 构建所有二进制文件
make build-api           # 构建 API 服务器
make build-cli           # 构建 CLI 工具
make run-api             # 运行 API 服务器
make run-cli             # 运行 CLI 工具
```

### 测试

```bash
make test                # 快速单元测试（无需数据库）
make test-integration    # 完整集成测试（需要数据库）
make test-coverage       # 生成覆盖率报告
make verify              # 完整验证

# 调用分析测试（跨语言互操作）
go test -v ./tests/integration -run TestCallAnalysis_AllFixtures
go test -v ./tests/integration -run TestCallAnalysis_AllSingleLanguage
```

**测试覆盖率目标**：90%

### 数据库管理

```bash
make db                  # 启动数据库容器
make db-init             # 初始化数据库模式
make db-stop             # 停止数据库
make db-logs             # 查看数据库日志
make clean-test-dbs      # 清理测试数据库
```

### 前端开发

```bash
cd web
pnpm install            # 安装依赖
pnpm dev                # 开发服务器
pnpm build              # 生产构建
```

## 环境配置

复制 `.env.example` 到 `.env` 并配置：

```bash
# 数据库
DB_HOST=localhost
DB_PORT=5432
DB_USER=codeatlas
DB_PASSWORD=codeatlas
DB_NAME=codeatlas

# API 服务
API_HOST=0.0.0.0
API_PORT=8080
ENABLE_AUTH=false

# 索引器
INDEXER_BATCH_SIZE=100
INDEXER_WORKER_COUNT=4
INDEXER_SKIP_VECTORS=false

# 嵌入模型
EMBEDDING_BACKEND=openai
EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
EMBEDDING_DIMENSIONS=1024
```

## 关键架构决策

### 1. 为什么使用 Tree-sitter？

- 精确的语法解析（优于正则表达式）
- 增量解析支持
- 错误恢复能力
- 多语言统一接口
- 高性能

### 2. 为什么使用 PostgreSQL（而非 Neo4j/Elasticsearch）？

- **单数据库简化架构** - 关系数据 + 向量检索 + 图查询
- **pgvector** - 生产级向量检索
- **关系表图查询** - 基于 edges/symbols/files 关系表的 SQL 查询实现代码图谱
- **ACID 保证** - 事务完整性
- **成熟稳定** - 丰富的工具和生态

### 3. 索引管道设计

**流水线阶段**（`internal/indexer/indexer.go:184-443`）：
1. **验证** - SchemaValidator 验证数据完整性
2. **写入仓库** - 创建仓库元数据
3. **过滤变更** - 增量索引时只处理变更文件
4. **写入数据** - Writer 批量写入文件/符号/节点/边（支持事务）
5. **关联头文件** - HeaderImplAssociator 关联 C/C++/ObjC 头文件和实现
6. **构建图** - GraphBuilder 创建图节点和边
7. **生成嵌入** - Embedder 生成向量嵌入

**关键优化**：
- 批量处理（可配置 batch_size）
- 工作池并发（可配置 worker_count）
- 流式处理（StreamProcessor）- 处理大型 AST 树
- 自适应批处理（BatchOptimizer）- 根据延迟调整批次大小
- 内存压力监控 - 自动降低并发度

### 4. 数据模型

**核心表**：
- `repositories` - 仓库元数据
- `files` - 源代码文件（带 checksum 用于增量索引）
- `symbols` - 代码符号（函数、类、变量等）
- `ast_nodes` - AST 节点（完整语法树）
- `edges` - 关系边（call, import, extends, implements）
- `vectors` - 向量嵌入（pgvector）

**特殊文件**：
- `<external>` - 用于表示外部依赖的虚拟文件

### 5. 跨语言调用分析

**检测机制**：
- **Kotlin → Java**: 检测 Java 类和方法调用（100% 检测率）
- **Swift → Objective-C**: 检测 Objective-C 类和方法调用（100% 检测率）
- **C++ → C**: 检测 `extern "C"` 块和函数调用（100% 检测率）
- **TypeScript → JavaScript**: 检测 ES 模块导入（62.5% 检测率）

详见：`tests/integration/CALL_ANALYSIS_SUMMARY.md`

## 代码质量标准

**重要！！！**

1. **先写代码，后写测试** - 完成代码后编写测试
2. **运行所有测试** - 确保测试通过后提交
3. **更新文档** - 代码实现后同步更新相关文档
4. **覆盖率目标** - 90% 代码覆盖率
5. **中文沟通** - 尽可能使用中文进行沟通

### 测试规范

- **单元测试** - 快速测试，无数据库依赖（`make test`）
- **集成测试** - 完整测试，需要数据库（`make test-integration`）
- **表驱动测试** - 使用表驱动测试模式
- **子测试** - 使用 `t.Run()` 组织测试
- **清理资源** - 使用 `defer` 确保清理
- **测试独立性** - 测试之间不应有依赖

### Go 代码规范

- 遵循 Go 项目结构约定（cmd, internal, pkg）
- 使用 Gin 实现 REST API
- 实现完整的错误处理和日志记录
- 为所有业务逻辑编写单元测试
- 使用依赖注入提高可测试性
- 有意义的变量和函数名
- 保持函数小而专注
- 避免代码重复

### 前端代码规范（Svelte）

- 使用 Svelte 实现响应式 UI 组件
- 遵循现代 CSS 实践
- 实现响应式设计
- 使用 Rsbuild 进行快速编译
- **避免使用 `any` 类型**

## 项目结构

```
.
├── cmd/
│   ├── api/                    # API 服务器入口点
│   │   └── main.go
│   └── cli/                    # CLI 工具入口点
│       ├── main.go             # CLI 主程序
│       ├── parse_command.go    # parse 命令实现
│       ├── index_command.go    # index 命令实现
│       └── search_command.go   # search 命令实现
├── internal/
│   ├── api/                    # API 服务实现
│   │   ├── server.go           # Gin 服务器
│   │   ├── handlers/           # HTTP 处理器
│   │   └── middleware/         # 中间件
│   ├── parser/                 # 代码解析引擎
│   │   ├── tree_sitter.go      # Tree-sitter 封装
│   │   ├── scanner.go          # 文件扫描器
│   │   ├── go_parser.go        # Go 解析器
│   │   ├── js_parser.go        # JS/TS 解析器
│   │   ├── python_parser.go    # Python 解析器
│   │   ├── java_parser.go      # Java 解析器
│   │   ├── kotlin_parser.go    # Kotlin 解析器
│   │   ├── swift_parser.go     # Swift 解析器
│   │   ├── c_parser.go         # C 解析器
│   │   ├── cpp_parser.go       # C++ 解析器
│   │   └── objc_parser.go      # Objective-C 解析器
│   ├── indexer/                # 索引管道
│   │   ├── indexer.go          # 索引编排器
│   │   ├── validator.go        # 数据验证器
│   │   ├── writer.go           # 数据库写入器
│   │   ├── graph_builder.go    # 图构建器
│   │   ├── embedder.go         # 向量嵌入生成器
│   │   ├── stream_processor.go # 流式处理器
│   │   ├── batch_optimizer.go  # 批处理优化器
│   │   └── header_impl.go      # 头文件关联器
│   ├── schema/                 # 内部数据表示
│   │   ├── parse_output.go     # 解析输出结构
│   │   ├── file.go             # 文件模型
│   │   ├── symbol.go           # 符号模型
│   │   └── edge.go             # 边模型
│   └── ...
├── pkg/
│   ├── models/                 # 共享数据模型
│   │   ├── repository.go       # 仓库模型
│   │   ├── file.go             # 文件模型
│   │   ├── symbol.go           # 符号模型
│   │   ├── vector.go           # 向量模型
│   │   ├── edge.go             # 边模型
│   │   └── db.go               # 数据库连接
│   └── utils/                  # 工具函数
├── web/                        # Svelte 前端
│   ├── src/
│   └── public/
├── scripts/                    # 开发脚本
│   ├── init_db.go              # 数据库初始化
│   ├── verify_test_setup.sh    # 测试验证
│   └── cleanup_test_databases.sh
├── deployments/                # 部署配置
│   ├── Dockerfile.api          # API 服务 Dockerfile
│   ├── Dockerfile.cli          # CLI 工具 Dockerfile
│   └── .env.example            # 环境变量示例
├── docker/                     # Docker 配置
│   ├── Dockerfile              # PostgreSQL Dockerfile
│   └── initdb/                 # 数据库初始化脚本
├── docs/                       # 文档
│   ├── quick-start.md          # 快速开始
│   ├── cli.md                  # CLI 文档
│   ├── api.md                  # API 文档
│   ├── configuration.md        # 配置指南
│   ├── architecture.md         # 架构设计
│   ├── development/
│   │   ├── testing.md          # 测试指南
│   │   └── scripts.md          # 脚本文档
│   └── ...
├── tests/                      # 测试
│   └── integration/            # 集成测试
│       ├── call_analysis_test.go  # 调用分析测试
│       └── CALL_ANALYSIS_SUMMARY.md
├── go.mod                      # Go 模块定义
├── go.sum                      # Go 依赖
├── package.json                # 前端依赖
├── docker-compose.yml          # 开发环境
├── Makefile                    # 构建脚本
├── .env.example                # 环境变量示例
├── CLAUDE.md                   # 本文件
└── README.md                   # 项目文档
```

## 扩展指南

### 添加新语言解析器

1. 在 `internal/parser/` 创建 `newlang_parser.go`
2. 实现 `Parser` 接口
3. 在 `TreeSitterParser.GetLanguage()` 中注册
4. 在 `internal/parser/scanner.go` 添加文件扩展名映射
5. 编写集成测试验证调用分析

### 添加新的关系类型

1. 在 `internal/schema/edge.go` 中定义新的 `EdgeType`
2. 在解析器中提取新关系
3. 在 `GraphBuilder` 中处理新关系
4. 更新 API 端点支持查询新关系

### 添加新的嵌入模型

1. 在 `internal/indexer/embedder.go` 中实现新模型
2. 更新 `.env.example` 配置
3. 在 API 服务器中注册新模型

## 参考资源

### 文档
- [快速开始](docs/quick-start.md)
- [CLI 使用指南](docs/cli.md)
- [API 文档](docs/api.md)
- [配置指南](docs/configuration.md)
- [架构设计](docs/architecture.md)
- [测试指南](docs/development/testing.md)
- [调用分析测试结果](tests/integration/CALL_ANALYSIS_SUMMARY.md)

### 技术文档
- [Tree-sitter](https://tree-sitter.github.io/tree-sitter/)
- [pgvector](https://github.com/pgvector/pgvector)
- [Gin Framework](https://gin-gonic.com/)
