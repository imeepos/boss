// mock data —— 师傅端视图。全部数据由 db.js 事实库派生(张师傅 workerId=1024):
// 工单/四码/激活/收款/绩效/消息/公告/FAQ/物料/健康/抢单池/排期无一本地手抄,
// 对应实体均可在 admin 端(worker.js 路由 + crud)管理。字段对齐 api/openapi/worker/schemas.yaml。
'use strict';

const db = require('../db.js');
const ME = 1024; // 当前登录师傅: 张师傅
const BIZ_LABEL = { INSTALL: '安装', CHANGE: '宽带变更', MOVE: '迁址移机', DISMANTLE: '拆机' };

// —— 工单: 派生自 db.orders(stage≥8 且已派给张师傅=doing;未指派=todo) + db.repairTickets ——
function installTicket(o) {
  const p = db.byProduct(o.productId) || { name: BIZ_LABEL[o.bizType] };
  const isScanStage = o.stage === 9;
  return {
    ticketNo: o.orderNo, type: o.bizType,
    typeLabel: o.bizType === 'INSTALL' ? (p.bandwidth || '') + ' 安装' : BIZ_LABEL[o.bizType],
    address: o.addrLabel.replace(/ · /g, '·'), distanceKm: 2.1,
    scheduleSlot: o.scheduleSlot || '', stage: o.stage, stageTotal: 12,
    status: o.bizType === 'DISMANTLE' ? 'DOING' : isScanStage ? 'SCAN_PENDING' : 'ACCEPTED',
    statusLabel: o.bizType === 'DISMANTLE' ? '拆机' : isScanStage ? '待扫码绑定' : '已接单',
    slaLeftMinutes: null,
    note: o.workerId ? null : o.bizType === 'MOVE' ? '迁址工单 · 新址资源已核查' : '空闲端口已预占 · 标签已预绑定',
  };
}
function repairTicket(t) {
  return { ticketNo: t.ticketNo, type: 'REPAIR', typeLabel: '断网抢修', address: t.addrLabel.replace(/ · /g, '·'), distanceKm: 1.8, scheduleSlot: '', stage: t.stage, stageTotal: 6, status: 'DOING', statusLabel: '修复中', slaLeftMinutes: t.slaLeftMinutes };
}

const activeInstalls = db.orders.filter((o) => o.stage >= 8 && o.stage < 12 && o.workerId === ME);
const todoInstalls = db.orders.filter((o) => o.stage >= 8 && o.stage < 12 && !o.workerId && !o.archived);
const doneInstalls = db.orders.filter((o) => o.stage === 12 && o.workerId === ME && !o.archived);
const doingRepairs = db.repairTickets.filter((t) => t.workerId === ME && t.status === 'PROCESSING');

const tickets = {
  doing: activeInstalls.map(installTicket).concat(doingRepairs.map(repairTicket)),
  todo: todoInstalls.map(installTicket),
  done: doneInstalls.map((o) => ({ ticketNo: o.orderNo, type: o.bizType, typeLabel: (db.byProduct(o.productId) || {}).bandwidth + ' 安装', address: o.addrLabel.replace(/ · /g, '·'), stage: 12, stageTotal: 12, status: 'DONE', statusLabel: '已激活', finishedAt: o.finishedAt })),
};

const history = db.orders.filter((o) => o.archived && o.workerId === ME).map((o) => ({
  ticketNo: o.orderNo, type: o.bizType, typeLabel: o.bizType === 'DISMANTLE' ? '拆机' : (db.byProduct(o.productId) || {}).bandwidth + ' 安装',
  address: o.addrLabel.replace(/ · /g, '·'), status: 'DONE', statusLabel: o.bizType === 'DISMANTLE' ? '已拆机' : '已激活', finishedAt: o.finishedAt,
}));

// —— 抢单池(hall): 见下方 liveOf(view.hall),未指派抢修单 + 应急/预告附加任务 ——

// —— 四码: 由 db.quads() 按单号反查 ——
function quadOf(no) {
  const q = db.quads().find((x) => x.orderNo === no);
  if (!q) return null;
  return { assetCode: q.assetCode, customerCode: q.customerCode, portCode: q.portCode, addrCode: q.addrCode, status: q.status, matched: q.status === 'LINKED' };
}
const quad = quadOf('ORD-20250817-001');

