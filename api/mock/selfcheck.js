// selfcheck.js —— 三端数据关系不变量自检
// 用法: node api/mock/selfcheck.js  (退出码 0=全部通过)
// 校验对象是 db.js 事实库与三端派生视图,规则来源 api/DATA-ALIGNMENT.md 第 2 节口径。
'use strict';

const db = require('./db.js');
const user = require('./data.js');
const worker = require('./worker/data.js');
const adminOrder = require('./admin/data/order.js');
const adminQuad = require('./admin/data/quad.js');
const adminOss = require('./admin/data/oss.js');
const adminBilling = require('./admin/data/billing.js');
const adminIntel = require('./admin/data/intel.js');

let failed = 0;
function check(name, cond) {
  if (cond) console.log('PASS ' + name);
  else { failed++; console.log('FAIL ' + name); }
}
const get = (h, k) => {
  const v = h['GET ' + k];
  return typeof v === 'function' ? v({ query: {}, params: {} }) : v;
};
const q = (k, v) => ({ query: new URLSearchParams({ [k]: v }) });

// 1. 引用完整性: 订单的客户/产品/端口/师傅/LOID 全部可解析
for (const o of db.orders) {
  check('order ' + o.orderNo + ' customer 存在', !!db.byCustomer(o.customerId));
  if (o.productId) check('order ' + o.orderNo + ' product 存在', !!db.byProduct(o.productId));
  if (o.workerId) check('order ' + o.orderNo + ' worker 存在', !!db.byWorker(o.workerId));
  if (o.portQuad) check('order ' + o.orderNo + ' port 存在', !!db.ports.find((p) => p.quadCode === o.portQuad));
}
// 2. 端口占用与订单状态一致: RESERVED/USED 必须可解析(订单或 usedBy LOID),且订单尚未拆机完成
for (const p of db.ports) {
  if (p.status === 'IDLE' || p.status === 'DISABLED') continue;
  if (!p.orderId) {
    check('port ' + p.quadCode + ' usedBy 认证账号存在', !!db.loids.find((l) => l.loid === p.usedBy));
    continue;
  }
  const o = db.byOrder(p.orderId);
  check('port ' + p.quadCode + ' 占用订单存在', !!o);
  if (o) check('port ' + p.quadCode + ' 与订单地址一致', o.addrCode === p.addrCode);
}
// 3. 未收费不派单: stage<8 的单不得有师傅
for (const o of db.orders) {
  if (o.bizType !== 'DISMANTLE' && o.stage < 8) check('order ' + o.orderNo + ' 未派单', !o.workerId);
}
// 4. 四码只出现在扫码绑定(环节9)之后
for (const qd of db.quads()) {
  const o = db.byOrder(qd.orderNo);
  if (o) check('quad ' + qd.assetCode + ' 属于 stage>=9 或拆机单', o.stage >= 9 || o.bizType === 'DISMANTLE');
  check('quad ' + qd.assetCode + ' 端口存在', !!db.ports.find((p) => p.quadCode === qd.portCode) || !o);
}
// 5. GIS 同步仅限 stage=12
const gis = get(adminIntel, '/gis/address-links').items;
for (const g of gis) {
  const o = db.byOrder(g.orderNo);
  check('gis ' + g.orderNo + ' 同步状态与环节一致', (o.stage >= 12) === (g.syncStatus === '已同步'));
}
// 6. 三端同单同口径: ORD-20250817-001 的 stage/status
const uo = user.orders.items.find((x) => x.orderNo === 'ORD-20250817-001');
const wo = worker.tickets.doing.find((x) => x.ticketNo === 'ORD-20250817-001');
const ao = get(adminOrder, '/orders').items.find((x) => x.orderNo === 'ORD-20250817-001');
check('三端 ORD-001 stage 一致=9', uo.stage === 9 && wo.stage === 9 && ao.stage === 9);
check('三端 ORD-001 status 一致=INSTALLING', uo.status === 'INSTALLING' && wo.status !== 'DONE' && ao.status === 'INSTALLING');
check('用户端详情师傅=张师傅/8899', user.orderDetail.technicianName === '张师傅' && user.orderDetail.technicianPhoneMasked === '138****8899');
// 7. 工单详情四码与工单实体一致
const rd = worker.repairDetail, dd = worker.dismantleDetail, id = worker.installDetail;
check('抢修详情四码=自身', rd.quad.addrCode === 'A-10-1801' && rd.quad.assetCode === rd.preBindTag);
check('拆机详情四码=自身', dd.quad.addrCode === 'A-2-902' && dd.quad.assetCode === dd.preBindTag);
check('安装详情四码=自身', id.quad.addrCode === 'A-3-501' && id.quad.assetCode === id.preBindTag);
// 8. 账单口径: 王先生 2025-08 三端均为未缴 ¥158
const ub = user.bills.items.find((b) => b.period === '2025-08');
const ab = get(adminBilling, '/bills').items.find((b) => b.period === '2025-08' && b.customerName === '王先生');
check('王先生 2025-08 未缴 158(用户端)', ub && ub.amount === 158 && ub.status === 'UNPAID');
check('王先生 2025-08 未缴 158(admin)', ab && ab.amount === '¥158' && ab.status === '未缴');
check('未缴账期不可开票', !user.invoice.availablePeriods.some((p) => p.period === '2025-08'));
check('缴费凭证与流水一一对应', Object.keys(user.receipts).every((k) => user.payments.items.some((p) => p.payNo === k)));
// 9. admin 视图引用完整性: 订单/拆机/投诉行的客户均在 db.customers
for (const r of get(adminOrder, '/orders').items) check('admin 订单 ' + r.orderNo + ' 客户在册', db.customers.some((c) => c.name === r.customer));
for (const r of get(adminOrder, '/dismantles').items.filter((x) => x.dismantleNo.startsWith('ORD-20250817'))) {
  check('admin 拆机 ' + r.dismantleNo + ' 客户在册', db.customers.some((c) => c.name === r.customer));
}
for (const t of db.repairTickets) check('报障 ' + t.ticketNo + ' 客户在册', !!db.byCustomer(t.customerId));
// 10. 扫码记录: 现场扫码=预绑定
for (const s of get(adminQuad, '/scan-logs').items.filter((x) => x.result === 'MATCH')) check('扫码 ' + s.orderNo + ' 两码一致', s.scannedTag === s.preboundTag);
// 11. 认证账号: loid 客户在册,RESERVED 端口占用订单存在(admin oss 视图)
for (const l of db.loids) check('loid ' + l.loid + ' 客户在册', !!db.byCustomer(l.customerId));
for (const r of get(adminOss, '/reserves').items.filter((x) => x.status === 'RESERVED')) check('预占 ' + r.reserveId + ' 订单存在', !!db.byOrder(r.orderId));
// 12. 师傅端空闲口不与 db.ports 占用冲突(SPL-03 下已占用 P7/P3/P9)
check('师傅端空闲口不含已占用口', !worker.resources.idlePonPorts.some((pon) => ['P7', 'P3', 'P9'].includes(pon)));
// 13. 预绑标签/认证账号引用完整: 订单与报障单的 epc/loid 一律可解析
for (const o of db.orders) {
  if (o.preBindTag) check('order ' + o.orderNo + ' 预绑标签在册', !!db.assets.find((a) => a.epc === o.preBindTag));
  if (o.loid) check('order ' + o.orderNo + ' 认证账号在册', !!db.loids.find((l) => l.loid === o.loid));
}
for (const t of db.repairTickets) {
  check('ticket ' + t.ticketNo + ' 预绑标签在册', !!db.assets.find((a) => a.epc === t.preBindTag));
  check('ticket ' + t.ticketNo + ' 认证账号在册', !!db.loids.find((l) => l.loid === t.loid));
  check('ticket ' + t.ticketNo + ' 端口在册', !!db.ports.find((p) => p.quadCode === t.portQuad));
}
// 14. 地区自洽: 实体地址可解析到统一维护的区域树,地址库 regionName 为有效外键
const regionNames = new Set(db.regions.map((r) => r.name));
for (const list of [db.orders, db.repairTickets, db.ports]) {
  for (const x of list) {
    const rg = db.regionOfAddr(x.addrCode);
    check((x.orderNo || x.ticketNo || x.quadCode) + ' 地址归属区域可解析', !!rg && regionNames.has(rg));
  }
}
for (const a of db.addresses) check('address ' + a.name + ' regionName 外键有效', regionNames.has(a.regionName));
// 15. 调度池引用完整: sourceNo 有实体,候选师傅在册
for (const w of get(adminOrder, '/dispatch/pool').items) {
  check('pool ' + w.ticketNo + ' sourceNo 存在', !!db.byOrder(w.sourceNo) || !!db.repairTickets.find((t) => t.ticketNo === w.sourceNo));
  for (const n of (w.candidates || '').split(/[\/·]/)) {
    const nm = n.trim().replace(' · ', '').trim();
    if (nm && nm.endsWith('师傅')) check('pool 师傅 ' + nm + ' 在册', db.workers.some((k) => k.name === nm));
  }
}
// 16. 师傅域实体管理完整: 师傅端用到的全部数据均在 db 实体表,且引用可解析、admin 可管
const adminWorker = require('./admin/data/worker.js');
for (const p of db.workerProfiles) check('workerProfile ' + p.workerId + ' 主档在册', !!db.byWorker(p.workerId));
for (const c of db.workerCommissions) check('commission ' + c.name + ' 师傅在册', !!db.byWorker(c.workerId));
for (const f of db.workerFeedbacks) {
  check('feedback ' + f.feedbackId + ' 师傅在册', !!db.byWorker(f.workerId));
  check('feedback ' + f.feedbackId + ' 单号可解析', !!db.byOrder(f.ticketNo) || !!db.repairTickets.find((t) => t.ticketNo === f.ticketNo));
}
for (const m of db.workerMaterials) check('material ' + m.itemId + ' 师傅在册', !!db.byWorker(m.workerId));
for (const t of db.workerTools) check('tool ' + t.toolId + ' 师傅在册', !!db.byWorker(t.workerId));
for (const r of db.assetReturns) {
  check('return ' + r.returnId + ' 师傅在册', !!db.byWorker(r.workerId));
  check('return ' + r.returnId + ' epc 在资产台账', !!db.assets.find((a) => a.epc === r.epc));
}
for (const d of db.deviceMaintenances) {
  const compact = (s) => String(s).replace(/[-\s]/g, '');
  check('maintenance 设备 ' + d.deviceNo + ' 在资产或端口台账',
    !!db.assets.find((a) => a.epc === d.deviceNo)
    || db.ports.some((p) => compact(p.quadCode).indexOf(compact(d.deviceNo).replace(/^P/, '')) >= 0)
    || compact(d.deviceNo).indexOf('OLT') === 0);
}
for (const h of worker.hall.items) {
  check('hall ' + h.ticketNo + ' 可溯源', !!db.repairTickets.find((t) => t.ticketNo === h.ticketNo) || !!db.hallExtras.find((x) => x.ticketNo === h.ticketNo));
}
check('师傅端消息全部来自 db.workerMessages', worker.messages.items.length === db.workerMessages.filter((m) => m.workerId === 1024).length);
check('师傅端公告全部来自 db.workerNotices(active)', worker.notices.items.every((n) => db.workerNotices.some((x) => x.noticeId === n.noticeId && x.active)));
check('师傅端 FAQ 全部来自 db.workerFaqs(active)', worker.faq.items.every((f) => db.workerFaqs.some((x) => x.faqId === f.faqId && x.active)));
check('师傅端绩效=提成合计', worker.performance.totalAmount === db.workerCommissions.reduce((s, c) => s + (c.workerId === 1024 ? c.amount : 0), 0));
check('师傅端排名覆盖全部师傅', worker.performance.ranking.length === db.workerProfiles.length);
check('师傅端排期来自 db.workerSchedules', Array.isArray(worker.schedule.busyDays) && worker.schedule.busyDays.every((d) => db.workerSchedules.some((s) => s.busyDays.includes(d))));
// 17. admin 师傅管理覆盖: 列表含全部在册师傅,且与师傅端档案同源
const aw = get(adminWorker, '/workers');
check('admin 师傅列表覆盖全部在册师傅', aw.items.length === db.workers.length && aw.items.every((x) => !!db.byWorker(x.workerId)));
check('admin 师傅月绩效与师傅端档案同源', get(adminWorker, '/worker-performances').items.every((r) => db.byWorkerProfile(r.workerId).month.finished === r.finished));
check('admin 公告与师傅端同源', get(adminWorker, '/notices').items.length === db.workerNotices.length);
check('admin FAQ 与师傅端同源', get(adminWorker, '/faqs').items.length === db.workerFaqs.length);
check('admin 抢单池与师傅端同源', get(adminWorker, '/hall-items').items.length === worker.hall.items.length);
check('admin 师傅域实体表全部可 crud', ['workerProfiles', 'workerCommissions', 'workerFeedbacks', 'workerMessages', 'workerNotices', 'workerFaqs', 'workerMaterials', 'workerTools', 'assetReturns', 'deviceMaintenances', 'hallExtras', 'serviceMessages', 'workerSchedules'].every((t) => db.TABLE_NAMES.includes(t) && !!db.tables[t]));

