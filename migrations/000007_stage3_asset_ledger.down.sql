-- 阶段3 回滚:删除资产台账。
BEGIN;
DROP TABLE IF EXISTS stocktakes;
DROP TABLE IF EXISTS replacements;
DROP TABLE IF EXISTS asset_lifecycles;
COMMIT;
