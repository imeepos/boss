BEGIN;
DROP INDEX IF EXISTS uq_invoice_tax_events_external_id;
ALTER TABLE invoice_tax_events DROP COLUMN IF EXISTS external_id;
COMMIT;
