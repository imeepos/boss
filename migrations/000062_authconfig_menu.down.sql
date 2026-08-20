-- 回滚 000062:摘除 menu:authconfig 权限及角色绑定。
BEGIN;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:authconfig');
DELETE FROM permissions WHERE code = 'menu:authconfig';
COMMIT;
