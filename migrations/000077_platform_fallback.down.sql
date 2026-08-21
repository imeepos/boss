BEGIN;
UPDATE regions
SET legal_entity_id = NULL
WHERE path = 'root' AND legal_entity_id = (SELECT id FROM legal_entities WHERE is_platform);
DROP INDEX IF EXISTS uq_legal_entities_platform;
ALTER TABLE legal_entities DROP COLUMN IF EXISTS is_platform;
DELETE FROM legal_entities WHERE code = 'LEG-PLAT';
COMMIT;
