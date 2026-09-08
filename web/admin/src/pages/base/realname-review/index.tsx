// 实名审核中心:聚合客户/师傅 verifications 列表 + 行内 PASS/驳回 + 证件照预览。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { Dropdown } from '../../../components/Dropdown'
import { fmtTime } from '../../../lib/format'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Textarea } from '../../../components/ui/textarea'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../../../components/ui/dialog'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'
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

const SUBJECT_BADGE = 'inline-flex h-6 items-center rounded-full px-2 text-[11px] font-medium'

function ResultBadge({ result, w }: { result: VerificationRow['result']; w: ReturnType<typeof useT>['pages']['realnameReview'] }) {
  if (result === 'PASS') return <span className={`${SUBJECT_BADGE} bg-[color-mix(in_srgb,var(--color-success)_15%,transparent)] text-[var(--color-success)]`}>{w.resultPass}</span>
  if (result === 'FAIL') return <span className={`${SUBJECT_BADGE} bg-[color-mix(in_srgb,var(--color-danger)_15%,transparent)] text-[var(--color-danger)]`}>{w.resultFail}</span>
  return <span className={`${SUBJECT_BADGE} bg-[color-mix(in_srgb,var(--color-brand-gold-600)_15%,transparent)] text-[var(--color-brand-gold-600)]`}>{w.resultPending}</span>
}

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
  const [actState, setActState] = useState<SubmitState>('idle')
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
    setRejectReason(''); setFormError(''); setActState('idle')
  }

  const verify = async (mode: 'pass' | 'fail') => {
    if (actSubject == null || busy) return
    if (mode === 'fail' && !rejectReason.trim()) {
      setFormError(w.rejectReasonLabel)
      return
    }
    setBusy(true)
    setFormError('')
    setActState('loading')
    try {
      await apiFetch(`/verifications/${actSubject.type}/${actSubject.id}/verify`, {
        method: 'POST',
        body: mode === 'pass' ? { result: 'PASS' } : { result: 'FAIL', reason: rejectReason.trim() },
      })
      closeDialog()
      load()
    } catch (e) {
      const msg = e instanceof Error ? e.message : w.actionFail
      setFormError(msg)
      setActState('failed')
      setTimeout(() => setActState((s) => (s === 'failed' ? 'idle' : s)), 2500)
    } finally { setBusy(false) }
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const openAct = (r: VerificationRow, mode: 'pass' | 'fail') => {
    setActId(r.id)
    setActSubject({ type: r.subjectType, id: r.subjectId })
    setActMode(mode)
  }

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-3 p-4">
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
          <div className="flex min-w-60 flex-1 items-center gap-2">
            <Input
              placeholder={w.keywordPlaceholder}
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') { setPage(1); load() } }}
            />
            <ToolbarButton primary disabled={busy} onClick={() => { setPage(1); load() }}>{w.refresh}</ToolbarButton>
          </div>
        </div>
      </Card>

      <Card>
        <div className="flex items-center gap-2 p-4 pb-3">
          <span className="text-sm text-[var(--shell-group-title)]">{w.total.replace('{count}', String(total))}</span>
          <span className="flex-1" />
          <ToolbarButton disabled={busy} onClick={load}>{w.refresh}</ToolbarButton>
        </div>
        {error ? <div className="mx-4 mb-3"><ErrorBanner message={error} /></div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {w.columns.map((c: string) => <TableHead key={c}>{c}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>
                      <span className={`${SUBJECT_BADGE} ${r.subjectType === 'customer' ? 'bg-[color-mix(in_srgb,var(--shell-fab-bg)_15%,transparent)] text-[var(--shell-fab-bg)]' : 'bg-[color-mix(in_srgb,var(--color-brand-gold-500)_15%,transparent)] text-[var(--color-brand-gold-600)]'}`}>
                        {r.subjectType === 'customer' ? w.subjectCustomer : w.subjectWorker}
                      </span>
                    </TableCell>
                    <TableCell>{r.subjectName || '—'}</TableCell>
                    <TableCell>{r.subjectPhone || '—'}</TableCell>
                    <TableCell>{r.idCardNoMasked || '—'}</TableCell>
                    <TableCell>{r.realName || '—'}</TableCell>
                    <TableCell>{r.method || '—'}</TableCell>
                    <TableCell><ResultBadge result={r.result} w={w} /></TableCell>
                    <TableCell>
                      {r.operatorName || '—'}
                      {r.idCardFrontId > 0 && (
                        <button className="ml-2 cursor-pointer border-none bg-none text-[11px] text-[var(--color-text-link)] hover:underline" onClick={() => setPreviewId(r.idCardFrontId)}>{w.idCardFront}</button>
                      )}
                      {r.idCardBackId > 0 && (
                        <button className="ml-2 cursor-pointer border-none bg-none text-[11px] text-[var(--color-text-link)] hover:underline" onClick={() => setPreviewId(r.idCardBackId)}>{w.idCardBack}</button>
                      )}
                    </TableCell>
                    <TableCell>{fmtTime(r.verifiedAt)}</TableCell>
                    <TableCell>
                      {r.result === 'PENDING' ? (
                        <div className="flex gap-2">
                          <button className="h-7 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-3 text-[12px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => openAct(r, 'pass')}>{w.pass}</button>
                          <button className="h-7 cursor-pointer rounded-sm border border-[var(--color-danger)] bg-transparent px-3 text-[12px] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)]" onClick={() => openAct(r, 'fail')}>{w.fail}</button>
                        </div>
                      ) : <span className="text-[12px] text-[var(--shell-group-title)]">—</span>}
                    </TableCell>
                  </TableRow>
                ))}
                {!rows.length && (
                  <TableRow><TableCell colSpan={w.columns.length}><EmptyState text={w.empty} /></TableCell></TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        )}
        {totalPages > 1 && (
          <div className="flex items-center justify-end gap-2 px-4 py-3 text-[12px] text-[var(--shell-group-title)]">
            <span>{w.pageOf.replace('{page}', String(page)).replace('{total}', String(totalPages))}</span>
            <ToolbarButton disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>{w.prev}</ToolbarButton>
            <ToolbarButton disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>{w.next}</ToolbarButton>
          </div>
        )}
      </Card>

      <Dialog open={actId != null && actMode !== null} onOpenChange={(v) => { if (!v) closeDialog() }}>
        <DialogContent className="w-96">
          <DialogHeader>
            <DialogTitle>{actMode === 'pass' ? w.pass : w.rejectTitle}</DialogTitle>
          </DialogHeader>
          {actMode === 'fail' && (
            <label className="mb-1 block text-sm text-[var(--shell-group-title)]">
              {w.rejectReasonLabel}
              <Textarea
                className="mt-1"
                rows={3}
                placeholder={w.rejectReasonPh}
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
              />
            </label>
          )}
          {formError && <div className="mb-3"><ErrorBanner message={formError} /></div>}
          <div className="flex justify-end gap-2">
            <ToolbarButton onClick={closeDialog}>{w.cancel}</ToolbarButton>
            {actMode === 'pass' ? (
              <SubmitButton
                state={actState}
                labels={{ idle: w.confirm, loading: t.common.loading, success: w.confirm, failed: w.confirm }}
                disabled={busy}
                onClick={() => verify('pass')}
              />
            ) : (
              <button
                className="inline-flex h-8 cursor-pointer items-center justify-center gap-1.5 rounded-sm border-none bg-[var(--color-danger)] px-4 text-[13px] text-white transition-colors disabled:cursor-not-allowed disabled:opacity-70"
                disabled={busy || actState === 'loading'}
                onClick={() => verify('fail')}
              >{w.confirm}</button>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {previewId != null && <AttachmentPreview attachmentId={previewId} onClose={() => setPreviewId(null)} />}
    </div>
  )
}
