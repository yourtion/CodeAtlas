# API 服务指南

> CodeAtlas HTTP API 完整参考

## 概述

CodeAtlas API 提供 RESTful 接口用于：
- 仓库管理
- 代码搜索
- 关系查询
- 向量检索

**Base URL**: `http://localhost:8080/api/v1`

## 快速开始

```bash
# 健康检查
curl http://localhost:8080/health

# 搜索函数
curl "http://localhost:8080/api/v1/search?q=main&type=function"

# 查找调用关系
curl "http://localhost:8080/api/v1/relationships?symbol_id=xxx&type=call"
```

## 端点参考

### 仓库管理

#### 创建仓库

```http
POST /api/v1/repositories
Content-Type: application/json

{
  "name": "myproject",
  "url": "https://github.com/user/repo",
  "branch": "main"
}
```

响应：
```json
{
  "repo_id": "uuid",
  "name": "myproject",
  "created_at": "2025-11-20T10:00:00Z"
}
```

#### 获取仓库列表

```http
GET /api/v1/repositories
```

响应：
```json
{
  "repositories": [
    {
      "repo_id": "uuid",
      "name": "myproject",
      "url": "https://github.com/user/repo",
      "branch": "main",
      "commit_hash": "abc123",
      "indexed_at": "2025-11-20T10:00:00Z"
    }
  ]
}
```

#### 获取仓库详情

```http
GET /api/v1/repositories/:repo_id
```

#### 删除仓库

```http
DELETE /api/v1/repositories/:repo_id
```

### 代码搜索

#### 搜索符号

```http
GET /api/v1/search?q=main&type=function&repo_id=xxx
```

参数：
- `q` - 搜索关键词（必需）
- `type` - 符号类型（可选：function, class, interface, variable）
- `repo_id` - 仓库 ID（可选）
- `language` - 语言（可选：go, javascript, python 等）
- `limit` - 结果数量（默认 20）

响应：
```json
{
  "results": [
    {
      "symbol_id": "uuid",
      "name": "main",
      "kind": "function",
      "signature": "func main()",
      "file_path": "src/main.go",
      "repo_name": "myproject",
      "span": {
        "start_line": 10,
        "end_line": 25
      }
    }
  ],
  "total": 1
}
```

#### 语义搜索

```http
POST /api/v1/search/semantic
Content-Type: application/json

{
  "query": "function that handles user authentication",
  "repo_id": "uuid",
  "limit": 10
}
```

响应：
```json
{
  "results": [
    {
      "symbol_id": "uuid",
      "name": "authenticateUser",
      "similarity": 0.92,
      "file_path": "src/auth.go",
      "snippet": "func authenticateUser(username, password string) error { ... }"
    }
  ]
}
```

### 关系查询

#### 查找调用关系

```http
GET /api/v1/relationships?symbol_id=xxx&type=call&direction=outgoing
```

参数：
- `symbol_id` - 符号 ID（必需）
- `type` - 关系类型（call, import, extends, implements, reference）
- `direction` - 方向（outgoing, incoming, both）
- `depth` - 深度（默认 1）

响应：
```json
{
  "relationships": [
    {
      "edge_id": "uuid",
      "source": {
        "symbol_id": "uuid",
        "name": "main",
        "file_path": "src/main.go"
      },
      "target": {
        "symbol_id": "uuid",
        "name": "processData",
        "file_path": "src/processor.go"
      },
      "edge_type": "call"
    }
  ]
}
```

#### 查找依赖关系

```http
GET /api/v1/dependencies?repo_id=xxx
```

响应：
```json
{
  "dependencies": [
    {
      "package_name": "github.com/gin-gonic/gin",
      "version": "v1.9.0",
      "type": "direct"
    }
  ]
}
```

### 文件和符号

#### 获取文件详情

```http
GET /api/v1/files/:file_id
```

响应：
```json
{
  "file_id": "uuid",
  "path": "src/main.go",
  "language": "go",
  "size": 1024,
  "symbols": [
    {
      "symbol_id": "uuid",
      "name": "main",
      "kind": "function"
    }
  ]
}
```

#### 获取符号详情

```http
GET /api/v1/symbols/:symbol_id
```

响应：
```json
{
  "symbol_id": "uuid",
  "name": "main",
  "kind": "function",
  "signature": "func main()",
  "file_path": "src/main.go",
  "span": {
    "start_line": 10,
    "end_line": 25,
    "start_byte": 200,
    "end_byte": 450
  },
  "docstring": "main is the entry point",
  "calls": ["processData", "handleError"],
  "called_by": []
}
```

## 认证

### API Key 认证（推荐）

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:8080/api/v1/search?q=main
```

配置：
```bash
export CODEATLAS_API_KEY=your-secret-key
```

### 基本认证

```bash
curl -u username:password \
  http://localhost:8080/api/v1/search?q=main
```

## 错误处理

### 错误响应格式

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Missing required parameter: q",
    "details": {
      "parameter": "q"
    }
  }
}
```

### 错误码

