// 入库确认抽屉:建入库单 + 确认(同事务建批次+资产);明细回带订单明细(可改),不再硬编码预填。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { ErrorBanner } from '../../../components/business/page-head'
import { unwrapOrderDetail } from './purchaseLogic'
import type { OrderRow, ReceiptRow } from '../types'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export function ConfirmReceiptDrawer({
  order, onClose, onSaved,
}: {
  order: OrderRow
  onClose: () => void
  onSaved: () => void
}) {
  const t = useT()
  const d = t.pages.purchasePage
  const [batchCode, setBatchCode] = useState('')
  const [batchName, setBatchName] = useState('')
  const [items, setItems] = useState<{ materialCode: string; spec: string; quantity: number; unitAmount: number }[]>([])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const [receipts, setReceipts] = useState<ReceiptRow[]>([])

  useEffect(() => {
    // 明细回带:以订单明细为初始值(含已收数量不改),替代旧的硬编码 ONU/1GE 预填。
    apiFetch<{ item: { items?: { materialCode: string; spec?: string; quantity: number; unitAmount: number }[] } }>('/procurement/orders/' + order.id)
      .then((x) => {
        const rows = (unwrapOrderDetail(x)?.items ?? []).map((i) => ({
          materialCode: i.materialCode, spec: i.spec ?? '', quantity: i.quantity, unitAmount: i.unitAmount,
        }))
        setItems(rows.length ? rows : [{ materialCode: '', spec: '', quantity: 1, unitAmount: 0 }])
      })
      .catch(() => setItems([{ materialCode: '', spec: '', quantity: 1, unitAmount: 0 }]))
    apiFetch<{ items: ReceiptRow[] }>('/procurement/receipts', { query: { orderId: order.id } })
      .then((r) => setReceipts(r?.items ?? []))
      .catch(() => setReceipts([]))
  }, [order.id])

  const submit = async () => {
    setErr('')
    setBusy(true)
    try {
      const created = await apiFetch<{ id: number }>('/procurement/receipts', {
        method: 'POST',
        body: { orderId: order.id, orderNo: order.procurementNo, legalEntityId: order.legalEntityId },
      })
      if (!created) throw new Error('create receipt failed')
      await apiFetch(`/procurement/receipts/${created.id}/confirm`, {
        method: 'POST',
        body: { batchCode, batchName, items },
      })
      toast.success(d.confirmOk)
      onSaved()
    } catch (e) {
      setErr(e instanceof Error ? e.message : d.opFail)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Drawer title={`${d.confirmReceipt} · ${order.procurementNo}`} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{d.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>
            {busy ? d.submitting : d.confirmReceipt}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        {err && <ErrorBanner message={err} />}
        {receipts.length > 0 && (
          <div className="text-[12px] text-[var(--shell-group-title)]">
            {d.historyReceipt}: {receipts.map((r) => `${r.receiptNo}(` + (t.common.statusTags[`receipt.${r.status}`] || r.status) + `)`).join(', ')}
          </div>
        )}
        <div className="flex flex-col gap-1.5">
          <label>{d.batchCode}</label>
          <input value={batchCode} onChange={(e) => setBatchCode(e.target.value)}
            placeholder={d.phBatchCode}
            className={input} />
        </div>
        <div className="flex flex-col gap-1.5">
          <label>{d.batchName}</label>
          <input value={batchName} onChange={(e) => setBatchName(e.target.value)}
            className={input} />
        </div>
        <div>
          <div className="mb-2 text-[13px] text-[var(--shell-content-text)]">{d.items}</div>
          {items.map((it, idx) => (
            <div key={idx} className="mb-2 grid grid-cols-12 gap-2">
              <input placeholder={d.phMaterial} value={it.materialCode}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, materialCode: e.target.value } : x))}
                className={input + ' col-span-4'} />
              <input placeholder={d.spec} value={it.spec}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, spec: e.target.value } : x))}
                className={input + ' col-span-3'} />
              <input type="number" placeholder={d.qty} value={it.quantity}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, quantity: Number(e.target.value) } : x))}
                className={input + ' col-span-2'} />
              <input type="number" placeholder={d.unitAmount} value={it.unitAmount}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, unitAmount: Number(e.target.value) } : x))}
                className={input + ' col-span-3'} />
            </div>
          ))}
        </div>
      </div>
    </Drawer>
  )
}