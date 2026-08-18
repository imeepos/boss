// 用户列表行类型与过滤(usersSQL 列名对齐 internal/domain/customer/userdata/pg_aggregate.go)。
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
