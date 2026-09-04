-- 回滚 000053:删除订单服务评价表。
BEGIN;
DROP TABLE IF EXISTS order_ratings;
COMMIT;
