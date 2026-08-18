-- 回滚三类用户登录字段与约束。
BEGIN;
DROP INDEX IF EXISTS uq_customers_app_login_phone;
ALTER TABLE customers
    DROP CONSTRAINT IF EXISTS customers_auth_status_check,
    DROP COLUMN IF EXISTS auth_status,
    DROP COLUMN IF EXISTS password_hash;
ALTER TABLE workers
    DROP CONSTRAINT IF EXISTS workers_status_check,
    DROP COLUMN IF EXISTS password_hash;
COMMIT;
