-- 000218 down: 移除里程碑表与项目预算列。
BEGIN;

DROP TABLE IF EXISTS construction_milestones;

ALTER TABLE construction_projects
    DROP COLUMN IF EXISTS budget_amount;

COMMIT;
