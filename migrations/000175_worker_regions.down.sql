BEGIN;

DROP INDEX IF EXISTS idx_worker_regions_region;
DROP TABLE IF EXISTS worker_regions;

COMMIT;
