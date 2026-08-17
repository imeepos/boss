-- 阶段2:产品资费三级模型(产品目录 + 区域运营包)。
-- 字段权威:docs/contract/fields.md §2.2 + server-ts/src/entities/customer.ts。
BEGIN;

CREATE TABLE product_offers (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    name            VARCHAR(128) NOT NULL,   -- 公司级名称,必填
    bandwidth       VARCHAR(16) NOT NULL,    -- 如 300M/500M/1000M
    monthly_fee     NUMERIC(10,2) NOT NULL,  -- 基础月费
    effective_at    TIMESTAMPTZ NOT NULL,    -- 上架/调价生效时间
    status          VARCHAR(16) NOT NULL DEFAULT 'DRAFT', -- DRAFT/PUBLISHED/OFFLINE
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_product_offers_entity_name UNIQUE (legal_entity_id, name)
);
CREATE INDEX idx_product_offers_entity ON product_offers(legal_entity_id);

CREATE TABLE region_offers (
    id          BIGSERIAL PRIMARY KEY,
    offer_id    BIGINT NOT NULL REFERENCES product_offers(id),
    region_path VARCHAR(128) NOT NULL,       -- 区域 ltree 路径,须落在公司经营区域
    name        VARCHAR(128),                -- 区域级产品名,可空回退公司名
    monthly_fee NUMERIC(10,2) NOT NULL,      -- 区域月费,覆盖基础价
    reason      VARCHAR(128),                -- 调价原因
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_region_offers_offer_region UNIQUE (offer_id, region_path)
);
CREATE INDEX idx_region_offers_offer ON region_offers(offer_id);

COMMIT;
