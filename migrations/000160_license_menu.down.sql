-- 回滚:撤销 sysadmin 的 menu:license 授权并删除权限码。
BEGIN;
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:license');

DELETE FROM permissions WHERE code = 'menu:license';
COMMIT;