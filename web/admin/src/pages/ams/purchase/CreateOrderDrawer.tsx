// 采购单创建抽屉:供应商 + 实体 + 明细行,POST /procurement/orders(自 index.tsx 拆出,零行为变更)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import type { OrderItemRow, SupplierRow } from '../types'
import { OrderItemsEditor } from './OrderItemsEditor'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
const errBanner = 'rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

export function CreateOrderDrawer({
  suppliers, onClose, onSaved,
}: {
  suppliers: SupplierRow[]
  onClose: () => void
  onSaved: () => void
}) {
  const t = useT()
  const d = t.pages.purchasePage
  const [supplierId, setSupplierId] = useState(suppliers[0]?.id ?? 0)
  const [legalEntityId, setLegalEntityId] = useState(1)
  const [remark, setRemark] = useState('')
  const [items, setItems] = useState<OrderItemRow[]>([{ materialCode: '', spec: '', quantity: 1, unitAmount: 0 }])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const submit = async () => {
    setErr('')
    if (!supplierId || supplierId <= 0) { setErr(d.errSupplier); return }
    if (items.some((i) => !i.materialCode || i.quantity <= 0)) { setErr(d.errItems); return }
    setBusy(true)
    try {
      await apiFetch('/procurement/orders', {
        method: 'POST',
        body: { supplierId, legalEntityId, remark, items },
      })
      onSaved()
    } catch (e) {
      setErr(e instanceof Error ? e.message : d.opFail)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Drawer title={d.newOrder} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{d.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>
            {busy ? d.submitting : d.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        {err && <div className={errBanner}>{err}</div>}
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.colSupplier}</label>
          <Dropdown
            value={String(supplierId)}
            options={[{ value: '', label: d.colSupplier }, ...suppliers.map((s) => ({ value: String(s.id), label: s.name }))]}
            onChange={(v) => setSupplierId(Number(v) || 0)}
            ariaLabel={d.colSupplier}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <label>{d.colEntity} ID</label>
          <input
            type="number"
            value={legalEntityId}
            onChange={(e) => setLegalEntityId(Number(e.target.value))}
            className={input}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <label>{d.remark}</label>
          <input
            type="text"
            value={remark}
            onChange={(e) => setRemark(e.target.value)}
            className={input}
          />
        </div>
        <OrderItemsEditor items={items} onChange={setItems} />
      </div>
    </Drawer>
  )
}
