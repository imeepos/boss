// OLT 设备页:契约 GET /device/metrics?resourceId + GET /device/maintenances(双页签)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type DeviceMetricRow, type MaintenanceRow, type ResourceRow } from '../types'
import '../../org/org.css'

export default function DevicePage() {
  const t = useT()
  const d = t.pages.devicePage
  const [tab, setTab] = useState<'metrics' | 'maint'>('metrics')
  const [devices, setDevices] = useState<ResourceRow[]>([])
  const [metrics, setMetrics] = useState<DeviceMetricRow[]>([])
  const [maints, setMaints] = useState<MaintenanceRow[]>([])
  const [error, setError] = useState('')
  const [resourceId, setResourceId] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const loadMetrics = (rid: number) => {
    setError('')
    setBusy(true)
    apiFetch<{ items: DeviceMetricRow[] }>('/device/metrics', { query: { resourceId: rid || undefined } })
      .then((x) => setMetrics(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  const loadMaints = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: MaintenanceRow[] }>('/device/maintenances')
      .then((x) => setMaints(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => {
    apiFetch<{ items: ResourceRow[] }>('/resources')
      .then((x) => setDevices(x?.items ?? []))
      .catch(() => setDevices([]))
    loadMetrics(0)
    loadMaints()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const deviceName = (rid: number) => devices.find((x) => x.id === rid)?.name ?? `#${rid}`
  const rows = tab === 'metrics' ? metrics.length : maints.length
  const slice = pageSlice<DeviceMetricRow | MaintenanceRow>(
    tab === 'metrics' ? metrics : maints, page, pageSize,
  )
  const fmtNum = (v?: number | null) => (v === undefined || v === null ? '—' : String(v))

  return (
    <div>
      <PageHead title={d.title} desc={d.desc} />
      <div className="org-card">
        <div style={{ display: 'flex', gap: 4, marginBottom: 12, borderBottom: '1px solid #f0f0f0' }}>
          {(['metrics', 'maint'] as const).map((key) => (
            <button key={key} onClick={() => { setTab(key); setPage(1) }}
              style={{
                padding: '8px 16px', fontSize: 14, cursor: 'pointer', background: 'none', border: 'none',
                borderBottom: tab === key ? '2px solid #1677ff' : '2px solid transparent',
                color: tab === key ? '#1677ff' : '#666', fontWeight: tab === key ? 600 : 400,
              }}>
              {key === 'metrics' ? d.tabMetrics : d.tabMaint}
            </button>
          ))}
          <span className="spacer" />
          {tab === 'metrics' && (
            <select className="org-select" value={resourceId ? String(resourceId) : ''}
              onChange={(e) => { const v = Number(e.target.value) || 0; setResourceId(v); setPage(1); loadMetrics(v) }}>
              <option value="">{d.allDevice}</option>
              {devices.map((x) => <option key={x.id} value={x.id}>{x.name} ({x.code})</option>)}
            </select>
          )}
          <button className="org-btn" disabled={busy}
            onClick={() => (tab === 'metrics' ? loadMetrics(resourceId) : loadMaints())}>
            {t.pages.audit.refresh}
          </button>
        </div>
        {error ? <div className="org-error">{error}</div> : tab === 'metrics' ? (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{d.metricColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {(slice as DeviceMetricRow[]).map((m) => (
                  <tr key={m.id}>
                    <td>#{m.id}</td>
                    <td>{deviceName(m.resourceId)}</td>
                    <td><StatusTag domain="resource" value={m.status} /></td>
                    <td>{fmtNum(m.opticalPower)}</td>
                    <td>{fmtNum(m.packetLoss)}</td>
                    <td>{fmtTime(m.collectedAt)}</td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{d.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{d.maintColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {(slice as MaintenanceRow[]).map((m) => (
                  <tr key={m.id}>
                    <td>{m.deviceNo}</td>
                    <td>{m.deviceType || '—'}</td>
                    <td>{m.healthScore}</td>
                    <td>{m.faultCount}</td>
                    <td>{fmtNum(m.ageYears)}</td>
                    <td><StatusTag domain="maintPriority" value={m.priority} /></td>
                    <td>{m.reason || '—'}</td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{d.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(d)} />
        </div>
      </div>
    </div>
  )
}
