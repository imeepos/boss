-- 000224 down: 回滚上报人代次列与点位索引。
BEGIN;
DROP INDEX IF EXISTS idx_construction_progress_points;
ALTER TABLE construction_progress DROP COLUMN IF EXISTS reporter_type;
COMMIT;