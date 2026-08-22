-- 回滚组织架构与人员菜单权限。
BEGIN;
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:staff');
DELETE FROM permissions WHERE code = 'menu:staff';
COMMIT;
