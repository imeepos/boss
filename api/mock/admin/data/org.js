// 假数据 —— 组织管理(契约: api/openapi/admin/org.yaml;字段: docs/contract/fields.md §1.3/1.4)。
// 区域树由 db.js regions 统一维护(uuid CRUD);其余逐行取自 docs/admin/*.html 硬编码表格。
'use strict';

const db = require('../../db.js');

var ok = { code: 0, message: 'success' };

var legalEntities = [
  { legalEntityId: 1, code: 'LEG-A', name: '主品牌·企业', brandName: '主品牌', brandTag: 'tag-blue',
    regionName: '吕宋大区', deptCount: 3, status: 1, statusLabel: '启用', statusTag: 'tag-green' },
  { legalEntityId: 2, code: 'LEG-B', name: '家庭宽带', brandName: '家庭宽带', brandTag: 'tag-orange',
    regionName: '吕宋 + 棉兰老', deptCount: 1, status: 1, statusLabel: '启用', statusTag: 'tag-green' },
  { legalEntityId: 3, code: 'LEG-C', name: '批发品牌', brandName: '批发', brandTag: 'tag-gray',
    regionName: '比萨扬大区', deptCount: 1, status: 1, statusLabel: '启用', statusTag: 'tag-green' },
];

// 品牌×区域交叉经营(company.html 第二张卡)
var crossRegions = [
  { legalEntity: 'LEG-B 家庭宽带', region: '吕宋大区 + 棉兰老大区', scope: '两个大区的客户/订单/资产合并' },
  { legalEntity: 'LEG-A 主品牌·企业', region: '吕宋大区', scope: '单一地区' },
  { legalEntity: 'LEG-C 批发品牌', region: '比萨扬大区', scope: '单一地区' },
];

var departments = [
  { deptId: 1, name: '装维调度部', legalEntityName: 'LEG-A 主品牌·企业', funcLabel: '调度', funcTag: 'tag-blue',
    postCount: 2, queueName: '跨区工单池' },
  { deptId: 2, name: '客服部', legalEntityName: 'LEG-A 主品牌·企业', funcLabel: '客服', funcTag: 'tag-green',
    postCount: 1, queueName: '客服工单队列' },
  { deptId: 3, name: '财务部', legalEntityName: 'LEG-B 家庭宽带', funcLabel: '财务', funcTag: 'tag-orange',
    postCount: 2, queueName: '现金台队列' },
  { deptId: 4, name: '网络运维部', legalEntityName: 'LEG-C 批发品牌', funcLabel: '运维', funcTag: 'tag-gray',
    postCount: 1, queueName: '告警值班队列' },
  { deptId: 5, name: '市场经营部', legalEntityName: 'LEG-A 主品牌·企业', funcLabel: '经营', funcTag: 'tag-blue',
    postCount: 2, queueName: '经营看板分发' },
];

// 部门数据范围(department.html 第二张卡)
var deptScopes = [
  { deptName: '客服部', visibleData: '本部门工单队列', denyHint: '他部门工单 → 无权限提示' },
  { deptName: '财务部', visibleData: '姓名/账单号/余额/金额（最小化）', denyHint: '客户其他档案 → 隐藏' },
];

var posts = [
  { postId: 1, code: 'field_tech', name: '装维师傅', deptName: '装维调度部', roleCode: 'technician',
    roleTag: 'tag-blue', scopeName: '派单辖区' },
  { postId: 2, code: 'dispatcher', name: '装维调度员', deptName: '装维调度部', roleCode: 'technician',
    roleTag: 'tag-blue', scopeName: '跨区工单池' },
  { postId: 3, code: 'agent', name: '客服坐席', deptName: '客服部', roleCode: 'ops',
    roleTag: 'tag-green', scopeName: '本部门队列' },
  { postId: 4, code: 'cashier', name: '财务收款员', deptName: '财务部', roleCode: 'ops',
    roleTag: 'tag-green', scopeName: '最小化视图' },
  { postId: 5, code: 'credit', name: '信用专员', deptName: '财务部', roleCode: 'analyst',
    roleTag: 'tag-orange', scopeName: '本公司欠费' },
  { postId: 6, code: 'region_mgr', name: '区域经理', deptName: '市场经营部', roleCode: 'analyst',
    roleTag: 'tag-orange', scopeName: '本经营区域' },
  { postId: 7, code: 'key_acct', name: '大客户经理', deptName: '市场经营部', roleCode: 'ops',
    roleTag: 'tag-green', scopeName: '本公司客户' },
  { postId: 8, code: 'noc_engineer', name: '网络运维工程师', deptName: '网络运维部', roleCode: 'resource_admin',
    roleTag: 'tag-gray', scopeName: '本公司资源' },
];

// 同名岗位跨子公司复用(post.html 第二张卡)
var postReuses = [
  { post: '客服坐席 agent', roleCode: 'ops', domainA: 'LEG-A 客户/工单', domainB: 'LEG-B 客户/工单（互不可见）' },
  { post: '装维师傅 field_tech', roleCode: 'technician', domainA: 'LEG-A 派单', domainB: 'LEG-B 派单（互不可见）' },
];

var regions = db.regions.map((r) => ({
  regionId: r.regionId, path: r.path, name: r.name, level: r.level, levelLabel: r.levelLabel,
  parentName: r.parentName, childCount: r.childCount,
}));

