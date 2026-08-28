-- 000168 down:收回 ops 的 menu:install-board 授权。
BEGIN;
DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'menu:install-board')
  AND role_id = (SELECT id FROM roles WHERE code = 'ops');
COMMIT;
