// 采购单页:契约 GET/POST /procurement/orders + /:id/submit + /:id/cancel +
// /procurement/suppliers + /procurement/receipts + /:id/confirm。
// 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import {
  pageSlice,
  type SupplierRow,
  type OrderRow,
  type OrderItemRow,
  type ReceiptRow,
} from '../types'

export default function PurchasePage() {
  const t = useT()
  const d = t.pages.purchasePage
  const [orders, setOrders] = useState<OrderRow[]>([])
  const [suppliers, setSuppliers] = useState<SupplierRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [statusFilter, setStatusFilter] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [confirming, setConfirming] = useState<OrderRow | null>(null)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: OrderRow[] }>('/procurement/orders', { query: { status: statusFilter || undefined } })
      .then((d) => setOrders(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [statusFilter]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    apiFetch<{ items: SupplierRow[] }>('/procurement/suppliers')
      .then((d) => setSuppliers((d?.items ?? []).filter((x) => x.status === 'ENABLED')))
      .catch(() => setSuppliers([]))
  }, [])

  const submitOrder = async (id: number) => {
    setBusy(true)
    try {
      await apiFetch(`/procurement/orders/${id}/submit`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : d.opFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(orders, page, pageSize)
  const statusOpts = [
    { value: '', label: '全部' },
    { value: 'DRAFT', label: '草稿' },
    { value: 'SUBMITTED', label: '已提交' },
    { value: 'PARTIAL', label: '部分到货' },
    { value: 'RECEIVED', label: '全部到货' },
    { value: 'CANCELLED', label: '已取消' },
  ]

  return (
    <div className="space-y-4">
      <PageHead title={d.title} desc={d.subtitle} />

      <div className="flex flex-wrap items-center gap-3">
        <Dropdown
          value={statusFilter}
          options={statusOpts}
          onChange={setStatusFilter}
          ariaLabel={d.statusFilter}
        />
        <button
          type="button"
          className="ml-auto h-9 px-4 rounded-md bg-brand-primary text-white text-sm"
          onClick={() => setCreateOpen(true)}
        >
          {d.newOrder}
        </button>
      </div>

      {error && <div className="text-sm text-status-danger">{error}</div>}

      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-shell-fg-muted">
            <th className="py-2 pr-4">{d.colNo}</th>
            <th className="py-2 pr-4">{d.colSupplier}</th>
            <th className="py-2 pr-4">{d.colEntity}</th>
            <th className="py-2 pr-4">{d.colStatus}</th>
            <th className="py-2 pr-4">{d.colTotal}</th>
            <th className="py-2 pr-4">{d.colActions}</th>
          </tr>
        </thead>
        <tbody>
          {slice.map((o) => (
            <tr key={o.id} className="border-t border-shell-divider">
              <td className="py-2 pr-4 font-mono">{o.procurementNo}</td>
              <td className="py-2 pr-4">{o.supplierName}</td>
              <td className="py-2 pr-4">{o.legalEntityName}</td>
              <td className="py-2 pr-4"><StatusTag domain="procurement" value={o.status} /></td>
              <td className="py-2 pr-4">{o.totalAmount.toFixed(2)}</td>
              <td className="py-2 pr-4 space-x-2">
                {o.status === 'DRAFT' && (
                  <button
                    type="button"
                    className="h-7 px-2 rounded border border-shell-divider text-xs"
                    onClick={() => submitOrder(o.id)}
                    disabled={busy}
                  >
                    {d.submit}
                  </button>
                )}
                {(o.status === 'SUBMITTED' || o.status === 'PARTIAL') && (
                  <button
                    type="button"
                    className="h-7 px-2 rounded border border-shell-divider text-xs"
                    onClick={() => setConfirming(o)}
                    disabled={busy}
                  >
                    {d.confirmReceipt}
                  </button>
                )}
              </td>
            </tr>
          ))}
          {orders.length === 0 && !busy && (
            <tr><td colSpan={6} className="py-6 text-center text-shell-fg-muted">{d.empty}</td></tr>
          )}
        </tbody>
      </table>

      <Pagination
        page={page}
        pageSize={pageSize}
        total={orders.length}
        onPage={setPage}
        onSize={setPageSize}
        {...pagerTexts(t.pages.company)}
      />

      {createOpen && (
        <CreateOrderDrawer
          suppliers={suppliers}
          onClose={() => setCreateOpen(false)}
          onSaved={() => { setCreateOpen(false); load() }}
        />
      )}

      {confirming && (
        <ConfirmReceiptDrawer
          order={confirming}
          onClose={() => setConfirming(null)}
          onSaved={() => { setConfirming(null); load() }}
        />
      )}
    </div>
  )
}

function CreateOrderDrawer({
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
          <button className="h-8 px-4 rounded border border-shell-divider text-sm" onClick={onClose}>{d.cancel}</button>
          <button className="h-8 px-4 rounded bg-brand-primary text-white text-sm" disabled={busy} onClick={submit}>
            {busy ? d.submitting : d.save}
          </button>
        </>
      }>
      <div className="space-y-3">
        {err && <div className="text-sm text-status-danger">{err}</div>}
        <div className="flex flex-col gap-1">
          <label className="text-sm text-shell-fg-muted">{d.colSupplier}</label>
          <Dropdown
            value={String(supplierId)}
            options={suppliers.map((s) => ({ value: String(s.id), label: s.name }))}
            onChange={(v) => setSupplierId(Number(v))}
            ariaLabel={d.colSupplier}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-sm text-shell-fg-muted">{d.colEntity} ID</label>
          <input
            type="number"
            value={legalEntityId}
            onChange={(e) => setLegalEntityId(Number(e.target.value))}
            className="h-9 px-3 rounded border border-shell-divider bg-shell-bg-card text-sm"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-sm text-shell-fg-muted">{d.remark}</label>
          <input
            type="text"
            value={remark}
            onChange={(e) => setRemark(e.target.value)}
            className="h-9 px-3 rounded border border-shell-divider bg-shell-bg-card text-sm"
          />
        </div>
        <div>
          <div className="text-sm text-shell-fg-muted mb-2">{d.items}</div>
          {items.map((it, idx) => (
            <div key={idx} className="grid grid-cols-12 gap-2 mb-2">
              <input
                placeholder="MI-ONU"
                value={it.materialCode}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, materialCode: e.target.value } : x))}
                className="col-span-4 h-9 px-2 rounded border border-shell-divider bg-shell-bg-card text-sm"
              />
              <input
                placeholder={d.spec}
                value={it.spec}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, spec: e.target.value } : x))}
                className="col-span-3 h-9 px-2 rounded border border-shell-divider bg-shell-bg-card text-sm"
              />
              <input
                type="number"
                placeholder={d.qty}
                value={it.quantity}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, quantity: Number(e.target.value) } : x))}
                className="col-span-2 h-9 px-2 rounded border border-shell-divider bg-shell-bg-card text-sm"
              />
              <input
                type="number"
                placeholder={d.unitAmount}
                value={it.unitAmount}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, unitAmount: Number(e.target.value) } : x))}
                className="col-span-3 h-9 px-2 rounded border border-shell-divider bg-shell-bg-card text-sm"
              />
            </div>
          ))}
          <button
            type="button"
            className="h-7 px-2 rounded border border-shell-divider text-xs"
            onClick={() => setItems([...items, { materialCode: '', spec: '', quantity: 1, unitAmount: 0 }])}
          >
            {d.addItem}
          </button>
        </div>
      </div>
    </Drawer>
  )
}

