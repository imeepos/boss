-- 最小 LOY 积分域(loy):积分账本 + 流水;积分换券价挂券模板(points_price)。
-- 范围裁定:等级/任务/缴费自动积分属完整 LOY(见 3-month-roadmap 排期),本期只做
-- 账本+手动调整+兑换换券闭环;LOY→PROMO 经服务调用(非同事务,失败补偿回补)。

CREATE TABLE loy_point_ledgers (
    customer_id BIGINT PRIMARY KEY REFERENCES customers (id),
    balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE loy_point_entries (
    entry_id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    delta BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    reason VARCHAR(64) NOT NULL,
    ref_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_loy_entries_customer ON loy_point_entries (customer_id, entry_id);

ALTER TABLE coupon_templates
    ADD COLUMN points_price BIGINT NOT NULL DEFAULT 0;
