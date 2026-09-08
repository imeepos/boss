// 全额退款弹层(000112):后端 reason 必填(binding:required),旧实现发空体被 422「参数非法」
// 全量拒绝——2026-09-08 102 实测确认;填退款原因即二次确认,失败原因原位透出可复制。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from '../../../components/ui/dialog'
import { FormField, SubmitButton } from '../../../components/business'
import { Input } from '../../../components/ui/input'
import { ToolbarButton } from '../../../components/business/page-head'

export function RefundDialog({ paymentId, payNo, onClose, onDone }: {
  paymentId: number
  payNo: string
  onClose: () => void
  onDone: () => void
}) {
  const t = useT()
  const f = t.pages.payment
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const submit = async () => {
    if (busy) return
    if (!reason.trim()) { setError(f.refundReasonRequired); return }
    setBusy(true)
    setError('')
    try {
      await apiFetch(`/payments/${paymentId}/refund`, { method: 'POST', body: { reason: reason.trim() } })
      toast.success(f.refundOk)
      onDone()
    } catch (e) {
      setError(e instanceof Error ? e.message : f.fail)
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{f.refundTitle}</DialogTitle>
          <DialogDescription className="text-[13px] leading-6">{f.refundConfirm}</DialogDescription>
        </DialogHeader>
        <div className="text-xs text-[var(--shell-group-title)]">{t.pages.payment.columns[0]}: <span className="font-mono">{payNo}</span></div>
        <FormField label={f.refundReason} required hint={f.refundReasonHint}>
          <Input value={reason} autoFocus placeholder={f.refundReasonPh}
            onChange={(e) => setReason(e.target.value)} />
        </FormField>
        {error && (
          <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">
            <span className="break-all">{error}</span>
          </div>
        )}
        <DialogFooter>
          <ToolbarButton onClick={onClose}>{t.pages.company.cancel}</ToolbarButton>
          <SubmitButton state={busy ? 'loading' : 'idle'} disabled={!reason.trim()} onClick={submit}
            labels={{ idle: f.refundSubmit, loading: f.submitting, success: f.refundSubmit, failed: f.refundSubmit }} />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
