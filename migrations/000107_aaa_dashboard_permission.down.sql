BEGIN;

DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'menu:aaadashboard');
DELETE FROM permissions WHERE code = 'menu:aaadashboard';

COMMIT;
