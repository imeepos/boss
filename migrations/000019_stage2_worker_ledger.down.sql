-- 阶段2 回滚:删除师傅台账。
BEGIN;
DROP TABLE IF EXISTS worker_messages;
DROP TABLE IF EXISTS worker_settings;
DROP TABLE IF EXISTS worker_group_memberships;
COMMIT;
