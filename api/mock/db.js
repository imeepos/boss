// db.js —— 三端共享的关系型事实库(单一事实源)
// 所有跨端实体在此建立外键关系,用户端/师傅端/管理后台的视图一律由本库派生,
// 禁止在各端再手抄实体数据。口径见 api/DATA-ALIGNMENT.md,枚举见 docs/contract/terms.md。
// 关系链: customer → order → (port / tagEpc / loid / worker) → quad → bill → payment
'use strict';

// —— 客户(customers) ——
const customers = [
  { customerId: 1, name: '王先生', phoneMasked: '138****1234', idType: '身份证', idNoMasked: '110***********1234', realNameStatus: 'VERIFIED', realNameLabel: '已实名', serviceStatus: 'ACTIVE', serviceLabel: '在网', verifyAt: '2023-05-11' },
  { customerId: 2, name: '吴女士', phoneMasked: '139****5678', idType: '身份证', idNoMasked: '110***********5678', realNameStatus: 'VERIFIED', realNameLabel: '已实名', serviceStatus: 'ARREARS', serviceLabel: '欠费', verifyAt: '2025-07-28' },
  { customerId: 3, name: '孙先生', phoneMasked: '137****9012', idType: '无', idNoMasked: '—', realNameStatus: 'PENDING', realNameLabel: '待补登', serviceStatus: 'ACTIVE', serviceLabel: '在网', verifyAt: null },
  { customerId: 4, name: '赵女士', phoneMasked: '136****3456', idType: '身份证', idNoMasked: '110***********3456', realNameStatus: 'VERIFIED', realNameLabel: '已实名', serviceStatus: 'ACTIVE', serviceLabel: '在网', verifyAt: '2025-06-12' },
  { customerId: 5, name: '郑先生', phoneMasked: '135****7890', idType: '身份证', idNoMasked: '110***********7890', realNameStatus: 'VERIFIED', realNameLabel: '已实名', serviceStatus: 'ACTIVE', serviceLabel: '在网', verifyAt: '2025-05-20' },
  { customerId: 6, name: '周女士', phoneMasked: '133****2468', idType: '身份证', idNoMasked: '110***********2468', realNameStatus: 'PENDING', realNameLabel: '待补登', serviceStatus: 'ACTIVE', serviceLabel: '在网', verifyAt: null },
  { customerId: 7, name: '刘女士', phoneMasked: '132****1357', idType: '身份证', idNoMasked: '110***********1357', realNameStatus: 'VERIFIED', realNameLabel: '已实名', serviceStatus: 'SUSPENDED', serviceLabel: '停机', verifyAt: '2025-04-02' },
  { customerId: 8, name: '陈先生', phoneMasked: '138****7788', idType: '身份证', idNoMasked: '110***********7788', realNameStatus: 'VERIFIED', realNameLabel: '已实名', serviceStatus: 'ACTIVE', serviceLabel: '在网', verifyAt: '2025-03-15' },
];

// —— 产品资费(products) ——
const products = [
  { productId: 'P-300', category: 'broadband', name: '300M 畅享宽带', bandwidth: '300M', monthlyFee: 99, contractMonths: 12, description: '下行 300M · 含光猫 · 合约 12 个月', featured: false, status: 'PUBLISHED' },
  { productId: 'P-500', category: 'broadband', name: '500M 畅享宽带', bandwidth: '500M', monthlyFee: 129, contractMonths: 12, description: '下行 500M · 含光猫 + 路由器 · 合约 12 个月', featured: false, status: 'PUBLISHED' },
  { productId: 'P-1000', category: 'broadband', name: '1000M 极速宽带', bandwidth: '1000M', monthlyFee: 199, contractMonths: 24, description: '下行 1000M · 含光猫 + 路由器 + IPTV · 合约 24 个月', featured: true, status: 'PUBLISHED' },
  { productId: 'P-BIZ-100', category: 'biz', name: '政企专线 100M', bandwidth: '100M', monthlyFee: 500, contractMonths: 12, description: '上下行对等 · SLA 保障', featured: false, status: 'DRAFT' },
];

