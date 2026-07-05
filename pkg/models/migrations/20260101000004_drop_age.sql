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

-- 删除 AGE 扩展及其创建的 ag_catalog schema（幂等）。
--
-- 历史实现尝试在 DO 块内 `LOAD 'age'; SELECT drop_graph(...)`，但 LOAD 是
-- 顶层命令、不能在 PL/pgSQL 函数体内执行，依赖 EXCEPTION 兜底，导致扩展
-- 已安装但 session 未 load 时 drop_graph 被静默跳过。
--
-- 由于 AGE 图为"只写冷数据"（关系表 edges 才是真源），无需保留 graph 内容：
-- 直接 DROP EXTENSION CASCADE 清理扩展对象，再 DROP SCHEMA ag_catalog
-- 兜底清理残留（IF EXISTS 幂等）。
DROP EXTENSION IF EXISTS age CASCADE;

-- 回收 ag_catalog 授权（幂等，对象不存在不报错）
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'ag_catalog') THEN
        REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA ag_catalog FROM codeatlas;
        REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA ag_catalog FROM codeatlas;
        REVOKE USAGE ON SCHEMA ag_catalog FROM codeatlas;
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'ag_catalog privilege revoke skipped: %', SQLERRM;
END $$;
-- +goose StatementEnd

-- 清理 AGE 自创建的 schema（CASCADE 一并清理其内残留对象）
DROP SCHEMA IF EXISTS ag_catalog CASCADE;


-- +goose Down

-- 不可逆：删除 AGE 后无法自动恢复。
-- 如需回退，需重新 CREATE EXTENSION age 并重建 code_graph（参见 git 历史
-- 中本迁移之前的 init 迁移版本）。
DO $$ BEGIN RAISE NOTICE 'drop_age migration is not reversible'; END $$;
