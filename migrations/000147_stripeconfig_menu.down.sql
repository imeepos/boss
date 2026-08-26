-- 回滚 000147:摘除 menu:stripeconfig 权限及角色绑定。
BEGIN;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:stripeconfig');
DELETE FROM permissions WHERE code = 'menu:stripeconfig';
COMMIT;