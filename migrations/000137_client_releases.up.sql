-- 客户端版本发布域(fields.md 8F):双端(user/worker)App 发版记录,
-- APK 对象入 MinIO(apk_object_key),灰度=比例分桶+白名单豁免。
BEGIN;
CREATE TABLE IF NOT EXISTS client_releases (
    id                BIGSERIAL PRIMARY KEY,
    app               TEXT        NOT NULL,              -- user / worker
    platform          TEXT        NOT NULL DEFAULT 'android',
    version           TEXT        NOT NULL,              -- 展示 semver,如 1.2.0
    version_code      INT         NOT NULL,              -- 单调递增,升级判定唯一依据
    min_supported_code INT        NOT NULL DEFAULT 1,    -- 低于此 versionCode 强制更新
    notes             TEXT        NOT NULL DEFAULT '',   -- 更新日志
    force             BOOLEAN     NOT NULL DEFAULT FALSE,-- true 弹框不可忽略
    status            TEXT        NOT NULL DEFAULT 'DRAFT', -- DRAFT/GRAY/PUBLISHED/ROLLED_BACK
    rollout_percent   INT         NOT NULL DEFAULT 0,    -- 0~100,GRAY 生效
    whitelist_ids     BIGINT[]    NOT NULL DEFAULT '{}', -- 灰度白名单(worker/customer id)
    apk_object_key    TEXT        NOT NULL DEFAULT '',
    apk_size          BIGINT      NOT NULL DEFAULT 0,
    sha256            TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_client_releases_app CHECK (app IN ('user','worker')),
    CONSTRAINT ck_client_releases_status CHECK (status IN ('DRAFT','GRAY','PUBLISHED','ROLLED_BACK')),
    CONSTRAINT ck_client_releases_rollout CHECK (rollout_percent BETWEEN 0 AND 100),
    CONSTRAINT ck_client_releases_vercode CHECK (version_code > 0),
    CONSTRAINT uq_client_releases_ver UNIQUE (app, platform, version_code)
);
CREATE INDEX IF NOT EXISTS idx_client_releases_active
    ON client_releases (app, platform, status, version_code DESC);

-- 菜单权限(menu:release,随功能迁移走,沿 000135 cms_menu 先例)。
INSERT INTO permissions (code, name) VALUES
    ('menu:release', '订单与工单·版本发布')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:release'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
