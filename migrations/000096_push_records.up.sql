-- 推送留痕表(push_records,"留痕为权威、推送尽力而为",docs/plan/push-integration.md §4)。
-- 每次定向推送逐设备一行:SENT/FAILED/LOG(日志通道)/SKIP_NO_DEVICE/SKIP_DISABLED。
BEGIN;
CREATE TABLE push_records (
    id              BIGSERIAL PRIMARY KEY,
    subject_type    TEXT NOT NULL,              -- user | worker
    subject_id      BIGINT NOT NULL,
    registration_id TEXT NOT NULL DEFAULT '',   -- SKIP_* 时为空
    title           TEXT NOT NULL DEFAULT '',
    alert           TEXT NOT NULL DEFAULT '',
    extras          JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL,              -- SENT/FAILED/LOG/SKIP_NO_DEVICE/SKIP_DISABLED
    msg_id          TEXT NOT NULL DEFAULT '',   -- 服务商消息 ID
    error           TEXT NOT NULL DEFAULT '',   -- FAILED 时原因
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_push_records_subject ON push_records (subject_type, subject_id, created_at DESC);
COMMIT;
