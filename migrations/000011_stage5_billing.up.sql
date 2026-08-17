-- 阶段5:计费账务(周期出账 + 缴费流水)。
-- 字段权威:docs/contract/fields.md §3.3 + server-ts/src/entities/worker.ts。
BEGIN;

CREATE TABLE bills (
    id                BIGSERIAL PRIMARY KEY,
    bill_no           VARCHAR(32) NOT NULL UNIQUE,  -- BILL-202608-201
    customer_id       BIGINT NOT NULL REFERENCES customers(id),
    customer_name     VARCHAR(64) NOT NULL,          -- 快照
    legal_entity_id   BIGINT NOT NULL,               -- 企业归属快照
    legal_entity_name VARCHAR(128) NOT NULL,
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    period            VARCHAR(16) NOT NULL,          -- 账期 2026-08
    amount            NUMERIC(12,2) NOT NULL,        -- 账单金额=订单成交价快照
    status            VARCHAR(16) NOT NULL DEFAULT 'UNPAID', -- UNPAID/PAID/OVERDUE
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_bills_customer_period UNIQUE (customer_id, period)
);
CREATE INDEX idx_bills_customer ON bills(customer_id);

CREATE TABLE payments (
    id         BIGSERIAL PRIMARY KEY,
    pay_no     VARCHAR(32) NOT NULL UNIQUE,          -- PAY-20260820-001
    bill_id    BIGINT NOT NULL REFERENCES bills(id),
    amount     NUMERIC(12,2) NOT NULL,
    method     VARCHAR(16) NOT NULL,                 -- wechat/alipay/card/cash
    status     VARCHAR(16) NOT NULL DEFAULT 'SUCCESS', -- SUCCESS/FAILED/REFUNDED
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_payments_bill ON payments(bill_id);

COMMIT;