// —— 工单详情 ——
function detailOf(no) {
  const o = db.byOrder(no);
  if (o) {
    const c = db.byCustomer(o.customerId);
    const p = db.byProduct(o.productId) || {};
    const stages = o.bizType === 'DISMANTLE' ? [] : db.timelineOf(o);
    return {
      ticketNo: o.orderNo, bizNo: o.orderNo, status: o.stage === 9 ? 'SCAN_PENDING' : 'DOING',
      statusLabel: o.stage === 9 ? '待扫码绑定' : BIZ_LABEL[o.bizType],
      product: o.bizType === 'INSTALL' ? p.name + '（有线）· ¥' + p.monthlyFee + '/月' : '',
      customerName: c.name, customerPhoneMasked: c.phoneMasked, address: o.addrLabel,
      splitterPort: o.splitterPort || '', preBindTag: o.preBindTag, scheduleSlot: o.scheduleSlot || '',
      stages: stages, quad: quadOf(no), riskCheck: { blacklistHit: false, graylistHit: false },
    };
  }
  const t = db.repairTickets.find((x) => x.ticketNo === no);
  const c = db.byCustomer(t.customerId);
  const REPAIR_NOTES = ['09:40 · 自动关联资产/端口', '09:42 · 远程诊断入工单', '09:45 · 张师傅已接单 · SLA 计时', '进行中 · 到场处理并上报结果', '系统自动验证网络恢复', '满意度调查+回填'];
  return {
    ticketNo: t.ticketNo, bizNo: t.ticketNo, status: 'DOING', statusLabel: '紧急 · 断网',
    product: '', customerName: c.name, customerPhoneMasked: c.phoneMasked, address: t.addrLabel,
    splitterPort: t.splitterPort, preBindTag: t.preBindTag,
    faultTypeLabel: t.faultTypeLabel, reportedAt: t.reportedAt.slice(5) + '（已受理）', slaLeftMinutes: t.slaLeftMinutes,
    remoteDiagnosis: t.diagnosis,
    stages: ['报障', '诊断', '派单', '修复', '复核', '回访'].map((name, i) => {
      const n = i + 1;
      return { stage: n, name, result: n < t.stage ? 'DONE' : n === t.stage ? 'DOING' : 'PENDING', finishedAt: '', durationMinutes: null, note: REPAIR_NOTES[i] };
    }),
    quad: quadOf(t.ticketNo), riskCheck: { blacklistHit: false, graylistHit: false },
  };
}

// —— 师傅档案/绩效/设置/消息等: 全部取自 db 师傅域实体表(见下方 view getters) ——
const me = db.byWorker(ME);

const home = {
  workerName: me.name, groupName: me.groupName, phoneMasked: me.phoneMasked,
  today: { accepted: tickets.doing.length, finished: tickets.done.length + 0, doing: tickets.doing.length, todo: tickets.todo.length },
  ongoing: tickets.doing,
};

// —— 实体派生视图用 getter 惰性求值: admin 端改 db 实体后师傅端即时可见 ——
function liveOf(fn) {
  return { get: fn, enumerable: true, configurable: true };
}

const view = {
  // 师傅域实体派生(可被 admin 端 /workers、/notices、/faqs、/worker-messages 等管理)
  hall: liveOf(() => ({ items: db.repairTickets
    .filter((t) => !t.workerId && t.status === 'PROCESSING')
    .map((t) => ({ ticketNo: t.ticketNo, type: 'REPAIR', typeLabel: '抢修', address: t.addrLabel, distanceKm: 1.5, status: 'TODO', statusLabel: '可抢' }))
    .concat(db.hallExtras.map((x) => ({ ticketNo: x.ticketNo, type: x.type, typeLabel: x.typeLabel, address: x.address, distanceKm: x.distanceKm, status: 'TODO', statusLabel: '可抢' }))) })),
  materials: liveOf(() => ({ items: db.workerMaterials.filter((m) => m.workerId === ME),
    tools: db.workerTools.filter((t) => t.workerId === ME),
    pendingReturn: db.assetReturns.filter((r) => r.workerId === ME && r.status === 'PENDING').map((r) => ({ epc: r.epc, reason: r.reason })),
    returned: db.returnStats })),
  maintenance: liveOf(() => ({ items: db.deviceMaintenances })),
  profile: liveOf(() => ({ workerId: ME, name: me.name, groupName: me.groupName, staffNo: me.staffNo, phoneMasked: me.phoneMasked, online: db.byWorkerProfile(ME).online, serveYears: db.byWorkerProfile(ME).serveYears, month: db.byWorkerProfile(ME).month })),
  performance: liveOf(() => {
    const p = db.byWorkerProfile(ME);
    const cs = db.workerCommissions.filter((c) => c.workerId === ME);
    return { period: p.month.period, summary: p.month, commissions: cs.map((c) => ({ name: c.name, formula: c.formula, amount: c.amount })),
      totalAmount: cs.reduce((s, c) => s + c.amount, 0),
      ranking: db.workerProfiles.slice().sort((a, b) => b.month.finished - a.month.finished)
        .map((r, i) => ({ name: db.byWorker(r.workerId).name + (r.workerId === ME ? '（本人）' : ''), rank: i + 1, self: r.workerId === ME })) };
  }),
  schedule: liveOf(() => {
    const s = db.workerSchedules.find((x) => x.workerId === ME) || { month: '', busyDays: [] };
    return { month: s.month, busyDays: s.busyDays,
      today: tickets.doing.concat(tickets.todo).filter((t) => t.scheduleSlot).map((t) => ({ time: (t.scheduleSlot.match(/\d{2}:\d{2}/) || [''])[0], ticketNo: t.ticketNo, address: t.address.replace(/^望京X · /, '') })) };
  }),
  settings: liveOf(() => ({ online: db.byWorkerProfile(ME).online, radiusKm: db.byWorkerProfile(ME).radiusKm, acceptTypes: db.byWorkerProfile(ME).acceptTypes })),
  feedbacks: liveOf(() => { const p = db.byWorkerProfile(ME);
    return { latest: (() => { const f = db.workerFeedbacks[0]; return { ticketNo: f.ticketNo, score: f.score }; })(),
      monthAvgScore: p.month.score, replyRate: 62,
      items: db.workerFeedbacks.filter((f) => f.workerId === ME).map((f) => ({ customerName: f.customerName, score: f.score, comment: f.comment, needReview: f.needReview })) }; }),
  messages: liveOf(() => ({ items: db.workerMessages.filter((m) => m.workerId === ME).map((m) => ({ level: m.level, title: m.title, content: m.content, sentAt: m.sentAt, read: m.read })) })),
  notices: liveOf(() => ({ items: db.workerNotices.filter((n) => n.active).map((n) => ({ noticeId: n.noticeId, title: n.title, category: n.category, publishedAt: n.publishedAt })) })),
  faq: liveOf(() => ({ items: db.workerFaqs.filter((f) => f.active).map((f) => ({ faqId: f.faqId, title: f.title, summary: f.summary })) })),
  serviceMessages: liveOf(() => ({ items: db.serviceMessages.filter((m) => m.workerId === ME).map((m) => ({ from: m.from, content: m.content })) })),
};

