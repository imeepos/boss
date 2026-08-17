// 假数据 —— 四码合一(契约: api/openapi/admin/quad.yaml;状态枚举见 terms.md 第 4 节)。
// 四码关联由 db.quads() 派生(资产/客户/端口/地址,fields.md 5.1;用户码取值=客户认证账号 LOID)。
'use strict';

const db = require('../../db.js');

const ok = { code: 0, message: 'success' };

// 四码关联: db 派生(含报障工单的绑定期四码与历史冲突行)
const links = db.quads().map((q) => ({
  assetCode: q.assetCode, customerCode: q.customerCode, portCode: q.portCode, addrCode: q.addrCode,
  status: q.status, statusLabel: q.statusLabel,
}));

const conflicts = [
  { conflictNo: 'CF-001', code: 'EPC-0002', conflictType: '资产码与端口码不一致', foundAt: '2025-08-17 02:00' },
  { conflictNo: 'CF-002', code: 'LOID-88A2', conflictType: '用户码关联地址漂移', foundAt: '2025-08-17 02:00' },
];

// 扫码绑定记录(订单第 9 环节): stage≥9 的单派生 MATCH + 历史样例
const scanLogs = db.orders
  .filter((o) => o.stage >= 9 && !o.archived)
  .map((o) => ({
    orderNo: o.orderNo, master: db.byWorker(o.workerId || 1024).name,
    scannedTag: o.preBindTag, preboundTag: o.preBindTag, result: 'MATCH', resultLabel: '一致', time: o.stage === 9 ? '10:40' : '08:55',
  }))
  .concat([
    { orderNo: 'ORD-20250816-019', master: '陈师傅', scannedTag: 'EPC-0019', preboundTag: 'EPC-0018', result: 'MISMATCH', resultLabel: '拒绝(不一致)', time: '09:20' },
  ]);

module.exports = {
  'GET /quad-links': ({ query }) => {
    const code = query.code || '';
    const items = code
      ? links.filter((x) => [x.assetCode, x.customerCode, x.portCode, x.addrCode].some((c) => c.indexOf(code) >= 0))
      : links;
    return { items };
  },
  'GET /quad-conflicts': () => ({ items: conflicts }),
  'POST /quad-conflicts/{conflictNo}/resolve': ok,
  'GET /scan-logs': ({ query }) => {
    const kw = query.keyword || '';
    const items = kw
      ? scanLogs.filter((x) => [x.orderNo, x.scannedTag, x.preboundTag, x.master].some((v) => v.indexOf(kw) >= 0))
      : scanLogs;
    return { items };
  },
};
