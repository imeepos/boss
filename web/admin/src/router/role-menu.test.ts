// 自定义角色动态菜单推导测试:按 menu:<key> 权限码过滤分组与页面可达性。
import { describe, expect, it } from 'vitest'
import { canAccess, filterGroupsByPerms, visibleGroupsForRole } from './role-menu'

describe('自定义角色(未知 roleCode)', () => {
  const perms = ['menu:dashboard', 'menu:order', 'menu:dispatch']

  it('filterGroupsByPerms 只留有权限项的分组,并过滤无权限项', () => {
    const groups = filterGroupsByPerms(perms)
    expect(groups.map((g) => g.id)).toEqual(['overview', 'boss'])
    const boss = groups.find((g) => g.id === 'boss')
    // boss 组含 9 项,仅持有 order/dispatch 两项权限,其余(含无权限码的 message)不出现
    expect(boss?.items.map((it) => it.key).sort()).toEqual(['dispatch', 'order'])
  })

  it('visibleGroupsForRole 走动态推导', () => {
    expect(visibleGroupsForRole('custom_ab', perms).map((g) => g.id)).toEqual(['overview', 'boss'])
    expect(visibleGroupsForRole('custom_ab', []).map((g) => g.id)).toEqual([])
  })

  it('canAccess 按权限码逐项判定', () => {
    expect(canAccess('custom_ab', 'order', perms)).toBe(true)
    expect(canAccess('custom_ab', 'worker', perms)).toBe(false)
  })
})

describe('内置角色不受影响', () => {
  it('静态映射优先于权限码', () => {
    const groups = visibleGroupsForRole('ops', [])
    expect(groups.map((g) => g.id)).toContain('bss')
    expect(canAccess('ops', 'customer', [])).toBe(true)
  })
})
