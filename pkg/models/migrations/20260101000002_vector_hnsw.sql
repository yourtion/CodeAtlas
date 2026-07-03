-- 向量相似度搜索 HNSW 索引
-- 替换原 IVFFlat 方案。HNSW 召回率更高、增量更新友好（无需重建），
-- 适配代码符号量级（通常 10 万级）。pgvector 0.5+ 支持。
-- m=16, ef_construction=64 为 pgvector 官方推荐默认值。

-- +goose Up

-- 若存在旧的 IVFFlat 索引则先删除
DROP INDEX IF EXISTS idx_vectors_embedding;
DROP INDEX IF EXISTS idx_vectors_embedding_cosine;

CREATE INDEX IF NOT EXISTS idx_vectors_embedding_hnsw
ON vectors USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);


-- +goose Down

DROP INDEX IF EXISTS idx_vectors_embedding_hnsw;
