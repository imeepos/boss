// regionCascadeCore 单测：懒加载下钻调用序列、直搜防抖、默认国家兜底、回显展开（node 环境）。
import { describe, expect, it, vi } from 'vitest'
import {
  DEFAULT_COUNTRY_FALLBACK, RegionCascadeSource, SUBDIV_LIMIT, debounce,
  fetchDefaultCountry, fetchSubdivisions, resolvePath,
  type FetchLike, type SubdivRow,
} from './regionCascadeCore'

// 顺序 fixtures 记录器：每次调用返回 fixtures 下一项，并记录 (path, query)。
function makeRecorder(fixtures: unknown[]) {
  const calls: { path: string; query?: Record<string, string | number | undefined> }[] = []
  let i = 0
  const fetchImpl: FetchLike = async (path, opts) => {
    calls.push({ path, query: opts?.query })
    return (fixtures[i++] ?? null) as never
  }
  return { calls, fetchImpl }
}

describe('fetchSubdivisions 契约参数与行归一化', () => {
  it('parentCode 下钻带 countryCode 与默认 limit=200', async () => {
    const { calls, fetchImpl } = makeRecorder([[{ code: 'PH-NCR', name: 'NCR', level: 1, hasChildren: true }]])
    const rows = await fetchSubdivisions({ countryCode: 'PH', parentCode: 'PH' }, fetchImpl)
    expect(rows).toHaveLength(1)
    expect(calls[0].path).toBe('/geo/subdivisions')
    expect(calls[0].query).toEqual({ countryCode: 'PH', parentCode: 'PH', limit: SUBDIV_LIMIT })
  })

  it('keyword 直搜携带 keyword 与 countryCode，limit 可覆盖', async () => {
    const { calls, fetchImpl } = makeRecorder([[]])
    await fetchSubdivisions({ countryCode: 'PH', keyword: '马尼拉', limit: 50 }, fetchImpl)
    expect(calls[0].query).toEqual({ countryCode: 'PH', keyword: '马尼拉', limit: 50 })
  })

  it('name 优先，displayName 回退，hasChildren 布尔化', async () => {
    const { fetchImpl } = makeRecorder([[{ code: 'A', name: '甲', level: 1, hasChildren: 1 } as never]])
    const [a] = await fetchSubdivisions({ countryCode: 'PH' }, fetchImpl)
    expect(a.name).toBe('甲')
    expect(a.hasChildren).toBe(true)
    const { fetchImpl: f2 } = makeRecorder([[{ code: 'B', displayName: '乙' }]])
    const [b] = await fetchSubdivisions({ countryCode: 'PH' }, f2)
    expect(b.name).toBe('乙')
    expect(b.level).toBe(0)
    expect(b.hasChildren).toBe(false)
  })
})

describe('fetchDefaultCountry 空值兜底 PH（契约 N3）', () => {
  it.each([
    ['对象 alpha2', { alpha2: 'MY' }, 'MY'],
    ['对象 countryCode', { countryCode: 'sg' }, 'SG'],
    ['裸字符串', 'CN', 'CN'],
    ['空对象', {}, DEFAULT_COUNTRY_FALLBACK],
    ['null', null, DEFAULT_COUNTRY_FALLBACK],
    ['undefined', undefined, DEFAULT_COUNTRY_FALLBACK],
  ] as const)('%s → %s', async (_name, body, expected) => {
    const { fetchImpl } = makeRecorder([body])
    await expect(fetchDefaultCountry(fetchImpl)).resolves.toBe(expected)
  })

  it('请求失败仍兜底 PH 且留有告警信号', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const fetchImpl: FetchLike = async () => { throw new Error('net down') }
    await expect(fetchDefaultCountry(fetchImpl)).resolves.toBe('PH')
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('[region-picker]'), expect.anything())
    warn.mockRestore()
  })
})

