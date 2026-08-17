// seed 自检: 外键可达性 + 分层闭合(上层只引低层) + 业务不变量。
// 用法: node scripts/check_seed.js  退出码 0=通过
'use strict';
const seed = require('../api/mock/admin/data/seed.js');

const errs = [];
const idx = (rows, key) => new Set(rows.map(r => r[key]));

// 1) 外键可达性: [实体, 字段, 目标实体, 目标主键, 是否可空]
const FKS = [
  ['addresses', 'path', 'addresses', 'path', false, true], // 树父(自身,同层白名单)
  ['legalEntities', 'crossRegionIds', 'regions', 'id', true, true],
  ['provisionTemplates', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['qosTemplates', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['workerGroups', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['productOffers', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['resources', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['tags', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['batches', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['customers', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['expansions', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['stocktakes', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['resources', 'addressId', 'addresses', 'id', false, false],
  ['resources', 'parentId', 'resources', 'id', true, true],
  ['workers', 'groupId', 'workerGroups', 'id', false, false],
  ['workers', 'regionId', 'regions', 'id', false, false],
  ['departments', 'legalEntityId', 'legalEntities', 'id', false, false],
  ['tags', 'boundAssetId', 'assets', 'assetId', true, true], // 预绑定回填: 跨层逆引,显式豁免(见 data-layers.md)
  ['posts', 'deptId', 'departments', 'id', false, false],
  ['accounts', 'roleId', 'roles', 'id', false, false],
  ['accounts', 'legalEntityId', 'legalEntities', 'id', true, false],
  ['accounts', 'deptId', 'departments', 'id', true, false],
  ['accounts', 'postId', 'posts', 'id', true, false],
  ['customers', 'addressId', 'addresses', 'id', false, false],
  ['assets', 'batchId', 'batches', 'id', false, false],
  ['assets', 'tagId', 'tags', 'tagId', true, false],
  ['assets', 'addressId', 'addresses', 'id', true, false],
  ['regionOffers', 'offerId', 'productOffers', 'offerId', false, false],
  ['loAccounts', 'customerId', 'customers', 'customerId', false, false],
  ['loAccounts', 'offerId', 'productOffers', 'offerId', false, false],
  ['loAccounts', 'qosTemplateId', 'qosTemplates', 'id', false, false],
  ['ports', 'resourceId', 'resources', 'id', false, false],
  ['ports', 'addressId', 'addresses', 'id', false, false],
  ['ports', 'orderId', 'orders', 'orderId', true, false],
  ['orders', 'customerId', 'customers', 'customerId', false, false],
  ['orders', 'offerId', 'productOffers', 'offerId', false, false],
  ['orders', 'addressId', 'addresses', 'id', false, false],
  ['expansions', 'regionId', 'regions', 'id', false, false],
  ['transfers', 'resourceId', 'resources', 'id', false, false],
  ['transfers', 'fromRegionId', 'regions', 'id', false, false],
  ['transfers', 'toRegionId', 'regions', 'id', false, false],
  ['orderStages', 'orderId', 'orders', 'orderId', false, false],
  ['dispatchTickets', 'orderId', 'orders', 'orderId', false, false],
  ['dispatchTickets', 'workerId', 'workers', 'workerId', false, false],
  ['reserveRecords', 'portId', 'ports', 'portId', false, false],
  ['reserveRecords', 'orderId', 'orders', 'orderId', false, false],
  ['quadLinks', 'assetId', 'assets', 'assetId', false, false],
  ['quadLinks', 'customerId', 'customers', 'customerId', false, false],
  ['quadLinks', 'portId', 'ports', 'portId', false, false],
  ['quadLinks', 'addressId', 'addresses', 'id', false, false],
  ['scanLogs', 'orderId', 'orders', 'orderId', false, false],
  ['scanLogs', 'workerId', 'workers', 'workerId', false, false],
  ['scanLogs', 'tagId', 'tags', 'tagId', false, false],
  ['activationCallbacks', 'orderId', 'orders', 'orderId', false, false],
  ['dismantles', 'orderId', 'orders', 'orderId', false, false],
  ['dismantles', 'assetId', 'assets', 'assetId', false, false],
  ['dismantles', 'portId', 'ports', 'portId', false, false],
  ['complaints', 'customerId', 'customers', 'customerId', false, false],
  ['complaints', 'orderId', 'orders', 'orderId', true, false],
  ['bills', 'customerId', 'customers', 'customerId', false, false],
  ['payments', 'billId', 'bills', 'billId', false, false],
  ['arrears', 'customerId', 'customers', 'customerId', false, false],
  ['stopResumeTasks', 'customerId', 'customers', 'customerId', false, false],
  ['provisionTasks', 'templateId', 'provisionTemplates', 'id', false, false],
  ['auditLogs', 'accountId', 'accounts', 'id', false, false],
  ['workerSettings', 'workerId', 'workers', 'workerId', false, false],
  ['workerMessages', 'workerId', 'workers', 'workerId', false, false],
  ['performances', 'workerId', 'workers', 'workerId', false, false],
  ['commissions', 'workerId', 'workers', 'workerId', false, false],
  ['schedules', 'workerId', 'workers', 'workerId', false, false],
  ['materials', 'workerId', 'workers', 'workerId', false, false],
  ['tools', 'workerId', 'workers', 'workerId', false, false],
  ['feedbacks', 'workerId', 'workers', 'workerId', false, false],
  ['feedbacks', 'ticketId', 'dispatchTickets', 'ticketId', false, false],
  ['feedbacks', 'customerId', 'customers', 'customerId', false, false],
  ['assetReturns', 'workerId', 'workers', 'workerId', false, false],
  ['assetReturns', 'assetId', 'assets', 'assetId', false, false],
  ['replacements', 'assetId', 'assets', 'assetId', false, false],
];

for (const [ent, field, target, tkey] of FKS) {
  const rows = seed[ent]; if (!rows) { errs.push('实体缺失: ' + ent); continue; }
  const keys = idx(seed[target], tkey);
  for (const r of rows) {
    const v = r[field];
    if (v == null) continue;
    const vals = Array.isArray(v) ? v : [v];
    for (const x of vals) if (!keys.has(x)) errs.push(`${ent}.${field}=${x} 悬空(不在 ${target}.${tkey})`);
  }
}

// 2) 业务不变量
const addr = new Map(seed.addresses.map(a => [a.id, a]));
for (const c of seed.customers) {
  if (addr.get(c.addressId).level !== 5) errs.push(`customer ${c.customerId} addressId 未挂楼栋级(level5)`);
}
for (const p of seed.ports) {
  if (!['RESERVED', 'USED'].includes(p.status) && p.orderId != null) errs.push(`port ${p.portCode} 非占用态却有 orderId`);
  if (p.status === 'RESERVED' && p.orderId == null) errs.push(`port ${p.portCode} RESERVED 但无 orderId`);
}
for (const q of seed.quadLinks) {
  if (!seed.customers.some(c => c.customerId === q.customerId)) errs.push('quad 第二码必须是客户');
}
const seen = new Set();
for (const b of seed.bills) {
  const k = b.customerId + ':' + b.period;
  if (seen.has(k)) errs.push('账单 客户×账期 重复: ' + k);
  seen.add(k);
}
// tags 未绑定必须显式 null(不得用"—")
for (const t of seed.tags) if (t.boundAssetId === '—') errs.push('tags.boundAssetId 使用了"—"伪空值');

// 5) 资费作用域不变量: 区域运营包必须落在该公司经营区域内;订单 offer 的公司必须覆盖订单区域;账单=快照价
const regionCover = (entityRegionIds, regionPath) => {
  const paths = new Set(seed.regions.filter(r => entityRegionIds.includes(r.id)).map(r => r.path));
  for (const p of paths) if (regionPath === p || regionPath.startsWith(p + '.')) return true;
  return false;
};
const offerById = new Map(seed.productOffers.map(o => [o.offerId, o]));
for (const o of seed.productOffers) {
  if (!o.name || !o.name.trim()) errs.push(`offer ${o.offerId}: 公司级名称必填`);
}
// 师傅服务区域必须落在其班组所属公司的经营区域内
const groupById = new Map(seed.workerGroups.map(g => [g.id, g]));
for (const w of seed.workers) {
  const ent = seed.legalEntities.find(e => e.id === groupById.get(w.groupId).legalEntityId);
  if (!regionCover(ent.crossRegionIds, seed.regions.find(r => r.id === w.regionId).path))
    errs.push(`worker ${w.staffNo}: ${ent.code} 未经营其服务区域`);
}
// 同公司一致性: 订单/账号/资产/模板不得跨公司引用
const custById = new Map(seed.customers.map(c => [c.customerId, c]));
const tplById = new Map([...seed.provisionTemplates, ...seed.qosTemplates].map(t => [t.id, t]));
for (const o of seed.orders) {
  if (offerById.get(o.offerId).legalEntityId !== custById.get(o.customerId).legalEntityId)
    errs.push(`order ${o.orderNo}: 产品公司与客户归属公司不一致`);
}
for (const l of seed.loAccounts) {
  if (tplById.get(l.qosTemplateId).legalEntityId !== offerById.get(l.offerId).legalEntityId)
    errs.push(`loAccount ${l.loid}: QoS模板公司与产品公司不一致`);
}
for (const t of seed.provisionTasks) {
  const lo = seed.loAccounts.find(x => x.loid === t.loid);
  if (tplById.get(t.templateId).legalEntityId !== offerById.get(lo.offerId).legalEntityId)
    errs.push(`provisionTask ${t.id}: 模板公司与账号公司不一致`);
}
for (const rp of seed.regionOffers) {
  const o = offerById.get(rp.offerId);
  const ent = seed.legalEntities.find(e => e.id === o.legalEntityId);
  if (!regionCover(ent.crossRegionIds, rp.regionPath)) errs.push(`regionOffers ${rp.id}: ${ent.code} 未经营 ${rp.regionPath} 却设了区域价`);
}
for (const o of seed.orders) {
  const offer = offerById.get(o.offerId);
  const ent = seed.legalEntities.find(e => e.id === offer.legalEntityId);
  if (!regionCover(ent.crossRegionIds, o.regionPath)) errs.push(`order ${o.orderNo}: ${ent.code} 未经营 ${o.regionPath},不可下单`);
  const rp = seed.regionOffers.find(r => r.offerId === o.offerId && o.regionPath.startsWith(r.regionPath));
  const expect = rp ? rp.monthlyFee : offer.monthlyFee;
  if (o.priceSnapshot !== expect) errs.push(`order ${o.orderNo}: 快照价 ${o.priceSnapshot} ≠ 生效价 ${expect}`);
}
// stopResumeTasks.loid 弱引用豁免检查: 允许为空,若非空应在 loAccounts
const loids = new Set(seed.loAccounts.map(l => l.loid));
for (const s of seed.stopResumeTasks) if (s.loid && !loids.has(s.loid)) errs.push(`stopResumeTasks.loid=${s.loid} 不在 loAccounts`);

// 3) 分层闭合: L 值越小越基础
const LAYER = { regions: 0, addresses: 0, roles: 0, permissions: 0, legalEntities: 0,
  provisionTemplates: 1, qosTemplates: 1, productOffers: 1, workerGroups: 1, resources: 1, workers: 2, departments: 2, regionOffers: 2, tags: 2, batches: 2, posts: 3, accounts: 3, customers: 3,
  assets: 4, loAccounts: 4, ports: 4, orders: 4, expansions: 4, transfers: 4, orderStages: 5, dispatchTickets: 5, reserveRecords: 5,
  quadLinks: 5, scanLogs: 5, activationCallbacks: 5, dismantles: 5, complaints: 5, bills: 5, payments: 5, arrears: 5,
  stopResumeTasks: 5, provisionTasks: 5, auditLogs: 5, performances: 5, commissions: 5, schedules: 5, materials: 5,
  tools: 5, feedbacks: 5, assetReturns: 5, replacements: 5, stocktakes: 5, workerSettings: 5, workerMessages: 5 };
for (const [ent, , target, , , sameLayerOK] of FKS) {
  if (sameLayerOK) continue;
  if (LAYER[target] > LAYER[ent]) errs.push(`分层违例: ${ent}(L${LAYER[ent]}) 引用了更低层的 ${target}(L${LAYER[target]})`);
}

// 4) tags 白名单(同层逆引是预绑定语义,显式豁免并登记)
// tags(L2).boundAssetId → assets(L4) 属"后期回填"例外,已在 data-layers.md 声明,不算违例。

if (errs.length) { console.log('FAIL ' + errs.length + ' 处:\n' + errs.join('\n')); process.exit(1); }
const total = Object.keys(seed).reduce((n, k) => n + (Array.isArray(seed[k]) ? seed[k].length : Object.keys(seed[k]).length), 0);
console.log(`PASS seed 自洽: ${Object.keys(seed).length} 类实体 / ${total} 行数据, ${FKS.length} 条外键规则 + 不变量全部通过`);
