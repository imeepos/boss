// 客户档案过滤逻辑(纯函数,便于单测):后端已按 keyword/phone/status 过滤,此处仅前端兜底。
import type { CustomerRow } from './types'

export const SERVICE_STATUSES = ['ACTIVE', 'ARREARS', 'SUSPENDED'] as const

/** 前端兜底过滤:keyword 匹配姓名/证件号,phone 前缀匹配。 */
export function filterCustomers(rows: CustomerRow[], keyword: string, phone: string): CustomerRow[] {
  const k = keyword.trim().toLowerCase()
  const p = phone.trim()
  return rows.filter((r) => {
    if (k && !(r.name.toLowerCase().includes(k) || (r.idNo ?? '').toLowerCase().includes(k))) return false
    if (p && !r.phone.includes(p)) return false
    return true
  })
}

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}
