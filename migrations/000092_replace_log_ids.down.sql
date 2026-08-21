BEGIN;
DROP INDEX IF EXISTS idx_worker_replace_ticket_id;
ALTER TABLE worker_replace_logs
    DROP COLUMN IF EXISTS new_tag_id,
    DROP COLUMN IF EXISTS old_tag_id,
    DROP COLUMN IF EXISTS dispatch_ticket_id;
COMMIT;
