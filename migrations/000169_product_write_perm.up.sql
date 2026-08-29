-- 000169: 产品资费写权限收窄(menu:product 拆读写)——何平 2026-08-28 裁定落地。
-- 裁定:资费是计费收入源头配置,属低频高危动作;ops(菜单面最宽的受理角色)
-- 职责是"卖产品"不是"定产品"。menu:product 语义降为读(存量持有者读不变),
-- 新增 menu:product-write 仅授 sysadmin;写路由(建档/编辑/状态/调价)细分门禁。
-- 实测依据:ops POST /products 原返回 200(权限内但超出职责),验收报告三轮记录。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:product-write', '客户与资费·产品资费写(建档/编辑/状态/调价)')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:product-write'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
