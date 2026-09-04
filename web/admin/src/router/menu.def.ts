// 菜单定义:按页面实际内容重划分组(2026-08-27 重组 16 组 81 页;
// 2026-08-28 增 procurement-install-gis 联动功能 +3 页 = 16 组 84 页;决策见
// docs/notes/adopted/2026-08-27-sidebar-regroup.md 与
// docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md)。
// 页面 key 与 path 保持原值,权限码 menu:<key>、路由生成、收藏均不受分组调整影响;
// 分组语义以能力域为准
// (docs/contract/domain-map.md A 列),原型分组留档 docs/admin/menu.js。
export interface MenuItem {
  key: string
  label: string
  path: string
}

export interface MenuGroup {
  id: string
  label: string
  items: MenuItem[]
}

export const MENU_GROUPS: MenuGroup[] = [
  { id: 'overview', label: '运营总览', items: [
    { key: 'dashboard', label: '工作台', path: '/dashboard' },
  ]},
  { id: 'bss', label: '客户与资费', items: [
    { key: 'customer', label: '客户档案', path: '/bss/customer' },
    { key: 'onboarding', label: '开户工作台', path: '/bss/onboarding' },
    { key: 'user', label: '用户列表', path: '/bss/user' },
    { key: 'userdata', label: '用户端配置', path: '/bss/userdata' },
    { key: 'realname-review', label: '实名审核中心', path: '/base/realname-review' },
    { key: 'product', label: '产品资费', path: '/bss/product' },
    { key: 'marketing', label: '营销与积分规则', path: '/bss/marketing' },
    { key: 'marketing-recon', label: '券积分对账', path: '/bss/marketing-recon' },
  ]},
  { id: 'billing', label: '计费与账务', items: [
    { key: 'billing', label: '出账管理', path: '/billing/billing' },
    { key: 'payment', label: '缴费管理', path: '/billing/payment' },
    { key: 'arrears', label: '欠费停复机', path: '/billing/arrears' },
    { key: 'collection-tasks', label: '催收任务队列', path: '/billing/collection-tasks' },
    { key: 'stopsrv', label: '停复机执行', path: '/billing/stopsrv' },
    { key: 'paycheck', label: '渠道对账', path: '/billing/paycheck' },
    { key: 'daily-close', label: '柜台日结', path: '/billing/daily-close' },
  ]},
  { id: 'boss', label: '订单与履约', items: [
    { key: 'order', label: '订单管理', path: '/boss/order' },
    { key: 'dispatch', label: '派单管理', path: '/boss/dispatch' },
    { key: 'dismantle', label: '拆机管理', path: '/boss/dismantle' },
    { key: 'callback', label: '激活回调', path: '/boss/callback' },
    { key: 'complaint', label: '报障与投诉', path: '/boss/complaint' },
    { key: 'feedback', label: '回访评价', path: '/boss/feedback' },
    { key: 'service-metrics', label: '客户服务指标', path: '/boss/service-metrics' },
    { key: 'install-board', label: '施工看板', path: '/boss/install-board' },
  ]},
  { id: 'worker', label: '装维管理', items: [
    { key: 'worker', label: '师傅管理', path: '/boss/worker' },
    { key: 'worker-reg', label: '师傅注册审核', path: '/boss/worker-reg' },
    { key: 'worker-ops', label: '师傅端内容', path: '/boss/worker-ops' },
  ]},
  { id: 'ams', label: '资产与标签', items: [
    { key: 'asset', label: '资产台账', path: '/ams/asset' },
    { key: 'tag', label: '电子标签', path: '/ams/tag' },
    { key: 'stock', label: '盘点管理', path: '/ams/stock' },
    { key: 'replace', label: '设备更换单', path: '/ams/replace' },
    { key: 'purchase', label: '采购单', path: '/ams/purchase' },
    { key: 'inventory', label: '库存查询', path: '/ams/inventory' },
  ]},
  { id: 'oss', label: '网络资源', items: [
    { key: 'resource', label: '端口台账', path: '/oss/resource' },
    { key: 'odn', label: 'ODN 无源网络', path: '/oss/odn' },
    { key: 'device', label: 'OLT 设备', path: '/oss/device' },
    { key: 'reserve', label: '预占与释放', path: '/oss/reserve' },
    { key: 'transfer', label: '跨区域调配', path: '/oss/transfer' },
    { key: 'expand', label: '扩容申请', path: '/oss/expand' },
    { key: 'loaccount', label: '认证账号', path: '/oss/loaccount' },
  ]},
  { id: 'provision', label: '配置下发', items: [
    { key: 'provision', label: '下发任务', path: '/provision/provision' },
    { key: 'template', label: '配置模板', path: '/provision/template' },
    { key: 'provlog', label: '下发日志', path: '/provision/provlog' },
  ]},
  { id: 'quad', label: '四码合一', items: [
    { key: 'quadlink', label: '关联查询', path: '/quad/quadlink' },
    { key: 'check', label: '对账与告警', path: '/quad/check' },
    { key: 'scanlog', label: '扫码绑定记录', path: '/quad/scanlog' },
  ]},
  { id: 'aaa', label: '认证与告警', items: [
    { key: 'aaadashboard', label: 'AAA 运行总览', path: '/aaa/dashboard' },
    { key: 'aaalog', label: '话单与认证日志', path: '/aaa/aaalog' },
    { key: 'alarm', label: '告警列表', path: '/alarm/alarm' },
  ]},
  { id: 'cms', label: '内容与消息', items: [
    { key: 'site', label: '官网内容', path: '/boss/site' },
    { key: 'site-cats', label: '官网分类', path: '/boss/site/cats' },
    { key: 'knowledge', label: '知识库', path: '/boss/knowledge' },
    { key: 'release', label: '版本发布', path: '/boss/release' },
    { key: 'message', label: '消息中心', path: '/boss/message' },
  ]},
  { id: 'intel', label: '数字孪生与经营', items: [
    { key: 'gis', label: 'GIS 地图', path: '/intel/gis' },
    { key: 'analytics', label: '经营分析', path: '/intel/analytics' },
    { key: 'report', label: '报告中心', path: '/intel/report' },
    { key: 'monthly', label: '月度填报', path: '/intel/monthly' },
  ]},
  { id: 'org', label: '组织与权限', items: [
    { key: 'account', label: '账号与角色', path: '/base/account' },
    { key: 'company', label: '子公司/法人', path: '/org/company' },
    { key: 'staff', label: '组织架构与人员', path: '/org/staff' },
    { key: 'department', label: '部门管理', path: '/org/department' },
    { key: 'post', label: '岗位管理', path: '/org/post' },
    { key: 'region', label: '经营区域', path: '/org/region' },
    { key: 'menuperm', label: '菜单权限', path: '/org/menuperm' },
    { key: 'datascope', label: '数据权限', path: '/org/datascope' },
    { key: 'partner', label: '入驻申请审核', path: '/org/partner' },
  ]},
  { id: 'channel', label: '集成与开放', items: [
    { key: 'authconfig', label: '认证配置', path: '/base/authconfig' },
    { key: 'smsconfig', label: '短信配置', path: '/base/smsconfig' },
    { key: 'pushconfig', label: '推送配置', path: '/base/pushconfig' },
    { key: 'realidconfig', label: '实名核验配置', path: '/base/realidconfig' },
    { key: 'stripeconfig', label: '支付配置', path: '/base/stripeconfig' },
    { key: 'storageconfig', label: 'MinIO 存储配置', path: '/base/storageconfig' },
    { key: 'apikey', label: 'API Key', path: '/org/apikey' },
    { key: 'openplat', label: '开放平台', path: '/org/openplat' },
    { key: 'apidocs', label: 'API 文档', path: '/base/apidocs' },
  ]},
  { id: 'system', label: '系统管理', items: [
    { key: 'license', label: '系统授权', path: '/base/license' },
    { key: 'params', label: '业务参数', path: '/base/params' },
    { key: 'servers', label: '服务端配置', path: '/base/servers' },
    { key: 'address', label: '地址层级', path: '/base/address' },
    { key: 'geo', label: '国家与行政区划', path: '/base/geo' },
    { key: 'audit', label: '审计日志', path: '/base/audit' },
    { key: 'crashlogs', label: '崩溃日志', path: '/base/crashlogs' },
    { key: 'importer', label: '数据导入中心', path: '/base/importer' },
    { key: 'backup', label: '数据备份迁移', path: '/base/backup' },
  ]},
  { id: 'partner', label: '企业工作台', items: [
    { key: 'partner-home', label: '我的企业', path: '/partner/home' },
    { key: 'partner-staff', label: '员工管理', path: '/partner/staff' },
    { key: 'partner-orders', label: '企业订单', path: '/partner/orders' },
  ]},
]

/** 页面扁平索引:key → {item, groupId}。 */
export const PAGE_BY_KEY: Map<string, { item: MenuItem; groupId: string }> = new Map(
  MENU_GROUPS.flatMap((g) => g.items.map((item) => [item.key, { item, groupId: g.id }])),
)

/** 路由 path → 页面 key 索引(直访未授权 URL 时判 403)。 */
export const KEY_BY_PATH: Map<string, string> = new Map(
  MENU_GROUPS.flatMap((g) => g.items.map((item) => [item.path, item.key])),
)

/** 侧栏激活判定:精确路径激活;深层路由仅当其不是其它菜单项的完整路径时算"同页"
 *  (如 /boss/site/new 高亮官网内容,而 /boss/site/cats 自身是菜单项,不高亮官网内容)。 */
export function isNavActive(to: string, pathname: string): boolean {
  if (pathname === to) return true
  if (!pathname.startsWith(`${to}/`)) return false
  return !KEY_BY_PATH.has(pathname)
}
