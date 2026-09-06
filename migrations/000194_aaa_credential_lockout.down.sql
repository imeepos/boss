-- 回滚 A1:凭据列 / 失败原因码 / 防爆破锁定表。
BEGIN;

DROP TABLE IF EXISTS lo_auth_lockouts;
ALTER TABLE auth_logs DROP COLUMN IF EXISTS fail_reason;
ALTER TABLE lo_accounts DROP COLUMN IF EXISTS password_credential;

COMMIT;