| 状态码 | 错误码 | 说明 |
|--------|--------|------|
| 400 | INVALID_REQUEST | 请求参数错误 |
| 401 | UNAUTHORIZED | 未授权 |
| 404 | NOT_FOUND | 资源不存在 |
| 429 | RATE_LIMIT_EXCEEDED | 超过速率限制 |
| 500 | INTERNAL_ERROR | 服务器内部错误 |

## 集成示例

### JavaScript

```javascript
// 使用 fetch
async function searchCode(query) {
  const response = await fetch(
    `http://localhost:8080/api/v1/search?q=${encodeURIComponent(query)}`,
    {
      headers: {
        'Authorization': 'Bearer YOUR_API_KEY'
      }
    }
  );
  return await response.json();
}

// 使用 axios
const axios = require('axios');

const client = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  headers: {
    'Authorization': 'Bearer YOUR_API_KEY'
  }
});

async function getRelationships(symbolId) {
  const { data } = await client.get('/relationships', {
    params: { symbol_id: symbolId, type: 'call' }
  });
  return data;
}
```

### Python

```python
import requests

class CodeAtlasClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url
        self.headers = {'Authorization': f'Bearer {api_key}'}
    
    def search(self, query, symbol_type=None):
        params = {'q': query}
        if symbol_type:
            params['type'] = symbol_type
        
        response = requests.get(
            f'{self.base_url}/search',
            params=params,
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()
    
    def get_relationships(self, symbol_id, edge_type='call'):
        response = requests.get(
            f'{self.base_url}/relationships',
            params={'symbol_id': symbol_id, 'type': edge_type},
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()

# 使用
client = CodeAtlasClient('http://localhost:8080/api/v1', 'YOUR_API_KEY')
results = client.search('main', symbol_type='function')
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
)

type Client struct {
    BaseURL string
    APIKey  string
    HTTP    *http.Client
}

func (c *Client) Search(query, symbolType string) (map[string]interface{}, error) {
    params := url.Values{}
    params.Add("q", query)
    if symbolType != "" {
        params.Add("type", symbolType)
    }
    
    req, _ := http.NewRequest("GET", 
        c.BaseURL+"/search?"+params.Encode(), nil)
    req.Header.Set("Authorization", "Bearer "+c.APIKey)
    
    resp, err := c.HTTP.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}
```

### cURL

```bash
# 搜索
curl -H "Authorization: Bearer YOUR_API_KEY" \
  "http://localhost:8080/api/v1/search?q=main&type=function"

# 创建仓库
curl -X POST \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"myproject","url":"https://github.com/user/repo"}' \
  http://localhost:8080/api/v1/repositories

# 查找关系
curl -H "Authorization: Bearer YOUR_API_KEY" \
  "http://localhost:8080/api/v1/relationships?symbol_id=xxx&type=call"
```

## 最佳实践

### 1. 分页

```bash
# 使用 limit 和 offset
curl "http://localhost:8080/api/v1/search?q=main&limit=20&offset=0"
curl "http://localhost:8080/api/v1/search?q=main&limit=20&offset=20"
```

### 2. 过滤

```bash
# 组合多个过滤条件
curl "http://localhost:8080/api/v1/search?q=main&type=function&language=go&repo_id=xxx"
```

### 3. 批量查询

```bash
# 使用 POST 批量查询符号
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"symbol_ids":["uuid1","uuid2","uuid3"]}' \
  http://localhost:8080/api/v1/symbols/batch
```

### 4. 缓存

```javascript
// 客户端缓存
const cache = new Map();

async function searchWithCache(query) {
  if (cache.has(query)) {
    return cache.get(query);
  }
  
  const result = await searchCode(query);
  cache.set(query, result);
  return result;
}
```

### 5. 错误重试

```python
import time
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry

# 配置重试策略
retry_strategy = Retry(
    total=3,
    backoff_factor=1,
    status_forcelist=[429, 500, 502, 503, 504]
)

adapter = HTTPAdapter(max_retries=retry_strategy)
session = requests.Session()
session.mount("http://", adapter)
session.mount("https://", adapter)
```

## 性能优化

### 1. 使用语义搜索

语义搜索比关键词搜索更准确：
```bash
# 关键词搜索（快但不准确）
curl "http://localhost:8080/api/v1/search?q=auth"

# 语义搜索（慢但准确）
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"query":"function that handles user authentication"}' \
  http://localhost:8080/api/v1/search/semantic
```

### 2. 限制结果数量

```bash
# 只获取前 10 个结果
curl "http://localhost:8080/api/v1/search?q=main&limit=10"
```

### 3. 指定仓库

```bash
# 在特定仓库中搜索（更快）
curl "http://localhost:8080/api/v1/search?q=main&repo_id=xxx"
```

## 速率限制

默认速率限制：
- 每分钟 60 次请求
- 每小时 1000 次请求

响应头：
```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 59
X-RateLimit-Reset: 1637654400
```

## WebSocket（实时更新）

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log('Index updated:', update);
};

// 订阅仓库更新
ws.send(JSON.stringify({
  type: 'subscribe',
  repo_id: 'xxx'
}));
```

## 下一步

- 查看 [CLI 工具](cli.md) 了解如何生成索引数据
- 查看 [配置指南](configuration.md) 自定义 API 行为
- 查看 [部署指南](deployment.md) 部署到生产环境
