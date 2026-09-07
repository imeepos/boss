// 采购单创建抽屉:供应商 + 法人选择器 + 明细行,POST /procurement/orders。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { ErrorBanner } from '../../../components/business/page-head'
import type { OrderItemRow, SupplierRow } from '../types'
import { OrderItemsEditor } from './OrderItemsEditor'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export function CreateOrderDrawer({
  suppliers, onClose, onSaved,
}: {
  suppliers: SupplierRow[]
  onClose: () => void
  onSaved: () => void
}) {
  const t = useT()
  const d = t.pages.purchasePage
  const [supplierId, setSupplierId] = useState(0)
  const [legalEntityId, setLegalEntityId] = useState(0)
  const [entities, setEntities] = useState<{ id: number; name: string }[]>([])
  const [remark, setRemark] = useState('')
  const [items, setItems] = useState<OrderItemRow[]>([{ materialCode: '', spec: '', quantity: 1, unitAmount: 0 }])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  useEffect(() => {
    apiFetch<{ id: number; name: string }[]>('/legal-entities').then((d) => setEntities(d ?? [])).catch(() => setEntities([]))
  }, [])

  const submit = async () => {
    setErr('')
    if (!supplierId || supplierId <= 0) { setErr(d.errSupplier); return }
    if (!legalEntityId) { setErr(d.pEntitySelect); return }
    if (items.some((i) => !i.materialCode || i.quantity <= 0)) { setErr(d.errItems); return }
    setBusy(true)
    try {
      await apiFetch('/procurement/orders', {
        method: 'POST',
        body: { supplierId, legalEntityId, remark, items },
      })
      toast.success(d.createOk)
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
        {err && <ErrorBanner message={err} />}
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.colSupplier}</label>
          <Dropdown
            value={supplierId ? String(supplierId) : ''}
            options={[{ value: '', label: d.errSupplier }, ...suppliers.map((s) => ({ value: String(s.id), label: s.name }))]}
            onChange={(v) => setSupplierId(Number(v) || 0)}
            ariaLabel={d.colSupplier}
            placeholder={d.errSupplier}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.colEntity}</label>
          <SimplePicker value={legalEntityId ? String(legalEntityId) : ''}
            onChange={(v) => setLegalEntityId(Number(v) || 0)}
            options={entities.map((e) => ({ value: String(e.id), label: e.name }))}
            placeholder={d.pEntitySelect} ariaLabel={d.colEntity} minWidth={260} />
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
        <OrderItemsEditor items={items} onChange={setItems} onDelete={(idx) => setItems(items.filter((_, i) => i !== idx))} />
      </div>
    </Drawer>
  )
}
