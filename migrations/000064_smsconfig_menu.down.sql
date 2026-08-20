-- 回滚 000064:摘除 menu:smsconfig 权限及角色绑定。
BEGIN;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:smsconfig');
DELETE FROM permissions WHERE code = 'menu:smsconfig';
COMMIT;
