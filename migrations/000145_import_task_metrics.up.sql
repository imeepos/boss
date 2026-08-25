-- 导入任务统计口径：总行数、成功、失败、跳过均持久化。
BEGIN;
ALTER TABLE import_tasks
  ADD COLUMN total INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN skipped INTEGER NOT NULL DEFAULT 0;
UPDATE import_tasks SET total = imported + failed + skipped;
COMMIT;
