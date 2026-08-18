-- 国际地理基础数据维护菜单权限(menu:geo,与 web/admin menu.def key 一一对应)。
-- 授予 sysadmin(基础数据维护职责);其余角色不授。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:geo', '基础配置·国家与行政区划')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:geo'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
