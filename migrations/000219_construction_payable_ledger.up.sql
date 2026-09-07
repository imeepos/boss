-- 000219(P-INFRA-1 W6,审查 F8): 工程应付台账。
-- SETTLED 结算单同事务自动生成应付记录(金额=结算应付,来源单据引用 settlement_id 唯一);
-- 台账承载付款计划口径(部分付款/分期=多次付款流水至余额耗尽)/已付/未付余额/付款流水/发票登记;
-- 核减=复审减项,append-only 明细留痕(原因必填),净应付=应付-核减合计,只读派生不落列;
-- 结算单 VOIDED 同事务冲销应付(status=VOIDED,原因同源),付款流水保留为历史,不做退款单复杂化。
BEGIN;

CREATE TABLE construction_payables (
    id              BIGSERIAL PRIMARY KEY,
    payable_no      VARCHAR(32) NOT NULL UNIQUE,
    settlement_id   BIGINT NOT NULL UNIQUE REFERENCES construction_settlements(id),
    settlement_no   VARCHAR(32) NOT NULL,
    project_id      BIGINT NOT NULL,
    project_no      VARCHAR(32) NOT NULL,
    contractor_id   BIGINT,
    contractor_name VARCHAR(128) NOT NULL DEFAULT '',
    payable_amount  NUMERIC(14,2) NOT NULL CHECK (payable_amount >= 0),
    status          VARCHAR(16) NOT NULL DEFAULT 'OPEN'
                    CHECK (status IN ('OPEN','PARTIAL','PAID','VOIDED')),
    void_reason     VARCHAR(255) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_construction_payables_project ON construction_payables(project_id);
CREATE INDEX idx_construction_payables_status  ON construction_payables(status);

CREATE TABLE construction_payable_payments (
    id         BIGSERIAL PRIMARY KEY,
    payable_id BIGINT NOT NULL REFERENCES construction_payables(id),
    payment_no VARCHAR(32) NOT NULL UNIQUE,
    amount     NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    method     VARCHAR(16) NOT NULL DEFAULT 'TRANSFER'
               CHECK (method IN ('TRANSFER','CASH','CHEQUE','OTHER')),
    paid_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    reference  VARCHAR(64) NOT NULL DEFAULT '',
    note       VARCHAR(255) NOT NULL DEFAULT '',
    created_by BIGINT REFERENCES accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_construction_payable_payments_payable ON construction_payable_payments(payable_id);

CREATE TABLE construction_payable_deductions (
    id         BIGSERIAL PRIMARY KEY,
    payable_id BIGINT NOT NULL REFERENCES construction_payables(id),
    amount     NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    reason     VARCHAR(255) NOT NULL,
    created_by BIGINT REFERENCES accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_construction_payable_deductions_payable ON construction_payable_deductions(payable_id);

CREATE TABLE construction_payable_invoices (
    id         BIGSERIAL PRIMARY KEY,
    payable_id BIGINT NOT NULL REFERENCES construction_payables(id),
    invoice_no VARCHAR(64) NOT NULL,
    amount     NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    invoiced_at DATE,
    note       VARCHAR(255) NOT NULL DEFAULT '',
    created_by BIGINT REFERENCES accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (payable_id, invoice_no)
);
CREATE INDEX idx_construction_payable_invoices_payable ON construction_payable_invoices(payable_id);

COMMENT ON TABLE  construction_payables            IS '工程应付台账(SETTLED 结算单自动生成,000219,P-INFRA-1 W6 审查 F8)';
COMMENT ON COLUMN construction_payables.payable_no     IS '应付单号 AP-YYYYMMDD-NNNNN(后端生成兜底)';
COMMENT ON COLUMN construction_payables.settlement_id  IS '来源结算单 1:1 唯一;VOIDED 冲销后保留历史';
COMMENT ON COLUMN construction_payables.payable_amount IS '应付金额=结算应付(settlements.total_amount 快照)';
COMMENT ON COLUMN construction_payables.status         IS 'OPEN 未付/PARTIAL 部分付款/PAID 已付清/VOIDED 已冲销;按付款与核减流水派生维护';
COMMENT ON COLUMN construction_payables.void_reason    IS '冲销原因(随结算单作废原因同源)';
COMMENT ON TABLE construction_payable_deductions IS '核减明细 append-only:原因必填;净应付=应付-核减合计,核减后净应付不得低于已付(防超付)';
COMMENT ON TABLE construction_payable_payments   IS '付款流水登记:部分付款与分期=多次登记至余额耗尽;累计不得超净应付';
COMMENT ON TABLE construction_payable_invoices   IS '发票登记:纯登记不联动税局;同应付内发票号唯一';

COMMIT;
