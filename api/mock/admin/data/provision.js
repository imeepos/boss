// 假数据 —— 配置下发(契约: api/openapi/admin/provision.yaml)。
'use strict';

const ok = { code: 0, message: 'success' };

const taskStats = [
  { key: 'todayCount', label: '今日下发', value: '1,286' },
  { key: 'successCount', label: '成功', value: '1,271' },
  { key: 'failCount', label: '失败待重试', value: '15' },
];

const tasks = [
  { taskNo: 'PRV-1001', device: 'EPC-0001', template: 'GPON-1000M-v3', status: '成功', retries: 0 },
  { taskNo: 'PRV-1002', device: 'OLT-02', template: 'OLT-VLAN-v2', status: '失败', retries: 3 },
];

const templates = [
  { templateNo: 'GPON-1000M-v3', name: '光猫预配置模板（1000M）', deviceType: '光猫', params: 'LOID/带宽模板', updatedAt: '2025-08-15' },
  { templateNo: 'GPON-500M-v2', name: '光猫预配置模板（500M）', deviceType: '光猫', params: 'LOID/带宽模板', updatedAt: '2025-08-15' },
  { templateNo: 'OLT-VLAN-v2', name: 'OLT 端口下发模板', deviceType: 'OLT', params: 'VLAN/QoS', updatedAt: '2025-08-12' },
];

const logs = [
  { taskNo: 'PRV-1001', device: 'EPC-0001', template: 'GPON-1000M-v3', result: '成功', retries: 0, time: '10:42' },
  { taskNo: 'PRV-1002', device: 'OLT-02', template: 'OLT-VLAN-v2', result: '失败', retries: 2, time: '09:15' },
  { taskNo: 'PRV-1003', device: 'EPC-0003', template: 'GPON-500M-v2', result: '回滚成功', retries: 1, time: '08:40' },
];

function matchKeyword(row, kw, fields) {
  if (!kw) return true;
  return fields.some((f) => String(row[f] || '').indexOf(kw) >= 0);
}

module.exports = {
  'GET /provision-tasks': { stats: taskStats, items: tasks },
  'POST /provision-tasks/{taskNo}/retry': ({ params }) =>
    Object.assign({}, ok, { taskNo: params.taskNo }),
  'GET /provision-templates': { items: templates },
  'POST /provision-templates': ok,
  'GET /provision-logs': ({ query }) => {
    const kw = query.keyword || '';
    const st = query.status || '';
    const items = logs.filter(
      (r) => matchKeyword(r, kw, ['taskNo', 'device']) && (!st || r.result === st)
    );
    return { items };
  },
};
