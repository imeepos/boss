-- 000166: API 在线文档菜单权限登记(menu:apidocs)。
-- 对应 web/admin/src/router/menu.def.ts channel(集成与开放)组 +1 项(/base/apidocs);
-- 后端 GET /api/admin/v1/docs/openapi 同权限码门禁。
-- 沿 000135/000165 先例:permissions + 授 sysadmin。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:apidocs', '集成与开放·API 文档')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:apidocs'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
