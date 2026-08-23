BEGIN;
ALTER TABLE arrears DROP COLUMN IF EXISTS overdue_since;
ALTER TABLE arrears DROP COLUMN IF EXISTS updated_at;
DROP INDEX IF EXISTS idx_complaints_status_created;
ALTER TABLE complaints DROP COLUMN IF EXISTS resolution;
ALTER TABLE complaints DROP COLUMN IF EXISTS closed_by;
ALTER TABLE complaints DROP COLUMN IF EXISTS closed_at;
COMMIT;
