// buildTrendSeries 回归测试:regression for B6 snap?.regionROI.reduce() NPE。
// 真实生产数据(Payload 字段缺失/旧快照缺 regionROI/maintenance)不应崩。
import { describe, expect, it } from 'vitest'
import { buildTrendSeries, type TrendSnap } from './trend'

const LEGENDS = ['收入', '投入', 'ROI', '告警', '待维护']

function snap(p: object | null): TrendSnap {
  return { windowStart: '2026-08-22T00:00:00Z', payload: p as never }
}

describe('buildTrendSeries', () => {
  it('空数组返回 5 条空线(由 UI 端 trendSnaps.length >= 2 决定是否渲染)', () => {
    const out = buildTrendSeries([], LEGENDS)
    expect(out).toHaveLength(5)
    for (const s of out) expect(s.values).toEqual([])
  })

  it('3 份正常快照倒序→正序,ROI/告警/待维护正确聚合', () => {
    const snaps: TrendSnap[] = [
      // 历史: 旧→新(传参顺序 = API 返回顺序 = 新→旧,所以下面这列是 [新, 中, 旧])
      snap({ regionROI: [{ revenue: 100, investment: 50, roi: 2 }], maintenance: [{ priority: 'MUST_REPLACE' }, { priority: 'WATCH' }] }),
      snap({ regionROI: [{ revenue: 200, investment: 80, roi: 2.5 }], maintenance: [] }),
      snap({ regionROI: [{ revenue: 300, investment: 100, roi: 3 }], maintenance: [{ priority: 'MUST_REPLACE' }] }),
    ]
    // 经过 [...snaps].reverse() 后,ordered = [旧=300, 中=200, 新=100]
    const out = buildTrendSeries(snaps, LEGENDS)
    expect(out[0].values).toEqual([300, 200, 100]) // 收入
    expect(out[1].values).toEqual([100, 80, 50])   // 投入
    expect(out[2].values).toEqual([3, 2.5, 2])     // ROI
    expect(out[3].values).toEqual([1, 0, 2])       // 告警
    expect(out[4].values).toEqual([1, 0, 1])       // MUST_REPLACE
  })

  it('regression: payload.regionROI 缺失(老快照 schema)→ 全部 0,不抛 NPE', () => {
    // B6 第一次写的 snap?.regionROI.reduce(...) 在这种快照上 .reduce 被 null 调,栈:
    // "Cannot read properties of null (reading 'reduce')" → useMemo crash → 整页白屏。
    const snaps: TrendSnap[] = [
      { windowStart: 'x', payload: { generatedAt: 'x', indicators: [], regionROI: null as never, heatmapTop: [], maintenance: [], conclusions: [] } },
      { windowStart: 'x', payload: null },
      { windowStart: 'x', payload: undefined as never },
    ]
    const out = buildTrendSeries(snaps, LEGENDS)
    expect(out[0].values).toEqual([0, 0, 0])
    expect(out[1].values).toEqual([0, 0, 0])
    expect(out[2].values).toEqual([0, 0, 0])
    expect(out[3].values).toEqual([0, 0, 0])
    expect(out[4].values).toEqual([0, 0, 0])
  })

  it('regression: payload.maintenance 缺失 → alerts/must 0', () => {
    const snaps: TrendSnap[] = [
      { windowStart: 'x', payload: { generatedAt: 'x', indicators: [], regionROI: [], heatmapTop: [], maintenance: null as never, conclusions: [] } },
    ]
    const out = buildTrendSeries(snaps, LEGENDS)
    expect(out[3].values).toEqual([0])
    expect(out[4].values).toEqual([0])
  })

  it('ROI 单条记录 revenue/investment/roi 字段为 undefined → 0', () => {
    const snaps: TrendSnap[] = [
      snap({ regionROI: [{ /* 全缺 */ } as never, { revenue: 50, investment: 25, roi: 2 }], maintenance: [] }),
    ]
    const out = buildTrendSeries(snaps, LEGENDS)
    expect(out[0].values).toEqual([50])   // 第一条 revenue=undefined → 0
    expect(out[1].values).toEqual([25])   // 第一条 investment=undefined → 0
    expect(out[2].values).toEqual([2])     // 第一条 roi=undefined → 0
  })

  it('ROI 字段非 number(String/NaN)兜底为 0', () => {
    const snaps: TrendSnap[] = [
      snap({ regionROI: [{ revenue: 'oops' as never, investment: NaN, roi: 1.5 }], maintenance: [] }),
    ]
    const out = buildTrendSeries(snaps, LEGENDS)
    expect(out[0].values).toEqual([0])
    expect(out[1].values).toEqual([0])
    expect(out[2].values).toEqual([1.5])
  })

  it('legends 不足 5 项时 name 兜底为 ""', () => {
    // 实现:第 i 条线绑 legends[i] → legends 长度 2 时,前 2 条用 [0,1]='A'/'B',后 3 条 undefined→''。
    const out = buildTrendSeries([], ['A', 'B'])
    expect(out.map((s) => s.name)).toEqual(['A', 'B', '', '', ''])
  })
})