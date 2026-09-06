// 入库单驳回抽屉:契约 POST /procurement/receipts/{id}/reject {reason}(原因必填,255 字内)。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { buildRejectPayload, rejectReasonErr, type RejectFormState } from './purchaseLogic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
const errBanner = 'rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

export function RejectReceiptDrawer({ receipt, onClose, onSaved }: {
  receipt: { id: number; receiptNo: string }
  onClose: () => void
  onSaved: () => void
}) {
  const t = useT()
  const d = t.pages.purchasePage
  const [form, setForm] = useState<RejectFormState>({ reason: '' })
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async () => {
    if (busy) return
    const key = rejectReasonErr(form)
    setErr(key)
    if (key) return
    setBusy(true)
    try {
      await apiFetch('/procurement/receipts/' + receipt.id + '/reject', {
        method: 'POST',
        body: buildRejectPayload(form),
      })
      toast.success(d.rejectOk)
      onSaved()
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : d.opFail)
      setBusy(false)
    }
  }

  return (
    <Drawer title={d.rejectTitle.replace('{no}', receipt.receiptNo || '#' + receipt.id)} onClose={onClose} width={440}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{d.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--color-danger)] px-4 text-[13px] text-white hover:opacity-90" disabled={busy} onClick={submit}>
            {busy ? d.submitting : d.reject}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.fRejectReason}</label>
          <input className={input} value={form.reason} placeholder={d.pRejectReason}
            onChange={(e) => setForm({ reason: e.target.value })} />
        </div>
        {err && <div className={errBanner}>{d[err as 'eRejectReason']}</div>}
      </div>
    </Drawer>
  )
}
