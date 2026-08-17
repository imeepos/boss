// 从 seed.js + check_seed.js 的 FKS/LAYER 生成按域拆分的 graph.json(每域一页)。
// 每页含本域实体 + 被引用的外域实体(浅色,标"外部引用")。用法:
//   node scripts/gen_model_pages.js /tmp/pages   (输出 L0.json... 每域一个)
'use strict';
const fs = require('fs');
const path = require('path');
const seed = require('../api/mock/admin/data/seed.js');

const chk = fs.readFileSync(path.join(__dirname, 'check_seed.js'), 'utf8');
const FKS = eval('[' + chk.match(/const FKS = \[([\s\S]*?)\n\];/)[1].replace(/\/\/[^\n]*/g, '') + ']');
const LAYER = JSON.parse('{' + chk.match(/const LAYER = \{([\s\S]*?)\};/)[1].replace(/(\w+):/g, '"$1":').replace(/,(\s*\})/, '$1') + '}');

const CN = {
  regions: '区域树', addresses: '地址树', legalEntities: '运营主体', roles: '角色', permissions: '权限码',
  productOffers: '产品目录', workerGroups: '师傅班组', provisionTemplates: '下发模板', qosTemplates: 'QoS模板', resources: '网络设备',
  workers: '师傅', departments: '部门', regionOffers: '区域运营包', tags: '电子标签池', batches: '入库批次',
  posts: '岗位', accounts: '员工账号', customers: '客户档案',
  assets: '资产台账', loAccounts: 'LO认证账号', ports: '端口', orders: '订单', expansions: '扩容单', transfers: '调拨单',
  orderStages: '订单环节轴', dispatchTickets: '派单工单', reserveRecords: '端口预占', quadLinks: '四码合一', scanLogs: '扫码记录',
  activationCallbacks: '激活回调', dismantles: '拆机单', complaints: '报障工单', bills: '账单', payments: '缴费流水', arrears: '欠费态',
  stopResumeTasks: '停复机任务', provisionTasks: '下发任务', auditLogs: '审计日志', performances: '师傅绩效', commissions: '师傅佣金',
  schedules: '师傅考勤', materials: '物料领用', tools: '工具借用', feedbacks: '服务评价', assetReturns: '资产归还', replacements: '换新单', stocktakes: '盘点任务',
};

// 域划分(页面): key=页面名
const DOMAINS = {
  '组织与权限': ['legalEntities', 'departments', 'posts', 'accounts', 'roles', 'workerGroups'],
  '客户与资费': ['customers', 'productOffers', 'regionOffers'],
  '订单与工单': ['orders', 'orderStages', 'dispatchTickets', 'scanLogs', 'activationCallbacks', 'dismantles', 'complaints'],
  '计费账务': ['bills', 'payments', 'arrears', 'stopResumeTasks'],
  '资产与四码': ['assets', 'tags', 'batches', 'quadLinks', 'replacements', 'stocktakes'],
  '网络资源': ['resources', 'ports', 'reserveRecords', 'transfers', 'expansions', 'loAccounts'],
  '师傅管理': ['workers', 'performances', 'commissions', 'schedules', 'materials', 'tools', 'feedbacks', 'assetReturns'],
  '模板与日志': ['provisionTemplates', 'qosTemplates', 'provisionTasks', 'auditLogs'],
};

const EXT_STYLE = 'rounded=0;dashed=1;whiteSpace=wrap;html=1;fillColor=#f5f5f5;strokeColor=#999999;fontColor=#777777;';
const inLayer = n => LAYER[n] != null && n in seed;

const outDir = process.argv[2] || '/tmp/pages';
fs.mkdirSync(outDir, { recursive: true });
for (const f of fs.readdirSync(outDir)) fs.unlinkSync(path.join(outDir, f));

for (const [page, owned] of Object.entries(DOMAINS)) {
  const own = new Set(owned);
  // 外部引用: 本域实体 FK 指向的、不在本域的实体
  const ext = new Set();
  for (const [ent, , target] of FKS) {
    if (own.has(ent) && !own.has(target) && inLayer(target)) ext.add(target);
  }
  const nodes = [];
  for (const n of owned) nodes.push({
    id: n, label: '<b>' + n + '</b><br/>' + (CN[n] || '') + ' · ' + seed[n].length + '行',
    group: 'L' + LAYER[n], width: 210, height: 56,
  });
  const extArr = [...ext].sort((a, b) => LAYER[a] - LAYER[b]);
  for (const n of extArr) nodes.push({
    id: n, label: n + '<br/>' + (CN[n] || '') + '(外部引用)', style: EXT_STYLE,
    group: 'L' + LAYER[n], width: 170, height: 46,
  });
  const edges = [];
  for (const [ent, field, target, tkey, , sameLayerOK] of FKS) {
    if (ent === target) continue;
    if (!(own.has(ent) && (own.has(target) || ext.has(target)))) continue;
    const isExt = ext.has(target);
    edges.push({ source: ent, target, label: (sameLayerOK ? '◆ ' : '▲ ') + field + (isExt ? '(外)' : '') });
  }
  const file = path.join(outDir, page + '.json');
  fs.writeFileSync(file, JSON.stringify({ direction: 'LR', nodes, edges }, null, 1));
  console.log(page, 'own=' + owned.length, 'ext=' + extArr.length, 'edges=' + edges.length);
}
