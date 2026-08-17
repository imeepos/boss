-- 阶段4 回滚:删除网络资源子表。
BEGIN;
DROP TABLE IF EXISTS port_change_history;
DROP TABLE IF EXISTS reserve_records;
DROP TABLE IF EXISTS expansions;
DROP TABLE IF EXISTS transfers;
COMMIT;
