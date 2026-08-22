// 回归:待办跳转目标必须是已注册路由(修复 source=告警中心 跳 /alarm 404)。
import { describe, it, expect } from 'vitest'
import { todoTarget } from './todoTarget'
import { KEY_BY_PATH } from '../../router/menu.def'

describe('todoTarget', () => {
  it('派单池 → 派单管理页(已注册)', () => {
    const path = todoTarget('派单池')
    expect(path).toBeTruthy()
    expect(KEY_BY_PATH.has(path!)).toBe(true)
  })

  it('告警中心 → 告警列表页(已注册,不再是 /alarm)', () => {
    const path = todoTarget('告警中心')
    expect(path).toBe('/alarm/alarm')
    expect(KEY_BY_PATH.has(path as string)).toBe(true)
  })

  it('未知 source 不跳转', () => {
    expect(todoTarget('其他')).toBeNull()
  })
})
