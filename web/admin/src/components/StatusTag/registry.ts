// 状态枚举注册表:与 server-ts/src/enums.ts / docs/contract/terms.md 严格对齐。
// 颜色语义集中定义一次(对齐原型各页 st-* 语义):绿=正常完成,蓝=进行中,橙=待处理,红=异常/阻断,灰=中性/未知。

export interface TagMeta {
  label: string
  color: string
}

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

type Registry = Partial<Record<StatusDomain, Record<string, TagMeta>>>

/** 域 → 值 → {label, color}。 */
export const REGISTRY: Registry = {
  order: {
    PENDING: { label: '待核查', color: ORANGE },
    RESERVED: { label: '已预占', color: BLUE },
    INSTALLING: { label: '装维中', color: BLUE },
    DONE: { label: '已完成', color: GREEN },
    CANCELLED: { label: '已取消', color: GRAY },
  },
  port: {
    IDLE: { label: '空闲', color: GREEN },
    RESERVED: { label: '已预占', color: ORANGE },
    USED: { label: '已占用', color: BLUE },
    DISABLED: { label: '已停用', color: GRAY },
  },
  asset: {
    IN_STOCK: { label: '在库', color: GREEN },
    DEPLOYED: { label: '在用', color: BLUE },
    MAINTENANCE: { label: '维护中', color: ORANGE },
    SCRAPPED: { label: '已报废', color: GRAY },
  },
  bill: {
    UNPAID: { label: '未支付', color: ORANGE },
    PAID: { label: '已支付', color: GREEN },
    OVERDUE: { label: '已逾期', color: RED },
  },
  payment: {
    SUCCESS: { label: '成功', color: GREEN },
    FAILED: { label: '失败', color: RED },
    REFUNDED: { label: '已退款', color: GRAY },
  },
  service: {
    ACTIVE: { label: '在服', color: GREEN },
    ARREARS: { label: '欠费', color: RED },
    SUSPENDED: { label: '停机', color: GRAY },
  },
  realName: {
    VERIFIED: { label: '已实名', color: GREEN },
    PENDING: { label: '待补登', color: ORANGE },
  },
  recon: {
    DIFF_PENDING: { label: '差异挂起', color: ORANGE },
    SETTLED: { label: '已平账', color: GREEN },
  },
  ledgerRecon: {
    UNPAID: { label: '未收', color: RED },
    PARTIAL: { label: '部分收', color: ORANGE },
    OVERPAID: { label: '多收', color: PURPLE },
    REFUNDED: { label: '退款未补', color: RED },
    PAID_NO_INVOICE: { label: '已收未开票', color: ORANGE },
    MATCH: { label: '一致', color: GREEN },
  },
  reserve: {
    HELD: { label: '预占中', color: ORANGE },
    RELEASED: { label: '已释放', color: GRAY },
    CONSUMED: { label: '已占用', color: BLUE },
  },
  product: {
    DRAFT: { label: '草稿', color: GRAY },
    PUBLISHED: { label: '已发布', color: GREEN },
    OFFLINE: { label: '已下线', color: GRAY },
  },
  quad: {
    LINKED: { label: '已关联', color: GREEN },
    CONFLICT: { label: '冲突', color: RED },
    UNLINKED: { label: '未关联', color: GRAY },
  },
  tag: {
    UNBOUND: { label: '未绑定', color: GRAY },
    BOUND: { label: '已绑定', color: GREEN },
    DISABLED: { label: '已停用', color: RED },
  },
  resource: {
    ONLINE: { label: '在线', color: GREEN },
    OFFLINE: { label: '离线', color: GRAY },
    FAULT: { label: '故障', color: RED },
  },
  loAccount: {
    ACTIVE: { label: '正常', color: GREEN },
    SUSPENDED: { label: '暂停', color: ORANGE },
    CLOSED: { label: '已销户', color: GRAY },
  },
  ticket: {
    PENDING: { label: '待处理', color: ORANGE },
    DOING: { label: '处理中', color: BLUE },
    DONE: { label: '已完成', color: GREEN },
    CANCELED: { label: '已取消', color: GRAY },
  },
  task: {
    PENDING: { label: '待执行', color: ORANGE },
    DOING: { label: '执行中', color: BLUE },
    DONE: { label: '已完成', color: GREEN },
    FAILED: { label: '失败', color: RED },
  },
  complaint: {
    OPEN: { label: '待受理', color: RED },
    PROCESSING: { label: '处理中', color: BLUE },
    CLOSED: { label: '已关闭', color: GREEN },
  },
  scan: {
    MATCH: { label: '匹配', color: GREEN },
    MISMATCH: { label: '不匹配', color: RED },
    OFFLINE_CACHED: { label: '离线缓存', color: PURPLE },
  },
  alarmLevel: {
    CRITICAL: { label: '紧急', color: RED },
    WARNING: { label: '警告', color: ORANGE },
    INFO: { label: '提示', color: BLUE },
  },
  alarmStatus: {
    OPEN: { label: '未确认', color: RED },
    ACKED: { label: '已确认', color: ORANGE },
    CLOSED: { label: '已关闭', color: GRAY },
  },
  maintPriority: {
    MUST_REPLACE: { label: '必须更换', color: RED },
    SUGGEST: { label: '建议更换', color: ORANGE },
    WATCH: { label: '观察', color: BLUE },
  },
  accountStatus: {
    '1': { label: '启用', color: GREEN },
    '0': { label: '停用', color: GRAY },
  },
  message: {
    INFO: { label: '信息', color: BLUE },
    WARN: { label: '警告', color: ORANGE },
    URGENT: { label: '紧急', color: RED },
  },
  backupStatus: {
    running: { label: '执行中', color: BLUE },
    succeeded: { label: '成功', color: GREEN },
    failed: { label: '失败', color: RED },
  },
}
