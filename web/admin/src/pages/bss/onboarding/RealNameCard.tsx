// 实名卡:展示实名状态;未实名时代录(复用 RealNameDrawer)+ 后台核验通过/驳回。
// 核验端点 POST /customers/:id/real-name/verify {result:PASS|FAIL, reason}(menu:customer)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useConfirm } from '../../../components/ConfirmDialog'
import { StatusTag } from '../../../components/StatusTag'
import { ErrorBanner } from '../../../components/business/page-head'
import { RealNameDrawer } from '../customer/RealNameDrawer'
import type { CustomerRow } from '../customer/types'
import { useT } from '../../../i18n'

export function RealNameCard({
  customer, onChanged,
}: { customer: CustomerRow | null; onChanged: () => void }) {
  const t = useT()
  const w = t.pages.onboardingPage
  const confirmDialog = useConfirm()
  const [rnOpen, setRnOpen] = useState(false)
  const [rejecting, setRejecting] = useState(false)
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  if (!customer) return null

  const verify = async (result: 'PASS' | 'FAIL') => {
    if (busy) return
    if (result === 'FAIL' && !reason.trim()) { setError(w.eReasonRequired); return }
    setBusy(true); setError(''); setMsg('')
    try {
      await apiFetch(`/customers/${customer.id}/real-name/verify`, {
        method: 'POST', body: { result, reason: reason.trim() || undefined },
      })
      setMsg(result === 'PASS' ? w.verified : w.realNameRejected)
      setRejecting(false); setReason('')
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : w.actionFail)
    } finally { setBusy(false) }
  }

  const pending = customer.realNameStatus === 'PENDING'
  return (
    <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
      <div className="flex flex-wrap items-center gap-2 p-4">
        <span className="text-[13px] font-medium text-[var(--shell-heading)]">2. {w.realNameTitle}</span>
        <StatusTag domain="realName" value={customer.realNameStatus} />
        <span className="spacer" />
        <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => { setRnOpen(true); setMsg(''); setError('') }}>{w.realNameEntry}</button>
        {pending && (
          <>
            <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--color-success)] px-4 text-[13px] text-white disabled:opacity-50" disabled={busy}
              onClick={() => { void confirmDialog(w.verifyPassConfirm).then((ok) => { if (ok) verify('PASS') }) }}>{w.verifyPass}</button>
            <button type="button" className="h-8 cursor-pointer rounded-sm border border-[var(--color-danger)] bg-transparent px-4 text-[13px] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)]" onClick={() => { setRejecting((v) => !v); setReason('') }}>{w.verifyReject}</button>
          </>
        )}
      </div>
      {rejecting && (
        <div className="px-4 pb-3">
          <textarea className="mb-2 h-16 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] p-2 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={reason} placeholder={w.rejectReasonPh} onChange={(e) => setReason(e.target.value)} />
          <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--color-danger)] px-4 text-[13px] text-white disabled:opacity-50" disabled={busy} onClick={() => verify('FAIL')}>{w.verifyReject}</button>
        </div>
      )}
      {msg && (
        <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-success)]">{msg}</div>
      )}
      {error && <ErrorBanner message={error} className="mx-4 mb-3" />}
      {rnOpen && (
        <RealNameDrawer customerId={customer.id} customerName={customer.name} onClose={() => setRnOpen(false)} onSubmitted={onChanged} />
      )}
    </div>
  )
}
