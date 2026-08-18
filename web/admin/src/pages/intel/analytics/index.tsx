// 经营分析页:契约 GET /analytics/indicators + /analytics/heatmap + /analytics/maintenance(三页签)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice } from '../types'
import type { AnalyticsMaintRow, HeatCellRow, IndicatorRow, RegionRoiRow } from '../types'
import '../../org/org.css'

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
      <div className="org-card">
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
          <button className="org-btn" disabled={busy} onClick={() => load(tab)}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : tab === 'indicator' ? (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{a.indicatorColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {indicators.map((x) => (
                  <tr key={x.key}>
                    <td>{x.key}</td>
                    <td>{x.name}</td>
                    <td>{x.value}</td>
                    <td style={{ maxWidth: 360 }}>{x.detail || '—'}</td>
                  </tr>
                ))}
                {!indicators.length && <tr><td colSpan={4}><div className="org-empty">{a.empty}</div></td></tr>}
              </tbody>
            </table>
            <h4 style={{ margin: '16px 0 8px' }}>{a.roiColumns.join(' / ')}</h4>
            <table className="org-table">
              <thead><tr>{a.roiColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {roi.map((x) => (
                  <tr key={x.regionId}>
                    <td>{x.regionName || `#${x.regionId}`}</td>
                    <td>{x.revenue.toFixed(2)}</td>
                    <td>{x.investment.toFixed(2)}</td>
                    <td>{x.roi.toFixed(2)}</td>
                  </tr>
                ))}
                {!roi.length && <tr><td colSpan={4}><div className="org-empty">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        ) : tab === 'heatmap' ? (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{a.heatColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {heatSlice.map((x) => (
                  <tr key={x.addressId}>
                    <td>{x.name || `#${x.addressId}`}</td>
                    <td>{x.portsTotal}</td>
                    <td>{x.portsUsed}</td>
                    <td>{(x.utilization * 100).toFixed(1)}%</td>
                  </tr>
                ))}
                {!heatSlice.length && <tr><td colSpan={4}><div className="org-empty">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{a.maintColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {maintSlice.map((x) => (
                  <tr key={x.deviceNo}>
                    <td>{x.deviceNo}</td>
                    <td>{x.deviceType || '—'}</td>
                    <td>{x.healthScore}</td>
                    <td>{x.faultCount}</td>
                    <td>{x.ageYears}</td>
                    <td><StatusTag domain="maintPriority" value={x.priority} /></td>
                    <td>{x.reason || '—'}</td>
                  </tr>
                ))}
                {!maintSlice.length && <tr><td colSpan={7}><div className="org-empty">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={count} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
    </div>
  )
}
