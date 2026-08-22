-- 下单幂等键:orders.request_id(可选,客户端 requestId)。
-- 裁定(adopted/2026-08-23-order-submit-idempotency):方案A 可选幂等键
-- 优于方案B 同客户同地址非终态拒单——不误伤合法二装,契约向后兼容。
-- 重复提交 → 命中唯一索引 → 返回已有订单(同 orderNo),不再发新号。
BEGIN;

ALTER TABLE orders ADD COLUMN request_id TEXT;

CREATE UNIQUE INDEX uq_orders_customer_request
    ON orders (customer_id, request_id)
    WHERE request_id IS NOT NULL AND request_id <> '';

COMMIT;
