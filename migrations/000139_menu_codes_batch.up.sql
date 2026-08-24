-- 消化 E 项 baseline 豁免:12 个复用他人权限码的存量页面补专属 menu:<key> 权限码
-- (仿 000135 cms_menu 先例)。统一授 sysadmin;后端端点权限码保持现状,
-- 专属码先服务于自定义角色的菜单可见性(menu:<key> 逐项推导),端点级
-- 权限收敛到各码随后续域改造分批进行(避免一次迁移改变多个域的鉴权行为)。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:collection-tasks', '计费账务·催收任务队列'),
    ('menu:feedback',        '订单与工单·回访评价'),
    ('menu:knowledge',       '订单与工单·知识库'),
    ('menu:marketing',       '营销促销·规则管理'),
    ('menu:marketing-recon', '营销促销·券积分对账'),
    ('menu:message',         '订单与工单·消息中心'),
    ('menu:servers',         '基础配置·服务端配置'),
    ('menu:service-metrics', '订单与工单·客户服务指标'),
    ('menu:storageconfig',   '基础配置·对象存储配置'),
    ('menu:worker',          '订单与工单·师傅管理'),
    ('menu:worker-ops',      '订单与工单·师傅端内容'),
    ('menu:worker-reg',      '订单与工单·师傅注册审核')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'sysadmin'
  AND p.code IN ('menu:collection-tasks','menu:feedback','menu:knowledge','menu:marketing',
                 'menu:marketing-recon','menu:message','menu:servers','menu:service-metrics',
                 'menu:storageconfig','menu:worker','menu:worker-ops','menu:worker-reg')
ON CONFLICT DO NOTHING;
COMMIT;
