import { describe, expect, it } from 'vitest'
import { MENU_GROUPS, PAGE_BY_KEY } from './menu.def'
import { visibleGroupIds, visiblePages } from './role-menu'

// 契约:docs/admin/menu.js 13 组结构照抄;分组 id/页面 key 唯一;sysadmin 全可见。
describe('menu.def', () => {
  it('13 个分组(与原型 menu.js 一致)', () => {
    expect(MENU_GROUPS).toHaveLength(13)
    expect(MENU_GROUPS.map((g) => g.id)).toEqual([
      'overview', 'base', 'org', 'bss', 'billing', 'ams', 'oss',
      'boss', 'quad', 'provision', 'alarm', 'aaa', 'intel',
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
  it('sysadmin 可见全部 13 组', () => {
    expect(visibleGroupIds('sysadmin')).toHaveLength(13)
  })

  it('每个 RoleCode 至少可见 overview(工作台兜底)', () => {
    for (const role of ['customer', 'technician', 'asset_admin', 'resource_admin', 'ops', 'analyst', 'sysadmin'] as const) {
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
