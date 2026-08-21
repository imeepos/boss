-- 允许同客户多订单:去掉 customer_id 唯一约束。
-- 一个客户可以同时有多个在途订单,每个订单各占一个端口,各有独立 quad_link 行。
-- customer_id 保留普通索引(按客户反查),不再唯一。
BEGIN;

DROP INDEX IF EXISTS uq_quad_links_customer;

-- 恢复为普通索引(按 customer_id 反查仍需)。
CREATE INDEX IF NOT EXISTS idx_quad_links_customer
    ON quad_links (customer_id) WHERE customer_id IS NOT NULL;

COMMIT;