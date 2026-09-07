-- 000208 down: 移除施工项目承包商两列。
ALTER TABLE construction_projects
    DROP COLUMN IF EXISTS contractor_name,
    DROP COLUMN IF EXISTS contractor_id;