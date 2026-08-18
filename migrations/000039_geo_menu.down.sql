-- 回滚 000039:摘除 menu:geo 权限及角色绑定。
BEGIN;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:geo');
DELETE FROM permissions WHERE code = 'menu:geo';
COMMIT;
