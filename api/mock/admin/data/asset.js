// 假数据 —— 资产与标签(契约: api/openapi/admin/asset.yaml)。
// 状态枚举见 docs/contract/terms.md 第 4 节;数据逐行搬运自 docs/admin/{asset,tag,stock,replace}.html。
'use strict';

const ok = { code: 0, message: 'success' };

function text(v) { return v === null || v === undefined ? '' : String(v); }
function hit(item, keyword) {
  if (!keyword) return true;
  const kw = String(keyword);
  return Object.keys(item).some((k) => text(item[k]).indexOf(kw) >= 0);
}
function pick(items, query) {
  let out = items;
  if (query && query.keyword) out = out.filter((it) => hit(it, query.keyword));
  if (query && query.status) {
    out = out.filter((it) => it.status === query.status || it.statusLabel === query.status);
  }
  return out;
}

// /assets 统计卡 + 清单(status: IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED)
const assets = [
  { assetCode: 'A-20250001', tagNo: 'TAG-0001', epcCode: 'EPC-0001', type: '光猫', batchNo: 'RK-202507-01', location: '望京·X小区·3栋', lifecycle: '在用', status: 'DEPLOYED', statusLabel: '正常', statusClass: 'tag-green' },
  { assetCode: 'A-20250002', tagNo: 'TAG-0002', epcCode: 'EPC-0002', type: '分光器', batchNo: 'RK-202506-03', location: '望京·X小区·弱电井2', lifecycle: '维修', status: 'MAINTENANCE', statusLabel: '维修中', statusClass: 'tag-orange' },
  { assetCode: 'A-20250003', tagNo: 'TAG-0003', epcCode: 'EPC-0003', type: '光猫', batchNo: 'RK-202508-02', location: '仓库', lifecycle: '在库', status: 'IN_STOCK', statusLabel: '待领用', statusClass: 'tag-gray' },
];

const lifecycle = [
  { assetCode: 'A-20250001', time: '2025-07-15 09:20', action: '入库', fromState: '—', toState: '在库', operator: 'asset01' },
  { assetCode: 'A-20250001', time: '2025-07-20 14:05', action: '领用', fromState: '在库', toState: '在用', operator: 'asset01' },
  { assetCode: 'A-20250002', time: '2025-08-10 11:30', action: '状态变更', fromState: '在用', toState: '故障', operator: 'oss01' },
  { assetCode: 'A-20250003', time: '2025-08-14 16:42', action: '入库', fromState: '—', toState: '在库', operator: 'asset01' },
];

// 电子标签(status: UNBOUND/BOUND/DISABLED)
const tags = [
  { tagNo: 'TAG-0001', epcCode: 'EPC-0001', band: 'UHF', boundAsset: 'A-20250001 光猫', battery: '86%', status: 'BOUND', statusLabel: '已绑定', statusClass: 'tag-green' },
  { tagNo: 'TAG-0002', epcCode: 'EPC-0002', band: 'UHF', boundAsset: 'A-20250002 分光器', battery: '72%', status: 'BOUND', statusLabel: '已绑定', statusClass: 'tag-green' },
  { tagNo: 'TAG-0100', epcCode: 'EPC-0100', band: 'HF', boundAsset: '—', battery: '100%', status: 'UNBOUND', statusLabel: '未绑定', statusClass: 'tag-gray' },
];

// 盘点任务
const stocktakes = [
  { taskId: 'PD-202508-01', scope: '望京X小区', progress: '86%', diffCount: 12, status: 'HANDLING', statusLabel: '处理中', statusClass: 'tag-orange' },
  { taskId: 'PD-202508-02', scope: '望京Y小区', progress: '100%', diffCount: 0, status: 'DONE', statusLabel: '已完成', statusClass: 'tag-green' },
];

// 设备更换单(优先级 高/中/低,状态 待派单/维修中/已更换)
const replacements = [
  { replacementNo: 'RPL-20250817-001', device: 'EPC-0002（光猫）', issue: '健康度 35 / 掉线3次', priority: '高', priorityClass: 'tag-red', status: 'PENDING_DISPATCH', statusLabel: '待派单', statusClass: 'tag-orange', action: '派单' },
  { replacementNo: 'RPL-20250814-003', device: 'SPL-03-11（分光器）', issue: '光功率异常', priority: '中', priorityClass: 'tag-orange', status: 'REPAIRING', statusLabel: '维修中', statusClass: 'tag-blue', action: '详情' },
  { replacementNo: 'RPL-20250811-002', device: 'EPC-0003（光猫）', issue: '使用年限 6 年', priority: '已完成', priorityClass: 'tag-gray', status: 'REPLACED', statusLabel: '已更换', statusClass: 'tag-green', action: '详情' },
];

module.exports = {
  'GET /assets': ({ query }) => ({
    stats: {
      total: '48,521', totalDelta: '▲ 1.2%',
      inStock: '12,430', deployed: '33,208', maintenance: '2,883',
    },
    items: pick(assets, query),
  }),
  'GET /assets/{assetCode}/lifecycle': () => ({ items: lifecycle }),
  'GET /tags': () => ({ items: tags }),
  'GET /stocktakes': () => ({ pendingDiffCount: 12, items: stocktakes }),
  'POST /stocktakes': () => ok,
  'POST /stocktakes/{taskId}/diff-handle': () => ok,
  'GET /replacements': ({ query }) => ({ items: pick(replacements, query) }),
  'POST /replacements': () => ok,
};
