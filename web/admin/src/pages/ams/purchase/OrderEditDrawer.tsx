// 采购单编辑抽屉(DRAFT 专用):GET /procurement/orders/{id} 回填单头+明细,
// PUT /procurement/orders/{id} 整体替换明细(载荷剥 receivedQty,见 purchaseLogic)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton } from '../../../components/business/submit-button'
import type { OrderItemRow, SupplierRow } from '../types'
import { OrderItemsEditor } from './OrderItemsEditor'
import { buildOrderEditPayload, orderEditErr, unwrapOrderDetail, type OrderEditFormState } from './purchaseLogic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export function OrderEditDrawer({ order, suppliers, onClose, onSaved }: {
  order: { id: number; procurementNo: string }
  suppliers: SupplierRow[]
  onClose: () => void
  onSaved: () => void
}) {
  const t = useT()
  const d = t.pages.purchasePage
  const [form, setForm] = useState<OrderEditFormState | null>(null)
  const [err, setErr] = useState('')
  const [loadErr, setLoadErr] = useState('')
  const [busy, setBusy] = useState(false)
  const [entities, setEntities] = useState<{ id: number; name: string }[]>([])

  useEffect(() => {
    apiFetch<{ id: number; name: string }[]>('/legal-entities').then((d) => setEntities(d ?? [])).catch(() => setEntities([]))
  }, [])

  useEffect(() => {
    apiFetch<{ item: { items: OrderItemRow[]; supplierId: number; legalEntityId: number; remark: string } }>(
      '/procurement/orders/' + order.id,
    )
      .then((res) => {
        const detail = unwrapOrderDetail(res)
        if (!detail) return
        setForm({
          supplierId: detail.supplierId,
          legalEntityId: detail.legalEntityId,
          remark: detail.remark ?? '',
          items: (detail.items ?? []).map((i) => ({
            materialCode: i.materialCode, spec: i.spec ?? '',
            quantity: i.quantity, unitAmount: i.unitAmount,
            receivedQty: i.receivedQty,
          })),
        })
      })
      .catch((e) => setLoadErr(e instanceof Error ? e.message : d.loadFail))
  }, [order.id]) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (busy || !form) return
    const key = orderEditErr(form)
    setErr(key)
    if (key) return
    setBusy(true)
    try {
      await apiFetch('/procurement/orders/' + order.id, { method: 'PUT', body: buildOrderEditPayload(form) })
      toast.success(d.orderSaveOk)
      onSaved()
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : d.opFail)
      setBusy(false)
    }
  }

  const errMsgKey = (k: string) =>
    k === 'errSupplier' ? d.errSupplier : d.errItems

  const submitState = busy ? 'loading' : (err ? 'failed' : 'idle')
  const submitLabels = { idle: d.save, loading: d.submitting, success: d.save, failed: d.opFail }

  return (
    <Drawer title={d.orderEditTitle.replace('{no}', order.procurementNo || '#' + order.id)} onClose={onClose}
      footer={
        <>
          <ToolbarButton onClick={onClose} disabled={busy || !form}>{d.cancel}</ToolbarButton>
          <SubmitButton state={submitState} labels={submitLabels} disabled={busy || !form} onClick={submit} />
        </>
      }>
      <div className="flex flex-col gap-3.5">
        {loadErr && <ErrorBanner message={loadErr} />}
        {!form && !loadErr && <div className="py-6 text-center text-[13px] text-[var(--shell-group-title)]">{t.common.loading}</div>}
        {form && (
          <>
            {err && <ErrorBanner message={errMsgKey(err)} />}
            <FormField label={d.colSupplier} required>
              <Dropdown
                value={String(form.supplierId)}
                options={[{ value: '', label: d.colSupplier }, ...suppliers.map((s) => ({ value: String(s.id), label: s.name }))]}
                onChange={(v) => setForm({ ...form, supplierId: Number(v) || 0 })}
                ariaLabel={d.colSupplier}
              />
            </FormField>
            <FormField label={d.colEntity}>
              <SimplePicker value={form.legalEntityId ? String(form.legalEntityId) : ''}
                onChange={(v) => setForm({ ...form, legalEntityId: Number(v) || 0 })}
                options={entities.map((e) => ({ value: String(e.id), label: e.name }))}
                ariaLabel={d.colEntity} placeholder={d.pEntitySelect}
                pinnedOptions={form.legalEntityId ? [{ value: String(form.legalEntityId), label: '#' + form.legalEntityId }] : undefined}
                minWidth={260} />
            </FormField>
            <FormField label={d.remark}>
              <input type="text" className={input} value={form.remark}
                onChange={(e) => setForm({ ...form, remark: e.target.value })} />
            </FormField>
            <OrderItemsEditor items={form.items} onChange={(items) => setForm({ ...form, items })}
              onDelete={(idx) => setForm({ ...form, items: form.items.filter((_, i) => i !== idx) })} />
          </>
        )}
      </div>
    </Drawer>
  )
}
