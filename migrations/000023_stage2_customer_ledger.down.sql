-- 阶段2 回滚:删除客户与资费台账。
BEGIN;
DROP TABLE IF EXISTS region_price_histories;
DROP TABLE IF EXISTS product_price_histories;
DROP TABLE IF EXISTS customer_histories;
COMMIT;
