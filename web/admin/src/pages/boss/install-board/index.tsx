// 施工看板:契约 GET /dispatch_tickets(派生 installed_at) + GET /install-logs。
// 状态机:DONE/CANCELED 置底;DOING 突出;展示 arrived_at + coords。
// 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md §决策 2。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'

interface TicketWithLogs {
  ticketId: number
  ticketNo: string
  workerName: string
  status: 'PENDING' | 'DOING' | 'DONE' | 'CANCELED'
  arrivedAt?: string | null
  arriveLat?: number | null
  arriveLng?: number | null
}

export default function InstallBoardPage() {
  const t = useT()
  const d = t.pages.installBoardPage
  const [rows, setRows] = useState<TicketWithLogs[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: any[] }>('/dispatch_tickets')
      .then((r) => {
        const ts: TicketWithLogs[] = (r?.items ?? []).map((x: any) => ({
          ticketId: x.ticketId,
          ticketNo: x.ticketNo,
          workerName: x.workerName || '—',
          status: x.status,
          arrivedAt: x.arrivedAt || null,
          arriveLat: x.arriveLat ?? null,
          arriveLng: x.arriveLng ?? null,
        }))
        setRows(ts)
      })
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [])

  const pending = rows.filter((r) => r.status === 'PENDING').length
  const doing = rows.filter((r) => r.status === 'DOING').length
  const arrived = rows.filter((r) => !!r.arrivedAt).length

  const start = (page - 1) * pageSize
  const slice = rows.slice(start, start + pageSize)

  return (
    <div className="space-y-4">
      <PageHead title={d.title} desc={d.subtitle} />

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div className="rounded-lg border border-shell-divider bg-shell-bg-card p-4">
          <div className="text-xs text-shell-fg-muted">{d.metricPending}</div>
          <div className="text-2xl font-semibold">{pending}</div>
        </div>
        <div className="rounded-lg border border-shell-divider bg-shell-bg-card p-4">
          <div className="text-xs text-shell-fg-muted">{d.metricDoing}</div>
          <div className="text-2xl font-semibold text-brand-primary">{doing}</div>
        </div>
        <div className="rounded-lg border border-shell-divider bg-shell-bg-card p-4">
          <div className="text-xs text-shell-fg-muted">{d.metricArrived}</div>
          <div className="text-2xl font-semibold text-status-success">{arrived}</div>
        </div>
      </div>

      {error && <div className="text-sm text-status-danger">{error}</div>}

      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-shell-fg-muted">
            <th className="py-2 pr-4">{d.colTicketNo}</th>
            <th className="py-2 pr-4">{d.colWorker}</th>
            <th className="py-2 pr-4">{d.colStatus}</th>
            <th className="py-2 pr-4">{d.colArrived}</th>
            <th className="py-2 pr-4">{d.colCoords}</th>
          </tr>
        </thead>
        <tbody>
          {slice.map((r) => (
            <tr key={r.ticketId} className="border-t border-shell-divider">
              <td className="py-2 pr-4 font-mono">{r.ticketNo}</td>
              <td className="py-2 pr-4">{r.workerName}</td>
              <td className="py-2 pr-4"><StatusTag domain="ticket" value={r.status} /></td>
              <td className="py-2 pr-4">{r.arrivedAt || '—'}</td>
              <td className="py-2 pr-4 font-mono text-xs">
                {r.arriveLat != null && r.arriveLng != null
                  ? `${r.arriveLat.toFixed(4)}, ${r.arriveLng.toFixed(4)}`
                  : '—'}
              </td>
            </tr>
          ))}
          {rows.length === 0 && !busy && (
            <tr><td colSpan={5} className="py-6 text-center text-shell-fg-muted">{d.empty}</td></tr>
          )}
        </tbody>
      </table>

      <Pagination
        page={page}
        pageSize={pageSize}
        total={rows.length}
        onPage={setPage}
        onSize={setPageSize}
        {...pagerTexts(t.pages.company)}
      />
    </div>
  )
}
