-- 阶段5 回滚:删除计费账务。
BEGIN;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS bills;
COMMIT;
