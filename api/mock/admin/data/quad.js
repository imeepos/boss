// 假数据 —— 四码合一(契约: api/openapi/admin/quad.yaml;状态枚举见 terms.md 第 4 节)。
'use strict';

const ok = { code: 0, message: 'success' };

// 四码合一: 资产/客户/端口/地址(fields.md 5.1:第二码为客户域,非系统账号;字段与 worker 端一致
// assetCode/customerCode/portCode/addrCode,用户码取值=客户认证账号 LOID)
const links = [
  { assetCode: 'EPC-0001', customerCode: 'LOID-88A1', portCode: 'P-SPL03-07', addrCode: 'A-3-501', status: 'LINKED', statusLabel: '一致' },
  { assetCode: 'EPC-0002', customerCode: 'LOID-88A2', portCode: 'P-SPL04-02', addrCode: 'A-5-302', status: 'CONFLICT', statusLabel: '冲突' },
  { assetCode: 'EPC-0003', customerCode: 'LOID-88A3', portCode: 'P-SPL02-03', addrCode: 'A-1-101', status: 'LINKED', statusLabel: '一致' },
];

const conflicts = [
  { conflictNo: 'CF-001', code: 'EPC-0002', conflictType: '资产码与端口码不一致', foundAt: '2025-08-17 02:00' },
  { conflictNo: 'CF-002', code: 'LOID-88A2', conflictType: '用户码关联地址漂移', foundAt: '2025-08-17 02:00' },
];

// 扫码绑定记录(订单第 9 环节,预绑定 vs 现场扫码校验)
const scanLogs = [
  { orderNo: 'ORD-20250817-001', master: '张师傅', scannedTag: 'EPC-0001', preboundTag: 'EPC-0001', result: 'MATCH', resultLabel: '一致', time: '10:40' },
  { orderNo: 'ORD-20250816-019', master: '陈师傅', scannedTag: 'EPC-0019', preboundTag: 'EPC-0018', result: 'MISMATCH', resultLabel: '拒绝(不一致)', time: '09:20' },
  { orderNo: 'ORD-20250816-020', master: '李师傅', scannedTag: 'EPC-0020', preboundTag: 'EPC-0020', result: 'MATCH', resultLabel: '一致', time: '08:55' },
];

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
