# 架构设计

> CodeAtlas 系统架构和设计决策

## 系统概述

CodeAtlas 是一个智能代码知识图谱平台，结合了：
- **代码解析**：Tree-sitter 静态分析
- **语义理解**：LLM 增强的语义提取
- **向量检索**：pgvector 语义搜索
- **图查询**：PostgreSQL AGE 关系遍历

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                         用户层                               │
├─────────────┬─────────────┬─────────────┬──────────────────┤
│   CLI 工具   │   Web UI    │  REST API   │   第三方集成      │
└─────────────┴─────────────┴─────────────┴──────────────────┘
                              │
┌─────────────────────────────┼─────────────────────────────┐
│                         API 层                              │
├─────────────────────────────┴─────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │  仓库管理 │  │  代码搜索 │  │  关系查询 │  │  索引管理 │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
└───────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────┼─────────────────────────────┐
│                         服务层                              │
├─────────────────────────────┴─────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ 解析引擎  │  │ 图服务   │  │ 检索服务  │  │ QA 引擎  │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
└───────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────┼─────────────────────────────┐
│                         数据层                              │
├─────────────────────────────┴─────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐ │
│  │ PostgreSQL   │  │  pgvector    │  │  PostgreSQL AGE │ │
│  │ (关系数据)    │  │  (向量检索)   │  │  (图查询)        │ │
│  └──────────────┘  └──────────────┘  └─────────────────┘ │
└───────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. CLI 工具

**职责**：本地代码解析和上传

**主要功能**：
- 代码解析（parse 命令）
- 索引上传（index 命令）
- 增量更新

**技术栈**：
- Go 1.25+
- urfave/cli/v2
- Tree-sitter

**工作流程**：
```
1. 扫描文件系统
2. 过滤文件（语言、忽略规则）
3. 并发解析文件
4. 生成 JSON 输出
5. 上传到 API 服务器
```

### 2. 解析引擎

**职责**：代码静态分析和 AST 提取

**支持语言**：
- Go, JavaScript, TypeScript
- Python, Java, Kotlin
- Swift, Objective-C, C, C++

**解析器架构**：

```go
// 解析器接口
type Parser interface {
    Parse(content []byte) (*ParseResult, error)
    Language() string
}

// 解析结果
type ParseResult struct {
    Files         []File
    Symbols       []Symbol
    Relationships []Relationship
    Metadata      Metadata
}
```

**关键特性**：
- Tree-sitter 精确解析
- 增量解析支持
- 错误恢复机制
- 跨语言调用分析

**示例：Go 解析器**

```go
type GoParser struct {
    parser *sitter.Parser
}

func (p *GoParser) Parse(content []byte) (*ParseResult, error) {
    // 1. 解析 AST
    tree := p.parser.Parse(nil, content)
    
    // 2. 提取符号
    symbols := p.extractSymbols(tree.RootNode(), content)
    
    // 3. 提取关系
    relationships := p.extractRelationships(tree.RootNode(), content)
    
    return &ParseResult{
        Symbols:       symbols,
        Relationships: relationships,
    }, nil
}
```

### 3. API 服务

**职责**：提供 RESTful API

**技术栈**：
- Gin Web Framework
- PostgreSQL 驱动
- JWT 认证

**端点设计**：

```
POST   /api/v1/repositories          # 创建仓库
GET    /api/v1/repositories          # 列出仓库
GET    /api/v1/repositories/:id      # 获取仓库
DELETE /api/v1/repositories/:id      # 删除仓库

GET    /api/v1/search                # 搜索符号
POST   /api/v1/search/semantic       # 语义搜索

GET    /api/v1/relationships         # 查询关系
GET    /api/v1/dependencies          # 查询依赖

GET    /api/v1/files/:id             # 获取文件
GET    /api/v1/symbols/:id           # 获取符号
```

**中间件**：
- 日志记录
- 错误处理
- CORS
- 认证
- 速率限制

### 4. 图服务

**职责**：代码关系图构建和查询

**使用 PostgreSQL AGE**：

```sql
-- 创建图
SELECT * FROM ag_catalog.create_graph('code_graph');

-- 添加顶点（符号）
SELECT * FROM cypher('code_graph', $$
    CREATE (s:Symbol {
        id: 'uuid',
        name: 'main',
        kind: 'function'
    })
$$) as (v agtype);

-- 添加边（调用关系）
SELECT * FROM cypher('code_graph', $$
    MATCH (a:Symbol {id: 'uuid1'})
    MATCH (b:Symbol {id: 'uuid2'})
    CREATE (a)-[:CALLS]->(b)
$$) as (e agtype);

-- 查询调用链
SELECT * FROM cypher('code_graph', $$
    MATCH path = (a:Symbol {name: 'main'})-[:CALLS*1..3]->(b:Symbol)
    RETURN path
$$) as (path agtype);
```

