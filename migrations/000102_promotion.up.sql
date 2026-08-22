-- promotion 域三新表 + coupons 增量改造(docs/design/promotion-coupon.md)
-- 设计要点:模板/兑换码/核销记录新建;coupons 保留存量,新列全部带默认值,
-- 老 INSERT(userdata.CreateCoupon 只写 6 列)不改即兼容。

CREATE TABLE coupon_templates (
    template_id BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities (id),
    name VARCHAR(128) NOT NULL,
    type VARCHAR(16) NOT NULL DEFAULT 'CASH',
    face_value BIGINT NOT NULL,
    threshold BIGINT NOT NULL DEFAULT 0,
    max_discount BIGINT,
    scope_type VARCHAR(16) NOT NULL DEFAULT 'ALL',
    scope_ref BIGINT,
    total_qty BIGINT NOT NULL DEFAULT 0,
    issued_qty BIGINT NOT NULL DEFAULT 0,
    per_customer_limit INT NOT NULL DEFAULT 1,
    valid_days INT,
    valid_from TIMESTAMPTZ,
    valid_to TIMESTAMPTZ,
    status VARCHAR(16) NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE coupon_codes (
    code_id BIGSERIAL PRIMARY KEY,
    code VARCHAR(32) NOT NULL UNIQUE,
    template_id BIGINT NOT NULL REFERENCES coupon_templates (template_id),
    status VARCHAR(16) NOT NULL DEFAULT 'UNUSED',
    redeemed_by BIGINT REFERENCES customers (id),
    redeemed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE coupon_redemptions (
    redemption_id BIGSERIAL PRIMARY KEY,
    coupon_id VARCHAR(32) NOT NULL REFERENCES coupons (coupon_id),
    payment_id BIGINT NOT NULL,
    customer_id BIGINT NOT NULL,
    deducted_amount BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- coupons 增量改造:状态值映射 + 模板快照列 + 转赠/核销载体列。
ALTER TABLE coupons
    ADD COLUMN template_id BIGINT REFERENCES coupon_templates (template_id),
    ADD COLUMN type VARCHAR(16) NOT NULL DEFAULT 'CASH',
    ADD COLUMN face_value BIGINT,
    ADD COLUMN threshold BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN max_discount BIGINT,
    ADD COLUMN scope_type VARCHAR(16) NOT NULL DEFAULT 'ALL',
    ADD COLUMN scope_ref BIGINT,
    ADD COLUMN source VARCHAR(16) NOT NULL DEFAULT 'ADMIN_ISSUE',
    ADD COLUMN code VARCHAR(32) UNIQUE,
    ADD COLUMN issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN used_at TIMESTAMPTZ,
    ADD COLUMN payment_id BIGINT;

UPDATE coupons SET status = 'ISSUED' WHERE status = 'active';
UPDATE coupons SET status = 'DISABLED' WHERE status = 'disabled';
UPDATE coupons SET status = 'USED', used_at = now() WHERE status = 'used';
UPDATE coupons SET face_value = amount WHERE face_value IS NULL;

ALTER TABLE coupons ALTER COLUMN status SET DEFAULT 'ISSUED';

CREATE INDEX idx_coupons_customer ON coupons (customer_id, status);
CREATE INDEX idx_coupons_template ON coupons (template_id);

-- 赠送时长阶梯规则(6送1/12送3/24送6)与发放记录。
CREATE TABLE gift_rules (
    rule_id BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities (id),
    name VARCHAR(128) NOT NULL,
    scope_type VARCHAR(16) NOT NULL DEFAULT 'ALL',
    scope_ref BIGINT,
    buy_months INT NOT NULL,
    gift_months INT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'ENABLED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE gift_records (
    record_id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT NOT NULL REFERENCES gift_rules (rule_id),
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    product_id BIGINT,
    buy_months INT NOT NULL,
    gift_months INT NOT NULL,
    payment_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
