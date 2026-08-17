-- 阶段3 回滚:删除资产持有台账。
BEGIN;
DROP TABLE IF EXISTS asset_assignments;
COMMIT;
