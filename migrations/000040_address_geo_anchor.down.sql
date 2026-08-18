-- 回滚 000040:撤约束/视图/索引;已回填的锚点数据保留(与 000038 down 同口径)。
BEGIN;
DROP VIEW IF EXISTS v_addresses_geo;
DROP INDEX IF EXISTS idx_addresses_geo_unlinked;
ALTER TABLE addresses DROP CONSTRAINT IF EXISTS chk_addresses_geo_match;
ALTER TABLE addresses DROP CONSTRAINT IF EXISTS chk_addresses_geo_root;
COMMIT;
