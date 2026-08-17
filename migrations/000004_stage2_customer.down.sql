-- 阶段2 回滚:删除客户档案。
BEGIN;
DROP TABLE IF EXISTS customers;
COMMIT;
