-- 税局网关层(多属地:CN 数电票 / PH BIR eIS);纯增量,不改变既有开票链路与语义。
-- 系统内发票=开票意图+内部留痕;法定票号经税局通道开具后回填 tax_no。
BEGIN;

ALTER TABLE invoices
  ADD COLUMN tax_jurisdiction VARCHAR(8)   NOT NULL DEFAULT '',       -- CN/PH,空=未定
  ADD COLUMN tax_channel      VARCHAR(16)  NOT NULL DEFAULT 'manual', -- manual/leqi/bir_eis
  ADD COLUMN tax_status       VARCHAR(16)  NOT NULL DEFAULT 'PENDING',-- PENDING/SUBMITTED/ISSUED/FAILED
  ADD COLUMN tax_no           VARCHAR(64)  NOT NULL DEFAULT '',       -- CN 数电票号(20位)/PH BIR 回执号
  ADD COLUMN tax_fail_reason  VARCHAR(255) NOT NULL DEFAULT '';
CREATE INDEX idx_invoices_tax_status ON invoices(tax_status);

COMMIT;
