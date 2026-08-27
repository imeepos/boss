import { describe, expect, it } from 'vitest'
import { MENU_GROUPS, PAGE_BY_KEY, isNavActive } from './menu.def'
import { visibleGroupIds, visiblePages } from './role-menu'

// 契约:2026-08-27 按实际内容重组为 16 组(决策见 docs/notes/adopted/2026-08-27-sidebar-regroup.md);
// 页面 key/path 与重组前完全一致;分组 id/页面 key 唯一;sysadmin 可见全部平台组(不含 partner)。
describe('menu.def', () => {
  it('16 个分组(按内容重组,总览→业务→资源→监控→内容→管理→企业工作台)', () => {
    expect(MENU_GROUPS).toHaveLength(16)
    expect(MENU_GROUPS.map((g) => g.id)).toEqual([
      'overview', 'bss', 'billing', 'boss', 'worker', 'ams', 'oss',
      'provision', 'quad', 'aaa', 'cms', 'intel', 'org', 'channel', 'system', 'partner',
    ])
  })

  it('页面总数不变(81 页,key 集合与重组前一致)', () => {
    expect(MENU_GROUPS.flatMap((g) => g.items)).toHaveLength(81)
  })

  it('重组后关键页面归属新分组(2026-08-27)', () => {
    const grp = (k: string) => PAGE_BY_KEY.get(k)?.groupId
    expect(grp('realname-review')).toBe('bss')
    expect(grp('account')).toBe('org')
    expect(grp('apikey')).toBe('channel')
    expect(grp('openplat')).toBe('channel')
    expect(grp('worker')).toBe('worker')
    expect(grp('alarm')).toBe('aaa')
    expect(grp('message')).toBe('cms')
    expect(grp('knowledge')).toBe('cms')
    expect(grp('release')).toBe('cms')
    expect(grp('license')).toBe('system')
    expect(grp('crashlogs')).toBe('system')
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

describe('isNavActive 侧栏激活判定', () => {
  it('精确路径激活;深层路由非菜单项时算同页', () => {
    expect(isNavActive('/boss/site', '/boss/site')).toBe(true)
    expect(isNavActive('/boss/site', '/boss/site/new')).toBe(true)
    expect(isNavActive('/boss/site', '/boss/site/123')).toBe(true)
  })

  it('深层路由自身是菜单项时不高亮父项(官网分类不再连带官网内容)', () => {
    expect(isNavActive('/boss/site', '/boss/site/cats')).toBe(false)
    expect(isNavActive('/boss/site/cats', '/boss/site/cats')).toBe(true)
    expect(isNavActive('/boss/site/cats', '/boss/site')).toBe(false)
  })

  it('同前缀菜单兄弟项互不高亮(/bss/marketing vs /bss/marketing-recon)', () => {
    expect(isNavActive('/bss/marketing', '/bss/marketing-recon')).toBe(false)
    expect(isNavActive('/bss/marketing-recon', '/bss/marketing-recon')).toBe(true)
    expect(isNavActive('/bss/marketing', '/bss/marketing')).toBe(true)
  })

  it('不同路径不高亮', () => {
    expect(isNavActive('/boss/site', '/boss/release')).toBe(false)
    expect(isNavActive('/boss/site', '/')).toBe(false)
  })
})

describe('role-menu 可见性', () => {
  it('sysadmin 可见 15 个平台组(不含 partner 企业工作台)', () => {
    expect(visibleGroupIds('sysadmin')).toHaveLength(15)
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
