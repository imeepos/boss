BEGIN;
DROP INDEX IF EXISTS uq_import_tasks_client_key;
ALTER TABLE import_tasks DROP COLUMN IF EXISTS client_key;
COMMIT;