module.exports = Object.defineProperties({
  quad: quad,
  tickets: tickets,
  history: { items: history, totalCount: db.byWorkerProfile(ME).month.finished },
  home: home,
  installDetail: detailOf('ORD-20250817-001'),
  repairDetail: detailOf('TKT-20250817-012'),
  dismantleDetail: detailOf('ORD-20250817-009'),
  navi: { address: '望京X · 3栋 · 501', distanceKm: 2.1, etaMinutes: 8, entrance: '东门进 · 3栋1单元' },
  scanOk: { matched: true, result: 'MATCH', message: '四码校验一致，绑定已通过，请确认装维完成情况。', quad: quad },
  scanBad: { matched: false, result: 'MISMATCH', message: '扫码标签与此单预绑定标签不符，绑定已被系统拒绝。请核实换机或重绑。', quad: Object.assign({}, quad, { status: 'CONFLICT', matched: false }) },
  photos: { items: [{ photoId: 'p1', fileName: 'photo_0809.jpg', linked: true }] },
  report: { ticketNo: 'ORD-20250817-001', quad: quad, checks: { powerOn: true, opticalPowerDbm: -16.2, provisionDone: true, loidAuthPassed: true }, provisionLog: { template: 'GPON-1000M-v3', preResult: '成功 · 08-17 09:18', onsiteResult: '成功 · 自动' } },
  activation: { ticketNo: 'ORD-20250817-001', loid: 'LOID-88A1', status: 'PENDING', statusLabel: '未生效', lastTry: '待扫码绑定(环节9)完成后触发' },
  activationDone: { ticketNo: 'ORD-20250817-001', loid: 'LOID-88A1', status: 'SUCCESS', statusLabel: '已生效', lastTry: '08-17 11:02 · 激活成功' },
  charge: { ticketNo: 'ORD-20250817-001', amountDue: db.byOrder('ORD-20250817-001').chargeAmount, amountDesc: '首月', payMethods: ['扫码支付', '现金', 'POS'] },
  replace: { ticketNo: 'ORD-20250817-001', oldEpc: 'EPC-0002', oldEpcStatus: '故障', replaceType: '故障调换 · 旧件返修', steps: [
    { name: '扫旧件', status: 'DONE', note: 'EPC-0002 已识别' }, { name: '扫新件', status: 'TODO', note: '待扫码' }, { name: '登记', status: 'TODO', note: '未完成' },
  ] },
  measure: { opticalPowerDbm: -16.2, opticalPowerLabel: '正常', downloadMbps: 942, uploadMbps: 96, packetLossRate: 0 },
  // 空闲口: 从 db.ports 动态排除 SPL-03 下已占用(RESERVED/USED)的 PON 口
  resources: { idlePorts: 3, nearestSplitter: 'SPL-03-07', idlePonPorts: ['P7', 'P9', 'P11', 'P13', 'P15'].filter((pon) => !db.ports.some((p) => p.quadCode === 'P-SPL03-0' + pon.slice(1))) },
}, {
  hall: view.hall,
  materials: view.materials,
  maintenance: view.maintenance,
  profile: view.profile,
  performance: view.performance,
  schedule: view.schedule,
  settings: view.settings,
  feedbacks: view.feedbacks,
  messages: view.messages,
  notices: view.notices,
  faq: view.faq,
  serviceMessages: view.serviceMessages,
});
