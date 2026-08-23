BEGIN;
ALTER TABLE invoice_tax_events ADD COLUMN external_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE UNIQUE INDEX uq_invoice_tax_events_external_id
  ON invoice_tax_events(invoice_id, external_id) WHERE external_id <> '';
COMMIT;
