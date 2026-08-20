BEGIN;
ALTER TABLE portal_sms_codes DROP COLUMN issued_at;
COMMIT;
