-- 开放平台 M1(执行计划 docs/plan/q4-open-platform-plan.md):
-- open_apps            外部集成方应用凭证(AppId+Secret,HMAC 验签)
-- open_webhook_subscriptions M2 预留:事件订阅(endpoint + 订阅事件类型)
-- open_usage_day       日配额计数(app × day upsert 自增)

CREATE TABLE open_apps (
    id              BIGSERIAL PRIMARY KEY,
    app_id          TEXT NOT NULL UNIQUE,          -- 公开标识,如 op_<16hex>
    secret          TEXT NOT NULL,                 -- HMAC 验签需原文,决策见 adopted note
    name            TEXT NOT NULL,
    status          SMALLINT NOT NULL DEFAULT 1,   -- 1启用 0停用
    rate_limit_rpm  INT NOT NULL DEFAULT 60,       -- 每分钟调用上限(令牌桶)
    daily_quota     INT NOT NULL DEFAULT 10000,    -- 日调用配额
    sandbox         BOOLEAN NOT NULL DEFAULT FALSE,-- 沙箱应用(M4)
    last_used_at    TIMESTAMPTZ,
    created_by      BIGINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE open_webhook_subscriptions (
    id          BIGSERIAL PRIMARY KEY,
    app_id      BIGINT NOT NULL REFERENCES open_apps(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL,                     -- 如 order.activated
    endpoint_url TEXT NOT NULL,
    status      SMALLINT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (app_id, event_type, endpoint_url)
);

CREATE TABLE open_usage_day (
    app_id      BIGINT NOT NULL REFERENCES open_apps(id) ON DELETE CASCADE,
    day         DATE NOT NULL,
    call_count  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (app_id, day)
);

-- menu:openplat 权限并绑定 sysadmin 角色(管理面门禁,与 000042 menu:apikey 同法)
INSERT INTO permissions (code, name)
SELECT 'menu:openplat', '开放平台管理'
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE code = 'menu:openplat');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.code = 'sysadmin' AND p.code = 'menu:openplat'
AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
);
