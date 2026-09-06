-- 000192: OSS 容量预警告警检索索引(P5-W1)。
-- 容量扫描判重按 (source='capacity', resource_id, status='OPEN') 反查 alarms;
-- 部分索引只覆盖容量告警行,周期巡检与手动扫描共用,避免整表扫 alarms。

BEGIN;

CREATE INDEX IF NOT EXISTS idx_alarms_capacity_open
    ON alarms(resource_id)
    WHERE source = 'capacity' AND status = 'OPEN';

COMMIT;