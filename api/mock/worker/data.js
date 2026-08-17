// mock data —— 师傅端视图。工单/四码/激活/收款一律由 db.js 派生(张师傅 workerId=1024),
// 仅本端专属展示(绩效/消息/公告/FAQ)保留本地。字段对齐 api/openapi/worker/schemas.yaml。
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

const hall = { items: [
  { ticketNo: 'EMG-20250817-001', type: 'EMERGENCY', typeLabel: '台风批量复测', address: '望京X片区 · 受影响 23 户', distanceKm: 0.8, status: 'TODO', statusLabel: '可抢' },
  { ticketNo: 'TKT-20250817-006', type: 'REPAIR', typeLabel: '抢修', address: '望京X · 8栋 · 断网', distanceKm: 1.2, status: 'TODO', statusLabel: '可抢' },
  { ticketNo: 'ORD-20250817-007', type: 'INSTALL', typeLabel: '新装', address: '望京X · 15栋', distanceKm: 2.8, status: 'TODO', statusLabel: '可抢' },
] };

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
    stages: db.STAGE_NAMES.slice(0, 0).concat(['报障', '诊断', '派单', '修复', '复核', '回访']).map((name, i) => {
      const n = i + 1;
      return { stage: n, name, result: n < t.stage ? 'DONE' : n === t.stage ? 'DOING' : 'PENDING', finishedAt: '', durationMinutes: null, note: REPAIR_NOTES[i] };
    }),
    quad: quadOf(t.ticketNo), riskCheck: { blacklistHit: false, graylistHit: false },
  };
}

const home = {
  workerName: db.byWorker(ME).name, groupName: db.byWorker(ME).groupName, phoneMasked: db.byWorker(ME).phoneMasked,
  today: { accepted: tickets.doing.length, finished: tickets.done.length + 0, doing: tickets.doing.length, todo: tickets.todo.length },
  ongoing: tickets.doing,
};

