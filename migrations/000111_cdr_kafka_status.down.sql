BEGIN;

DROP INDEX IF EXISTS idx_cdrs_kafka_status;
ALTER TABLE cdrs DROP COLUMN IF EXISTS kafka_status;

COMMIT;
