// 调价抽屉:POST /products/{id}/price-history(customer.yaml changeProductPrice)。
// 只收新月费+原因,立即生效;生效时间由服务端落 now,与调价台账同口径。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
import { useT } from '../../../i18n'
import { fmtFee } from '../../../lib/format'

export function PriceChangeDrawer({
  productId, productName, currentFee, onClose, onDone,
}: {
  productId: number
  productName: string
  currentFee: number
  onClose: () => void
  onDone: () => void
}) {
  const t = useT()
  const p = t.pages.product
  const [fee, setFee] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const feeOk = fee !== '' && Number(fee) > 0 && !Number.isNaN(Number(fee))

  const submit = async () => {
    if (busy || !feeOk) return
    setBusy(true)
    setError('')
    try {
      await apiFetch(`/products/${productId}/price-history`, {
        method: 'POST',
        body: { newPrice: Number(fee), reason: reason.trim() },
      })
      onDone()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

  return (
    <Drawer title={`${p.priceChangeTitle} · ${productName}`} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
            disabled={busy || !feeOk} onClick={submit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <FormField label={p.currentFee}>
          <div className="flex h-8 items-center text-[13px] text-[var(--shell-content-text)]">{fmtFee(currentFee)}</div>
        </FormField>
        <FormField label={p.fNewFee} required>
          <input className={input} value={fee} placeholder="0.00" inputMode="decimal"
            onChange={(e) => setFee(e.target.value)} />
          {!feeOk && fee !== '' && <span className="text-[11px] text-[var(--color-danger)]">{p.eNewFee}</span>}
        </FormField>
        <FormField label={p.fReason}>
          <input className={input} value={reason} placeholder={p.pReason}
            onChange={(e) => setReason(e.target.value)} />
        </FormField>
        <p className="text-xs text-[var(--shell-group-title)]">{p.priceChangeTip}</p>
        {error && <ErrorBanner message={error} />}
      </div>
    </Drawer>
  )
}
