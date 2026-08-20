-- 回滚渠道对账行级明细。
BEGIN;
DROP TABLE IF EXISTS reconciliation_items;
COMMIT;
