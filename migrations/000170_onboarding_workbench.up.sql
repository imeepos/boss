-- 000170: 开户工作台页(/bss/onboarding)权限码登记——一页聚合 代客建档→实名→
-- 下单→环节推进→派单。目标"聚合工作台完成开户全流程"(2026-08-29)。
-- 仿 000139 先例:专属 menu:<key> 服务于自定义角色的菜单可见性(门禁 E 项);
-- 端点级权限保持现状(建档/实名=menu:customer,下单/推进=menu:order,
-- 指派/激活=menu:dispatch),受理角色 ops 已持有全部相关码。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:onboarding', '客户与资费·开户工作台')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:onboarding'
WHERE r.code IN ('sysadmin', 'ops')
ON CONFLICT DO NOTHING;
COMMIT;
