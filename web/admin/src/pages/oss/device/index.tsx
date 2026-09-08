// OLT 设备页:契约 GET /device/metrics?resourceId + GET /device/maintenances(双页签)。
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { fmtTime } from '../../../lib/format'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { pageSlice, type DeviceMetricRow, type MaintenanceRow, type ResourceRow } from '../types'
import { TableStateRow, TabBar, ErrorBanner, ToolbarButton } from '../../../components/business'

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

  const handleLoadFail = (e: unknown) => {
    const msg = e instanceof Error ? e.message : d.loadFail
    setError(msg)
    toast.error(d.loadFail, { description: msg })
  }

  const loadMetrics = (rid: number) => {
    setError('')
    setBusy(true)
    apiFetch<{ items: DeviceMetricRow[] }>('/device/metrics', { query: { resourceId: rid || undefined } })
      .then((x) => setMetrics(x?.items ?? []))
      .catch(handleLoadFail)
      .finally(() => setBusy(false))
  }
  const loadMaints = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: MaintenanceRow[] }>('/device/maintenances')
      .then((x) => setMaints(x?.items ?? []))
      .catch(handleLoadFail)
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
      <Card className="mb-4">
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
          <ToolbarButton onClick={() => (tab === 'metrics' ? loadMetrics(resourceId) : loadMaints())} disabled={busy}>
            {t.pages.audit.refresh}
          </ToolbarButton>
        </div>
        {error ? <div className="px-4 pb-3"><ErrorBanner message={error} /></div> : tab === 'metrics' ? (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {d.metricColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {(slice as DeviceMetricRow[]).map((m) => (
                  <TableRow key={m.id}>
                    <TableCell>{devices.find((x) => x.id === m.resourceId)?.code ?? `#${m.resourceId}`}</TableCell>
                    <TableCell>{deviceName(m.resourceId)}</TableCell>
                    <TableCell><StatusTag domain="resource" value={m.status} /></TableCell>
                    <TableCell>{fmtNum(m.opticalPower)}</TableCell>
                    <TableCell>{fmtNum(m.packetLoss)}</TableCell>
                    <TableCell>{fmtTime(m.collectedAt)}</TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={d.empty} />}
              </TableBody>
            </Table>
          </div>
        ) : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {d.maintColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {(slice as MaintenanceRow[]).map((m) => (
                  <TableRow key={m.id}>
                    <TableCell>{m.deviceNo}</TableCell>
                    <TableCell>{m.deviceType || '—'}</TableCell>
                    <TableCell>{m.healthScore}</TableCell>
                    <TableCell>{m.faultCount}</TableCell>
                    <TableCell>{fmtNum(m.ageYears)}</TableCell>
                    <TableCell><StatusTag domain="maintPriority" value={m.priority} /></TableCell>
                    <TableCell>{m.reason || '—'}</TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={d.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(d)} />
        </CardFooter>
      </Card>
    </div>
  )
}
