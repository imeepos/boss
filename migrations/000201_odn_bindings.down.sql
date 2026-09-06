-- 回滚 ODN 绑定表。
BEGIN;

DROP TABLE IF EXISTS odn_bindings;

COMMIT;
