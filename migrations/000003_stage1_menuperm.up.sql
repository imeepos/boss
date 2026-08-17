-- 阶段1扩展:菜单权限(功能权限之菜单可见性)种子。
-- 依据 docs/admin/simulation-register.md §四「角色→菜单可见范围」+ docs/admin/org-perm-simulation.md。
-- 三层权限模型:菜单权限(role→menu,本迁移) / 功能权限(role→perm code,000001 已建表) / 数据权限(account→org,000002)。
BEGIN;

-- 菜单权限码:menu:<key>,与 docs/admin/menu.js 的 key 一一对应(source-of-truth,幂等)。
INSERT INTO permissions (code, name) VALUES
    ('menu:dashboard',   '运营总览·工作台'),
    ('menu:account',     '基础配置·账号与角色'),
    ('menu:address',     '基础配置·地址层级'),
    ('menu:params',      '基础配置·业务参数'),
    ('menu:audit',       '基础配置·审计日志'),
    ('menu:importer',    '基础配置·数据导入中心'),
    ('menu:company',     '组织与权限·子公司/法人'),
    ('menu:department',  '组织与权限·部门管理'),
    ('menu:post',        '组织与权限·岗位管理'),
    ('menu:region',      '组织与权限·经营区域'),
    ('menu:datascope',   '组织与权限·数据权限'),
    ('menu:menuperm',    '组织与权限·菜单权限'),
    ('menu:customer',    '客户与资费·客户档案'),
    ('menu:product',     '客户与资费·产品资费'),
    ('menu:billing',     '计费与账务·出账管理'),
    ('menu:payment',     '计费与账务·缴费管理'),
    ('menu:arrears',     '计费与账务·欠费停复机'),
    ('menu:stopsrv',     '计费与账务·停复机执行'),
    ('menu:paycheck',    '计费与账务·渠道对账'),
    ('menu:asset',       '资产与标签·资产台账'),
    ('menu:tag',         '资产与标签·电子标签'),
    ('menu:stock',       '资产与标签·盘点管理'),
    ('menu:replace',     '资产与标签·设备更换单'),
    ('menu:resource',    '网络资源·端口台账'),
    ('menu:reserve',     '网络资源·预占与释放'),
    ('menu:transfer',    '网络资源·跨区域调配'),
    ('menu:device',      '网络资源·OLT 设备'),
    ('menu:loaccount',   '网络资源·认证账号'),
    ('menu:expand',      '网络资源·扩容申请'),
    ('menu:order',       '订单与工单·订单管理'),
    ('menu:dispatch',    '订单与工单·派单管理'),
    ('menu:dismantle',   '订单与工单·拆机管理'),
    ('menu:complaint',   '订单与工单·报障与投诉'),
    ('menu:callback',    '订单与工单·激活回调'),
    ('menu:quadlink',    '四码合一·关联查询'),
    ('menu:check',       '四码合一·对账与告警'),
    ('menu:scanlog',     '四码合一·扫码绑定记录'),
    ('menu:provision',   '配置下发·下发任务'),
    ('menu:template',    '配置下发·配置模板'),
    ('menu:provlog',     '配置下发·下发日志'),
    ('menu:alarm',       '告警中心·告警列表'),
    ('menu:aaalog',      '认证计费·话单与认证日志'),
    ('menu:gis',         '数字孪生与经营·GIS 地图'),
    ('menu:analytics',   '数字孪生与经营·经营分析'),
    ('menu:report',      '数字孪生与经营·报告中心')
ON CONFLICT (code) DO NOTHING;

-- 角色→菜单权限绑定(依据 simulation-register.md §四)。
-- sysadmin=全量;customer 走用户端,管理后台不授菜单;其余按职责范围授。
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = ANY (ARRAY[
    'menu:order','menu:dispatch',
    'menu:quadlink','menu:scanlog',
    'menu:provision','menu:provlog'
])
WHERE r.code = 'technician'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = ANY (ARRAY[
    'menu:asset','menu:tag','menu:stock','menu:replace'
])
WHERE r.code = 'asset_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = ANY (ARRAY[
    'menu:resource','menu:reserve','menu:transfer','menu:device','menu:loaccount','menu:expand',
    'menu:alarm','menu:check','menu:template','menu:aaalog'
])
WHERE r.code = 'resource_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = ANY (ARRAY[
    'menu:customer','menu:product',
    'menu:order','menu:dispatch','menu:dismantle','menu:complaint','menu:callback',
    'menu:billing','menu:payment','menu:arrears','menu:stopsrv','menu:paycheck'
])
WHERE r.code = 'ops'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = ANY (ARRAY[
    'menu:dashboard','menu:gis','menu:analytics','menu:report'
])
WHERE r.code = 'analyst'
ON CONFLICT DO NOTHING;

COMMIT;
