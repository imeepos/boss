-- 阶段5:订单 + 环节时间轴(12 环节)。
-- 字段权威:docs/contract/fields.md §3.1/§3.2 + server-ts/src/entities/order.ts。
BEGIN;

CREATE TABLE orders (
    id              BIGSERIAL PRIMARY KEY,
    order_no        VARCHAR(32) NOT NULL UNIQUE,  -- ORD-20250817-001
    customer_id     BIGINT NOT NULL,
    offer_id        BIGINT NOT NULL,              -- → product_offers
    address_id      BIGINT NOT NULL,
    stage           SMALLINT NOT NULL DEFAULT 1,  -- 1~12
    status          VARCHAR(16) NOT NULL DEFAULT 'PENDING', -- PENDING/RESERVED/INSTALLING/DONE/CANCELLED
    channel_id      BIGINT NOT NULL,              -- → channels(REQ-ORD-006)
    legal_entity_id BIGINT NOT NULL,              -- 企业归属快照
    region_path     VARCHAR(128),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_orders_customer ON orders(customer_id);
CREATE INDEX idx_orders_offer ON orders(offer_id);

CREATE TABLE order_stages (
    id          BIGSERIAL PRIMARY KEY,
    order_id    BIGINT NOT NULL REFERENCES orders(id),
    stage       SMALLINT NOT NULL,                 -- 1~12
    result      VARCHAR(8) NOT NULL DEFAULT 'PENDING', -- PENDING/DOING/DONE
    retries     SMALLINT NOT NULL DEFAULT 0,
    finished_at TIMESTAMPTZ
);
CREATE INDEX idx_order_stages_order ON order_stages(order_id, stage);

COMMIT;
