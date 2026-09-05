-- 网维工程师(resource_admin)缺 menu:provision 置备权限(2026-09-05 S11/T21 发现)。
-- 现象:noc_chen(resource_admin)调 POST /provision/resources|ports 被拒
-- 403 no permission: menu:provision,无法履行端口/资源置备职责;
-- test-accounts.json 账号台账标注 noc 负责域即含 /provision/*。
-- 机理:/provision/* 置备路由(资源/端口/渠道/标签/资产)统一挂 menu:provision,
-- 000003 种子只授了 sysadmin,resource_admin 仅得 menu:resource 等查询/台账位。
-- 处置:对齐 000080(menu:odn)先例,补授 resource_admin;down 回收该对授权。
BEGIN;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:provision'
WHERE r.code = 'resource_admin'
ON CONFLICT DO NOTHING;
COMMIT;
