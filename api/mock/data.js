// mock data —— 用户端视图。实体数据一律由 db.js 派生(王先生 customerId=1),
// 用户域专属实体(地址簿/套餐/增值服务/订阅/FAQ/消息/优惠券/流量/排障/协议/余额/发票/投诉)
// 亦存于 db(见 user/entities.js),本文件只做形状映射。字段对齐 api/openapi/user.yaml。
'use strict';

const db = require('./db.js');
const ME = 1; // 当前登录客户: 王先生
const me = () => db.byCustomer(ME);
const my = (table) => db[table].filter((r) => r.customerId === ME);
const addrLabel = (a) => a.community + ' · ' + a.building + ' · ' + a.door;

const myAddr = my('userAddresses');
const planRow = db.byPlanOf(ME);
const planProduct = db.byProduct(planRow.productId);
const profile = {
  customerId: ME,
  name: me().name,
  phoneMasked: me().phoneMasked,
  realName: { nameMasked: '王**', idType: me().idType, idNoMasked: me().idNoMasked, status: me().realNameStatus },
  addresses: myAddr.map((a) => ({ addressId: a.addressId, label: addrLabel(a), isDefault: a.isDefault, contact: a.contact, phoneMasked: a.phoneMasked, community: a.community, building: a.building, door: a.door })),
  plan: { planId: planRow.planId, name: planProduct.name, monthlyFee: planRow.monthlyFee, contractEnd: planRow.contractEnd, status: planRow.status, installAddress: addrLabel(myAddr.find((a) => a.isDefault) || myAddr[0]) },
};

const myAccount = my('userAccounts')[0];
const myNotify = db.notifyOf(ME) || { business: {}, marketing: {}, channels: {} };
const security = { realNameStatus: me().realNameStatus, nameMasked: '王**', idNoMasked: me().idNoMasked, verifyAt: me().verifyAt, passwordUpdatedAt: myAccount.passwordUpdatedAt, phoneMasked: me().phoneMasked };
const verify = { status: me().realNameStatus, nameMasked: '王**', idNoMasked: me().idNoMasked, verifyAt: me().verifyAt, records: my('userVerifyRecords') };
const notifySettings = myNotify;

const subscribedOf = (addonId) => my('addonSubscriptions').some((s) => s.addonId === addonId && s.status === 'ACTIVE');
const addonView = (a, subscribed) => ({ addonId: a.addonId, name: a.name, monthlyFee: a.monthlyFee, description: a.description, subscribed: subscribed });

// 产品目录: db.products 的 broadband 子集 + 增值服务目录(addonCatalog 在售)
const products = {
  items: db.products.filter((p) => p.category === 'broadband').map((p) => ({
    productId: p.productId, category: p.category, name: p.name, bandwidth: p.bandwidth,
    monthlyFee: p.monthlyFee, description: p.description, contractMonths: p.contractMonths, featured: p.featured,
  })),
  addons: db.addonCatalog.filter((a) => a.active).map((a) => addonView(a, subscribedOf(a.addonId))),
};

const specsOf = (productId) => (db.productSpecs.find((s) => s.productId === productId) || {}).specs || [];
const productDetail = {
  product: products.items.find((p) => p.productId === 'P-1000'),
  specs: specsOf('P-1000'),
  compare: products.items.filter((p) => p.productId !== 'P-1000'),
};

