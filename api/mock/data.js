// mock data —— 用户端视图。实体数据一律由 db.js 派生(王先生 customerId=1),
// 仅本端专属展示(脱敏档案/FAQ/消息等)保留本地。字段对齐 api/openapi/user.yaml。
'use strict';

const db = require('./db.js');
const ME = 1; // 当前登录客户: 王先生
const me = () => db.byCustomer(ME);

const profile = {
  customerId: ME,
  name: me().name,
  phoneMasked: me().phoneMasked,
  realName: { nameMasked: '王**', idType: '身份证', idNoMasked: me().idNoMasked, status: me().realNameStatus },
  addresses: [
    { addressId: 'ADDR-001', label: '望京X · 3栋 · 501', isDefault: true, contact: '王先生', phoneMasked: '138****1234', community: '望京X', building: '3栋', door: '501' },
    { addressId: 'ADDR-002', label: '望京Y · 1栋 · 101', isDefault: false, contact: '王先生', phoneMasked: '138****1234', community: '望京Y', building: '1栋', door: '101' },
  ],
  plan: { planId: 'PLAN-1000', name: '1000M 极速宽带', monthlyFee: 199, contractEnd: '2026-08', status: 'ACTIVE', installAddress: '望京X · 3栋 · 501' },
};

const security = { realNameStatus: 'VERIFIED', nameMasked: '王**', idNoMasked: '110***********1234', verifyAt: '2023-05-11', passwordUpdatedAt: '2025-01-03', phoneMasked: '138****1234' };
const verify = { status: 'VERIFIED', nameMasked: '王**', idNoMasked: '110***********1234', verifyAt: '2023-05-11', records: [
  { method: '证件 + 人像比对', time: '2023-05-11 10:12', result: 'PASS' },
  { method: '证件 OCR', time: '2023-05-11 10:08', result: 'PASS' },
] };
const notifySettings = { business: { bill: true, suspendResume: true, faultNotice: true, installProgress: true }, marketing: { promo: true, planRecommend: false }, channels: { inApp: true, sms: true, push: true } };

// 产品目录: db.products 的 broadband 子集
const products = {
  items: db.products.filter((p) => p.category === 'broadband').map((p) => ({
    productId: p.productId, category: p.category, name: p.name, bandwidth: p.bandwidth,
    monthlyFee: p.monthlyFee, description: p.description, contractMonths: p.contractMonths, featured: p.featured,
  })),
  addons: [
    { addonId: 'A-IPTV', name: 'IPTV 高清电视', monthlyFee: 10, description: '¥10/月 · 200+ 频道', subscribed: false },
    { addonId: 'A-WIFI', name: '全屋 WiFi', monthlyFee: 15, description: '¥15/月 · 信号覆盖', subscribed: false },
  ],
};

const productDetail = {
  product: products.items.find((p) => p.productId === 'P-1000'),
  specs: [
    { label: '安装费', value: '首装 ¥0' },
    { label: '光猫 / 路由器', value: '含设备 · 押金 ¥0' },
    { label: 'IPTV', value: '200+ 频道' },
    { label: '合约期', value: '24 个月 · 到期自动续约' },
  ],
  compare: products.items.filter((p) => p.productId !== 'P-1000'),
};

const addons = {
  available: products.addons.concat([
    { addonId: 'A-CLOUD', name: '云盘存储', monthlyFee: 8, description: '¥8/月 · 500GB', subscribed: false },
    { addonId: 'A-CAM', name: '家庭安全看护', monthlyFee: 20, description: '¥20/月 · 摄像头云存储', subscribed: false },
  ]),
  subscribed: [
    { addonId: 'A-IPTV-OLD', name: 'IPTV 高清电视', monthlyFee: 10, description: '2023-05-11 订购 · ¥10/月', subscribed: true },
    { addonId: 'A-WIFI-OLD', name: '全屋 WiFi', monthlyFee: 15, description: '2025-02-01 订购 · ¥15/月', subscribed: true },
  ],
};

