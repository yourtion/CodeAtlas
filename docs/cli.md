# CLI 工具指南

> CodeAtlas 命令行工具完整参考

## 概述

CodeAtlas CLI 提供两个主要命令：
- `parse` - 解析代码生成 AST
- `index` - 将解析结果索引到数据库

## Parse 命令

### 基本用法

```bash
# 解析仓库
codeatlas parse --path /path/to/repo

# 解析单个文件
codeatlas parse --file main.go

# 保存到文件
codeatlas parse --path /path/to/repo --output result.json
```

### 选项

#### 必需（二选一）
- `--path, -p` - 仓库或目录路径
- `--file, -f` - 单个文件路径

#### 可选
- `--output, -o` - 输出文件路径（默认 stdout）
- `--language, -l` - 过滤语言（go, javascript, typescript, python, kotlin, java, swift, objc, c, cpp）
- `--workers, -w` - 并发数（默认 CPU 核心数）
- `--verbose, -v` - 详细日志
- `--ignore-pattern` - 忽略模式（可重复）
- `--no-ignore` - 禁用所有忽略规则
- `--semantic` - 启用 LLM 语义增强

### 常用示例

```bash
# 只解析 Go 文件
codeatlas parse --path . --language go

# 使用 4 个并发
codeatlas parse --path . --workers 4

# 忽略测试文件
codeatlas parse --path . \
  --ignore-pattern "*.test.js" \
  --ignore-pattern "*.spec.ts"

# 启用语义增强
export CODEATLAS_LLM_API_KEY=sk-xxx
codeatlas parse --path . --semantic
```

### 输出格式

```json
{
  "files": [
    {
      "file_id": "uuid",
      "path": "src/main.go",
      "language": "go",
      "symbols": [
        {
          "symbol_id": "uuid",
          "name": "main",
          "kind": "function",
          "signature": "func main()",
          "span": {
            "start_line": 10,
            "end_line": 25
          }
        }
      ]
    }
  ],
  "relationships": [
    {
      "edge_id": "uuid",
      "source_id": "uuid",
      "target_id": "uuid",
      "edge_type": "call"
    }
  ],
  "metadata": {
    "total_files": 150,
    "success_count": 145,
    "failure_count": 5
  }
}
```

## Index 命令

### 基本用法

```bash
# 索引解析结果
codeatlas index --input result.json --repo-name myproject

# 指定服务器地址
codeatlas index --input result.json \
  --repo-name myproject \
  --server http://localhost:8080
```

### 选项

- `--input, -i` - 解析结果文件路径（必需）
- `--repo-name, -r` - 仓库名称（必需）
- `--server, -s` - API 服务器地址（默认 http://localhost:8080）
- `--batch-size` - 批处理大小（默认 100）
- `--verbose, -v` - 详细日志

### 示例

```bash
# 基本索引
codeatlas index -i result.json -r myproject

# 大批量索引
codeatlas index -i result.json -r myproject --batch-size 500

# 远程服务器
codeatlas index -i result.json -r myproject \
  --server https://codeatlas.example.com
```

## 环境变量

### LLM 配置（用于 --semantic）

```bash
# OpenAI
export CODEATLAS_LLM_API_KEY=sk-xxx
export CODEATLAS_LLM_MODEL=text-embedding-3-small
export CODEATLAS_LLM_API_URL=https://api.openai.com/v1

# 本地模型
export CODEATLAS_LLM_API_URL=http://localhost:8000/v1
export CODEATLAS_LLM_MODEL=local-model
```

### 数据库配置（用于 index）

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=codeatlas
export DB_NAME=codeatlas
```

## 支持的语言

| 语言 | 扩展名 | 特性 |
|------|--------|------|
| Go | .go | 包、函数、结构体、接口 |
| JavaScript | .js, .jsx | 模块、函数、类 |
| TypeScript | .ts, .tsx | 类型、接口、装饰器 |
| Python | .py | 模块、函数、类、装饰器 |
| Kotlin | .kt, .kts | 类、函数、属性 |
| Java | .java | 类、方法、注解 |
| Swift | .swift | 类、函数、协议 |
| Objective-C | .h, .m, .mm | 接口、实现、协议 |
| C | .c, .h | 函数、结构体 |
| C++ | .cpp, .hpp | 类、模板、命名空间 |

## 默认忽略规则

### 目录
- `.git/`, `node_modules/`, `vendor/`
- `__pycache__/`, `.venv/`, `venv/`
- `dist/`, `build/`, `.next/`

### 文件
- 二进制: `.exe`, `.dll`, `.so`
- 图片: `.jpg`, `.png`, `.gif`
- 压缩: `.zip`, `.tar`, `.gz`

使用 `--no-ignore` 禁用所有规则。

## 性能优化

### 1. 调整并发数

```bash
# CPU 密集型（小文件）
codeatlas parse --path . --workers 8

