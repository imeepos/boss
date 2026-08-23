DROP INDEX IF EXISTS uq_loy_entries_payearn;
ALTER TABLE loy_point_entries
    DROP COLUMN IF EXISTS expired,
    DROP COLUMN IF EXISTS expires_at;
DROP TABLE IF EXISTS loy_earn_rules;
DROP TABLE IF EXISTS loy_task_completions;
DROP TABLE IF EXISTS loy_tasks;
DROP TABLE IF EXISTS loy_levels;