const addons = {
  available: products.addons,
  subscribed: my('addonSubscriptions').filter((s) => s.status === 'ACTIVE').map((s) => {
    const a = db.byAddon(s.addonId);
    return { addonId: s.subId, name: a.name, monthlyFee: a.monthlyFee, description: s.subscribedAt + ' 订购 · ¥' + a.monthlyFee + '/月', subscribed: true };
  }),
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

// 账单/缴费/凭证/发票: db.bills/payments + 用户域 userBillItems/userInvoices
const myBills = db.bills.filter((b) => b.customerId === ME);
const bills = {
  currentDue: myBills.filter((b) => b.status === 'UNPAID').reduce((s, b) => s + b.amount, 0),
  currentPeriod: '2025-08',
  items: myBills.map((b) => ({ billNo: b.billNo, period: b.period, productName: planProduct.name, periodRange: b.period.slice(5) + '-01 ~ ' + b.period.slice(5) + '-31', amount: b.amount, status: b.status, statusLabel: b.statusLabel })),
};
const curBill = myBills[0];
const billDetail = {
  bill: bills.items[0],
  items: db.userBillItems.filter((i) => i.billNo === curBill.billNo),
  totalDue: curBill.amount, autoPayEnabled: myAccount.autoPayEnabled,
};
const myPays = db.payments.filter((p) => p.customerId === ME);
const paymentsView = { items: myPays.map((p) => ({ payNo: p.payNo, amount: p.amount, period: p.period, payMethod: p.method, paidAt: p.paidAt })) };
const receipts = {};
for (const p of myPays) receipts[p.payNo] = { receiptNo: 'OR-' + p.paidAt.slice(0, 10).replace(/-/g, '') + '-0001', customerName: me().name, phoneMasked: me().phoneMasked, amount: p.amount, period: p.period, payMethod: p.method, paidAt: p.paidAt, payNo: p.payNo };
const lastInvoice = db.userInvoices[db.userInvoices.length - 1] || {};
const invoice = {
  titleType: lastInvoice.titleType || '个人', title: lastInvoice.title || me().name, taxNo: lastInvoice.taxNo || null,
  availablePeriods: myBills.filter((b) => b.status === 'PAID').map((b) => ({ billNo: b.billNo, period: b.period, productName: planProduct.name, periodRange: b.period.slice(5) + '-01 ~ 31', amount: b.amount, status: 'PAID', statusLabel: '已缴' })),
  records: my('userInvoices').map((r) => ({ period: r.period, amount: r.amount, issuedAt: r.issuedAt, pdfUrl: r.pdfUrl })),
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

const complaints = { items: my('userComplaints').map((c) => ({ complaintId: c.complaintId, type: c.type, typeLabel: c.typeLabel, relOrderNo: c.relOrderNo, description: c.description, status: c.status, statusLabel: c.statusLabel, createdAt: c.createdAt })) };
const faq = { items: db.userFaqs.filter((f) => f.active).sort((a, b) => a.sort - b.sort).map((f) => ({ id: f.faqId, question: f.question, answer: f.answer })) };

const myLoid = db.loids.find((l) => l.customerId === ME);
const home = {
  customerName: me().name, phoneMasked: me().phoneMasked, onlineStatus: '服务在线 · 网络正常', hasUnread: my('userMessages').some((m) => !m.read),
  plan: profile.plan, currentBill: bills.currentDue, balance: db.byBalanceOf(ME).balance, contractEnd: planRow.contractEnd,
  ongoingOrders: myOrders.filter((o) => o.stage > 0 && o.stage < 12).map(orderView),
  services: [
    { name: '宽带上网', desc: 'LOID 认证 · 带宽 ' + (myLoid ? myLoid.bandwidth : '—'), status: 'NORMAL', statusLabel: '正常' },
  ].concat(addons.subscribed.map((s) => s.name === 'IPTV 高清电视' ? { name: 'IPTV 电视', desc: '增值服务 · 高清频道', status: 'NORMAL', statusLabel: '正常' } : null).filter(Boolean)),
};

const messages = { items: my('userMessages') };
const coupons = { items: my('coupons').filter((c) => c.active).map((c) => ({ couponId: c.couponId, amount: c.amount, threshold: c.threshold, title: c.title, expireAt: c.expireAt, status: c.status })), inviteLink: db.inviteConfig[0].link };

const myUsage = db.byUsageOf(ME, '2025-08') || {};
const usage = {
  periodLabel: '本月已用流量（08-01 ~ 08-31）', used: myUsage.used, unit: myUsage.unit, quota: myUsage.quota,
  percent: Math.round(myUsage.used / myUsage.quota * 100), dailyAvg: myUsage.dailyAvg, forecastRemain: Math.round(myUsage.quota - myUsage.used),
  detail: { down: myUsage.down, up: myUsage.up, iptvNote: myUsage.iptvNote },
  online: { duration: myUsage.onlineDuration, lastOnlineAt: myUsage.lastOnlineAt },
};
const diySteps = { items: db.diyGuides.filter((g) => g.active).sort((a, b) => a.sort - b.sort).map((g) => ({ id: g.guideId, title: g.title, desc: g.desc, steps: g.steps })) };
const agreement = {
  userAgreement: db.agreements.find((a) => a.type === 'userAgreement').clauses,
  privacyPolicy: db.agreements.find((a) => a.type === 'privacyPolicy').clauses,
};

module.exports = {
  profile, security, verify, notifySettings, products, productDetail, addons,
  orders, orderDetail, bills, billDetail, payments: paymentsView, receipts, balance: { balance: db.byBalanceOf(ME).balance, denominations: db.topupDenominations.map((d) => d.amount) },
  invoice, faults, faultDetail, complaints, faq, home, messages, coupons, usage, diySteps, agreement,
};
