// 假数据 —— 订单与工单(契约: api/openapi/admin/order.yaml)。
// 实体一律由 db.js 派生;环节/状态见 docs/contract/terms.md。本文件只保留调度池/回调等流程样例。
'use strict';

const db = require('../../db.js');

const ok = { code: 0, message: 'success' };
const BIZ_LABEL = { INSTALL: '', CHANGE: '宽带变更', MOVE: '迁址移机', DISMANTLE: '拆机' };

// 订单列表: db.orders 未归档非拆机单(拆机单走 /dismantles)
const orders = db.orders
  .filter((o) => !o.archived && o.bizType !== 'DISMANTLE')
  .map((o) => {
    const st = db.statusOfStage(o.stage);
    const stageLabel = o.orderNo === 'ORD-20250817-003' ? '待领取(8/12)'
      : o.orderNo === 'ORD-20250817-004' ? '变更(待领取)'
      : o.orderNo === 'ORD-20250817-008' ? '迁址工单'
      : db.STAGE_NAMES[o.stage - 1].replace('用户', '') + '(' + o.stage + '/12)';
    return {
      orderNo: o.orderNo, customer: db.byCustomer(o.customerId).name,
      product: o.bizType === 'INSTALL' ? db.byProduct(o.productId).bandwidth : BIZ_LABEL[o.bizType],
      address: o.addrLabel.replace(/ · /g, '·'), stage: o.stage, stageLabel,
      status: st, statusLabel: db.STATUS_LABEL[st],
      ops: st === 'RESERVED' ? ['跟踪', '变更', '退订'] : st === 'DONE' ? ['跟踪'] : ['跟踪', '变更'],
    };
  });

// 时间轴: db.timelineOf 映射为 admin 列(shape: stageNo/stageName/finishedAt/duration/retryCount/result)
function buildTimeline(orderNo) {
  const o = db.byOrder(orderNo) || db.orders[0];
  return db.timelineOf(o).map((s) => ({
    stageNo: s.stage, stageName: s.name, finishedAt: s.finishedAt || '—',
    duration: s.duration || '—', retryCount: s.stage === 9 ? o.scanRetries : 0,
    result: s.result,
  }));
}

function findByKeyword(items, kw) {
  if (!kw) return items;
  return items.filter((x) => ['orderNo', 'customer', 'product', 'address', 'dismantleNo', 'ticketNo', 'callbackId', 'content'].some((k) => (x[k] || '').indexOf(kw) >= 0));
}

// 调度池: 待派/可抢单样例(候选师傅为展示文案)
const pool = [
  { ticketNo: 'WO-20250817-05', sourceNo: 'ORD-20250817-003', region: '吕宋大区', skill: '光纤熔接', candidates: '李师傅 / 王师傅', scope: '跨区' },
  { ticketNo: 'WO-20250817-06', sourceNo: 'TKT-20250817-005', region: '比萨扬大区', skill: '抢修', candidates: '王师傅 · 抢修组', scope: '本区' },
  { ticketNo: 'WO-20250817-07', sourceNo: 'TKT-20250817-006', region: '棉兰老大区', skill: '线路排障', candidates: '赵师傅 / 孙师傅', scope: '跨区' },
];

// 我的工单: db.orders 中已派单的在途/完成单
const myTickets = db.orders
  .filter((o) => o.workerId && !o.archived && o.bizType !== 'DISMANTLE')
  .map((o, i) => ({
    ticketNo: 'WO-20250817-0' + (i + 1), orderNo: o.orderNo, master: db.byWorker(o.workerId).name,
    address: o.addrLabel.replace(/ · /g, '·'),
    status: o.stage >= 12 ? 'DONE' : o.stage >= 9 ? 'WORKING' : 'PENDING',
    statusLabel: o.stage >= 12 ? '已完成' : o.stage >= 9 ? '施工中' : '待接单',
    duration: o.stage >= 12 ? '28min' : o.stage >= 9 ? '35min' : '—',
  }));

const transfers = [
  { orderNo: 'ORD-20250817-001', fromMaster: '张师傅', reason: '现场无法处理（需其他专业）', toTarget: '退回调度中心', status: 'WAIT_REASSIGN', statusLabel: '待重新派单' },
  { orderNo: 'ORD-20250817-003', fromMaster: '李师傅', reason: '跨片区/无资源', toTarget: '王师傅 · 抢修组', status: 'REASSIGNED', statusLabel: '已改派' },
];

// 拆机单: db.orders 的 DISMANTLE 单 + 历史归档样例
const dismantles = db.orders
  .filter((o) => o.bizType === 'DISMANTLE' && !o.archived)
  .map((o) => ({
    dismantleNo: o.orderNo, customer: db.byCustomer(o.customerId).name, assetCode: o.preBindTag,
    portCode: o.portQuad, scanStatus: o.scanStatus,
    scanStatusLabel: o.scanStatus === 'WAIT_SCAN' ? '待扫码' : '已解绑',
    stageLabel: o.scanStatus === 'WAIT_SCAN' ? '扫码解绑(未完成则拦截)' : '端口已释放', op: '处理',
  }))
  .concat([
    { dismantleNo: 'ORD-20250816-004', customer: '吴女士', assetCode: 'EPC-0004', portCode: 'P-SPL02-02', scanStatus: 'UNBOUND', scanStatusLabel: '已解绑', stageLabel: '端口已释放', op: '详情' },
    { dismantleNo: 'ORD-20250816-002', customer: '郑先生', assetCode: 'EPC-0002', portCode: 'P-SPL01-01', scanStatus: 'DISMANTLED', scanStatusLabel: '已拆机', stageLabel: '完成', op: '详情' },
  ]);

// 投诉工单: db.repairTickets 处理中单 + 受理样例
const complaints = db.repairTickets
  .filter((t) => t.status === 'PROCESSING')
  .map((t) => ({ ticketNo: t.ticketNo, customer: db.byCustomer(t.customerId).name, type: 'NETWORK_FAULT', typeLabel: '网络故障', content: t.faultTypeLabel, acceptor: '客服张', status: 'PROCESSING', statusLabel: '处理中', op: '跟进' }))
  .concat([
    { ticketNo: 'CP-20250817-02', customer: '吴女士', type: 'TARIFF_DISPUTE', typeLabel: '资费投诉', content: '账单金额异议', acceptor: '—', status: 'PENDING', statusLabel: '待处理', op: '受理' },
    { ticketNo: 'CP-20250816-09', customer: '李女士', type: 'SERVICE_COMPLAINT', typeLabel: '服务投诉', content: '装维师傅态度', acceptor: '客服王', status: 'DONE', statusLabel: '已办结', op: '详情' },
  ]);

// 激活回调: 仅 stage≥11 的订单才应有回调记录(ORD-20250817-000 历史失败已重试成功)
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
    return { order: order, timeline: buildTimeline(params.orderNo) };
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
