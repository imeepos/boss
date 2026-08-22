-- AAA 管理总览菜单权限(增量迁移,不修改已执行的 000003)。
BEGIN;

INSERT INTO permissions (code, name)
VALUES ('menu:aaadashboard', '认证计费·AAA 运行总览')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:aaadashboard'
WHERE r.code IN ('sysadmin', 'resource_admin', 'analyst')
ON CONFLICT DO NOTHING;

COMMIT;
