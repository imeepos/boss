// 假数据 —— 用户端数据管理(契约: api/openapi/admin/userdata.yaml)。
// 产品化结构: /users 用户列表 + /users/{customerId} 用户详情聚合(含概述统计 stats/近期动态 recent);
// 其余端点为用户域实体表的管理视图。列表由 db 派生;写操作直接改 db 事实库。
'use strict';

const db = require('../../db.js');

const ok = { code: 0, message: 'success' };
const cname = (id) => db.byCustomer(id).name;

// 用户列表行: 客户主档 join 账户/套餐/余额/在途单/欠费
function userRow(c) {
  const acct = db.userAccounts.find((a) => a.customerId === c.customerId) || {};
  const plan = db.userPlans.find((p) => p.customerId === c.customerId && p.status === 'ACTIVE');
  const prod = plan ? db.byProduct(plan.productId) || {} : {};
  const unpaid = db.bills.filter((b) => b.customerId === c.customerId && b.status !== 'PAID');
  const ongoing = db.orders.filter((o) => o.customerId === c.customerId && !o.archived && o.stage > 0 && o.stage < 12);
  return {
    customerId: c.customerId, name: c.name, phoneMasked: c.phoneMasked, username: acct.username || '—',
    realName: c.realNameLabel, service: c.serviceLabel, productName: prod.name || '—',
    balance: '¥' + db.byBalanceOf(c.customerId).balance, unpaidCount: unpaid.length,
    ongoingCount: ongoing.length, registeredAt: acct.registeredAt || '—',
  };
}

// 概述统计: 该用户历史记录各域计数(含归档单),口径与 db 一致
function userStats(id) {
  const orders = db.orders.filter((o) => o.customerId === id);
  const faults = db.repairTickets.filter((t) => t.customerId === id);
  const pays = db.payments.filter((p) => p.customerId === id);
  const bills = db.bills.filter((b) => b.customerId === id);
  const coupons = db.coupons.filter((c) => c.customerId === id);
  const complaints = db.userComplaints.filter((c) => c.customerId === id);
  const messages = db.userMessages.filter((m) => m.customerId === id);
  const acct = db.userAccounts.find((a) => a.customerId === id) || {};
  const plan = db.userPlans.find((p) => p.customerId === id && p.status === 'ACTIVE');
  const onNetMonths = acct.registeredAt
    ? Math.max(1, Math.round((new Date('2025-08-17') - new Date(acct.registeredAt)) / (30.5 * 24 * 3600 * 1000)))
    : null;
  const payTotal = pays.reduce((s, p) => s + p.amount, 0);
  return {
    orders: { total: orders.length, completed: orders.filter((o) => o.stage >= 12).length, ongoing: orders.filter((o) => !o.archived && o.stage > 0 && o.stage < 12).length },
    faults: { total: faults.length, resolved: faults.filter((t) => t.status === 'RESOLVED').length },
    payments: { count: pays.length, totalAmount: payTotal },
    bills: { total: bills.length, unpaidAmount: bills.filter((b) => b.status !== 'PAID').reduce((s, b) => s + b.amount, 0) },
    coupons: { total: coupons.length, available: coupons.filter((c) => c.active).length },
    complaints: { total: complaints.length, resolved: complaints.filter((c) => c.status === 'RESOLVED').length },
    messages: { total: messages.length, unread: messages.filter((m) => !m.read).length },
    onNet: { since: acct.registeredAt || null, months: onNetMonths, avgMonthlySpend: onNetMonths ? Math.round(payTotal / onNetMonths) : null },
    contract: plan ? { contractEnd: plan.contractEnd, remainMonths: Math.max(0, Math.round((new Date(plan.contractEnd + '-28') - new Date('2025-08-17')) / (30.5 * 24 * 3600 * 1000))) } : null,
  };
}

