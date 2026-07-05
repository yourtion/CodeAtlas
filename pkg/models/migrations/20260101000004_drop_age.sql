-- 移除 Apache AGE 图数据库支持
--
-- CodeAtlas 已将"代码知识图谱"查询从 AGE Cypher 迁移到关系表（edges/
-- symbols/files）上的 SQL 查询。AGE 图此前为"只写冷数据"——写入真实但
-- 读取路径全部静默降级到 N+1 SQL，且 Cypher 字符串拼接存在注入风险。
-- 关系表是完整真源（索引流水线 + header_impl_associator 均写入 edges 表），
-- 去 AGE 不丢失任何关系数据。
--
-- 本迁移清理已应用旧 init 迁移的环境中的 AGE 残留；新部署的 init 迁移
-- 已不再创建 AGE。

-- +goose Up

-- 删除代码图谱（若存在）。DROP GRAPH 会一并删除其所有顶点与边标签。
-- 注意：不同 AGE 版本的 DROP GRAPH 语法略有差异，用 DO 块容错。
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'age') THEN
        -- 加载 AGE 以使用其函数
        LOAD 'age';
        EXECUTE 'SELECT * FROM ag_catalog.drop_graph(''code_graph'', true)';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'drop_graph skipped: %', SQLERRM;
END $$;
-- +goose StatementEnd

-- 回收 ag_catalog 授权（幂等，对象不存在不报错）
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA ag_catalog FROM codeatlas;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA ag_catalog FROM codeatlas;
REVOKE USAGE ON SCHEMA ag_catalog FROM codeatlas;

-- 删除 AGE 扩展（依赖对象已在上一步清理）
DROP EXTENSION IF EXISTS age;


-- +goose Down

-- 不可逆：删除 AGE 后无法自动恢复。
-- 如需回退，需重新 CREATE EXTENSION age 并重建 code_graph（参见 git 历史
-- 中本迁移之前的 init 迁移版本）。
DO $$ BEGIN RAISE NOTICE 'drop_age migration is not reversible'; END $$;
