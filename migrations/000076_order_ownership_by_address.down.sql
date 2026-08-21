BEGIN;
DROP INDEX IF EXISTS idx_addresses_region;
ALTER TABLE addresses DROP COLUMN IF EXISTS region_id;
DROP INDEX IF EXISTS idx_regions_legal_entity;
ALTER TABLE regions DROP COLUMN IF EXISTS legal_entity_id;
COMMIT;
