-- 免登录 API key(CLI/pipeline 自动化):密钥与账号绑定,权限随账号角色走 RBAC。
-- 安全约定:
--   1) 只存 sha256(key) 哈希,永不落明文(密钥仅在创建接口返回一次);
--   2) 与账号绑定(account_id FK),禁用该账号即禁用其全部 key;
--   3) status 1启用 0停用;last_used_at 供审计/巡检。
BEGIN;

CREATE TABLE api_keys (
    id            BIGSERIAL PRIMARY KEY,
    account_id    BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name          VARCHAR(128) NOT NULL,          -- 用途说明,如 ci-pipeline / bossctl-local
    key_hash      CHAR(64) NOT NULL UNIQUE,       -- sha256 十六进制
    status        SMALLINT NOT NULL DEFAULT 1,    -- 1启用 0停用
    last_used_at  TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ,                    -- 空=永不过期
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by    BIGINT REFERENCES accounts(id)
);
CREATE INDEX idx_api_keys_account ON api_keys (account_id, status);

-- 添加 menu:apikey 权限并绑定 sysadmin 角色
INSERT INTO permissions (code, name)
SELECT 'menu:apikey', 'API key 管理'
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'menu:apikey');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.code = 'sysadmin' AND p.code = 'menu:apikey'
AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

COMMIT;