-- 阶段4:网络资源(设备树 OLT/分光器 + 端口)。
-- 字段权威:docs/contract/fields.md §4.2 + server-ts/src/entities/oss.ts。
BEGIN;

CREATE TABLE resources (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    code            VARCHAR(32) NOT NULL UNIQUE,  -- OLT-01/SPL-01
    name            VARCHAR(64) NOT NULL,
    type            VARCHAR(16) NOT NULL,         -- OLT/SPLITTER
    parent_id       BIGINT REFERENCES resources(id), -- 上级设备(树内上下级)
    address_id      BIGINT NOT NULL REFERENCES addresses(id),
    status          VARCHAR(16) NOT NULL DEFAULT 'ONLINE', -- ONLINE/OFFLINE/FAULT
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_resources_parent ON resources(parent_id);
CREATE INDEX idx_resources_entity ON resources(legal_entity_id);

CREATE TABLE ports (
    id                BIGSERIAL PRIMARY KEY,
    port_code         VARCHAR(32) NOT NULL UNIQUE,  -- P-SPL01-01
    quad_code         VARCHAR(32) NOT NULL,         -- 四码端口码 P-SPLxx-yy
    resource_id       BIGINT NOT NULL REFERENCES resources(id),
    legal_entity_id   BIGINT NOT NULL,              -- 所属设备企业快照
    legal_entity_name VARCHAR(128) NOT NULL,
    address_id        BIGINT NOT NULL REFERENCES addresses(id),
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    order_id          BIGINT,                       -- 占用订单,空=空闲
    status            VARCHAR(16) NOT NULL DEFAULT 'IDLE', -- IDLE/RESERVED/USED/DISABLED
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ports_resource ON ports(resource_id);
CREATE INDEX idx_ports_address ON ports(address_id);

COMMIT;
