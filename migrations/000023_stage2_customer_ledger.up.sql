-- 阶段2:客户与资费台账(客户归属台账 / 产品调价台账 / 区域调价台账)。
-- 字段权威:server-ts/src/entities/customer.ts。
BEGIN;

CREATE TABLE customer_histories (
    id                  BIGSERIAL PRIMARY KEY,
    customer_id         BIGINT NOT NULL REFERENCES customers(id),
    legal_entity_id     BIGINT NOT NULL REFERENCES legal_entities(id),
    legal_entity_name   VARCHAR(128) NOT NULL,
    address_id          BIGINT NOT NULL REFERENCES addresses(id),
    address_name        VARCHAR(64) NOT NULL,
    region_id           INTEGER NOT NULL,
    region_name         VARCHAR(64) NOT NULL,
    reason              VARCHAR(128),
    operator_account_id BIGINT,
    effective_from      TIMESTAMPTZ NOT NULL,
    effective_to        TIMESTAMPTZ               -- null=至今
);
CREATE INDEX idx_customer_histories ON customer_histories(customer_id, effective_from);

CREATE TABLE product_price_histories (
    id                  BIGSERIAL PRIMARY KEY,
    offer_id            BIGINT NOT NULL REFERENCES product_offers(id),
    old_monthly_fee     NUMERIC(10,2) NOT NULL,
    new_monthly_fee     NUMERIC(10,2) NOT NULL,
    effective_at        TIMESTAMPTZ NOT NULL,
    reason              VARCHAR(128),
    operator_account_id BIGINT
);
CREATE INDEX idx_product_price ON product_price_histories(offer_id, effective_at);

CREATE TABLE region_price_histories (
    id                  BIGSERIAL PRIMARY KEY,
    region_offer_id     BIGINT NOT NULL REFERENCES region_offers(id),
    old_monthly_fee     NUMERIC(10,2) NOT NULL,
    new_monthly_fee     NUMERIC(10,2) NOT NULL,
    effective_at        TIMESTAMPTZ NOT NULL,
    reason              VARCHAR(128),
    operator_account_id BIGINT
);
CREATE INDEX idx_region_price ON region_price_histories(region_offer_id, effective_at);

COMMIT;
