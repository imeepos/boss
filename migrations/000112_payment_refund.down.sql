BEGIN;

ALTER TABLE payments
  DROP COLUMN IF EXISTS refunded_at,
  DROP COLUMN IF EXISTS refund_reason;

COMMIT;
