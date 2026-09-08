// 入库单驳回抽屉:契约 POST /procurement/receipts/{id}/reject {reason}(原因必填,255 字内)。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton } from '../../../components/business/submit-button'
import { buildRejectPayload, rejectReasonErr, type RejectFormState } from './purchaseLogic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

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

  const submitState = busy ? 'loading' : (err ? 'failed' : 'idle')
  const submitLabels = { idle: d.reject, loading: d.submitting, success: d.rejectOk, failed: d.opFail }

  return (
    <Drawer title={d.rejectTitle.replace('{no}', receipt.receiptNo || '#' + receipt.id)} onClose={onClose} width={440}
      footer={
        <>
          <ToolbarButton onClick={onClose} disabled={busy}>{d.cancel}</ToolbarButton>
          <SubmitButton state={submitState} labels={submitLabels} disabled={busy} onClick={submit} />
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <FormField label={d.fRejectReason} required>
          <input className={input} value={form.reason} placeholder={d.pRejectReason}
            onChange={(e) => setForm({ reason: e.target.value })} />
        </FormField>
        {err && <ErrorBanner message={d[err as 'eRejectReason']} />}
      </div>
    </Drawer>
  )
}