// —— 装维师傅(workers) ——
const workers = [
  { workerId: 1024, name: '张师傅', phoneMasked: '138****8899', groupName: '装机一组', staffNo: 'WK-1024' },
  { workerId: 1036, name: '王师傅', phoneMasked: '137****3366', groupName: '抢修组', staffNo: 'WK-1036' },
];

// —— 端口(ports): portNo=物理端口, quadCode=四码端口码, orderId 反向占用 ——
const ports = [
  { portNo: 'P-001-01', quadCode: 'P-SPL03-01', parentName: 'OLT-01 · SPL-03', addrCode: 'A-2-902', status: 'USED', orderId: 'ORD-20250817-009' },
  { portNo: 'P-001-02', quadCode: 'P-SPL03-07', parentName: 'OLT-01 · SPL-03', addrCode: 'A-3-501', status: 'RESERVED', orderId: 'ORD-20250817-001' },
  { portNo: 'P-001-03', quadCode: 'P-SPL03-03', parentName: 'OLT-01 · SPL-03', addrCode: 'A-1-101', status: 'USED', orderId: 'ORD-20250816-018' },
  { portNo: 'P-001-04', quadCode: 'P-SPL03-09', parentName: 'OLT-01 · SPL-03', addrCode: 'A-12-906', status: 'RESERVED', orderId: 'ORD-20250817-003' },
  { portNo: 'P-002-09', quadCode: 'P-SPL01-09', parentName: 'OLT-01 · SPL-01', addrCode: 'A-5-302', status: 'RESERVED', orderId: 'ORD-20250817-002' },
  { portNo: 'P-003-02', quadCode: 'P-SPL04-02', parentName: 'OLT-02 · SPL-04', addrCode: 'A-6-701', status: 'USED', orderId: 'ORD-20250816-011' },
];

// —— 资产/标签(assets): epc 即电子标签码,tagNo 冗余展示 ——
const assets = [
  { assetNo: 'A-20250001', tagNo: 'TAG-0001', epc: 'EPC-0001', type: '光猫', batchNo: 'RK-202507-01', location: '望京·X小区·3栋501', lifecycle: '在用', status: 'DEPLOYED', customerId: 1 },
  { assetNo: 'A-20250002', tagNo: 'TAG-0002', epc: 'EPC-0002', type: '分光器', batchNo: 'RK-202506-03', location: '望京·X小区·弱电井2', lifecycle: '维修', status: 'MAINTENANCE', customerId: null },
  { assetNo: 'A-20250003', tagNo: 'TAG-0003', epc: 'EPC-0003', type: '光猫', batchNo: 'RK-202508-02', location: '望京·Y小区·1栋101', lifecycle: '在用', status: 'DEPLOYED', customerId: 3 },
  { assetNo: 'A-20250013', tagNo: 'TAG-0013', epc: 'EPC-0013', type: '光猫', batchNo: 'RK-202508-02', location: '仓库', lifecycle: '在库', status: 'IN_STOCK', customerId: null },
];

// —— 认证账号(loids): 客户上网认证账号,quad 用户码取值 ——
const loids = [
  { loid: 'LOID-88A1', customerId: 1, bandwidth: '1000M', qos: 'QoS-VIP', status: 'ACTIVE', statusLabel: '在服' },
  { loid: 'LOID-88A3', customerId: 3, bandwidth: '300M', qos: 'QoS-STD', status: 'ACTIVE', statusLabel: '在服' },
  { loid: 'LOID-88A4', customerId: 6, bandwidth: '300M', qos: 'QoS-STD', status: 'SUSPENDED', statusLabel: '停机' },
  { loid: 'LOID-88A5', customerId: 2, bandwidth: '1000M', qos: 'QoS-STD', status: 'SUSPENDED', statusLabel: '停机' },
  { loid: 'LOID-88A7', customerId: 8, bandwidth: '300M', qos: 'QoS-STD', status: 'ACTIVE', statusLabel: '在服' },
  { loid: 'LOID-88A9', customerId: 5, bandwidth: '300M', qos: 'QoS-STD', status: 'PENDING', statusLabel: '待激活' },
];

