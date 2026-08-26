-- 支付配置菜单权限(menu:stripeconfig,与 web/admin menu.def key 一一对应)。
-- 授予 sysadmin;其余角色不授。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:stripeconfig', '基础配置·支付配置')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:stripeconfig'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;