describe('debounce 直搜防抖（契约 N2）', () => {
  it('尾缘触发：连续输入只发最后一次', () => {
    vi.useFakeTimers()
    try {
      const fn = vi.fn()
      const d = debounce(fn, 300)
      d('a'); d('b'); d('c')
      vi.advanceTimersByTime(299)
      expect(fn).not.toHaveBeenCalled()
      vi.advanceTimersByTime(1)
      expect(fn).toHaveBeenCalledTimes(1)
      expect(fn).toHaveBeenCalledWith('c')
    } finally { vi.useRealTimers() }
  })

  it('cancel 取消未触发的调用（卸载清理）', () => {
    vi.useFakeTimers()
    try {
      const fn = vi.fn()
      const d = debounce(fn, 300)
      d('x')
      d.cancel()
      vi.advanceTimersByTime(400)
      expect(fn).not.toHaveBeenCalled()
    } finally { vi.useRealTimers() }
  })
})

describe('resolvePath 回显展开（契约 N4）', () => {
  const manila = { code: 'PH-1300-MNL', name: 'Manila', level: 3, hasChildren: false, parentCode: 'PH-1300' }
  const ncr = { code: 'PH-1300', name: 'NCR', level: 2, hasChildren: true, parentCode: 'PH' }
  const ph = { code: 'PH', name: 'Philippines', level: 1, hasChildren: true, parentCode: 'PH' }

  it('按 code 反查并沿 parentCode 逐级上溯成 [一级..目标] 链', async () => {
    const { calls, fetchImpl } = makeRecorder([[manila], [ncr], [ph]])
    const chain = await resolvePath('PH', 'PH-1300-MNL', fetchImpl)
    expect(chain.map((n) => n.code)).toEqual(['PH', 'PH-1300', 'PH-1300-MNL'])
    // 调用序列：先找目标，再逐级找父
    expect(calls.map((c) => c.query?.keyword)).toEqual(['PH-1300-MNL', 'PH-1300', 'PH'])
    expect(calls.every((c) => c.query?.countryCode === 'PH' && c.query?.limit === 50)).toBe(true)
  })

  it('行无 parentCode 增强字段时退化为单节点链（不拉全量）', async () => {
    const { calls, fetchImpl } = makeRecorder([[{ ...manila, parentCode: undefined }]])
    const chain = await resolvePath('PH', 'PH-1300-MNL', fetchImpl)
    expect(chain.map((n) => n.code)).toEqual(['PH-1300-MNL'])
    expect(calls).toHaveLength(1)
  })

  it('查不到目标返回空链', async () => {
    const { fetchImpl } = makeRecorder([[]])
    await expect(resolvePath('PH', 'NOPE', fetchImpl)).resolves.toEqual([])
  })
})

describe('RegionCascadeSource 懒加载下钻调用序列（契约 N1）', () => {
  it('levelOne → children → children 的 parentCode 链与 limit', async () => {
    const { calls, fetchImpl } = makeRecorder([
      [{ code: 'PH-NCR', name: 'NCR', level: 1, hasChildren: true }],
      [{ code: 'PH-NCR-MNL', name: 'Manila', level: 2, hasChildren: true }],
      [{ code: 'PH-NCR-MNL-001', name: 'Intramuros', level: 3, hasChildren: false }],
    ])
    const src = new RegionCascadeSource(fetchImpl)
    const l1 = await src.children('PH', 'PH')
    const l2 = await src.children(l1[0].code, 'PH')
    const l3 = await src.children(l2[0].code, 'PH')
    expect(l3[0].hasChildren).toBe(false)
    expect(calls.map((c) => c.query?.parentCode)).toEqual(['PH', 'PH-NCR', 'PH-NCR-MNL'])
    expect(calls.every((c) => c.query?.countryCode === 'PH' && c.query?.limit === SUBDIV_LIMIT)).toBe(true)
  })

  it('countries 直读 /geo/countries', async () => {
    const { calls, fetchImpl } = makeRecorder([[{ alpha2: 'PH', displayName: 'Philippines' }]])
    const list = await new RegionCascadeSource(fetchImpl).countries()
    expect(list).toHaveLength(1)
    expect(calls[0].path).toBe('/geo/countries')
  })

  it('search 走 keyword 参数跨层级直搜', async () => {
    const { calls, fetchImpl } = makeRecorder([[{ code: 'PH-1300-MNL', name: 'Manila', level: 3, hasChildren: false } as SubdivRow]])
    const hits = await new RegionCascadeSource(fetchImpl).search('PH', 'Manila')
    expect(hits[0].code).toBe('PH-1300-MNL')
    expect(calls[0].query).toMatchObject({ countryCode: 'PH', keyword: 'Manila' })
  })
})
