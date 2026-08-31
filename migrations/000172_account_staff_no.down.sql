BEGIN;

DROP INDEX IF EXISTS uq_accounts_staff_no;
ALTER TABLE accounts DROP COLUMN IF EXISTS staff_no;

COMMIT;
