-- 回滚 000116:删幂等键列与索引(request_id 数据不保留)。
BEGIN;

DROP INDEX IF EXISTS uq_orders_customer_request;
ALTER TABLE orders DROP COLUMN IF EXISTS request_id;

COMMIT;
