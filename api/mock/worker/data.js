// mock data —— 师傅端假数据。字段对齐 api/openapi/worker/schemas.yaml。
// JSON 字段统一 lowerCamelCase(见 docs/contract/fields.md 第 0 节)。
'use strict';

// 新装 12 环节(terms.md 第 1 节,禁止增删改序)
function installStages(doneCount) {
  const names = ['用户下单', '资源核查', '端口预占', '合同收费', '标签预绑定', '创建认证账号', '预下发配置', '派单', '扫码绑定', '激活用户', '激活回调', '更新 GIS 地图'];
  const notes = ['08-17 09:02 · 完成', '08-17 09:05 · 有空闲端口 · 3m', '08-17 09:09 · 4m', '08-17 09:11 · 已收款 · 2m', '08-17 09:12 · EPC-0001 · 3m', '08-17 09:15 · 3m', '08-17 09:18 · 成功 · 3m', '08-17 09:20 · 张师傅已接单 · 2m', '进行中 · 请现场扫描光猫电子标签', '待扫码绑定完成后触发', '待处理', '待处理'];
  return names.map(function (name, i) {
    const n = i + 1;
    const result = n <= doneCount ? 'DONE' : n === doneCount + 1 ? 'DOING' : 'PENDING';
    return { stage: n, name: name, result: result, finishedAt: n <= doneCount ? notes[i].split(' · ')[0] : '', durationMinutes: n <= doneCount ? 3 : null, note: notes[i] };
  });
}

// 报障 6 环节(报障闭环)
function repairStages() {
  const items = [
    { stage: 1, name: '报障', note: '09:40 · 自动关联资产/端口' },
    { stage: 2, name: '诊断', note: '09:42 · 远程诊断入工单' },
    { stage: 3, name: '派单', note: '09:45 · 张师傅已接单 · SLA 计时' },
    { stage: 4, name: '修复', note: '进行中 · 到场处理并上报结果' },
    { stage: 5, name: '复核', note: '系统自动验证网络恢复' },
    { stage: 6, name: '回访', note: '满意度调查+回填' },
  ];
  return items.map(function (it) {
    it.result = it.stage <= 3 ? 'DONE' : it.stage === 4 ? 'DOING' : 'PENDING';
    return it;
  });
}

const quad = { assetCode: 'EPC-0001', customerCode: 'LOID-88A1', portCode: 'P-SPL03-07', addrCode: 'A-3-501', status: 'LINKED', matched: true };

const tickets = {
  doing: [
    { ticketNo: 'ORD-20250817-001', type: 'INSTALL', typeLabel: '1000M 安装', address: '望京X · 3栋501', distanceKm: 2.1, scheduleSlot: '今天 10:00-12:00', stage: 8, stageTotal: 12, status: 'SCAN_PENDING', statusLabel: '待扫码绑定', slaLeftMinutes: null },
    { ticketNo: 'ORD-20250817-002', type: 'INSTALL', typeLabel: '500M 安装', address: '望京X · 5栋302', distanceKm: 3.4, scheduleSlot: '今天 14:00-16:00', stage: 7, stageTotal: 12, status: 'ACCEPTED', statusLabel: '已接单', slaLeftMinutes: null },
    { ticketNo: 'TKT-20250817-012', type: 'REPAIR', typeLabel: '断网抢修', address: '望京X · 10栋1801', distanceKm: 1.8, scheduleSlot: '', stage: 4, stageTotal: 6, status: 'DOING', statusLabel: '修复中', slaLeftMinutes: 52 },
  ],
  todo: [
    { ticketNo: 'ORD-20250817-003', type: 'INSTALL', typeLabel: '300M 安装', address: '望京X · 12栋906', distanceKm: 1.7, scheduleSlot: '', stage: 8, stageTotal: 12, status: 'TODO', statusLabel: '待领取', note: '空闲端口已预占 · 标签已预绑定' },
    { ticketNo: 'ORD-20250817-004', type: 'CHANGE', typeLabel: '宽带变更', address: '望京X · 3栋501', distanceKm: 4.0, scheduleSlot: '', stage: 8, stageTotal: 12, status: 'TODO', statusLabel: '待领取', note: '空闲端口已预占 · 标签已预绑定' },
  ],
  done: [
    { ticketNo: 'ORD-20250817-000', type: 'INSTALL', typeLabel: '1000M 安装', address: '望京X · 1栋303', stage: 12, stageTotal: 12, status: 'DONE', statusLabel: '已激活', finishedAt: '08-17' },
  ],
};

