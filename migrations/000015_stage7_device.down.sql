-- 阶段7 回滚:删除设备监控。
BEGIN;
DROP TABLE IF EXISTS device_maintenances;
DROP TABLE IF EXISTS device_metrics;
COMMIT;
