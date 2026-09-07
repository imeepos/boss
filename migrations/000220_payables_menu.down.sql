-- 回滚 000220:移除 menu:payables 权限登记(先解绑角色,再删权限码)。
BEGIN;
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:payables');
DELETE FROM permissions WHERE code = 'menu:payables';
COMMIT;
