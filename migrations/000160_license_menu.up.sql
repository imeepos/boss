-- 系统授权菜单权限(menu:license,与 web/admin menu.def key 一一对应)。
-- 授予 sysadmin(平台授权管理);其余角色不授,自定义角色经菜单权限页勾选。
-- 沿 000135 cms_menu / 000149 realname_review_menu 先例。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:license', '系统管理·系统授权')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:license'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;