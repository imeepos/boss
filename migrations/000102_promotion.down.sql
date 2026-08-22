DROP INDEX IF EXISTS idx_coupons_template;
DROP INDEX IF EXISTS idx_coupons_customer;
ALTER TABLE coupons
    ALTER COLUMN status SET DEFAULT 'active',
    DROP COLUMN payment_id,
    DROP COLUMN used_at,
    DROP COLUMN issued_at,
    DROP COLUMN code,
    DROP COLUMN source,
    DROP COLUMN scope_ref,
    DROP COLUMN scope_type,
    DROP COLUMN max_discount,
    DROP COLUMN threshold,
    DROP COLUMN face_value,
    DROP COLUMN type,
    DROP COLUMN template_id;
UPDATE coupons SET status = 'active' WHERE status = 'ISSUED';
UPDATE coupons SET status = 'disabled' WHERE status = 'DISABLED';
UPDATE coupons SET status = 'used' WHERE status = 'USED';
DROP TABLE coupon_redemptions;
DROP TABLE coupon_codes;
DROP TABLE coupon_templates;
DROP TABLE gift_records;
DROP TABLE gift_rules;
