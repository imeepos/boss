// 假数据 —— 认证/登录 + 工作台(契约: api/openapi/admin/{auth,dashboard}.yaml)。
'use strict';

const ok = { code: 0, message: 'success' };

module.exports = {
  'POST /auth/login': () => ({
    token: 'mock-admin-jwt-' + Date.now(),
    accountId: 1,
    realName: '系统管理员',
    roleName: 'sysadmin',
  }),
  'POST /auth/logout': ok,
  'GET /auth/me': {
    accountId: 1,
    username: 'admin',
    realName: '系统管理员',
    roleName: 'sysadmin',
    legalEntityName: 'LEG-A 主品牌·企业',
    regionScope: '',
  },
  'GET /dashboard': {
    stats: [
      { key: 'todayOrders', label: '今日新增订单', value: '128', delta: '▲ 12.5% 较昨日', trend: 'up' },
      { key: 'activeTickets', label: '进行中工单', value: '342', delta: '▲ 5.2%', trend: 'up' },
      { key: 'pendingAlarms', label: '待处理告警', value: '17', delta: '▼ 3.1%', trend: 'down' },
      { key: 'assetConsistency', label: '资产账实相符率', value: '98.6%', delta: '▲ 0.4%', trend: 'up' },
    ],
    orderStatusDist: [
      { status: 'PENDING', statusLabel: '待核查', count: 52, percent: '15%' },
      { status: 'RESERVED', statusLabel: '已预占', count: 86, percent: '25%' },
      { status: 'INSTALLING', statusLabel: '装维中', count: 134, percent: '39%' },
      { status: 'DONE', statusLabel: '已完成', count: 70, percent: '20%' },
    ],
    todos: {
      items: [
        { todoId: 1, subject: 'ORD-20250817-002 端口预占即将超时(剩 2h)', source: '资源核查', time: '10:21' },
        { todoId: 2, subject: '扩容申请 EXP-2025-031 待审批', source: '网络资源', time: '09:47' },
        { todoId: 3, subject: '对账批次 RC-20250816 渠道差异 ¥120.00', source: '渠道对账', time: '09:12' },
        { todoId: 4, subject: 'OLT-望京-07 光功率越限告警', source: '告警中心', time: '08:55' },
      ],
    },
    trend: {
      days: ['08-11', '08-12', '08-13', '08-14', '08-15', '08-16', '08-17'],
      values: [96, 110, 102, 121, 134, 118, 128],
    },
  },
};
