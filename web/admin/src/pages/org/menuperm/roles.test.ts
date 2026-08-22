// 角色管理逻辑测试:模板复用/单独调整/错误码映射/排序。
import { describe, expect, it } from 'vitest'
import {
  applyTemplate, roleErrorCode, rolePayload, splitPerms, toRoleRows,
  type RoleDetail,
} from './roles'

describe('splitPerms', () => {
  it('menu:* 与功能权限分组', () => {
    const { menu, action } = splitPerms([
      { code: 'menu:dashboard', name: '工作台' },
      { code: 'asset:create', name: '新建资产' },
    ])
    expect(menu).toHaveLength(1)
    expect(action).toHaveLength(1)
  })
})

describe('applyTemplate', () => {
  it('整体复制内置角色权限集(模板复用)', () => {
    const tpl: RoleDetail = {
      id: 1, code: 'ops', name: '运营', isBuiltin: true,
      permissionCodes: ['menu:dashboard', 'menu:order'],
    }
    expect(applyTemplate(tpl)).toEqual(['menu:dashboard', 'menu:order'])
  })
  it('无模板=空白起步', () => {
    expect(applyTemplate(null)).toEqual([])
  })
})

describe('rolePayload', () => {
  it('名称去空白 + 权限码去重', () => {
    expect(rolePayload('  运维班长  ', ['menu:dashboard', 'menu:dashboard']))
      .toEqual({ name: '运维班长', permissionCodes: ['menu:dashboard'] })
  })
})

describe('roleErrorCode', () => {
  it('40300 内置保护 / 40900 引用中 / 其余 null', () => {
    expect(roleErrorCode({ code: 40300 })).toBe('protected')
    expect(roleErrorCode({ code: 40900 })).toBe('inUse')
    expect(roleErrorCode({ code: 0 })).toBeNull()
    expect(roleErrorCode(new Error('x'))).toBeNull()
  })
})

describe('toRoleRows', () => {
  it('内置在前,同类按 id 升序', () => {
    const rows = toRoleRows([
      { id: 9, code: 'custom_a', name: 'A', isBuiltin: false, permissionCodes: [] },
      { id: 2, code: 'ops', name: '运营', isBuiltin: true, permissionCodes: [] },
      { id: 1, code: 'sysadmin', name: '系统管理员', isBuiltin: true, permissionCodes: [] },
    ])
    expect(rows.map((r) => r.code)).toEqual(['sysadmin', 'ops', 'custom_a'])
  })
})
