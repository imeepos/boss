// 三语言键集一致性:zh-CN 为源,en-US/ms-MY 必须逐层同键(缺译/漂移即挂)。
import { describe, it, expect } from 'vitest'
import zhCN from './zh-CN'
import enUS from './en-US'
import msMY from './ms-MY'

function keyPaths(obj: unknown, prefix = ''): string[] {
  if (typeof obj !== 'object' || obj === null || Array.isArray(obj)) return [prefix]
  return Object.entries(obj).flatMap(([k, v]) => keyPaths(v, prefix ? `${prefix}.${k}` : k))
}

describe('i18n 键集一致性', () => {
  const zh = new Set(keyPaths(zhCN))

  it.each([
    ['en-US', enUS],
    ['ms-MY', msMY],
  ] as const)('%s 与 zh-CN 键集完全一致', (_name, locale) => {
    const other = new Set(keyPaths(locale))
    const missing = [...zh].filter((k) => !other.has(k))
    const extra = [...other].filter((k) => !zh.has(k))
    expect(missing, '该语言缺键(zh-CN 有而它没有)').toEqual([])
    expect(extra, '该语言多键(zh-CN 没有而它有)').toEqual([])
  })
})
