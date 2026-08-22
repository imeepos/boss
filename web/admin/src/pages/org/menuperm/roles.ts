// 角色管理纯逻辑:契约形状(GET /role-details、GET /permissions)+ 模板复用/单独调整规则。

export interface RoleDetail {
  id: number
  code: string
  name: string
  isBuiltin: boolean
  permissionCodes: string[]
}

export interface PermissionRow {
  code: string
  name: string
}

/** 权限按"菜单可见性 menu:* / 功能操作"分组展示。 */
export function splitPerms(perms: PermissionRow[]): { menu: PermissionRow[]; action: PermissionRow[] } {
  const menu: PermissionRow[] = []
  const action: PermissionRow[] = []
  for (const p of perms) (p.code.startsWith('menu:') ? menu : action).push(p)
  return { menu, action }
}

/** 模板复用:以内置角色权限集整体替换当前勾选(单独调整=在此基础上继续增删)。 */
export function applyTemplate(template: RoleDetail | null): string[] {
  return template ? [...template.permissionCodes] : []
}

/** 提交载荷:名称去空白 + 勾选集去重。 */
export function rolePayload(name: string, codes: string[]): { name: string; permissionCodes: string[] } {
  return {
    name: name.trim(),
    permissionCodes: [...new Set(codes)],
  }
}

/** 后端错误码 → 展示文案 key(40300 内置保护 / 40900 引用中)。 */
export function roleErrorCode(e: unknown): 'protected' | 'inUse' | null {
  const code = (e as { code?: number })?.code
  if (code === 40300) return 'protected'
  if (code === 40900) return 'inUse'
  return null
}

/** 角色表展示行。 */
export function toRoleRows(details: RoleDetail[]): RoleDetail[] {
  return [...details].sort((a, b) => Number(b.isBuiltin) - Number(a.isBuiltin) || a.id - b.id)
}
