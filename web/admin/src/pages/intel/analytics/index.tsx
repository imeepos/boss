// 经营分析页:契约 GET /analytics/indicators + /analytics/heatmap + /analytics/maintenance。
// 顶部概览(4 统计卡 + 3 图:环形/横条/柱状);下方仅保留维护清单表格作运维入口。
// v1 三页签(指标/热力/维护)已收敛为单视图(commit A6)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice } from '../types'
import type { AnalyticsMaintRow, HeatCellRow, IndicatorRow, RegionRoiRow } from '../types'
import { TableStateRow } from '../../../components/business'
import { CardShell, Donut, HorizontalBar, StatCard, VerticalBars } from '../../../components/business/charts'
import { formatSegmentValue } from '../../../components/business/charts/format-value'

export default function AnalyticsPage() {
  const t = useT()
  const a = t.pages.analyticsPage
  const [indicators, setIndicators] = useState<IndicatorRow[]>([])
  const [roi, setRoi] = useState<RegionRoiRow[]>([])
  const [heat, setHeat] = useState<HeatCellRow[]>([])
  const [maints, setMaints] = useState<AnalyticsMaintRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  // 三个接口并发拉;互不依赖,单 reload 全取。
  const load = () => {
    setError('')
    setBusy(true)
    const done = () => setBusy(false)
    Promise.all([
      apiFetch<{ indicators: IndicatorRow[]; regionROI: RegionRoiRow[] }>('/analytics/indicators')
        .then((d) => { setIndicators(d?.indicators ?? []); setRoi(d?.regionROI ?? []) })
        .catch((e) => setError(e instanceof Error ? e.message : a.loadFail)),
      apiFetch<{ items: HeatCellRow[] }>('/analytics/heatmap')
        .then((d) => setHeat(d?.items ?? []))
        .catch((e) => setError(e instanceof Error ? e.message : a.loadFail)),
      apiFetch<{ items: AnalyticsMaintRow[] }>('/analytics/maintenance')
        .then((d) => setMaints(d?.items ?? []))
        .catch((e) => setError(e instanceof Error ? e.message : a.loadFail)),
    ]).finally(done)
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const maintSlice = pageSlice(maints, page, pageSize)

  // 概览计算:三大聚合 + 4 张卡(总收入/总投入/综合 ROI/待维护)
  const totalRev = roi.reduce((s, r) => s + r.revenue, 0)
  const totalInv = roi.reduce((s, r) => s + r.investment, 0)
  const compositeRoi = totalInv > 0 ? (totalRev / totalInv) * 100 : 0
  const heatTop10 = [...heat].sort((x, y) => y.utilization - x.utilization).slice(0, 10)
  const heatUtilLabels = heatTop10.map((x) => x.name || `#${x.addressId}`)
  const heatUtilValues = heatTop10.map((x) => Math.round(x.utilization * 100))

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <section className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard label={a.roiColumns[1]} value={totalRev.toFixed(2)} />
        <StatCard label={a.roiColumns[2]} value={totalInv.toFixed(2)} />
        <StatCard label={a.roiColumns[3]} value={`${compositeRoi.toFixed(2)}%`} />
        <StatCard label={a.overviewMaint} value={maints.length} />
      </section>
      <section className="mb-6 grid grid-cols-1 gap-4 xl:grid-cols-3">
        <CardShell title={a.overviewIndicator}>
          <Donut
            segments={indicators.map((x) => ({ label: x.name || x.key, value: x.value }))}
            formatValue={(_v, seg) => {
              // 通过 label 反查原 IndicatorRow(segment 仅有 label/value,key 丢在 map 过程中)
              const orig = indicators.find((i) => i.name === seg.label || i.key === seg.label)
              return orig ? formatSegmentValue(orig) : seg.value.toString()
            }}
            sublabel={indicators.length > 0 ? `${indicators.length} 项` : undefined}
          />
        </CardShell>
        <CardShell title={a.overviewRoi}>
          <HorizontalBar
            items={roi.map((r) => ({ label: r.regionName || `#${r.regionId}`, value: r.roi }))}
            format={(v) => a.roiFmt.replace('{value}', v.toFixed(2))}
          />
        </CardShell>
        <CardShell title={a.overviewHeat}>
          <VerticalBars
            labels={heatUtilLabels}
            values={heatUtilValues}
            valueLabel="%"
            emptyText="尚无地址端口数据"
          />
        </CardShell>
      </section>

      <CardShell>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{a.tabMaint}</h3>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-2 mb-2 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.maintColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {maintSlice.map((x) => (
                  <tr key={x.deviceNo}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.deviceNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.deviceType || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.healthScore}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.faultCount}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.ageYears}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="maintPriority" value={x.priority} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.reason || '—'}</td>
                  </tr>
                ))}
                {!maintSlice.length && <TableStateRow colSpan={7} loading={busy} text={a.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end pt-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={maints.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </CardShell>
    </div>
  )
}