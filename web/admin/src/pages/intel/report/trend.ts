// 报告趋势纯函数:从 /reports/history 返回的快照列表抽 5 条指标线。
// 抽离以便 vitest 覆盖(history/UI 共用同一份逻辑)。
// 关键鲁棒性:任一快照 payload 字段缺失(regionROI/maintenance/indicators/conclusions)都
// 兜底为 0 或空数组,不抛 TypeError(regression: B6 第一次写时用 snap?.regionROI.reduce(...)
// 在 regionROI 缺失场景会触发 "Cannot read properties of null (reading 'reduce')")。

import type { ReportPayload } from '../types'

export interface TrendSnap {
  windowStart: string
  payload: ReportPayload | null
}

export interface TrendPoint {
  name: string
  values: number[]
}

/** 单份 payload 的 ROI 字段求和;任一环节缺失返回 0。 */
function sumRoi(p: ReportPayload | null | undefined, k: 'revenue' | 'investment' | 'roi'): number {
  const roi = p?.regionROI
  if (!Array.isArray(roi)) return 0
  let s = 0
  for (const r of roi) {
    const v = r?.[k]
    if (typeof v === 'number' && Number.isFinite(v)) s += v
  }
  return s
}

function lenMaintenance(p: ReportPayload | null | undefined, onlyMustReplace = false): number {
  const m = p?.maintenance
  if (!Array.isArray(m)) return 0
  if (!onlyMustReplace) return m.length
  let n = 0
  for (const x of m) if (x?.priority === 'MUST_REPLACE') n++
  return n
}

/** 把 history items 倒序→反转成时间正序,抽 5 条线(收入/投入/ROI/告警/待维护)。 */
export function buildTrendSeries(
  snaps: TrendSnap[],
  legends: readonly string[],
): TrendPoint[] {
  // 旧→新(从左到右画);空入参安全(返回 5 条空线,UI 端用 trendSnaps.length >= 2 判定渲染)。
  const ordered = [...snaps].reverse()
  const at = (idx: number) => ordered[idx]?.payload
  return [
    { name: legends[0] ?? '', values: ordered.map((_, i) => sumRoi(at(i), 'revenue')) },
    { name: legends[1] ?? '', values: ordered.map((_, i) => sumRoi(at(i), 'investment')) },
    { name: legends[2] ?? '', values: ordered.map((_, i) => sumRoi(at(i), 'roi')) },
    { name: legends[3] ?? '', values: ordered.map((_, i) => lenMaintenance(at(i))) },
    { name: legends[4] ?? '', values: ordered.map((_, i) => lenMaintenance(at(i), true)) },
  ]
}