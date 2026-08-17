-- 阶段5 回滚:删除订单。
BEGIN;
DROP TABLE IF EXISTS order_stages;
DROP TABLE IF EXISTS orders;
COMMIT;
