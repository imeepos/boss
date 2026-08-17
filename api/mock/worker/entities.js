// entities.js —— 师傅域实体表(单一事实源,由 db.js 聚合导出)
// 师傅端用到的全部专属数据在此建表: 档案/绩效提成/评价/消息/公告/FAQ/
// 物料工具/旧件回收/设备健康/抢单池附加任务/调度会话/排期。
// admin 后经 api/mock/admin/data/worker.js 管理这些表,师傅端视图一律派生,不得手抄。
'use strict';

// —— 师傅档案扩展(workerProfiles): 主档见 db.workers,此处存接单设置与月绩效 ——
const workerProfiles = [
  { workerId: 1024, online: true, serveYears: 3, radiusKm: 5, acceptTypes: ['新装宽带', '宽带变更', '拆机'], month: { period: '2025-08', finished: 46, onTimeRate: 98, score: 4.9 } },
  { workerId: 1036, online: true, serveYears: 5, radiusKm: 6, acceptTypes: ['抢修'], month: { period: '2025-08', finished: 38, onTimeRate: 96, score: 4.8 } },
  { workerId: 1037, online: true, serveYears: 2, radiusKm: 4, acceptTypes: ['新装宽带'], month: { period: '2025-08', finished: 41, onTimeRate: 95, score: 4.7 } },
  { workerId: 1038, online: false, serveYears: 1, radiusKm: 4, acceptTypes: ['新装宽带', '宽带变更'], month: { period: '2025-08', finished: 29, onTimeRate: 92, score: 4.6 } },
  { workerId: 1039, online: true, serveYears: 4, radiusKm: 5, acceptTypes: ['抢修'], month: { period: '2025-08', finished: 35, onTimeRate: 97, score: 4.8 } },
  { workerId: 1040, online: true, serveYears: 2, radiusKm: 5, acceptTypes: ['新装宽带', '拆机'], month: { period: '2025-08', finished: 33, onTimeRate: 94, score: 4.7 } },
];

// —— 提成明细(workerCommissions): 月度结算口径,period 与 workerProfiles.month 对齐 ——
const workerCommissions = [
  { workerId: 1024, period: '2025-08', name: '装机提成', formula: '46 × ¥30', amount: 1380 },
  { workerId: 1024, period: '2025-08', name: '抢修提成', formula: '12 × ¥20', amount: 240 },
  { workerId: 1024, period: '2025-08', name: '满意度奖励', formula: '', amount: 120 },
];

// —— 客户评价(workerFeedbacks): 差评转复核(needReview) ——
const workerFeedbacks = [
  { feedbackId: 'fb1', workerId: 1024, ticketNo: 'ORD-20250817-000', customerName: '王先生', score: 5.0, comment: '服务态度好', needReview: false },
  { feedbackId: 'fb2', workerId: 1024, ticketNo: 'TKT-20250730-005', customerName: '李女士', score: 2.0, comment: '已转复核', needReview: true },
];

// —— 师傅消息(workerMessages): 调度/系统下发,level err/warn/ok ——
const workerMessages = [
  { messageId: 'wm1', workerId: 1024, level: 'err', title: '台风应急', content: '台风后批量复测任务已下发，请核对受影响客户清单', sentAt: '08-16 18:00', read: false },
  { messageId: 'wm2', workerId: 1024, level: 'warn', title: '超时预警', content: 'TKT-20250817-012 抢修单 SLA 剩余不足 1 小时，请尽快到场处理', sentAt: '08-17 10:30', read: false },
  { messageId: 'wm3', workerId: 1024, level: 'err', title: '改派通知', content: '新单 TKT-20250817-005 抢修已分派给您', sentAt: '08-17 09:45', read: false },
  { messageId: 'wm4', workerId: 1024, level: 'ok', title: '配置下发', content: '全部预下发成功', sentAt: '08-17 09:18', read: true },
  { messageId: 'wm5', workerId: 1024, level: 'warn', title: '标签电量', content: 'EPC-0023 电量低，请携备用', sentAt: '08-17 08:00', read: true },
];

