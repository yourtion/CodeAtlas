# CLI 工具完整指南

> CodeAtlas 命令行工具使用手册

## 目录

- [概述](#概述)
- [安装](#安装)
- [Parse 命令](#parse-命令)
- [Index 命令](#index-命令)
- [环境变量](#环境变量)
- [故障排除](#故障排除)

## 概述

CodeAtlas CLI 提供两个主要命令：

1. **parse** - 解析源代码并输出结构化 JSON AST
2. **index** - 将解析结果索引到知识图谱

### 支持的语言

| 语言 | 扩展名 | 特性 |
|----------|-----------|----------|
| Go | .go | 包、导入、函数、方法、结构体、接口、类型 |
| JavaScript | .js, .jsx | ES6 模块、CommonJS、函数、类、箭头函数 |
| TypeScript | .ts, .tsx | 所有 JavaScript 特性 + 类型注解 |
| Python | .py | 导入、函数、类、装饰器、类型提示、文档字符串 |

## 安装

```bash
# 构建 CLI 工具
make build-cli

# 二进制文件位于
./bin/cli

# 可选：添加到 PATH
export PATH=$PATH:$(pwd)/bin
```

## Parse 命令

### 基本用法

```bash
# 解析整个仓库
codeatlas parse --path /path/to/repository

# 解析单个文件
codeatlas parse --file /path/to/file.go

# 保存输出到文件
codeatlas parse --path /path/to/repository --output result.json
```

### 命令行选项

#### 必需选项（二选一）

| 选项 | 说明 | 示例 |
|------|-------------|---------|
| `--path`, `-p` | 仓库或目录路径 | `--path ./myproject` |
| `--file`, `-f` | 单个文件路径 | `--file main.go` |

#### 可选选项

| 选项 | 说明 | 默认值 | 示例 |
|------|-------------|---------|---------|
| `--output`, `-o` | 输出文件路径 | stdout | `--output result.json` |
| `--language`, `-l` | 按语言过滤 | 全部 | `--language go` |
| `--workers`, `-w` | 并发工作线程数 | CPU 核心数 | `--workers 4` |
| `--verbose`, `-v` | 启用详细日志 | false | `--verbose` |
| `--ignore-pattern` | 忽略模式（可重复） | 无 | `--ignore-pattern "*.test.js"` |
| `--no-ignore` | 禁用所有忽略规则 | false | `--no-ignore` |
| `--semantic` | 启用 LLM 语义增强 | false | `--semantic` |

### 常用示例

#### 1. 解析 Go 仓库

```bash
codeatlas parse --path ./mygoproject --language go --output go-ast.json
```

#### 2. 使用自定义忽略模式

```bash
codeatlas parse --path ./project \
  --ignore-pattern "*.test.js" \
  --ignore-pattern "*.spec.ts" \
  --ignore-pattern "mock_*.go"
```

#### 3. 详细输出

```bash
codeatlas parse --file src/main.go --verbose
```

输出：
```
[INFO] Starting parse command
[INFO] Parsing single file: src/main.go
[INFO] Detected language: go
[INFO] Parsing file: src/main.go
[DEBUG] Extracted 5 symbols
[DEBUG] Extracted 3 relationships
[INFO] Parse complete: 1 file, 0 errors
```

#### 4. 多线程解析

```bash
codeatlas parse --path ./large-repo --workers 8 --verbose
```

#### 5. 语义增强（需要 API 密钥）

```bash
export CODEATLAS_LLM_API_KEY=sk-your-api-key
codeatlas parse --path ./project --semantic --output enhanced.json
```

#### 6. 禁用所有忽略规则

```bash
codeatlas parse --path ./project --no-ignore
```

### 输出格式

Parse 命令输出符合 CodeAtlas 统一 Schema 的 JSON：

```json
{
  "files": [
    {
      "file_id": "550e8400-e29b-41d4-a716-446655440000",
      "path": "src/main.go",
      "language": "go",
      "size": 1024,
      "checksum": "a3b2c1d4e5f6...",
      "symbols": [
        {
          "symbol_id": "770e8400-e29b-41d4-a716-446655440002",
          "name": "main",
          "kind": "function",
          "signature": "func main()",
          "span": {
            "start_line": 10,
            "end_line": 25
          },
          "docstring": "main is the entry point"
        }
      ]
    }
  ],
  "relationships": [
    {
      "edge_id": "880e8400-e29b-41d4-a716-446655440003",
      "source_id": "770e8400-e29b-41d4-a716-446655440002",
      "target_id": "990e8400-e29b-41d4-a716-446655440004",
      "edge_type": "call"
    }
  ],
  "metadata": {
    "version": "1.0.0",
    "timestamp": "2025-11-06T10:30:00Z",
    "total_files": 150,
    "success_count": 145,
    "failure_count": 5
  }
}
```

### 使用 jq 处理输出

```bash
# 提取所有函数名
codeatlas parse --path ./project | jq -r '.files[].symbols[] | select(.kind == "function") | .name'

# 按类型统计符号
codeatlas parse --path ./project | jq '[.files[].symbols[].kind] | group_by(.) | map({kind: .[0], count: length})'

# 查看错误
codeatlas parse --path ./project | jq '.metadata.errors'

# 只查看 Go 文件
codeatlas parse --path ./project | jq '.files[] | select(.language == "go")'
```

### 默认忽略模式

以下模式默认被忽略（除非使用 `--no-ignore`）：

**目录**：
- `.git/`, `node_modules/`, `vendor/`, `__pycache__/`
- `.venv/`, `venv/`, `dist/`, `build/`, `.next/`, `.nuxt/`

**文件扩展名**：
- 二进制：`.exe`, `.dll`, `.so`, `.dylib`, `.a`
- 图片：`.jpg`, `.jpeg`, `.png`, `.gif`, `.svg`, `.ico`
- 文档：`.pdf`, `.doc`, `.docx`
- 压缩包：`.zip`, `.tar`, `.gz`, `.rar`

## Index 命令

### 基本用法

```bash
# 从仓库路径解析并索引
codeatlas index --path /path/to/repo --api-url http://localhost:8080

# 从解析输出文件索引
codeatlas index --input parsed.json --api-url http://localhost:8080

# 带仓库元数据索引
codeatlas index \
  --path /path/to/repo \
  --repo-name "my-project" \
  --repo-url "https://github.com/user/my-project" \
  --branch "main" \
  --api-url http://localhost:8080
```

### 命令行选项

#### 必需选项（二选一）

| 选项 | 说明 | 示例 |
|------|-------------|---------|
| `--path`, `-p` | 仓库路径（解析并索引） | `--path ./myproject` |
| `--input`, `-i` | 解析输出 JSON 文件 | `--input parsed.json` |

#### API 配置

| 选项 | 说明 | 默认值 | 示例 |
|------|-------------|---------|---------|
| `--api-url` | API 服务器 URL | `http://localhost:8080` | `--api-url http://api.example.com` |
| `--api-token` | 认证令牌 | `` | `--api-token token123` |
| `--timeout` | 请求超时 | `5m` | `--timeout 10m` |

#### 仓库元数据

| 选项 | 说明 | 默认值 | 示例 |
|------|-------------|---------|---------|
| `--repo-name`, `-n` | 仓库名称 | 目录名 | `--repo-name my-project` |
| `--repo-url`, `-u` | 仓库 URL | `` | `--repo-url https://github.com/user/repo` |
| `--branch`, `-b` | Git 分支 | `main` | `--branch develop` |
| `--commit-hash` | Git 提交哈希 | 自动检测 | `--commit-hash abc123` |

#### 索引选项

| 选项 | 说明 | 默认值 | 示例 |
|------|-------------|---------|---------|
| `--incremental` | 只处理变更文件 | `false` | `--incremental` |
| `--skip-vectors` | 跳过向量生成 | `false` | `--skip-vectors` |
| `--batch-size` | 批处理大小 | `100` | `--batch-size 200` |

### 常用示例

#### 1. 基本索引

```bash
codeatlas index --path ./myproject --api-url http://localhost:8080
```

输出：
```
Parsing repository...
Parsed 150 files, 1250 symbols, 3400 edges

Indexing to knowledge graph...
Repository ID: 550e8400-e29b-41d4-a716-446655440000
Files processed: 150
Symbols created: 1250
Edges created: 3400
Vectors created: 1250
Duration: 45.2s

✓ Indexing completed successfully
```

#### 2. 增量更新

```bash
# 初始索引
codeatlas index --path ./myproject --api-url http://localhost:8080

# 修改代码后，只重新索引变更的文件
codeatlas index --path ./myproject --incremental --api-url http://localhost:8080
```

输出：
```
Detected 5 changed files (145 unchanged, skipped)
Parsed 5 files, 42 symbols, 87 edges
Files processed: 5
Duration: 3.1s
```

#### 3. 快速索引（跳过向量）

```bash
codeatlas index \
  --path ./large-project \
  --skip-vectors \
  --batch-size 200 \
  --workers 8 \
  --api-url http://localhost:8080
```

#### 4. 带认证索引

```bash
export API_TOKEN=my-secret-token

codeatlas index \
  --path ./myproject \
  --api-url https://api.example.com \
  --api-token $API_TOKEN
```

#### 5. 从已有解析结果索引

```bash
# 先解析
codeatlas parse --path ./myproject --output parsed.json

# 索引到多个服务器
codeatlas index --input parsed.json --api-url http://dev.example.com
codeatlas index --input parsed.json --api-url http://staging.example.com
```

#### 6. 完整元数据索引

```bash
codeatlas index \
  --path ./myproject \
  --repo-name "CodeAtlas" \
  --repo-url "https://github.com/yourtionguo/CodeAtlas" \
  --branch "main" \
  --commit-hash "$(git rev-parse HEAD)" \
  --api-url http://localhost:8080
```

### 工作流示例

#### 工作流 1: 初始仓库设置

```bash
# 1. 启动 API 服务器
make run-api

# 2. 索引仓库
codeatlas index \
  --path /path/to/repo \
  --repo-name "my-project" \
  --repo-url "https://github.com/user/my-project" \
  --api-url http://localhost:8080

# 3. 验证索引
curl http://localhost:8080/api/v1/repositories
```

#### 工作流 2: 持续集成

```bash
#!/bin/bash
# ci-index.sh

# 解析代码
codeatlas parse --path . --output parsed.json

# 索引到 staging
codeatlas index \
  --input parsed.json \
  --repo-name "$CI_PROJECT_NAME" \
  --repo-url "$CI_PROJECT_URL" \
  --branch "$CI_COMMIT_BRANCH" \
  --commit-hash "$CI_COMMIT_SHA" \
  --api-url "$STAGING_API_URL" \
  --api-token "$STAGING_API_TOKEN"

# 主分支索引到生产环境
if [ "$CI_COMMIT_BRANCH" = "main" ]; then
  codeatlas index \
    --input parsed.json \
    --api-url "$PROD_API_URL" \
    --api-token "$PROD_API_TOKEN"
fi
```

#### 工作流 3: 增量更新

```bash
#!/bin/bash
# update-index.sh

# 拉取最新变更
git pull

# 只重新索引变更的文件
codeatlas index \
  --path . \
  --incremental \
  --api-url http://localhost:8080

echo "Index updated successfully"
```

## 环境变量

### Parse 命令环境变量

#### LLM 增强（仅在使用 `--semantic` 时需要）

| 变量 | 说明 | 必需 | 示例 |
|----------|-------------|----------|---------|
| `CODEATLAS_LLM_API_KEY` | LLM API 密钥 | 是 | `export CODEATLAS_LLM_API_KEY=sk-...` |
| `CODEATLAS_LLM_API_URL` | 自定义 LLM API 端点 | 否 | `export CODEATLAS_LLM_API_URL=https://api.openai.com/v1` |
| `CODEATLAS_LLM_MODEL` | LLM 模型名称 | 否 | `export CODEATLAS_LLM_MODEL=gpt-4` |

**配置示例**：

```bash
# OpenAI
export CODEATLAS_LLM_API_KEY=sk-your-openai-key
export CODEATLAS_LLM_MODEL=gpt-3.5-turbo

# Azure OpenAI
export CODEATLAS_LLM_API_KEY=your-azure-key
export CODEATLAS_LLM_API_URL=https://your-resource.openai.azure.com/...
export CODEATLAS_LLM_MODEL=gpt-35-turbo

# 本地 LLM（如 Ollama）
export CODEATLAS_LLM_API_URL=http://localhost:11434/v1
export CODEATLAS_LLM_MODEL=llama2
# 本地不需要 API 密钥
```

### Index 命令环境变量

| 变量 | 说明 | 示例 |
|----------|-------------|---------|
| `CODEATLAS_API_URL` | 默认 API URL | `export CODEATLAS_API_URL=http://localhost:8080` |
| `CODEATLAS_API_TOKEN` | 默认 API 令牌 | `export CODEATLAS_API_TOKEN=token123` |
| `CODEATLAS_BATCH_SIZE` | 默认批处理大小 | `export CODEATLAS_BATCH_SIZE=200` |

### 使用 .env 文件

创建 `.env` 文件：

```bash
# .env
CODEATLAS_LLM_API_KEY=sk-your-key
CODEATLAS_LLM_API_URL=https://api.openai.com/v1
CODEATLAS_LLM_MODEL=gpt-3.5-turbo
CODEATLAS_API_URL=http://localhost:8080
```

加载并使用：

```bash
# 加载环境变量
export $(cat .env | xargs)

# 运行命令
codeatlas parse --path ./project --semantic
codeatlas index --path ./project
```

**安全提示**：将 `.env` 添加到 `.gitignore`：

```bash
echo ".env" >> .gitignore
```

## 故障排除

### Parse 命令问题

#### 1. "No files found to parse"

**原因**：所有文件被忽略规则过滤或不存在支持的文件。

**解决方案**：
```bash
# 使用详细输出调试
codeatlas parse --path ./project --verbose

# 禁用忽略规则
codeatlas parse --path ./project --no-ignore

# 检查路径是否正确
ls -la ./project
```

#### 2. "Syntax error: unexpected token"

**原因**：文件包含 Tree-sitter 无法解析的语法错误。

**解决方案**：
- 修复源文件中的语法错误
- 解析器会继续处理其他文件并在元数据中报告错误
- 检查输出 JSON 中的 `metadata.errors`

#### 3. "Permission denied"

**原因**：文件或目录权限不足。

**解决方案**：
```bash
# 检查权限
ls -la ./project

# 修复权限
chmod -R u+r ./project
```

#### 4. "Out of memory" 或性能慢

**原因**：仓库太大或并发线程过多。

**解决方案**：
```bash
# 减少工作线程
codeatlas parse --path ./huge-repo --workers 2

# 只解析特定语言
codeatlas parse --path ./huge-repo --language go

# 排除大目录
codeatlas parse --path ./huge-repo \
  --ignore-pattern "vendor/*" \
  --ignore-pattern "node_modules/*"
```

#### 5. "LLM API error"

**原因**：API 密钥未设置、速率限制或网络问题。

**解决方案**：
```bash
# 验证 API 密钥
echo $CODEATLAS_LLM_API_KEY

# 设置 API 密钥
export CODEATLAS_LLM_API_KEY=sk-your-key

# 测试 API 连接
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $CODEATLAS_LLM_API_KEY"
```

### Index 命令问题

#### 1. "Connection refused"

**错误**：
```
Error: failed to connect to API server: connection refused
```

**解决方案**：
```bash
# 检查 API 服务器是否运行
curl http://localhost:8080/health

# 启动 API 服务器
make run-api

# 验证 API URL
codeatlas index --path . --api-url http://localhost:8080
```

#### 2. "Authentication failed"

**错误**：
```
Error: authentication failed: invalid token
```

**解决方案**：
```bash
# 检查服务器是否启用认证
echo $ENABLE_AUTH

# 提供有效令牌
codeatlas index --path . --api-token your-token --api-url http://localhost:8080

# 验证令牌在服务器的 AUTH_TOKENS 中
echo $AUTH_TOKENS
```

#### 3. "Timeout"

**错误**：
```
Error: request timeout after 5m0s
```

**解决方案**：
```bash
# 增加超时
codeatlas index --path . --timeout 10m --api-url http://localhost:8080

# 使用更小的批处理大小
codeatlas index --path . --batch-size 50 --api-url http://localhost:8080

# 跳过向量生成
codeatlas index --path . --skip-vectors --api-url http://localhost:8080
```

## 性能优化

### Parse 命令优化

#### 1. 优化工作线程数

```bash
# CPU 密集型系统
codeatlas parse --path ./project --workers 4

# I/O 密集型系统（许多小文件）
codeatlas parse --path ./project --workers 8
```

#### 2. 使用语言过滤

```bash
codeatlas parse --path ./project --language go
```

#### 3. 排除不必要的文件

```bash
codeatlas parse --path ./project \
  --ignore-pattern "vendor/*" \
  --ignore-pattern "node_modules/*" \
  --ignore-pattern "*.min.js"
```

#### 4. 批量处理大仓库

```bash
codeatlas parse --path ./project/backend --output backend.json
codeatlas parse --path ./project/frontend --output frontend.json
```

### Index 命令优化

#### 1. 使用增量索引

```bash
codeatlas index --path . --incremental --api-url http://localhost:8080
```

**优势**：
- 只处理变更的文件
- 对于小变更快 10-100 倍
- 减少 API 服务器负载

#### 2. 初始索引时跳过向量

```bash
codeatlas index --path . --skip-vectors --api-url http://localhost:8080
```

**优势**：
- 索引速度快 5-10 倍
- 减少 API 成本
- 可以稍后生成向量

#### 3. 优化批处理大小

```bash
# 小仓库（< 100 文件）
codeatlas index --path . --batch-size 50 --api-url http://localhost:8080

# 大仓库（> 1000 文件）
codeatlas index --path . --batch-size 200 --api-url http://localhost:8080
```

#### 4. 解析一次，多次索引

```bash
# 解析一次
codeatlas parse --path . --output parsed.json

# 索引到多个服务器
codeatlas index --input parsed.json --api-url http://dev.example.com
codeatlas index --input parsed.json --api-url http://staging.example.com
codeatlas index --input parsed.json --api-url http://prod.example.com
```

## 性能基准

预期性能（典型硬件：4 核 CPU，8GB RAM）：

| 仓库大小 | 文件数 | Parse 时间 | Index 时间 | 工作线程 |
|----------------|-------|------|------|---------|
| 小型 | <100 | <10s | <5s | 4 |
| 中型 | 100-1000 | <2min | <1min | 4-8 |
| 大型 | 1000+ | <5min | <3min | 8 |

## 退出代码

| 代码 | 含义 |
|------|---------|
| 0 | 成功 |
| 1 | 一般错误 |
| 2 | 无效参数 |
| 3 | API 连接错误 |
| 4 | 认证错误 |
| 5 | 验证错误 |
| 6 | 超时 |

检查退出代码：
```bash
codeatlas parse --path ./project
echo $?  # 应该输出 0 表示成功
```

## 相关文档

- [快速开始指南](../../getting-started/quick-start.md)
- [API 使用指南](../api/README.md)
- [配置参考](../../configuration/README.md)
- [故障排除](../../troubleshooting/cli.md)

## 获取帮助

```bash
# 显示帮助
codeatlas --help
codeatlas parse --help
codeatlas index --help

# 显示版本
codeatlas --version
```

---

**最后更新**: 2025-11-06  
**维护者**: CodeAtlas Team
