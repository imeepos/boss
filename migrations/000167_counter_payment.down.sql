-- 柜面收款回滚:删权限码绑定/权限码、日结表、payments 凭证要素列。
BEGIN;

DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('menu:payment:cash','menu:daily-close'));
DELETE FROM permissions WHERE code IN ('menu:payment:cash','menu:daily-close');

DROP TABLE IF EXISTS payment_daily_closings;
DROP INDEX IF EXISTS idx_payments_site_date;
ALTER TABLE payments
    DROP COLUMN IF EXISTS site_name,
    DROP COLUMN IF EXISTS counter_code,
    DROP COLUMN IF EXISTS operator_name;

COMMIT;
