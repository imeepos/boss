-- 阶段7:设备监控(指标采集 + 健康观察)。
-- 字段权威:server-ts/src/entities/device.ts。
BEGIN;

CREATE TABLE device_metrics (
    id            BIGSERIAL PRIMARY KEY,
    resource_id   BIGINT NOT NULL,       -- 软引用 resources
    optical_power NUMERIC(5,2),          -- 光功率(dBm),空=离线无数据
    packet_loss   NUMERIC(5,2),          -- 丢包率(%)
    status        VARCHAR(16) NOT NULL,  -- ONLINE/OFFLINE/FAULT
    collected_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_device_metrics_resource ON device_metrics(resource_id, collected_at);

CREATE TABLE device_maintenances (
    id           BIGSERIAL PRIMARY KEY,
    device_no    VARCHAR(64) NOT NULL,   -- OLT-01
    device_type  VARCHAR(32),            -- PON 9口
    health_score SMALLINT NOT NULL,      -- 0~100
    fault_count  INTEGER NOT NULL,
    age_years    NUMERIC(4,1),           -- 服役年数
    reason       VARCHAR(128),           -- 观察原因
    priority     VARCHAR(16) NOT NULL    -- MUST_REPLACE/SUGGEST/WATCH
);
CREATE INDEX idx_device_maintenances_no ON device_maintenances(device_no);

COMMIT;
