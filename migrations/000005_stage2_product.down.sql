-- 阶段2 回滚:删除产品资费。
BEGIN;
DROP TABLE IF EXISTS region_offers;
DROP TABLE IF EXISTS product_offers;
COMMIT;
