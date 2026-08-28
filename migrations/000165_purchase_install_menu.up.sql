-- 000165: 采购/库存/施工看板菜单权限登记(menu:purchase / menu:inventory / menu:install-board)。
-- 对应 web/admin/src/router/menu.def.ts ams 组 +2 项 + boss 组 +1 项。
-- 沿 000135/000149/000160 先例:permissions + 授 sysadmin。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:purchase',      '资产与标签·采购单'),
    ('menu:inventory',     '资产与标签·库存查询'),
    ('menu:install-board', '订单与履约·施工看板')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('menu:purchase','menu:inventory','menu:install-board')
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
