-- Q3 发票税局事件轨迹:提交/回执/回填/作废/重开留痕,补齐"可追踪"验收。
-- 现状只有终态(tax_status/tax_no/tax_fail_reason),无历史轨迹可回放。
BEGIN;

CREATE TABLE invoice_tax_events (
    id                  BIGSERIAL PRIMARY KEY,
    invoice_id          BIGINT NOT NULL REFERENCES invoices(id),
    event               VARCHAR(24) NOT NULL,   -- RECEIPT/BACKFILL/VOID/REISSUE
    tax_status_after    VARCHAR(16) NOT NULL DEFAULT '', -- 事件后税局状态(VOID/REISSUE 记当时税局状态)
    tax_no              VARCHAR(64) NOT NULL DEFAULT '',
    fail_reason         VARCHAR(255) NOT NULL DEFAULT '',
    operator_account_id BIGINT NOT NULL DEFAULT 0, -- 0=网关/系统自动
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_invoice_tax_events_invoice ON invoice_tax_events(invoice_id);

COMMIT;
