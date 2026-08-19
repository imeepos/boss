BEGIN;
DROP INDEX IF EXISTS idx_invoices_tax_status;
ALTER TABLE invoices
  DROP COLUMN IF EXISTS tax_jurisdiction,
  DROP COLUMN IF EXISTS tax_channel,
  DROP COLUMN IF EXISTS tax_status,
  DROP COLUMN IF EXISTS tax_no,
  DROP COLUMN IF EXISTS tax_fail_reason;
COMMIT;