// —— 公告(workerNotices): admin 发布/下架,active 控制师傅端可见 ——
const workerNotices = [
  { noticeId: 'n1', title: '台风季弱电井防水作业提示', category: '安全作业提醒', publishedAt: '08-16', active: true },
  { noticeId: 'n2', title: '本周 GPU 千兆套餐物料配发说明', category: '物料公告', publishedAt: '08-15', active: true },
  { noticeId: 'n3', title: '扫码绑定弱网离线功能上线', category: '功能公告', publishedAt: '08-14', active: true },
];

// —— FAQ(workerFaqs): 知识库,admin 维护 ——
const workerFaqs = [
  { faqId: 'f1', title: '光猫红灯/无法注册', summary: 'LOID 认证失败排查', active: true },
  { faqId: 'f2', title: '光功率偏低', summary: '分光比与接头损耗排查', active: true },
  { faqId: 'f3', title: '测速不达标', summary: '线路/终端/WiFi 分段定位', active: true },
  { faqId: 'f4', title: '扫码绑定四码不一致', summary: '换机/重绑处理', active: true },
];

// —— 随车物料(workerMaterials) / 工具(workerTools) ——
const workerMaterials = [
  { itemId: 'm1', workerId: 1024, name: '光猫', qty: 2, spec: 'GPON 千兆 · 含标签', outBound: false },
  { itemId: 'm2', workerId: 1024, name: '机顶盒', qty: 1, spec: 'IPTV 4K', outBound: false },
  { itemId: 'm3', workerId: 1024, name: '光纤跳线', qty: 5, spec: 'SC/APC 2m', outBound: false },
];
const workerTools = [
  { toolId: 't1', workerId: 1024, name: '光功率计', borrowed: false },
  { toolId: 't2', workerId: 1024, name: '光纤熔接机', borrowed: true },
];

// —— 旧件待回收(assetReturns): epc 挂 db.assets,reason 区分返修/拆机回收 ——
const assetReturns = [
  { returnId: 'ar1', workerId: 1024, epc: 'EPC-0002', reason: '故障 · 返修（换件）', status: 'PENDING' },
  { returnId: 'ar2', workerId: 1024, epc: 'EPC-0110', reason: '拆机回收（ORD-20250817-009）', status: 'PENDING' },
];
const returnStats = { repairCount: 3, dismantleCount: 12 };

// —— 设备健康观察(deviceMaintenances): 师傅端"主动运维"列表 ——
const deviceMaintenances = [
  { deviceNo: 'EPC-0023', deviceType: '光猫', healthScore: 31, faultCount: 5, ageYears: 4, reason: '', priority: 'MUST_REPLACE', priorityLabel: '强替换' },
  { deviceNo: 'SPL-03-07', deviceType: '分光器', healthScore: 45, faultCount: 0, ageYears: 3, reason: '信号衰减', priority: 'SUGGEST', priorityLabel: '建议替换' },
  { deviceNo: 'OLT-01', deviceType: 'PON 9口', healthScore: 58, faultCount: 0, ageYears: 0, reason: '丢包率偏高', priority: 'WATCH', priorityLabel: '观察' },
];

// —— 抢单池附加任务(hallExtras): 应急/预告单,常规抢单由 db.repairTickets 未指派行派生 ——
const hallExtras = [
  { ticketNo: 'EMG-20250817-001', type: 'EMERGENCY', typeLabel: '台风批量复测', address: '望京X片区 · 受影响 23 户', distanceKm: 0.8 },
  { ticketNo: 'ORD-20250817-007', type: 'INSTALL', typeLabel: '新装', address: '望京X · 15栋', distanceKm: 2.8 },
];

// —— 调度会话(serviceMessages): from=dispatcher/worker ——
const serviceMessages = [
  { msgId: 'sm1', workerId: 1024, from: 'dispatcher', content: '已为您接通调度中心，此单可支持改派/咨询。' },
  { msgId: 'sm2', workerId: 1024, from: 'worker', content: 'ORD-20250817-001 关联工单，可快捷转单。' },
];

// —— 月排期(workerSchedules): busyDays 供师傅端日历 ——
const workerSchedules = [
  { workerId: 1024, month: '2025-08', busyDays: [5, 6, 10, 17, 30] },
];

module.exports = {
  workerProfiles, workerCommissions, workerFeedbacks, workerMessages, workerNotices, workerFaqs,
  workerMaterials, workerTools, assetReturns, returnStats, deviceMaintenances, hallExtras,
  serviceMessages, workerSchedules,
};
