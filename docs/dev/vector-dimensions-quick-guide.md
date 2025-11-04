# Vector Dimensions Quick Guide

快速指南：如何为不同的 embedding 模型配置向量维度。

## 常用模型配置

### OpenAI text-embedding-3-small (1536维)

```bash
# 1. 更新 .env 文件
cat >> .env << EOF
EMBEDDING_MODEL=text-embedding-3-small
EMBEDDING_API_KEY=sk-your-key-here
EMBEDDING_DIMENSIONS=1536
EOF

# 2. 修改数据库向量维度
make alter-vector-dimension VECTOR_DIM=1536

# 3. 重新索引
./bin/cli parse -p /path/to/repo
```

### Qwen3 Embedding (1024维)

```bash
# 1. 更新 .env 文件
cat >> .env << EOF
EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
EMBEDDING_DIMENSIONS=1024
EOF

# 2. 修改数据库向量维度
make alter-vector-dimension VECTOR_DIM=1024

# 3. 重新索引
./bin/cli parse -p /path/to/repo
```

### Nomic Embed Text (768维)

```bash
# 1. 更新 .env 文件
cat >> .env << EOF
EMBEDDING_MODEL=nomic-embed-text
EMBEDDING_API_ENDPOINT=http://localhost:1234/v1/embeddings
EMBEDDING_DIMENSIONS=768
EOF

# 2. 修改数据库向量维度
make alter-vector-dimension VECTOR_DIM=768

# 3. 重新索引
./bin/cli parse -p /path/to/repo
```

## 常见场景

### 场景1：全新安装

```bash
# 设置环境变量
export EMBEDDING_DIMENSIONS=1536

# 启动数据库
make docker-up

# 初始化（会自动使用环境变量中的维度）
make init-db
```

### 场景2：切换 embedding 模型

```bash
# 假设从 1024 维切换到 1536 维

# 1. 修改向量维度（会清空现有向量）
make alter-vector-dimension-force VECTOR_DIM=1536

# 2. 更新配置
sed -i '' 's/EMBEDDING_DIMENSIONS=1024/EMBEDDING_DIMENSIONS=1536/' .env

# 3. 重新索引所有仓库
./bin/cli parse -p /path/to/repo1
./bin/cli parse -p /path/to/repo2
```

### 场景3：测试不同维度

```bash
# 测试 768 维
EMBEDDING_DIMENSIONS=768 go test ./tests/integration/... -v

# 测试 1536 维
EMBEDDING_DIMENSIONS=1536 go test ./tests/integration/... -v
```

## 检查当前维度

```bash
# 方法1：使用 psql
psql -U codeatlas -d codeatlas -c "
SELECT 
    format_type(atttypid, atttypmod) as embedding_type,
    COUNT(*) as vector_count
FROM pg_attribute, vectors
WHERE attrelid = 'vectors'::regclass 
AND attname = 'embedding'
GROUP BY format_type(atttypid, atttypmod);
"

# 方法2：查看配置
grep EMBEDDING_DIMENSIONS .env
```

## 故障排除

### 错误：dimension mismatch

```
ERROR: dimension of vector (768) does not match column dimension (1536)
```

**解决方案**：
```bash
# 检查当前维度
psql -U codeatlas -d codeatlas -c "
SELECT format_type(atttypid, atttypmod) 
FROM pg_attribute
WHERE attrelid = 'vectors'::regclass AND attname = 'embedding';
"

# 修改为正确的维度
make alter-vector-dimension-force VECTOR_DIM=768
```

### 错误：cannot alter type with existing data

```
ERROR: cannot alter type of a column used in a vector index
```

**解决方案**：
```bash
# 使用 force 模式（会清空数据）
make alter-vector-dimension-force VECTOR_DIM=1536

# 或手动清空
psql -U codeatlas -d codeatlas -c "TRUNCATE TABLE vectors;"
make alter-vector-dimension VECTOR_DIM=1536
```

## 最佳实践

1. **文档化选择**：在 `.env` 文件中注释说明使用的模型和维度
2. **测试先行**：在测试环境验证维度配置后再应用到生产环境
3. **备份数据**：切换维度前备份重要数据
4. **批量重建**：切换维度后需要重新索引所有仓库
5. **环境一致**：确保开发、测试、生产环境使用相同的 embedding 模型和维度

## 性能考虑

不同维度对性能的影响：

| 维度 | 存储空间 | 查询速度 | 精度 |
|------|---------|---------|------|
| 768  | 小      | 快      | 中   |
| 1024 | 中      | 中      | 中   |
| 1536 | 大      | 慢      | 高   |
| 3072 | 很大    | 很慢    | 很高 |

建议：
- 开发环境：使用 768 或 1024 维（快速）
- 生产环境：使用 1536 维（平衡性能和精度）
- 高精度需求：使用 3072 维（需要更多资源）
