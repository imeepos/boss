-- 000181 down:严格逆序回滚(先事实表后白名单,再权限)。
BEGIN;
DROP TABLE IF EXISTS monthly_finance_cost;
DROP TABLE IF EXISTS monthly_network_delivery;
DROP TABLE IF EXISTS monthly_user_revenue;
DROP TABLE IF EXISTS monthly_regions;
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code = 'menu:monthly'
);
DELETE FROM permissions WHERE code = 'menu:monthly';
COMMIT;
