BEGIN;

DROP TRIGGER IF EXISTS trg_customers_set_code ON customers;
DROP FUNCTION IF EXISTS customers_set_code();

DROP INDEX IF EXISTS uq_customers_customer_code;

ALTER TABLE customers
    DROP COLUMN IF EXISTS customer_code;

COMMIT;