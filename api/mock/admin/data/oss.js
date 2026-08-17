// 假数据 —— 网络资源 Oss(契约: api/openapi/admin/oss.yaml)。
// 端口状态枚举 IDLE/RESERVED/USED/DISABLED(docs/contract/terms.md 第 4 节);
// 数据逐行搬运自 docs/admin/{resource,reserve,transfer,device,loaccount,expand}.html。
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
  if (query && query.type) {
    out = out.filter((it) => it.resourceCode === query.type || it.resourceLabel === query.type);
  }
  return out;
}

// /ports 统计卡 + 清单(status: IDLE/RESERVED/USED/DISABLED)
const ports = [
  { portCode: 'P-001-01', quadCode: 'P-SPL03-01', parentName: 'OLT-01 · SPL-03', address: '望京·X小区·3栋', status: 'IDLE', statusLabel: '空闲', statusClass: 'tag-green', orderId: '' },
  { portCode: 'P-001-02', quadCode: 'P-SPL03-07', parentName: 'OLT-01 · SPL-03', address: '望京·X小区·3栋', status: 'RESERVED', statusLabel: '预占', statusClass: 'tag-orange', orderId: 'ORD-20250817-001' },
  { portCode: 'P-001-03', quadCode: 'P-SPL03-03', parentName: 'OLT-01 · SPL-03', address: '望京·X小区·3栋', status: 'USED', statusLabel: '在用', statusClass: 'tag-blue', orderId: 'ORD-20250816-018' },
];

const portChanges = [
  { portCode: 'P-001-02', time: '2025-08-17 09:09', fromStatus: '空闲', toStatus: '预占', source: '订单端口预占', operator: 'oss01' },
  { portCode: 'P-001-03', time: '2025-08-10 15:22', fromStatus: '预占', toStatus: '在用', source: '扫码激活', operator: 'tech07' },
  { portCode: 'P-001-01', time: '2025-08-09 10:05', fromStatus: '禁用', toStatus: '空闲', source: '修复恢复启用', operator: 'oss01' },
];

// 端口预占记录(状态 预占中/已超时/已释放)
const reserves = [
  { reserveId: 'RSV-0001', portCode: 'P-SPL03-07', splitterName: 'SPL-03', orderId: 'ORD-20250817-001', reservedAt: '10:20', expireAt: '12:20', status: 'RESERVED', statusLabel: '预占中', statusClass: 'tag-blue', action: '释放' },
  { reserveId: 'RSV-0002', portCode: 'P-SPL01-09', splitterName: 'SPL-01', orderId: 'ORD-20250817-002', reservedAt: '09:05', expireAt: '11:05', status: 'TIMEOUT', statusLabel: '已超时', statusClass: 'tag-red', action: '释放' },
  { reserveId: 'RSV-0003', portCode: 'P-SPL02-03', splitterName: 'SPL-02', orderId: 'ORD-20250816-018', reservedAt: '昨日', expireAt: '—', status: 'RELEASED', statusLabel: '已释放', statusClass: 'tag-gray', action: '详情' },
];

// 跨区域调配单(状态 待审批/已批准/已驳回/已完成;类型 PORT/ASSET/DEVICE)
const transfers = [
  { transferNo: 'TR-20250817-01', resourceCode: 'PORT', resourceLabel: '端口', fromRegion: '比萨扬大区（空闲）', toRegion: '吕宋大区（紧缺）', quantity: 120, status: 'PENDING', statusLabel: '待审批', statusClass: 'tag-orange', approver: '—', actions: ['审批', '驳回'] },
  { transferNo: 'TR-20250816-04', resourceCode: 'DEVICE', resourceLabel: 'OLT 设备', fromRegion: '棉兰老大区', toRegion: '吕宋大区', quantity: 3, status: 'APPROVED', statusLabel: '已批准', statusClass: 'tag-blue', approver: 'admin', actions: ['执行'] },
  { transferNo: 'TR-20250815-02', resourceCode: 'ASSET', resourceLabel: '分光器', fromRegion: '吕宋大区', toRegion: '比萨扬大区', quantity: 15, status: 'DONE', statusLabel: '已完成', statusClass: 'tag-green', approver: 'admin', actions: ['查看'] },
];

