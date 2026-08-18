// roleCode → 分组可见性映射(前端仅渲染;接口级权限由后端 permCode 拦截,直访越权路由 → 403 页)。
// 角色职责依据 server-ts/src/enums.ts RoleCode 与 domain-map.md 能力域。
import { MENU_GROUPS, PAGE_BY_KEY, type MenuItem } from './menu.def'

export type RoleCode =
  | 'customer' | 'technician' | 'asset_admin'
  | 'resource_admin' | 'ops' | 'analyst' | 'sysadmin'

const OVERVIEW = 'overview'

/** 各角色可见分组(sysadmin 全量)。 */
export const ROLE_GROUPS: Record<RoleCode, string[]> = {
  sysadmin: MENU_GROUPS.map((g) => g.id),
  ops: [OVERVIEW, 'bss', 'billing', 'boss', 'quad', 'alarm', 'aaa', 'intel', 'provision'],
  asset_admin: [OVERVIEW, 'ams'],
  resource_admin: [OVERVIEW, 'oss', 'provision', 'alarm'],
  customer: [OVERVIEW, 'bss'],
  technician: [OVERVIEW, 'boss', 'quad'],
  analyst: [OVERVIEW, 'intel', 'aaa'],
}

/** 可见分组(保序按 MENU_GROUPS;未知角色兜底仅工作台)。 */
export function visibleGroupIds(role: string): string[] {
  const allowed = new Set(ROLE_GROUPS[role as RoleCode] ?? [OVERVIEW])
  return MENU_GROUPS.filter((g) => allowed.has(g.id)).map((g) => g.id)
}

export interface VisiblePage extends MenuItem {
  groupId: string
}

/** 可见页面全集。 */
export function visiblePages(role: string): VisiblePage[] {
  const groups = new Set(visibleGroupIds(role))
  const out: VisiblePage[] = []
  for (const g of MENU_GROUPS) {
    if (!groups.has(g.id)) continue
    for (const item of g.items) out.push({ ...item, groupId: g.id })
  }
  return out
}

/** 直访路由是否对该角色可见(403 判定)。 */
export function canAccess(role: string, pageKey: string): boolean {
  const page = PAGE_BY_KEY.get(pageKey)
  if (!page) return false
  return visibleGroupIds(role).includes(page.groupId)
}
