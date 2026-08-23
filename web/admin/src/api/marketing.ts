// 营销与忠诚度域接口(契约:internal/httpapi/admin/promotion.go / loy.go)。
import { apiFetch } from './client'

// ---- 券模板 ----

export interface CouponTemplate {
  templateId: number
  legalEntityId: number
  name: string
  type: 'FULL_CUT' | 'DISCOUNT' | 'CASH'
  faceValue: number
  threshold: number
  maxDiscount: number
  totalQty: number
  perCustomerLimit: number
  validDays: number
  status: string
  issuedQty: number
  pointsPrice: number
}

export function listCouponTemplates() {
  return apiFetch<{ items: CouponTemplate[] }>('/coupon-templates')
}

export function createCouponTemplate(input: Partial<CouponTemplate>) {
  return apiFetch<{ templateId: number }>('/coupon-templates', { method: 'POST', body: input })
}

export function disableCouponTemplate(id: number) {
  return apiFetch(`/coupon-templates/${id}/disable`, { method: 'PUT' })
}

// ---- 赠送时长规则 ----

export interface GiftRule {
  ruleId: number
  legalEntityId: number
  name: string
  scopeType: string
  buyMonths: number
  giftMonths: number
  status: string
}

export function listGiftRules() {
  return apiFetch<{ items: GiftRule[] }>('/gift-rules')
}

export function createGiftRule(input: Partial<GiftRule>) {
  return apiFetch<{ ruleId: number }>('/gift-rules', { method: 'POST', body: input })
}

export function disableGiftRule(id: number) {
  return apiFetch(`/gift-rules/${id}/disable`, { method: 'PUT' })
}

// ---- 缴费送积分规则 ----

export interface EarnRule {
  ruleId: number
  pointsPerYuan: number
  minCents: number
  expireDays: number
  status: string
}

export function getEarnRule() {
  return apiFetch<{ rule: EarnRule | null }>('/loy/earn-rule')
}

export function saveEarnRule(input: Partial<EarnRule>) {
  return apiFetch<{ ruleId: number }>('/loy/earn-rule', { method: 'PUT', body: input })
}

// ---- 积分等级 / 任务 ----

export interface LoyLevel {
  levelId: number
  name: string
  minPoints: number
  status: string
}

export function listLevels() {
  return apiFetch<{ levels: LoyLevel[] }>('/loy/levels')
}

export function createLevel(input: Partial<LoyLevel>) {
  return apiFetch<{ levelId: number }>('/loy/levels', { method: 'POST', body: input })
}

export function disableLevel(id: number) {
  return apiFetch(`/loy/levels/${id}/disable`, { method: 'POST' })
}

export interface LoyTask {
  taskId: number
  code: string
  name: string
  points: number
  period: 'ONE_TIME' | 'DAILY' | 'MONTHLY'
  status: string
}

export function listTasks() {
  return apiFetch<{ tasks: LoyTask[] }>('/loy/tasks')
}

export function createTask(input: Partial<LoyTask>) {
  return apiFetch<{ taskId: number }>('/loy/tasks', { method: 'POST', body: input })
}

export function disableTask(id: number) {
  return apiFetch(`/loy/tasks/${id}/disable`, { method: 'POST' })
}

// ---- 券积分对账 ----

export interface CouponReconRow {
  templateId: number
  name: string
  type: string
  status: string
  issuedQty: number
  actualIssued: number
  byStatus: Record<string, number>
  usedCount: number
  redemptionCnt: number
  redeemedAmount: number
  faceValueTotal: number
  diffKind: 'MATCH' | 'COUNTER_DRIFT' | 'REDEMPTION_LOST'
}

export interface CouponReconSummary {
  templates: number
  actualIssued: number
  usedCount: number
  redeemedAmount: number
  faceValueTotal: number
  byDiff: Record<string, number>
}

export function couponRecon(diff?: 'drift') {
  return apiFetch<{ rows: CouponReconRow[]; summary: CouponReconSummary }>(
    '/coupon-recon', { query: { diff } })
}

export interface PointReconRow {
  customerId: number
  balance: number
  entriesSum: number
  lifetimeEarn: number
  expiredTotal: number
  entryCount: number
  byReason: Record<string, number>
  diffKind: 'MATCH' | 'BALANCE_DRIFT'
}

export interface PointReconSummary {
  customers: number
  balanceTotal: number
  lifetimeEarn: number
  expiredTotal: number
  byDiff: Record<string, number>
}

export function pointsRecon(diff?: 'drift') {
  return apiFetch<{ rows: PointReconRow[]; summary: PointReconSummary }>(
    '/loy/points-recon', { query: { diff } })
}
