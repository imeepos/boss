-- 阶段7 回滚:删除配置下发。
BEGIN;
DROP TABLE IF EXISTS provision_logs;
DROP TABLE IF EXISTS provision_tasks;
DROP TABLE IF EXISTS provision_templates;
COMMIT;
