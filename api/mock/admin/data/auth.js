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
      { key: 'todayOrders', label: '今日新增订单', value: '128' },
      { key: 'activeTickets', label: '进行中工单', value: '23' },
      { key: 'pendingAlarms', label: '待处理告警', value: '5' },
      { key: 'assetConsistency', label: '四码一致率', value: '98.6%' },
    ],
    orderStatusDist: [
      { status: 'PENDING', statusLabel: '待核查', count: 32, percent: '15%' },
      { status: 'RESERVED', statusLabel: '已预占', count: 45, percent: '22%' },
      { status: 'INSTALLING', statusLabel: '装维中', count: 28, percent: '14%' },
      { status: 'DONE', statusLabel: '已完成', count: 51, percent: '25%' },
    ],
    todos: {
      items: [
        { todoId: 1, subject: 'TK-001 待指派师傅', source: '派单池', time: '16:32' },
        { todoId: 2, subject: 'ALM-告警内容-002', source: '告警中心', time: '17:00' },
        { todoId: 3, subject: 'TK-003 待指派师傅', source: '派单池', time: '18:28' },
        { todoId: 4, subject: 'ALM-告警内容-004', source: '告警中心', time: '19:16' },
        { todoId: 5, subject: 'TK-005 待指派师傅', source: '派单池', time: '20:04' },
      ],
    },
    trend: {
      days: ['05-14', '05-15', '05-16', '05-17', '05-18', '05-19', '05-20'],
      values: [50, 65, 75, 60, 90, 70, 75],
    },
  },
};
