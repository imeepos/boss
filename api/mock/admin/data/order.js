// 假数据 —— 订单与工单(契约: api/openapi/admin/order.yaml;环节/状态见 docs/contract/terms.md)。
'use strict';

const ok = { code: 0, message: 'success' };

const orders = [
  { orderNo: 'ORD-20250817-001', customer: '王先生', product: '1000M', address: '望京X·3栋501', stage: 9, stageLabel: '扫码绑定(9/12)', status: 'INSTALLING', statusLabel: '装维中', ops: ['跟踪', '变更'] },
  { orderNo: 'ORD-20250817-002', customer: '赵女士', product: '500M', address: '望京X·5栋302', stage: 3, stageLabel: '端口预占(3/12)', status: 'RESERVED', statusLabel: '已预占', ops: ['跟踪', '变更', '退订'] },
  { orderNo: 'ORD-20250817-003', customer: '郑先生', product: '300M', address: '望京X·12栋906', stage: 8, stageLabel: '待领取(8/12)', status: 'INSTALLING', statusLabel: '装维中', ops: ['跟踪', '变更'] },
  { orderNo: 'ORD-20250817-004', customer: '王先生', product: '宽带变更', address: '望京X·3栋501', stage: 8, stageLabel: '变更(待领取)', status: 'INSTALLING', statusLabel: '装维中', ops: ['跟踪', '退订'] },
  { orderNo: 'ORD-20250817-008', customer: '王先生', product: '迁址移机', address: '望京X·4栋', stage: 8, stageLabel: '迁址工单', status: 'INSTALLING', statusLabel: '装维中', ops: ['跟踪'] },
  { orderNo: 'ORD-20250816-018', customer: '孙先生', product: '300M', address: '望京Y·1栋101', stage: 12, stageLabel: '已完成(12/12)', status: 'DONE', statusLabel: '已完成', ops: ['跟踪'] },
];

const STATUS_LABEL = { PENDING: '待核查', RESERVED: '已预占', INSTALLING: '装维中', DONE: '已完成' };

// 12 环节名(terms.md 第 1 节,禁止 11 环节表述)
const STAGE_NAMES = ['下单', '资源核查', '端口预占', '合同收费', '标签预绑定', '创建账号', '预下发', '派单', '扫码绑定', '激活', '激活回调', '更新GIS'];

function buildTimeline() {
  const rows = [
    ['08-17 09:02', '—', 0, 'DONE'], ['08-17 09:05', '3m', 0, 'DONE'],
    ['08-17 09:09', '4m', 1, 'DONE'], ['08-17 09:11', '2m', 0, 'DONE'],
    ['08-17 09:12', '3m', 0, 'DONE'], ['08-17 09:15', '3m', 0, 'DONE'],
    ['08-17 09:18', '3m', 0, 'DONE'], ['08-17 09:20', '2m', 0, 'DONE'],
    ['08-17 10:40', '1h20m', 2, 'DOING'], ['—', '—', '—', 'WAIT'],
    ['—', '—', '—', 'WAIT'], ['—', '—', '—', 'WAIT'],
  ];
  return rows.map((r, i) => ({ stageNo: i + 1, stageName: STAGE_NAMES[i], finishedAt: r[0], duration: r[1], retryCount: r[2], result: r[3] }));
}

function findByKeyword(items, kw) {
  if (!kw) return items;
  return items.filter((x) => ['orderNo', 'customer', 'product', 'address', 'dismantleNo', 'ticketNo', 'callbackId', 'content'].some((k) => (x[k] || '').indexOf(kw) >= 0));
}

const pool = [
  { ticketNo: 'WO-20250817-05', sourceNo: 'ORD-20250817-003', region: '吕宋大区', skill: '光纤熔接', candidates: '李师傅 / 王师傅', scope: '跨区' },
  { ticketNo: 'WO-20250817-06', sourceNo: 'TKT-20250817-005', region: '比萨扬大区', skill: '抢修', candidates: '王师傅 · 抢修组', scope: '本区' },
  { ticketNo: 'WO-20250817-07', sourceNo: 'TKT-20250817-006', region: '棉兰老大区', skill: '线路排障', candidates: '赵师傅 / 孙师傅', scope: '跨区' },
];

const myTickets = [
  { ticketNo: 'WO-20250817-01', orderNo: 'ORD-20250817-001', master: '张师傅', address: '望京X·3栋501', status: 'WORKING', statusLabel: '施工中', duration: '35min' },
  { ticketNo: 'WO-20250817-02', orderNo: 'ORD-20250817-004', master: '王师傅', address: '望京X·3栋501', status: 'PENDING', statusLabel: '待接单', duration: '—' },
  { ticketNo: 'WO-20250816-18', orderNo: 'ORD-20250816-018', master: '张师傅', address: '望京Y·1栋101', status: 'DONE', statusLabel: '已完成', duration: '28min' },
];

