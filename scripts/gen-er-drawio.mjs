#!/usr/bin/env node
// 从 migrations 提炼的表关系规格生成 docs/boss-entities-er.drawio(唯一活 ER 图)。
// 用法: node scripts/gen-er-drawio.mjs && 用 draw.io.app 导出 png/svg(scripts/export-er.sh)。
// 规格≠DDL 全量字段,只列定位关系所需:主键 + 外键 + 关键唯一约束。

import { writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');

// 域分组: [域中文名, 域色, [[表名, 显示行...], ...]]
// 行约定: PK=主键 FK=硬外键(有REFERENCES) sFK=软引用(无约束) UQ=唯一
const GROUPS = [
  ['权限·组织·地理', '#dae8fc', [
    ['roles', '角色', 'PK id', 'UQ code'],
    ['permissions', '权限码', 'PK id', 'UQ code'],
    ['role_permissions', '角色-权限', 'FK role_id', 'FK permission_id'],
    ['accounts', '后台账号', 'PK id', 'FK role_id', 'FK legal_entity_id?', 'FK dept_id? / post_id?', 'region_scope(LTREE)'],
    ['account_org_histories', '账号归属台账', 'FK account_id', 'FK legal_entity_id?/dept_id?/post_id?', 'effective_from/to + reason'],
    ['audit_logs', '审计日志(按月分区)', 'sFK account_id', 'sFK target_type/target_id', '事发快照列'],
    ['biz_params', '全局参数', 'PK key', 'sFK updated_by→accounts'],
    ['api_keys', 'API密钥', 'PK id', 'FK account_id CASCADE', 'sFK created_by'],
    ['import_tasks', '导入任务', 'PK id', 'FK operator_id→accounts'],
    ['attachments', '通用附件(000065)', 'PK id', 'UQ object_key', "sFK uploader_type('account'/'worker'/'customer')+uploader_id"],
    ['regions', '区域树(LTREE)', 'PK id', 'path 1集团→4城市'],
    ['legal_entities', '运营主体', 'PK id'],
    ['departments', '部门', 'PK id', 'FK legal_entity_id'],
    ['posts', '岗位', 'PK id', 'FK dept_id'],
    ['post_roles', '岗位-角色', 'FK post_id', 'FK role_id'],
    ['geo_country', '国家', 'PK alpha2'],
    ['geo_country_i18n', '国家多语', 'FK country_code'],
    ['geo_subdivision', '行政区树', 'PK code', 'FK country_code', 'sFK parent_code(树)'],
    ['geo_subdivision_i18n', '行政区多语', 'FK subdivision_code'],
    ['country_time_zone', '国家时区', 'FK country_code'],
    ['country_currency', '国家货币', 'FK country_code'],
    ['country_calling_code', '国家区号', 'FK country_code'],
  ]],
  ['地址·客户·产品', '#d5e8d4', [
    ['addresses', '地址树(LTREE)', 'PK id', 'sFK parent_id(树)', 'FK country_code/admin_code(geo)', 'geom'],
    ['customers', '客户', 'PK id', 'FK legal_entity_id', 'FK address_id', 'UQ phone', 'UQ customer_code(000085)'],
    ['customer_histories', '客户归属台账', 'FK customer_id', 'FK legal_entity_id/address_id'],
    ['customer_registrations', '客户注册申请', 'FK legal_entity_id/address_id', 'sFK customer_id(通过后回填)', 'sFK reviewer_account_id'],
    ['verifications', '统一实名(000059)', "sFK subject_type('customer'/'worker')+subject_id", 'method/real_name/id_card_no/result'],
    ['product_offers', '产品套餐', 'PK id', 'FK legal_entity_id'],
    ['product_price_histories', '产品调价台账', 'FK offer_id'],
    ['region_offers', '区域定价', 'PK id', 'FK offer_id'],
    ['region_price_histories', '区域调价台账', 'FK region_offer_id'],
    ['channels', '下单渠道目录', 'PK id', 'UQ code'],
  ]],
  ['订单·派工·计费·税务', '#ffe6cc', [
    ['orders', '订单(12环节)', 'PK id', 'UQ order_no', 'sFK customer_id/offer_id/address_id', 'sFK channel_id', 'FK legal_entity_id/region_id(000076)', 'legal_entity快照+region_path'],
    ['order_stages', '环节时间轴', 'FK order_id', 'stage 1~12'],
    ['dispatch_tickets', '派工工单', 'PK id', 'FK order_id UQ(1:1)', 'sFK worker_id/group_id'],
    ['dispatch_transfers', '改派记录', 'FK ticket_id', 'sFK from/to_worker_id'],
    ['complaints', '投诉', 'PK id', 'FK customer_id', 'sFK order_id'],
    ['scan_logs', '扫码日志', 'FK order_id'],
    ['dismantles', '拆机单', 'FK order_id'],
    ['activation_callbacks', '激活回调(环节11)', 'FK order_id'],
    ['order_ratings', '订单评价', 'PK id', 'FK customer_id', 'UQ order_no(软)'],
    ['bills', '账单', 'PK id', 'FK customer_id', '客户x账期 1:1'],
    ['payments', '缴费', 'PK id', 'FK bill_id(可空,000068)', 'FK customer_id(000068)'],
    ['arrears', '欠费态', 'FK customer_id UQ(1:1)'],
    ['stop_resume_tasks', '停复机任务', 'sFK customer_id/loid'],
    ['reconciliation_batches', '渠道对账批次', 'PK id', 'UQ batch_no'],
    ['arn_sequences', '税号序列', 'PK doc_type'],
    ['invoices', '发票(税局网关)', 'PK id', 'FK bill_id', 'customer快照', 'tax_no/tax_status'],
  ]],
  ['资产·四码', '#e1d5e7', [
    ['asset_batches', '入库批次', 'PK id', 'FK legal_entity_id'],
    ['tags', '电子标签', 'PK id', 'FK legal_entity_id', 'UQ tag_no/epc'],
    ['assets', '资产', 'PK id', 'FK batch_id', 'sFK tag_id/address_id', 'UQ asset_code'],
    ['asset_lifecycles', '资产状态轨迹', 'FK asset_id'],
    ['asset_assignments', '资产归属台账', 'FK asset_id', 'sFK worker/address'],
    ['stocktakes', '盘点任务', 'FK legal_entity_id'],
    ['replacements', '换新记录', 'sFK asset'],
    ['quad_links', '四码合一', 'PK id', 'sFK asset/customer/port/address', '部分UQ asset/port/address(000086)', 'customer 1:N(000088)', 'legal_entity快照'],
  ]],
  ['网络资源·监控·开通', '#f8cecc', [
    ['resources', 'OLT/分光器(树)', 'PK id', 'FK legal_entity_id/address_id', 'sFK parent_id(树)'],
    ['ports', '端口', 'PK id', 'FK resource_id/address_id', 'UQ port_code'],
    ['port_change_history', '端口变更历史', 'FK port_id'],
    ['reserve_records', '端口预占', 'sFK port_id/order_id'],
    ['resource_assignments', '资源归属台账', 'FK resource_id/legal_entity_id', 'sFK address_id'],
    ['transfers', '调拨单', 'sFK resource_id', 'region快照'],
    ['expansions', '扩容单', 'FK legal_entity_id'],
    ['device_metrics', '设备指标', 'sFK resource'],
    ['device_maintenances', '设备维护', 'sFK resource'],
    ['alarms', '告警', 'PK id', 'UQ alarm_no', 'sFK resource_id'],
    ['alarm_retest_tasks', '告警复测', 'sFK alarm'],
    ['lo_accounts', 'LO账号(1:1客户)', 'PK id', 'UQ loid', 'sFK customer_id/offer_id/qos_template_id'],
    ['qos_templates', 'QoS模板', 'PK id', 'FK legal_entity_id'],
    ['provision_templates', '开通模板', 'PK id', 'FK legal_entity_id'],
    ['provision_tasks', '开通任务', 'PK id', 'FK template_id', 'sFK loid'],
    ['provision_logs', '开通日志', 'sFK task_id'],
    ['cdrs', '话单CDR', 'sFK loid'],
    ['auth_logs', '认证日志', 'sFK loid'],
  ]],
  ['师傅域', '#fff2cc', [
    ['worker_groups', '班组', 'PK id', 'FK legal_entity_id', 'sFK leader_id'],
    ['workers', '师傅', 'PK id', 'FK group_id'],
    ['worker_group_memberships', '班组归属台账', 'FK worker_id/group_id', 'group_name快照'],
    ['worker_settings', '师傅设置(1:1)', 'FK worker_id UQ'],
    ['worker_messages', '师傅消息', 'FK worker_id'],
    ['worker_notices', '师傅公告', 'PK id(全局)'],
    ['worker_registrations', '师傅注册申请', 'FK group_id(可空,000089)', 'sFK worker_id(通过后回填)'],
    ['worker_performances', '绩效(师傅x月x班组)', 'FK worker_id/group_id', 'UQ(worker,period,group)'],
    ['worker_commissions', '提成(同粒度)', 'FK worker_id/group_id'],
    ['worker_schedules', '排班(同粒度)', 'FK worker_id/group_id'],
    ['worker_materials', '物料', 'FK worker_id/group_id'],
    ['worker_tools', '工具', 'FK worker_id/group_id'],
    ['material_items', '物料主档(000073)', 'PK id', 'UQ code(MI-*)'],
    ['material_tools', '工具主档(000073)', 'PK id', 'UQ code(TL-*)'],
    ['worker_attendance', '考勤打卡(000071)', 'FK worker_id', 'clock_type IN/OUT'],
    ['worker_safety_checks', '安全检查(000072)', 'FK worker_id'],
    ['worker_replace_logs', '换标签记录(000074)', 'FK worker_id', 'sFK ticket_no(工单号)'],
    ['worker_feedbacks', '评价反馈', 'FK worker_id/group_id'],
    ['asset_returns', '资产退回', 'FK worker_id/group_id'],
  ]],
  ['用户端(userdata)', '#d4e6f7', [
    ['user_accounts', '用户端账号', 'FK customer_id'],
    ['user_addresses', '用户地址簿', 'FK customer_id'],
    ['user_plans', '在用套餐', 'FK customer_id'],
    ['addons', '加装目录', 'PK addon_id'],
    ['addon_subscriptions', '加装订阅', 'FK customer_id/addon_id'],
    ['user_notify_settings', '通知偏好(1:1)', 'FK customer_id UQ'],
    ['user_faqs', 'FAQ(全局)', 'PK id'],
    ['user_messages', '用户消息', 'FK customer_id'],
    ['coupons', '优惠券', 'FK customer_id(可空)', 'PK coupon_id'],
    ['invite_config', '邀请配置(全局)', 'PK id'],
    ['user_usages', '用量', 'FK customer_id'],
    ['diy_guides', '自助指南(全局)', 'PK id'],
    ['agreements', '协议(全局)', 'PK id'],
    ['user_balances', '余额(1:1)', 'PK customer_id'],
    ['topup_denominations', '充值面额(全局)', 'PK id'],
    ['user_invoices', '用户发票申请', 'FK customer_id'],
    ['user_complaints', '用户投诉', 'FK customer_id'],
    ['user_verify_records', '核销记录', 'FK customer_id'],
    ['product_specs', '产品规格(全局)', 'PK id'],
    ['user_bill_items', '账单明细', 'FK customer_id'],
  ]],
  ['门户(portal)', '#fad7ac', [
    ['portal_sms_codes', '短信验证码', 'PK(phone,scene)'],
    ['portal_accounts', '门户账号(1:1客户)', 'PK phone', 'UQ customer_id(软,隔离空间可合成)'],
    ['portal_prefs', '偏好(1:1)', 'PK customer_id(软)'],
    ['portal_wallets', '钱包(1:1)', 'PK customer_id(软)'],
    ['portal_billing_prefs', '自动缴费(1:1)', 'PK customer_id(软)'],
    ['portal_messages', '门户消息', 'sFK customer_id'],
    ['portal_seq', '门户单号序列', 'PK kind'],
  ]],
  ['报表', '#f5f5f5', [
    ['report_snapshots', '报表快照', 'PK id', 'UQ(period,window_start)'],
  ]],
  ['ODN 无源网络(000075-81)', '#e6ffe6', [
    ['odn_region_code', 'ODN省码映射', 'PK prv_code(PHL001)', 'FK psgc_code→geo_subdivision'],
    ['odn_city_code', 'ODN城市前缀', 'PK(prv_code,city_prefix)', 'FK prv_code→odn_region_code', 'FK psgc_code→geo_subdivision'],
    ['odn_grid', '网格分区01~99', 'PK(prv,city,grid_code)', 'FK 复合→odn_city_code'],
    ['odn_facility', '无源设施(P/MH/TW/CLS/TBX)', 'PK code', 'FK 复合→odn_city_code', 'FK 网格(可空,TW/CLS/TBX=NULL)'],
    ['odn_cable_segment', '光缆段落', 'PK id', 'UQ(a_code,b_code)', 'sFK a/b_code→设施或核心设备码'],
    ['odn_fiber', '纤芯G01~99', 'PK(segment_id,g_no)', 'FK segment_id CASCADE'],
    ['odn_site', '局点(001~999)', 'PK(prv,city,site_no)', 'FK 复合→odn_city_code'],
    ['odn_device', '核心链路设备(SNW/OLT/ODF/...)', 'PK id', 'UQ(prv,city,code)', 'FK 复合→odn_city_code', 'sFK parent_id(树)', 'sFK site_no(可空,市域设备)'],
  ]],
];

// 关系边: [源表, 目标表, 标签, 样式key]
// 样式: fk=硬外键实线 sfk=软引用虚线
const EDGES = [
  ['role_permissions', 'roles', 'N:1', 'fk'], ['role_permissions', 'permissions', 'N:1', 'fk'],
  ['post_roles', 'posts', 'N:1', 'fk'], ['post_roles', 'roles', 'N:1', 'fk'],
  ['accounts', 'roles', 'N:1', 'fk'], ['accounts', 'legal_entities', '0..1', 'fk'],
  ['accounts', 'departments', '0..1', 'fk'], ['accounts', 'posts', '0..1', 'fk'],
  ['account_org_histories', 'accounts', 'N:1', 'fk'],
  ['audit_logs', 'accounts', '弱引用', 'sfk'],
  ['api_keys', 'accounts', 'N:1 CASCADE', 'fk'], ['import_tasks', 'accounts', 'N:1', 'fk'],
  ['departments', 'legal_entities', 'N:1', 'fk'], ['posts', 'departments', 'N:1', 'fk'],
  ['geo_country_i18n', 'geo_country', 'N:1', 'fk'], ['geo_subdivision', 'geo_country', 'N:1', 'fk'],
  ['geo_subdivision_i18n', 'geo_subdivision', 'N:1', 'fk'],
  ['country_time_zone', 'geo_country', 'N:1', 'fk'], ['country_currency', 'geo_country', 'N:1', 'fk'],
  ['country_calling_code', 'geo_country', 'N:1', 'fk'],
  ['addresses', 'geo_country', '锚点', 'sfk'], ['addresses', 'geo_subdivision', '一级行政区', 'sfk'],
  ['customers', 'legal_entities', 'N:1', 'fk'], ['customers', 'addresses', 'N:1', 'fk'],
  ['customer_histories', 'customers', 'N:1', 'fk'],
  ['verifications', 'customers', "subject_type='customer'", 'sfk'],
  ['verifications', 'workers', "subject_type='worker'", 'sfk'],
  ['customer_registrations', 'legal_entities', 'N:1', 'fk'], ['customer_registrations', 'addresses', 'N:1', 'fk'],
  ['product_offers', 'legal_entities', 'N:1', 'fk'],
  ['product_price_histories', 'product_offers', 'N:1', 'fk'],
  ['region_offers', 'product_offers', 'N:1', 'fk'],
  ['region_price_histories', 'region_offers', 'N:1', 'fk'],
  ['orders', 'customers', 'N:1', 'sfk'], ['orders', 'product_offers', 'N:1', 'sfk'],
  ['orders', 'channels', 'N:1', 'sfk'],
  ['orders', 'legal_entities', 'N:1(000076)', 'fk'], ['orders', 'regions', '装机区域(000076)', 'fk'],
  ['order_stages', 'orders', 'N:1', 'fk'],
  ['dispatch_tickets', 'orders', '1:1', 'fk'], ['dispatch_tickets', 'workers', '0..1', 'sfk'],
  ['dispatch_transfers', 'dispatch_tickets', 'N:1', 'fk'], ['dispatch_transfers', 'workers', 'from/to', 'sfk'],
  ['complaints', 'customers', 'N:1', 'fk'], ['complaints', 'orders', '0..1', 'sfk'],
  ['scan_logs', 'orders', 'N:1', 'fk'], ['dismantles', 'orders', 'N:1', 'fk'],
  ['activation_callbacks', 'orders', 'N:1', 'fk'],
  ['order_ratings', 'customers', 'N:1', 'fk'],
  ['bills', 'customers', 'N:1', 'fk'], ['payments', 'bills', '0..1(可空,000068)', 'fk'],
  ['payments', 'customers', 'N:1(000068)', 'fk'],
  ['arrears', 'customers', '1:1', 'fk'],
  ['stop_resume_tasks', 'customers', '软', 'sfk'],
  ['invoices', 'bills', 'N:1', 'fk'],
  ['asset_batches', 'legal_entities', 'N:1', 'fk'], ['tags', 'legal_entities', 'N:1', 'fk'],
  ['assets', 'asset_batches', 'N:1', 'fk'], ['assets', 'tags', '预绑定1:1', 'sfk'],
  ['asset_lifecycles', 'assets', 'N:1', 'fk'], ['asset_assignments', 'assets', 'N:1', 'fk'],
  ['stocktakes', 'legal_entities', 'N:1', 'fk'],
  ['quad_links', 'assets', '四码1', 'sfk'], ['quad_links', 'customers', '四码2', 'sfk'],
  ['quad_links', 'ports', '四码3', 'sfk'], ['quad_links', 'addresses', '四码4', 'sfk'],
  ['resources', 'legal_entities', 'N:1', 'fk'], ['resources', 'addresses', 'N:1', 'fk'],
  ['resources', 'resources', '父设备', 'sfk'],
  ['ports', 'resources', 'N:1', 'fk'], ['ports', 'addresses', 'N:1', 'fk'],
  ['port_change_history', 'ports', 'N:1', 'fk'],
  ['reserve_records', 'ports', 'N:1', 'sfk'], ['reserve_records', 'orders', 'N:1', 'sfk'],
  ['resource_assignments', 'resources', 'N:1', 'fk'],
  ['transfers', 'resources', '软', 'sfk'], ['expansions', 'legal_entities', 'N:1', 'fk'],
  ['alarms', 'resources', '软', 'sfk'], ['alarm_retest_tasks', 'alarms', '软', 'sfk'],
  ['lo_accounts', 'customers', '1:1', 'sfk'], ['lo_accounts', 'qos_templates', '软', 'sfk'],
  ['qos_templates', 'legal_entities', 'N:1', 'fk'],
  ['provision_templates', 'legal_entities', 'N:1', 'fk'],
  ['provision_tasks', 'provision_templates', 'N:1', 'fk'],
  ['provision_logs', 'provision_tasks', '软', 'sfk'],
  ['cdrs', 'lo_accounts', 'loid', 'sfk'], ['auth_logs', 'lo_accounts', 'loid', 'sfk'],
  ['worker_groups', 'legal_entities', 'N:1', 'fk'], ['worker_groups', 'workers', '组长', 'sfk'],
  ['workers', 'worker_groups', 'N:1', 'fk'],
  ['worker_group_memberships', 'workers', 'N:1', 'fk'], ['worker_group_memberships', 'worker_groups', 'N:1', 'fk'],
  ['worker_settings', 'workers', '1:1', 'fk'], ['worker_messages', 'workers', 'N:1', 'fk'],
  ['worker_registrations', 'worker_groups', 'N:1', 'fk'],
  ['worker_performances', 'workers', 'N:1', 'fk'], ['worker_performances', 'worker_groups', 'N:1', 'fk'],
  ['worker_commissions', 'workers', 'N:1', 'fk'], ['worker_schedules', 'workers', 'N:1', 'fk'],
  ['worker_materials', 'workers', 'N:1', 'fk'], ['worker_tools', 'workers', 'N:1', 'fk'],
  ['worker_attendance', 'workers', 'N:1', 'fk'], ['worker_safety_checks', 'workers', 'N:1', 'fk'],
  ['worker_replace_logs', 'workers', 'N:1', 'fk'],
  ['worker_feedbacks', 'workers', 'N:1', 'fk'], ['asset_returns', 'workers', 'N:1', 'fk'],
  ['asset_returns', 'assets', '软', 'sfk'],
  ['user_accounts', 'customers', 'N:1', 'fk'], ['user_addresses', 'customers', 'N:1', 'fk'],
  ['user_plans', 'customers', 'N:1', 'fk'], ['addon_subscriptions', 'customers', 'N:1', 'fk'],
  ['addon_subscriptions', 'addons', 'N:1', 'fk'], ['user_notify_settings', 'customers', '1:1', 'fk'],
  ['user_messages', 'customers', 'N:1', 'fk'], ['coupons', 'customers', '0..1', 'fk'],
  ['user_usages', 'customers', 'N:1', 'fk'], ['user_balances', 'customers', '1:1', 'fk'],
  ['user_invoices', 'customers', 'N:1', 'fk'], ['user_complaints', 'customers', 'N:1', 'fk'],
  ['user_verify_records', 'customers', 'N:1', 'fk'], ['user_bill_items', 'customers', 'N:1', 'fk'],
  ['portal_accounts', 'customers', '1:1软', 'sfk'], ['portal_messages', 'customers', '软', 'sfk'],
  ['odn_region_code', 'geo_subdivision', 'psgc映射', 'fk'],
  ['odn_city_code', 'odn_region_code', 'N:1', 'fk'], ['odn_city_code', 'geo_subdivision', 'psgc映射', 'fk'],
  ['odn_grid', 'odn_city_code', '复合FK', 'fk'],
  ['odn_facility', 'odn_city_code', '复合FK', 'fk'], ['odn_facility', 'odn_grid', '0..1', 'fk'],
  ['odn_fiber', 'odn_cable_segment', 'N:1 CASCADE', 'fk'],
  ['odn_cable_segment', 'odn_facility', 'A/B端编码', 'sfk'],
  ['odn_site', 'odn_city_code', '复合FK', 'fk'],
  ['odn_device', 'odn_city_code', '复合FK', 'fk'], ['odn_device', 'odn_device', '父设备', 'sfk'],
  ['odn_device', 'odn_site', '0..1', 'sfk'],
];

const TABLE_W = 230;
const ROW_H = 15;
const HEAD_H = 26;
const GAP_Y = 26;
const GROUP_PAD = 40;
const GROUP_TITLE = 34;
const GROUP_GAP = 60;

function tableHeight(t) { return HEAD_H + t.length * ROW_H + 8; }

const nodes = new Map(); // table -> {x,y,w,h,group}
const groups = [];
let x = 40;
for (const [name, color, tables] of GROUPS) {
  const specs = tables.map(([tbl, ...rows]) => [tbl, rows]);
  const h = GROUP_TITLE + specs.reduce((s, [, r]) => s + tableHeight(r) + GAP_Y, 0) + GROUP_PAD;
  groups.push({ name, color, x, y: 40, w: TABLE_W + GROUP_PAD * 2, h, specs });
  let ty = 40 + GROUP_TITLE;
  for (const [tbl, rows] of specs) {
    nodes.set(tbl, { x: x + GROUP_PAD, y: ty, w: TABLE_W, h: tableHeight(rows), rows, group: name, color });
    ty += tableHeight(rows) + GAP_Y;
  }
  x += TABLE_W + GROUP_PAD * 2 + GROUP_GAP;
}

function esc(s) {
  return String(s).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

const cells = [];
for (const g of groups) {
  cells.push(`<mxCell id="grp-${esc(g.name)}" value="${esc(g.name)}" style="rounded=1;whiteSpace=wrap;html=1;fillColor=none;strokeColor=#666666;dashed=1;verticalAlign=top;fontSize=13;fontStyle=1;" vertex="1" parent="1"><mxGeometry x="${g.x}" y="${g.y}" width="${g.w}" height="${g.h}" as="geometry"/></mxCell>`);
}
for (const [tbl, n] of nodes) {
  const body = n.rows.map((r) => {
    const line = r.startsWith('PK') || r.startsWith('UQ') || r.startsWith('FK') || r.startsWith('sFK')
      ? `<font color="${r.startsWith('PK') ? '#9673a6' : r.startsWith('FK') ? '#d79b00' : '#999999'}">${esc(r)}</font>`
      : `<font color="#888888">${esc(r)}</font>`;
    return line;
  }).join('<br>');
  const label = `<b>${esc(tbl)}</b><br>${body}`;
  cells.push(`<mxCell id="t-${esc(tbl)}" value="${esc(label)}" style="rounded=0;whiteSpace=wrap;html=1;fillColor=${n.color};strokeColor=#666666;fontSize=11;align=left;spacingLeft=8;verticalAlign=top;spacingTop=4;" vertex="1" parent="1"><mxGeometry x="${n.x}" y="${n.y}" width="${n.w}" height="${n.h}" as="geometry"/></mxCell>`);
}
const STYLE = {
  fk: 'edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;strokeColor=#d79b00;strokeWidth=1.4;fontSize=9;fontColor=#d79b00;endArrow=ERoneToOne;startArrow=ERmany;',
  sfk: 'edgeStyle=orthogonalEdgeStyle;rounded=0;html=1;dashed=1;strokeColor=#999999;strokeWidth=1;fontSize=9;fontColor=#999999;endArrow=open;startArrow=open;',
};
for (const [src, dst, label, kind] of EDGES) {
  if (!nodes.has(src) || !nodes.has(dst)) { console.error(`missing node: ${src}->${dst}`); process.exitCode = 1; continue; }
  cells.push(`<mxCell id="e-${esc(src)}-${esc(dst)}-${esc(label)}" value="${esc(label)}" style="${STYLE[kind]}" edge="1" parent="1" source="t-${esc(src)}" target="t-${esc(dst)}"><mxGeometry relative="1" as="geometry"/></mxCell>`);
}
// 图例
cells.push(`<mxCell id="legend" value="&lt;b&gt;图例&lt;/b&gt;&lt;br&gt;实线橙 = 硬外键(REFERENCES)&lt;br&gt;虚线灰 = 软引用(无FK约束)&lt;br&gt;PK 紫 / FK 橙 / sFK 灰&lt;br&gt;分组虚线框 = 业务域&lt;br&gt;生成: scripts/gen-er-drawio.mjs" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#ffffff;strokeColor=#666666;fontSize=10;align=left;spacingLeft=8;" vertex="1" parent="1"><mxGeometry x="40" y="900" width="230" height="110" as="geometry"/></mxCell>`);

const xml = `<mxfile>
  <diagram id="autolayout" name="Page-1">
    <mxGraphModel dx="1400" dy="900" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="0" pageScale="1" math="0" shadow="0">
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>
        ${cells.join('\n        ')}
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
`;
writeFileSync(join(root, 'docs/boss-entities-er.drawio'), xml);
const total = nodes.size;
console.log(`written docs/boss-entities-er.drawio: ${total} tables, ${EDGES.length} edges, ${groups.length} groups`);
