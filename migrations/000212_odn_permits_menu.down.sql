-- 回滚 000212:移除 menu:permits 权限登记(先解绑角色,再删权限码)。
BEGIN;
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:permits');
DELETE FROM permissions WHERE code = 'menu:permits';
COMMIT;