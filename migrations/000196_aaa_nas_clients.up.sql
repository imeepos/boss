-- AAA-A5(G7+G8):per-NAS 客户端注册表(名称/来源 IP/共享密钥密文/厂商/CoA 端口/启停)。
-- 密钥落库沿用 AAA 凭据密文体系 v1$gcm$<nonce-b64>$<ct-b64>(AES-256-GCM,密钥外置,库内无明文)。
-- 来源 IP 唯一:RADIUS 按请求来源 IP 查表校验密钥,未注册/停用拒绝并留痕(留痕红线)。
-- 字段权威:docs/contract/fields.md §8J;决策记录:docs/notes/adopted/2026-09-06-aaa-per-nas-vsa.md。
BEGIN;

CREATE TABLE aaa_nas_clients (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(64) NOT NULL,                   -- 展示名(告警与 admin 列表用)
    nas_ip     VARCHAR(64) NOT NULL,                   -- 请求来源 IP(RADIUS/CoA 定位键)
    secret_enc TEXT        NOT NULL,                   -- 共享密钥密文(v1$gcm$,绝不回显)
    vendor     VARCHAR(16) NOT NULL DEFAULT 'GENERIC', -- HUAWEI/ZTE/GENERIC(VSA 限速下发依据)
    coa_port   INT         NOT NULL DEFAULT 3799,      -- 该 NAS 的 CoA/DM 端口(RFC 5176)
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE,      -- 停用=认证/计费/CoA 一律拒绝
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_aaa_nas_clients_ip UNIQUE (nas_ip),
    CONSTRAINT ck_aaa_nas_clients_vendor CHECK (vendor IN ('HUAWEI', 'ZTE', 'GENERIC'))
);

COMMIT;