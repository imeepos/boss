-- 阶段2 回滚:删除师傅域。
BEGIN;
DROP TABLE IF EXISTS workers;
DROP TABLE IF EXISTS worker_groups;
COMMIT;
