// 柜面收款登记表单(纪要 2026-08-28-柜面现金收款)。
// 方式按资金通道归类:现金 cash/扫码 wechat|alipay/POS card;offline 专属师傅代收,柜面不开。
// 操作员由服务端取登录态归因,前端不传;网点/柜台班次为凭证要素必填。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'

interface Props {
  onDone: (payNo: string) => void
  onClose: () => void
}

const inputCls = 'h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
const labelCls = 'mb-1 block text-xs text-[var(--shell-group-title)]'

export function CounterPaymentForm({ onDone, onClose }: Props) {
  const t = useT()
  const f = t.pages.payment
  const confirm = useConfirm()
  const [customer, setCustomer] = useState('')
  const [bill, setBill] = useState('')
  const [amount, setAmount] = useState('')
  const [method, setMethod] = useState('cash')
  const [site, setSite] = useState('')
  const [counter, setCounter] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const num = (v: string) => (v.trim() === '' ? 0 : Number(v))
  const submit = async () => {
    if (customer.trim() === '' && bill.trim() === '') { setError(f.customerOrBill); return }
    const amt = Number(amount)
    if (!Number.isFinite(amt) || amt <= 0) { setError(f.amount); return }
    if (!(await confirm(f.confirmText, { title: f.formTitle }))) return
    setBusy(true)
    setError('')
    try {
      const d = await apiFetch<{ id: number; payNo: string }>('/payments', {
        method: 'POST',
        body: {
          billId: num(bill), customerId: num(customer), amount: amt, method,
          siteName: site.trim(), counterCode: counter.trim(),
        },
      })
      onDone(d?.payNo ?? '')
    } catch (e) {
      setError(e instanceof Error ? e.message : f.fail)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="fixed inset-0 z-page-modal flex items-center justify-center bg-black/40" onClick={onClose}>
      <div className="w-[440px] rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-[var(--shell-card-shadow)]" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 text-center text-[15px] font-medium text-[var(--shell-heading)]">{f.formTitle}</div>
        {error && <div className="mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
        <div className="grid grid-cols-2 gap-x-4 gap-y-3">
          <div>
            <label className={labelCls}>{f.customer}</label>
            <input className={inputCls} type="number" value={customer} onChange={(e) => setCustomer(e.target.value)} />
          </div>
          <div>
            <label className={labelCls}>{f.bill}</label>
            <input className={inputCls} type="number" value={bill} onChange={(e) => setBill(e.target.value)} />
          </div>
          <div>
            <label className={labelCls}>{f.amount}</label>
            <input className={inputCls} type="number" min="0" step="0.01" value={amount} onChange={(e) => setAmount(e.target.value)} />
          </div>
          <div>
            <label className={labelCls}>{f.method}</label>
            <Dropdown value={method} ariaLabel={f.method} onChange={setMethod} options={[
              { value: 'cash', label: f.methods.cash ?? 'cash' },
              { value: 'wechat', label: f.methods.wechat ?? 'wechat' },
              { value: 'alipay', label: f.methods.alipay ?? 'alipay' },
              { value: 'card', label: f.methods.card ?? 'card' },
            ]} />
          </div>
          <div>
            <label className={labelCls}>{f.site}</label>
            <input className={inputCls} value={site} onChange={(e) => setSite(e.target.value)} />
          </div>
          <div>
            <label className={labelCls}>{f.counter}</label>
            <input className={inputCls} value={counter} onChange={(e) => setCounter(e.target.value)} />
          </div>
        </div>
        <div className="mt-5 flex justify-center gap-3">
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-5 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={onClose}>{t.common.confirmDialog.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm bg-[var(--color-brand-bg)] px-5 text-[13px] text-white hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60" disabled={busy} onClick={submit}>{busy ? f.submitting : f.submit}</button>
        </div>
      </div>
    </div>
  )
}
