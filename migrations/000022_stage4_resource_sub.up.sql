-- 阶段4:网络资源子表(调拨单/扩容单/预占记录/端口变更历史)。
-- 字段权威:server-ts/src/entities/oss.ts。
BEGIN;

CREATE TABLE transfers (
    id                BIGSERIAL PRIMARY KEY,
    transfer_no       VARCHAR(32) NOT NULL UNIQUE,
    resource_id       BIGINT NOT NULL,   -- 软引用 resources
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    from_region_id    INTEGER NOT NULL,
    to_region_id      INTEGER NOT NULL,
    status            VARCHAR(16) NOT NULL DEFAULT 'PENDING' -- PENDING/DOING/DONE
);

CREATE TABLE expansions (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    expansion_no    VARCHAR(32) NOT NULL UNIQUE,
    region_id       INTEGER NOT NULL,
    expected_ports  INTEGER NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'PENDING' -- PENDING/DOING/DONE
);

CREATE TABLE reserve_records (
    id       BIGSERIAL PRIMARY KEY,
    port_id  BIGINT NOT NULL,   -- 软引用 ports
    order_id BIGINT NOT NULL,   -- 软引用 orders
    status   VARCHAR(16) NOT NULL -- HELD/RELEASED/CONSUMED
);
CREATE INDEX idx_reserve_port ON reserve_records(port_id);

CREATE TABLE port_change_history (
    id         BIGSERIAL PRIMARY KEY,
    port_id    BIGINT NOT NULL REFERENCES ports(id),
    status     VARCHAR(16) NOT NULL,   -- IDLE/RESERVED/USED/DISABLED
    order_id   BIGINT,                 -- 占用订单快照
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_port_history ON port_change_history(port_id, changed_at);

COMMIT;
