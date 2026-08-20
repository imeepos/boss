-- 渠道对账批次行级明细(db-design-review D6):逐行定位差异 payment。
BEGIN;

CREATE TABLE reconciliation_items (
    id          BIGSERIAL PRIMARY KEY,
    batch_id    BIGINT NOT NULL REFERENCES reconciliation_batches(id) ON DELETE CASCADE,
    -- payment_id 为软引用:渠道侧流水可能无对应系统 payment(MISSING_SYSTEM 行 payment_id=NULL),
    -- 故不设 FK;系统侧存在时仅存 id 快照。
    payment_id  BIGINT,
    channel_ref VARCHAR(64) NOT NULL,             -- 渠道流水号(对齐 payments.pay_no 比对)
    amount      NUMERIC(14,2) NOT NULL,
    diff_kind   VARCHAR(16) NOT NULL CHECK (diff_kind IN
        ('MATCH','MISSING_SYSTEM','MISSING_CHANNEL','AMOUNT_MISMATCH')),
    note        VARCHAR(255) NOT NULL DEFAULT ''
);
CREATE INDEX idx_recon_items_batch ON reconciliation_items(batch_id);

COMMIT;
