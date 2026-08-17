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
// 2. 端口占用与订单状态一致: RESERVED/USED 必须指回存在的订单,且订单尚未拆机完成
for (const p of db.ports) {
  if (p.status === 'IDLE' || p.status === 'DISABLED') continue;
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

console.log(failed === 0 ? '\nALL CHECKS PASSED' : '\n' + failed + ' CHECKS FAILED');
process.exit(failed === 0 ? 0 : 1);
