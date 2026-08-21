-- 推送设备注册表(push_devices,NOT 域推送通道支撑实体,docs/plan/push-integration.md §4)。
-- RegistrationID 全局唯一:换账号登录同一设备时 rebinding(UPSERT 改绑到新主体)。
BEGIN;
CREATE TABLE push_devices (
    id              BIGSERIAL PRIMARY KEY,
    subject_type    TEXT NOT NULL,              -- user | worker(与 apikey 主体词一致)
    subject_id      BIGINT NOT NULL,
    registration_id TEXT NOT NULL,
    vendor          TEXT NOT NULL DEFAULT '',   -- jpush(预留多供应商)
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (registration_id)
);
CREATE INDEX idx_push_devices_subject ON push_devices (subject_type, subject_id);
COMMIT;
