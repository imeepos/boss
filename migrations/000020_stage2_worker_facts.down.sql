-- 阶段2 回滚:删除师傅月度事实表。
BEGIN;
DROP TABLE IF EXISTS worker_schedules;
DROP TABLE IF EXISTS worker_commissions;
DROP TABLE IF EXISTS worker_performances;
COMMIT;
