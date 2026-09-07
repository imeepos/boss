-- W6 工程应付台账菜单权限(menu:payables,web/admin menu.def key 一一对应)。
-- 先例 000212(permits)/000207(grid-investment):独立页面配独立权限码;
-- 工程应付属应付域(投建侧,区别于 billing 应收域),admin 呈现挂 billing 分组(domain-map §2.1);
-- 授予 sysadmin(全量)与 ops(工程履约职责),授权对齐三方对账门禁。
BEGIN;

INSERT INTO permissions (code, name) VALUES
    ('menu:payables', '计费与账务·工程应付台账')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:payables'
WHERE r.code IN ('sysadmin', 'ops')
ON CONFLICT DO NOTHING;
COMMIT;