// 订单: db.orders 中我的未归档单
function orderView(o) {
  const p = db.byProduct(o.productId) || {};
  return {
    orderNo: o.orderNo, status: db.statusOfStage(o.bizType === 'DISMANTLE' ? 0 : o.stage), statusLabel: db.STATUS_LABEL[db.statusOfStage(o.stage)],
    productName: p.name, address: o.addrLabel.replace(/ · /g, ''), stage: o.stage,
    stageLabel: db.STAGE_NAMES[o.stage - 1] || '', estimateFinish: o.stage === 9 ? '08-17 完成' : null, canRate: o.stage >= 12,
  };
}
const myOrders = db.orders.filter((o) => o.customerId === ME && !o.archived);
const orders = { items: myOrders.map(orderView) };

const cur = db.byOrder('ORD-20250817-001');
const curProduct = db.byProduct(cur.productId);
const curWorker = db.byWorker(cur.workerId);
const orderDetail = {
  order: {
    orderNo: cur.orderNo, status: 'INSTALLING', statusLabel: '装维中',
    productName: curProduct.name + ' · ¥' + curProduct.monthlyFee + '/月', address: cur.addrLabel,
    stage: cur.stage, stageLabel: db.STAGE_NAMES[cur.stage - 1], canRate: false,
  },
  submitedAt: cur.submittedAt,
  technicianName: curWorker.name, technicianPhoneMasked: curWorker.phoneMasked,
  completedStage: cur.stage - 1,
  timeline: db.timelineOf(cur).map((s) => ({
    stage: s.stage, title: s.name, result: s.result,
    meta: s.result === 'DONE' ? (s.finishedAt || '') + (s.duration && s.duration !== '—' ? ' · ' + s.duration : '') : s.result === 'DOING' ? '进行中 · 师傅现场操作' : '待处理',
  })),
};

// 账单/缴费/凭证/发票: db.bills/payments 中我的记录
const myBills = db.bills.filter((b) => b.customerId === ME);
const bills = {
  currentDue: myBills.filter((b) => b.status === 'UNPAID').reduce((s, b) => s + b.amount, 0),
  currentPeriod: '2025-08',
  items: myBills.map((b) => ({ billNo: b.billNo, period: b.period, productName: '1000M 极速宽带', periodRange: b.period.slice(5) + '-01 ~ ' + b.period.slice(5) + '-31', amount: b.amount, status: b.status, statusLabel: b.statusLabel })),
};
const curBill = myBills[0];
const billDetail = {
  bill: bills.items[0],
  items: [
    { name: '1000M 极速宽带套餐费', range: '08-01 ~ 08-31', amount: 199 },
    { name: 'IPTV 高清电视', range: '08-01 ~ 08-31', amount: 10 },
    { name: '全屋 WiFi', range: '08-01 ~ 08-31', amount: 15 },
    { name: '合约减免', range: '首年立减', amount: -66 },
  ],
  totalDue: curBill.amount, autoPayEnabled: false,
};
const myPays = db.payments.filter((p) => p.customerId === ME);
const paymentsView = { items: myPays.map((p) => ({ payNo: p.payNo, amount: p.amount, period: p.period, payMethod: p.method, paidAt: p.paidAt })) };
const receipts = {};
for (const p of myPays) receipts[p.payNo] = { receiptNo: 'OR-' + p.paidAt.slice(0, 10).replace(/-/g, '') + '-0001', customerName: me().name, phoneMasked: me().phoneMasked, amount: p.amount, period: p.period, payMethod: p.method, paidAt: p.paidAt, payNo: p.payNo };
const invoice = {
  titleType: '个人', title: '王先生', taxNo: null,
  availablePeriods: myBills.filter((b) => b.status === 'PAID').map((b) => ({ billNo: b.billNo, period: b.period, productName: '1000M 极速宽带', periodRange: b.period.slice(5) + '-01 ~ 31', amount: b.amount, status: 'PAID', statusLabel: '已缴' })),
  records: [{ period: '2025-07', amount: 158, issuedAt: '2025-07-26', pdfUrl: '#' }],
};

