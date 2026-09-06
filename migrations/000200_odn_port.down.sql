-- 回滚 ODN 物理端口表。
BEGIN;

DROP TABLE IF EXISTS odn_port;

COMMIT;
