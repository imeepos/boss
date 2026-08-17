// 假数据 —— 系统管理(契约: api/openapi/admin/sys.yaml;字段: docs/contract/fields.md §1)。
// 地址库由 db.js addresses 统一维护(uuid CRUD,regionName 为区域外键);其余取自 docs/admin/*.html。
'use strict';

const db = require('../../db.js');

var ok = { code: 0, message: 'success' };

var accounts = [
  { accountId: 1, username: 'admin', realName: '系统管理员', roleName: '系统管理员', roleTag: 'tag-blue',
    legalEntityName: '集团', deptName: '—', postName: '—', scopeName: '全集团', scopeTag: 'tag-green',
    status: 1, statusLabel: '启用', statusTag: 'tag-green' },
  { accountId: 2, username: 'agent-a01', realName: '陈客服', roleName: '业务运营/客服', roleTag: 'tag-green',
    legalEntityName: 'LEG-A', deptName: '客服部', postName: '客服坐席', scopeName: '本部门', scopeTag: 'tag-orange',
    status: 1, statusLabel: '启用', statusTag: 'tag-green' },
  { accountId: 3, username: 'agent-b01', realName: '王客服', roleName: '业务运营/客服', roleTag: 'tag-green',
    legalEntityName: 'LEG-B', deptName: '客服部', postName: '客服坐席', scopeName: '本部门', scopeTag: 'tag-orange',
    status: 1, statusLabel: '启用', statusTag: 'tag-green' },
  { accountId: 4, username: 'asset01', realName: '张资产', roleName: '资产管理员', roleTag: 'tag-gray',
    legalEntityName: 'LEG-A', deptName: '网络运维部', postName: '—', scopeName: '本公司', scopeTag: 'tag-orange',
    status: 1, statusLabel: '启用', statusTag: 'tag-green' },
  { accountId: 5, username: 'oss01', realName: '李运维', roleName: '资源运维管理员', roleTag: 'tag-orange',
    legalEntityName: 'LEG-B', deptName: '网络运维部', postName: '网络运维工程师', scopeName: '本公司资源', scopeTag: 'tag-orange',
    status: 1, statusLabel: '启用', statusTag: 'tag-green' },
  { accountId: 6, username: 'rmgr-luzon', realName: '刘经理', roleName: '经营分析', roleTag: 'tag-orange',
    legalEntityName: '—', deptName: '市场经营部', postName: '区域经理', scopeName: '吕宋大区', scopeTag: 'tag-orange',
    status: 1, statusLabel: '启用', statusTag: 'tag-green' },
];

var roles = [
  { roleId: 1, code: 'sysadmin', name: '系统管理员', tag: 'tag-blue' },
  { roleId: 2, code: 'customer', name: '客户', tag: 'tag-gray' },
  { roleId: 3, code: 'technician', name: '装维师傅', tag: 'tag-blue' },
  { roleId: 4, code: 'asset_admin', name: '资产管理员', tag: 'tag-gray' },
  { roleId: 5, code: 'resource_admin', name: '资源运维管理员', tag: 'tag-orange' },
  { roleId: 6, code: 'ops', name: '业务运营/客服', tag: 'tag-green' },
  { roleId: 7, code: 'analyst', name: '经营分析', tag: 'tag-orange' },
];

var addresses = db.addresses.map((a) => ({
  addressId: a.addressId, path: a.path, name: a.name, level: a.level, levelLabel: a.levelLabel,
  childCount: a.childCount, regionName: a.regionName,
}));

var params = [
  { key: 'arrear_stop_threshold', label: '欠费停机阈值', value: '30 天', desc: '欠费达到阈值自动停机' },
  { key: 'port_reserve_ttl', label: '端口预占有效期', value: '24 小时', desc: '超时自动释放端口' },
  { key: 'quad_recon_period', label: '四码对账周期', value: '每日', desc: '自动核对四码关系' },
  { key: 'report_period', label: '报告生成周期', value: '周报', desc: '经营分析报告自动生成' },
  { key: 'resume_sla', label: '复机恢复时限', value: '5 分钟', desc: '缴费后网络侧恢复时限' },
];

var auditLogs = [
  { logId: 1, time: '2025-08-17 10:32:11', operator: 'admin', type: '权限变更', typeTag: 'tag-blue',
    action: '修改角色权限', ip: '10.0.0.12' },
  { logId: 2, time: '2025-08-17 09:15:44', operator: 'asset01', type: '数据变更', typeTag: 'tag-green',
    action: '资产盘点导入', ip: '10.0.0.23' },
  { logId: 3, time: '2025-08-17 08:40:02', operator: 'oss01', type: '状态变更', typeTag: 'tag-orange',
    action: '端口故障上报', ip: '10.0.0.31' },
];

var importTasks = [
  { taskId: 31, taskNo: 'IMP-031', type: '地址层级', total: 12000, success: 11980, failed: 20,
    diffCount: 20, status: '待核对', statusTag: 'tag-orange', action: '核对纠错' },
  { taskId: 32, taskNo: 'IMP-032', type: '资产台账', total: 5000, success: 4985, failed: 15,
    diffCount: 15, status: '已完成', statusTag: 'tag-green', action: '详情' },
  { taskId: 30, taskNo: 'IMP-030', type: '客户档案', total: 8000, success: 8000, failed: 0,
    diffCount: 0, status: '已完成', statusTag: 'tag-green', action: '详情' },
];

module.exports = {
  'GET /accounts': { items: accounts, total: accounts.length },
  'POST /accounts': ok,
  'PUT /accounts/{accountId}': ok,
  'DELETE /accounts/{accountId}': ok,
  'GET /roles': { items: roles, total: roles.length },
  'GET /addresses': { items: addresses, total: addresses.length },
  'GET /params': { items: params, total: params.length },
  'PUT /params/{key}': ok,
  'GET /audit-logs': { items: auditLogs, total: auditLogs.length },
  'GET /import-tasks': { items: importTasks, total: importTasks.length },
  'POST /import-tasks': ok,
};
