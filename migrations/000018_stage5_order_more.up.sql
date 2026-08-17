-- 阶段5:订单子表(拆机单 / 激活回调 / 改派台账)。
-- 字段权威:server-ts/src/entities/order.ts。
BEGIN;

CREATE TABLE dismantles (
    id                BIGSERIAL PRIMARY KEY,
    dismantle_no      VARCHAR(32) NOT NULL UNIQUE,
    order_id          BIGINT NOT NULL REFERENCES orders(id),
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    asset_id          BIGINT NOT NULL,  -- 软引用 assets(回收资产)
    port_id           BIGINT NOT NULL,  -- 软引用 ports(释放端口)
    status            VARCHAR(16) NOT NULL DEFAULT 'PENDING' -- PENDING/DOING/DONE/FAILED
);

CREATE TABLE activation_callbacks (
    id       BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id),
    result   VARCHAR(16) NOT NULL,      -- SUCCESS/FAILED
    retries  SMALLINT NOT NULL DEFAULT 0
);
CREATE INDEX idx_activation_callbacks_order ON activation_callbacks(order_id);

CREATE TABLE dispatch_transfers (
    id                  BIGSERIAL PRIMARY KEY,
    ticket_id           BIGINT NOT NULL REFERENCES dispatch_tickets(id),
    from_worker_id      BIGINT REFERENCES workers(id),
    from_worker_name    VARCHAR(64),
    to_worker_id        BIGINT REFERENCES workers(id),
    to_worker_name      VARCHAR(64),
    reason              VARCHAR(128),
    operator_account_id BIGINT,
    transferred_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_dispatch_transfers_ticket ON dispatch_transfers(ticket_id, transferred_at);

COMMIT;
