-- 阶段5 回滚:删除订单子表。
BEGIN;
DROP TABLE IF EXISTS scan_logs;
DROP TABLE IF EXISTS complaints;
DROP TABLE IF EXISTS dispatch_tickets;
COMMIT;