# I/O 密集型（大文件）
codeatlas parse --path . --workers 4
```

### 2. 过滤语言

```bash
# 只解析需要的语言
codeatlas parse --path . --language go
```

### 3. 排除不需要的文件

```bash
codeatlas parse --path . \
  --ignore-pattern "vendor/*" \
  --ignore-pattern "*.min.js"
```

### 4. 批量处理

```bash
# 分目录解析
codeatlas parse --path ./backend -o backend.json
codeatlas parse --path ./frontend -o frontend.json

# 分别索引
codeatlas index -i backend.json -r myproject-backend
codeatlas index -i frontend.json -r myproject-frontend
```

## 故障排除

### 解析失败

```bash
# 使用 verbose 查看详细信息
codeatlas parse --path . --verbose

# 检查特定文件
codeatlas parse --file problematic.go --verbose
```

### 内存不足

```bash
# 减少并发数
codeatlas parse --path . --workers 2

# 分批处理
codeatlas parse --path ./src --output src.json
codeatlas parse --path ./lib --output lib.json
```

### 语义增强失败

```bash
# 检查 API key
echo $CODEATLAS_LLM_API_KEY

# 测试连接
curl -H "Authorization: Bearer $CODEATLAS_LLM_API_KEY" \
  https://api.openai.com/v1/models
```

### 索引失败

```bash
# 检查数据库连接
psql -U codeatlas -d codeatlas -c "SELECT 1;"

# 检查 API 服务器
curl http://localhost:8080/health

# 使用小批量
codeatlas index -i result.json -r myproject --batch-size 50
```

## 集成示例

### CI/CD 集成

```yaml
# .github/workflows/parse.yml
name: Parse Codebase
on: [push]
jobs:
  parse:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Build CLI
        run: make build-cli
      - name: Parse code
        run: ./bin/cli parse --path . --output ast.json
      - name: Upload artifact
        uses: actions/upload-artifact@v2
        with:
          name: ast-output
          path: ast.json
```

### 与 jq 结合

```bash
# 提取所有函数名
codeatlas parse --path . | \
  jq -r '.files[].symbols[] | select(.kind == "function") | .name'

# 统计符号类型
codeatlas parse --path . | \
  jq '[.files[].symbols[].kind] | group_by(.) | 
      map({kind: .[0], count: length})'

# 查找特定文件的符号
codeatlas parse --path . | \
  jq '.files[] | select(.path | contains("main.go")) | .symbols'
```

### 定时任务

```bash
#!/bin/bash
# daily-parse.sh

REPO_PATH="/path/to/repo"
OUTPUT_DIR="/path/to/output"
DATE=$(date +%Y%m%d)

# 解析
codeatlas parse --path "$REPO_PATH" \
  --output "$OUTPUT_DIR/parse-$DATE.json"

# 索引
codeatlas index \
  --input "$OUTPUT_DIR/parse-$DATE.json" \
  --repo-name "myproject"

# 清理旧文件（保留 7 天）
find "$OUTPUT_DIR" -name "parse-*.json" -mtime +7 -delete
```

## 最佳实践

1. **开发环境**: 使用 `--language` 过滤，加快解析速度
2. **生产环境**: 使用 `--semantic` 获得更好的语义理解
3. **大型仓库**: 分目录解析，避免内存问题
4. **CI/CD**: 只解析变更的文件，节省时间
5. **定期索引**: 使用定时任务保持索引最新

## 下一步

- 查看 [API 指南](api.md) 了解如何查询索引数据
- 查看 [配置指南](configuration.md) 自定义解析行为
- 查看 [架构设计](architecture.md) 了解解析器原理
