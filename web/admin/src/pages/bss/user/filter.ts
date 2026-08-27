// 用户列表行类型与过滤(usersSQL 列名对齐 internal/domain/customer/userdata/pg_aggregate.go)。
import { fmtTime } from '../../../lib/format'

export interface UserRow {
  customerId: number
  name: string
  phone: string
  loginName?: string
  planName?: string
  balance?: number
  createdAt?: string
  [k: string]: unknown
}

export function filterUsers(rows: UserRow[], keyword: string): UserRow[] {
  const kw = keyword.trim()
  if (!kw) return rows
  return rows.filter((r) =>
    r.name.includes(kw) || r.phone.includes(kw) || (r.loginName ?? '').includes(kw))
}

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}

// 列渲染兜底:接口缺字段(如测试数据无账户行)时渲染占位符,
// 禁止 String() 强转把 undefined/null 打成字面量(2026-09 注册时间 undefined 事故)。
export function loginNameCell(r: UserRow): string {
  return r.loginName || '—'
}

export function createdAtCell(r: UserRow): string {
  return fmtTime(r.createdAt)
}
