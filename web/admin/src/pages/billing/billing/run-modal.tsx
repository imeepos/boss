// 出账+自动开票(POST /billing-runs):批量生成账单并开票(CT-007 同账期幂等)。
// 表单弹层模式件参考实现(page-patterns.md §2):Dialog+FormField+Input+Button。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '../../../components/ui/dialog'
import { FormField } from '../../../components/business/form-field'
import { Input } from '../../../components/ui/input'
import { Button } from '../../../components/ui/button'

/** InvoicePanel 监听此事件刷新(出账会产生新发票)。 */
export const INVOICES_REFRESH = 'boss:invoices-refresh'

interface RunResult {
  bills: number
  invoices: { issued: number; failedIds?: number[] }
}

// 账期格式与 paycheck 页 PERIOD_RE 同口径(YYYY-MM)。
const PERIOD_RE = /^\d{4}-(0[1-9]|1[0-2])$/

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

  const submit = () => {
    if (busy || !period.trim()) return
    if (!PERIOD_RE.test(period.trim())) { setError(r.periodInvalid); return }
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
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{r.title}</DialogTitle>
        </DialogHeader>
        <div className="mt-2 flex flex-col gap-3">
          <FormField label={r.title} required error={error || undefined}>
            <Input value={period} autoFocus placeholder={r.periodPh}
              onChange={(e) => setPeriod(e.target.value)} />
          </FormField>
          {result && <p className="m-0 text-[13px] text-[var(--color-success)]">{result}</p>}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>{t.pages.company.cancel}</Button>
          <Button disabled={busy || !period.trim() || !PERIOD_RE.test(period.trim())} onClick={submit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
