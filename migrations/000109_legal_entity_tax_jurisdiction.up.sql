-- 税务属地配置(TAX/Q3):开票主体(法人)决定税务属地与通道。
-- 发票开具时经 bills.legal_entity_id 快照落 invoices.tax_jurisdiction/tax_channel,
-- 使 tax-submit 能按属地路由到对应税局网关(CN 数电票 / PH BIR eIS)。
BEGIN;

ALTER TABLE legal_entities
  ADD COLUMN tax_jurisdiction VARCHAR(8)  NOT NULL DEFAULT '',       -- CN/PH,空=未定(人工通道)
  ADD COLUMN tax_channel      VARCHAR(16) NOT NULL DEFAULT 'manual', -- manual/leqi/bir_eis
  ADD CONSTRAINT ck_legal_entities_tax_jurisdiction
    CHECK (tax_jurisdiction IN ('', 'CN', 'PH')),
  ADD CONSTRAINT ck_legal_entities_tax_channel
    CHECK (tax_channel IN ('manual', 'leqi', 'bir_eis'));

COMMIT;
