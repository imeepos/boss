-- 回滚 000069:摘除 menu:realidconfig 权限及角色绑定。
BEGIN;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:realidconfig');
DELETE FROM permissions WHERE code = 'menu:realidconfig';
COMMIT;
