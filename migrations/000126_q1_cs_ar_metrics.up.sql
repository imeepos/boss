-- Q1 CS/AR 指标基础：办结留痕与欠费快照新鲜度。
BEGIN;
ALTER TABLE complaints ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;
ALTER TABLE complaints ADD COLUMN IF NOT EXISTS closed_by BIGINT;
ALTER TABLE complaints ADD COLUMN IF NOT EXISTS resolution TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_complaints_status_created ON complaints(status, created_at);
ALTER TABLE arrears ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE arrears ADD COLUMN IF NOT EXISTS overdue_since TIMESTAMPTZ;
COMMIT;
