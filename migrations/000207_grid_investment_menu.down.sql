-- 回滚 000207:移除 menu:grid-investment 权限登记(先解绑角色,再删权限码)。
BEGIN;
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:grid-investment');
DELETE FROM permissions WHERE code = 'menu:grid-investment';
COMMIT;
