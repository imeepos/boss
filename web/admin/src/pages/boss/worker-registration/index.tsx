// 师傅注册审核页:待审核列表(后端 GET /worker-registrations?status=PENDING) + 通过/驳回动作。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { fmtTime } from '../../../lib/format'

interface WorkerRegistrationRow {
  id: number
  name: string
  phone: string
  idCardNo: string
  groupId: number
  regionId: number
  status: string
  reviewNote: string
  submittedAt: string
  reviewedAt: string
}

export default function WorkerRegistrationPage() {
  const t = useT()
  const w = t.pages.workerRegPage
  const [rows, setRows] = useState<WorkerRegistrationRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [actId, setActId] = useState<number | null>(null)
  const [actMode, setActMode] = useState<'approve' | 'reject' | null>(null)
  const [groupId, setGroupId] = useState('')
  const [regionId, setRegionId] = useState('')
  const [rejectNote, setRejectNote] = useState('')
  const [formError, setFormError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: WorkerRegistrationRow[] }>('/worker-registrations', { query: { status: 'PENDING' } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleApprove = async () => {
    if (actId == null || busy) return
    if (!/^\d+$/.test(groupId) || Number(groupId) <= 0) { setFormError(w.eGroupRequired); return }
    if (!/^\d+$/.test(regionId) || Number(regionId) <= 0) { setFormError(w.eRegionRequired); return }
    setBusy(true)
    setFormError('')
    try {
      await apiFetch(`/worker-registrations/${actId}/approve`, {
        method: 'POST', body: { groupId: Number(groupId), regionId: Number(regionId) },
      })
      closeDialog()
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : w.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const handleReject = async () => {
    if (actId == null || busy) return
    if (!rejectNote.trim()) { setFormError(w.eNoteRequired); return }
    setBusy(true)
    setFormError('')
    try {
      await apiFetch(`/worker-registrations/${actId}/reject`, {
        method: 'POST', body: { note: rejectNote.trim() },
      })
      closeDialog()
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : w.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const closeDialog = () => {
    setActId(null)
    setActMode(null)
    setGroupId('')
    setRegionId('')
    setRejectNote('')
    setFormError('')
  }

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex items-center gap-2 p-4">
          <span className="text-sm text-[var(--shell-group-title)]">{w.total.replace('{count}', String(rows.length))}</span>
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{w.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">
                <tr>
                  {w.columns.map((x: string) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.id}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.phone}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.idCardNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(r.submittedAt)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <div className="flex gap-2">
                        <button className="h-7 cursor-pointer rounded-sm border border-[var(--color-brand-bg)] bg-[var(--color-brand-bg)] px-3 text-[12px] text-white hover:opacity-80" onClick={() => { setActId(r.id); setActMode('approve') }}>{w.approve}</button>
                        <button className="h-7 cursor-pointer rounded-sm border border-[var(--color-danger)] bg-transparent px-3 text-[12px] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)]" onClick={() => { setActId(r.id); setActMode('reject') }}>{w.reject}</button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!rows.length && (
                  <tr><td colSpan={6} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                    <div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{w.empty}</div>
                  </td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* 审批对话框 */}
      {actId != null && actMode && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={closeDialog}>
          <div className="w-96 rounded-lg bg-white p-6 shadow-xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-4 text-lg font-semibold">{actMode === 'approve' ? w.approveTitle : w.rejectTitle}</h3>
            {actMode === 'approve' ? (
              <>
                <label className="mb-1 block text-sm text-gray-600">{w.groupId}</label>
                <input className="mb-3 w-full rounded border border-gray-300 px-3 py-2 text-sm" type="number" placeholder={w.groupIdPlaceholder} value={groupId} onChange={(e) => setGroupId(e.target.value)} />
                <label className="mb-1 block text-sm text-gray-600">{w.regionId}</label>
                <input className="mb-3 w-full rounded border border-gray-300 px-3 py-2 text-sm" type="number" placeholder={w.regionIdPlaceholder} value={regionId} onChange={(e) => setRegionId(e.target.value)} />
              </>
            ) : (
              <>
                <label className="mb-1 block text-sm text-gray-600">{w.rejectNote}</label>
                <textarea className="mb-3 w-full rounded border border-gray-300 px-3 py-2 text-sm" rows={3} placeholder={w.rejectNotePlaceholder} value={rejectNote} onChange={(e) => setRejectNote(e.target.value)} />
              </>
            )}
            {formError && <div className="mb-3 rounded bg-red-50 px-3 py-2 text-sm text-red-600">{formError}</div>}
            <div className="flex justify-end gap-2">
              <button className="rounded px-4 py-2 text-sm text-gray-600 hover:bg-gray-100" onClick={closeDialog}>{w.cancel}</button>
              <button className={`rounded px-4 py-2 text-sm text-white ${actMode === 'approve' ? 'bg-blue-600 hover:bg-blue-700' : 'bg-red-600 hover:bg-red-700'}`} disabled={busy} onClick={actMode === 'approve' ? handleApprove : handleReject}>{actMode === 'approve' ? w.confirmApprove : w.confirmReject}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
