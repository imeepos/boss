// 施工看板:契约 GET /dispatch_tickets(派生 installed_at) + GET /install-logs。
// 状态机:DONE/CANCELED 置底;DOING 突出;展示 arrived_at + coords。
// 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md §决策 2。
// 样式对齐 provision:大卡片 + StatCard + TableStateRow。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { StatCard } from '../../../components/business/charts'

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
    <div>
      <PageHead title={d.title} desc={d.subtitle} />

      <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
        <StatCard label={d.metricPending} value={pending} />
        <StatCard label={d.metricDoing} value={doing} />
        <StatCard label={d.metricArrived} value={arrived} />
      </div>

      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <button
            type="button"
            onClick={load}
            className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
          >
            {t.pages.audit.refresh}
          </button>
        </div>
        {error ? (
          <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
        ) : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead>
                <tr>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colTicketNo}</th>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colWorker}</th>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colStatus}</th>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colArrived}</th>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colCoords}</th>
                </tr>
              </thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.ticketId} className="hover:bg-[var(--shell-menu-hover-bg)]">
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] font-mono">{r.ticketNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.workerName}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]"><StatusTag domain="ticket" value={r.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.arrivedAt || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] font-mono text-xs">
                      {r.arriveLat != null && r.arriveLng != null
                        ? `${r.arriveLat.toFixed(4)}, ${r.arriveLng.toFixed(4)}`
                        : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={d.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination
            total={rows.length}
            page={page}
            pageSize={pageSize}
            onPage={setPage}
            onSize={setPageSize}
            {...pagerTexts(t.pages.company)}
          />
        </div>
      </div>
    </div>
  )
}
