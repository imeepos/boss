// 应付登记表单组:付款/核减/发票(fields.md §1.5.8e;40900=超余额/核减超限,由后端拒绝并 toast 透出)。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { SubmitButton } from '../../../components/business'
import { Input } from '../../../components/ui/input'
import { Card } from '../../../components/ui/card'

const METHOD_KEYS = ['TRANSFER', 'CASH', 'CHEQUE', 'OTHER'] as const

export function PayRegistrationForms({ payableId, onDone }: {
  payableId: number
  onDone: () => void
}) {
  const t = useT()
  const p = t.pages.payablesPage
  const [pay, setPay] = useState({ amount: '', method: 'TRANSFER', paidAt: '', reference: '', note: '' })
  const [deduct, setDeduct] = useState({ amount: '', reason: '' })
  const [invoice, setInvoice] = useState({ invoiceNo: '', amount: '', invoicedAt: '', note: '' })
  const [payErr, setPayErr] = useState('')
  const [deductErr, setDeductErr] = useState('')
  const [invoiceErr, setInvoiceErr] = useState('')
  const [busyKey, setBusyKey] = useState('')

  const post = async (key: string, url: string, body: Record<string, unknown>, okMsg: string) => {
    setBusyKey(key)
    try {
      await apiFetch(url, { method: 'POST', body })
      toast.success(okMsg)
      onDone()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : p.actionFail)
    } finally {
      setBusyKey('')
    }
  }

  const registerPay = () => {
    const amt = Number(pay.amount)
    if (!Number.isFinite(amt) || amt <= 0) { setPayErr(p.amountInvalid); return }
    setPayErr('')
    void post('pay', `/odn/payables/${payableId}/payments`, {
      amount: amt, method: pay.method, paidAt: pay.paidAt.trim(),
      reference: pay.reference.trim(), note: pay.note.trim(),
    }, p.payOk).then(() => setPay({ amount: '', method: 'TRANSFER', paidAt: '', reference: '', note: '' }))
  }

  const registerDeduct = () => {
    const amt = Number(deduct.amount)
    if (!Number.isFinite(amt) || amt <= 0) { setDeductErr(p.amountInvalid); return }
    if (!deduct.reason.trim()) { setDeductErr(p.deductReasonRequired); return }
    setDeductErr('')
    void post('deduct', `/odn/payables/${payableId}/deductions`, {
      amount: amt, reason: deduct.reason.trim(),
    }, p.deductOk).then(() => setDeduct({ amount: '', reason: '' }))
  }

  const registerInvoice = () => {
    const amt = Number(invoice.amount)
    if (!invoice.invoiceNo.trim()) { setInvoiceErr(p.invoiceNoRequired); return }
    if (!Number.isFinite(amt) || amt <= 0) { setInvoiceErr(p.amountInvalid); return }
    setInvoiceErr('')
    void post('invoice', `/odn/payables/${payableId}/invoices`, {
      invoiceNo: invoice.invoiceNo.trim(), amount: amt,
      invoicedAt: invoice.invoicedAt.trim(), note: invoice.note.trim(),
    }, p.invoiceOk).then(() => setInvoice({ invoiceNo: '', amount: '', invoicedAt: '', note: '' }))
  }

  const setPayField = (k: 'amount' | 'method' | 'paidAt' | 'reference' | 'note', v: string) => setPay((m) => ({ ...m, [k]: v }))
  const setDeductField = (k: 'amount' | 'reason', v: string) => setDeduct((m) => ({ ...m, [k]: v }))
  const setInvoiceField = (k: 'invoiceNo' | 'amount' | 'invoicedAt' | 'note', v: string) => setInvoice((m) => ({ ...m, [k]: v }))

  const setTitle = 'mb-2 text-sm font-semibold text-[var(--shell-heading)]'
  return (
    <div className='grid grid-cols-1 gap-3 lg:grid-cols-3'>
      <Card className='mb-0 p-4'>
        <div className={setTitle}>{p.payTitle}</div>
        <div className='space-y-2'>
          <FormField label={p.payAmount} required hint={p.payAmountHint} error={payErr || undefined}>
            <Input inputMode='decimal' value={pay.amount} placeholder='0.00'
              onChange={(e) => setPayField('amount', e.target.value.replace(/[^0-9.]/g, ''))} />
          </FormField>
          <FormField label={p.payMethod} required>
            <Dropdown value={pay.method} ariaLabel={p.payMethod}
              options={METHOD_KEYS.map((m) => ({ value: m, label: p.methodTexts[m] }))}
              onChange={(v) => setPay((m) => ({ ...m, method: v }))} />
          </FormField>
          <FormField label={p.payAt} hint={p.payAtHint}>
            <Input value={pay.paidAt} placeholder='2026-09-07 12:00:00'
              onChange={(e) => setPay((m) => ({ ...m, paidAt: e.target.value }))} />
          </FormField>
          <FormField label={p.payReference}>
            <Input value={pay.reference} placeholder={p.optional}
              onChange={(e) => setPay((m) => ({ ...m, reference: e.target.value }))} />
          </FormField>
          <FormField label={p.payNote}>
            <Input value={pay.note} placeholder={p.optional}
              onChange={(e) => setPay((m) => ({ ...m, note: e.target.value }))} />
          </FormField>
          <div className='flex justify-end'>
            <SubmitButton state={busyKey === 'pay' ? 'loading' : 'idle'} onClick={registerPay}
              labels={{ idle: p.paySubmit, loading: p.paySubmit, success: p.payOk, failed: p.paySubmit }} />
          </div>
        </div>
      </Card>
      <Card className='mb-0 p-4'>
        <div className={setTitle}>{p.deductTitle}</div>
        <div className='space-y-2'>
          <FormField label={p.payAmount} required hint={p.payAmountHint} error={deductErr || undefined}>
            <Input inputMode='decimal' value={deduct.amount} placeholder='0.00'
              onChange={(e) => setDeductField('amount', e.target.value.replace(/[^0-9.]/g, ''))} />
          </FormField>
          <FormField label={p.deductReason} required>
            <Input value={deduct.reason} placeholder={p.deductReasonPh}
              onChange={(e) => setDeduct((m) => ({ ...m, reason: e.target.value }))} />
          </FormField>
          <div className='flex justify-end'>
            <SubmitButton state={busyKey === 'deduct' ? 'loading' : 'idle'} onClick={registerDeduct}
              labels={{ idle: p.deductSubmit, loading: p.deductSubmit, success: p.deductOk, failed: p.deductSubmit }} />
          </div>
        </div>
      </Card>
      <Card className='mb-0 p-4'>
        <div className={setTitle}>{p.invoiceTitle}</div>
        <div className='space-y-2'>
          <FormField label={p.invoiceNo} required error={invoiceErr || undefined}>
            <Input value={invoice.invoiceNo}
              onChange={(e) => setInvoice((m) => ({ ...m, invoiceNo: e.target.value }))} />
          </FormField>
          <FormField label={p.payAmount} required>
            <Input inputMode='decimal' value={invoice.amount} placeholder='0.00'
              onChange={(e) => setInvoiceField('amount', e.target.value.replace(/[^0-9.]/g, ''))} />
          </FormField>
          <FormField label={p.invoicedAt} hint={p.invoicedAtHint}>
            <Input value={invoice.invoicedAt} placeholder={p.optional}
              onChange={(e) => setInvoice((m) => ({ ...m, invoicedAt: e.target.value }))} />
          </FormField>
          <FormField label={p.payNote}>
            <Input value={invoice.note} placeholder={p.optional}
              onChange={(e) => setInvoice((m) => ({ ...m, note: e.target.value }))} />
          </FormField>
          <div className='flex justify-end'>
            <SubmitButton state={busyKey === 'invoice' ? 'loading' : 'idle'} onClick={registerInvoice}
              labels={{ idle: p.invoiceSubmit, loading: p.invoiceSubmit, success: p.invoiceOk, failed: p.invoiceSubmit }} />
          </div>
        </div>
      </Card>
    </div>
  )
}
