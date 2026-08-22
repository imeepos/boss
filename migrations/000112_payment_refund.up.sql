-- 退款收口(BILL/Q3):payments 加退款留痕列;全额退款=流水置 REFUNDED + 账单回 UNPAID。
BEGIN;

ALTER TABLE payments
  ADD COLUMN refund_reason VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN refunded_at  TIMESTAMPTZ;

COMMIT;
