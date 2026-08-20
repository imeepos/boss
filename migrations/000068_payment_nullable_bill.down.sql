-- 回滚:先删无账单的充值流水,再去列/恢复 NOT NULL。
DROP INDEX IF EXISTS idx_payments_customer;
DELETE FROM payments WHERE bill_id IS NULL;
ALTER TABLE payments DROP COLUMN IF EXISTS customer_id;
ALTER TABLE payments ALTER COLUMN bill_id SET NOT NULL;
