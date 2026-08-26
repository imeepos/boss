-- 实名审核中心菜单权限(menu:realname-review,与 web/admin menu.def key 一一对应)。
-- 聚合客户/师傅 verifications 待核验/历史条目(后端 GET /verifications)。
-- 授予 sysadmin;其余角色不授(后续按需扩展 ops/dispatch 子角色)。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:realname-review', '基础配置·实名审核中心')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
  FROM roles r
  JOIN permissions p ON p.code = 'menu:realname-review'
 WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
