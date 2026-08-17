// seed.js —— 关系自洽的模拟数据基座(契约: docs/contract/data-layers.md)
// 规则: 全部 numeric id 强引用; code 仅展示冗余(xxx_id + xxx_name 双列);
//       分层 L0基础→L5流水,上层只引用低层;四码第二码=客户(非LOID)。
// 校验: node scripts/check_seed.js
'use strict';

// ===== L0 基础数据(零依赖) =====
const regions = [
  { id: 1, path: 'CN', level: 1, name: '集团' },
  { id: 2, path: 'CN.NORTH', level: 2, name: '华北大区' },
  { id: 3, path: 'CN.NORTH.BJ', level: 3, name: '北京省' },
  { id: 4, path: 'CN.NORTH.BJ.CITY', level: 4, name: '北京市' },
];
const addresses = [
  { id: 101, path: 'BJ', level: 1, name: '北京市' },
  { id: 102, path: 'BJ.CHAOYANG', level: 2, name: '朝阳区' },
  { id: 103, path: 'BJ.CHAOYANG.WJ', level: 3, name: '望京街道' },
  { id: 104, path: 'BJ.CHAOYANG.WJ.XQ', level: 4, name: 'X小区' },
  { id: 105, path: 'BJ.CHAOYANG.WJ.XQ.B3', level: 5, name: '3栋' },
  { id: 106, path: 'BJ.CHAOYANG.WJ.XQ.B5', level: 5, name: '5栋' },
];
const roles = [
  { id: 1, code: 'sysadmin', name: '系统管理员' },
  { id: 2, code: 'ops', name: '运营人员' },
  { id: 3, code: 'dispatcher', name: '调度员' },
  { id: 4, code: 'asset_admin', name: '资产管理员' },
  { id: 5, code: 'resource_admin', name: '资源管理员' },
  { id: 6, code: 'technician', name: '装维师傅' },
  { id: 7, code: 'analyst', name: '经营分析' },
];
const permissions = ['asset:create', 'asset:scrap', 'order:create', 'order:dispatch', 'port:reserve', 'port:release', 'bill:adjust', 'menu:order', 'menu:asset', 'menu:billing'].map((code, i) => ({ id: i + 1, code }));
const rolePermissions = { 1: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10], 2: [3, 4, 7, 8, 10], 3: [4, 5], 4: [1, 2, 9], 5: [5, 6] };

// ===== L0 运营主体(与 regions 同层软关联,是各归属数据的挂靠根) =====
const legalEntities = [
  { id: 41, code: 'LEG-A', name: '北京XX网络有限公司', crossRegionIds: [4] },
  { id: 42, code: 'LEG-B', name: '天津XX网络有限公司', crossRegionIds: [] },
];

// ===== L1 依赖运营主体 =====
const workerGroups = [ // 运营主体自定义的班组(非集团基础数据)
  { id: 11, legalEntityId: 41, name: '装机一组' },
  { id: 12, legalEntityId: 41, name: '装机二组' },
];
const provisionTemplates = [ // 下发模板: 各公司设备型号不同
  { id: 21, legalEntityId: 41, code: 'TPL-FTTH', name: 'FTTH标准开通' },
  { id: 22, legalEntityId: 42, code: 'TPL-FTTH-B', name: 'FTTH开通(B厂设备)' },
];
const qosTemplates = [ // QoS模板: 各公司策略不同
  { id: 31, legalEntityId: 41, code: 'QoS-STD', name: '标准' },
  { id: 32, legalEntityId: 41, code: 'QoS-VIP', name: 'VIP' },
];
const productOffers = [ // L1 各子公司自己的产品: 名称/带宽/价格全属公司,无集团规格层
  { offerId: 9001, legalEntityId: 41, name: '畅享宽带 300M', bandwidth: '300M', monthlyFee: 99, effectiveAt: '2026-07-01', status: 'PUBLISHED' },
  { offerId: 9002, legalEntityId: 41, name: '尊享宽带 500M', bandwidth: '500M', monthlyFee: 139, effectiveAt: '2026-07-01', status: 'PUBLISHED' },
  { offerId: 9003, legalEntityId: 41, name: '极速宽带 1000M', bandwidth: '1000M', monthlyFee: 199, effectiveAt: '2026-08-01', status: 'PUBLISHED' },
  { offerId: 9004, legalEntityId: 42, name: '极速畅游版 300M', bandwidth: '300M', monthlyFee: 89, effectiveAt: '2026-07-01', status: 'PUBLISHED' },
  { offerId: 9005, legalEntityId: 42, name: '极速畅游版 500M', bandwidth: '500M', monthlyFee: 129, effectiveAt: '2026-07-01', status: 'OFFLINE' },  // LEG-B 无 1000M
];
const resources = [ // OLT→分光器 树,各公司建设的网络设备
  { id: 51, legalEntityId: 41, code: 'OLT-01', name: '望京OLT-01', type: 'OLT', parentId: null, addressId: 104 },
  { id: 52, legalEntityId: 41, code: 'SPL-01', name: 'X小区分光器1', type: 'SPLITTER', parentId: 51, addressId: 104 },
  { id: 53, legalEntityId: 41, code: 'SPL-02', name: 'X小区分光器2', type: 'SPLITTER', parentId: 51, addressId: 104 },
];