const history = [
  { ticketNo: 'ORD-20250816-011', type: 'INSTALL', typeLabel: '500M 安装', address: '望京X · 6栋701', status: 'DONE', statusLabel: '已激活', finishedAt: '08-16' },
  { ticketNo: 'ORD-20250816-010', type: 'INSTALL', typeLabel: '300M 安装', address: '望京X · 9栋305', status: 'DONE', statusLabel: '已激活', finishedAt: '08-16' },
  { ticketNo: 'ORD-20250815-008', type: 'DISMANTLE', typeLabel: '拆机', address: '望京X · 2栋902', status: 'DONE', statusLabel: '已拆机', finishedAt: '08-15' },
];

const hall = [
  { ticketNo: 'EMG-20250817-001', type: 'EMERGENCY', typeLabel: '台风批量复测', address: '望京X片区 · 受影响 23 户', distanceKm: 0.8, status: 'TODO', statusLabel: '可抢' },
  { ticketNo: 'TKT-20250817-006', type: 'REPAIR', typeLabel: '抢修', address: '望京X · 8栋 · 断网', distanceKm: 1.2, status: 'TODO', statusLabel: '可抢' },
  { ticketNo: 'ORD-20250817-007', type: 'INSTALL', typeLabel: '新装', address: '望京X · 15栋', distanceKm: 2.8, status: 'TODO', statusLabel: '可抢' },
  { ticketNo: 'ORD-20250817-008', type: 'CHANGE', typeLabel: '变更', address: '望京X · 4栋', distanceKm: 3.5, status: 'TODO', statusLabel: '可抢' },
];

const installDetail = {
  ticketNo: 'ORD-20250817-001', bizNo: 'ORD-20250817-001', status: 'SCAN_PENDING', statusLabel: '待扫码绑定',
  product: '1000M 极速宽带（有线）· ¥199/月', customerName: '王先生', customerPhoneMasked: '138****1234',
  address: '望京X · 3栋 · 501', splitterPort: 'SPL-03-07 · PON 7口', preBindTag: 'EPC-0001', scheduleSlot: '今天 10:00-12:00',
  stages: installStages(8), quad: quad, riskCheck: { blacklistHit: false, graylistHit: false },
};

const repairDetail = {
  ticketNo: 'TKT-20250817-012', bizNo: 'TKT-20250817-012', status: 'DOING', statusLabel: '紧急 · 断网',
  product: '', customerName: '陈先生', customerPhoneMasked: '138****7788',
  address: '望京X · 10栋 · 1801', splitterPort: 'SPL-05-02 · PON 3口', preBindTag: 'EPC-0088',
  faultTypeLabel: '单户断网（紧急 SLA ≤4h）', reportedAt: '今天 09:40（已受理）', slaLeftMinutes: 52,
  remoteDiagnosis: '疑似光猫离线，光功率 -18.6 dBm（偏低），建议现场复核。',
  stages: repairStages(), quad: quad, riskCheck: { blacklistHit: false, graylistHit: false },
};

const dismantleDetail = {
  ticketNo: 'ORD-20250817-009', bizNo: 'ORD-20250817-009', status: 'DOING', statusLabel: '拆机',
  product: '', customerName: '刘女士', customerPhoneMasked: '138****5678',
  address: '望京X · 2栋 · 902', splitterPort: '', preBindTag: 'EPC-0110',
  stages: [], quad: quad, riskCheck: { blacklistHit: false, graylistHit: false },
};

