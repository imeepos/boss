// 假数据 —— AAA 话单与认证日志(契约: api/openapi/admin/aaa.yaml)。
'use strict';

// type: CDR 话单 / AUTH_OK 认证成功 / AUTH_FAIL 认证失败
const logs = [
  { logId: 1, loid: 'LOID-88A1', type: 'AUTH_OK', typeLabel: '认证', result: '成功', usage: '—', billing: '—', time: '10:46' },
  { logId: 2, loid: 'LOID-88A4', type: 'AUTH_FAIL', typeLabel: '认证', result: '失败', usage: '—', billing: '—', time: '10:30' },
  { logId: 3, loid: 'LOID-88A1', type: 'CDR', typeLabel: '话单', result: '正常', usage: '3.2GB / 2h', billing: '未入账', time: '10:00' },
];

module.exports = {
  'GET /aaa-logs': ({ query }) => {
    const kw = query.keyword || '';
    const type = query.type || '';
    const items = logs.filter(
      (r) => String(r.loid).indexOf(kw) >= 0 && (!type || r.type === type)
    );
    return { items };
  },
  'POST /aaa-logs/mend': ({ body }) => {
    if (!body || !body.loid || !body.period) return { code: 1, message: 'loid/period 必填' };
    logs.push({
      logId: logs.length + 1, loid: body.loid, type: 'CDR', typeLabel: '话单',
      result: '补单', usage: '—', billing: '待入账', time: body.period,
    });
    return { code: 0, message: 'ok' };
  },
};