**关系类型**：
- `CALLS` - 函数调用
- `IMPORTS` - 模块导入
- `EXTENDS` - 类继承
- `IMPLEMENTS` - 接口实现
- `REFERENCES` - 引用

### 5. 检索服务

**职责**：语义搜索和向量检索

**使用 pgvector**：

```sql
-- 创建向量表
CREATE TABLE vectors (
    id UUID PRIMARY KEY,
    symbol_id UUID REFERENCES symbols(id),
    embedding vector(1536),
    created_at TIMESTAMP DEFAULT NOW()
);

-- 创建向量索引
CREATE INDEX ON vectors USING ivfflat (embedding vector_cosine_ops);

-- 向量搜索
SELECT symbol_id, 1 - (embedding <=> query_vector) as similarity
FROM vectors
ORDER BY embedding <=> query_vector
LIMIT 10;
```

**向量生成**：

```go
// 使用 OpenAI API
func GenerateEmbedding(text string) ([]float32, error) {
    resp, err := openai.CreateEmbedding(
        context.Background(),
        openai.EmbeddingRequest{
            Model: "text-embedding-3-small",
            Input: text,
        },
    )
    return resp.Data[0].Embedding, err
}
```

## 数据模型

### 核心表结构

```sql
-- 仓库
CREATE TABLE repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    url TEXT,
    branch TEXT,
    commit_hash TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 文件
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    language TEXT NOT NULL,
    size INTEGER,
    checksum TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(repo_id, path)
);

-- 符号
CREATE TABLE symbols (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,  -- function, class, interface, variable
    signature TEXT,
    start_line INTEGER,
    end_line INTEGER,
    docstring TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 关系
CREATE TABLE relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID REFERENCES symbols(id) ON DELETE CASCADE,
    target_id UUID REFERENCES symbols(id) ON DELETE CASCADE,
    edge_type TEXT NOT NULL,  -- call, import, extends, implements
    created_at TIMESTAMP DEFAULT NOW()
);

-- 向量
CREATE TABLE vectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol_id UUID REFERENCES symbols(id) ON DELETE CASCADE,
    embedding vector(1536),
    created_at TIMESTAMP DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_files_repo_id ON files(repo_id);
CREATE INDEX idx_symbols_file_id ON symbols(file_id);
CREATE INDEX idx_symbols_name ON symbols(name);
CREATE INDEX idx_relationships_source ON relationships(source_id);
CREATE INDEX idx_relationships_target ON relationships(target_id);
CREATE INDEX idx_vectors_symbol ON vectors(symbol_id);
CREATE INDEX idx_vectors_embedding ON vectors USING ivfflat (embedding vector_cosine_ops);
```

### 数据流

```
1. 解析阶段
   CLI → 文件扫描 → Tree-sitter 解析 → JSON 输出

2. 索引阶段
   JSON → API 接收 → 数据库写入 → 向量生成 → 图构建

3. 查询阶段
   用户查询 → API 路由 → 服务处理 → 数据库查询 → 结果返回
```

## 设计决策

### 1. 为什么选择 Tree-sitter？

**优势**：
- ✅ 精确的语法解析
- ✅ 增量解析支持
- ✅ 错误恢复能力
- ✅ 多语言支持
- ✅ 高性能

**替代方案**：
- ❌ 正则表达式：不够精确
- ❌ 语言特定解析器：维护成本高
- ❌ LSP：需要语言服务器

### 2. 为什么选择 PostgreSQL？

**优势**：
- ✅ 成熟稳定
- ✅ 丰富的扩展（pgvector, AGE）
- ✅ 强大的查询能力
- ✅ ACID 保证
- ✅ 开源免费

**替代方案**：
- ❌ Neo4j：图查询强但向量支持弱
- ❌ Elasticsearch：搜索强但关系查询弱
- ❌ 多数据库：复杂度高

### 3. 为什么选择 Go？

**优势**：
- ✅ 高性能
- ✅ 并发支持好
- ✅ 静态类型
- ✅ 部署简单（单二进制）
- ✅ 丰富的生态

**替代方案**：
- ❌ Python：性能较低
- ❌ Rust：学习曲线陡
- ❌ Node.js：类型安全弱

### 4. 为什么使用 pgvector？

**优势**：
- ✅ 与 PostgreSQL 集成
- ✅ 支持多种距离度量
- ✅ 索引优化（IVFFlat, HNSW）
- ✅ 开源免费

**替代方案**：
- ❌ Pinecone：商业服务，成本高
- ❌ Milvus：独立部署，复杂度高
- ❌ Weaviate：功能过于复杂

## 性能优化

### 1. 解析性能

**策略**：
- 并发解析（worker pool）
- 增量解析（只解析变更文件）
- 缓存机制（文件 checksum）

