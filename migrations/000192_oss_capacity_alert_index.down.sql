-- 000192 down: 删除容量告警检索索引;告警数据本身不受影响。
DROP INDEX IF EXISTS idx_alarms_capacity_open;