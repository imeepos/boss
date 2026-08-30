import { describe, expect, it } from 'vitest'
import { isVersionDrift } from './version'

describe('isVersionDrift', () => {
  it('双侧 commit 一致 = 无漂移', () => {
    expect(isVersionDrift('abc1234', 'abc1234')).toBe(false)
  })
  it('不一致 = 漂移(前端落后于已部署的服务端)', () => {
    expect(isVersionDrift('abc1234', 'def5678')).toBe(true)
  })
  it('任一侧为空不判漂移:未配置服务/旧服务端无字段/本地 dev', () => {
    expect(isVersionDrift('', 'def5678')).toBe(false)
    expect(isVersionDrift('abc1234', '')).toBe(false)
    expect(isVersionDrift('', '')).toBe(false)
  })
})
