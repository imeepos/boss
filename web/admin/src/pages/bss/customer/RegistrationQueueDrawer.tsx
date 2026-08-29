// 客户注册审核抽屉:GET /customer-registrations(status 过滤) + approve/reject。
// 审核通过后端建 customers 主档并回填 customerId(fields.md §7.6);驳回必填审核意见。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'
import { fmtTime } from '../../../lib/format'
import { useT } from '../../../i18n'
import type { RegistrationRow } from './types'

const REG_STATUSES = ['PENDING', 'APPROVED', 'REJECTED'] as const

export function RegistrationQueueDrawer({
  open, onClose, onChanged,
}: { open: boolean; onClose: () => void; onChanged: () => void }) {
  const t = useT()
  const c = t.pages.customer
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<RegistrationRow[]>([])
  const [status, setStatus] = useState('PENDING')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [rejectId, setRejectId] = useState<number | null>(null)
  const [note, setNote] = useState('')
  const [rejectError, setRejectError] = useState('')

  const load = (st: string) => {
    setError('')
    setBusy(true)
    apiFetch<{ items: RegistrationRow[] }>('/customer-registrations', { query: { status: st || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : c.regLoadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => {
    if (open) load(status)
  }, [open, status]) // eslint-disable-line react-hooks/exhaustive-deps

  const approve = async (row: RegistrationRow) => {
    if (busy) return
    setBusy(true); setError(''); setNotice('')
    try {
      const d = await apiFetch<{ customerId: number }>(`/customer-registrations/${row.id}/approve`, { method: 'POST' })
      setNotice(c.regApproved.replace('{name}', row.name).replace('{id}', String(d?.customerId ?? 0)))
      load(status)
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.regActionFail)
    } finally { setBusy(false) }
  }

  const reject = async () => {
    if (rejectId == null || busy) return
    if (!note.trim()) { setRejectError(c.regNoteRequired); return }
    setBusy(true); setRejectError('')
    try {
      await apiFetch(`/customer-registrations/${rejectId}/reject`, { method: 'POST', body: { note: note.trim() } })
      setRejectId(null); setNote('')
      setNotice(c.regRejected)
      load(status)
      onChanged()
    } catch (e) {
      setRejectError(e instanceof Error ? e.message : c.regActionFail)
    } finally { setBusy(false) }
  }

  if (!open) return null
  return (
    <Drawer title={c.regTitle} onClose={onClose}
      footer={<button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={onClose}>{t.pages.company.cancel}</button>}>
      <div className="mb-3 flex items-center gap-2">
        <Dropdown value={status} ariaLabel={c.regStatusAll}
          options={[{ value: '', label: c.regStatusAll }, ...REG_STATUSES.map((s, i) => ({ value: s, label: c.regStatusOptions[i] }))]}
          onChange={setStatus} />
        <span className="spacer" />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" disabled={busy} onClick={() => load(status)}>{t.pages.audit.refresh}</button>
      </div>
      {notice && <div className="mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-success)]">{notice}</div>}
      {error && <div className="mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
          <thead><tr>{c.regColumns.map((x) => (
            <th key={x} className="h-10 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>
          ))}</tr></thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id}>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.name}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.phone}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.idCardNo || '—'}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.source || '—'}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{c.regStatusOptions[REG_STATUSES.indexOf(r.status as typeof REG_STATUSES[number])] ?? r.status}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{fmtTime(r.submittedAt)}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">
                  {r.status === 'PENDING' ? (
                    <span className="inline-flex items-center gap-2">
                      <button type="button" disabled={busy} onClick={() => approve(r)}>{c.regApprove}</button>
                      <span className="text-[var(--shell-side-border)]">|</span>
                      <button type="button" onClick={() => { setRejectId(r.id); setNote(''); setRejectError('') }}>{c.regReject}</button>
                    </span>
                  ) : r.customerId > 0 ? `#${r.customerId}` : (r.reviewNote || '—')}
                </td>
              </tr>
            ))}
            {!rows.length && (
              <tr><td colSpan={7} className="h-11 px-3 text-center text-[var(--shell-group-title)]">{busy ? t.common.loading : c.regEmpty}</td></tr>
            )}
          </tbody>
        </table>
      </div>
      {rejectId != null && (
        <div className="mt-4 rounded-md border border-[var(--shell-card-border)] p-3">
          <div className="mb-2 text-[13px] font-medium text-[var(--shell-heading)]">{c.regRejectTitle}</div>
          <textarea className="mb-2 h-20 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] p-2 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={note} placeholder={c.regNotePh} onChange={(e) => setNote(e.target.value)} />
          {rejectError && <div className="mb-2 text-[12px] text-[var(--color-danger)]">{rejectError}</div>}
          <div className="flex justify-end gap-2">
            <button type="button" className="h-8 rounded-sm border border-[var(--shell-input-border)] px-4 text-[13px]" onClick={() => setRejectId(null)}>{t.pages.company.cancel}</button>
            <button type="button" className="h-8 rounded-sm border-none bg-[var(--color-danger)] px-4 text-[13px] text-white disabled:opacity-50" disabled={busy}
              onClick={() => { void confirmDialog(c.regRejectConfirm, { danger: true }).then((ok) => { if (ok) reject() }) }}>{c.regReject}</button>
          </div>
        </div>
      )}
    </Drawer>
  )
}
