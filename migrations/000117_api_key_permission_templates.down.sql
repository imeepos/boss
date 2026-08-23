BEGIN;
DROP TABLE IF EXISTS api_key_template_permissions;
DROP TABLE IF EXISTS api_key_permission_templates;
ALTER TABLE api_keys DROP COLUMN IF EXISTS template_code;
COMMIT;
