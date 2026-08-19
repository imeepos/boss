-- 发票税务(TAX/AG-04):发票 + ARN 连续发号(CT-007 出账→开票)。
BEGIN;

-- ARN 序列:发票/收据各自连续(TAX-004);UPDATE..RETURNING 原子占用,
-- 与发票 INSERT 同事务,回滚则号回退,保证连续无跳号。
CREATE TABLE arn_sequences (
    doc_type VARCHAR(16) PRIMARY KEY,          -- INVOICE / RECEIPT
    prefix   VARCHAR(8)  NOT NULL,             -- INV- / OR-
    next_no  BIGINT       NOT NULL DEFAULT 1,  -- 下一号
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO arn_sequences(doc_type, prefix) VALUES ('INVOICE', 'INV-'), ('RECEIPT', 'OR-');

CREATE TABLE invoices (
    id            BIGSERIAL PRIMARY KEY,
    invoice_no    VARCHAR(32) NOT NULL UNIQUE,      -- ARN: INV-00000001,作废保留不回收
    bill_id       BIGINT NOT NULL REFERENCES bills(id),
    bill_no       VARCHAR(32) NOT NULL,
    customer_id   BIGINT NOT NULL,
    customer_name VARCHAR(64) NOT NULL,             -- 快照
    title         VARCHAR(128) NOT NULL DEFAULT '', -- 发票抬头,默认同客户名
    net_amount    NUMERIC(12,2) NOT NULL,           -- 不含税净额(=账单金额)
    vat_rate      NUMERIC(5,4)  NOT NULL DEFAULT 0.1200,
    vat_amount    NUMERIC(12,2) NOT NULL,           -- = ROUND(net*rate,2),GEN-006
    total_amount  NUMERIC(12,2) NOT NULL,           -- = net + vat,TAX-002
    status        VARCHAR(16) NOT NULL DEFAULT 'ISSUED', -- ISSUED 已生成 / VOIDED 已作废(编号保留)
    void_reason   VARCHAR(255) NOT NULL DEFAULT '',
    issued_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    voided_at     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- 同一账单仅一张在发票(CT-007 幂等:同账期不重复开票);作废重开为多行历史。
CREATE UNIQUE INDEX uq_invoices_bill_issued ON invoices(bill_id) WHERE status = 'ISSUED';
CREATE INDEX idx_invoices_customer ON invoices(customer_id);

COMMIT;
