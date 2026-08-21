// 菜单定义:结构照抄 docs/admin/menu.js(13 组 45 页),路由 path 与 href 同名映射。
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
  { id: 'base', label: '基础配置', items: [
    { key: 'account', label: '账号与角色', path: '/base/account' },
    { key: 'address', label: '地址层级', path: '/base/address' },
    { key: 'geo', label: '国家与行政区划', path: '/base/geo' },
    { key: 'params', label: '业务参数', path: '/base/params' },
    { key: 'authconfig', label: '认证配置', path: '/base/authconfig' },
    { key: 'smsconfig', label: '短信配置', path: '/base/smsconfig' },
    { key: 'pushconfig', label: '推送配置', path: '/base/pushconfig' },
    { key: 'realidconfig', label: '实名核验配置', path: '/base/realidconfig' },
    { key: 'storageconfig', label: 'MinIO 存储配置', path: '/base/storageconfig' },
    { key: 'servers', label: '服务端配置', path: '/base/servers' },
    { key: 'audit', label: '审计日志', path: '/base/audit' },
    { key: 'importer', label: '数据导入中心', path: '/base/importer' },
    { key: 'backup', label: '数据备份迁移', path: '/base/backup' },
  ]},
  { id: 'org', label: '组织与权限', items: [
    { key: 'company', label: '子公司/法人', path: '/org/company' },
    { key: 'department', label: '部门管理', path: '/org/department' },
    { key: 'post', label: '岗位管理', path: '/org/post' },
    { key: 'region', label: '经营区域', path: '/org/region' },
    { key: 'menuperm', label: '菜单权限', path: '/org/menuperm' },
    { key: 'datascope', label: '数据权限', path: '/org/datascope' },
    { key: 'apikey', label: 'API Key', path: '/org/apikey' },
  ]},
  { id: 'bss', label: '客户与资费', items: [
    { key: 'customer', label: '客户档案', path: '/bss/customer' },
    { key: 'product', label: '产品资费', path: '/bss/product' },
    { key: 'user', label: '用户列表', path: '/bss/user' },
    { key: 'userdata', label: '用户端配置', path: '/bss/userdata' },
  ]},
  { id: 'billing', label: '计费与账务', items: [
    { key: 'billing', label: '出账管理', path: '/billing/billing' },
    { key: 'payment', label: '缴费管理', path: '/billing/payment' },
    { key: 'arrears', label: '欠费停复机', path: '/billing/arrears' },
    { key: 'stopsrv', label: '停复机执行', path: '/billing/stopsrv' },
    { key: 'paycheck', label: '渠道对账', path: '/billing/paycheck' },
  ]},
  { id: 'ams', label: '资产与标签', items: [
    { key: 'asset', label: '资产台账', path: '/ams/asset' },
    { key: 'tag', label: '电子标签', path: '/ams/tag' },
    { key: 'stock', label: '盘点管理', path: '/ams/stock' },
    { key: 'replace', label: '设备更换单', path: '/ams/replace' },
  ]},
  { id: 'oss', label: '网络资源', items: [
    { key: 'resource', label: '端口台账', path: '/oss/resource' },
    { key: 'odn', label: 'ODN 无源网络', path: '/oss/odn' },
    { key: 'reserve', label: '预占与释放', path: '/oss/reserve' },
    { key: 'transfer', label: '跨区域调配', path: '/oss/transfer' },
    { key: 'device', label: 'OLT 设备', path: '/oss/device' },
    { key: 'loaccount', label: '认证账号', path: '/oss/loaccount' },
    { key: 'expand', label: '扩容申请', path: '/oss/expand' },
  ]},
  { id: 'boss', label: '订单与工单', items: [
    { key: 'order', label: '订单管理', path: '/boss/order' },
    { key: 'worker', label: '师傅管理', path: '/boss/worker' },
    { key: 'worker-reg', label: '师傅注册审核', path: '/boss/worker-reg' },
    { key: 'worker-ops', label: '师傅端内容', path: '/boss/worker-ops' },
    { key: 'message', label: '消息中心', path: '/boss/message' },
    { key: 'dispatch', label: '派单管理', path: '/boss/dispatch' },
    { key: 'dismantle', label: '拆机管理', path: '/boss/dismantle' },
    { key: 'complaint', label: '报障与投诉', path: '/boss/complaint' },
    { key: 'callback', label: '激活回调', path: '/boss/callback' },
  ]},
  { id: 'quad', label: '四码合一', items: [
    { key: 'quadlink', label: '关联查询', path: '/quad/quadlink' },
    { key: 'check', label: '对账与告警', path: '/quad/check' },
    { key: 'scanlog', label: '扫码绑定记录', path: '/quad/scanlog' },
  ]},
  { id: 'provision', label: '配置下发', items: [
    { key: 'provision', label: '下发任务', path: '/provision/provision' },
    { key: 'template', label: '配置模板', path: '/provision/template' },
    { key: 'provlog', label: '下发日志', path: '/provision/provlog' },
  ]},
  { id: 'alarm', label: '告警中心', items: [
    { key: 'alarm', label: '告警列表', path: '/alarm/alarm' },
  ]},
  { id: 'aaa', label: '认证计费', items: [
    { key: 'aaalog', label: '话单与认证日志', path: '/aaa/aaalog' },
  ]},
  { id: 'intel', label: '数字孪生与经营', items: [
    { key: 'gis', label: 'GIS 地图', path: '/intel/gis' },
    { key: 'analytics', label: '经营分析', path: '/intel/analytics' },
    { key: 'report', label: '报告中心', path: '/intel/report' },
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
