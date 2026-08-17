-- 阶段4 回滚:删除网络资源。
BEGIN;
DROP TABLE IF EXISTS ports;
DROP TABLE IF EXISTS resources;
COMMIT;
