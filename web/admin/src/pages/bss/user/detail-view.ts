// 用户详情视图规格(纯数据,无 React/i18n 依赖):页签 → 分区 → 列。
// 数据源契约:GET /users/{customerId} 聚合(api/openapi/admin/userdata.yaml),
// 14 段数组 + 主档/notify;列名对齐 docs/contract/fields.md。
import type { StatusDomain } from '../../../components/StatusTag/registry'

export type ColSpec =
  | { kind: 'text' }
  | { kind: 'time' }
  | { kind: 'fee' }                                  // 元口径 NUMERIC(12,2)
  | { kind: 'feeCents' }                             // 分口径 BIGINT(coupons.amount)
  | { kind: 'bool' }
  | { kind: 'tag'; domain: StatusDomain }            // StatusTag 注册域
  | { kind: 'enum'; map: Record<string, string> }    // 值 → dFields 文案键

export interface DetailCol {
  /** 聚合行字段名(JSON key)。 */
  key: string
  /** 列名文案键(dFields 字典)。 */
  k: string
  spec?: ColSpec
}export interface DetailSection {
  /** 聚合返回的数组段名(= userPage.sectionNames 的键);notifyPrefs 为特殊卡片段。 */
  key: string
  cols: DetailCol[]
  /** 分段列表展示上限(超限折叠,抽屉内可展开查看全部);缺省 SECTION_LIMIT。 */
  limit?: number
}

export interface DetailTab {
  key: string
  sections: DetailSection[]
}

/** 特殊卡片段:非表格,不经 sectionNames 渲染(detail-drawer 直读 sectionNames.notify)。
 * 门禁 walker 跳过它——它不是字典引用键。 */
export const NOTIFY_CARD_KEY = 'notifyPrefs'

/** 分段列表展示上限(默认 5 行,超限折叠、展开查看全部——'有节制地展示')。
 * 覆盖策略:plans 收缩为 3(当前在用套餐),usages 放宽到 6(月度流量粒度)。 */
export const SECTION_LIMIT = 5

// 订单环节文案键(terms.md 第 1 节 12 环节)。
const STAGE: Record<string, string> = Object.fromEntries(
  Array.from({ length: 12 }, (_, i) => [String(i + 1), `dStage${i + 1}`]),
)

// 报障类型 → 文案键(faults 段,后端已 strip "用户报障: " 前缀)。
// 两套口径:用户端 POST /faults(no_internet/slow/ont_fault/other)+ 装维域故障码
// (SINGLE_OUTAGE 等,complaint-type-map.md);未知值回退显示原文。
const FAULT_TYPE: Record<string, string> = {
  no_internet: 'dFaultNoInternet',
  slow: 'dFaultSlow',
  ont_fault: 'dFaultOntFault',
  other: 'dFaultOtherUser',
  SINGLE_OUTAGE: 'dFaultSingleOutage',
  PARTIAL_OUTAGE: 'dFaultPartialOutage',
  SLOW_NET: 'dFaultSlowNet',
  WIFI_ISSUE: 'dFaultWifiIssue',
  DEVICE_FAULT: 'dFaultDeviceFault',
  OTHER: 'dFaultOther',
}

// 投诉分类 → 文案键(用户端 POST /complaints type,落库带"用户投诉: "前缀,
// 见 internal/httpapi/user/complaint_handlers.go;后端按该前缀归入 complaints 段)。
const COMPLAINT_CATEGORY: Record<string, string> = {
  '用户投诉: attitude': 'dCpnAttitude',
  '用户投诉: quality': 'dCpnQuality',
  '用户投诉: billing': 'dCpnBilling',
  '用户投诉: suggestion': 'dCpnSuggestion',
  '用户投诉: other': 'dCpnOther',
}

