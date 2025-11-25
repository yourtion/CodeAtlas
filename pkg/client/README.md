# API 客户端开发指南

> CLI 工具与 API 服务器通信的 HTTP 客户端

## 功能特性

- 完整的 API 端点覆盖
- 指数退避重试逻辑
- 连接池管理
- Bearer Token 认证
- 可配置超时
- 健康检查
- 详细错误响应

## 快速开始

### 基础使用

```go
import "github.com/yourtionguo/CodeAtlas/pkg/client"

// 创建客户端
apiClient := client.NewAPIClient("http://localhost:8080")

// 健康检查
if err := apiClient.Health(ctx); err != nil {
    log.Fatalf("服务器不健康: %v", err)
}
```

### 带认证

```go
apiClient := client.NewAPIClient(
    "http://localhost:8080",
    client.WithToken("your-api-token"),
)
```

### 自定义配置

```go
apiClient := client.NewAPIClient(
    "http://localhost:8080",
    client.WithTimeout(10*time.Minute),
    client.WithMaxRetries(5),
    client.WithToken("your-api-token"),
)
```

## API 方法

### Index - 索引仓库

```go
req := &client.IndexRequest{
    RepoName: "my-project",
    RepoURL:  "https://github.com/user/my-project",
    Branch:   "main",
    ParseOutput: parseOutput,
    Options: client.IndexOptions{
        Incremental:    false,
        SkipVectors:    false,
        BatchSize:      100,
        WorkerCount:    4,
        EmbeddingModel: "text-embedding-3-small",
    },
}

resp, err := apiClient.Index(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("索引了 %d 个文件，创建了 %d 个符号\n", 
    resp.FilesProcessed, resp.SymbolsCreated)
```

### Search - 语义搜索

```go
embedding := []float32{0.1, 0.2, 0.3}
filters := client.SearchFilters{
    RepoID:   "repo-123",
    Language: "go",
    Kind:     []string{"function", "class"},
    Limit:    10,
}

resp, err := apiClient.Search(ctx, "authentication function", embedding, filters)
if err != nil {
    log.Fatal(err)
}

for _, result := range resp.Results {
    fmt.Printf("%s (%s) - 相似度: %.2f\n", 
        result.Name, result.Kind, result.Similarity)
}
```

### GetCallers - 查找调用者

```go
resp, err := apiClient.GetCallers(ctx, "symbol-id-123")
if err != nil {
    log.Fatal(err)
}

for _, symbol := range resp.Symbols {
    fmt.Printf("%s in %s\n", symbol.Name, symbol.FilePath)
}
```

### GetCallees - 查找被调用者

```go
resp, err := apiClient.GetCallees(ctx, "symbol-id-123")
if err != nil {
    log.Fatal(err)
}

for _, symbol := range resp.Symbols {
    fmt.Printf("%s in %s\n", symbol.Name, symbol.FilePath)
}
```

### GetDependencies - 查找依赖

```go
resp, err := apiClient.GetDependencies(ctx, "symbol-id-123")
if err != nil {
    log.Fatal(err)
}

for _, dep := range resp.Dependencies {
    fmt.Printf("%s (%s) via %s\n", dep.Name, dep.Kind, dep.EdgeType)
}
```

### GetFileSymbols - 获取文件符号

```go
resp, err := apiClient.GetFileSymbols(ctx, "file-id-123")
if err != nil {
    log.Fatal(err)
}

for _, symbol := range resp.Symbols {
    fmt.Printf("%s (%s) at line %d\n", 
        symbol.Name, symbol.Kind, symbol.StartLine)
}
```

## 配置选项

### WithTimeout - 设置超时

```go
client.WithTimeout(5 * time.Minute)
```

### WithToken - 设置认证令牌

```go
client.WithToken("your-api-token")
```

### WithMaxRetries - 设置最大重试次数

```go
client.WithMaxRetries(5)
```

## 错误处理

```go
resp, err := apiClient.Index(ctx, req)
if err != nil {
    if apiErr, ok := err.(*client.APIError); ok {
        fmt.Printf("API 错误 (状态 %d): %s\n", 
            apiErr.StatusCode, apiErr.Message)
        if apiErr.Details != nil {
            fmt.Printf("详情: %v\n", apiErr.Details)
        }
    } else {
        fmt.Printf("网络错误: %v\n", err)
    }
    return
}
```

## 重试逻辑

客户端自动重试失败的请求：

- 服务器错误 (5xx 状态码)
- 速率限制 (429 状态码)
- 网络错误
- 指数退避: 1s, 2s, 4s, 8s, ... (最大 30s)
- 可配置最大重试次数（默认: 3）

不可重试的错误（4xx 除了 429）立即失败。

## 连接池

客户端使用连接池进行高效的 HTTP 通信：

- 最大空闲连接: 100
- 每个主机最大空闲连接: 10
- 空闲连接超时: 90 秒

## 线程安全

`APIClient` 可安全地被多个 goroutine 并发使用。

## 测试

```bash
# 运行测试
go test ./pkg/client/...

# 带覆盖率
go test ./pkg/client/... -cover
```

## 使用示例

完整示例请参考 `pkg/client/example_test.go`。

## 参考资料

- [API 文档](../../docs/api.md)
- [CLI 文档](../../docs/cli.md)
- [配置指南](../../docs/configuration.md)
