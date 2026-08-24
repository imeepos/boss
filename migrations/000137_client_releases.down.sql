BEGIN;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:release');
DELETE FROM permissions WHERE code = 'menu:release';
DROP TABLE IF EXISTS client_releases;
COMMIT;
