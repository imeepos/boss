// 欠费停复机页:契约 GET /arrears;操作 POST /arrears/:customerId/stop|resume(W6 即时生效)。
// 指标卡标签独立 i18n(原复用列名导致「状态/欠费天数」标签错位);催收任务卡独立标题。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { pageSlice, type ArrearsRow, type ARMetrics, type CollectionTaskRow } from '../types'
import { fmtFee } from '../../../lib/format'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, ErrorBanner } from '../../../components/business'
import { ToolbarButton } from '../../../components/business/page-head'

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
      // 行内动作失败:toast 完整透出接口原因(后端 42200 等拒绝原因可见可复制)。
      toast.error(e instanceof Error ? e.message : a.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const metricCards: [string, string | number][] = metrics
    ? [
      [a.mTotalAmount, fmtFee(metrics.totalAmount)],
      [a.mCustomers, metrics.customerCount],
      [a.mStopped, metrics.stoppedCount],
      [a.mOverdueBills, metrics.overdueBillCount],
    ]
    : []

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      {metricCards.length > 0 && (
        <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
          {metricCards.map(([label, value]) => (
            <div key={label} className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-3">
              <div className="text-xs text-[var(--shell-group-title)]">{label}</div>
              <div className="mt-1 text-xl font-semibold text-[var(--shell-heading)]">{value}</div>
            </div>
          ))}
        </div>
      )}
      {tasks.length > 0 && (
        <Card className="p-4">
          <div className="mb-3 text-sm font-semibold text-[var(--shell-heading)]">{a.pendingTasksTitle}</div>
          <div className="grid gap-2 md:grid-cols-2">
            {tasks.slice(0, 6).map((task) => (
              <div key={task.id} className="flex items-center justify-between rounded-sm bg-[var(--shell-menu-hover-bg)] px-3 py-2 text-xs">
                <span>{task.customer || `#${task.customerId}`}</span>
                <span className="text-[var(--color-danger)]">{fmtFee(task.amount)} · {task.days}d</span>
              </div>
            ))}
          </div>
        </Card>
      )}
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader><TableRow>{a.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.customerId}>
                    <TableCell>{r.customer || `#${r.customerId}`}</TableCell>
                    <TableCell>{fmtFee(r.amount)}</TableCell>
                    <TableCell>{r.days}</TableCell>
                    <TableCell>{r.status || '—'}</TableCell>
                    <TableCell>
                      <span className="inline-flex items-center">
                        {r.status !== 'STOPPED' && (
                          <button disabled={busy} onClick={() => act(r.customerId, 'stop')}
                            className="px-1 text-xs text-[var(--color-danger)] bg-none border-none cursor-pointer hover:underline disabled:cursor-not-allowed disabled:opacity-50">{a.stop}</button>
                        )}
                        {r.status === 'STOPPED' && (
                          <button disabled={busy} onClick={() => act(r.customerId, 'resume')}
                            className="px-1 text-xs text-[var(--color-text-link)] bg-none border-none cursor-pointer hover:underline disabled:cursor-not-allowed disabled:opacity-50">{a.resume}</button>
                        )}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={a.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </CardFooter>
      </Card>
    </div>
  )
}
