// 数据权限筛选纯函数:账号/角色(功能)关键字命中;数据源与 /accounts 同构(AccountRow)。

export interface ScopeRow {
  id: number
  username: string
  realName: string
  roleName: string
  legalEntityName: string
  deptName: string
  postName: string
  regionScope: string
}

/** 关键字命中 账号/姓名/角色名(大小写不敏感)。 */
export function filterDataScopes(list: ScopeRow[], keyword: string): ScopeRow[] {
  const kw = keyword.trim().toLowerCase()
  if (!kw) return list
  return list.filter((r) =>
    r.username.toLowerCase().includes(kw)
    || r.realName.toLowerCase().includes(kw)
    || r.roleName.toLowerCase().includes(kw))
}

/** 分页切片(账号量级小,前端分页)。 */
export function pageSlice<T>(list: T[], page: number, pageSize: number): T[] {
  const safe = Math.max(1, page)
  return list.slice((safe - 1) * pageSize, safe * pageSize)
}
