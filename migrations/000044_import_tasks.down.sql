-- 回滚 000044:移除导入任务记录表。
BEGIN;
DROP TABLE IF EXISTS import_tasks CASCADE;
COMMIT;
