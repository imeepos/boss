-- 回滚 ODN 地理空间编码映射。
BEGIN;
DROP TABLE IF EXISTS odn_city_code;
DROP TABLE IF EXISTS odn_region_code;
COMMIT;
