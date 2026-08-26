-- 000151 down: 回滚 description / contact / rel_order_no 字段与分页索引。
BEGIN;

DROP INDEX IF EXISTS idx_complaints_status;
DROP INDEX IF EXISTS idx_complaints_customer_created;

ALTER TABLE complaints DROP COLUMN IF EXISTS rel_order_no;
ALTER TABLE complaints DROP COLUMN IF EXISTS contact;
ALTER TABLE complaints DROP COLUMN IF EXISTS description;

COMMIT;