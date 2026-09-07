-- W4 ROW 路权与 PECE 许可工作流菜单权限(menu:permits,web/admin menu.def key 一一对应)。
-- 先例 000207(grid-investment):独立页面配独立权限码,授权对齐三方对账门禁;
-- 授予 sysadmin(全量)与 resource_admin(ODN 管理职责)。
BEGIN;

INSERT INTO permissions (code, name) VALUES
    ('menu:permits', '网络资源·许可与路权')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:permits'
WHERE r.code IN ('sysadmin', 'resource_admin')
ON CONFLICT DO NOTHING;
COMMIT;