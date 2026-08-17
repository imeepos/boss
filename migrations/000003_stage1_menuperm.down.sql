BEGIN;
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code LIKE 'menu:%'
);
DELETE FROM permissions WHERE code LIKE 'menu:%';
COMMIT;
