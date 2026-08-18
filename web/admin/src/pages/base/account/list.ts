// 账号列表纯逻辑:三条件筛选(关键字/角色/状态)。

export interface AccountRow {
  id: number
  username: string
  realName: string
  phone?: string
  roleCode?: string
  roleName: string
  legalEntityId?: number
  deptId?: number
  postId?: number
  legalEntityName: string
  deptName: string
  postName: string
  regionScope: string
  status: number
}

/** 关键字命中 账号/姓名/角色名;角色按名精确;状态 1/0。 */
export function filterAccounts(list: AccountRow[], keyword: string, role: string, status: string): AccountRow[] {
  const kw = keyword.trim().toLowerCase()
  return list.filter((r) => {
    const hitKw = !kw
      || r.username.toLowerCase().includes(kw)
      || r.realName.toLowerCase().includes(kw)
      || r.roleName.toLowerCase().includes(kw)
    const hitRole = !role || r.roleName === role
    const hitStatus = status === '' || String(r.status) === status
    return hitKw && hitRole && hitStatus
  })
}

/** 分页切片(账号量级小,前端分页)。 */
export function pageSlice<T>(list: T[], page: number, pageSize: number): T[] {
  const safe = Math.max(1, page)
  return list.slice((safe - 1) * pageSize, safe * pageSize)
}
