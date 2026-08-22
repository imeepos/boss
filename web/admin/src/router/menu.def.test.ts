import { describe, expect, it } from 'vitest'
import { MENU_GROUPS, PAGE_BY_KEY } from './menu.def'
import { visibleGroupIds, visiblePages } from './role-menu'

// 契约:docs/admin/menu.js 13 组结构照抄 + partner 企业工作台组(000098 入驻域,仅 partner_* 角色);
// 分组 id/页面 key 唯一;sysadmin 可见全部平台组(不含 partner)。
describe('menu.def', () => {
  it('14 个分组(原型 13 组 + partner 企业工作台)', () => {
    expect(MENU_GROUPS).toHaveLength(14)
    expect(MENU_GROUPS.map((g) => g.id)).toEqual([
      'overview', 'base', 'org', 'bss', 'billing', 'ams', 'oss',
      'boss', 'quad', 'provision', 'alarm', 'aaa', 'partner', 'intel',
    ])
  })

  it('分组 id 与页面 key 全局唯一', () => {
    const gids = MENU_GROUPS.map((g) => g.id)
    expect(new Set(gids).size).toBe(gids.length)
    const keys = MENU_GROUPS.flatMap((g) => g.items.map((i) => i.key))
    expect(new Set(keys).size).toBe(keys.length)
  })

  it('页面路由以 / 为前缀且含 dashboard', () => {
    for (const g of MENU_GROUPS) {
      for (const it of g.items) expect(it.path.startsWith('/')).toBe(true)
    }
    expect(PAGE_BY_KEY.get('dashboard')).toBeDefined()
  })
})

describe('role-menu 可见性', () => {
  it('sysadmin 可见 13 个平台组(不含 partner 企业工作台)', () => {
    expect(visibleGroupIds('sysadmin')).toHaveLength(13)
    expect(visibleGroupIds('sysadmin')).not.toContain('partner')
  })

  it('partner 角色可见企业工作台(partner_admin 不含平台组)', () => {
    expect(visibleGroupIds('partner_admin')).toEqual(['partner'])
    expect(visibleGroupIds('partner_staff')).toContain('partner')
    expect(visibleGroupIds('partner_staff')).toContain('overview')
  })

  it('每个平台 RoleCode 至少可见 overview(工作台兜底)', () => {
    for (const role of ['asset_admin', 'resource_admin', 'ops', 'analyst', 'sysadmin'] as const) {
      expect(visibleGroupIds(role)).toContain('overview')
    }
  })

  it('未知 roleCode 只见 overview(接口级权限由后端 permCode 兜底)', () => {
    expect(visibleGroupIds('nobody' as never)).toEqual(['overview'])
  })

  it('visiblePages 只含可见分组的页面', () => {
    const pages = visiblePages('analyst')
    const groupIds = new Set(visibleGroupIds('analyst'))
    for (const p of pages) expect(groupIds.has(p.groupId)).toBe(true)
  })
})
