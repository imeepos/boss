BEGIN;

ALTER TABLE portal_accounts DROP CONSTRAINT chk_portal_customer_id_sign;

COMMIT;
