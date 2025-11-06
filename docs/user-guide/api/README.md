# API 完整指南

> CodeAtlas HTTP API 使用手册

## 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [认证](#认证)
- [端点参考](#端点参考)
- [搜索和关系查询](#搜索和关系查询)
- [错误处理](#错误处理)
- [集成示例](#集成示例)

## 概述

CodeAtlas API 提供 RESTful HTTP 接口，用于：

- **代码索引**：将解析的代码索引到知识图谱
- **语义搜索**：使用自然语言搜索代码
- **关系查询**：查询调用关系、依赖关系
- **仓库管理**：管理代码仓库

### 基础信息

- **Base URL**: `http://localhost:8080`
- **API 版本**: v1
- **数据格式**: JSON
- **认证方式**: Bearer Token（可选）

### 架构

```
┌─────────────────────────────────────────────────────────────┐
│                         API Server                          │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                    Middleware                        │  │
│  │  • Recovery  • Logging  • CORS  • Authentication    │  │
│  └──────────────────────────────────────────────────────┘  │
│                            ↓                                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                      Handlers                        │  │
│  │  • Index  • Search  • Relationships  • Repositories │  │
│  └──────────────────────────────────────────────────────┘  │
│                            ↓                                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   Data Layer                         │  │
│  │  • Models  • Repositories  • Database Access        │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                      │
│  • pgvector (embeddings)  • AGE (graph)  • JSONB          │
└─────────────────────────────────────────────────────────────┘
```

## 快速开始

### 启动 API 服务器

```bash
# 启动数据库
make docker-up

# 运行 API 服务器
make run-api

# 或使用自定义配置
ENABLE_AUTH=true AUTH_TOKENS=dev-token make run-api
```

### 健康检查

```bash
curl http://localhost:8080/health
```

响应：
```json
{
  "status": "ok",
  "message": "CodeAtlas API server is running"
}
```

### 基本请求示例

```bash
# 列出所有仓库
curl http://localhost:8080/api/v1/repositories

# 创建仓库
curl -X POST http://localhost:8080/api/v1/repositories \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-project",
    "url": "https://github.com/user/my-project",
    "branch": "main"
  }'
```

## 认证

### 启用认证

在环境变量中配置：

```bash
export ENABLE_AUTH=true
export AUTH_TOKENS="token1,token2,token3"
```

### 使用认证

在请求头中包含 Bearer Token：

```bash
curl -H "Authorization: Bearer token1" \
  http://localhost:8080/api/v1/repositories
```

### 生成安全令牌

```bash
# 使用 openssl
openssl rand -hex 32

# 使用 uuidgen
uuidgen
```

## 端点参考

### 健康检查

#### GET /health

检查 API 服务器是否运行。无需认证。

**响应**：
```json
{
  "status": "ok",
  "message": "CodeAtlas API server is running"
}
```

---

### 仓库管理

#### GET /api/v1/repositories

列出所有仓库。

**响应 (200 OK)**：
```json
{
  "repositories": [
    {
      "repo_id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "my-project",
      "url": "https://github.com/user/repo",
      "branch": "main",
      "commit_hash": "abc123",
      "created_at": "2025-11-06T10:00:00Z",
      "updated_at": "2025-11-06T12:00:00Z"
    }
  ],
  "total": 1
}
```

#### GET /api/v1/repositories/:id

获取特定仓库。

**路径参数**：
- `id`: 仓库 UUID

**响应 (200 OK)**：
```json
{
  "repo_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "my-project",
  "url": "https://github.com/user/repo",
  "branch": "main",
  "commit_hash": "abc123",
  "created_at": "2025-11-06T10:00:00Z",
  "updated_at": "2025-11-06T12:00:00Z"
}
```

**响应 (404 Not Found)**：
```json
{
  "error": "Repository not found"
}
```

#### POST /api/v1/repositories

创建新仓库。

**请求体**：
```json
{
  "name": "my-project",
  "url": "https://github.com/user/repo",
  "branch": "main"
}
```

**必需字段**：
- `name`: 仓库名称

**可选字段**：
- `url`: 仓库 URL
- `branch`: 分支名（默认 "main"）

**响应 (201 Created)**：
```json
{
  "repo_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "my-project",
  "url": "https://github.com/user/repo",
  "branch": "main",
  "created_at": "2025-11-06T10:00:00Z"
}
```

---

### 代码索引

#### POST /api/v1/index

将解析的代码索引到知识图谱。

**请求体**：
```json
{
  "repo_name": "my-project",
  "repo_url": "https://github.com/user/repo",
  "branch": "main",
  "commit_hash": "abc123",
  "parse_output": {
    "files": [
      {
        "path": "src/main.go",
        "language": "go",
        "size": 1024,
        "checksum": "sha256:...",
        "symbols": [
          {
            "name": "main",
            "kind": "function",
            "signature": "func main()",
            "start_line": 10,
            "end_line": 20,
            "docstring": "Main entry point"
          }
        ],
        "edges": [
          {
            "source_symbol": "main",
            "target_symbol": "initServer",
            "edge_type": "call"
          }
        ]
      }
    ]
  },
  "options": {
    "incremental": false,
    "skip_vectors": false,
    "batch_size": 100
  }
}
```

**必需字段**：
- `repo_name`: 仓库名称
- `parse_output`: 解析输出
- `parse_output.files`: 至少一个文件

**可选字段**：
- `repo_url`: 仓库 URL
- `branch`: 分支名（默认 "main"）
- `commit_hash`: 提交哈希
- `options`: 索引选项

**响应 (200 OK)**：
```json
{
  "repo_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "success",
  "files_processed": 10,
  "symbols_created": 45,
  "edges_created": 78,
  "vectors_created": 45,
  "errors": [],
  "duration": "2.5s"
}
```

**状态值**：
- `success`: 所有操作成功完成
- `partial_success`: 部分操作失败但索引继续
- `success_with_warnings`: 完成但有非关键警告
- `failed`: 索引失败

---

### 语义搜索

#### POST /api/v1/search

使用自然语言搜索代码符号。

**请求体**：
```json
{
  "query": "authentication middleware",
  "repo_id": "550e8400-e29b-41d4-a716-446655440000",
  "language": "go",
  "kind": ["function", "class"],
  "limit": 10
}
```

**必需字段**：
- `query`: 搜索查询文本（自然语言）

**可选字段**：
- `repo_id`: 按仓库过滤
- `language`: 按编程语言过滤
- `kind`: 按符号类型过滤
- `limit`: 最大结果数（默认 10）

**响应 (200 OK)**：
```json
{
  "results": [
    {
      "symbol_id": "770e8400-e29b-41d4-a716-446655440002",
      "name": "AuthMiddleware",
      "kind": "function",
      "signature": "func AuthMiddleware() gin.HandlerFunc",
      "file_path": "internal/api/middleware/auth.go",
      "docstring": "AuthMiddleware provides token-based authentication",
      "similarity": 0.92
    }
  ],
  "total": 1
}
```

**工作原理**：
1. API 接收自然语言查询
2. 使用配置的嵌入服务生成向量
3. 对索引的代码符号执行向量相似度搜索
4. 返回按相似度排序的结果

**配置**：

搜索 API 使用通过环境变量配置的嵌入服务。详见[配置指南](../../configuration/README.md)。

默认配置：
```bash
EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
EMBEDDING_DIMENSIONS=1024
```

使用 OpenAI：
```bash
export EMBEDDING_API_ENDPOINT=https://api.openai.com/v1/embeddings
export EMBEDDING_API_KEY=sk-...
export EMBEDDING_MODEL=text-embedding-3-small
export EMBEDDING_DIMENSIONS=1536
```

---

### 关系查询

#### GET /api/v1/symbols/:id/callers

获取调用指定符号的所有函数。

**路径参数**：
- `id`: 符号 UUID

**响应 (200 OK)**：
```json
{
  "symbols": [
    {
      "symbol_id": "880e8400-e29b-41d4-a716-446655440003",
      "name": "SetupRouter",
      "kind": "function",
      "file_path": "internal/api/server.go",
      "signature": "func (s *Server) SetupRouter() *gin.Engine"
    }
  ],
  "total": 1
}
```

#### GET /api/v1/symbols/:id/callees

获取指定符号调用的所有函数。

**路径参数**：
- `id`: 符号 UUID

**响应 (200 OK)**：
```json
{
  "symbols": [
    {
      "symbol_id": "990e8400-e29b-41d4-a716-446655440004",
      "name": "validateToken",
      "kind": "function",
      "file_path": "internal/api/middleware/auth.go",
      "signature": "func validateToken(token string) bool"
    }
  ],
  "total": 1
}
```

#### GET /api/v1/symbols/:id/dependencies

获取指定符号的所有依赖（导入、继承、实现）。

**路径参数**：
- `id`: 符号 UUID

**响应 (200 OK)**：
```json
{
  "dependencies": [
    {
      "symbol_id": "aa0e8400-e29b-41d4-a716-446655440005",
      "name": "gin.HandlerFunc",
      "kind": "type",
      "file_path": "vendor/github.com/gin-gonic/gin/context.go",
      "module": "github.com/gin-gonic/gin",
      "edge_type": "import",
      "signature": "type HandlerFunc func(*Context)"
    }
  ],
  "total": 1
}
```

#### GET /api/v1/files/:id/symbols

获取文件中定义的所有符号。

**路径参数**：
- `id`: 文件 UUID

**响应 (200 OK)**：
```json
{
  "symbols": [
    {
      "symbol_id": "bb0e8400-e29b-41d4-a716-446655440006",
      "name": "AuthMiddleware",
      "kind": "function",
      "signature": "func AuthMiddleware() gin.HandlerFunc",
      "start_line": 10,
      "end_line": 50,
      "docstring": "AuthMiddleware provides token-based authentication"
    }
  ],
  "total": 1
}
```

## 搜索和关系查询

### 语义搜索最佳实践

#### 1. 使用描述性查询

```bash
# 好的查询
"function that validates user authentication tokens"
"class for handling HTTP requests"
"method to parse JSON configuration files"

# 不太好的查询
"auth"
"http"
"json"
```

#### 2. 使用过滤器

```bash
# 只搜索 Go 函数
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "authentication",
    "language": "go",
    "kind": ["function"],
    "limit": 5
  }'
```

#### 3. 调整结果数量

```bash
# 获取更多结果
{
  "query": "database connection",
  "limit": 20
}
```

### 关系查询模式

#### 模式 1: 查找函数的调用者

```bash
# 1. 搜索函数
curl -X POST http://localhost:8080/api/v1/search \
  -d '{"query": "validateToken", "kind": ["function"]}'

# 2. 获取调用者
curl http://localhost:8080/api/v1/symbols/$SYMBOL_ID/callers
```

#### 模式 2: 追踪调用链

```bash
# 1. 获取函数调用的其他函数
curl http://localhost:8080/api/v1/symbols/$SYMBOL_ID/callees

# 2. 递归获取每个被调用函数的调用者
# 构建完整的调用图
```

#### 模式 3: 分析依赖关系

```bash
# 1. 获取符号的依赖
curl http://localhost:8080/api/v1/symbols/$SYMBOL_ID/dependencies

# 2. 分析外部依赖
# 过滤 edge_type == "import"
```

## 错误处理

### 常见响应码

| 代码 | 说明 |
|------|-------------|
| 200 | 成功 |
| 201 | 已创建 |
| 204 | 无内容 |
| 400 | 错误请求 - 无效输入 |
| 401 | 未授权 - 缺少或无效令牌 |
| 404 | 未找到 |
| 500 | 内部服务器错误 |

### 错误响应格式

#### 400 Bad Request

```json
{
  "error": "Invalid request body",
  "details": "missing required field: repo_name"
}
```

#### 401 Unauthorized

```json
{
  "error": "Missing authorization header"
}
```

```json
{
  "error": "Invalid authorization header format. Expected: Bearer <token>"
}
```

```json
{
  "error": "Invalid or expired token"
}
```

#### 404 Not Found

```json
{
  "error": "Repository not found"
}
```

#### 500 Internal Server Error

```json
{
  "error": "Failed to retrieve repositories",
  "details": "database connection failed"
}
```

### 错误处理最佳实践

#### 1. 检查响应状态码

```javascript
const response = await fetch('http://localhost:8080/api/v1/repositories');

if (!response.ok) {
  const error = await response.json();
  console.error('API Error:', error);
  throw new Error(error.error);
}

const data = await response.json();
```

#### 2. 实现重试逻辑

```javascript
async function fetchWithRetry(url, options, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch(url, options);
      if (response.ok) return await response.json();
      
      if (response.status >= 500) {
        // 服务器错误，重试
        await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)));
        continue;
      }
      
      // 客户端错误，不重试
      throw new Error(await response.text());
    } catch (error) {
      if (i === maxRetries - 1) throw error;
    }
  }
}
```

#### 3. 处理超时

```javascript
const controller = new AbortController();
const timeout = setTimeout(() => controller.abort(), 5000);

try {
  const response = await fetch(url, {
    signal: controller.signal
  });
  clearTimeout(timeout);
} catch (error) {
  if (error.name === 'AbortError') {
    console.error('Request timeout');
  }
}
```

## 集成示例

### CLI 集成

```bash
# 使用 CLI 上传仓库
./bin/cli index \
  --path /path/to/repo \
  --api-url http://localhost:8080 \
  --api-token your-token
```

### JavaScript/TypeScript

```typescript
// API 客户端类
class CodeAtlasClient {
  constructor(
    private baseUrl: string = 'http://localhost:8080',
    private token?: string
  ) {}

  private async request(endpoint: string, options: RequestInit = {}) {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error);
    }

    return response.json();
  }

  async listRepositories() {
    return this.request('/api/v1/repositories');
  }

  async search(query: string, options: {
    repo_id?: string;
    language?: string;
    kind?: string[];
    limit?: number;
  } = {}) {
    return this.request('/api/v1/search', {
      method: 'POST',
      body: JSON.stringify({ query, ...options }),
    });
  }

  async getSymbolCallers(symbolId: string) {
    return this.request(`/api/v1/symbols/${symbolId}/callers`);
  }
}

// 使用示例
const client = new CodeAtlasClient('http://localhost:8080', 'your-token');

// 搜索代码
const results = await client.search('authentication middleware', {
  language: 'go',
  kind: ['function'],
  limit: 10,
});

console.log('Found:', results.total, 'results');
results.results.forEach(r => {
  console.log(`- ${r.name} (${r.file_path})`);
});
```

### Python

```python
import requests
from typing import Optional, List, Dict, Any

class CodeAtlasClient:
    def __init__(self, base_url: str = 'http://localhost:8080', token: Optional[str] = None):
        self.base_url = base_url
        self.token = token
        self.session = requests.Session()
        if token:
            self.session.headers['Authorization'] = f'Bearer {token}'

    def _request(self, method: str, endpoint: str, **kwargs) -> Dict[str, Any]:
        url = f'{self.base_url}{endpoint}'
        response = self.session.request(method, url, **kwargs)
        response.raise_for_status()
        return response.json()

    def list_repositories(self) -> Dict[str, Any]:
        return self._request('GET', '/api/v1/repositories')

    def search(
        self,
        query: str,
        repo_id: Optional[str] = None,
        language: Optional[str] = None,
        kind: Optional[List[str]] = None,
        limit: int = 10
    ) -> Dict[str, Any]:
        payload = {'query': query, 'limit': limit}
        if repo_id:
            payload['repo_id'] = repo_id
        if language:
            payload['language'] = language
        if kind:
            payload['kind'] = kind
        
        return self._request('POST', '/api/v1/search', json=payload)

    def get_symbol_callers(self, symbol_id: str) -> Dict[str, Any]:
        return self._request('GET', f'/api/v1/symbols/{symbol_id}/callers')

# 使用示例
client = CodeAtlasClient('http://localhost:8080', 'your-token')

# 搜索代码
results = client.search(
    'authentication middleware',
    language='go',
    kind=['function'],
    limit=10
)

print(f"Found: {results['total']} results")
for result in results['results']:
    print(f"- {result['name']} ({result['file_path']})")
```

### cURL 示例

```bash
# 健康检查
curl http://localhost:8080/health

# 列出仓库
curl http://localhost:8080/api/v1/repositories

# 带认证
curl -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/v1/repositories

# 创建仓库
curl -X POST http://localhost:8080/api/v1/repositories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "name": "my-project",
    "url": "https://github.com/user/repo",
    "branch": "main"
  }'

# 搜索代码
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "query": "authentication function",
    "language": "go",
    "limit": 5
  }'

# 获取符号调用者
curl -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/v1/symbols/$SYMBOL_ID/callers
```

## 数据模型

### 符号类型 (Symbol Kinds)

- `function`: 函数或方法
- `class`: 类或结构体
- `interface`: 接口
- `variable`: 变量或常量
- `type`: 类型定义
- `module`: 模块或包

### 边类型 (Edge Types)

- `call`: 函数调用关系
- `import`: 导入/依赖关系
- `extends`: 继承关系
- `implements`: 接口实现
- `reference`: 对另一个符号的引用

### 支持的语言

- `go`
- `python`
- `javascript`
- `typescript`
- `java`
- `c`
- `cpp`
- `rust`

## 性能优化

### 1. 批量索引

```bash
# 使用较大的批处理大小
{
  "options": {
    "batch_size": 200
  }
}
```

### 2. 跳过向量生成

```bash
# 快速索引，稍后生成向量
{
  "options": {
    "skip_vectors": true
  }
}
```

### 3. 增量索引

```bash
# 只处理变更的文件
{
  "options": {
    "incremental": true
  }
}
```

### 4. 连接池配置

```bash
export DB_MAX_OPEN_CONNS=50
export DB_MAX_IDLE_CONNS=10
```

## 故障排除

### 连接被拒绝

```bash
# 检查服务器是否运行
curl http://localhost:8080/health

# 启动服务器
make run-api
```

### 401 未授权

```bash
# 检查认证是否启用
echo $ENABLE_AUTH

# 提供有效令牌
curl -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/v1/repositories
```

### CORS 错误

```bash
# 允许所有源（开发环境）
export CORS_ORIGINS=*

# 或特定源（生产环境）
export CORS_ORIGINS=https://app.example.com
```

### 数据库错误

```bash
# 检查数据库是否运行
docker-compose ps

# 启动数据库
make docker-up

# 验证连接
psql -h localhost -U codeatlas -d codeatlas
```

## 相关文档

- [快速开始指南](../../getting-started/quick-start.md)
- [CLI 工具指南](../cli/README.md)
- [配置指南](../../configuration/README.md)
- [部署指南](../../deployment/README.md)
- [HTTP 请求示例](../../../example.http)

## 测试 API

使用 VS Code REST Client 扩展测试 API：

1. 安装 [REST Client 扩展](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)
2. 打开 `example.http`
3. 更新变量（token、ID）
4. 点击 "Send Request"

---

**最后更新**: 2025-11-06  
**API 版本**: v1  
**维护者**: CodeAtlas Team
