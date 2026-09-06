// 报废弹窗:必填原因(<=64 字),契约 POST /assets/:assetId/scrap {reason}。
// 终态幂等由后端兜底;前端 SCRAPPED 行不出现报废入口。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../../../components/ui/dialog'
import { useT } from '../../../i18n'
import type { AssetRow } from '../types'
import { scrapReasonErr } from './logic'

const input = 'h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export function ScrapDialog({ asset, onClose, onSaved }: {
  asset: AssetRow
  onClose: () => void
  onSaved: () => void
}) {
  const t = useT()
  const a = t.pages.assetPage
  const [reason, setReason] = useState('')
  const [invalid, setInvalid] = useState(false)
  const [apiError, setApiError] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async () => {
    if (busy) return
    const bad = scrapReasonErr(reason)
    setInvalid(bad)
    if (bad) return
    setBusy(true)
    setApiError('')
    try {
      await apiFetch('/assets/' + String(asset.assetId) + '/scrap', { method: 'POST', body: { reason: reason.trim() } })
      toast.success(a.scrapOk)
      onSaved()
      onClose()
    } catch (err) {
      setApiError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-sm border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-0">
        <DialogHeader className="space-y-0 border-b border-[var(--shell-side-border)] px-5 py-4">
          <DialogTitle className="text-[15px] font-semibold text-[var(--shell-heading)]">{a.scrapTitle}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-2 px-5 py-4">
          <DialogDescription className="text-[13px] text-[var(--shell-content-text)]">{asset.assetCode}</DialogDescription>
          <label className="text-[13px] text-[var(--shell-content-text)]"><span className="mr-0.5 text-[var(--color-danger)]">*</span>{a.fReason}</label>
          <input className={input} value={reason} maxLength={64} placeholder={a.pReason}
            onChange={(e) => { setReason(e.target.value); setInvalid(false) }} />
          {invalid && <span className="text-[11px] text-[var(--color-danger)]">{a.eReason}</span>}
          {apiError && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{apiError}</div>}
        </div>
        <DialogFooter className="gap-2 border-t border-[var(--shell-side-border)] px-5 py-3 sm:space-x-0">
          <button className="h-8 min-w-20 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 min-w-20 cursor-pointer rounded-sm border-none bg-[var(--color-danger)] px-4 text-[13px] text-white hover:opacity-90" disabled={busy} onClick={submit}>
            {busy ? t.pages.account.submitting : a.scrapAction}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}