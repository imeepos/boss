// 假数据 —— 师傅管理(契约: admin.yaml worker 分组;实体: db.js 师傅域表)。
// 师傅端用到的全部数据在此可视/可管: 档案与接单设置/绩效提成/评价/消息/公告/FAQ/
// 物料工具/旧件回收/设备健康/抢单池/调度会话/排期。底层实体另可经 /crud/{table} 增删改。
'use strict';

const db = require('../../db.js');

const ok = { code: 0, message: 'success' };

function pick(items, query) {
  let out = items;
  if (query && query.keyword) {
    const kw = String(query.keyword);
    out = out.filter((it) => Object.keys(it).some((k) => String(it[k] == null ? '' : it[k]).indexOf(kw) >= 0));
  }
  if (query && query.workerId) out = out.filter((it) => String(it.workerId) === String(query.workerId));
  return out;
}

// 师傅列表: 主档(db.workers) × 档案扩展(db.workerProfiles) × 在途工单数
function workerRows() {
  return db.workers.map((w) => {
    const p = db.byWorkerProfile(w.workerId);
    const doing = db.orders.filter((o) => o.workerId === w.workerId && o.stage >= 8 && o.stage < 12).length
      + db.repairTickets.filter((t) => t.workerId === w.workerId && t.status === 'PROCESSING').length;
    return {
      workerId: w.workerId, name: w.name, staffNo: w.staffNo, phoneMasked: w.phoneMasked, groupName: w.groupName,
      online: p.online, onlineLabel: p.online ? '在线' : '离线', onlineClass: p.online ? 'tag-green' : 'tag-gray',
      serveYears: p.serveYears, doingCount: doing,
      monthFinished: p.month.finished, onTimeRate: p.month.onTimeRate, score: p.month.score,
      radiusKm: p.radiusKm, acceptTypes: (p.acceptTypes || []).join(' / '),
    };
  });
}

function commissionRows() {
  const totals = {};
  return db.workerCommissions.map((c) => {
    const w = db.byWorker(c.workerId);
    totals[c.workerId] = (totals[c.workerId] || 0) + c.amount;
    return { period: c.period, workerId: c.workerId, workerName: w.name, groupName: w.groupName, name: c.name, formula: c.formula, amount: c.amount, _workerId: c.workerId };
  }).map((r, _i, arr) => ({ ...r, totalAmount: arr.filter((x) => x.workerId === r.workerId).reduce((s, x) => s + x.amount, 0) }));
}

