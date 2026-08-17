-- 阶段6 回滚:删除四码合一。
BEGIN;
DROP TABLE IF EXISTS quad_links;
COMMIT;
