BEGIN;
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:pushconfig');
DELETE FROM permissions WHERE code = 'menu:pushconfig';
COMMIT;
