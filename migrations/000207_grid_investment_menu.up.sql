-- W2 网格投资测算读模型菜单权限(menu:grid-investment,web/admin menu.def key 一一对应)。
-- 让号说明:计划预分配 000205 已被 feat/infra-w1-contractor-settlement(000205/000206)占用,
-- 按迁移编号规则未进库迁移直接改名让号至 000207(000207 开工前全网核查空闲)。
-- 读模型零 DDL(纯只读 SQL 聚合,fields.md 1.5.11);本迁移仅登记权限,先例 000080/000181。
-- 授予 sysadmin(全量)与 analyst(经营分析职责);ops 经 intel 组级映射可见,授权对齐三方对账门禁。
BEGIN;

INSERT INTO permissions (code, name) VALUES
    ('menu:grid-investment', '数字孪生与经营·投资测算')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:grid-investment'
WHERE r.code IN ('sysadmin', 'analyst', 'ops')
ON CONFLICT DO NOTHING;
COMMIT;
