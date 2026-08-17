-- 阶段5 回滚:删除订单子表。
BEGIN;
DROP TABLE IF EXISTS dispatch_transfers;
DROP TABLE IF EXISTS activation_callbacks;
DROP TABLE IF EXISTS dismantles;
COMMIT;