// 区域数据范围(region.html 第二张卡)
var regionScopes = [
  { region: 'root.luzon', visibleData: '吕宋区订单/资源/GIS 图层', denyHint: '查比萨扬 → 空结果 + 审计' },
  { region: 'root.visayas', visibleData: '比萨扬区订单/资源/GIS 图层', denyHint: '查吕宋 → 空结果 + 审计' },
];

// 菜单权限矩阵(menuperm.html): 三层权限模型 + 角色×菜单组可见性
var menuPerms = {
  model: {
    layers: [
      { layer: '菜单权限', carrier: 'role → menu', semantic: '角色可见哪些菜单/入口', landing: '本页 + `role_permissions`(menu:*)' },
      { layer: '功能权限', carrier: 'role → perm code', semantic: '菜单内可执行的操作（增删改查）', landing: '`role_permissions`(perm:*) + Authz' },
      { layer: '数据权限', carrier: 'account → org', semantic: '可见哪些组织的数据范围', landing: '`datascope.html` + HasDataScope' },
    ],
  },
  matrix: {
    roleColumns: [
      { code: 'sysadmin', name: '系统管理员' },
      { code: 'technician', name: '装维师傅' },
      { code: 'asset_admin', name: '资产管理员' },
      { code: 'resource_admin', name: '资源运维管理员' },
      { code: 'ops', name: '业务运营/客服' },
      { code: 'analyst', name: '经营分析' },
      { code: 'customer', name: '客户' },
    ],
    rows: [
      { menu: '运营总览', visible: ['sysadmin', 'analyst'] },
      { menu: '基础配置', visible: ['sysadmin'] },
      { menu: '组织与权限', visible: ['sysadmin'] },
      { menu: '客户与资费', visible: ['sysadmin', 'ops'] },
      { menu: '计费与账务', visible: ['sysadmin', 'ops'] },
      { menu: '资产与标签', visible: ['sysadmin', 'asset_admin'] },
      { menu: '网络资源', visible: ['sysadmin', 'resource_admin'] },
      { menu: '订单与工单', visible: ['sysadmin', 'technician', 'ops'] },
      { menu: '四码合一', visible: ['sysadmin', 'technician', 'resource_admin'] },
      { menu: '配置下发', visible: ['sysadmin', 'technician', 'resource_admin'] },
      { menu: '告警中心', visible: ['sysadmin', 'resource_admin'] },
      { menu: '认证计费', visible: ['sysadmin', 'resource_admin'] },
      { menu: '数字孪生与经营', visible: ['sysadmin', 'analyst'] },
    ],
  },
};

var dataScopes = [
  { accountId: 1, username: 'admin', roleName: '系统管理员', roleTag: 'tag-blue',
    legalEntityName: '集团', deptName: '—', postName: '—', scopeName: '全集团', scopeTag: 'tag-green' },
  { accountId: 2, username: 'agent-a01', roleName: '业务运营/客服', roleTag: 'tag-green',
    legalEntityName: 'LEG-A', deptName: '客服部', postName: '客服坐席', scopeName: '本部门', scopeTag: 'tag-orange' },
  { accountId: 3, username: 'agent-b01', roleName: '业务运营/客服', roleTag: 'tag-green',
    legalEntityName: 'LEG-B', deptName: '客服部', postName: '客服坐席', scopeName: '本部门', scopeTag: 'tag-orange' },
  { accountId: 4, username: 'cashier-a01', roleName: '业务运营/客服', roleTag: 'tag-green',
    legalEntityName: 'LEG-A', deptName: '财务部', postName: '财务收款员', scopeName: '最小化视图', scopeTag: 'tag-orange' },
  { accountId: 5, username: 'rmgr-luzon', roleName: '经营分析', roleTag: 'tag-orange',
    legalEntityName: '—', deptName: '市场经营部', postName: '区域经理', scopeName: '吕宋大区', scopeTag: 'tag-orange' },
  { accountId: 6, username: 'noc-b01', roleName: '资源运维管理员', roleTag: 'tag-gray',
    legalEntityName: 'LEG-B', deptName: '网络运维部', postName: '网络运维工程师', scopeName: '本公司资源', scopeTag: 'tag-orange' },
];

// 授权策略说明(datascope.html 第二张卡)
var scopeStrategies = [
  { scopeType: '全集团', basis: 'region_scope 为空 + 不限子公司', example: '总部/审计/管理员' },
  { scopeType: '限经营区域子树', basis: 'region_scope 绑定 path', example: '区域经理只看吕宋' },
  { scopeType: '限子公司', basis: 'legal_entity_id', example: 'LEG-A 客服看不到 LEG-B 客户' },
  { scopeType: '限部门', basis: 'dept_id', example: '坐席只看本部门队列' },
];

module.exports = {
  'GET /legal-entities': { items: legalEntities, total: legalEntities.length, crossRegions: crossRegions },
  'POST /legal-entities': ok,
  'PUT /legal-entities/{legalEntityId}': ok,
  'GET /departments': { items: departments, total: departments.length, scopes: deptScopes },
  'GET /posts': { items: posts, total: posts.length, reuses: postReuses },
  'GET /regions': { items: regions, total: regions.length, scopes: regionScopes },
  'GET /menu-perms': menuPerms,
  'GET /data-scopes': { items: dataScopes, total: dataScopes.length, strategies: scopeStrategies },
};
