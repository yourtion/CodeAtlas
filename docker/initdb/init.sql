-- CodeAtlas 数据库初始化（PostgreSQL 容器首次启动时执行）
-- 仅创建所需扩展；表结构由应用层的 goose 迁移统一管理
-- （pkg/models/migrations/*.sql，由 SchemaManager.CreateSchema 触发）。
-- 不要在此添加 CREATE TABLE，避免与迁移真源产生不一致。

CREATE EXTENSION IF NOT EXISTS age;
CREATE EXTENSION IF NOT EXISTS vector;
