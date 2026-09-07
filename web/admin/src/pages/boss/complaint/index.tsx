// 报障与投诉页(CS 域):契约 GET /complaints(裸列表)+ POST /complaints/:ticketNo/close
// + POST /complaints/:ticketNo/status(受理流转)+ GET /complaints/:ticketNo/events(事件轨迹)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type ComplaintRow, type CSMetrics } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow } from '../../../components/business'

interface TicketEvent {
  id: number
  eventType: string
  fromStatus: string
  toStatus: string
  note: string
  createdAt: string
}

const BANNER = 'mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

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
        ].map(([label, value]) => <div key={String(label)} className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-3"><div className="text-xs text-[var(--shell-group-title)]">{label}</div><div className="mt-1 text-xl font-semibold text-[var(--shell-heading)]">{value}</div></div>)}
      </div>}
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error && <div className={BANNER}>{error}</div>}
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{c.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
            <tbody>
              {slice.map((x) => (
                <tr key={x.id}>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.ticketNo}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]" title={'customerId=' + x.customerId}>#{x.customerId}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]" title={x.orderId ? 'orderId=' + x.orderId : undefined}>{x.orderId ? '#' + x.orderId : '—'}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{c.types[x.type] ?? x.type}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="complaint" value={x.status} /></td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                    <span className="inline-flex items-center gap-2">
                      {x.status === 'OPEN' && <button disabled={busy} onClick={() => accept(x.ticketNo)}>{c.accept}</button>}
                      {x.status !== 'CLOSED' && <button disabled={busy} onClick={() => close(x.ticketNo)}>{c.close}</button>}
                      <button disabled={busy} onClick={() => openEvents(x.ticketNo)}>{c.eventsBtn}</button>
                    </span>
                  </td>
                </tr>
              ))}
              {!slice.length && <TableStateRow colSpan={6} loading={busy} text={c.empty} />}
            </tbody>
          </table>
        </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(c)} />
        </div>
      </div>
      {events && (
        <Drawer title={c.eventsTitle + ' · ' + events.no} onClose={() => setEvents(null)}>
          {events.error ? <div className={BANNER}>{events.error}</div> : events.loading ? (
            <div className="p-4 text-[13px] text-[var(--shell-group-title)]">…</div>
          ) : (
            <div className="overflow-x-auto px-4 pb-4">
              <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
                <thead><tr>{c.eventsColumns.map((x) => <th key={x} className="h-9 px-2 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
                <tbody>
                  {events.items.map((ev) => (
                    <tr key={ev.id}>
                      <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{fmtTime(ev.createdAt)}</td>
                      <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{ev.eventType}</td>
                      <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{ev.fromStatus || '—'} → {ev.toStatus || '—'}</td>
                      <td className="h-9 px-2 border-b border-[var(--shell-side-border)]">{ev.note || '—'}</td>
                    </tr>
                  ))}
                  {!events.items.length && <tr><td colSpan={4} className="h-11 px-3 text-center text-[var(--shell-group-title)]">{c.eventsEmpty}</td></tr>}
                </tbody>
              </table>
            </div>
          )}
        </Drawer>
      )}
    </div>
  )
}
