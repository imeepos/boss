// 欠费停复机页:契约 GET /arrears;操作 POST /arrears/:customerId/stop|resume(W6 即时生效)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ArrearsRow, type ARMetrics, type CollectionTaskRow } from '../types'
import { fmtFee } from '../../../lib/format'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, ErrorBanner } from '../../../components/business'

export default function ArrearsPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const a = t.pages.arrearsPage
  const [rows, setRows] = useState<ArrearsRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [metrics, setMetrics] = useState<ARMetrics | null>(null)
  const [tasks, setTasks] = useState<CollectionTaskRow[]>([])

  const load = () => {
    setError('')
    setBusy(true)
    Promise.all([
      apiFetch<{ items: ArrearsRow[] }>('/arrears'),
      apiFetch<ARMetrics>('/ar-metrics'),
      apiFetch<{ items: CollectionTaskRow[] }>('/collection-tasks?status=PENDING'),
    ])
      .then(([d, m, q]) => { setRows(d?.items ?? []); setMetrics(m); setTasks(q?.items ?? []) })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const act = async (customerId: number, action: 'stop' | 'resume') => {
    if (busy) return
    const confirmMsg = action === 'stop' ? a.stopConfirm : a.resumeConfirm
    if (!(await confirmDialog(confirmMsg, { danger: action === 'stop' }))) return
    setBusy(true)
    try {
      await apiFetch(`/arrears/${customerId}/${action}`, { method: 'POST' })
      toast.success(action === 'stop' ? a.stopOk : a.resumeOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : a.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      {metrics && <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        {[
          [a.columns[1], fmtFee(metrics.totalAmount)], [a.columns[0], metrics.customerCount],
          [a.columns[3], metrics.stoppedCount], [a.columns[2], metrics.overdueBillCount],
        ].map(([label, value]) => <div key={String(label)} className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-3"><div className="text-xs text-[var(--shell-group-title)]">{label}</div><div className="mt-1 text-xl font-semibold text-[var(--shell-heading)]">{value}</div></div>)}
      </div>}
      {tasks.length > 0 && <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4"><div className="mb-3 text-sm font-semibold text-[var(--shell-heading)]">{a.title}</div><div className="grid gap-2 md:grid-cols-2">{tasks.slice(0, 6).map((task) => <div key={task.id} className="flex items-center justify-between rounded-sm bg-[var(--shell-menu-hover-bg)] px-3 py-2 text-xs"><span>{task.customer || `#${task.customerId}`}</span><span className="text-[var(--color-danger)]">{fmtFee(task.amount)} · {task.days}d</span></div>)}</div></div>}
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.customerId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.customer || `#${r.customerId}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.amount)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.days}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.status || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        {r.status !== 'STOPPED' && <button disabled={busy} onClick={() => act(r.customerId, 'stop')}>{a.stop}</button>}
                        {r.status === 'STOPPED' && <button disabled={busy} onClick={() => act(r.customerId, 'resume')}>{a.resume}</button>}
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={a.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
    </div>
  )
}
