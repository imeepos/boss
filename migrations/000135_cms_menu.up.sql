-- 官网内容管理菜单权限(menu:site,与 web/admin menu.def key 一一对应)。
-- 授予 sysadmin(平台内容运营职责);其余内置角色不授,自定义角色经菜单权限页勾选。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:site', '订单与工单·官网内容')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:site'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
