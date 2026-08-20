-- 补授权:000047 种了 menu:user/menu:userdata 权限码但漏授 sysadmin,
-- 导致 sysadmin 访问用户端数据域接口 403(对齐 000039/000042 的显式授权模式)。
BEGIN;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.code = 'sysadmin' AND p.code IN ('menu:user', 'menu:userdata')
ON CONFLICT DO NOTHING;

COMMIT;
