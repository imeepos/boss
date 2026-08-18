BEGIN;
DELETE FROM permissions WHERE code IN ('menu:user', 'menu:userdata');
COMMIT;
