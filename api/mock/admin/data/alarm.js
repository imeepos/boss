// 假数据 —— 告警中心(契约: api/openapi/admin/alarm.yaml)。
'use strict';

const ok = { code: 0, message: 'success' };

const alarms = [
  { alarmId: 'ALM-001', level: 'CRITICAL', levelLabel: '严重', source: 'device', content: 'OLT-15 离线', time: '08:40' },
  { alarmId: 'ALM-002', level: 'WARNING', levelLabel: '警告', source: 'device', content: 'OLT-02 光功率异常', time: '09:12' },
  { alarmId: 'ALM-003', level: 'WARNING', levelLabel: '警告', source: 'quadlink', content: '四码对账冲突 2 笔', time: '10:32' },
];

module.exports = {
  'GET /alarms': ({ query }) => {
    const lv = query.level || '';
    return { items: alarms.filter((a) => !lv || a.level === lv) };
  },
  'POST /alarms/{alarmId}/ack': ({ params }) =>
    Object.assign({}, ok, { alarmId: params.alarmId }),
  'POST /alarms/batch-retest': ({ body }) => ({
    code: 0,
    message: 'success',
    taskNo: 'RT-20250817-' + String(Date.now()).slice(-4),
    scope: body.scope || '',
  }),
};
