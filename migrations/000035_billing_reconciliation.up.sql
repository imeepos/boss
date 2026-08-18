-- 缴费渠道对账批次(billing.yaml /reconciliations)。
BEGIN;

CREATE TABLE reconciliation_batches (
    id             BIGSERIAL PRIMARY KEY,
    batch_no       VARCHAR(32) NOT NULL UNIQUE,   -- 对账批次号 PC-YYYYMMDD-NN
    channel        VARCHAR(32) NOT NULL,          -- 微信/支付宝/线下营业厅
    channel_amount NUMERIC(14,2) NOT NULL,        -- 渠道侧金额
    system_amount  NUMERIC(14,2) NOT NULL,        -- 系统侧金额
    status         VARCHAR(16) NOT NULL,          -- DIFF_PENDING 差异挂起 / SETTLED 已平账
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    settled_at     TIMESTAMPTZ                    -- null=未平账
);
CREATE INDEX idx_recon_status ON reconciliation_batches(status, batch_no);

COMMIT;
