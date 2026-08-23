-- 渠道佣金结算台账：按渠道订单记录应计、结算与作废，不涉及跨运营商批发结算。
BEGIN;
CREATE TABLE partner_commission_ledger (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id),
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    order_amount NUMERIC(18,2) NOT NULL CHECK (order_amount >= 0),
    commission_rate NUMERIC(8,5) NOT NULL CHECK (commission_rate >= 0 AND commission_rate <= 1),
    commission_amount NUMERIC(18,2) NOT NULL CHECK (commission_amount >= 0),
    status VARCHAR(16) NOT NULL DEFAULT 'ACCRUED' CHECK (status IN ('ACCRUED','SETTLED','VOID')),
    settled_at TIMESTAMPTZ,
    settled_by BIGINT REFERENCES accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (order_id, legal_entity_id)
);
CREATE INDEX idx_partner_commission_entity_status ON partner_commission_ledger(legal_entity_id, status, created_at DESC);
COMMIT;
