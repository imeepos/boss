// 实名审核中心:聚合客户/师傅 verifications 列表 + 行内 PASS/驳回 + 证件照预览。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { EmptyState } from '../../../components/business'
import { Dropdown } from '../../../components/Dropdown'
import { fmtTime } from '../../../lib/format'
import { AttachmentPreview } from './AttachmentPreview'

type SubjectType = '' | 'customer' | 'worker'
type ResultFilter = '' | 'PENDING' | 'PASS' | 'FAIL'

interface VerificationRow {
  id: number
  subjectType: 'customer' | 'worker'
  subjectId: number
  subjectName: string
  subjectPhone: string
  idCardNoMasked: string
  method: string
  realName: string
  result: 'PENDING' | 'PASS' | 'FAIL'
  rejectReason: string
  verifiedAt: string
  operatorName: string
  operatorAccountId: number
  idCardFrontId: number
  idCardBackId: number
}

interface ListResp {
  items: VerificationRow[]
  total: number
  page: number
  pageSize: number
}

const PAGE_SIZE = 20

export default function RealnameReviewPage() {
  const t = useT()
  const w = t.pages.realnameReview

  const [subjectType, setSubjectType] = useState<SubjectType>('')
  const [result, setResult] = useState<ResultFilter>('PENDING')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [rows, setRows] = useState<VerificationRow[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [actId, setActId] = useState<number | null>(null)
  const [actSubject, setActSubject] = useState<{ type: 'customer' | 'worker'; id: number } | null>(null)
  const [actMode, setActMode] = useState<'pass' | 'fail' | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [formError, setFormError] = useState('')
  const [previewId, setPreviewId] = useState<number | null>(null)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<ListResp>('/verifications', {
      query: {
        subjectType: subjectType || undefined,
        result: result || undefined,
        keyword: keyword.trim() || undefined,
        page,
        pageSize: PAGE_SIZE,
      },
    })
      .then((d) => { setRows(d?.items ?? []); setTotal(d?.total ?? 0) })
      .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => { load() }, [subjectType, result, page]) // eslint-disable-line react-hooks/exhaustive-deps

  const closeDialog = () => {
    setActId(null); setActSubject(null); setActMode(null)
    setRejectReason(''); setFormError('')
  }

  const handlePass = async () => {
    if (actSubject == null || busy) return
    setBusy(true); setFormError('')
    try {
      await apiFetch(`/verifications/${actSubject.type}/${actSubject.id}/verify`, {
        method: 'POST', body: { result: 'PASS' },
      })
      closeDialog(); load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : w.actionFail)
    } finally { setBusy(false) }
  }

  const handleFail = async () => {
    if (actSubject == null || busy) return
    if (!rejectReason.trim()) {
      setFormError(w.rejectReasonLabel); return
    }
    setBusy(true); setFormError('')
    try {
      await apiFetch(`/verifications/${actSubject.type}/${actSubject.id}/verify`, {
        method: 'POST', body: { result: 'FAIL', reason: rejectReason.trim() },
      })
      closeDialog(); load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : w.actionFail)
    } finally { setBusy(false) }
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />

      {/* 筛选条:主体 / 状态 / 关键词 */}
      <div className="mb-4 flex flex-wrap items-center gap-3 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4 shadow-[var(--shell-card-shadow)]">
        <div className="flex items-center gap-2">
          <span className="text-[13px] text-[var(--shell-group-title)]">{w.subject}</span>
          <Dropdown
            value={subjectType}
            options={[{ value: '', label: w.allSubject }, { value: 'customer', label: w.subjectCustomer }, { value: 'worker', label: w.subjectWorker }]}
            onChange={(v) => { setSubjectType(v as SubjectType); setPage(1) }}
            ariaLabel={w.subject}
          />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-[13px] text-[var(--shell-group-title)]">{w.result}</span>
          <Dropdown
            value={result}
            options={[{ value: '', label: w.allResult }, { value: 'PENDING', label: w.resultPending }, { value: 'PASS', label: w.resultPass }, { value: 'FAIL', label: w.resultFail }]}
            onChange={(v) => { setResult(v as ResultFilter); setPage(1) }}
            ariaLabel={w.result}
          />
        </div>
        <div className="flex flex-1 items-center gap-2">
          <input
            type="text"
            className="h-9 min-w-[240px] flex-1 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-hover)]"
            placeholder={w.keywordPlaceholder}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') { setPage(1); load() } }}
          />
          <button
            className="h-9 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
            disabled={busy}
            onClick={() => { setPage(1); load() }}
          >
            {w.refresh}
          </button>
        </div>
      </div>

      {/* 列表 */}
      <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex items-center gap-2 p-4 pb-3">
          <span className="text-sm text-[var(--shell-group-title)]">{w.total.replace('{count}', String(total))}</span>
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{w.refresh}</button>
        </div>

        {error ? (
          <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
        ) : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">
                <tr>{w.columns.map((c: string) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className={`inline-flex h-6 items-center rounded-full px-2 text-[11px] font-medium ${r.subjectType === 'customer' ? 'bg-[color-mix(in_srgb,var(--shell-fab-bg)_15%,transparent)] text-[var(--shell-fab-bg)]' : 'bg-[color-mix(in_srgb,var(--color-brand-gold-500)_15%,transparent)] text-[var(--color-brand-gold-600)]'}`}>
                        {r.subjectType === 'customer' ? w.subjectCustomer : w.subjectWorker}
                      </span>
                    </td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">{r.subjectName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">{r.subjectPhone || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">{r.idCardNoMasked || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">{r.realName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">{r.method || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <ResultBadge result={r.result} w={w} />
                    </td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {r.operatorName || '—'}
                      {r.idCardFrontId > 0 && (
                        <button className="ml-2 cursor-pointer text-[11px] text-[var(--color-text-link)] hover:underline" onClick={() => setPreviewId(r.idCardFrontId)}>{w.idCardFront}</button>
                      )}
                      {r.idCardBackId > 0 && (
                        <button className="ml-2 cursor-pointer text-[11px] text-[var(--color-text-link)] hover:underline" onClick={() => setPreviewId(r.idCardBackId)}>{w.idCardBack}</button>
                      )}
                    </td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(r.verifiedAt)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {r.result === 'PENDING' ? (
                        <div className="flex gap-2">
                          <button className="h-7 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-3 text-[12px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => { setActId(r.id); setActSubject({ type: r.subjectType, id: r.subjectId }); setActMode('pass') }}>{w.pass}</button>
                          <button className="h-7 cursor-pointer rounded-sm border border-[var(--color-danger)] bg-transparent px-3 text-[12px] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)]" onClick={() => { setActId(r.id); setActSubject({ type: r.subjectType, id: r.subjectId }); setActMode('fail') }}>{w.fail}</button>
                        </div>
                      ) : <span className="text-[12px] text-[var(--shell-group-title)]">—</span>}
                    </td>
                  </tr>
                ))}
                {!rows.length && (
                  <tr><td colSpan={w.columns.length} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">
                    <EmptyState text={w.empty} />
                  </td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}

        {/* 分页 */}
        {totalPages > 1 && (
          <div className="flex items-center justify-end gap-2 px-4 py-3 text-[12px] text-[var(--shell-group-title)]">
            <span>{w.pageOf.replace('{page}', String(page)).replace('{total}', String(totalPages))}</span>
            <button className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[12px] disabled:opacity-50" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>{w.prev}</button>
            <button className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[12px] disabled:opacity-50" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>{w.next}</button>
          </div>
        )}
      </div>

      {/* 驳回原因对话框 */}
      {actId != null && actMode && actSubject && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={closeDialog}>
          <div className="w-96 rounded-lg border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-6 shadow-[var(--shell-card-shadow)]" onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-4 text-lg font-semibold text-[var(--shell-heading)]">{actMode === 'pass' ? w.pass : w.rejectTitle}</h3>
            {actMode === 'fail' && (
              <>
                <label className="mb-1 block text-sm text-[var(--shell-group-title)]">{w.rejectReasonLabel}</label>
                <textarea
                  className="mb-3 w-full rounded border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 py-2 text-sm text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
                  rows={3}
                  placeholder={w.rejectReasonPh}
                  value={rejectReason}
                  onChange={(e) => setRejectReason(e.target.value)}
                />
              </>
            )}
            {formError && <div className="mb-3 rounded border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-sm text-[var(--color-danger)]">{formError}</div>}
            <div className="flex justify-end gap-2">
              <button className="rounded px-4 py-2 text-sm text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]" onClick={closeDialog}>{w.cancel}</button>
              <button
                className={`rounded px-4 py-2 text-sm text-[var(--shell-fab-icon)] ${actMode === 'pass' ? 'bg-[var(--shell-fab-bg)] hover:bg-[var(--shell-fab-bg-hover)]' : 'bg-[var(--color-danger)] hover:opacity-80'}`}
                disabled={busy}
                onClick={actMode === 'pass' ? handlePass : handleFail}
              >
                {w.confirm}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 证件照预览 */}
      {previewId != null && <AttachmentPreview attachmentId={previewId} onClose={() => setPreviewId(null)} />}
    </div>
  )
}

function ResultBadge({ result, w }: { result: VerificationRow['result']; w: ReturnType<typeof useT>['pages']['realnameReview'] }) {
  if (result === 'PASS') return <span className="inline-flex h-6 items-center rounded-full bg-[color-mix(in_srgb,var(--color-success)_15%,transparent)] px-2 text-[11px] font-medium text-[var(--color-success)]">{w.resultPass}</span>
  if (result === 'FAIL') return <span className="inline-flex h-6 items-center rounded-full bg-[color-mix(in_srgb,var(--color-danger)_15%,transparent)] px-2 text-[11px] font-medium text-[var(--color-danger)]">{w.resultFail}</span>
  return <span className="inline-flex h-6 items-center rounded-full bg-[color-mix(in_srgb,var(--color-brand-gold-600)_15%,transparent)] px-2 text-[11px] font-medium text-[var(--color-brand-gold-600)]">{w.resultPending}</span>
}