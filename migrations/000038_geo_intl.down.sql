-- 回滚 000031:先摘除 addresses 挂接列,再按依赖逆序删表。
BEGIN;
ALTER TABLE addresses
    DROP COLUMN IF EXISTS country_code,
    DROP COLUMN IF EXISTS admin_code;
DROP TABLE IF EXISTS country_calling_code;
DROP TABLE IF EXISTS country_currency;
DROP TABLE IF EXISTS country_time_zone;
DROP TABLE IF EXISTS geo_subdivision_i18n;
DROP TABLE IF EXISTS geo_subdivision;
DROP TABLE IF EXISTS geo_country_i18n;
DROP TABLE IF EXISTS geo_country;
COMMIT;
