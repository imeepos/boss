BEGIN;

ALTER TABLE legal_entities
  DROP CONSTRAINT IF EXISTS ck_legal_entities_tax_channel,
  DROP CONSTRAINT IF EXISTS ck_legal_entities_tax_jurisdiction,
  DROP COLUMN IF EXISTS tax_channel,
  DROP COLUMN IF EXISTS tax_jurisdiction;

COMMIT;
