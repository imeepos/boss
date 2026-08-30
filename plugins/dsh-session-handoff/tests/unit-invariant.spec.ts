import { describe, expect, it } from 'vitest'
import { DEFAULT_CONFIG, resolveConfig } from '../src/invariant.js'
import type { HandoffConfig } from '../src/invariant.js'

describe('resolveConfig 缺省合并', () => {
  it('无参时返回完整默认值', () => {
    expect(resolveConfig()).toEqual(DEFAULT_CONFIG)
  })

  it('undefined 与空对象等价', () => {
    expect(resolveConfig(undefined)).toEqual(resolveConfig({}))
  })

  it('显式 undefined 字段回落默认值', () => {
    const config = resolveConfig({ outputDir: undefined, remote: 'origin' })
    expect(config.outputDir).toBe(DEFAULT_CONFIG.outputDir)
    expect(config.remote).toBe('origin')
    expect(config.mainBranch).toBe(DEFAULT_CONFIG.mainBranch)
  })

  it('部分覆盖不串位', () => {
    const config = resolveConfig({ minIntervalMs: 5_000, autoExclude: false })
    expect(config.minIntervalMs).toBe(5_000)
    expect(config.autoExclude).toBe(false)
    expect(config.fileName).toBe('LATEST.md')
  })

  it('返回值与默认对象不共享引用', () => {
    const config = resolveConfig()
    expect(config).not.toBe(DEFAULT_CONFIG)
  })
})

describe('resolveConfig 输出目录归一化', () => {
  const cases: Array<{ name: string; input: string; expect: string }> = [
    { name: '普通目录名原样保留', input: 'handoff', expect: 'handoff' },
    { name: '点开头默认目录保留', input: '.handoff', expect: '.handoff' },
    { name: '去前缀 ./', input: './handoff', expect: 'handoff' },
    { name: '去前导斜杠', input: '/handoff', expect: 'handoff' },
    { name: '去尾部斜杠', input: 'handoff/', expect: 'handoff' },
    { name: '前后都清理', input: './handoff/', expect: 'handoff' },
    { name: '多级子目录保留', input: 'docs/handoff', expect: 'docs/handoff' },
  ]
  for (const c of cases) {
    it(c.name, () => {
      expect(resolveConfig({ outputDir: c.input }).outputDir).toBe(c.expect)
    })
  }
})

describe('resolveConfig 非法输入 fail-closed', () => {
  const bad: Array<{ name: string; input: Partial<HandoffConfig>; field: string }> = [
    { name: '空 outputDir', input: { outputDir: '' }, field: 'outputDir' },
    { name: '非字符串 outputDir', input: { outputDir: 3 as unknown as string }, field: 'outputDir' },
    { name: '含斜杠的 fileName', input: { fileName: 'a/b.md' }, field: 'fileName' },
    { name: '含反斜杠的 fileName', input: { fileName: 'a\\b.md' }, field: 'fileName' },
    { name: '空 fileName', input: { fileName: '' }, field: 'fileName' },
    { name: '负数 minIntervalMs', input: { minIntervalMs: -1 }, field: 'minIntervalMs' },
    { name: 'NaN minIntervalMs', input: { minIntervalMs: Number.NaN }, field: 'minIntervalMs' },
    { name: 'Infinity minIntervalMs', input: { minIntervalMs: Number.POSITIVE_INFINITY }, field: 'minIntervalMs' },
    { name: '字符串 minIntervalMs', input: { minIntervalMs: '100' as unknown as number }, field: 'minIntervalMs' },
    { name: '空 mainBranch', input: { mainBranch: '  ' }, field: 'mainBranch' },
    { name: '空 remote', input: { remote: '' }, field: 'remote' },
    { name: '字符串 autoExclude', input: { autoExclude: 'yes' as unknown as boolean }, field: 'autoExclude' },
  ]
  for (const c of bad) {
    it(`拒绝：${c.name}`, () => {
      expect(() => resolveConfig(c.input)).toThrow(/session-handoff config\./)
    })
  }
})

describe('resolveConfig 合法边界', () => {
  const good: Array<{ name: string; input: Partial<HandoffConfig>; field: keyof HandoffConfig; expect: unknown }> = [
    { name: 'minIntervalMs 允许 0（测试场景）', input: { minIntervalMs: 0 }, field: 'minIntervalMs', expect: 0 },
    { name: 'mainBranch 去空白', input: { mainBranch: ' master ' }, field: 'mainBranch', expect: 'master' },
    { name: 'remote 去空白', input: { remote: ' origin ' }, field: 'remote', expect: 'origin' },
    { name: 'autoExclude false 合法', input: { autoExclude: false }, field: 'autoExclude', expect: false },
    { name: '多级 outputDir 合法', input: { outputDir: 'a/b/c' }, field: 'outputDir', expect: 'a/b/c' },
  ]
  for (const c of good) {
    it(c.name, () => {
      expect(resolveConfig(c.input)[c.field]).toBe(c.expect)
    })
  }
})
