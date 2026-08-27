BEGIN;

DROP INDEX IF EXISTS idx_replacements_worker;
ALTER TABLE replacements
    DROP COLUMN IF EXISTS finished_at,
    DROP COLUMN IF EXISTS worker_name,
    DROP COLUMN IF EXISTS worker_id;

COMMIT;
