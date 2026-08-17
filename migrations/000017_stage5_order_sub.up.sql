-- 阶段5:订单子表(派单工单 / 报障工单 / 扫码绑定)。
-- 字段权威:server-ts/src/entities/order.ts。
BEGIN;

CREATE TABLE dispatch_tickets (
    id                BIGSERIAL PRIMARY KEY,
    ticket_no         VARCHAR(32) NOT NULL UNIQUE,
    order_id          BIGINT NOT NULL UNIQUE REFERENCES orders(id), -- 订单1:1
    worker_id         BIGINT REFERENCES workers(id),   -- 指派师傅,可空
    worker_name       VARCHAR(64),
    group_id          BIGINT REFERENCES worker_groups(id),
    group_name        VARCHAR(64),
    region_id         INTEGER,                          -- 服务区域快照,可空
    region_name       VARCHAR(64),
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    status            VARCHAR(16) NOT NULL DEFAULT 'PENDING' -- PENDING/DOING/DONE/CANCELED
);
CREATE INDEX idx_dispatch_tickets_worker ON dispatch_tickets(worker_id);

CREATE TABLE complaints (
    id                BIGSERIAL PRIMARY KEY,
    ticket_no         VARCHAR(32) NOT NULL UNIQUE,
    customer_id       BIGINT NOT NULL REFERENCES customers(id),
    order_id          BIGINT REFERENCES orders(id),     -- 关联订单,可空
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    type              VARCHAR(32) NOT NULL,             -- 报障类型
    status            VARCHAR(16) NOT NULL DEFAULT 'OPEN' -- OPEN/PROCESSING/CLOSED
);
CREATE INDEX idx_complaints_customer ON complaints(customer_id);

CREATE TABLE scan_logs (
    id          BIGSERIAL PRIMARY KEY,
    order_id    BIGINT NOT NULL REFERENCES orders(id),
    worker_id   BIGINT NOT NULL,
    worker_name VARCHAR(64) NOT NULL,
    tag_id      BIGINT NOT NULL,                        -- 软引用 tags
    result      VARCHAR(16) NOT NULL                    -- MATCH/MISMATCH/OFFLINE_CACHED
);
CREATE INDEX idx_scan_logs_order ON scan_logs(order_id);

COMMIT;