module.exports = {
  quad: quad,
  tickets: tickets,
  history: { items: history, totalCount: 46 },
  hall: { items: hall },

  home: {
    workerName: '张师傅', groupName: '装机一组', phoneMasked: '138****8899',
    today: { accepted: 3, finished: 2, doing: 2, todo: 1 },
    ongoing: tickets.doing,
  },

  installDetail: installDetail,
  repairDetail: repairDetail,
  dismantleDetail: dismantleDetail,

  navi: { address: '望京X · 3栋 · 501', distanceKm: 2.1, etaMinutes: 8, entrance: '东门进 · 3栋1单元' },

  scanOk: { matched: true, result: 'MATCH', message: '四码校验一致，绑定已通过，请确认装维完成情况。', quad: quad },
  scanBad: { matched: false, result: 'MISMATCH', message: '扫码标签与此单预绑定标签不符，绑定已被系统拒绝。请核实换机或重绑。', quad: Object.assign({}, quad, { status: 'CONFLICT', matched: false }) },

  photos: { items: [{ photoId: 'p1', fileName: 'photo_0809.jpg', linked: true }] },

  report: {
    ticketNo: 'ORD-20250817-001', quad: quad,
    checks: { powerOn: true, opticalPowerDbm: -16.2, provisionDone: true, loidAuthPassed: true },
    provisionLog: { template: 'GPON-1000M-v3', preResult: '成功 · 08-17 09:18', onsiteResult: '成功 · 自动' },
  },

  activation: { ticketNo: 'ORD-20250817-001', loid: 'LOID-88A1', status: 'PENDING', statusLabel: '未生效', lastTry: '08-17 10:32 · 回调超时' },
  activationDone: { ticketNo: 'ORD-20250817-001', loid: 'LOID-88A1', status: 'SUCCESS', statusLabel: '已生效', lastTry: '08-17 11:02 · 激活成功' },

  charge: { ticketNo: 'ORD-20250817-001', amountDue: 199, amountDesc: '首月', payMethods: ['扫码支付', '现金', 'POS'] },

  replace: {
    ticketNo: 'ORD-20250817-001', oldEpc: 'EPC-0002', oldEpcStatus: '故障', replaceType: '故障调换 · 旧件返修',
    steps: [
      { name: '扫旧件', status: 'DONE', note: 'EPC-0002 已识别' },
      { name: '扫新件', status: 'TODO', note: '待扫码' },
      { name: '登记', status: 'TODO', note: '未完成' },
    ],
  },

  materials: {
    items: [
      { itemId: 'm1', name: '光猫', qty: 2, spec: 'GPON 千兆 · 含标签', outBound: false },
      { itemId: 'm2', name: '机顶盒', qty: 1, spec: 'IPTV 4K', outBound: false },
      { itemId: 'm3', name: '光纤跳线', qty: 5, spec: 'SC/APC 2m', outBound: false },
    ],
    tools: [
      { toolId: 't1', name: '光功率计', borrowed: false },
      { toolId: 't2', name: '光纤熔接机', borrowed: true },
    ],
    pendingReturn: [
      { epc: 'EPC-0002', reason: '故障 · 返修（换件）' },
      { epc: 'EPC-0110', reason: '拆机回收（ORD-20250817-009）' },
    ],
    returned: { repairCount: 3, dismantleCount: 12 },
  },

  maintenance: {
    items: [
      { deviceNo: 'EPC-0023', deviceType: '光猫', healthScore: 31, faultCount: 5, ageYears: 4, reason: '', priority: 'MUST_REPLACE', priorityLabel: '强替换' },
      { deviceNo: 'SPL-03-07', deviceType: '分光器', healthScore: 45, faultCount: 0, ageYears: 3, reason: '信号衰减', priority: 'SUGGEST', priorityLabel: '建议替换' },
      { deviceNo: 'OLT-01', deviceType: 'PON 9口', healthScore: 58, faultCount: 0, ageYears: 0, reason: '丢包率偏高', priority: 'WATCH', priorityLabel: '观察' },
    ],
  },

  measure: { opticalPowerDbm: -16.2, opticalPowerLabel: '正常', downloadMbps: 942, uploadMbps: 96, packetLossRate: 0 },
  resources: { idlePorts: 3, nearestSplitter: 'SPL-03-07', idlePonPorts: ['P7', 'P9', 'P11'] },

  profile: {
    workerId: 1024, name: '张师傅', groupName: '装机一组', staffNo: 'WK-1024', phoneMasked: '138****8899',
    online: true, serveYears: 3,
    month: { finished: 46, onTimeRate: 98, score: 4.9 },
  },

  performance: {
    period: '2025-08',
    summary: { finished: 46, onTimeRate: 98, score: 4.9 },
    commissions: [
      { name: '装机提成', formula: '46 × ¥30', amount: 1380 },
      { name: '抢修提成', formula: '12 × ¥20', amount: 240 },
      { name: '满意度奖励', formula: '', amount: 120 },
    ],
    totalAmount: 1740,
    ranking: [
      { name: '张师傅（本人）', rank: 1, self: true },
      { name: '李师傅', rank: 2, self: false },
      { name: '王师傅', rank: 3, self: false },
    ],
  },

  schedule: {
    month: '2025-08', busyDays: [5, 6, 10, 17, 30], today: [
      { time: '10:00', ticketNo: 'ORD-20250817-001', address: '3栋501' },
      { time: '14:00', ticketNo: 'ORD-20250817-002', address: '5栋302' },
      { time: '16:00', ticketNo: 'ORD-20250817-003', address: '12栋906' },
    ],
  },

  settings: { online: true, radiusKm: 5, acceptTypes: ['新装宽带', '宽带变更', '拆机'] },

  feedbacks: {
    latest: { ticketNo: 'ORD-20250817-000', score: 5.0 },
    monthAvgScore: 4.9, replyRate: 62,
    items: [
      { customerName: '王先生', score: 5.0, comment: '服务态度好', needReview: false },
      { customerName: '李女士', score: 2.0, comment: '已转复核', needReview: true },
    ],
  },

  messages: {
    items: [
      { level: 'err', title: '台风应急', content: '台风后批量复测任务已下发，请核对受影响客户清单', sentAt: '08-16 18:00', read: false },
      { level: 'warn', title: '超时预警', content: 'ORD-20250817-002 已上门超时 25 分钟', sentAt: '08-17 10:30', read: false },
      { level: 'err', title: '改派通知', content: '新单 TKT-20250817-005 抢修已分派给您', sentAt: '08-17 09:45', read: false },
      { level: 'ok', title: '配置下发', content: '全部预下发成功', sentAt: '08-17 09:18', read: true },
      { level: 'warn', title: '标签电量', content: 'EPC-0023 电量低，请携备用', sentAt: '08-17 08:00', read: true },
    ],
  },

  notices: {
    items: [
      { noticeId: 'n1', title: '台风季弱电井防水作业提示', category: '安全作业提醒', publishedAt: '08-16' },
      { noticeId: 'n2', title: '本周 GPU 千兆套餐物料配发说明', category: '物料公告', publishedAt: '08-15' },
      { noticeId: 'n3', title: '扫码绑定弱网离线功能上线', category: '功能公告', publishedAt: '08-14' },
    ],
  },

  faq: {
    items: [
      { faqId: 'f1', title: '光猫红灯/无法注册', summary: 'LOID 认证失败排查' },
      { faqId: 'f2', title: '光功率偏低', summary: '分光比与接头损耗排查' },
      { faqId: 'f3', title: '测速不达标', summary: '线路/终端/WiFi 分段定位' },
      { faqId: 'f4', title: '扫码绑定四码不一致', summary: '换机/重绑处理' },
    ],
  },

  serviceMessages: {
    items: [
      { from: 'dispatcher', content: '已为您接通调度中心，此单可支持改派/咨询。' },
      { from: 'worker', content: 'ORD-20250817-001 关联工单，可快捷转单。' },
    ],
  },
};
