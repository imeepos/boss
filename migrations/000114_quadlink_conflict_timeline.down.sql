BEGIN;

DROP INDEX IF EXISTS idx_quad_links_conflict_at;
ALTER TABLE quad_links
    DROP COLUMN IF EXISTS conflict_at,
    DROP COLUMN IF EXISTS cleared_at;

COMMIT;
