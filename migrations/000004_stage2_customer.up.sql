-- 阶段2:客户档案(REQ-CRM-001 一人一档)。
-- 字段权威:docs/contract/fields.md §2.1 + server-ts/src/entities/customer.ts。
BEGIN;

CREATE TABLE customers (
    id               BIGSERIAL PRIMARY KEY,
    legal_entity_id  BIGINT NOT NULL REFERENCES legal_entities(id),
    address_id       BIGINT NOT NULL REFERENCES addresses(id),
    region_id        INTEGER NOT NULL,             -- 地址所在经营区域快照(regions.id)
    region_name      VARCHAR(64) NOT NULL,         -- 区域名快照:改名不改历史
    name             VARCHAR(64) NOT NULL,
    phone            VARCHAR(32) NOT NULL,
    id_type          VARCHAR(16) NOT NULL,         -- 身份证/护照/营业执照/无
    id_no            VARCHAR(64),                  -- 证件号(敏感,默认不查)
    real_name_status VARCHAR(16) NOT NULL DEFAULT 'PENDING',  -- VERIFIED/PENDING
    service_status   VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',   -- ACTIVE/ARREARS/SUSPENDED
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_customers_legal_entity ON customers(legal_entity_id);
CREATE INDEX idx_customers_address ON customers(address_id);
CREATE INDEX idx_customers_phone ON customers(phone);

COMMIT;