// 18. 用户域实体引用完整: customerId/addonId/billNo/productId 一律可解析
const USER_TABLES = ['userAccounts', 'userAddresses', 'userPlans', 'addonSubscriptions', 'notifyPrefs', 'userMessages', 'coupons', 'usageRecords', 'balances', 'userInvoices', 'userComplaints', 'userVerifyRecords'];
for (const t of USER_TABLES) for (const r of db[t]) check('user ' + t + ' 客户在册', !!db.byCustomer(r.customerId));
for (const s of db.addonSubscriptions) check('订购 ' + s.subId + ' 增值服务在册', !!db.addonCatalog.find((a) => a.addonId === s.addonId));
for (const p of db.userPlans) check('订购 ' + p.planId + ' 产品在册', !!db.byProduct(p.productId));
for (const a of db.userAddresses) check('地址簿 ' + a.addressId + ' addrCode 在册', !!db.orders.some((o) => o.addrCode === a.addrCode) || db.ports.some((p) => p.addrCode === a.addrCode));
// 19. 发票口径: 仅已缴账期可开票,账单明细项挂在真实账单上
for (const r of db.userInvoices) {
  const b = db.bills.find((x) => x.billNo === r.billNo);
  check('发票 ' + r.invoiceId + ' 账单在册且已缴', !!b && b.status === 'PAID' && b.period === r.period);
}
for (const i of db.userBillItems) check('账单明细 ' + i.name + ' 挂真实账单', !!db.bills.find((b) => b.billNo === i.billNo));
// 20. 用户端视图与 db 同源(用户端用到的数据一律由 db 派生,admin 端可管理)
const adminUser = require('./admin/data/userdata.js');
check('用户端消息与 db 同源', user.messages.items.length === db.userMessages.filter((m) => m.customerId === 1).length);
check('用户端 FAQ 与 db 同源', user.faq.items.length === db.userFaqs.filter((f) => f.active).length);
check('用户端优惠券与 db 同源', user.coupons.items.length === db.coupons.filter((c) => c.active).length);
check('用户端排障指南与 db 同源', user.diySteps.items.length === db.diyGuides.filter((g) => g.active).length);
check('用户端协议与 db 同源', user.agreement.userAgreement.length === db.agreements.find((a) => a.type === 'userAgreement').clauses.length);
check('用户端余额/流量/通知与 db 同源', user.balance.balance === db.byBalanceOf(1).balance && user.usage.used === db.byUsageOf(1, '2025-08').used && user.notifySettings === db.notifyOf(1));
check('用户端地址簿与 db 同源', user.profile.addresses.length === db.userAddresses.length && user.profile.addresses[0].label.indexOf(db.userAddresses[0].community) === 0);
check('用户端套餐/增值服务与 db 同源', user.profile.plan.planId === db.byPlanOf(1).planId && user.addons.subscribed.length === db.addonSubscriptions.filter((s) => s.customerId === 1 && s.status === 'ACTIVE').length);
// 21. admin 用户端数据管理覆盖: 每张用户域实体表都有管理视图且可 crud
const USER_ADMIN_PATHS = { userAccounts: '/user-accounts', userAddresses: '/user-addresses', userPlans: '/user-plans', addonCatalog: '/addons', addonSubscriptions: '/addon-subscriptions', notifyPrefs: '/user-notify-settings', userFaqs: '/user-faqs', userMessages: '/user-messages', coupons: '/coupons', inviteConfig: '/invite-config', usageRecords: '/user-usages', diyGuides: '/diy-guides', agreements: '/agreements', balances: '/user-balances', topupDenominations: '/topup-denominations', userInvoices: '/user-invoices', userComplaints: '/user-complaints', userVerifyRecords: '/user-verify-records', productSpecs: '/product-specs', userBillItems: '/user-bill-items' };
for (const t of Object.keys(USER_ADMIN_PATHS)) {
  const res = get(adminUser, USER_ADMIN_PATHS[t]);
  check('admin 管理用户域 ' + t + ' 列表可用', !!res && Array.isArray(res.items) && res.items.length === db[t].length);
  check('用户域 ' + t + ' 可 crud', db.TABLE_NAMES.includes(t) && !!db.tables[t]);
}
// 22. 产品化入口: 用户列表覆盖全部在册客户,用户详情聚合与 db 各域同源
const userList = get(adminUser, '/users');
check('admin 用户列表覆盖全部在册客户', userList.items.length === db.customers.length && userList.items.every((r) => !!db.byCustomer(r.customerId)));
for (const c of db.customers) {
  const d = adminUser['GET /users/{customerId}']({ params: { customerId: c.customerId } });
  const of = (t) => db[t].filter((r) => r.customerId === c.customerId);
  check('详情 ' + c.name + ' 聚合与 db 同源', !!d
    && d.addresses.length === of('userAddresses').length
    && d.messages.length === of('userMessages').length
    && d.coupons.length === of('coupons').length
    && d.complaints.length === of('userComplaints').length
    && d.invoices.length === of('userInvoices').length
    && d.verifyRecords.length === of('userVerifyRecords').length
    && d.orders.length === db.orders.filter((o) => o.customerId === c.customerId && !o.archived).length
    && d.faults.length === db.repairTickets.filter((t) => t.customerId === c.customerId).length
    && d.bills.length === db.bills.filter((b) => b.customerId === c.customerId).length
    && d.payments.length === db.payments.filter((p) => p.customerId === c.customerId).length
    && d.balance.balance === db.byBalanceOf(c.customerId).balance);
}
// 23. 概述统计口径: 详情 stats/recent 与 db 各域计数一致(订单含归档单)
for (const c of db.customers) {
  const d = adminUser['GET /users/{customerId}']({ params: { customerId: c.customerId } });
  const all = (t, f) => db[t].filter(f);
  const mine = (t) => all(t, (r) => r.customerId === c.customerId);
  const s = d.stats;
  check('概述 ' + c.name + ' 订单统计含归档单', s.orders.total === mine('orders').length
    && s.orders.completed === mine('orders').filter((o) => o.stage >= 12).length
    && s.orders.ongoing === db.orders.filter((o) => o.customerId === c.customerId && !o.archived && o.stage > 0 && o.stage < 12).length);
  check('概述 ' + c.name + ' 缴费/账单统计一致', s.payments.totalAmount === mine('payments').reduce((x, p) => x + p.amount, 0)
    && s.bills.unpaidAmount === mine('bills').filter((b) => b.status !== 'PAID').reduce((x, b) => x + b.amount, 0));
  check('概述 ' + c.name + ' 报障/投诉/消息/优惠券统计一致', s.faults.total === mine('repairTickets').length
    && s.complaints.total === mine('userComplaints').length
    && s.messages.unread === mine('userMessages').filter((m) => !m.read).length
    && s.coupons.available === mine('coupons').filter((x) => x.active).length);
  check('概述 ' + c.name + ' 近期动态≤8且倒序', d.recent.length <= 8
    && d.recent.every((e, i) => i === 0 || String(d.recent[i - 1].at) >= String(e.at)));
}

console.log(failed === 0 ? '\nALL CHECKS PASSED' : '\n' + failed + ' CHECKS FAILED');
process.exit(failed === 0 ? 0 : 1);
