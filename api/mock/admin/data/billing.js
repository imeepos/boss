// 假数据 —— 计费账单/缴费/欠费停复机/渠道对账(契约: api/openapi/admin/billing.yaml)。
'use strict';

const ok = { code: 0, message: 'success' };

const payments = [
  { paymentId: 1, payNo: 'PAY-0001', customerName: '王先生', amount: '¥299', method: '微信', paidAt: '2025-08-16 14:20', voucher: '查看' },
  { paymentId: 2, payNo: 'PAY-0002', customerName: '李女士', amount: '¥199', method: '支付宝', paidAt: '2025-08-16 10:05', voucher: '查看' },
];

const stopResumeTasks = [
  { taskId: 'T-0001', customerName: '吴女士', loid: 'LOID-88A5', action: '停机', reason: '欠费达阈值', netResult: '生效', executedAt: '08:00' },
  { taskId: 'T-0002', customerName: '孙先生', loid: 'LOID-88A3', action: '复机', reason: '缴费入账', netResult: '生效', executedAt: '09:30' },
  { taskId: 'T-0003', customerName: '李女士', loid: 'LOID-88A4', action: '停机', reason: '欠费达阈值', netResult: '失败', executedAt: '10:10' },
];

const reconciliations = [
  { batchNo: 'PC-20250816-05', channel: '微信', channelAmount: '¥1,284,320', systemAmount: '¥1,284,320', diff: '¥0', status: '已平账' },
  { batchNo: 'PC-20250816-04', channel: '支付宝', channelAmount: '¥862,005', systemAmount: '¥861,905', diff: '¥-100', status: '差异挂起' },
  { batchNo: 'PC-20250815-09', channel: '线下营业厅', channelAmount: '¥215,600', systemAmount: '¥215,600', diff: '¥0', status: '已平账' },
];

function byKeyword(rows, kw, fields) {
  if (!kw) return rows;
  return rows.filter((r) => fields.some((f) => String(r[f] || '').indexOf(kw) >= 0));
}

module.exports = {
  'GET /bills': () => ({
    stats: { receivable: '¥1,286,400', received: '¥1,103,215', arrearsCount: '1,024' },
    items: [
      { billId: 1, billNo: 'BILL-202508-0001', customerName: '王先生', period: '2025-08', amount: '¥299', status: '已缴' },
      { billId: 2, billNo: 'BILL-202508-0002', customerName: '吴女士', period: '2025-08', amount: '¥199', status: '欠费' },
      { billId: 3, billNo: 'BILL-202507-0341', customerName: '孙先生', period: '2025-07', amount: '¥199', status: '已缴清' },
    ],
  }),

  'GET /payments': ({ query }) => ({ items: byKeyword(payments, query.keyword, ['payNo', 'customerName']) }),

  'GET /arrears': () => ({
    stats: { arrearsCount: '1,024', stoppedCount: '386', pendingCount: '128' },
    items: [
      { customerId: 2, customerName: '吴女士', arrearsAmount: '¥199', arrearsDays: '12 天', status: '已停机', netStatus: '已停服' },
      { customerId: 3, customerName: '孙先生', arrearsAmount: '¥398', arrearsDays: '已缴清', status: '已复机', netStatus: '在线' },
      { customerId: 4, customerName: '周女士', arrearsAmount: '¥199', arrearsDays: '3 天', status: '正常', netStatus: '在线' },
    ],
  }),

  'POST /arrears/{customerId}/stop': ok,
  'POST /arrears/{customerId}/resume': ok,

  'GET /stop-resume-tasks': ({ query }) => {
    const ST = {
      '停机成功': (r) => r.action === '停机' && r.netResult === '生效',
      '复机成功': (r) => r.action === '复机' && r.netResult === '生效',
      '执行失败': (r) => r.netResult === '失败',
    };
    let rows = byKeyword(stopResumeTasks, query.keyword, ['customerName', 'loid']);
    const pred = ST[query.status];
    if (pred) rows = rows.filter(pred);
    return { items: rows };
  },

  'POST /stop-resume-tasks/{taskId}/retry': ok,

  'GET /reconciliations': ({ query }) => {
    let rows = byKeyword(reconciliations, query.keyword, ['batchNo', 'channel']);
    if (query.status && query.status !== '全部状态') rows = rows.filter((r) => r.status === query.status);
    return { items: rows };
  },

  'POST /reconciliations/{batchNo}/settle': ok,
};
