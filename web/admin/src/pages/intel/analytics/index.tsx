// 经营分析页:契约 GET /analytics/indicators + /analytics/heatmap + /analytics/maintenance。
// 顶部概览(4 统计卡 + 3 图:环形/横条/直方);下方三页签表格保留作明细。
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

export default function AnalyticsPage() {
  const t = useT()
  const a = t.pages.analyticsPage
  const [tab, setTab] = useState<'indicator' | 'heatmap' | 'maint'>('indicator')
  const [indicators, setIndicators] = useState<IndicatorRow[]>([])
  const [roi, setRoi] = useState<RegionRoiRow[]>([])
  const [heat, setHeat] = useState<HeatCellRow[]>([])
  const [maints, setMaints] = useState<AnalyticsMaintRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = (key: string) => {
    setError('')
    setBusy(true)
    const done = () => setBusy(false)
    if (key === 'indicator') {
      apiFetch<{ indicators: IndicatorRow[]; regionROI: RegionRoiRow[] }>('/analytics/indicators')
        .then((d) => { setIndicators(d?.indicators ?? []); setRoi(d?.regionROI ?? []) })
        .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
        .finally(done)
    } else if (key === 'heatmap') {
      apiFetch<{ items: HeatCellRow[] }>('/analytics/heatmap')
        .then((d) => setHeat(d?.items ?? []))
        .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
        .finally(done)
    } else {
      apiFetch<{ items: AnalyticsMaintRow[] }>('/analytics/maintenance')
        .then((d) => setMaints(d?.items ?? []))
        .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
        .finally(done)
    }
  }
  useEffect(() => { load(tab) }, [tab]) // eslint-disable-line react-hooks/exhaustive-deps

  const count = tab === 'indicator' ? indicators.length + roi.length : tab === 'heatmap' ? heat.length : maints.length
  const maintSlice = pageSlice(maints, page, pageSize)
  const heatSlice = pageSlice(heat, page, pageSize)

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
          <Donut segments={indicators.map((x) => ({ label: x.name || x.key, value: x.value }))} />
        </CardShell>
        <CardShell title={a.overviewRoi}>
          <HorizontalBar
            items={roi.map((r) => ({ label: r.regionName || `#${r.regionId}`, value: r.roi }))}
            format={(v) => a.roiFmt.replace('{value}', v.toFixed(2))}
          />
        </CardShell>
        <CardShell title={a.overviewHeat}>
          <VerticalBars labels={heatUtilLabels} values={heatUtilValues} />
        </CardShell>
      </section>

      <CardShell>
        <div className="mb-3 flex items-center gap-2 border-b border-[var(--shell-side-border)]">
          {(['indicator', 'heatmap', 'maint'] as const).map((key) => (
            <button key={key} onClick={() => { setTab(key); setPage(1) }}
              className={'h-8 cursor-pointer border-0 bg-transparent px-4 text-[13px] ' + (
                tab === key
                  ? 'border-b-2 border-[var(--color-border-focus)] font-semibold text-[var(--color-border-focus)]'
                  : 'text-[var(--shell-content-text)] hover:text-[var(--shell-heading)]'
              )}>
              {key === 'indicator' ? a.tabIndicator : key === 'heatmap' ? a.tabHeatmap : a.tabMaint}
            </button>
          ))}
          <span className="flex-1" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={() => load(tab)}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-2 mb-2 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : tab === 'indicator' ? (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.indicatorColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {indicators.map((x) => (
                  <tr key={x.key}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.key}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.value}</td>
                    <td className="h-11 px-3 max-w-90 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.detail || '—'}</td>
                  </tr>
                ))}
                {!indicators.length && <TableStateRow colSpan={4} loading={busy} text={a.empty} />}
              </tbody>
            </table>
            <h4 className="mt-4 mb-2 text-sm font-semibold">{a.roiColumns.join(' / ')}</h4>
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.roiColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {roi.map((x) => (
                  <tr key={x.regionId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.regionName || `#${x.regionId}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.revenue.toFixed(2)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.investment.toFixed(2)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.roi.toFixed(2)}</td>
                  </tr>
                ))}
                {!roi.length && <TableStateRow colSpan={4} loading={busy} text={a.empty} />}
              </tbody>
            </table>
          </div>
        ) : tab === 'heatmap' ? (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.heatColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {heatSlice.map((x) => (
                  <tr key={x.addressId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.name || `#${x.addressId}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.portsTotal}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.portsUsed}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{(x.utilization * 100).toFixed(1)}%</td>
                  </tr>
                ))}
                {!heatSlice.length && <TableStateRow colSpan={4} loading={busy} text={a.empty} />}
              </tbody>
            </table>
          </div>
        ) : (
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
          <Pagination total={count} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </CardShell>
    </div>
  )
}