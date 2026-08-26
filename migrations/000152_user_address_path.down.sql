BEGIN;
ALTER TABLE user_addresses DROP COLUMN address_path;
COMMIT;
