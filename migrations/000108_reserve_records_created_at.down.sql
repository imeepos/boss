BEGIN;

DROP INDEX IF EXISTS idx_reserve_status_created;
ALTER TABLE reserve_records DROP COLUMN IF EXISTS created_at;
DELETE FROM biz_params WHERE key = 'order.reserve.timeoutMinutes';

COMMIT;
