-- 000165 down:严格逆序回滚
BEGIN;
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('menu:purchase','menu:inventory','menu:install-board')
);
DELETE FROM permissions WHERE code IN ('menu:purchase','menu:inventory','menu:install-board');
COMMIT;