// —— 12 环节(terms.md 第 1 节,禁止增删改序) ——
const STAGE_NAMES = ['用户下单', '资源核查', '端口预占', '合同收费', '标签预绑定', '创建认证账号', '预下发配置', '派单', '扫码绑定', '激活用户', '激活回调', '更新 GIS 地图'];
const STAGE_TIMES_001 = ['08-17 09:02', '08-17 09:05', '08-17 09:09', '08-17 09:11', '08-17 09:12', '08-17 09:15', '08-17 09:18', '08-17 09:20', '08-17 10:40', null, null, null];
const STAGE_DURATIONS = ['—', '3m', '4m', '2m', '3m', '3m', '3m', '2m', '1h20m', null, null, null];

// stage → 订单状态(terms.md 第 3 节口径:1-2 待核查,3-7 已预占,8-11 装维中,12 已完成)
function statusOfStage(stage) {
  if (stage >= 12) return 'DONE';
  if (stage >= 8) return 'INSTALLING';
  if (stage >= 3) return 'RESERVED';
  return 'PENDING';
}
const STATUS_LABEL = { PENDING: '待核查', RESERVED: '已预占', INSTALLING: '装维中', DONE: '已完成' };

// —— 订单(orders): stage=当前环节;bizType INSTALL/CHANGE/MOVE/DISMANTLE ——
const orders = [
  { orderNo: 'ORD-20250817-001', customerId: 1, productId: 'P-1000', bizType: 'INSTALL', addrCode: 'A-3-501', addrLabel: '望京X · 3栋 · 501', stage: 9, workerId: 1024, preBindTag: 'EPC-0001', loid: 'LOID-88A1', portQuad: 'P-SPL03-07', splitterPort: 'SPL-03-07 · PON 7口', scheduleSlot: '今天 10:00-12:00', submittedAt: '2025-08-17 09:02', scanRetries: 2, chargeAmount: 199 },
  { orderNo: 'ORD-20250817-002', customerId: 4, productId: 'P-500', bizType: 'INSTALL', addrCode: 'A-5-302', addrLabel: '望京X · 5栋302', stage: 3, workerId: null, preBindTag: null, loid: null, portQuad: 'P-SPL01-09', splitterPort: 'SPL-01 · PON 9口', scheduleSlot: null, submittedAt: '2025-08-17 09:05', scanRetries: 0, chargeAmount: 0 },
  { orderNo: 'ORD-20250817-003', customerId: 5, productId: 'P-300', bizType: 'INSTALL', addrCode: 'A-12-906', addrLabel: '望京X · 12栋906', stage: 8, workerId: null, preBindTag: 'EPC-0012', loid: 'LOID-88A9', portQuad: 'P-SPL03-09', splitterPort: 'SPL-03-09 · PON 9口', scheduleSlot: '今天 16:00', submittedAt: '2025-08-17 08:40', scanRetries: 0, chargeAmount: 99 },
  { orderNo: 'ORD-20250817-004', customerId: 1, productId: 'P-1000', bizType: 'CHANGE', addrCode: 'A-3-501', addrLabel: '望京X · 3栋501', stage: 8, workerId: null, preBindTag: 'EPC-0013', loid: 'LOID-88A1', portQuad: 'P-SPL03-07', splitterPort: 'SPL-03-07 · PON 7口', scheduleSlot: null, submittedAt: '2025-08-17 08:10', scanRetries: 0, chargeAmount: 0 },
  { orderNo: 'ORD-20250817-008', customerId: 1, productId: 'P-1000', bizType: 'MOVE', addrCode: 'A-4-102', addrLabel: '望京X · 4栋102', stage: 8, workerId: null, preBindTag: null, loid: 'LOID-88A1', portQuad: null, splitterPort: null, scheduleSlot: null, submittedAt: '2025-08-16 16:20', scanRetries: 0, chargeAmount: 0 },
  { orderNo: 'ORD-20250817-009', customerId: 7, productId: 'P-300', bizType: 'DISMANTLE', addrCode: 'A-2-902', addrLabel: '望京X · 2栋902', stage: 0, workerId: 1024, preBindTag: 'EPC-0110', loid: 'LOID-88B2', portQuad: 'P-SPL03-01', splitterPort: null, scheduleSlot: null, submittedAt: '2025-08-17 08:00', scanRetries: 0, chargeAmount: 0, scanStatus: 'WAIT_SCAN' },
  { orderNo: 'ORD-20250817-000', customerId: 1, productId: 'P-1000', bizType: 'INSTALL', addrCode: 'A-3-501', addrLabel: '望京X · 3栋501', stage: 12, workerId: 1024, preBindTag: 'EPC-0001', loid: 'LOID-88A1', portQuad: 'P-SPL03-07', splitterPort: null, scheduleSlot: null, submittedAt: '2023-05-11 09:00', finishedAt: '05-11', scanRetries: 0, chargeAmount: 199 },
  { orderNo: 'ORD-20250816-018', customerId: 3, productId: 'P-300', bizType: 'INSTALL', addrCode: 'A-1-101', addrLabel: '望京Y · 1栋101', stage: 12, workerId: 1024, preBindTag: 'EPC-0003', loid: 'LOID-88A3', portQuad: 'P-SPL03-03', splitterPort: null, scheduleSlot: null, submittedAt: '2025-08-15 10:00', finishedAt: '08-16', scanRetries: 0, chargeAmount: 99 },
  // 归档历史单(仅师傅端绩效/历史视图引用)
  { orderNo: 'ORD-20250816-011', customerId: 4, productId: 'P-500', bizType: 'INSTALL', addrCode: 'A-6-701', addrLabel: '望京X · 6栋701', stage: 12, workerId: 1024, finishedAt: '08-16', archived: true },
  { orderNo: 'ORD-20250816-010', customerId: 6, productId: 'P-300', bizType: 'INSTALL', addrCode: 'A-9-305', addrLabel: '望京X · 9栋305', stage: 12, workerId: 1024, finishedAt: '08-16', archived: true },
  { orderNo: 'ORD-20250815-008', customerId: 2, productId: 'P-300', bizType: 'DISMANTLE', addrCode: 'A-2-902', addrLabel: '望京X · 2栋902', stage: 0, workerId: 1024, finishedAt: '08-15', archived: true, scanStatus: 'DISMANTLED' },
];

