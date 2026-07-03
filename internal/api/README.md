# CodeAtlas API 服务架构

本文档详细说明 CodeAtlas API 服务的架构设计、实现细节和扩展指南。

## 目录

- [概述](#概述)
- [架构设计](#架构设计)
- [核心组件](#核心组件)
- [路由配置](#路由配置)
- [Handler 实现](#handler-实现)
- [中间件机制](#中间件机制)
- [错误处理](#错误处理)
- [数据验证](#数据验证)
- [性能优化](#性能优化)
- [扩展指南](#扩展指南)

## 概述

CodeAtlas API 服务是一个基于 **Gin** 框架构建的 REST API 服务,提供代码仓库的索引、检索和关系查询功能。

### 技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| **Web 框架** | Gin | 高性能 HTTP 路由框架 |
| **数据库** | PostgreSQL + AGE | 关系数据库 + 图查询 |
| **向量检索** | pgvector | 语义相似度搜索 |
| **认证** | Bearer Token | 可选的 API 认证 |
| **日志** | 结构化日志 | 基于 `internal/utils` |

### 核心功能

- **仓库索引** - 解析并存储代码库
- **语义搜索** - 基于向量的代码搜索
- **关系查询** - 调用关系、依赖分析
- **仓库管理** - CRUD 操作
- **图查询** - Apache AGE Cypher 查询

## 架构设计

### 分层架构

```
┌─────────────────────────────────────────┐
│           HTTP Clients                  │
│         (curl, web, CLI)                │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│         Middleware Layer                │
│    (Auth | CORS | Logging | Recovery)   │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│          Handler Layer                  │
│  (Repository | Search | Index | Relation)│
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│         Repository Layer                │
│  (Repository | File | Symbol | Vector)   │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│           Database Layer                │
│   PostgreSQL + AGE + pgvector           │
└─────────────────────────────────────────┘
```

### 数据流

```
Request → Middleware → Handler → Repository → Database
            │            │          │            │
            │            │          │            └── SQL/Cypher
            │            │          └──────────────┘
            │            └─────────────────────────┘
            └─────────────────────────────────────────┘
Response ← Handler ← Repository ← Database
```

## 核心组件

### Server

`Server` 是 API 服务的核心结构体:

```go
type Server struct {
    db                   *models.DB
    config               *ServerConfig
    repoRepository       *models.RepositoryRepository
    fileRepository       *models.FileRepository
    indexHandler         *handlers.IndexHandler
    repoHandler          *handlers.RepositoryHandler
    searchHandler        *handlers.SearchHandler
    relationshipHandler  *handlers.RelationshipHandler
}

type ServerConfig struct {
    EnableAuth     bool
    AuthTokens     []string
    CORSOrigins    []string
    EmbedderConfig *handlers.EmbedderConfig
}
```

**职责**:
- 初始化所有 Handlers
- 配置路由
- 管理中间件

**创建服务器**:

```go
config := &ServerConfig{
    EnableAuth:  true,
    AuthTokens:  []string{"token1", "token2"},
    CORSOrigins: []string{"https://example.com"},
}

server := NewServer(db, config)
router := server.SetupRouter()
router.Run(":8080")
```

### Handlers

所有 Handler 都遵循相同的模式:

```go
type XxxHandler struct {
    db     *models.DB
    repoX  *models.XxxRepository
}

func NewXxxHandler(db *models.DB) *XxxHandler
func (h *XxxHandler) Handle(c *gin.Context)
```

**Handler 列表**:

| Handler | 职责 | 端点 |
|---------|------|------|
| **IndexHandler** | 代码索引 | `POST /api/v1/index` |
| **RepositoryHandler** | 仓库管理 | `GET/POST /api/v1/repositories` |
| **SearchHandler** | 语义搜索 | `POST /api/v1/search` |
| **RelationshipHandler** | 关系查询 | `GET /api/v1/symbols/:id/*` |

## 路由配置

### 路由结构

```
/
├── /health (健康检查)
└── /api/v1/
    ├── POST /index (索引代码)
    ├── GET  /repositories (列出仓库)
    ├── GET  /repositories/:id (获取仓库)
    ├── POST /repositories (创建仓库)
    ├── POST /search (搜索代码)
    ├── GET  /symbols/:id/callers (获取调用者)
    ├── GET  /symbols/:id/callees (获取被调用方)
    ├── GET  /symbols/:id/dependencies (获取依赖)
    ├── GET  /files/:id/symbols (获取文件符号)
    ├── POST /files (创建文件)
    └── POST /commits (创建提交)
```

### 路由注册

```go
func (s *Server) RegisterRoutes(r *gin.Engine) {
    // Health check (no auth required)
    r.GET("/health", s.healthCheck)

    // API v1 routes
    v1 := r.Group("/api/v1")
    {
        // Index
        v1.POST("/index", s.indexHandler.Index)

        // Repositories
        v1.GET("/repositories", s.repoHandler.GetAll)
        v1.GET("/repositories/:id", s.repoHandler.GetByID)
        v1.POST("/repositories", s.createRepository)

        // Search
        v1.POST("/search", s.searchHandler.Search)

        // Relationships
        v1.GET("/symbols/:id/callers", s.relationshipHandler.GetCallers)
        v1.GET("/symbols/:id/callees", s.relationshipHandler.GetCallees)
        v1.GET("/symbols/:id/dependencies", s.relationshipHandler.GetDependencies)
        v1.GET("/files/:id/symbols", s.relationshipHandler.GetFileSymbols)

        // Files
        v1.POST("/files", s.createFile)

        // Commits
        v1.POST("/commits", s.createCommit)
    }
}
```

### 路由参数

路径参数通过 `c.Param()` 获取:

```go
// GET /api/v1/repositories/:id
func (h *RepositoryHandler) GetByID(c *gin.Context) {
    repoID := c.Param("id")  // 获取路径参数
    // ...
}
```

查询参数通过 `c.Query()` 获取:

```go
// GET /api/v1/repositories?limit=10&offset=20
func (h *RepositoryHandler) GetAll(c *gin.Context) {
    limit := c.Query("limit")
    offset := c.Query("offset")
    // ...
}
```

## Handler 实现

### IndexHandler

**功能**: 接收解析结果并索引到数据库。

**请求示例**:

```json
POST /api/v1/index
{
  "repo_name": "myproject",
  "repo_url": "https://github.com/user/myproject",
  "branch": "main",
  "parse_output": {
    "files": [
      {
        "path": "main.go",
        "language": "Go",
        "symbols": [...],
        "dependencies": [...]
      }
    ]
  },
  "options": {
    "incremental": true,
    "skip_vectors": false,
    "batch_size": 100,
    "worker_count": 4
  }
}
```

**响应示例**:

```json
{
  "repo_id": "uuid",
  "status": "success",
  "files_processed": 42,
  "symbols_created": 256,
  "edges_created": 384,
  "vectors_created": 256,
  "duration": "2.5s",
  "errors": []
}
```

**实现要点**:

```go
func (h *IndexHandler) Index(c *gin.Context) {
    // 1. 绑定和验证请求
    var req IndexRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 2. 创建 Indexer
    config := &indexer.IndexerConfig{...}
    idx := indexer.NewIndexer(h.db, config)

    // 3. 执行索引
    ctx := context.Background()
    result, err := idx.Index(ctx, &req.ParseOutput)

    // 4. 返回响应
    c.JSON(statusCode, IndexResponse{...})
}
```

### RepositoryHandler

**功能**: 仓库的 CRUD 操作。

**GetAll 实现**:

```go
func (h *RepositoryHandler) GetAll(c *gin.Context) {
    ctx := context.Background()

    repos, err := h.repoRepository.GetAll(ctx)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to retrieve repositories",
            "details": err.Error(),
        })
        return
    }

    // 转换为响应格式
    response := ListRepositoriesResponse{
        Repositories: make([]RepositoryResponse, len(repos)),
        Total:        len(repos),
    }

    for i, repo := range repos {
        response.Repositories[i] = RepositoryResponse{
            RepoID:     repo.RepoID,
            Name:       repo.Name,
            URL:        repo.URL,
            Branch:     repo.Branch,
            // ...
        }
    }

    c.JSON(http.StatusOK, response)
}
```

**GetByID 实现**:

```go
func (h *RepositoryHandler) GetByID(c *gin.Context) {
    repoID := c.Param("id")
    if repoID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Repository ID is required",
        })
        return
    }

    ctx := context.Background()
    repo, err := h.repoRepository.GetByID(ctx, repoID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to retrieve repository",
            "details": err.Error(),
        })
        return
    }

    if repo == nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error": "Repository not found",
        })
        return
    }

    response := RepositoryResponse{...}
    c.JSON(http.StatusOK, response)
}
```

### SearchHandler

**功能**: 基于语义相似度的代码搜索。

**请求示例**:

```json
POST /api/v1/search
{
  "query": "how to connect to database",
  "repo_id": "optional-uuid",
  "language": "Go",
  "kind": ["function", "method"],
  "limit": 10
}
```

**实现流程**:

```go
func (h *SearchHandler) Search(c *gin.Context) {
    // 1. 解析请求
    var req SearchRequest
    c.ShouldBindJSON(&req)

    // 2. 生成查询向量
    ctx := context.Background()
    embedding, err := h.embedder.GenerateEmbedding(ctx, req.Query)

    // 3. 向量相似度搜索
    vectorResults, err := h.vectorRepo.SimilaritySearchWithFilters(ctx, embedding, filters)

    // 4. 获取符号详情并应用过滤器
    results := make([]SearchResult, 0)
    for _, vr := range vectorResults {
        symbol, _ := h.symbolRepo.GetByID(ctx, vr.EntityID)

        // 应用类型、语言、仓库过滤器
        if matchesFilters(symbol, req) {
            results = append(results, SearchResult{
                SymbolID:   symbol.SymbolID,
                Name:       symbol.Name,
                Similarity: vr.Similarity,
                // ...
            })
        }
    }

    c.JSON(http.StatusOK, SearchResponse{Results: results})
}
```

### RelationshipHandler

**功能**: 查询代码符号之间的调用关系和依赖关系。

#### GetCallers

查找调用指定符号的所有函数:

```go
func (h *RelationshipHandler) GetCallers(c *gin.Context) {
    symbolID := c.Param("id")

    // 使用 AGE Cypher 查询
    query := `
        SELECT * FROM cypher('code_graph', $$
            MATCH (caller)-[r:CALLS]->(callee)
            WHERE callee.symbol_id = $symbol_id
            RETURN caller.symbol_id, caller.name, caller.kind, ...
        $$) AS (...)
    `

    rows, _ := h.db.QueryContext(ctx, query)
    // 处理结果...

    // 如果 AGE 不可用,回退到 SQL
    h.getCallersSQL(c, symbolID)
}
```

#### GetCallees

查找指定符号调用的所有函数:

```go
func (h *RelationshipHandler) GetCallees(c *gin.Context) {
    // 类似 GetCallers,但查询方向相反
    query := `
        MATCH (caller)-[r:CALLS]->(callee)
        WHERE caller.symbol_id = $symbol_id
        RETURN callee.symbol_id, callee.name, ...
    `
}
```

#### GetDependencies

查找指定符号的所有依赖关系 (import, extends, implements):

```go
func (h *RelationshipHandler) GetDependencies(c *gin.Context) {
    query := `
        MATCH (source)-[r]->(target)
        WHERE source.symbol_id = $symbol_id
          AND type(r) IN ['IMPORTS', 'EXTENDS', 'IMPLEMENTS', 'REFERENCES']
        RETURN target.symbol_id, type(r), ...
    `
}
```

## 中间件机制

中间件按以下顺序执行:

```
Request → Recovery → Logging → CORS → Auth → Handler
```

### Recovery 中间件

Gin 内置的 panic 恢复中间件:

```go
r.Use(gin.Recovery())
```

**功能**: 捕获 panic,返回 500 错误,防止服务器崩溃。

### Logging 中间件

记录所有 HTTP 请求:

```go
r.Use(middleware.Logging())
```

**日志格式**:

```
INFO  HTTP request method=POST path=/api/v1/search status=200 latency=45ms client_ip=127.0.0.1
WARN  HTTP request client error method=GET path=/api/v1/repositories/invalid status=404 latency=12ms
ERROR HTTP request failed method=POST path=/api/v1/index status=500 latency=1.2s error_count=1
```

**配置**:

```go
// 使用默认 logger
middleware.Logging()

// 使用自定义 logger
logger := utils.NewLogger(true) // verbose
middleware.LoggingWithLogger(logger)
```

### CORS 中间件

处理跨域请求:

```go
corsConfig := middleware.NewCORSConfig([]string{"*"})
r.Use(middleware.CORS(corsConfig))
```

**配置选项**:

```go
type CORSConfig struct {
    AllowedOrigins []string  // 允许的源
    AllowedMethods []string  // 允许的方法: GET, POST, PUT, DELETE, OPTIONS
    AllowedHeaders []string  // 允许的头: Origin, Content-Type, Accept, Authorization
    AllowAll       bool      // 是否允许所有源 (当 origins 包含 "*")
}
```

**示例**:

```go
// 允许所有源 (开发环境)
corsConfig := middleware.NewCORSConfig([]string{"*"})

// 允许特定源 (生产环境)
corsConfig := middleware.NewCORSConfig([]string{
    "https://codeatlas.example.com",
    "https://app.example.com",
})
```

### Auth 中间件

可选的 Bearer Token 认证:

```go
authConfig := middleware.NewAuthConfig(true, []string{"secret-token"})
r.Use(middleware.Auth(authConfig))
```

**配置**:

```go
type AuthConfig struct {
    Enabled bool              // 是否启用认证
    Tokens  map[string]bool   // 有效的 token 集合
}
```

**使用方法**:

```bash
# 不带 token (认证关闭时)
curl http://localhost:8080/api/v1/repositories

# 带 token (认证开启时)
curl -H "Authorization: Bearer secret-token" http://localhost:8080/api/v1/repositories
```

**跳过认证的端点**:

- `/health` - 健康检查端点始终不需要认证

## 错误处理

### 错误响应格式

所有错误响应遵循统一格式:

```json
{
  "error": "错误类型描述",
  "details": "详细错误信息"
}
```

### HTTP 状态码

| 状态码 | 场景 | 示例 |
|--------|------|------|
| **200** | 成功 | `GET /api/v1/repositories` |
| **207** | 部分成功 | 索引时有部分文件失败 |
| **400** | 请求错误 | 缺少必需字段、格式错误 |
| **401** | 未认证 | 缺少或无效的 token |
| **404** | 资源不存在 | 仓库 ID 不存在 |
| **500** | 服务器错误 | 数据库连接失败 |
| **503** | 服务不可用 | 向量服务不可用 |

### 错误处理模式

#### 1. 请求验证错误

```go
var req SearchRequest
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
        "error":   "Invalid request body",
        "details": err.Error(),
    })
    return
}
```

#### 2. 资源不存在

```go
repo, err := h.repoRepository.GetByID(ctx, repoID)
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{
        "error":   "Failed to retrieve repository",
        "details": err.Error(),
    })
    return
}

if repo == nil {
    c.JSON(http.StatusNotFound, gin.H{
        "error": "Repository not found",
    })
    return
}
```

#### 3. 服务不可用

```go
embedding, err := h.embedder.GenerateEmbedding(ctx, req.Query)
if err != nil {
    c.JSON(http.StatusServiceUnavailable, gin.H{
        "error":   "Embedding service unavailable",
        "details": err.Error(),
    })
    return
}
```

#### 4. 部分成功

```go
statusCode := http.StatusOK
switch result.Status {
case "success":
    statusCode = http.StatusOK
case "partial_success":
    statusCode = http.StatusMultiStatus  // 207
case "success_with_warnings":
    statusCode = http.StatusOK
case "failed":
    statusCode = http.StatusInternalServerError
}

c.JSON(statusCode, response)
```

## 数据验证

### Gin 绑定验证

使用 struct tags 进行验证:

```go
type SearchRequest struct {
    Query    string   `json:"query" binding:"required"`
    RepoID   string   `json:"repo_id,omitempty"`
    Language string   `json:"language,omitempty"`
    Kind     []string `json:"kind,omitempty"`
    Limit    int      `json:"limit,omitempty"`
}

type IndexRequest struct {
    RepoID      string              `json:"repo_id,omitempty"`
    RepoName    string              `json:"repo_name" binding:"required"`
    RepoURL     string              `json:"repo_url,omitempty"`
    Branch      string              `json:"branch,omitempty"`
    ParseOutput schema.ParseOutput  `json:"parse_output" binding:"required"`
    Options     IndexOptions        `json:"options,omitempty"`
}
```

**常用验证标签**:

| 标签 | 说明 | 示例 |
|------|------|------|
| `required` | 必填字段 | `binding:"required"` |
| `omitempty` | 可选字段 | `json:"field,omitempty"` |
| `min` | 最小值 | `binding:"min=1"` |
| `max` | 最大值 | `binding:"max=100"` |
| `email` | 邮箱格式 | `binding:"required,email"` |

### 自定义验证

在 Handler 中添加额外的业务逻辑验证:

```go
// 验证 parse_output 不为空
if len(req.ParseOutput.Files) == 0 {
    c.JSON(http.StatusBadRequest, gin.H{
        "error": "Parse output must contain at least one file",
    })
    return
}

// 验证 symbol_id 格式
symbolID := c.Param("id")
if symbolID == "" {
    c.JSON(http.StatusBadRequest, gin.H{
        "error": "Symbol ID is required",
    })
    return
}
```

### 参数验证示例

```go
func (h *RepositoryHandler) GetByID(c *gin.Context) {
    // 1. 路径参数验证
    repoID := c.Param("id")
    if repoID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Repository ID is required",
        })
        return
    }

    // 2. UUID 格式验证 (可选)
    if _, err := uuid.Parse(repoID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid repository ID format",
        })
        return
    }

    // ... 继续处理
}
```

## 性能优化

### 1. 数据库连接池

```go
// 在 DB 初始化时配置
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)
```

### 2. 上下文超时

为长时间操作设置超时:

```go
func (h *SearchHandler) Search(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 使用 ctx 进行数据库查询
    vectorResults, err := h.vectorRepo.SimilaritySearchWithFilters(ctx, embedding, filters)
    // ...
}
```

### 3. 批量操作

索引时使用批量操作:

```go
// 配置批量大小
config := &indexer.IndexerConfig{
    BatchSize:   100,  // 每批处理 100 个符号
    WorkerCount: 4,    // 4 个并发 worker
}
```

### 4. 查询优化

#### 使用索引

```sql
-- 确保查询字段有索引
CREATE INDEX idx_symbols_file_id ON symbols(file_id);
CREATE INDEX idx_edges_source_id ON edges(source_id);
CREATE INDEX idx_edges_target_id ON edges(target_id);
```

#### 避免 N+1 查询

```go
// ❌ 不好的做法: N+1 查询
for _, edge := range edges {
    symbol, _ := h.symbolRepo.GetByID(ctx, edge.TargetID)
    // 每次循环都查询一次数据库
}

// ✅ 好的做法: 批量查询
symbolIDs := getSymbolIDs(edges)
symbols, _ := h.symbolRepo.GetByIDs(ctx, symbolIDs)
// 一次查询获取所有符号
```

### 5. 向量检索优化

```go
// 限制返回数量
filters := models.VectorSearchFilters{
    EntityType: "symbol",
    Limit:      10,  // 最多返回 10 个结果
}

// 使用 IVFFlat 索引 (已配置)
-- CREATE INDEX ON vectors USING ivfflat (embedding vector_cosine_ops);
```

### 6. 缓存策略 (未来)

可以考虑添加缓存层:

```go
// 伪代码
func (h *RepositoryHandler) GetByID(c *gin.Context) {
    // 1. 尝试从缓存获取
    if cached, found := cache.Get("repo:" + repoID); found {
        return cached
    }

    // 2. 从数据库获取
    repo, _ := h.repoRepository.GetByID(ctx, repoID)

    // 3. 写入缓存
    cache.Set("repo:"+repoID, repo, 5*time.Minute)

    return repo
}
```

## 扩展指南

### 添加新端点

#### 步骤 1: 创建 Handler

在 `handlers/` 目录创建新文件:

```go
// internal/api/handlers/statistics_handler.go

package handlers

import (
    "context"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/yourtionguo/CodeAtlas/pkg/models"
)

type StatisticsHandler struct {
    db         *models.DB
    symbolRepo *models.SymbolRepository
    fileRepo   *models.FileRepository
}

func NewStatisticsHandler(db *models.DB) *StatisticsHandler {
    return &StatisticsHandler{
        db:         db,
        symbolRepo: models.NewSymbolRepository(db),
        fileRepo:   models.NewFileRepository(db),
    }
}

type StatisticsResponse struct {
    TotalRepositories int `json:"total_repositories"`
    TotalFiles       int `json:"total_files"`
    TotalSymbols     int `json:"total_symbols"`
}

// GetStatistics returns overall statistics
func (h *StatisticsHandler) GetStatistics(c *gin.Context) {
    ctx := context.Background()

    // 查询统计数据
    repoCount, _ := h.repoRepository.Count(ctx)
    fileCount, _ := h.fileRepository.Count(ctx)
    symbolCount, _ := h.symbolRepository.Count(ctx)

    response := StatisticsResponse{
        TotalRepositories: repoCount,
        TotalFiles:       fileCount,
        TotalSymbols:     symbolCount,
    }

    c.JSON(http.StatusOK, response)
}
```

#### 步骤 2: 在 Server 中注册

```go
// internal/api/server.go

type Server struct {
    // ... 现有字段
    statisticsHandler *handlers.StatisticsHandler
}

func NewServer(db *models.DB, config *ServerConfig) *Server {
    return &Server{
        // ... 现有初始化
        statisticsHandler: handlers.NewStatisticsHandler(db),
    }
}

func (s *Server) RegisterRoutes(r *gin.Engine) {
    // ... 现有路由

    // 新增统计端点
    v1.GET("/statistics", s.statisticsHandler.GetStatistics)
}
```

#### 步骤 3: 添加测试

```go
// internal/api/handlers/statistics_handler_test.go

package handlers

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestStatisticsHandler_GetStatistics(t *testing.T) {
    // 设置测试数据库
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    handler := NewStatisticsHandler(db)

    // 创建测试请求
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)

    // 调用 handler
    handler.GetStatistics(c)

    // 验证响应
    assert.Equal(t, http.StatusOK, w.Code)

    var response StatisticsResponse
    json.Unmarshal(w.Body.Bytes(), &response)

    assert.Greater(t, response.TotalRepositories, 0)
}
```

#### 步骤 4: 更新文档

在 API 文档中添加新端点说明。

### 添加新中间件

#### 创建自定义中间件

```go
// internal/api/middleware/rate_limit.go

package middleware

import (
    "net/http"
    "sync"
    "time"
    "github.com/gin-gonic/gin"
)

type RateLimiter struct {
    visitors map[string]*Visitor
    mu       sync.RWMutex
    rate     int           // 每分钟请求数
    burst    int           // 突发请求数
}

type Visitor struct {
    tokens    int
    lastSeen  time.Time
}

func NewRateLimiter(rate, burst int) *RateLimiter {
    rl := &RateLimiter{
        visitors: make(map[string]*Visitor),
        rate:     rate,
        burst:    burst,
    }

    // 清理过期访客
    go rl.cleanupVisitors()

    return rl
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()

        rl.mu.Lock()
        visitor, exists := rl.visitors[ip]
        if !exists {
            visitor = &Visitor{tokens: rl.burst - 1, lastSeen: time.Now()}
            rl.visitors[ip] = visitor
        } else {
            // 补充 tokens
            elapsed := time.Since(visitor.lastSeen)
            visitor.tokens += int(elapsed.Minutes()) * rl.rate
            if visitor.tokens > rl.burst {
                visitor.tokens = rl.burst
            }
            visitor.lastSeen = time.Now()
        }

        if visitor.tokens <= 0 {
            rl.mu.Unlock()
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "Rate limit exceeded",
            })
            c.Abort()
            return
        }

        visitor.tokens--
        rl.mu.Unlock()

        c.Next()
    }
}

func (rl *RateLimiter) cleanupVisitors() {
    for {
        time.Sleep(time.Minute)
        rl.mu.Lock()
        for ip, visitor := range rl.visitors {
            if time.Since(visitor.lastSeen) > 3*time.Minute {
                delete(rl.visitors, ip)
            }
        }
        rl.mu.Unlock()
    }
}
```

#### 注册中间件

```go
// internal/api/server.go

func (s *Server) SetupRouter() *gin.Engine {
    r := gin.New()

    r.Use(gin.Recovery())
    r.Use(middleware.Logging())
    r.Use(middleware.CORS(corsConfig))

    // 添加限流中间件
    rateLimiter := middleware.NewRateLimiter(60, 10) // 60 req/min, burst 10
    r.Use(rateLimiter.Middleware())

    // ... 其他配置

    return r
}
```

### 添加新的响应类型

当需要修改响应格式时,遵循以下模式:

```go
// 1. 定义响应结构
type DetailedSymbolResponse struct {
    SymbolID        string   `json:"symbol_id"`
    Name            string   `json:"name"`
    Kind            string   `json:"kind"`
    Signature       string   `json:"signature"`
    FilePath        string   `json:"file_path"`
    StartLine       int      `json:"start_line"`
    EndLine         int      `json:"end_line"`
    Docstring       string   `json:"docstring,omitempty"`
    Callers         []string `json:"callers,omitempty"`       // 新增
    Callees         []string `json:"callees,omitempty"`       // 新增
    RelatedSymbols  []string `json:"related_symbols,omitempty"` // 新增
}

// 2. 在 Handler 中构建响应
func (h *SymbolHandler) GetDetailed(c *gin.Context) {
    symbol, _ := h.symbolRepo.GetByID(ctx, symbolID)

    // 获取调用者
    callers, _ := h.edgeRepo.GetCallers(ctx, symbolID)

    // 获取被调用方
    callees, _ := h.edgeRepo.GetCallees(ctx, symbolID)

    // 构建响应
    response := DetailedSymbolResponse{
        SymbolID:       symbol.SymbolID,
        Name:           symbol.Name,
        Kind:           symbol.Kind,
        Callers:        callers,
        Callees:        callees,
        // ...
    }

    c.JSON(http.StatusOK, response)
}
```

## 测试

### Handler 测试示例

```go
func TestRepositoryHandler_GetAll(t *testing.T) {
    // 设置测试数据库
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    // 创建 handler
    handler := NewRepositoryHandler(db)

    // 创建测试上下文
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)

    // 调用 handler
    handler.GetAll(c)

    // 验证响应
    assert.Equal(t, http.StatusOK, w.Code)

    var response ListRepositoriesResponse
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Greater(t, response.Total, 0)
}
```

### 集成测试

```go
func TestAPIIntegration(t *testing.T) {
    // 启动测试服务器
    config := &ServerConfig{EnableAuth: false}
    server := NewServer(testDB, config)
    router := server.SetupRouter()

    ts := httptest.NewServer(router)
    defer ts.Close()

    // 测试索引端点
    resp := indexTestRepo(t, ts.URL+"/api/v1/index")
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // 测试搜索端点
    searchResp := searchCode(t, ts.URL+"/api/v1/search", "database")
    assert.Equal(t, http.StatusOK, searchResp.StatusCode)
}
```

## 监控和日志

### 结构化日志

```go
fields := []utils.Field{
    {Key: "endpoint", Value: "/api/v1/search"},
    {Key: "query_length", Value: len(req.Query)},
    {Key: "results_count", Value: len(results)},
    {Key: "search_duration", Value: duration},
}

logger.InfoWithFields("Search completed", fields...)
```

### 性能监控

```go
func (h *SearchHandler) Search(c *gin.Context) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        logger.Info("Search completed", "duration", duration)
    }()

    // 搜索逻辑...
}
```

### 错误追踪

```go
if err != nil {
    logger.Error("Database query failed",
        "query", query,
        "error", err,
        "symbol_id", symbolID,
    )
    c.JSON(http.StatusInternalServerError, gin.H{
        "error": "Internal server error",
    })
    return
}
```

## 安全建议

### 1. 认证

生产环境**必须启用认证**:

```go
config := &ServerConfig{
    EnableAuth: true,
    AuthTokens: []string{
        os.Getenv("API_TOKEN_1"),
        os.Getenv("API_TOKEN_2"),
    },
}
```

### 2. CORS

生产环境**不要使用通配符**:

```go
corsConfig := middleware.NewCORSConfig([]string{
    "https://your-frontend.example.com",
})
```

### 3. SQL 注入

使用**参数化查询**:

```go
// ✅ 好的做法
query := "SELECT * FROM repositories WHERE repo_id = $1"
db.QueryContext(ctx, query, repoID)

// ❌ 不好的做法
query := fmt.Sprintf("SELECT * FROM repositories WHERE repo_id = '%s'", repoID)
db.QueryContext(ctx, query)
```

### 4. 输入验证

始终验证用户输入:

```go
// 限制查询长度
if len(req.Query) > 1000 {
    c.JSON(http.StatusBadRequest, gin.H{
        "error": "Query too long (max 1000 characters)",
    })
    return
}

// 限制结果数量
if req.Limit > 100 {
    req.Limit = 100
}
```

## 参考资源

### 相关文档

- [Gin 框架文档](https://gin-gonic.com/docs/)
- [PostgreSQL AGE 文档](https://age.apache.org/)
- [pgvector 文档](https://github.com/pgvector/pgvector)
- [索引器文档](/docs/architecture.md)
- [API 端点文档](/docs/api.md)

### 相关代码

- [`server.go`](/internal/api/server.go) - 服务器主文件
- [`handlers/`](/internal/api/handlers/) - Handler 实现
- [`middleware/`](/internal/api/middleware/) - 中间件实现
- [`pkg/models/`](/pkg/models/) - 数据模型和仓库

## 常见问题

### Q: 如何添加新的认证方式?

A: 创建新的认证中间件替换 `middleware.Auth()`:

```go
func JWTAuth(config *JWTConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)
        claims, err := validateJWT(token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "Invalid token",
            })
            return
        }

        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

### Q: 如何处理大量数据的分页?

A: 使用 LIMIT 和 OFFSET:

```go
type ListRequest struct {
    Limit  int `json:"limit" binding:"min=1,max=100"`
    Offset int `json:"offset" binding:"min=0"`
}

// 设置默认值
if req.Limit == 0 {
    req.Limit = 20
}

// 传递给 repository
repos, err := h.repoRepository.GetPaginated(ctx, req.Limit, req.Offset)
```

### Q: AGE 图查询失败怎么办?

A: 所有关系查询都有 SQL 回退:

```go
rows, err := h.db.QueryContext(ctx, cypherQuery)
if err != nil {
    // 自动回退到 SQL
    h.getCallersSQL(c, symbolID)
    return
}
```

### Q: 如何调试 API 请求?

A: 使用日志中间件和 `curl`:

```bash
# 启用详细日志
export LOG_LEVEL=debug

# 使用 curl 测试
curl -v -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "database", "limit": 5}'
```
