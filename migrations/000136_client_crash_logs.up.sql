-- 客户端崩溃日志(worker 端 App 启动补传;App 侧 CrashLog 本地留痕后上传)
CREATE TABLE IF NOT EXISTS client_crash_logs (
    id           BIGSERIAL PRIMARY KEY,
    subject_type TEXT        NOT NULL DEFAULT 'worker',
    subject_id   BIGINT      NOT NULL DEFAULT 0,
    app          TEXT        NOT NULL DEFAULT '',
    log          TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_client_crash_logs_subject
    ON client_crash_logs (subject_type, subject_id, created_at DESC);
