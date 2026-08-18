-- 回滚 000044:恢复仅账号绑定的 api_keys 结构。
BEGIN;
ALTER TABLE api_keys
    ADD COLUMN account_id BIGINT REFERENCES accounts(id) ON DELETE CASCADE;
UPDATE api_keys SET account_id = subject_ref WHERE subject_type = 'account';
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS api_keys_subject_type_check;
DROP INDEX IF EXISTS idx_api_keys_subject;
ALTER TABLE api_keys DROP COLUMN subject_type, DROP COLUMN subject_ref;
COMMIT;