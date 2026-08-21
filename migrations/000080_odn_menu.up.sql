-- ODN 无源物理层维护菜单权限(menu:odn,与 web/admin menu.def key 一一对应)。
-- 授予 sysadmin 与 resource_admin(网络资源维护职责)。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:odn', '资源管理·ODN 无源网络')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:odn'
WHERE r.code IN ('sysadmin', 'resource_admin')
ON CONFLICT DO NOTHING;
COMMIT;
