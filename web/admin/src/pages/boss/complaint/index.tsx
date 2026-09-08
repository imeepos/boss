// 报障与投诉页(CS 域):契约 GET /complaints(裸列表)+ POST /complaints/:ticketNo/close
// + POST /complaints/:ticketNo/status(受理流转)+ GET /complaints/:ticketNo/events(事件轨迹)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton, IdRef } from '../../../components/business'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type ComplaintRow, type CSMetrics } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, LoadingState, EmptyState } from '../../../components/business'
import { StatCard } from '../../../components/business/charts'

interface TicketEvent {
  id: number
  eventType: string
  fromStatus: string
  toStatus: string
  note: string
  createdAt: string
}

/** 行内动作链接:busy 提交中禁用。 */
function RowAction({ label, disabled, onClick }: { label: string; disabled?: boolean; onClick: () => void }) {
  return (
    <button type="button" disabled={disabled} onClick={onClick}
      className="cursor-pointer border-none bg-none px-0 text-xs text-[var(--color-text-link)] hover:underline disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:no-underline">
      {label}
    </button>
  )
}

export default function ComplaintPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const c = t.pages.complaintPage
  const [rows, setRows] = useState<ComplaintRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [metrics, setMetrics] = useState<CSMetrics | null>(null)
  const [events, setEvents] = useState<{ no: string; items: TicketEvent[]; loading: boolean; error: string } | null>(null)

  const load = () => {
    setError('')
    setBusy(true)
    Promise.all([
      apiFetch<ComplaintRow[]>('/complaints'),
      apiFetch<CSMetrics>('/complaint-metrics'),
    ])
      .then(([x, m]) => { setRows(Array.isArray(x) ? x : []); setMetrics(m) })
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const close = async (ticketNo: string) => {
    if (busy || !(await confirmDialog(c.closeConfirm, { danger: true }))) return
    setBusy(true)
    try {
      await apiFetch('/complaints/' + encodeURIComponent(ticketNo) + '/close', { method: 'POST' })
      toast.success(c.closeOk.replace('{no}', ticketNo))
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  // 受理 = 流转到 PROCESSING(后端 POST /complaints/:ticketNo/status,actor=0 系统代)。
  const accept = async (ticketNo: string) => {
    if (busy) return
    setBusy(true)
    try {
      await apiFetch('/complaints/' + encodeURIComponent(ticketNo) + '/status', {
        method: 'POST', body: { status: 'PROCESSING' },
      })
      toast.success(c.acceptOk.replace('{no}', ticketNo))
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const openEvents = (ticketNo: string) => {
    setEvents({ no: ticketNo, items: [], loading: true, error: '' })
    apiFetch<{ items: TicketEvent[] }>('/complaints/' + encodeURIComponent(ticketNo) + '/events')
      .then((d) => setEvents({ no: ticketNo, items: d?.items ?? [], loading: false, error: '' }))
      .catch((e) => setEvents({ no: ticketNo, items: [], loading: false, error: e instanceof Error ? e.message : c.loadFail }))
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      {metrics && <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        {[
          [c.columns[1], metrics.openCount], [c.columns[4], metrics.processingCount],
          [c.columns[3], metrics.slaBreachedOpen], [c.columns[5], metrics.avgCloseHours.toFixed(1) + 'h'],
        ].map(([label, value]) => <StatCard key={String(label)} label={String(label)} value={value} />)}
      </div>}
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error && <ErrorBanner message={error} />}
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>{c.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((x) => (
                <TableRow key={x.id}>
                  <TableCell>{x.ticketNo}</TableCell>
                  <TableCell><IdRef value={x.customerId} /></TableCell>
                  <TableCell>{x.orderId ? <IdRef value={x.orderId} /> : '—'}</TableCell>
                  <TableCell>{c.types[x.type] ?? x.type}</TableCell>
                  <TableCell><StatusTag domain="complaint" value={x.status} /></TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-2">
                      {x.status === 'OPEN' && <RowAction label={c.accept} disabled={busy} onClick={() => accept(x.ticketNo)} />}
                      {x.status !== 'CLOSED' && <RowAction label={c.close} disabled={busy} onClick={() => close(x.ticketNo)} />}
                      <RowAction label={c.eventsBtn} disabled={busy} onClick={() => openEvents(x.ticketNo)} />
                    </span>
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={6} loading={busy} text={c.empty} />}
            </TableBody>
          </Table>
        </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(c)} />
        </div>
      </Card>
      {events && (
        <Drawer title={c.eventsTitle + ' · ' + events.no} onClose={() => setEvents(null)}>
          {events.error ? <ErrorBanner message={events.error} /> : events.loading ? (
            <LoadingState />
          ) : (
            <div className="px-4 pb-4">
              <Table>
                <TableHeader>
                  <TableRow>{c.eventsColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
                </TableHeader>
                <TableBody>
                  {events.items.map((ev) => (
                    <TableRow key={ev.id}>
                      <TableCell>{fmtTime(ev.createdAt)}</TableCell>
                      <TableCell>{ev.eventType}</TableCell>
                      <TableCell>{ev.fromStatus || '—'} → {ev.toStatus || '—'}</TableCell>
                      <TableCell className="whitespace-normal">{ev.note || '—'}</TableCell>
                    </TableRow>
                  ))}
                  {!events.items.length && <TableRow><TableCell colSpan={4}><EmptyState text={c.eventsEmpty} /></TableCell></TableRow>}
                </TableBody>
              </Table>
            </div>
          )}
        </Drawer>
      )}
    </div>
  )
}
