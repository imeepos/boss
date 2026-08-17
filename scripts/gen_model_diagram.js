// 从 seed.js + check_seed.js 的 FKS/LAYER 生成 graph.json,供 autolayout.py 自动布局。
// 保证图与数据/校验规则 100% 同源。用法: node scripts/gen_model_diagram.js > /tmp/graph.json
'use strict';
const fs = require('fs');
const path = require('path');
const seed = require('../api/mock/admin/data/seed.js');

const chk = fs.readFileSync(path.join(__dirname, 'check_seed.js'), 'utf8');
const FKS = eval('[' + chk.match(/const FKS = \[([\s\S]*?)\n\];/)[1].replace(/\/\/[^\n]*/g, '') + ']');
const LAYER = JSON.parse('{'+chk.match(/const LAYER = \{([\s\S]*?)\};/)[1].replace(/(\w+):/g,'"$1":').replace(/,(\s*\})/,'$1')+'}');

const CN = {
  regions: '区域树 LTREE', addresses: '地址树 LTREE', legalEntities: '运营主体(子公司)', roles: '角色', permissions: '权限码',
  productOffers: '产品目录', workerGroups: '师傅班组', provisionTemplates: '下发模板', qosTemplates: 'QoS模板', resources: '网络设备 OLT/分光器',
  workers: '师傅', departments: '部门', regionOffers: '区域运营包', tags: '电子标签池', batches: '入库批次',
  posts: '岗位', accounts: '员工账号(权限主体)', customers: '客户档案',
  assets: '资产台账', loAccounts: 'LO认证账号', ports: '端口', orders: '订单(12环节)', expansions: '扩容单', transfers: '调拨单',
  orderStages: '订单环节轴', dispatchTickets: '派单工单', reserveRecords: '端口预占', quadLinks: '四码合一', scanLogs: '扫码记录',
  activationCallbacks: '激活回调', dismantles: '拆机单', complaints: '报障工单', bills: '账单', payments: '缴费流水', arrears: '欠费态',
  stopResumeTasks: '停复机任务', provisionTasks: '下发任务', auditLogs: '审计日志', performances: '师傅绩效', commissions: '师傅佣金',
  schedules: '师傅考勤', materials: '物料领用', tools: '工具借用', feedbacks: '服务评价', assetReturns: '资产归还', replacements: '换新单', stocktakes: '盘点任务',
};
const GL = { 0: 'L0 基础数据(集团共享)', 1: 'L1 运营主体自定义', 2: 'L2 依赖传导层', 3: 'L3 组织与主体', 4: 'L4 业务主单', 5: 'L5 单据派生与流水' };

const nodes = Object.keys(LAYER)
  .filter(n => n in seed)
  .map(n => ({
    id: n,
    label: '<b>' + n + '</b><br/>' + (CN[n] || '') + ' · ' + seed[n].length + '行',
    group: 'L' + LAYER[n],
    groupLabel: GL[LAYER[n]],
    width: 210, height: 56,
  }));

const edges = [];
for (const [ent, field, target, tkey, , sameLayerOK] of FKS) {
  if (ent === target) continue;
  if (!(ent in LAYER) || !(target in LAYER)) continue;
  edges.push({ source: ent, target, label: (sameLayerOK ? '◆ ' : '▲ ') + field });
}

console.log(JSON.stringify({ direction: 'LR', nodes, edges }, null, 1));
