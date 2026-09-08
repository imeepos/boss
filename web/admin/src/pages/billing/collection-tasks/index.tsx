// 催收任务队列页(`/billing/collection-tasks`):契约 GET /collection-tasks?status=...
// + POST /collection-tasks/{id}/status(AR 域 ar_collection_tasks,PENDING/DOING/DONE/FAILED,
// 人工接管)。字段口径对齐 docs/contract/fields.md §8B 与 internal/domain/billing。
// 文案/颜色走 i18n + 主题令牌;状态过滤/动作按钮全部 i18n 化;优先级 URGENT/HIGH/NORMAL 走映射。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts, ErrorBanner, ToolbarButton, TableStateRow } from '../../../components/business'
import { Pagination } from '../../../components/Pagination'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { pageSlice, type CollectionTaskRow } from '../types'
import { fmtFee } from '../../../lib/format'

const STATUSES = ['PENDING', 'DOING', 'DONE', 'FAILED'] as const
type CollectionStatus = (typeof STATUSES)[number]

export default function CollectionTasksPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const c = t.pages.collectionTasksPage
  const [rows, setRows] = useState<CollectionTaskRow[]>([])
  const [status, setStatus] = useState<CollectionStatus>('PENDING')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const load = () => {
    setBusy(true)
    setError('')
    apiFetch<{ items: CollectionTaskRow[] }>(`/collection-tasks?status=${status}`)
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [status])

  const update = async (task: CollectionTaskRow, next: CollectionStatus) => {
    const msg = c.actionConfirm.replace('{status}', c.statuses[next] ?? next)
    if (!(await confirmDialog(msg, { danger: next === 'FAILED' }))) return
    setBusy(true)
    setError('')
    try {
      await apiFetch(`/collection-tasks/${task.id}/status`, { method: 'POST', body: { status: next } })
      toast.success(c.updateOk.replace('{status}', c.statuses[next] ?? next))
      load()
    } catch (e) {
      // 行内接管动作失败:toast 完整透出接口原因(原仅顶部 banner,行内语境弱)。
      toast.error(e instanceof Error ? e.message : c.actionFailMsg)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const priorityText = (v: string) => c.priorityTexts[v] ?? v
  const dueText = (v: string) => (v.length > 10 ? v.slice(0, 10) : v || '—')

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          {STATUSES.map((x) => (
            <button
              key={x}
              disabled={busy}
              onClick={() => { setStatus(x); setPage(1) }}
              className={`h-8 cursor-pointer rounded-sm border px-3 text-xs transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${status === x
                ? 'border-[var(--color-brand-gold-500)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-heading)]'
                : 'border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]'}`}
            >
              {c.statuses[x] ?? x}
            </button>
          ))}
          <span className="flex-1" />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {c.columns.map((label) => (
                    <TableHead key={label}>{label}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell className="font-mono text-[var(--shell-group-title)]">#{x.id}</TableCell>
                    <TableCell>{x.customer || `#${x.customerId}`}</TableCell>
                    <TableCell>{fmtFee(x.amount)} / {x.days}d</TableCell>
                    <TableCell>{priorityText(x.priority)}</TableCell>
                    <TableCell>{c.statuses[x.status] ?? x.status}</TableCell>
                    <TableCell>{dueText(x.dueAt)}</TableCell>
                    <TableCell>
                      <span className="inline-flex items-center gap-2">
                        {x.status === 'PENDING' && (
                          <button
                            disabled={busy}
                            onClick={() => update(x, 'DOING')}
                            className="h-7 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-3 text-[12px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed disabled:opacity-50"
                          >{c.actionStart}</button>
                        )}
                        {x.status === 'DOING' && (
                          <>
                            <button
                              disabled={busy}
                              onClick={() => update(x, 'DONE')}
                              className="h-7 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-3 text-[12px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed disabled:opacity-50"
                            >{c.actionDone}</button>
                            <button
                              disabled={busy}
                              onClick={() => update(x, 'FAILED')}
                              className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[12px] text-[var(--color-danger)] hover:border-[var(--color-border-hover)] disabled:cursor-not-allowed disabled:opacity-50"
                            >{c.actionFail}</button>
                          </>
                        )}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={c.columns.length} loading={busy} text={c.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize} onPage={setPage} onSize={setPageSize} {...pagerTexts(c)} />
        </CardFooter>
      </Card>
    </div>
  )
}
