// 回访评价页(`/boss/feedback`,菜单码 menu:order):契约 GET /worker-feedbacks
// (裸列表)+ POST /worker-feedbacks/{feedbackId}/review(差评复核即时落账,
// worker.yaml + worker_handlers_fact.go)。字段口径对齐 docs/contract/fields.md §7.3/§7.5
// 与 internal/domain/worker/events.go Feedback 结构体。文案/颜色走 i18n + 主题令牌。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input
            type="text"
            value={workerFilter}
            onChange={(e) => { setWorkerFilter(e.target.value); setPage(1) }}
            placeholder={c.filterWorkerPh}
            className="h-8 w-56 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-crumb-text)] focus:border-[var(--color-border-focus)]"
          />
          <label className="inline-flex h-8 cursor-pointer items-center gap-2 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]">
            <input
              type="checkbox"
              checked={reviewOnly}
              onChange={(e) => { setReviewOnly(e.target.checked); setPage(1) }}
              className="h-3.5 w-3.5 cursor-pointer accent-[var(--shell-fab-bg)]"
            />
            <span>{c.filterReviewOnly}</span>
          </label>
          <span className="spacer" />
          <button
            className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
            disabled={busy}
            onClick={load}
          >{t.pages.audit.refresh}</button>
        </div>
        {error && (
          <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
        )}
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">
                <tr>{c.columns.map((x) => (
                  <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap">{x}</th>
                ))}</tr>
              </thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id} className="border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">
                    <td className="h-11 px-3 whitespace-nowrap">#{x.id}</td>
                    <td className="h-11 px-3 whitespace-nowrap">{x.customerName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap">{x.workerName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap">#{x.ticketId}</td>
                    <td className="h-11 px-3 whitespace-nowrap">{x.groupName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap">{x.regionName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap">{x.legalEntityName || '—'}</td>
                    <td className="h-11 px-3 font-medium whitespace-nowrap text-[var(--shell-heading)]">{c.scoreFmt.replace('{score}', String(x.score))}</td>
                    <td className="h-11 px-3 whitespace-nowrap">
                      {x.needReview ? (
                        <span className="inline-flex items-center gap-2">
                          <span className="inline-flex h-5 items-center rounded-sm bg-[var(--shell-menu-hover-bg)] px-2 text-[11px] text-[var(--shell-heading)]">{c.needReviewYes}</span>
                          <button
                            className="h-7 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-3 text-[12px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed disabled:opacity-50"
                            disabled={busy}
                            onClick={() => review(x.id)}
                          >{c.review}</button>
                        </span>
                      ) : <span className="text-[var(--shell-crumb-text)]">{c.needReviewNo}</span>}
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={c.columns.length} loading={busy} text={c.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(c)} />
        </div>
      </div>
    </div>
  )
}