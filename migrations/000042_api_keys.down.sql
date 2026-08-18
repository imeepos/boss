-- 回滚 000042:移除 api_keys 表及 menu:apikey 权限。
BEGIN;
DROP TABLE IF EXISTS api_keys;
DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'menu:apikey');
DELETE FROM permissions WHERE code = 'menu:apikey';
COMMIT;