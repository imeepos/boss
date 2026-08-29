-- 000169 down:收回 product-write 授权并注销权限码。
BEGIN;
DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'menu:product-write');
DELETE FROM permissions WHERE code = 'menu:product-write';
COMMIT;
