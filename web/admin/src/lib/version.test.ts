import { describe, expect, it } from 'vitest'
import { isVersionDrift, shouldShowVersionBadge } from './version'

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

describe('shouldShowVersionBadge(seen 记忆:每次部署只提示一次)', () => {
  it('有漂移且未提示过 = 提示', () => {
    expect(shouldShowVersionBadge('abc1234', 'def5678', '')).toBe(true)
    expect(shouldShowVersionBadge('abc1234', 'def5678', 'aaa1111')).toBe(true)
  })
  it('该服务端 commit 已提示过 = 不再提示(server-only 部署防永久驻留)', () => {
    expect(shouldShowVersionBadge('abc1234', 'def5678', 'def5678')).toBe(false)
  })
  it('无漂移一律不提示', () => {
    expect(shouldShowVersionBadge('abc1234', 'abc1234', '')).toBe(false)
    expect(shouldShowVersionBadge('', 'def5678', '')).toBe(false)
  })
})
