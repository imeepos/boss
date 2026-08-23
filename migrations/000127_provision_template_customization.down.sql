BEGIN;
DROP INDEX IF EXISTS idx_provision_templates_status;
ALTER TABLE provision_templates DROP COLUMN IF EXISTS updated_at;
ALTER TABLE provision_templates DROP COLUMN IF EXISTS status;
ALTER TABLE provision_templates DROP COLUMN IF EXISTS version;
ALTER TABLE provision_templates DROP COLUMN IF EXISTS content;
COMMIT;
