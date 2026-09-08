// 柜面收款登记表单(纪要 2026-08-28-柜面现金收款;关联字段全下拉——用户裁定)。
// 客户:复用 components/pickers/CustomerPicker(服务端检索+详情+跳转客户管理页);
// 账单:选客户后联动该客户未结账单,含"无账单·预存"选项,状态文案复用 common.statusTags;
// 网点:主数据下拉(biz_params counter.sites);柜台/班次为描述字段保留文本。
// 方式按资金通道归类:现金 cash/扫码 wechat|alipay/POS card;offline 专属师傅代收,柜面不开。
// 操作员由服务端取登录态归因,前端不传。弹层统一 ui/dialog(对齐 run-modal 先例)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown, type DropdownOption } from '../../../components/Dropdown'
import { CustomerPicker } from '../../../components/pickers/CustomerPicker'
import { FormField, SubmitButton } from '../../../components/business'
import { Input } from '../../../components/ui/input'
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from '../../../components/ui/dialog'
import { ToolbarButton } from '../../../components/business/page-head'
import { fmtFee } from '../../../lib/format'

interface Props {
  onDone: (payNo: string) => void
  onClose: () => void
}

interface BillOption { billId: number; billNo: string; period: string; amount: number; status: string }

export function CounterPaymentForm({ onDone, onClose }: Props) {
  const t = useT()
  const f = t.pages.payment
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
  // 主数据拉取失败:网点/账单任一失败即提示并阻断提交(防止资金错通道入账)。
  const [loadFail, setLoadFail] = useState('')
  const [billTick, setBillTick] = useState(0)
  const [siteTick, setSiteTick] = useState(0)

  // 网点主数据:进入表单即拉清单(biz_params counter.sites)。
  useEffect(() => {
    apiFetch<{ items: string[] }>('/daily-closings/sites')
      .then((d) => { setSiteOptions((d?.items ?? []).map((s) => ({ value: s, label: s }))); setLoadFail((cur) => (cur === f.siteLoadFail ? '' : cur)) })
      .catch(() => { setSiteOptions([]); setLoadFail(f.siteLoadFail) })
  }, [siteTick]) // eslint-disable-line react-hooks/exhaustive-deps

  // 账单联动:选客户后带出未结账单;value='0' 表示充值/预存(无账单)。
  useEffect(() => {
    setBill('')
    if (customer === '') { setBillOptions([]); return }
    apiFetch<{ items: BillOption[] }>('/bills', { query: { customerId: Number(customer) } })
      .then((d) => { setBillOptions([
        { value: '0', label: f.noBill },
        ...(d?.items ?? []).filter((b) => b.status !== 'PAID').map((b) => ({
          value: String(b.billId),
          label: [b.billNo, b.period, fmtFee(b.amount), t.common.statusTags['bill.' + b.status] ?? b.status].join(' '),
        })),
      ]); setLoadFail((cur) => (cur === f.billLoadFail ? '' : cur)) })
      .catch(() => { setBillOptions([]); setLoadFail(f.billLoadFail) })
  }, [customer, billTick]) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (loadFail) { setError(loadFail); return }
    if (customer === '') { setError(f.customerOrBill); return }
    const amt = Number(amount)
    if (!Number.isFinite(amt) || amt <= 0) { setError(f.amount); return }
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
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(v) => { if (!v && !busy) onClose() }}>
      <DialogContent className="max-w-[460px]">
        <DialogHeader>
          <DialogTitle className="text-[15px]">{f.formTitle}</DialogTitle>
        </DialogHeader>
        {error && (
          <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">
            <span className="break-all">{error}</span>
          </div>
        )}
        {loadFail && (
          <div className="flex items-center justify-between gap-2 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">
            <span className="break-all">{loadFail}</span>
            <button
              className="shrink-0 cursor-pointer border-none bg-none text-[12px] text-[var(--color-text-link)] underline"
              onClick={() => { if (loadFail === f.siteLoadFail) setSiteTick((n) => n + 1); else setBillTick((n) => n + 1) }}
            >{t.pages.pickers.common.retry}</button>
          </div>
        )}
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
            <Input inputMode="decimal" value={amount} placeholder="0.00"
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
        <DialogFooter>
          <ToolbarButton onClick={onClose}>{t.common.confirmDialog.cancel}</ToolbarButton>
          <SubmitButton state={busy ? 'loading' : 'idle'} onClick={submit}
            labels={{ idle: f.submit, loading: f.submitting, success: f.submit, failed: f.submit }} />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
