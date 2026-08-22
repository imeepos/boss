BEGIN;

DROP INDEX IF EXISTS idx_admin_notif_sla;
ALTER TABLE admin_notifications
    DROP COLUMN IF EXISTS due_at,
    DROP COLUMN IF EXISTS resolved_at;

COMMIT;