// —— 报障工单(repairTickets): 6 环节闭环 ——
const repairTickets = [
  { ticketNo: 'TKT-20250817-012', customerId: 8, addrCode: 'A-10-1801', addrLabel: '望京X · 10栋1801', faultType: 'no_internet', faultTypeLabel: '单户断网（紧急 SLA ≤4h）', stage: 4, stageTotal: 6, workerId: 1024, slaLeftMinutes: 52, preBindTag: 'EPC-0088', loid: 'LOID-88A7', portQuad: 'P-SPL05-03', splitterPort: 'SPL-05-03 · PON 3口', reportedAt: '2025-08-17 09:40', status: 'PROCESSING', diagnosis: '疑似光猫离线，光功率 -18.6 dBm（偏低），建议现场复核。' },
  { ticketNo: 'TKT-20250817-015', customerId: 1, addrCode: 'A-3-501', addrLabel: '望京X · 3栋501', faultType: 'no_internet', faultTypeLabel: '单户断网（紧急 SLA≤4h）', stage: 4, stageTotal: 6, workerId: 1024, slaLeftMinutes: 96, preBindTag: 'EPC-0001', loid: 'LOID-88A1', portQuad: 'P-SPL03-07', splitterPort: 'SPL-03-07 · PON 7口', reportedAt: '2025-08-17 09:40', status: 'PROCESSING', diagnosis: '远程诊断:路由器 WAN 口异常,建议上门。' },
  { ticketNo: 'TKT-20250730-005', customerId: 1, addrCode: 'A-3-501', addrLabel: '望京X · 3栋501', faultType: 'slow', faultTypeLabel: '网速慢', stage: 6, stageTotal: 6, workerId: 1024, slaLeftMinutes: null, preBindTag: 'EPC-0001', loid: 'LOID-88A1', portQuad: 'P-SPL03-07', splitterPort: 'SPL-03-07 · PON 7口', reportedAt: '2025-07-30 15:02', status: 'RESOLVED', diagnosis: '已解决' },
];