// 近期动态: 订单/缴费/报障按时间倒序合并取前 8 条
function userRecent(id) {
  const events = [];
  for (const o of db.orders.filter((x) => x.customerId === id)) events.push({ type: '订单', no: o.orderNo, desc: (o.bizType || 'INSTALL') + ' · ' + o.addrLabel, result: db.STATUS_LABEL[db.statusOfStage(o.stage)], at: o.submittedAt || '' });
  for (const p of db.payments.filter((x) => x.customerId === id)) events.push({ type: '缴费', no: p.payNo, desc: p.method + ' · 账期 ' + p.period, result: '¥' + p.amount, at: p.paidAt });
  for (const t of db.repairTickets.filter((x) => x.customerId === id)) events.push({ type: '报障', no: t.ticketNo, desc: t.faultTypeLabel, result: t.status === 'RESOLVED' ? '已解决' : '处理中', at: t.reportedAt });
  for (const c of db.userComplaints.filter((x) => x.customerId === id)) events.push({ type: '投诉', no: c.complaintId, desc: c.typeLabel, result: c.statusLabel, at: c.createdAt });
  return events.sort((a, b) => String(b.at).localeCompare(String(a.at))).slice(0, 8);
}

// 用户详情聚合: 该用户在用户端可见的全部数据,按域分组
function userDetail(id) {
  const c = db.byCustomer(id);
  if (!c) return null;
  const acct = db.userAccounts.find((a) => a.customerId === id) || {};
  const of = (t) => db[t].filter((r) => r.customerId === id);
  const plans = of('userPlans').map((p) => ({ ...p, productName: (db.byProduct(p.productId) || {}).name }));
  const addons = of('addonSubscriptions').map((s) => ({ subId: s.subId, addonName: db.byAddon(s.addonId).name, monthlyFee: db.byAddon(s.addonId).monthlyFee, subscribedAt: s.subscribedAt, status: s.status }));
  return {
    customer: { ...c, username: acct.username, registeredAt: acct.registeredAt, passwordUpdatedAt: acct.passwordUpdatedAt, autoPayEnabled: acct.autoPayEnabled },
    notify: db.notifyOf(id) || null,
    balance: db.byBalanceOf(id),
    usage: db.usageRecords.find((u) => u.customerId === id) || null,
    stats: userStats(id),
    recent: userRecent(id),
    addresses: of('userAddresses'), plans, addons,
    orders: db.orders.filter((o) => o.customerId === id && !o.archived).map((o) => ({ orderNo: o.orderNo, bizType: o.bizType, addrLabel: o.addrLabel, stage: o.stage, statusLabel: db.STATUS_LABEL[db.statusOfStage(o.stage)] })),
    faults: db.repairTickets.filter((t) => t.customerId === id).map((t) => ({ ticketNo: t.ticketNo, faultTypeLabel: t.faultTypeLabel, reportedAt: t.reportedAt, status: t.status })),
    bills: db.bills.filter((b) => b.customerId === id), payments: db.payments.filter((p) => p.customerId === id),
    invoices: of('userInvoices'), messages: of('userMessages'), coupons: of('coupons'),
    complaints: of('userComplaints'), verifyRecords: of('userVerifyRecords'),
  };
}

