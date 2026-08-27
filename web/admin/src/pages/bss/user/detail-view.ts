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
}

export interface DetailTab {
  key: string
  sections: DetailSection[]
}

/** 特殊卡片段:非表格,不经 sectionNames 渲染(detail-drawer 直读 sectionNames.notify)。
 * 门禁 walker 跳过它——它不是字典引用键。 */
export const NOTIFY_CARD_KEY = 'notifyPrefs'

// 订单环节文案键(terms.md 第 1 节 12 环节)。
const STAGE: Record<string, string> = Object.fromEntries(
  Array.from({ length: 12 }, (_, i) => [String(i + 1), `dStage${i + 1}`]),
)

export const DETAIL_TABS: DetailTab[] = [
  {
    key: 'service',
    sections: [
      { key: 'plans', cols: [
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
      { key: 'usages', cols: [
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
        { key: 'type', k: 'dFaultType' },
        { key: 'status', k: 'dStatus', spec: { kind: 'tag', domain: 'complaint' } },
      ] },
      { key: 'complaints', cols: [
        { key: 'complaintId', k: 'dComplaintId' },
        { key: 'type', k: 'dFaultType' },
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
  'dYes', 'dNo',
] as const

/** 概览统计:dBalances=余额段,套餐取 plans 最后一条(ORDER BY id 升序即最新)。 */
export function latestPlanName(plans: unknown): string {
  if (!Array.isArray(plans) || plans.length === 0) return ''
  const last = plans[plans.length - 1] as Record<string, unknown>
  return typeof last.planName === 'string' ? last.planName : ''
}
