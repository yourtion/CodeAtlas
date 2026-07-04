-- 关键词检索（BM25 风格）与模糊匹配支持
--
-- 为混合检索（向量召回 + 关键词召回 + 重排）建立数据基础：
-- 1. pg_trgm 扩展：支持符号名的模糊匹配（拼写容错、部分匹配）
-- 2. split_identifier 函数：把驼峰/下划线标识符拆分为空格分隔的 token，
--    使 getUserName 可被 "user"/"name"/"username" 命中（默认 simple 配置不拆分）
-- 3. vectors.content_tsv 生成列：基于 content 的 tsvector，simple 配置（无词干、
--    无停用词），匹配代码场景的可预测分词需求
-- 4. GIN 索引：加速 tsvector 与 trigram 查询

-- +goose Up

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- split_identifier: 把代码标识符拆分为空格分隔的 token。
-- getUserName -> "get User Name"，get_user_name -> "get user name"，
-- HTTPServer -> "HTTP Server"。IMMUTABLE 以便用于生成列索引。
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION split_identifier(ident text)
RETURNS text AS $$
SELECT regexp_replace(
         regexp_replace(
           regexp_replace($1, '([a-z0-9])([A-Z])', '\1 \2', 'g'),
           '([A-Z]+)([A-Z][a-z])', '\1 \2', 'g'),
         '[^a-zA-Z0-9]+', ' ', 'g')
$$ LANGUAGE SQL IMMUTABLE;
-- +goose StatementEnd

-- vectors.content_tsv: 基于 content（signature + docstring + summary 拼接）的
-- 全文检索向量。simple 配置：无词干、无停用词，适配代码符号的可预测分词。
-- 对 content 先做 split_identifier 处理驼峰，再 to_tsvector。
-- +goose StatementBegin
ALTER TABLE vectors
ADD COLUMN IF NOT EXISTS content_tsv tsvector
GENERATED ALWAYS AS (
    to_tsvector('simple', split_identifier(coalesce(content, '')))
) STORED;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_vectors_content_tsv
ON vectors USING gin (content_tsv);

-- symbols.name 的 trigram 索引：支持符号名模糊匹配（拼写容错）。
-- 例：查询 "findContain" 可命中 findContainingFunction。
CREATE INDEX IF NOT EXISTS idx_symbols_name_trgm
ON symbols USING gin (name gin_trgm_ops);

-- symbols.name_tsv: 符号名的全文检索向量，配合 split_identifier
-- 使驼峰/下划线符号名可被关键词命中。
-- +goose StatementBegin
ALTER TABLE symbols
ADD COLUMN IF NOT EXISTS name_tsv tsvector
GENERATED ALWAYS AS (
    to_tsvector('simple', split_identifier(coalesce(name, '')))
) STORED;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_symbols_name_tsv
ON symbols USING gin (name_tsv);


-- +goose Down

DROP INDEX IF EXISTS idx_symbols_name_tsv;
ALTER TABLE symbols DROP COLUMN IF EXISTS name_tsv;
DROP INDEX IF EXISTS idx_symbols_name_trgm;
DROP INDEX IF EXISTS idx_vectors_content_tsv;
ALTER TABLE vectors DROP COLUMN IF EXISTS content_tsv;
DROP FUNCTION IF EXISTS split_identifier(text);
-- btree_gin / pg_trgm 扩展可能被其他对象使用，不自动删除
