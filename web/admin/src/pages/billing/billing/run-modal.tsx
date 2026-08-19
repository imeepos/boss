// 出账+自动开票(POST /billing-runs):批量生成账单并开票(CT-007 同账期幂等)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'

/** InvoicePanel 监听此事件刷新(出账会产生新发票)。 */
export const INVOICES_REFRESH = 'boss:invoices-refresh'

interface RunResult {
  bills: number
  invoices: { issued: number; failedIds?: number[] }
}

export function BillingRunModal({ open, onClose, onDone }: {
  open: boolean
  onClose: () => void
  onDone: () => void
}) {
  const t = useT()
  const r = t.pages.billPage.run
  const [period, setPeriod] = useState('')
  const [result, setResult] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  if (!open) return null
  const submit = () => {
    if (busy || !period.trim()) return
    setBusy(true); setError('')
    apiFetch<RunResult>('/billing-runs', { method: 'POST', body: { period: period.trim() } })
      .then((d) => {
        const failed = d?.invoices?.failedIds?.length ?? 0
        setResult(r.result.replace('{bills}', String(d?.bills ?? 0)).replace('{issued}', String(d?.invoices?.issued ?? 0)).replace('{failed}', String(failed)))
        onDone()
      })
      .catch((e) => setError(e instanceof Error ? e.message : r.fail))
      .finally(() => setBusy(false))
  }

  return (
    <div className="fixed inset-0 z-[120] flex items-center justify-center bg-black/45">
      <div className="w-90 rounded-md bg-[var(--shell-card-bg)] p-5">
        <p>{r.title}</p>
        <input className="org-input" value={period} autoFocus placeholder={r.periodPh}
          onChange={(e) => setPeriod(e.target.value)} />
        {result && <p style={{ color: '#30a46c', margin: '8px 0 0' }}>{result}</p>}
        {error && <p className="org-error" style={{ margin: '8px 0 0' }}>{error}</p>}
        <div className="mt-4 flex justify-end gap-2">
          <button className="org-btn" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="org-btn org-btn-primary" disabled={busy || !period.trim()} onClick={submit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </div>
      </div>
    </div>
  )
}