// 报障: db.repairTickets 中我的工单
const myFaults = db.repairTickets.filter((t) => t.customerId === ME);
const faults = { items: myFaults.map((t) => ({ ticketNo: t.ticketNo, faultType: t.faultType, faultTypeLabel: t.faultTypeLabel, address: t.addrLabel, createdAt: t.reportedAt, status: t.status, statusLabel: t.status === 'PROCESSING' ? '处理中' : '已解决' })) };
const curFault = myFaults.find((t) => t.status === 'PROCESSING') || myFaults[0];
const fWorker = db.byWorker(curFault.workerId);
const REPAIR_STEPS = ['报障', '诊断', '派单', '修复', '复核', '回访'];
const faultDetail = {
  fault: faults.items[0],
  technicianName: fWorker.name, technicianPhoneMasked: fWorker.phoneMasked,
  sla: '≤4h',
  timeline: REPAIR_STEPS.map((title, i) => ({
    step: i + 1, title, result: i + 1 < curFault.stage ? 'DONE' : i + 1 === curFault.stage ? 'DOING' : 'PENDING',
    meta: ['09:40 · 已自动关联资产/端口', '09:42 · 远程诊断入工单', '09:45 · ' + fWorker.name + '已接单 · SLA 计时', '进行中 · 到场处理并上报结果', '系统自动验证网络恢复', '满意度调查+回填'][i],
  })),
};

const complaints = { items: [{ complaintId: 'CP-001', type: 'billing', typeLabel: '计费争议 · 2025-07 账期', relOrderNo: '', description: '', status: 'RESOLVED', statusLabel: '已解决', createdAt: '2025-07-28' }] };
const faq = { items: [
  { id: 'faq-1', question: '如何修改套餐？', answer: '进入「我的套餐」→「改套餐」选择新套餐并提交。' },
  { id: 'faq-2', question: '如何开具电子发票？', answer: '进入「电子发票」选择可开票账期申请开票。' },
  { id: 'faq-3', question: '账单怎么看？', answer: '进入「我的账单」查看账单明细与费用构成。' },
  { id: 'faq-4', question: '如何迁址移机？', answer: '进入「我的套餐」→「迁址」，填写新地址提交。' },
  { id: 'faq-5', question: '上网故障如何自检？', answer: '进入「故障报修」→「自助排障」按引导排查。' },
] };

const home = {
  customerName: me().name, phoneMasked: me().phoneMasked, onlineStatus: '服务在线 · 网络正常', hasUnread: true,
  plan: profile.plan, currentBill: bills.currentDue, balance: 42, contractEnd: '2026-08',
  ongoingOrders: myOrders.filter((o) => o.stage > 0 && o.stage < 12).map(orderView),
  services: [
    { name: '宽带上网', desc: 'LOID 认证 · 带宽 1000M', status: 'NORMAL', statusLabel: '正常' },
    { name: 'IPTV 电视', desc: '增值服务 · 高清频道', status: 'NORMAL', statusLabel: '正常' },
  ],
};

