// 柜面收款登记表单(纪要 2026-08-28-柜面现金收款;关联字段全下拉——用户裁定)。
// 客户:复用 components/pickers/CustomerPicker(服务端检索+详情+跳转客户管理页);
// 账单:选客户后联动该客户未结账单,含"无账单·预存"选项;
// 网点:主数据下拉(biz_params counter.sites);柜台/班次为描述字段保留文本。
// 方式按资金通道归类:现金 cash/扫码 wechat|alipay/POS card;offline 专属师傅代收,柜面不开。
// 操作员由服务端取登录态归因,前端不传。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown, type DropdownOption } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'
import { CustomerPicker } from '../../../components/pickers/CustomerPicker'
import { FormField, SubmitButton, ToolbarButton } from '../../../components/business'
import { Input } from '../../../components/ui/input'
import { fmtFee } from '../../../lib/format'

interface Props {
  onDone: (payNo: string) => void
  onClose: () => void
}

interface BillOption { billId: number; billNo: string; period: string; amount: number; status: string }

export function CounterPaymentForm({ onDone, onClose }: Props) {
  const t = useT()
  const f = t.pages.payment
  const confirm = useConfirm()
  const [customer, setCustomer] = useState('')
  const [bill, setBill] = useState('')
  const [billOptions, setBillOptions] = useState<DropdownOption[]>([])
  const [amount, setAmount] = useState('')
  const [method, setMethod] = useState('cash')
  const [site, setSite] = useState('')
  const [siteOptions, setSiteOptions] = useState<DropdownOption[]>([])
  const [counter, setCounter] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  // 网点主数据:进入表单即拉清单(biz_params counter.sites)。
  useEffect(() => {
    apiFetch<{ items: string[] }>('/daily-closings/sites')
      .then((d) => setSiteOptions((d?.items ?? []).map((s) => ({ value: s, label: s }))))
      .catch(() => setSiteOptions([]))
  }, [])

  // 账单联动:选客户后带出未结账单;value='0' 表示充值/预存(无账单)。
  useEffect(() => {
    setBill('')
    if (customer === '') { setBillOptions([]); return }
    apiFetch<{ items: BillOption[] }>('/bills', { query: { customerId: Number(customer) } })
      .then((d) => setBillOptions([
        { value: '0', label: f.noBill },
        ...(d?.items ?? []).filter((b) => b.status !== 'PAID').map((b) => ({
          value: String(b.billId), label: b.billNo + ' ' + b.period + ' ' + fmtFee(b.amount) + ' ' + b.status,
        })),
      ]))
      .catch(() => setBillOptions([]))
  }, [customer]) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (customer === '') { setError(f.customerOrBill); return }
    const amt = Number(amount)
    if (!Number.isFinite(amt) || amt <= 0) { setError(f.amount); return }
    if (!(await confirm(f.confirmText, { title: f.formTitle }))) return
    setBusy(true)
    setError('')
    try {
      const d = await apiFetch<{ id: number; payNo: string }>('/payments', {
        method: 'POST',
        body: {
          billId: Number(bill || 0), customerId: Number(customer), amount: amt, method,
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
          <div className="col-span-2">
            <FormField label={f.customer}>
              <CustomerPicker value={customer} onChange={setCustomer} />
            </FormField>
          </div>
          <div className="col-span-2">
            <FormField label={f.bill}>
              <Dropdown
                value={bill}
                options={billOptions}
                onChange={setBill}
                ariaLabel={f.bill}
                disabled={customer === ''}
              />
            </FormField>
          </div>
          <FormField label={f.amount}>
            <Input inputMode="decimal" value={amount}
              onChange={(e) => setAmount(e.target.value)} />
          </FormField>
          <FormField label={f.method}>
            <Dropdown value={method} ariaLabel={f.method} onChange={setMethod} options={[
              { value: 'cash', label: f.methods.cash ?? 'cash' },
              { value: 'wechat', label: f.methods.wechat ?? 'wechat' },
              { value: 'alipay', label: f.methods.alipay ?? 'alipay' },
              { value: 'card', label: f.methods.card ?? 'card' },
            ]} />
          </FormField>
          <FormField label={f.site}>
            <Dropdown value={site} options={siteOptions} onChange={setSite} ariaLabel={f.site} />
          </FormField>
          <FormField label={f.counter}>
            <Input value={counter} onChange={(e) => setCounter(e.target.value)} />
          </FormField>
        </div>
        <div className="mt-5 flex justify-center gap-3">
          <ToolbarButton onClick={onClose}>{t.common.confirmDialog.cancel}</ToolbarButton>
          <SubmitButton state={busy ? 'loading' : 'idle'} onClick={submit}
            labels={{ idle: f.submit, loading: f.submitting, success: f.submit, failed: f.submit }} />
        </div>
      </div>
    </div>
  )
}