// ===== L2 依赖L0/L1 =====
const workers = [ // L2 依赖班组(L1)
  { workerId: 1024, staffNo: 'WK-1024', name: '张师傅', groupId: 11, regionId: 4, phone: '138****8899' },
  { workerId: 1025, staffNo: 'WK-1025', name: '李师傅', groupId: 11, regionId: 4, phone: '139****7788' },
  { workerId: 1026, staffNo: 'WK-1026', name: '王师傅', groupId: 12, regionId: 4, phone: '137****6655' },
];
const departments = [
  { id: 61, legalEntityId: 41, name: '装维部' },
  { id: 62, legalEntityId: 41, name: '资源部' },
  { id: 63, legalEntityId: 42, name: '综合部' },
];
const regionOffers = [ // L2 区域运营包: 区域级名称+价格均可覆盖(展示名回退: 区域名→公司名→规格名)
  { id: 9101, offerId: 9003, regionPath: 'CN.NORTH.BJ.CITY', name: '北京千兆特惠版', monthlyFee: 189, reason: '城市促销' },
  { id: 9102, offerId: 9001, regionPath: 'CN.NORTH.BJ.CITY', name: null, monthlyFee: 95, reason: '新装立减' }, // 仅改价不改名
];
const tags = [ // 标签池: 未绑定 boundAssetId=null(禁用"—"伪值)
  { tagId: 71, legalEntityId: 41, tagNo: 'TAG-0001', epcCode: 'EPC-0001', band: 'UHF', boundAssetId: 801, battery: '86%' },
  { tagId: 72, legalEntityId: 41, tagNo: 'TAG-0002', epcCode: 'EPC-0002', band: 'UHF', boundAssetId: 802, battery: '92%' },
  { tagId: 73, legalEntityId: 41, tagNo: 'TAG-0003', epcCode: 'EPC-0003', band: 'UHF', boundAssetId: null, battery: '78%' },
];
const batches = [ // 入库批次:公司采购行为
  { id: 91, legalEntityId: 41, code: 'RK-202607-01', name: '7月光猫批次' },
  { id: 92, legalEntityId: 41, code: 'RK-202608-01', name: '8月光猫批次' }];

// ===== L3 依赖L0~L2 =====
const posts = [
  { id: 611, deptId: 61, code: 'dispatcher', name: '调度岗', roleIds: [3] },
  { id: 612, deptId: 61, code: 'field_tech', name: '外勤岗', roleIds: [6] },
  { id: 613, deptId: 62, code: 'resource_admin', name: '资源管理岗', roleIds: [5] },
];
const accounts = [
  { id: 1, username: 'admin', realName: '系统管理员', roleId: 1, legalEntityId: null, deptId: null, postId: null, regionScope: 'CN', status: 1 },
  { id: 2, username: 'dispatch01', realName: '刘调度', roleId: 3, legalEntityId: 41, deptId: 61, postId: 611, regionScope: 'CN.NORTH.BJ', status: 1 },
  { id: 3, username: 'asset01', realName: '陈资产', roleId: 4, legalEntityId: 41, deptId: 62, postId: 613, regionScope: 'CN.NORTH.BJ.CITY', status: 1 },
];
const customers = [ // addressId 必须楼栋级(level 5)
  { customerId: 201, legalEntityId: 41, name: '王先生', phone: '138****1234', idType: '身份证', idNo: '110***1234', realNameStatus: 'VERIFIED', serviceStatus: 'ACTIVE', addressId: 105 },
  { customerId: 202, legalEntityId: 41, name: '吴女士', phone: '139***5678', idType: '护照', idNo: 'P***56', realNameStatus: 'PENDING', serviceStatus: 'ARREARS', addressId: 105 },
  { customerId: 203, legalEntityId: 41, name: '郑先生', phone: '136***9012', idType: '身份证', idNo: '110***9012', realNameStatus: 'VERIFIED', serviceStatus: 'ACTIVE', addressId: 106 },
];