module.exports = {
  quad: quad,
  tickets: tickets,
  history: { items: history, totalCount: 46 },
  hall: hall,
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
  materials: { items: [
    { itemId: 'm1', name: '光猫', qty: 2, spec: 'GPON 千兆 · 含标签', outBound: false },
    { itemId: 'm2', name: '机顶盒', qty: 1, spec: 'IPTV 4K', outBound: false },
    { itemId: 'm3', name: '光纤跳线', qty: 5, spec: 'SC/APC 2m', outBound: false },
  ], tools: [{ toolId: 't1', name: '光功率计', borrowed: false }, { toolId: 't2', name: '光纤熔接机', borrowed: true } ],
    pendingReturn: [{ epc: 'EPC-0002', reason: '故障 · 返修（换件）' }, { epc: 'EPC-0110', reason: '拆机回收（ORD-20250817-009）' }],
    returned: { repairCount: 3, dismantleCount: 12 } },
  maintenance: { items: [
    { deviceNo: 'EPC-0023', deviceType: '光猫', healthScore: 31, faultCount: 5, ageYears: 4, reason: '', priority: 'MUST_REPLACE', priorityLabel: '强替换' },
    { deviceNo: 'SPL-03-07', deviceType: '分光器', healthScore: 45, faultCount: 0, ageYears: 3, reason: '信号衰减', priority: 'SUGGEST', priorityLabel: '建议替换' },
    { deviceNo: 'OLT-01', deviceType: 'PON 9口', healthScore: 58, faultCount: 0, ageYears: 0, reason: '丢包率偏高', priority: 'WATCH', priorityLabel: '观察' },
  ] },
  measure: { opticalPowerDbm: -16.2, opticalPowerLabel: '正常', downloadMbps: 942, uploadMbps: 96, packetLossRate: 0 },
  // 空闲口排除 db.ports 中 RESERVED/USED 的 PON 口
  // 空闲口: 从 db.ports 动态排除 SPL-03 下已占用(RESERVED/USED)的 PON 口
  resources: { idlePorts: 3, nearestSplitter: 'SPL-03-07', idlePonPorts: ['P7', 'P9', 'P11', 'P13', 'P15'].filter((pon) => !db.ports.some((p) => p.quadCode === 'P-SPL03-0' + pon.slice(1))) },
  profile: { workerId: ME, name: db.byWorker(ME).name, groupName: db.byWorker(ME).groupName, staffNo: db.byWorker(ME).staffNo, phoneMasked: db.byWorker(ME).phoneMasked, online: true, serveYears: 3, month: { finished: 46, onTimeRate: 98, score: 4.9 } },
  performance: { period: '2025-08', summary: { finished: 46, onTimeRate: 98, score: 4.9 }, commissions: [
    { name: '装机提成', formula: '46 × ¥30', amount: 1380 }, { name: '抢修提成', formula: '12 × ¥20', amount: 240 }, { name: '满意度奖励', formula: '', amount: 120 },
  ], totalAmount: 1740, ranking: [{ name: '张师傅（本人）', rank: 1, self: true }, { name: '李师傅', rank: 2, self: false }, { name: '王师傅', rank: 3, self: false }] },
  schedule: { month: '2025-08', busyDays: [5, 6, 10, 17, 30], today: tickets.doing.concat(tickets.todo).filter((t) => t.scheduleSlot).map((t) => ({ time: (t.scheduleSlot.match(/\d{2}:\d{2}/) || [''])[0], ticketNo: t.ticketNo, address: t.address.replace(/^望京X · /, '') })) },
  settings: { online: true, radiusKm: 5, acceptTypes: ['新装宽带', '宽带变更', '拆机'] },
  feedbacks: { latest: { ticketNo: 'ORD-20250817-000', score: 5.0 }, monthAvgScore: 4.9, replyRate: 62, items: [
    { customerName: '王先生', score: 5.0, comment: '服务态度好', needReview: false }, { customerName: '李女士', score: 2.0, comment: '已转复核', needReview: true },
  ] },
  messages: { items: [
    { level: 'err', title: '台风应急', content: '台风后批量复测任务已下发，请核对受影响客户清单', sentAt: '08-16 18:00', read: false },
    { level: 'warn', title: '超时预警', content: 'TKT-20250817-012 抢修单 SLA 剩余不足 1 小时，请尽快到场处理', sentAt: '08-17 10:30', read: false },
    { level: 'err', title: '改派通知', content: '新单 TKT-20250817-005 抢修已分派给您', sentAt: '08-17 09:45', read: false },
    { level: 'ok', title: '配置下发', content: '全部预下发成功', sentAt: '08-17 09:18', read: true },
    { level: 'warn', title: '标签电量', content: 'EPC-0023 电量低，请携备用', sentAt: '08-17 08:00', read: true },
  ] },
  notices: { items: [
    { noticeId: 'n1', title: '台风季弱电井防水作业提示', category: '安全作业提醒', publishedAt: '08-16' },
    { noticeId: 'n2', title: '本周 GPU 千兆套餐物料配发说明', category: '物料公告', publishedAt: '08-15' },
    { noticeId: 'n3', title: '扫码绑定弱网离线功能上线', category: '功能公告', publishedAt: '08-14' },
  ] },
  faq: { items: [
    { faqId: 'f1', title: '光猫红灯/无法注册', summary: 'LOID 认证失败排查' },
    { faqId: 'f2', title: '光功率偏低', summary: '分光比与接头损耗排查' },
    { faqId: 'f3', title: '测速不达标', summary: '线路/终端/WiFi 分段定位' },
    { faqId: 'f4', title: '扫码绑定四码不一致', summary: '换机/重绑处理' },
  ] },
  serviceMessages: { items: [
    { from: 'dispatcher', content: '已为您接通调度中心，此单可支持改派/咨询。' },
    { from: 'worker', content: 'ORD-20250817-001 关联工单，可快捷转单。' },
  ] },
};