const transfers = [
  { orderNo: 'ORD-20250817-001', fromMaster: '张师傅', reason: '现场无法处理（需其他专业）', toTarget: '退回调度中心', status: 'WAIT_REASSIGN', statusLabel: '待重新派单' },
  { orderNo: 'ORD-20250817-003', fromMaster: '李师傅', reason: '跨片区/无资源', toTarget: '王师傅 · 抢修组', status: 'REASSIGNED', statusLabel: '已改派' },
];

const dismantles = [
  { dismantleNo: 'ORD-20250817-009', customer: '刘女士', assetCode: 'EPC-0110', portCode: 'P-SPL03-01', scanStatus: 'WAIT_SCAN', scanStatusLabel: '待扫码', stageLabel: '扫码解绑(未完成则拦截)', op: '处理' },
  { dismantleNo: 'ORD-20250816-004', customer: '吴女士', assetCode: 'EPC-0004', portCode: 'P-SPL02-02', scanStatus: 'UNBOUND', scanStatusLabel: '已解绑', stageLabel: '端口已释放', op: '详情' },
  { dismantleNo: 'ORD-20250816-002', customer: '郑先生', assetCode: 'EPC-0002', portCode: 'P-SPL01-01', scanStatus: 'DISMANTLED', scanStatusLabel: '已拆机', stageLabel: '完成', op: '详情' },
];

const complaints = [
  { ticketNo: 'TKT-20250817-012', customer: '陈先生', type: 'NETWORK_FAULT', typeLabel: '网络故障', content: '单户断网（紧急 SLA≤4h）', acceptor: '客服张', status: 'PROCESSING', statusLabel: '处理中', op: '跟进' },
  { ticketNo: 'CP-20250817-02', customer: '吴女士', type: 'TARIFF_DISPUTE', typeLabel: '资费投诉', content: '账单金额异议', acceptor: '—', status: 'PENDING', statusLabel: '待处理', op: '受理' },
  { ticketNo: 'CP-20250816-09', customer: '李女士', type: 'SERVICE_COMPLAINT', typeLabel: '服务投诉', content: '装维师傅态度', acceptor: '客服王', status: 'DONE', statusLabel: '已办结', op: '详情' },
];

const callbacks = [
  { callbackId: 'CB-8841', orderNo: 'ORD-20250817-000', source: 'aaa', result: 'FAILED', resultLabel: '失败', retryCount: 1, time: '10:32', op: '重试' },
  { callbackId: 'CB-8842', orderNo: 'ORD-20250816-018', source: 'order', result: 'FAILED', resultLabel: '失败', retryCount: 2, time: '09:12', op: '重试' },
  { callbackId: 'CB-8843', orderNo: 'ORD-20250816-021', source: 'aaa', result: 'RETRYING', resultLabel: '重试中', retryCount: 1, time: '08:50', op: '详情' },
];

module.exports = {
  'GET /orders': ({ query }) => {
    let items = findByKeyword(orders, query.keyword);
    if (query.status) items = items.filter((x) => x.status === query.status);
    return { items };
  },
  'POST /orders': ok,
  'GET /orders/{orderNo}': ({ params }) => {
    const order = orders.find((x) => x.orderNo === params.orderNo) || orders[0];
    return { order: order, timeline: buildTimeline() };
  },
  'GET /dispatch/pool': () => ({ items: pool }),
  'POST /dispatch/pool/{ticketNo}/assign': ok,
  'GET /dispatch/my-tickets': () => ({ items: myTickets }),
  'GET /dispatch/transfers': () => ({ items: transfers }),
  'POST /dispatch/tickets/{ticketNo}/transfer': ok,
  'GET /dismantles': ({ query }) => {
    let items = findByKeyword(dismantles, query.keyword);
    if (query.status) items = items.filter((x) => x.scanStatusLabel === query.status || x.scanStatus === query.status);
    return { items };
  },
  'POST /dismantles': ok,
  'GET /complaints': ({ query }) => {
    let items = complaints;
    if (query.type) items = items.filter((x) => x.type === query.type);
    if (query.status) items = items.filter((x) => x.status === query.status || x.statusLabel === query.status);
    return { items };
  },
  'POST /complaints/{ticketNo}/close': ok,
  'GET /activation-callbacks': ({ query }) => {
    let items = findByKeyword(callbacks, query.keyword);
    if (query.status) items = items.filter((x) => x.result === query.status || x.resultLabel === query.status);
    return { items };
  },
  'POST /activation-callbacks/{callbackId}/retry': ok,
};