**基准测试**：
```
BenchmarkGoParser-8     1000    1.2 ms/op    500 KB/op
BenchmarkJSParser-8     800     1.5 ms/op    600 KB/op
```

### 2. 数据库性能

**索引策略**：
```sql
-- B-tree 索引（精确查询）
CREATE INDEX idx_symbols_name ON symbols(name);

-- 向量索引（相似度搜索）
CREATE INDEX ON vectors USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- 部分索引（常用查询）
CREATE INDEX idx_active_repos ON repositories(id) 
WHERE deleted_at IS NULL;
```

**查询优化**：
```sql
-- 使用 CTE 优化复杂查询
WITH symbol_calls AS (
    SELECT source_id, target_id
    FROM relationships
    WHERE edge_type = 'call'
)
SELECT s.name, COUNT(*) as call_count
FROM symbols s
JOIN symbol_calls sc ON s.id = sc.source_id
GROUP BY s.name
ORDER BY call_count DESC
LIMIT 10;
```

### 3. API 性能

**缓存策略**：
- 内存缓存（热点数据）
- Redis 缓存（分布式）
- HTTP 缓存（ETag, Last-Modified）

**连接池**：
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

## 扩展性

### 水平扩展

```
┌─────────┐  ┌─────────┐  ┌─────────┐
│ API-1   │  │ API-2   │  │ API-3   │
└────┬────┘  └────┬────┘  └────┬────┘
     │            │            │
     └────────────┼────────────┘
                  │
          ┌───────┴───────┐
          │ Load Balancer │
          └───────┬───────┘
                  │
          ┌───────┴───────┐
          │  PostgreSQL   │
          │  (Primary)    │
          └───────┬───────┘
                  │
     ┌────────────┼────────────┐
     │            │            │
┌────┴────┐  ┌───┴────┐  ┌───┴────┐
│Replica-1│  │Replica-2│  │Replica-3│
└─────────┘  └─────────┘  └─────────┘
```

### 数据分片

```sql
-- 按仓库分片
CREATE TABLE symbols_shard_1 (
    CHECK (repo_id >= '00000000-0000-0000-0000-000000000000' 
       AND repo_id < '55555555-5555-5555-5555-555555555555')
) INHERITS (symbols);

CREATE TABLE symbols_shard_2 (
    CHECK (repo_id >= '55555555-5555-5555-5555-555555555555')
) INHERITS (symbols);
```

## 安全性

### 1. 认证和授权

```go
// JWT 认证
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        claims, err := ValidateJWT(token)
        if err != nil {
            c.AbortWithStatus(401)
            return
        }
        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

### 2. SQL 注入防护

```go
// 使用参数化查询
db.Query("SELECT * FROM symbols WHERE name = $1", name)

// 避免字符串拼接
// ❌ db.Query("SELECT * FROM symbols WHERE name = '" + name + "'")
```

### 3. 速率限制

```go
// 使用 rate limiter
limiter := rate.NewLimiter(rate.Limit(60), 60) // 60 req/min

func RateLimitMiddleware(limiter *rate.Limiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatus(429)
            return
        }
        c.Next()
    }
}
```

## 监控和可观测性

### 指标收集

```go
// Prometheus 指标
var (
    parseRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "parse_requests_total",
            Help: "Total number of parse requests",
        },
        []string{"language", "status"},
    )
    
    parseDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "parse_duration_seconds",
            Help: "Parse duration in seconds",
        },
        []string{"language"},
    )
)
```

### 日志记录

```go
// 结构化日志
log.WithFields(log.Fields{
    "repo_id": repoID,
    "file_count": fileCount,
    "duration": duration,
}).Info("Repository indexed successfully")
```

## 未来规划

### 短期（3-6 月）
- [ ] 增量索引优化
- [ ] 更多语言支持（Rust, Ruby）
- [ ] 实时索引更新
- [ ] Web UI 改进

### 中期（6-12 月）
- [ ] 分布式解析
- [ ] 多租户支持
- [ ] 高级图查询
- [ ] AI 辅助代码理解

### 长期（12+ 月）
- [ ] 代码生成
- [ ] 自动重构建议
- [ ] 漏洞检测
- [ ] 性能分析

## 参考资源

### 技术文档
- [Tree-sitter](https://tree-sitter.github.io/tree-sitter/)
- [pgvector](https://github.com/pgvector/pgvector)
- [PostgreSQL AGE](https://age.apache.org/)
- [Gin Framework](https://gin-gonic.com/)

### 相关项目
- [Sourcegraph](https://sourcegraph.com/)
- [GitHub Code Search](https://github.com/features/code-search)
- [Kythe](https://kythe.io/)

## 下一步

- 查看 [开发指南](development.md) 开始开发
- 查看 [API 文档](api.md) 了解 API 使用
- 查看 [部署指南](deployment.md) 部署到生产环境