// 归属变更台账(总部汇总 = 各区域之和)
const ledger = [
  { region: '吕宋大区', transferOut: 0, transferIn: 120, available: '1,240', consistent: true, consistentLabel: '一致', statusClass: 'tag-green' },
  { region: '比萨扬大区', transferOut: 120, transferIn: 0, available: '2,180', consistent: true, consistentLabel: '一致', statusClass: 'tag-green' },
  { region: '棉兰老大区', transferOut: 3, transferIn: 0, available: '980', consistent: true, consistentLabel: '一致', statusClass: 'tag-green' },
];

// /olt-devices 统计卡 + 设备状态
const oltDevices = [
  { deviceName: 'OLT-01', address: '望京机房', status: 'ONLINE', statusLabel: '在线', statusClass: 'tag-green', opticalPower: '-18.2', packetLoss: '0.01%', alarmLabel: '无', alarmClass: '' },
  { deviceName: 'OLT-02', address: '望京机房', status: 'ONLINE', statusLabel: '在线', statusClass: 'tag-green', opticalPower: '-18.6', packetLoss: '0.03%', alarmLabel: '无', alarmClass: '' },
  { deviceName: 'OLT-15', address: '朝阳机房', status: 'OFFLINE', statusLabel: '离线', statusClass: 'tag-red', opticalPower: '—', packetLoss: '—', alarmLabel: '3 条', alarmClass: 'tag-red' },
];

// 认证账号 LOID(认证状态 在服/停机/待激活)
const loAccounts = [
  { loid: 'LOID-88A1', customerName: '王先生', productBandwidth: '1000M', qosTemplate: 'QoS-VIP', status: 'ACTIVE', statusLabel: '在服', statusClass: 'tag-green', lastAuth: '2 分钟前', action: '详情' },
  { loid: 'LOID-88A5', customerName: '吴女士', productBandwidth: '1000M', qosTemplate: 'QoS-STD', status: 'SUSPENDED', statusLabel: '停机', statusClass: 'tag-orange', lastAuth: '昨日', action: '复机' },
  { loid: 'LOID-88A3', customerName: '孙先生', productBandwidth: '300M', qosTemplate: 'QoS-STD', status: 'ACTIVE', statusLabel: '在服', statusClass: 'tag-green', lastAuth: '—', action: '详情' },
];

// 扩容申请单(状态 待审批/实施中/已完成)
const expansions = [
  { expansionNo: 'EXP-20250817-001', region: '望京X·3栋', expectedPorts: 128, reason: '端口利用率 92%', status: 'PENDING', statusLabel: '待审批', statusClass: 'tag-orange', action: '审批' },
  { expansionNo: 'EXP-20250814-002', region: '望京Y小区', expectedPorts: 512, reason: '高潜扩容热区', status: 'IN_PROGRESS', statusLabel: '实施中', statusClass: 'tag-blue', action: '详情' },
  { expansionNo: 'EXP-20250810-003', region: '望京Z', expectedPorts: 256, reason: '投资回报率高', status: 'DONE', statusLabel: '已完成', statusClass: 'tag-green', action: '详情' },
];

module.exports = {
  'GET /ports': () => ({
    stats: { total: '96,340', idle: '18,720', reserved: '2,156', disabled: '1,043' },
    items: ports,
  }),
  'GET /ports/{portCode}/change-history': () => ({ items: portChanges }),
  'GET /reserves': ({ query }) => ({ items: pick(reserves, query) }),
  'POST /reserves/{reserveId}/release': () => ok,
  'GET /transfers': ({ query }) => ({
    items: pick(transfers, query),
    ledger: { items: ledger },
  }),
  'POST /transfers': () => ok,
  'POST /transfers/{transferNo}/approve': () => ok,
  'POST /transfers/{transferNo}/reject': () => ok,
  'GET /olt-devices': () => ({
    stats: { total: '86', online: '82', offline: '4', activeAlarms: '17' },
    items: oltDevices,
  }),
  'GET /lo-accounts': ({ query }) => ({ items: pick(loAccounts, query) }),
  'GET /expansions': ({ query }) => ({ items: pick(expansions, query) }),
  'POST /expansions': () => ok,
};
