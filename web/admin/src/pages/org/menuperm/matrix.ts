// 菜单权限矩阵纯逻辑:契约形状(user.MenuPermMatrix)+ 筛选(菜单名/角色持权)。

export interface MenuRoleCol {
  roleCode: string
  roleName: string
}

export interface MenuPermRow {
  code: string
  name: string
  roles: string[]
}

export interface MenuPermData {
  layers?: string[]
  roleColumns: MenuRoleCol[]
  rows: MenuPermRow[]
}

export type MenuPermViewRow = MenuPermRow

/** 筛选:关键字命中菜单名/code;角色命中=该角色在 row.roles 中。 */
export function filterMatrixRows(rows: MenuPermRow[], keyword: string, role: string): MenuPermRow[] {
  const kw = keyword.trim().toLowerCase()
  return rows.filter((r) => {
    const hitKw = !kw || r.name.toLowerCase().includes(kw) || r.code.toLowerCase().includes(kw)
    const hitRole = !role || r.roles.includes(role)
    return hitKw && hitRole
  })
}

/** 分页切片(菜单组量级小,前端分页)。 */
export function pageSlice<T>(list: T[], page: number, pageSize: number): T[] {
  const safe = Math.max(1, page)
  return list.slice((safe - 1) * pageSize, safe * pageSize)
}
