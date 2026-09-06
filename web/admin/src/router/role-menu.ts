// roleCode → 分组可见性映射(前端仅渲染;接口级权限由后端 permCode 拦截,直访越权路由 → 403 页)。
// 角色职责依据 server-ts/src/enums.ts RoleCode 与 domain-map.md 能力域。
import { MENU_GROUPS, PAGE_BY_KEY, type MenuGroup, type MenuItem } from './menu.def'

/** 后台 accounts 专用角色；customer/workers 属于其他端，不进入管理后台。 */
export type RoleCode =
  | 'asset_admin' | 'resource_admin' | 'ops' | 'analyst' | 'sysadmin'
  | 'partner_admin' | 'partner_staff'

const OVERVIEW = 'overview'
// 入驻企业专属组:仅 partner_* 角色可见;sysadmin 是平台方账号无 legal_entity 归属,不进企业工作台。
const PARTNER_GROUPS = ['partner']

/** 各角色可见分组(sysadmin 全量)。 */
export const ROLE_GROUPS: Record<RoleCode, string[]> = {
  sysadmin: MENU_GROUPS.map((g) => g.id).filter((id) => !PARTNER_GROUPS.includes(id)),
  partner_admin: [...PARTNER_GROUPS],
  partner_staff: [OVERVIEW, ...PARTNER_GROUPS],
  // 告警并入 aaa(认证与告警);boss 拆出 worker/cms,ops 职责覆盖履约+装维+内容触达。
  ops: [OVERVIEW, 'bss', 'billing', 'boss', 'worker', 'quad', 'aaa', 'cms', 'intel', 'provision'],
  asset_admin: [OVERVIEW, 'ams'],
  resource_admin: [OVERVIEW, 'oss', 'provision', 'aaa'],
  analyst: [OVERVIEW, 'intel', 'aaa'],
}

/** 可见分组(保序按 MENU_GROUPS;未知角色无组级授权,走权限码推导)。 */
export function visibleGroupIds(role: string): string[] {
  const allowed = new Set(ROLE_GROUPS[role as RoleCode] ?? [])
  return MENU_GROUPS.filter((g) => allowed.has(g.id)).map((g) => g.id)
}

/** 登录落地页:按权限码(menu:<key>)取菜单定义序首个有权页;后端权限为单一事实源(2026-08-28 裁定A)。 */
export function landingPathFor(permCodes?: string[]): string | null {
  const held = new Set(permCodes ?? [])
  for (const g of MENU_GROUPS) {
    for (const it of g.items) {
      if (held.has(`menu:${it.key}`)) return it.path
    }
  }
  return null
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

/** 直访路由是否对该角色可见(403 判定;自定义角色按 menu:<key> 权限码逐项判定)。 */
export function canAccess(role: string, pageKey: string, permCodes?: string[]): boolean {
  const page = PAGE_BY_KEY.get(pageKey)
  if (!page) return false
  if (ROLE_GROUPS[role as RoleCode]) return visibleGroupIds(role).includes(page.groupId)
  return new Set(permCodes ?? []).has(`menu:${page.item.perm ?? pageKey}`)
}

/** 自定义角色:按持有权限码过滤菜单项(perm 覆盖优先),空组剔除(无独立权限码的项不出现)。 */
export function filterGroupsByPerms(permCodes: string[]): MenuGroup[] {
  const held = new Set(permCodes)
  return MENU_GROUPS
    .map((g) => ({ ...g, items: g.items.filter((it) => held.has(`menu:${it.perm ?? it.key}`)) }))
    .filter((g) => g.items.length > 0)
}

/** 侧栏可见分组:内置角色走静态映射,自定义角色按权限码动态推导。 */
export function visibleGroupsForRole(role: string, permCodes?: string[]): MenuGroup[] {
  if (ROLE_GROUPS[role as RoleCode]) {
    const ids = new Set(visibleGroupIds(role))
    return MENU_GROUPS.filter((g) => ids.has(g.id))
  }
  return filterGroupsByPerms(permCodes ?? [])
}
