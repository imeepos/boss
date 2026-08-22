BEGIN;

ALTER TABLE orders DROP CONSTRAINT ck_orders_billing_mode;
ALTER TABLE orders DROP COLUMN billing_mode;

ALTER TABLE lo_accounts DROP CONSTRAINT ck_lo_accounts_billing_mode;
ALTER TABLE lo_accounts DROP COLUMN billing_mode;

COMMIT;