// ===== L4 业务主单 =====
const assets = [
  { assetId: 801, assetCode: 'A-20260001', type: '光猫', batchId: 91, tagId: 71, addressId: 105, status: 'DEPLOYED' },
  { assetId: 802, assetCode: 'A-20260002', type: '光猫', batchId: 91, tagId: 72, addressId: 106, status: 'DEPLOYED' },
  { assetId: 803, assetCode: 'A-20260003', type: 'ONU', batchId: 92, tagId: null, addressId: null, status: 'IN_STOCK' },
];
const loAccounts = [
  { loid: 'LOID-88A1', customerId: 201, offerId: 9003, qosTemplateId: 32, status: 'ACTIVE' },
  { loid: 'LOID-88A2', customerId: 202, offerId: 9001, qosTemplateId: 31, status: 'SUSPENDED' },
  { loid: 'LOID-88B2', customerId: 203, offerId: 9002, qosTemplateId: 31, status: 'ACTIVE' },
];
const ports = [ // RESERVED/USED 才有 orderId
  { portId: 5011, portCode: 'P-SPL01-01', resourceId: 52, addressId: 105, quadCode: 'Q-5011', status: 'USED', orderId: 3001 },
  { portId: 5012, portCode: 'P-SPL01-02', resourceId: 52, addressId: 105, quadCode: 'Q-5012', status: 'RESERVED', orderId: 3002 },
  { portId: 5021, portCode: 'P-SPL02-01', resourceId: 53, addressId: 106, quadCode: 'Q-5021', status: 'USED', orderId: 3003 },
  { portId: 5022, portCode: 'P-SPL02-02', resourceId: 53, addressId: 106, quadCode: 'Q-5022', status: 'IDLE', orderId: null },
];
const orders = [ // offerId 定目录, priceSnapshot=成交时生效价(区域价优先于基础价)
  { orderId: 3001, orderNo: 'ORD-20260817-001', customerId: 201, offerId: 9003, priceSnapshot: 189, addressId: 105, regionPath: 'CN.NORTH.BJ.CITY', stage: 9, status: 'INSTALLING' },
  { orderId: 3002, orderNo: 'ORD-20260817-002', customerId: 202, offerId: 9001, priceSnapshot: 95, addressId: 105, regionPath: 'CN.NORTH.BJ.CITY', stage: 4, status: 'RESERVED' },
  { orderId: 3003, orderNo: 'ORD-20260816-003', customerId: 203, offerId: 9002, priceSnapshot: 139, addressId: 106, regionPath: 'CN.NORTH.BJ.CITY', stage: 12, status: 'DONE' },
];
const expansions = [{ id: 5101, legalEntityId: 41, expansionNo: 'EXP-2026-001', regionId: 4, expectedPorts: 48, status: 'PENDING' }];
const transfers = [{ id: 5201, transferNo: 'TRF-2026-001', resourceId: 53, fromRegionId: 4, toRegionId: 4, status: 'PENDING' }];