module.exports = {
  // —— 用户列表/详情(产品化入口) ——
  'GET /users': ({ query }) => {
    let items = db.customers.map(userRow);
    if (query && query.keyword) {
      const kw = query.keyword;
      items = items.filter((r) => [r.name, r.phoneMasked, r.username, r.productName].some((f) => String(f).indexOf(kw) >= 0));
    }
    return { total: items.length, items };
  },
  'GET /users/{customerId}': ({ params }) => userDetail(Number(params.customerId) || params.customerId),
  'PUT /users/{customerId}/account': ({ params, body }) => {
    const a = db.userAccounts.find((x) => x.customerId === (Number(params.customerId) || params.customerId));
    if (!a) return null;
    if (typeof body.autoPayEnabled === 'boolean') a.autoPayEnabled = body.autoPayEnabled;
    return { updated: true, account: a };
  },

  // —— 用户账户(登录/密码/自动缴费) ——
  'GET /user-accounts': () => ({ items: db.userAccounts.map((a) => ({
    customerId: a.customerId, customerName: cname(a.customerId), username: a.username,
    registeredAt: a.registeredAt, passwordUpdatedAt: a.passwordUpdatedAt,
    autoPay: a.autoPayEnabled ? '已开通' : '未开通', status: a.statusLabel,
  })) }),

  // —— 地址簿 ——
  'GET /user-addresses': () => ({ items: db.userAddresses.map((a) => ({
    addressId: a.addressId, customerName: cname(a.customerId), addrCode: a.addrCode,
    address: a.community + ' · ' + a.building + ' · ' + a.door, contact: a.contact,
    phoneMasked: a.phoneMasked, isDefault: a.isDefault ? '默认' : '—',
  })) }),
  'POST /user-addresses': ok,

  // —— 套餐订购关系 ——
  'GET /user-plans': () => ({ items: db.userPlans.map((p) => {
    const prod = db.byProduct(p.productId) || {};
    return { planId: p.planId, customerName: cname(p.customerId), productName: prod.name,
      monthlyFee: '¥' + p.monthlyFee, contractEnd: p.contractEnd, status: p.statusLabel };
  }) }),
  'POST /user-plans': ok,

  // —— 增值服务目录与订购 ——
  'GET /addons': () => ({ items: db.addonCatalog.map((a) => ({
    addonId: a.addonId, name: a.name, monthlyFee: '¥' + a.monthlyFee, description: a.description,
    subscribedCount: db.addonSubscriptions.filter((s) => s.addonId === a.addonId && s.status === 'ACTIVE').length,
    status: a.active ? '在售' : '已下架',
  })) }),
  'POST /addons': ({ body }) => {
    const row = { addonId: body.addonId || 'A-NEW', name: body.name || '新增值服务', monthlyFee: body.monthlyFee || 0, description: body.description || '', active: true };
    db.tables.addonCatalog.insert(row);
    return { created: true, row };
  },
  'PUT /addons/{addonId}/toggle': ({ params }) => {
    const a = db.addonCatalog.find((x) => x.addonId === params.addonId);
    if (a) a.active = !a.active;
    return a ? { toggled: true, active: a.active } : null;
  },
  'GET /addon-subscriptions': () => ({ items: db.addonSubscriptions.map((s) => ({
    subId: s.subId, customerName: cname(s.customerId), addonName: db.byAddon(s.addonId).name,
    monthlyFee: '¥' + db.byAddon(s.addonId).monthlyFee, subscribedAt: s.subscribedAt, status: s.status === 'ACTIVE' ? '生效中' : '已退订',
  })) }),
  'POST /addon-subscriptions': ({ body }) => {
    const row = { subId: 'AS-' + String(db.addonSubscriptions.length + 1).padStart(3, '0'), customerId: body.customerId || 1, addonId: body.addonId, subscribedAt: '2025-08-17', status: body.action === 'unsubscribe' ? 'CANCELLED' : 'ACTIVE' };
    db.tables.addonSubscriptions.insert(row);
    return { created: true, row };
  },

  // —— 通知订阅 ——
  'GET /user-notify-settings': () => ({ items: db.notifyPrefs.map((n) => ({
    customerName: cname(n.customerId),
    business: ['bill', 'suspendResume', 'faultNotice', 'installProgress'].filter((k) => n.business[k]).map((k) => ({ bill: '账单', suspendResume: '停复机', faultNotice: '故障', installProgress: '装维进度' }[k])).join('/'),
    marketing: ['promo', 'planRecommend'].filter((k) => n.marketing[k]).map((k) => ({ promo: '优惠活动', planRecommend: '套餐推荐' }[k])).join('/') || '—',
    channels: ['inApp', 'sms', 'push'].filter((k) => n.channels[k]).map((k) => ({ inApp: '站内', sms: '短信', push: '推送' }[k])).join('/'),
  })) }),
  'PUT /user-notify-settings/{customerId}': ok,

  // —— FAQ 知识库(active 控制用户端可见) ——
  'GET /user-faqs': () => ({ items: db.userFaqs.map((f) => ({
    faqId: f.faqId, question: f.question, answer: f.answer, sort: f.sort,
    status: f.active ? '启用' : '停用', statusClass: f.active ? 'tag-green' : 'tag-gray',
  })) }),
  'POST /user-faqs': ({ body }) => {
    const row = { faqId: 'faq-' + (db.userFaqs.length + 1), question: body.question || '新问题', answer: body.answer || '', sort: db.userFaqs.length + 1, active: true };
    db.tables.userFaqs.insert(row);
    return { created: true, row };
  },
  'PUT /user-faqs/{faqId}/toggle': ({ params }) => {
    const f = db.userFaqs.find((x) => x.faqId === params.faqId);
    if (f) f.active = !f.active;
    return f ? { toggled: true, active: f.active } : null;
  },

  // —— 用户消息(营销/提醒下发) ——
  'GET /user-messages': ({ query }) => {
    let items = db.userMessages.map((m) => ({ ...m, customerName: cname(m.customerId) }));
    if (query && query.keyword) items = items.filter((m) => (m.title + m.content).indexOf(query.keyword) >= 0);
    return { items };
  },
  'POST /user-messages': ({ body }) => {
    const row = { messageId: 'M-' + String(db.userMessages.length + 1).padStart(3, '0'), customerId: body.customerId || 1,
      category: body.category || 'promo', title: body.title || '新消息', content: body.content || '',
      tag: '公告', tagLevel: 'info', createdAt: '2025-08-17', read: false };
    db.tables.userMessages.insert(row);
    return { created: true, row };
  },
  'PUT /user-messages/read-all': ok,

  // —— 优惠券 ——
  'GET /coupons': () => ({ items: db.coupons.map((c) => ({
    couponId: c.couponId, customerName: cname(c.customerId), title: c.title,
    amount: '¥' + c.amount, threshold: '满 ¥' + c.threshold, expireAt: c.expireAt,
    status: c.active ? c.statusLabel : '已停发', statusClass: c.active ? 'tag-green' : 'tag-gray',
  })) }),
  'POST /coupons': ({ body }) => {
    const row = { couponId: 'CPN-' + String(db.coupons.length + 1).padStart(3, '0'), customerId: body.customerId || 1,
      amount: body.amount || 10, threshold: body.threshold || 100, title: body.title || '新优惠券',
      expireAt: body.expireAt || '2025-09-30', status: 'available', statusLabel: '可用', active: true };
    db.tables.coupons.insert(row);
    return { created: true, row };
  },
  'PUT /coupons/{couponId}/disable': ({ params }) => {
    const c = db.coupons.find((x) => x.couponId === params.couponId);
    if (c) { c.active = false; c.status = 'disabled'; c.statusLabel = '已停发'; }
    return c ? { disabled: true } : null;
  },
  'GET /invite-config': () => ({ items: db.inviteConfig }),

  // —— 流量使用 ——
  'GET /user-usages': () => ({ items: db.usageRecords.map((u) => ({
    customerName: cname(u.customerId), period: u.period, used: u.used + u.unit, quota: u.quota + u.unit,
    percent: Math.round(u.used / u.quota * 100) + '%', dailyAvg: u.dailyAvg + u.unit + '/天',
    onlineDuration: u.onlineDuration, lastOnlineAt: u.lastOnlineAt,
  })) }),

  // —— 自助排障指南 ——
  'GET /diy-guides': () => ({ items: db.diyGuides.map((g) => ({
    guideId: g.guideId, title: g.title, desc: g.desc, stepCount: g.steps.length,
    status: g.active ? '启用' : '停用', statusClass: g.active ? 'tag-green' : 'tag-gray',
  })) }),
  'PUT /diy-guides/{guideId}/toggle': ({ params }) => {
    const g = db.diyGuides.find((x) => x.guideId === params.guideId);
    if (g) g.active = !g.active;
    return g ? { toggled: true, active: g.active } : null;
  },

  // —— 协议版本 ——
  'GET /agreements': () => ({ items: db.agreements.map((a) => ({
    agreementId: a.agreementId, typeLabel: a.typeLabel, version: a.version,
    effectiveAt: a.effectiveAt, clauseCount: a.clauses.length,
    status: a.active ? '现行' : '已废止', statusClass: a.active ? 'tag-green' : 'tag-gray',
  })) }),
  'PUT /agreements/{agreementId}': ok,

  // —— 余额与充值面额 ——
  'GET /user-balances': () => ({ items: db.balances.map((b) => ({
    customerName: cname(b.customerId), balance: '¥' + b.balance, lowThreshold: '¥' + b.lowThreshold,
    updatedAt: b.updatedAt, status: b.balance < b.lowThreshold ? '低于预警线' : '正常',
    statusClass: b.balance < b.lowThreshold ? 'tag-orange' : 'tag-green',
  })) }),
  'POST /user-balances/{customerId}/adjust': ({ params, body }) => {
    const b = db.balances.find((x) => x.customerId === Number(params.customerId) || x.customerId === params.customerId);
    if (!b) return null;
    b.balance = Math.max(0, b.balance + (Number(body.delta) || 0));
    b.updatedAt = '2025-08-17 11:00';
    return { adjusted: true, balance: b.balance };
  },
  'GET /topup-denominations': () => ({ items: db.topupDenominations }),
  'PUT /topup-denominations/{denomId}': ok,

  // —— 电子发票(仅已缴账期) ——
  'GET /user-invoices': () => ({ items: db.userInvoices.map((r) => ({
    invoiceId: r.invoiceId, customerName: cname(r.customerId), period: r.period, billNo: r.billNo,
    amount: '¥' + r.amount, title: r.titleType + ' · ' + r.title, issuedAt: r.issuedAt, pdfUrl: r.pdfUrl,
  })) }),
  'POST /user-invoices': ({ body }) => {
    const bill = db.bills.find((b) => b.billNo === body.billNo);
    if (!bill || bill.status !== 'PAID') return { code: 1, message: '仅已缴账期可开票' };
    const row = { invoiceId: 'INV-' + bill.period + '-00' + (db.userInvoices.length + 1), customerId: bill.customerId,
      period: bill.period, billNo: bill.billNo, amount: bill.amount, titleType: '个人',
      title: cname(bill.customerId), taxNo: null, issuedAt: '2025-08-17', pdfUrl: '#' };
    db.tables.userInvoices.insert(row);
    return { created: true, row };
  },

  // —— 投诉(非报障类) ——
  'GET /user-complaints': () => ({ items: db.userComplaints.map((c) => ({
    complaintId: c.complaintId, customerName: cname(c.customerId), typeLabel: c.typeLabel,
    relOrderNo: c.relOrderNo || '—', status: c.statusLabel, createdAt: c.createdAt, handler: c.handler,
  })) }),
  'POST /user-complaints/{complaintId}/close': ({ params }) => {
    const c = db.userComplaints.find((x) => x.complaintId === params.complaintId);
    if (c) { c.status = 'RESOLVED'; c.statusLabel = '已解决'; }
    return c ? { closed: true } : null;
  },

  // —— 实名核验记录 ——
  'GET /user-verify-records': () => ({ items: db.userVerifyRecords.map((r) => ({
    customerName: cname(r.customerId), method: r.method, time: r.time,
    result: r.result === 'PASS' ? '通过' : '未通过',
  })) }),

  // —— 产品卖点/账单明细项配置 ——
  'GET /product-specs': () => ({ items: db.productSpecs.map((s) => ({
    productId: s.productId, productName: (db.byProduct(s.productId) || {}).name, specs: s.specs,
  })) }),
  'PUT /product-specs/{productId}': ok,
  'GET /user-bill-items': ({ query }) => {
    let items = db.userBillItems.map((i) => ({ ...i, customerName: cname((db.bills.find((b) => b.billNo === i.billNo) || {}).customerId) }));
    if (query && query.billNo) items = items.filter((i) => i.billNo === query.billNo);
    return { items };
  },
};
