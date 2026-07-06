# Indexer 开发指南

> 索引器组件的开发文档

## 概述

索引器负责将解析后的代码结构转换为可查询的知识库，协调验证、数据库写入、图构建和向量嵌入生成。

## 核心组件

### Indexer 主协调器

多阶段处理流程：
1. **验证** - 验证输入的 schema 约束
2. **仓库元数据** - 创建或更新仓库记录
3. **数据库写入** - 持久化文件、符号、AST 节点和边
4. **图构建** - 构建关系表图节点和关系
5. **嵌入生成** - 创建语义搜索的向量嵌入

### 配置

```go
type IndexerConfig struct {
    RepoID          string
    RepoName        string
    RepoURL         string
    Branch          string
    BatchSize       int  // 默认: 100
    WorkerCount     int  // 默认: 4
    SkipVectors     bool
    Incremental     bool
    UseTransactions bool
    EmbeddingModel  string
}
```

## 使用示例

### 基础索引

```go
config := &indexer.IndexerConfig{
    RepoID:      "repo-123",
    RepoName:    "my-project",
    BatchSize:   100,
    WorkerCount: 4,
}

idx := indexer.NewIndexer(db, config)
result, err := idx.Index(ctx, parseOutput)
```

### 增量索引

```go
config := indexer.DefaultIndexerConfig()
config.Incremental = true

idx := indexer.NewIndexer(db, config)
result, err := idx.Index(ctx, parseOutput)
```

### 进度跟踪

```go
progressChan := make(chan indexer.IndexProgress, 10)

go func() {
    for progress := range progressChan {
        fmt.Printf("[%s] %s (%.0f%%)\n", 
            progress.Stage, progress.Message, progress.Progress)
    }
}()

result, err := idx.IndexWithProgress(ctx, parseOutput, progressChan)
```

## 错误处理

### 错误类型

| 类型 | 描述 | 可重试 |
|------|------|--------|
| `ErrorTypeValidation` | Schema 验证失败 | 否 |
| `ErrorTypeDatabase` | 数据库操作失败 | 取决于具体情况 |
| `ErrorTypeGraph` | 图构建失败 | 是 |
| `ErrorTypeEmbedding` | 向量嵌入失败 | 是 |
| `ErrorTypeTransaction` | 事务失败 | 否 |

### 错误收集

```go
result, err := idx.Index(ctx, parseOutput)

if len(result.Errors) > 0 {
    for _, err := range result.Errors {
        log.Printf("[%s] %s: %s\n", err.Type, err.EntityID, err.Message)
    }
}
```

## 性能优化

### 批处理大小

- 较大批次减少数据库往返但使用更多内存
- 推荐: 50-200（取决于实体大小）
- 默认: 100

### Worker 数量

- 更多 worker 增加嵌入生成的并行度
- 受 CPU 核心数和 API 速率限制约束
- 推荐: 4-8
- 默认: 4

### 增量模式

- 使用文件校验和检测变化
- 显著加快重新索引速度
- 推荐用于 CI/CD 流程

## 向量嵌入

### 配置

```go
type EmbedderConfig struct {
    Backend              string        // "openai" 或 "local"
    APIEndpoint          string
    APIKey               string
    Model                string
    Dimensions           int
    BatchSize            int
    MaxRequestsPerSecond int
    MaxRetries           int
    Timeout              time.Duration
}
```

### 常用模型维度

- `nomic-embed-text`: 768
- `text-embedding-qwen3-embedding-0.6b`: 1024
- `text-embedding-3-small` (OpenAI): 1536
- `text-embedding-3-large` (OpenAI): 3072

### 本地开发

使用 LM Studio:
```go
config := &indexer.EmbedderConfig{
    Backend:     "openai",
    APIEndpoint: "http://localhost:1234/v1/embeddings",
    Model:       "nomic-embed-text",
    Dimensions:  768,
}
```

## 测试

```bash
# 运行所有索引器测试
go test -v ./internal/indexer

# 运行特定测试
go test -v ./internal/indexer -run TestIndexValidInput

# 带覆盖率
go test -cover ./internal/indexer
```

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                         Indexer                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │Validator │→ │  Writer  │→ │  Graph   │→ │ Embedder │   │
│  └──────────┘  └──────────┘  │ Builder  │  └──────────┘   │
│                               └──────────┘                   │
└─────────────────────────────────────────────────────────────┘
                          ↓
        ┌─────────────────┴─────────────────┐
        ↓                 ↓                  ↓
   ┌──────────┐    ┌──────────┐      ┌──────────┐
   │PostgreSQL│    │ 关系表   │      │ pgvector │
   │(关系型)  │    │ (图查询) │      │(向量)    │
   └──────────┘    └──────────┘      └──────────┘
```

## 配置最佳实践

### 开发环境

```go
config := &indexer.IndexerConfig{
    BatchSize:       50,
    WorkerCount:     2,
    SkipVectors:     true,  // 开发时更快
    Incremental:     false,
    UseTransactions: true,
}
```

### 生产环境

```go
config := &indexer.IndexerConfig{
    BatchSize:       100,
    WorkerCount:     8,
    SkipVectors:     false,
    Incremental:     true,  // 更快的重新索引
    UseTransactions: true,
}
```

### 大型代码库 (>10,000 文件)

```go
config := &indexer.IndexerConfig{
    BatchSize:       200,
    WorkerCount:     16,
    SkipVectors:     false,
    Incremental:     true,
    UseTransactions: false, // 批量操作性能更好
}
```

## 故障排除

### 索引缓慢

- 增加 `BatchSize` 减少数据库往返
- 增加 `WorkerCount` 提高并行度
- 启用 `Incremental` 模式用于重新索引
- 考虑对大型导入禁用事务

### 内存问题

- 减少 `BatchSize` 使用更少内存
- 减少 `WorkerCount` 限制并发操作
- 分块处理仓库

### 验证错误

- 检查输入结构是否匹配 schema
- 验证所有必需字段是否存在
- 确保引用完整性（边中的符号 ID 存在）

## 参考资料

- [Schema 文档](../../docs/schema.md)
- [API 文档](../../docs/api.md)
- [配置指南](../../docs/configuration.md)
