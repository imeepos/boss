-- 阶段2 回滚:删除师傅事件事实表。
BEGIN;
DROP TABLE IF EXISTS asset_returns;
DROP TABLE IF EXISTS worker_feedbacks;
DROP TABLE IF EXISTS worker_tools;
DROP TABLE IF EXISTS worker_materials;
COMMIT;
