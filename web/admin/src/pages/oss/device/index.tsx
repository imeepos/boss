// OLT 设备页:契约 GET /device/metrics?resourceId + GET /device/maintenances(双页签)。
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type DeviceMetricRow, type MaintenanceRow, type ResourceRow } from '../types'
import { TableStateRow, TabBar, ErrorBanner } from '../../../components/business'

export default function DevicePage() {
  const t = useT()
  const d = t.pages.devicePage
  const [tab, setTab] = useState<'metrics' | 'maint'>('metrics')
  const [devices, setDevices] = useState<ResourceRow[]>([])
  const [metrics, setMetrics] = useState<DeviceMetricRow[]>([])
  const [maints, setMaints] = useState<MaintenanceRow[]>([])
  const [error, setError] = useState('')
  const [resourceId, setResourceId] = useState(0)
  // 跨页联动:ODN 页关联 OLT chip 经 ?resourceId= 预过滤(spec oss-odn-v2 §3)。
  const [sp] = useSearchParams()
  const ridParam = Number(sp.get('resourceId') ?? 0) || 0
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
    if (ridParam) setResourceId(ridParam)
    loadMetrics(ridParam)
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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="px-4 pt-3">
          <TabBar<'metrics' | 'maint'>
            tabs={[{ key: 'metrics' as const, label: d.tabMetrics }, { key: 'maint' as const, label: d.tabMaint }]}
            value={tab}
            onChange={(key) => { setTab(key); setPage(1) }}
          />
        </div>
        <div className="flex flex-wrap items-center gap-2 px-4 pb-3">
          <span className="flex-1" />
          {tab === 'metrics' && (
            <div className="w-52">
              <Dropdown
                value={resourceId ? String(resourceId) : ''}
                options={[{ value: '', label: d.allDevice }, ...devices.map((x) => ({ value: String(x.id), label: `${x.name} (${x.code})` }))]}
                onChange={(v) => { const n = Number(v) || 0; setResourceId(n); setPage(1); loadMetrics(n) }}
                ariaLabel={d.allDevice}
              />
            </div>
          )}
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy}
            onClick={() => (tab === 'metrics' ? loadMetrics(resourceId) : loadMaints())}>
            {t.pages.audit.refresh}
          </button>
        </div>
        {error ? <ErrorBanner message={error} /> : tab === 'metrics' ? (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{d.metricColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {(slice as DeviceMetricRow[]).map((m) => (
                  <tr key={m.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{devices.find((x) => x.id === m.resourceId)?.code ?? `#${m.resourceId}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{deviceName(m.resourceId)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="resource" value={m.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtNum(m.opticalPower)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtNum(m.packetLoss)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(m.collectedAt)}</td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={d.empty} />}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{d.maintColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {(slice as MaintenanceRow[]).map((m) => (
                  <tr key={m.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.deviceNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.deviceType || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.healthScore}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.faultCount}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtNum(m.ageYears)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="maintPriority" value={m.priority} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.reason || '—'}</td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={d.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(d)} />
        </div>
      </div>
    </div>
  )
}
