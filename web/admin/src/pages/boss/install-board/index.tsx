// 施工看板:契约 GET /dispatch-tickets(全量工单,含到场打卡事实) + GET /install-logs。
// 状态机:DONE/CANCELED 置底;DOING 突出;展示 arrived_at + coords。
// 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md §决策 2。
// 样式对齐 provision:大卡片 + StatCard + TableStateRow。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { fmtTime } from '../../../lib/format'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton } from '../../../components/business'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { StatCard } from '../../../components/business/charts'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'

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
    apiFetch<{ items: TicketWithLogs[] }>('/dispatch-tickets')
      .then((r) => {
        const ts: TicketWithLogs[] = (r?.items ?? []).map((x) => ({
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

      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? (
          <ErrorBanner message={error} />
        ) : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{d.colTicketNo}</TableHead>
                  <TableHead>{d.colWorker}</TableHead>
                  <TableHead>{d.colStatus}</TableHead>
                  <TableHead>{d.colArrived}</TableHead>
                  <TableHead>{d.colCoords}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.ticketId}>
                    <TableCell className="font-mono">{r.ticketNo}</TableCell>
                    <TableCell>{r.workerName}</TableCell>
                    <TableCell><StatusTag domain="ticket" value={r.status} /></TableCell>
                    <TableCell>{fmtTime(r.arrivedAt)}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {r.arriveLat != null && r.arriveLng != null
                        ? `${r.arriveLat.toFixed(4)}, ${r.arriveLng.toFixed(4)}`
                        : '—'}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={d.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination
            total={rows.length}
            page={page}
            pageSize={pageSize}
            onPage={setPage}
            onSize={(s) => { setPageSize(s); setPage(1) }}
            {...pagerTexts(t.pages.company)}
          />
        </div>
      </Card>
    </div>
  )
}