module.exports = {
  // 师傅管理
  'GET /workers': ({ query }) => ({ stats: { total: db.workers.length, online: db.workerProfiles.filter((p) => p.online).length }, items: pick(workerRows(), query) }),
  'GET /workers/{workerId}': ({ params }) => workerRows().find((w) => String(w.workerId) === params.workerId) || null,
  'PUT /workers/{workerId}/settings': ({ params, body }) => {
    const p = db.workerProfiles.find((x) => String(x.workerId) === params.workerId);
    if (!p) return null;
    if (body && body.online !== undefined) p.online = !!body.online;
    if (body && body.radiusKm) p.radiusKm = body.radiusKm;
    if (body && Array.isArray(body.acceptTypes)) p.acceptTypes = body.acceptTypes;
    return { updated: true, settings: { online: p.online, radiusKm: p.radiusKm, acceptTypes: p.acceptTypes } };
  },
  // 绩效与提成
  'GET /worker-performances': ({ query }) => {
    const rank = db.workerProfiles.slice().sort((a, b) => b.month.finished - a.month.finished)
      .map((p, i) => ({ rank: i + 1, workerId: p.workerId, workerName: db.byWorker(p.workerId).name, groupName: db.byWorker(p.workerId).groupName, period: p.month.period, finished: p.month.finished, onTimeRate: p.month.onTimeRate, score: p.month.score }));
    return { items: pick(rank, query) };
  },
  'GET /worker-commissions': ({ query }) => ({ items: pick(commissionRows(), query) }),
  // 客户评价(差评复核)
  'GET /worker-feedbacks': ({ query }) => ({
    items: pick(db.workerFeedbacks.map((f) => ({
      feedbackId: f.feedbackId, workerId: f.workerId, workerName: db.byWorker(f.workerId).name, ticketNo: f.ticketNo,
      customerName: f.customerName, score: f.score, comment: f.comment,
      needReview: f.needReview, needReviewLabel: f.needReview ? '待复核' : '—', needReviewClass: f.needReview ? 'tag-orange' : 'tag-gray',
    })), query),
  }),
  'POST /worker-feedbacks/{feedbackId}/review': () => ok,
  // 消息下发(admin → 师傅)
  'GET /worker-messages': ({ query }) => ({
    items: pick(db.workerMessages.map((m) => ({ ...m, workerName: db.byWorker(m.workerId).name, readLabel: m.read ? '已读' : '未读' })), query),
  }),
  'POST /worker-messages': ({ body }) => {
    const b = body || {};
    const row = { messageId: 'wm' + (db.workerMessages.length + 1), workerId: b.workerId || 1024, level: b.level || 'info', title: b.title || '通知', content: b.content || '', sentAt: '08-17 11:00', read: false };
    db.workerMessages.push(row);
    return { created: true, row };
  },
  // 公告 / FAQ
  'GET /notices': () => ({ items: db.workerNotices.map((n) => ({ ...n, activeLabel: n.active ? '发布中' : '已下架', activeClass: n.active ? 'tag-green' : 'tag-gray' })) }),
  'POST /notices': ({ body }) => {
    const b = body || {};
    const row = { noticeId: 'n' + (db.workerNotices.length + 1), title: b.title || '新公告', category: b.category || '功能公告', publishedAt: '08-17', active: true };
    db.workerNotices.push(row);
    return { created: true, row };
  },
  'PUT /notices/{noticeId}/toggle': ({ params }) => {
    const n = db.workerNotices.find((x) => x.noticeId === params.noticeId);
    if (!n) return null;
    n.active = !n.active;
    return { updated: true, active: n.active };
  },
  'GET /faqs': () => ({ items: db.workerFaqs.map((f) => ({ ...f, activeLabel: f.active ? '启用' : '停用', activeClass: f.active ? 'tag-green' : 'tag-gray' })) }),
  'POST /faqs': ({ body }) => {
    const b = body || {};
    const row = { faqId: 'f' + (db.workerFaqs.length + 1), title: b.title || '新条目', summary: b.summary || '', active: true };
    db.workerFaqs.push(row);
    return { created: true, row };
  },
  'PUT /faqs/{faqId}/toggle': ({ params }) => {
    const f = db.workerFaqs.find((x) => x.faqId === params.faqId);
    if (!f) return null;
    f.active = !f.active;
    return { updated: true, active: f.active };
  },
  // 物料 / 工具 / 旧件回收
  'GET /worker-materials': ({ query }) => ({ items: pick(db.workerMaterials.map((m) => ({ ...m, workerName: db.byWorker(m.workerId).name, outBoundLabel: m.outBound ? '已出库' : '在库' })), query) }),
  'GET /worker-tools': ({ query }) => ({ items: pick(db.workerTools.map((t) => ({ ...t, workerName: db.byWorker(t.workerId).name, borrowedLabel: t.borrowed ? '借用中' : '已归还' })), query) }),
  'GET /asset-returns': ({ query }) => ({
    stats: db.returnStats,
    items: pick(db.assetReturns.map((r) => ({
      ...r, workerName: db.byWorker(r.workerId).name, assetNo: (db.assets.find((a) => a.epc === r.epc) || {}).assetNo || '—',
      statusLabel: r.status === 'PENDING' ? '待返库' : '已返库', statusClass: r.status === 'PENDING' ? 'tag-orange' : 'tag-green',
    })), query),
  }),
  'POST /asset-returns/{returnId}/confirm': ({ params }) => {
    const r = db.assetReturns.find((x) => x.returnId === params.returnId);
    if (!r) return null;
    r.status = 'RETURNED';
    return { updated: true, status: r.status };
  },
  // 设备健康 / 抢单池 / 调度会话 / 排期
  'GET /device-maintenances': () => ({ items: db.deviceMaintenances.map((d) => ({ ...d, priorityClass: { MUST_REPLACE: 'tag-red', SUGGEST: 'tag-orange', WATCH: 'tag-blue' }[d.priority] })) }),
  'GET /hall-items': () => ({
    items: db.repairTickets.filter((t) => !t.workerId && t.status === 'PROCESSING')
      .map((t) => ({ ticketNo: t.ticketNo, type: 'REPAIR', typeLabel: '抢修', address: t.addrLabel, source: 'repairTickets' }))
      .concat(db.hallExtras.map((x) => ({ ticketNo: x.ticketNo, type: x.type, typeLabel: x.typeLabel, address: x.address, source: 'hallExtras' }))),
  }),
  'GET /service-messages': ({ query }) => ({ items: pick(db.serviceMessages.map((m) => ({ ...m, workerName: db.byWorker(m.workerId).name })), query) }),
  'POST /service-messages': ({ body }) => {
    const b = body || {};
    const row = { msgId: 'sm' + (db.serviceMessages.length + 1), workerId: b.workerId || 1024, from: 'dispatcher', content: b.content || '' };
    db.serviceMessages.push(row);
    return { created: true, row };
  },
  'GET /worker-schedules': ({ query }) => ({ items: pick(db.workerSchedules.map((s) => ({ ...s, workerName: db.byWorker(s.workerId).name, busyDaysLabel: s.busyDays.join('、') })), query) }),
};
