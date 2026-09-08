// 回访评价页(`/boss/feedback`,菜单码 menu:order):契约 GET /worker-feedbacks
// (裸列表)+ POST /worker-feedbacks/{feedbackId}/review(差评复核即时落账,
// worker.yaml + worker_handlers_fact.go)。字段口径对齐 docs/contract/fields.md §7.3/§7.5
// 与 internal/domain/worker/events.go Feedback 结构体。文案/颜色走 i18n + 主题令牌。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton, IdRef } from '../../../components/business'
import { Pagination } from '../../../components/Pagination'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Checkbox } from '../../../components/ui/checkbox'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { TableStateRow } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import { pageSlice, type FeedbackRow } from '../types'
import { filterFeedback } from './filter'

export default function FeedbackPage() {
  const t = useT()
  const c = t.pages.feedbackPage
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<FeedbackRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [workerFilter, setWorkerFilter] = useState('')
  const [reviewOnly, setReviewOnly] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: FeedbackRow[] }>('/worker-feedbacks')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const review = async (id: number) => {
    if (busy || !(await confirmDialog(c.reviewConfirm))) return
    setBusy(true)
    try {
      await apiFetch(`/worker-feedbacks/${id}/review`, { method: 'POST' })
      toast.success(c.toastReviewOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const filtered = useMemo(
    () => filterFeedback(rows, { workerKw: workerFilter, reviewOnly }),
    [rows, workerFilter, reviewOnly],
  )

  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-56" value={workerFilter}
            onChange={(e) => { setWorkerFilter(e.target.value); setPage(1) }}
            placeholder={c.filterWorkerPh} />
          <label className="inline-flex h-8 cursor-pointer items-center gap-2 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]">
            <Checkbox checked={reviewOnly} onCheckedChange={(v) => { setReviewOnly(v === true); setPage(1) }} />
            <span>{c.filterReviewOnly}</span>
          </label>
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
                  <TableCell><IdRef value={x.id} /></TableCell>
                  <TableCell>{x.customerName || '—'}</TableCell>
                  <TableCell>{x.workerName || '—'}</TableCell>
                  <TableCell><IdRef value={x.ticketId} /></TableCell>
                  <TableCell>{x.groupName || '—'}</TableCell>
                  <TableCell>{x.regionName || '—'}</TableCell>
                  <TableCell>{x.legalEntityName || '—'}</TableCell>
                  <TableCell className="font-medium text-[var(--shell-heading)]">{c.scoreFmt.replace('{score}', String(x.score))}</TableCell>
                  <TableCell>
                    {x.needReview ? (
                      <span className="inline-flex items-center gap-2">
                        <span className="inline-flex h-5 items-center rounded-sm bg-[var(--shell-menu-hover-bg)] px-2 text-[11px] text-[var(--shell-heading)]">{c.needReviewYes}</span>
                        <button
                          type="button"
                          className="h-7 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-3 text-[12px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed disabled:opacity-50"
                          disabled={busy}
                          onClick={() => review(x.id)}
                        >{c.review}</button>
                      </span>
                    ) : <span className="text-[var(--shell-crumb-text)]">{c.needReviewNo}</span>}
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={c.columns.length} loading={busy} text={c.empty} />}
            </TableBody>
          </Table>
        </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(c)} />
        </div>
      </Card>
    </div>
  )
}
