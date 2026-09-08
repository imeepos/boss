// 状态枚举注册表:与 server-ts/src/enums.ts / docs/contract/terms.md 严格对齐。
// 颜色语义集中定义一次(对齐原型各页 st-* 语义):绿=正常完成,蓝=进行中,橙=待处理,红=异常/阻断,灰=中性/未知。

const GREEN = '#52c41a'
const BLUE = '#1677ff'
const ORANGE = '#fa8c16'
const RED = '#e54545'
const GRAY = '#8c8c8c'
const PURPLE = '#722ed1'

export type StatusDomain =
  | 'order' | 'port' | 'asset' | 'bill' | 'payment' | 'service'
  | 'product' | 'quad' | 'tag' | 'resource' | 'loAccount' | 'ticket'
  | 'task' | 'complaint' | 'scan' | 'alarmLevel' | 'alarmStatus' | 'maintPriority'
  | 'accountStatus' | 'message' | 'realName' | 'recon' | 'reserve' | 'ledgerRecon'
  | 'backupStatus'
  | 'procurement' | 'receipt' | 'installLog' | 'userdata'
  | 'aaaSession'
  | 'callbackResult'

type Registry = Partial<Record<StatusDomain, Record<string, string>>>

/** 域 → 值 → 色名。标签(i18n)在 locales *.ts common.statusTags 平铺键 "domain.VALUE",由 StatusTag.test 完备性测试双向锁定。 */
export const REGISTRY: Registry = {
  order: {
    PENDING: ORANGE,
    RESERVED: BLUE,
    INSTALLING: BLUE,
    DONE: GREEN,
    CANCELLED: GRAY,
  },
  port: {
    IDLE: GREEN,
    RESERVED: ORANGE,
    USED: BLUE,
    DISABLED: GRAY,
  },
  asset: {
    IN_STOCK: GREEN,
    DEPLOYED: BLUE,
    MAINTENANCE: ORANGE,
    SCRAPPED: GRAY,
  },
  bill: {
    UNPAID: ORANGE,
    PAID: GREEN,
    OVERDUE: RED,
  },
  payment: {
    SUCCESS: GREEN,
    FAILED: RED,
    REFUNDED: GRAY,
  },
  service: {
    ACTIVE: GREEN,
    ARREARS: RED,
    SUSPENDED: GRAY,
  },
  realName: {
    VERIFIED: GREEN,
    PENDING: ORANGE,
  },
  recon: {
    DIFF_PENDING: ORANGE,
    SETTLED: GREEN,
  },
  ledgerRecon: {
    UNPAID: RED,
    PARTIAL: ORANGE,
    OVERPAID: PURPLE,
    REFUNDED: RED,
    PAID_NO_INVOICE: ORANGE,
    MATCH: GREEN,
  },
  reserve: {
    HELD: ORANGE,
    RELEASED: GRAY,
    CONSUMED: BLUE,
  },
  product: {
    DRAFT: GRAY,
    PUBLISHED: GREEN,
    OFFLINE: GRAY,
  },
  quad: {
    LINKED: GREEN,
    CONFLICT: RED,
    UNLINKED: GRAY,
  },
  tag: {
    UNBOUND: GRAY,
    BOUND: GREEN,
    DISABLED: RED,
  },
  resource: {
    ONLINE: GREEN,
    OFFLINE: GRAY,
    FAULT: RED,
  },
  loAccount: {
    ACTIVE: GREEN,
    SUSPENDED: ORANGE,
    CLOSED: GRAY,
  },
  aaaSession: {
    ONLINE: GREEN,
    PENDING_OFFLINE: ORANGE,
  },
  // 激活回调结果(orders 第 11 环节 activation_callbacks.result,§6.3 registry 收编)
  callbackResult: {
    SUCCESS: GREEN,
    FAILED: RED,
  },
  ticket: {
    PENDING: ORANGE,
    DOING: BLUE,
    DONE: GREEN,
    CANCELED: GRAY,
  },
  task: {
    PENDING: ORANGE,
    DOING: BLUE,
    DONE: GREEN,
    FAILED: RED,
  },
  complaint: {
    OPEN: RED,
    PROCESSING: BLUE,
    CLOSED: GREEN,
  },
  scan: {
    MATCH: GREEN,
    MISMATCH: RED,
    OFFLINE_CACHED: PURPLE,
  },
  alarmLevel: {
    CRITICAL: RED,
    WARNING: ORANGE,
    INFO: BLUE,
  },
  alarmStatus: {
    OPEN: RED,
    ACKED: ORANGE,
    CLOSED: GRAY,
  },
  maintPriority: {
    MUST_REPLACE: RED,
    SUGGEST: ORANGE,
    WATCH: BLUE,
  },
  accountStatus: {
    '1': GREEN,
    '0': GRAY,
  },
  message: {
    INFO: BLUE,
    WARN: ORANGE,
    URGENT: RED,
  },
  backupStatus: {
    running: BLUE,
    succeeded: GREEN,
    failed: RED,
  },
  procurement: {
    DRAFT: GRAY,
    SUBMITTED: BLUE,
    PARTIAL: ORANGE,
    RECEIVED: GREEN,
    CANCELLED: GRAY,
  },
  receipt: {
    DRAFT: GRAY,
    CONFIRMED: GREEN,
    REJECTED: RED,
  },
  installLog: {
    OPEN: ORANGE,
    COMPLETED: GREEN,
    REJECTED: RED,
  },
  // userdata(用户端配置:增值服务 on/off;优惠券迁移 000102 后 ISSUED/USED/DISABLED,
  // 小写 disabled 为 userdata_disable 动作现写值;布尔配置映射 active/disabled)
  userdata: {
    on: GREEN,
    off: GRAY,
    active: GREEN,
    disabled: GRAY,
    ISSUED: GREEN,
    USED: GRAY,
    DISABLED: GRAY,
    EXPIRED: ORANGE,
  },
}
