-- A1(认证凭证与防爆破):LOID 密码凭据 + 认证失败原因码 + 防爆破锁定表。
-- 字段权威:docs/contract/fields.md(auth_logs.fail_reason / lo_accounts.password_credential)。
-- 凭据落库形态:v1$gcm$<nonce-b64>$<ct-b64>(AES-256-GCM,密钥外置,库内无明文;空=NULL=未设密)。
-- 决策记录:docs/notes/adopted/2026-09-06-aaa-credential-storage.md。
BEGIN;

ALTER TABLE lo_accounts ADD COLUMN password_credential TEXT;

ALTER TABLE auth_logs ADD COLUMN fail_reason VARCHAR(32) NOT NULL DEFAULT '';
-- 枚举:BAD_CREDENTIAL / LOCKED / NOT_FOUND / SUSPENDED / CLOSED(空=SUCCESS 或存量行)

CREATE TABLE lo_auth_lockouts (
    loid         VARCHAR(32) PRIMARY KEY,
    fail_count   INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,                  -- NULL=未锁定;到期行由认证路径清理解锁
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;