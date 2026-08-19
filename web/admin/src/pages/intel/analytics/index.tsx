// 经营分析页:契约 GET /analytics/indicators + /analytics/heatmap + /analytics/maintenance(三页签)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice } from '../types'
import type { AnalyticsMaintRow, HeatCellRow, IndicatorRow, RegionRoiRow } from '../types'

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

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div style={{ display: 'flex', gap: 4, marginBottom: 12, borderBottom: '1px solid #f0f0f0' }}>
          {(['indicator', 'heatmap', 'maint'] as const).map((key) => (
            <button key={key} onClick={() => { setTab(key); setPage(1) }}
              style={{
                padding: '8px 16px', fontSize: 14, cursor: 'pointer', background: 'none', border: 'none',
                borderBottom: tab === key ? '2px solid #1677ff' : '2px solid transparent',
                color: tab === key ? '#1677ff' : '#666', fontWeight: tab === key ? 600 : 400,
              }}>
              {key === 'indicator' ? a.tabIndicator : key === 'heatmap' ? a.tabHeatmap : a.tabMaint}
            </button>
          ))}
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={() => load(tab)}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : tab === 'indicator' ? (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.indicatorColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {indicators.map((x) => (
                  <tr key={x.key}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.key}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.value}</td>
                    <td style={{ maxWidth: 360 }} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.detail || '—'}</td>
                  </tr>
                ))}
                {!indicators.length && <tr><td colSpan={4} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{a.empty}</div></td></tr>}
              </tbody>
            </table>
            <h4 style={{ margin: '16px 0 8px' }}>{a.roiColumns.join(' / ')}</h4>
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
                {!roi.length && <tr><td colSpan={4} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        ) : tab === 'heatmap' ? (
          <div className="overflow-x-auto px-4 pb-4">
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
                {!heatSlice.length && <tr><td colSpan={4} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="overflow-x-auto px-4 pb-4">
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
                {!maintSlice.length && <tr><td colSpan={7} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={count} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
    </div>
  )
}