export const DETAIL_TABS: DetailTab[] = [
  {
    key: 'service',
    sections: [
      { key: 'plans', limit: 3, cols: [
        { key: 'planName', k: 'dPlanName' },
        { key: 'status', k: 'dStatus', spec: { kind: 'enum', map: { ACTIVE: 'dActive', active: 'dActive', EXPIRED: 'dCpnExpired' } } },
        { key: 'effectiveAt', k: 'dEffectiveAt', spec: { kind: 'time' } },
      ] },
      { key: 'addons', cols: [
        { key: 'name', k: 'dAddonName' },
        { key: 'addonId', k: 'dAddonId' },
        { key: 'action', k: 'dAction', spec: { kind: 'enum', map: { subscribe: 'dSubscribe', unsubscribe: 'dUnsubscribe' } } },
        { key: 'createdAt', k: 'dCreatedAt', spec: { kind: 'time' } },
      ] },
      { key: 'usages', limit: 6, cols: [
        { key: 'month', k: 'dMonth' },
        { key: 'uploadGb', k: 'dUploadGb' },
        { key: 'downloadGb', k: 'dDownloadGb' },
        { key: 'totalGb', k: 'dTotalGb' },
      ] },
    ],
  },
  {
    key: 'finance',
    sections: [
      { key: 'bills', cols: [
        { key: 'billNo', k: 'dBillNo' },
        { key: 'period', k: 'dPeriod' },
        { key: 'amount', k: 'dAmount', spec: { kind: 'fee' } },
        { key: 'status', k: 'dStatus', spec: { kind: 'tag', domain: 'bill' } },
      ] },
      { key: 'payments', cols: [
        { key: 'billNo', k: 'dBillNo' },
        { key: 'channel', k: 'dChannel', spec: { kind: 'enum', map: { wechat: 'dPayWechat', alipay: 'dPayAlipay', card: 'dPayCard', cash: 'dPayCash' } } },
        { key: 'amount', k: 'dAmount', spec: { kind: 'fee' } },
        { key: 'paidAt', k: 'dPaidAt', spec: { kind: 'time' } },
      ] },
      { key: 'invoices', cols: [
        { key: 'invoiceNo', k: 'dInvoiceNo' },
        { key: 'title', k: 'dInvoiceTitle' },
        { key: 'amount', k: 'dAmount', spec: { kind: 'fee' } },
        { key: 'status', k: 'dStatus', spec: { kind: 'enum', map: { ISSUED: 'dInvIssued', VOIDED: 'dInvVoided' } } },
      ] },
    ],
  },
  {
    key: 'orders',
    sections: [
      { key: 'orders', cols: [
        { key: 'orderNo', k: 'dOrderNo' },
        { key: 'stage', k: 'dStage', spec: { kind: 'enum', map: STAGE } },
        { key: 'status', k: 'dStatus', spec: { kind: 'tag', domain: 'order' } },
        { key: 'createdAt', k: 'dCreatedAt', spec: { kind: 'time' } },
      ] },
      { key: 'faults', cols: [
        { key: 'ticketNo', k: 'dTicketNo' },
        { key: 'type', k: 'dFaultType', spec: { kind: 'enum', map: FAULT_TYPE } },
        { key: 'status', k: 'dStatus', spec: { kind: 'tag', domain: 'complaint' } },
      ] },
      { key: 'complaints', cols: [
        { key: 'complaintId', k: 'dComplaintId' },
        { key: 'type', k: 'dCpnType', spec: { kind: 'enum', map: COMPLAINT_CATEGORY } },
        { key: 'content', k: 'dBizNo' },
        { key: 'status', k: 'dStatus', spec: { kind: 'tag', domain: 'complaint' } },
      ] },
    ],
  },
  {
    key: 'profile',
    sections: [
      { key: NOTIFY_CARD_KEY, cols: [] },
      { key: 'addresses', cols: [
        { key: 'addrCode', k: 'dAddrCode' },
        { key: 'contact', k: 'dContact' },
        { key: 'phone', k: 'dPhoneCol' },
        { key: 'detail', k: 'dAddrDetail' },
        { key: 'isDefault', k: 'dIsDefault', spec: { kind: 'bool' } },
      ] },
      { key: 'verifyRecords', cols: [
        { key: 'step', k: 'dVerifyStep' },
        { key: 'result', k: 'dResult', spec: { kind: 'enum', map: { PENDING: 'dVerifyPending', PASS: 'dVerifyPass', FAIL: 'dVerifyFail' } } },
        { key: 'createdAt', k: 'dCreatedAt', spec: { kind: 'time' } },
      ] },
    ],
  },
  {
    key: 'marketing',
    sections: [
      { key: 'coupons', cols: [
        { key: 'name', k: 'dCouponName' },
        { key: 'amount', k: 'dCouponAmount', spec: { kind: 'feeCents' } },
        { key: 'status', k: 'dStatus', spec: { kind: 'enum', map: { active: 'dCpnAvailable', available: 'dCpnAvailable', used: 'dCpnUsed', USED: 'dCpnUsed', expired: 'dCpnExpired', EXPIRED: 'dCpnExpired', ISSUED: 'dCpnIssued', disabled: 'dCpnDisabled', DISABLED: 'dCpnDisabled' } } },
        { key: 'expireAt', k: 'dExpireAt', spec: { kind: 'time' } },
      ] },
      { key: 'messages', cols: [
        { key: 'title', k: 'dMsgTitle' },
        { key: 'content', k: 'dMsgContent' },
        { key: 'read', k: 'dRead', spec: { kind: 'bool' } },
        { key: 'createdAt', k: 'dCreatedAt', spec: { kind: 'time' } },
      ] },
    ],
  },
]

/** 主档展示项:k → dFields 键,value 取聚合主档字段。 */
export const PROFILE_FIELDS = ['phone', 'idType', 'idNo', 'regionName'] as const

/** 抽屉级字面量文案键:detail-drawer.tsx 直接引用、不在 DETAIL_TABS 列规格内
 * (概览芯片/通知偏好/布尔标记)。保持与 detail-drawer.tsx 字面量同步,
 * 由 detail-i18n.test.ts 兜底——打错引用键即门禁红灯,而非线上露出英文键。 */
export const DRAWER_D_KEYS = [
  'dStatBalance', 'dStatPlan', 'dStatOrders', 'dStatCoupons',
  'dNotifyBusiness', 'dNotifyMarketing', 'dNotifyChannel', 'dOn', 'dOff',
  'dYes', 'dNo', 'dSeeAll', 'dCollapse',
] as const

/** 当前在用套餐名(派生口径,对齐用户端 portalHomePlan):取 ACTIVE 中最近生效者,
 * 无 ACTIVE 回退列表末条;后端 plans 段已按 ACTIVE 过滤,此函数为口径兜底。 */
export function currentPlanName(plans: unknown): string {
  if (!Array.isArray(plans) || plans.length === 0) return ''
  let active: Record<string, unknown> | undefined
  for (const p of plans as Record<string, unknown>[]) {
    if (String(p.status).toUpperCase() === 'ACTIVE') active = p
  }
  const pick = active ?? (plans[plans.length - 1] as Record<string, unknown>)
  return typeof pick.planName === 'string' ? pick.planName : ''
}