const messages = { items: [
  { messageId: 'M-001', category: 'balance', title: '余额预警', content: '账户余额 ¥5.00 低于 ¥50，请及时充值避免停机', tag: '预警', tagLevel: 'balance', createdAt: '2025-08-17', read: false },
  { messageId: 'M-002', category: 'balance', title: '到期停机提醒', content: '账户余额不足，将于明日到期停机，充值后自动复机', tag: '停机', tagLevel: 'balance', createdAt: '2025-08-17', read: false },
  { messageId: 'M-003', category: 'billing', title: '8 月账单已出账', content: '应缴 ¥158.00 · 账期 08-01 ~ 08-31', tag: '未缴', tagLevel: 'bill', createdAt: '2025-08-17', read: false },
  { messageId: 'M-004', category: 'billing', title: '装维进度更新', content: '订单 ORD-20250817-001 已进入「扫码绑定」环节', tag: '新', tagLevel: 'info', createdAt: '2025-08-17', read: false },
  { messageId: 'M-005', category: 'fault', title: '故障公告', content: '望京X 片区 8-17 02:00~04:00 计划割接，可能短暂中断', tag: '公告', tagLevel: 'fault', createdAt: '2025-08-17', read: true },
  { messageId: 'M-006', category: 'promo', title: '夏日宽带优惠', content: '1000M 套餐首月立减 ¥40，点击领取', tag: '活动', tagLevel: 'promo', createdAt: '2025-08-17', read: true },
] };
const coupons = { items: [
  { couponId: 'CPN-001', amount: 40, threshold: 100, title: '1000M 套餐首月立减', expireAt: '2025-08-31', status: 'available' },
  { couponId: 'CPN-002', amount: 20, threshold: 200, title: '余额充值满 200 减 20', expireAt: '2025-09-30', status: 'available' },
], inviteLink: 'https://boss.example.com/invite/8f3a' };
const usage = { periodLabel: '本月已用流量（08-01 ~ 08-31）', used: 286.4, unit: 'GB', quota: 1000, percent: 29, dailyAvg: 9.2, forecastRemain: 698, detail: { down: 248.1, up: 38.3, iptvNote: '不计入流量' }, online: { duration: '3 天 14 小时', lastOnlineAt: '2025-08-17 07:41' } };
const diySteps = { items: [
  { id: 's1', title: '无法上网', desc: '光猫灯异常 / 完全断网', steps: [{ title: '1 检查光猫指示灯', desc: 'LOS 红灯说明光纤未接通，需报修' }, { title: '2 重启光猫与路由器', desc: '断电 30 秒后重新上电' }, { title: '3 检查认证状态', desc: '确认账号未欠费停机' }] },
  { id: 's2', title: '网速慢', desc: '视频卡顿 / 下载缓慢', steps: [{ title: '1 检查 WiFi 信号', desc: '靠近路由器重测' }, { title: '2 减少并发大流量', desc: '暂停下载/大流量应用' }] },
  { id: 's3', title: 'IPTV 无信号', desc: '电视黑屏 / 无频道', steps: [{ title: '1 检查机顶盒连接', desc: '确认网线与 HDMI 连接' }, { title: '2 重启机顶盒', desc: '断电重启' }] },
  { id: 's4', title: '光猫告警', desc: '红灯 / 闪烁异常', steps: [{ title: '1 检查光纤连接', desc: '确认光纤接头插紧' }, { title: '2 联系报修', desc: 'LOS 红灯持续需报修' }] },
] };
const agreement = {
  userAgreement: [
    '本平台为宽带装维全流程自助服务平台，用户使用前须完成实名认证。',
    '套餐资费以办理时页面展示为准，合约期内退订须按合约约定承担相应责任。',
    '用户应保证所提交地址、联系方式真实有效，用于资源核查与装维上门。',
    '缴费、充值、报障等操作记录与核心系统一致，用户可随时查询。',
    '平台对关键业务（余额/到期/停机/复机/账单/故障）按通知订阅设置推送消息。',
  ],
  privacyPolicy: [
    '收集范围：手机号、实名信息、家庭地址、联系方式，仅用于业务办理与装维服务。',
    '不向第三方出售或泄露用户个人信息，法律法规另有规定除外。',
    '用户可随时在「账号安全」中查看与维护实名信息。',
    '审计日志保留 36 个月，用户关键操作留痕可追溯。',
  ],
};

module.exports = {
  profile, security, verify, notifySettings, products, productDetail, addons,
  orders, orderDetail, bills, billDetail, payments: paymentsView, receipts, balance: { balance: 42, denominations: [50, 100, 200] },
  invoice, faults, faultDetail, complaints, faq, home, messages, coupons, usage, diySteps, agreement,
};
