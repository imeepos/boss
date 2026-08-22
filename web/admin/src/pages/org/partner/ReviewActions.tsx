// 审核行内操作:待审 = 通过/驳回(驳回弹意见框);已审 = 无操作。
import { useState } from 'react'
import type { PartnerApplication } from '../../../api/partner'
import { useT } from '../../../i18n'
import { ToolbarButton } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'

export function PartnerReviewActions({
  row, onApprove, onReject,
}: {
  row: PartnerApplication
  onApprove: (id: number) => Promise<void>
  onReject: (id: number, note: string) => Promise<void>
}) {
  const t = useT()
  const [note, setNote] = useState('')
  const [rejecting, setRejecting] = useState(false)
  const [busy, setBusy] = useState(false)

  if (row.status !== 'PENDING') return <span className="text-xs text-[var(--shell-crumb-text)]">-</span>

  const approve = async () => {
    if (busy) return
    setBusy(true)
    try { await onApprove(row.id) } finally { setBusy(false) }
  }

  const reject = async () => {
    if (busy || !note.trim()) return
    setBusy(true)
    try {
      await onReject(row.id, note.trim())
      setRejecting(false)
      setNote('')
    } finally { setBusy(false) }
  }

  if (!rejecting) {
    return (
      <span className="inline-flex gap-2">
        <ToolbarButton primary disabled={busy} onClick={approve}>
          {t.pages.partnerReview.approve}
        </ToolbarButton>
        <ToolbarButton disabled={busy} onClick={() => setRejecting(true)}>
          {t.pages.partnerReview.reject}
        </ToolbarButton>
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-2">
      <Input
        className="w-45"
        placeholder={t.pages.partnerReview.rejectReasonPlaceholder}
        value={note}
        onChange={(e) => setNote(e.target.value)}
      />
      <ToolbarButton primary disabled={busy || !note.trim()} onClick={reject}>
        {t.pages.partnerReview.confirmReject}
      </ToolbarButton>
      <ToolbarButton disabled={busy} onClick={() => setRejecting(false)}>
        {t.pages.partnerReview.cancel}
      </ToolbarButton>
    </span>
  )
}