// ===== L5 单据派生与流水 =====
const STAGE_NAMES = ['下单', '资源核查', '端口预占', '合同收费', '标签预绑定', '创建账号', '预下发', '派单', '扫码绑定', '激活', '激活回调', '更新GIS'];
const orderStages = [];
for (const o of orders) {
  STAGE_NAMES.forEach((n, i) => {
    const s = o.stage || 0;
    orderStages.push({ orderId: o.orderId, stage: i + 1, name: n, result: i + 1 < s ? 'DONE' : i + 1 === s ? 'DOING' : 'PENDING', retries: 0 });
  });
}
const dispatchTickets = [
  { ticketId: 7001, ticketNo: 'TKT-001', orderId: 3001, workerId: 1024, status: 'DOING' },
  { ticketId: 7002, ticketNo: 'TKT-002', orderId: 3003, workerId: 1025, status: 'DONE' },
];
const reserveRecords = [
  { id: 7101, portId: 5012, orderId: 3002, status: 'HELD' },
];
const quadLinks = [ // 四码=资产-客户-端口-地址(第二码是客户!)
  { id: 7201, assetId: 801, customerId: 201, portId: 5011, addressId: 105, status: 'LINKED' },
  { id: 7202, assetId: 802, customerId: 203, portId: 5021, addressId: 106, status: 'LINKED' },
];
const scanLogs = [{ id: 7301, orderId: 3001, workerId: 1024, tagId: 71, result: 'MATCH' }];
const activationCallbacks = [{ id: 7401, orderId: 3003, result: 'SUCCESS', retries: 0 }];
const dismantles = [{ id: 7501, dismantleNo: 'DIS-2026-001', orderId: 3003, assetId: 802, portId: 5021, status: 'PENDING' }];
const complaints = [{ id: 7601, ticketNo: 'CMP-2026-001', customerId: 201, orderId: 3001, type: '网速慢', status: 'OPEN' }];
const bills = [ // 金额 = 订单成交价快照
  { billId: 8101, billNo: 'BILL-202608-201', customerId: 201, period: '2026-08', amount: 189, status: 'UNPAID' },
  { billId: 8102, billNo: 'BILL-202608-203', customerId: 203, period: '2026-08', amount: 139, status: 'PAID' },
  { billId: 8103, billNo: 'BILL-202607-202', customerId: 202, period: '2026-07', amount: 95, status: 'OVERDUE' },
];
const payments = [{ id: 8201, payNo: 'PAY-20260820-001', billId: 8102, amount: 139, method: '微信', status: 'SUCCESS' }];
const arrears = [{ customerId: 202, amount: 95, days: 15, status: 'SUSPENDED' }];
const stopResumeTasks = [{ id: 8301, customerId: 202, loid: 'LOID-88A2', action: 'STOP', status: 'DONE' }];
const provisionTasks = [{ id: 8401, loid: 'LOID-88A1', templateId: 21, status: 'DONE' }];
const auditLogs = [{ id: 8501, accountId: 2, action: '状态变更', targetType: 'order', targetId: '3001', ip: '10.0.0.2' }];
const workerSettings = [ // 师傅接单设置(1:1)
  { id: 8941, workerId: 1024, accepting: true, radiusKm: 5, acceptTypes: '新装宽带/宽带变更/拆机' },
];
const workerMessages = [ // 师傅站内消息
  { id: 8942, workerId: 1024, level: 'URGENT', title: '今日工单催办', content: 'TKT-001 客户等待中,请尽快上门', sentAt: '2026-08-17 09:00', read: false },
  { id: 8943, workerId: 1025, level: 'INFO', title: '月度绩效已出', content: '2026-08 绩效可查', sentAt: '2026-08-16 18:00', read: true },
];
const performances = [{ id: 8601, workerId: 1024, period: '2026-08', finished: 46, onTimeRate: 98, score: 4.9 }];
const commissions = [{ id: 8701, workerId: 1024, period: '2026-08', formula: '新装×40', amount: 1840 }];
const schedules = [{ id: 8801, workerId: 1024, month: '2026-08', busyDays: 22 }];
const materials = [{ id: 8901, workerId: 1024, name: '光纤跳线', qty: 20 }];
const tools = [{ id: 8951, workerId: 1025, name: '熔纤机', borrowed: true }];
const feedbacks = [{ id: 8961, workerId: 1024, ticketId: 7002, customerId: 203, score: 5, needReview: false }];
const assetReturns = [{ id: 8971, workerId: 1025, assetId: 803, reason: '完工回收', status: 'PENDING' }];
const replacements = [{ id: 8981, assetId: 802, reason: '光猫故障', priority: 'HIGH', status: 'PENDING' }];
const stocktakes = [{ id: 8991, legalEntityId: 41, scope: 'CN.NORTH.BJ.CITY', progress: 60, diffCount: 2, status: 'DOING' }];

module.exports = {
  regions, addresses, roles, permissions, rolePermissions, workerGroups,
  provisionTemplates, qosTemplates,
  legalEntities, productOffers, resources, workers,
  departments, regionOffers, tags, batches,
  posts, accounts, customers,
  assets, loAccounts, ports, orders, expansions, transfers,
  orderStages, dispatchTickets, reserveRecords, quadLinks, scanLogs, activationCallbacks,
  dismantles, complaints, bills, payments, arrears, stopResumeTasks, provisionTasks, auditLogs,
  performances, commissions, schedules, materials, tools, feedbacks, assetReturns, replacements, stocktakes, workerSettings, workerMessages,
};
