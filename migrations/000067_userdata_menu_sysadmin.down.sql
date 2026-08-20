BEGIN;

DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id AND rp.permission_id = p.id
  AND r.code = 'sysadmin' AND p.code IN ('menu:user', 'menu:userdata');

COMMIT;
