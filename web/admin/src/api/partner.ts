// 招商入驻域接口(契约:internal/httpapi/admin/partner*.go,迁移 000098)。
import { apiFetch } from './client'

export type PartnerStatus = 'PENDING' | 'APPROVED' | 'REJECTED'

export interface PartnerApplication {
  id: number
  companyName: string
  creditCode: string
  contactName: string
  contactPhone: string
  email: string
  businessDesc: string
  status: PartnerStatus
  reviewNote: string
  legalEntityId: number
  adminAccountId: number
  submittedAt: string
  reviewedAt?: string
}

export interface PartnerApproveResult {
  applicationId: number
  legalEntityId: number
  adminAccountId: number
  username: string
  initialPassword: string
}

export interface PartnerProfile {
  legalEntityId: number
  companyName: string
  creditCode: string
  contactName: string
  contactPhone: string
  email: string
  appliedAt: string
  approvedAt: string
}

export interface PartnerStaff {
  id: number
  username: string
  realName: string
  phone: string
  roleCode: 'partner_admin' | 'partner_staff'
  status: number
  createdAt: string
}

export interface PartnerOrder {
  id: number
  orderNo: string
  customerName: string
  stage: number
  status: string
  createdAt: string
}

/** 公开提交入驻申请(免登录)。 */
export function submitPartnerApplication(input: {
  companyName: string; creditCode: string; contactName: string
  contactPhone: string; email: string; businessDesc: string
}): Promise<{ applicationId: number } | null> {
  return apiFetch('/partner/applications', { method: 'POST', body: input })
}

/** 审核队列(status 空=全部)。 */
export function listPartnerApplications(status = ''): Promise<{ items: PartnerApplication[] } | null> {
  return apiFetch('/partner/applications', { query: { status } })
}

/** 审核通过:开通子公司 + 管理账号,初始口令仅本次返回。 */
export function approvePartnerApplication(id: number): Promise<PartnerApproveResult | null> {
  return apiFetch(`/partner/applications/${id}/approve`, { method: 'POST' })
}

/** 审核驳回(意见必填)。 */
export function rejectPartnerApplication(id: number, note: string): Promise<unknown> {
  return apiFetch(`/partner/applications/${id}/reject`, { method: 'POST', body: { note } })
}

/** 本企业档案。 */
export function fetchPartnerProfile(): Promise<PartnerProfile | null> {
  return apiFetch('/partner/me')
}

/** 本企业员工列表。 */
export function listPartnerStaff(): Promise<{ items: PartnerStaff[] } | null> {
  return apiFetch('/partner/staff')
}

/** 新建员工账号(partner_staff)。 */
export function createPartnerStaff(input: {
  username: string; password: string; realName: string; phone: string
}): Promise<{ staffId: number } | null> {
  return apiFetch('/partner/staff', { method: 'POST', body: input })
}

/** 启用/停用员工。 */
export function setPartnerStaffStatus(id: number, status: 0 | 1): Promise<unknown> {
  return apiFetch(`/partner/staff/${id}/status`, { method: 'PUT', body: { status } })
}

/** 本企业订单(只读)。 */
export function listPartnerOrders(): Promise<{ items: PartnerOrder[] } | null> {
  return apiFetch('/partner/orders')
}