// —— 账单(bills)/缴费(payments): 客户×账期 ——
const bills = [
  { billNo: 'BILL-202508', customerId: 1, period: '2025-08', amount: 158, status: 'UNPAID', statusLabel: '未缴' },
  { billNo: 'BILL-202508-0002', customerId: 2, period: '2025-08', amount: 199, status: 'OVERDUE', statusLabel: '欠费' },
  { billNo: 'BILL-202507', customerId: 1, period: '2025-07', amount: 158, status: 'PAID', statusLabel: '已缴' },
  { billNo: 'BILL-202506', customerId: 1, period: '2025-06', amount: 129, status: 'PAID', statusLabel: '已缴' },
  { billNo: 'BILL-202507-0341', customerId: 3, period: '2025-07', amount: 199, status: 'PAID', statusLabel: '已缴清' },
];
const payments = [
  { payNo: 'PAY20250725001', customerId: 1, amount: 158, period: '2025-07', method: '微信支付', paidAt: '2025-07-25 10:12', billNo: 'BILL-202507' },
  { payNo: 'PAY20250624001', customerId: 1, amount: 129, period: '2025-06', method: '支付宝', paidAt: '2025-06-24 09:40', billNo: 'BILL-202506' },
  { payNo: 'PAY-0002', customerId: 6, amount: 199, period: '2025-08', method: '支付宝', paidAt: '2025-08-16 10:05', billNo: null },
];

// —— 查询/派生 helpers ——
function byCustomer(id) { return customers.find((c) => c.customerId === id); }
function byProduct(id) { return products.find((p) => p.productId === id); }
function byWorker(id) { return workers.find((w) => w.workerId === id) || {}; }
function byOrder(no) { return orders.find((o) => o.orderNo === no); }
function portOf(order) { return ports.find((p) => p.quadCode === order.portQuad); }

// 四码: 环节≥9 的新装/变更单 + 报障单(绑定期) 派生 LINKED;历史冲突行手工保留
function quads() {
  const rows = [];
  for (const o of orders) {
    if (!o.preBindTag || !o.portQuad || !o.loid) continue;
    if (o.bizType !== 'INSTALL' && o.bizType !== 'CHANGE' && o.bizType !== 'DISMANTLE') continue;
    if (o.bizType !== 'DISMANTLE' && o.stage < 9) continue;
    rows.push({ assetCode: o.preBindTag, customerCode: o.loid, portCode: o.portQuad, addrCode: o.addrCode, status: 'LINKED', statusLabel: '一致', orderNo: o.orderNo });
  }
  for (const t of repairTickets) {
    rows.push({ assetCode: t.preBindTag, customerCode: t.loid, portCode: t.portQuad, addrCode: t.addrCode, status: 'LINKED', statusLabel: '一致', orderNo: t.ticketNo });
  }
  rows.push({ assetCode: 'EPC-0002', customerCode: 'LOID-88A2', portCode: 'P-SPL04-02', addrCode: 'A-6-701', status: 'CONFLICT', statusLabel: '冲突', orderNo: 'ORD-20250816-011' });
  // 同一客户/地址的多张单(历史单+在途单+报障)四码相同,按四码去重只保留一条
  const seen = new Set();
  return rows.filter((r) => {
    const key = [r.assetCode, r.customerCode, r.portCode, r.addrCode, r.status].join('|');
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

// 订单时间轴: 三端字段形状不同,统一由本函数生成(stage=12 时全部 DONE)
function timelineOf(order) {
  const done = order.bizType === 'DISMANTLE' ? 0 : order.stage;
  const finished = done >= 12;
  return STAGE_NAMES.map((name, i) => {
    const n = i + 1;
    const result = finished || n < done ? 'DONE' : n === done ? 'DOING' : 'PENDING';
    const time = order.orderNo === 'ORD-20250817-001' ? STAGE_TIMES_001[i] : null;
    return { stage: n, name, result, finishedAt: result === 'DONE' ? time || '' : '', duration: STAGE_DURATIONS[i], note: '' };
  });
}

module.exports = {
  customers, products, workers, ports, assets, loids, orders, repairTickets, bills, payments,
  STAGE_NAMES, STATUS_LABEL, statusOfStage,
  byCustomer, byProduct, byWorker, byOrder, portOf, quads, timelineOf,
};