function ConfirmReceiptDrawer({
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
  const [items, setItems] = useState([{ materialCode: 'ONU', spec: '1GE', quantity: 1, unitAmount: 0 }])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const [receipts, setReceipts] = useState<ReceiptRow[]>([])

  useEffect(() => {
    apiFetch<{ items: ReceiptRow[] }>('/procurement/receipts', { query: { orderId: order.id } })
      .then((r) => setReceipts(r?.items ?? []))
      .catch(() => setReceipts([]))
  }, [order.id])

  const submit = async () => {
    setErr('')
    setBusy(true)
    try {
      const created = await apiFetch<{ id: number }>(`/procurement/receipts`, {
        method: 'POST',
        body: { orderId: order.id, orderNo: order.procurementNo, legalEntityId: order.legalEntityId },
      })
      if (!created) throw new Error('create receipt failed')
      await apiFetch(`/procurement/receipts/${created.id}/confirm`, {
        method: 'POST',
        body: { batchCode, batchName, items },
      })
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
          <button className="h-8 px-4 rounded border border-shell-divider text-sm" onClick={onClose}>{d.cancel}</button>
          <button className="h-8 px-4 rounded bg-brand-primary text-white text-sm" disabled={busy} onClick={submit}>
            {busy ? d.submitting : d.confirmReceipt}
          </button>
        </>
      }>
      <div className="space-y-3">
        {err && <div className="text-sm text-status-danger">{err}</div>}
        {receipts.length > 0 && (
          <div className="text-xs text-shell-fg-muted">
            {d.historyReceipt}: {receipts.map((r) => `${r.receiptNo}(${r.status})`).join(', ')}
          </div>
        )}
        <div className="flex flex-col gap-1">
          <label className="text-sm text-shell-fg-muted">{d.batchCode}</label>
          <input value={batchCode} onChange={(e) => setBatchCode(e.target.value)}
            className="h-9 px-3 rounded border border-shell-divider bg-shell-bg-card text-sm"
            placeholder="RK-20260828-001" />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-sm text-shell-fg-muted">{d.batchName}</label>
          <input value={batchName} onChange={(e) => setBatchName(e.target.value)}
            className="h-9 px-3 rounded border border-shell-divider bg-shell-bg-card text-sm" />
        </div>
        <div>
          <div className="text-sm text-shell-fg-muted mb-2">{d.items}</div>
          {items.map((it, idx) => (
            <div key={idx} className="grid grid-cols-12 gap-2 mb-2">
              <input placeholder="MI-ONU" value={it.materialCode}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, materialCode: e.target.value } : x))}
                className="col-span-4 h-9 px-2 rounded border border-shell-divider bg-shell-bg-card text-sm" />
              <input placeholder={d.spec} value={it.spec}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, spec: e.target.value } : x))}
                className="col-span-3 h-9 px-2 rounded border border-shell-divider bg-shell-bg-card text-sm" />
              <input type="number" placeholder={d.qty} value={it.quantity}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, quantity: Number(e.target.value) } : x))}
                className="col-span-2 h-9 px-2 rounded border border-shell-divider bg-shell-bg-card text-sm" />
              <input type="number" placeholder={d.unitAmount} value={it.unitAmount}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, unitAmount: Number(e.target.value) } : x))}
                className="col-span-3 h-9 px-2 rounded border border-shell-divider bg-shell-bg-card text-sm" />
            </div>
          ))}
        </div>
      </div>
    </Drawer>
  )
}
