-- 阶段7:网络监控告警(MON,承接 alarm.html)。
-- 字段权威:server-ts/src/entities/oss.ts(Alarm)。
BEGIN;

CREATE TABLE alarms (
    id          BIGSERIAL PRIMARY KEY,
    alarm_no    VARCHAR(32) NOT NULL UNIQUE,   -- ALM-001
    level       VARCHAR(16) NOT NULL,          -- CRITICAL严重/WARNING警告/INFO提示
    source      VARCHAR(32) NOT NULL,          -- device/quadlink/aaa
    content     VARCHAR(255) NOT NULL,
    resource_id BIGINT,                        -- 软引用 resources.id
    status      VARCHAR(16) NOT NULL DEFAULT 'OPEN',  -- OPEN待处理/ACKED已确认/CLOSED已关闭
    created_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_alarms_level ON alarms(level, created_at);

COMMIT;
