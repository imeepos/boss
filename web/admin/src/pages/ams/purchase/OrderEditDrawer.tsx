// 采购单编辑抽屉(DRAFT 专用):GET /procurement/orders/{id} 回填单头+明细,
// PUT /procurement/orders/{id} 整体替换明细(载荷剥 receivedQty,见 purchaseLogic)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import type { OrderItemRow, SupplierRow } from '../types'
import { OrderItemsEditor } from './OrderItemsEditor'
import { buildOrderEditPayload, orderEditErr, unwrapOrderDetail, type OrderEditFormState } from './purchaseLogic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
const errBanner = 'rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

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

  return (
    <Drawer title={d.orderEditTitle.replace('{no}', order.procurementNo || '#' + order.id)} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{d.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !form} onClick={submit}>
            {busy ? d.submitting : d.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        {loadErr && <div className={errBanner}>{loadErr}</div>}
        {!form && !loadErr && <div className="py-6 text-center text-[13px] text-[var(--shell-group-title)]">{t.common.loading}</div>}
        {form && (
          <>
            {err && <div className={errBanner}>{errMsgKey(err)}</div>}
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.colSupplier}</label>
              <Dropdown
                value={String(form.supplierId)}
                options={[{ value: '', label: d.colSupplier }, ...suppliers.map((s) => ({ value: String(s.id), label: s.name }))]}
                onChange={(v) => setForm({ ...form, supplierId: Number(v) || 0 })}
                ariaLabel={d.colSupplier}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label>{d.colEntity} ID</label>
              <input type="number" className={input} value={form.legalEntityId}
                onChange={(e) => setForm({ ...form, legalEntityId: Number(e.target.value) })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <label>{d.remark}</label>
              <input type="text" className={input} value={form.remark}
                onChange={(e) => setForm({ ...form, remark: e.target.value })} />
            </div>
            <OrderItemsEditor items={form.items} onChange={(items) => setForm({ ...form, items })} />
          </>
        )}
      </div>
    </Drawer>
  )
}
