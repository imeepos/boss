// 采购单页:契约 GET/POST /procurement/orders + /:id/submit + /:id/cancel +
// /procurement/suppliers + /procurement/receipts + /:id/confirm。
// 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md。
// 样式对齐 stock:大卡片 + TableStateRow + Drawer footer + CSS 变量主题适配。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { TableStateRow } from '../../../components/business'
import {
  pageSlice,
  type SupplierRow,
  type OrderRow,
  type OrderItemRow,
  type ReceiptRow,
} from '../types'

const STATUS_FILTERS = ['DRAFT', 'SUBMITTED', 'PARTIAL', 'RECEIVED', 'CANCELLED'] as const

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
  useEffect(load, [statusFilter])

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
      setBusy(false)
    }
  }

  const filtered = statusFilter ? orders.filter((o) => o.status === statusFilter) : orders
  const slice = pageSlice(filtered, page, pageSize)
  const statusOpts = [
    { value: '', label: d.allStatus },
    ...STATUS_FILTERS.map((s) => ({ value: s, label: t.common.statusTags[`procurement.${s}`] || s })),
  ]

  return (
    <div>
      <PageHead title={d.title} desc={d.subtitle} />

      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Dropdown
            value={statusFilter}
            options={statusOpts}
            onChange={setStatusFilter}
            ariaLabel={d.statusFilter}
          />
          <span className="spacer" />
          <button
            type="button"
            onClick={load}
            className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
          >
            {t.pages.audit.refresh}
          </button>
          <button
            type="button"
            onClick={() => setCreateOpen(true)}
            className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
          >
            {d.newOrder}
          </button>
        </div>
        {error ? (
          <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
        ) : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead>
                <tr>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colNo}</th>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colSupplier}</th>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colEntity}</th>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colStatus}</th>
                  <th className="h-11 px-3 text-right text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colTotal}</th>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colActions}</th>
                </tr>
              </thead>
              <tbody>
                {slice.map((o) => (
                  <tr key={o.id} className="hover:bg-[var(--shell-menu-hover-bg)]">
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] font-mono">{o.procurementNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{o.supplierName}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{o.legalEntityName}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]"><StatusTag domain="procurement" value={o.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-right">{o.totalAmount.toFixed(2)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">
                      <span className="inline-flex items-center gap-2">
                        {o.status === 'DRAFT' && (
                          <button type="button" disabled={busy} onClick={() => submitOrder(o.id)} className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]">{d.submit}</button>
                        )}
                        {(o.status === 'SUBMITTED' || o.status === 'PARTIAL') && (
                          <button type="button" disabled={busy} onClick={() => setConfirming(o)} className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]">{d.confirmReceipt}</button>
                        )}
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={d.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination
            total={filtered.length}
            page={page}
            pageSize={pageSize}
            onPage={setPage}
            onSize={setPageSize}
            {...pagerTexts(t.pages.company)}
          />
        </div>
      </div>

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
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{d.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>
            {busy ? d.submitting : d.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        {err && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{err}</div>}
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
            className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <label>{d.remark}</label>
          <input
            type="text"
            value={remark}
            onChange={(e) => setRemark(e.target.value)}
            className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
          />
        </div>
        <div>
          <div className="mb-2 text-[13px] text-[var(--shell-content-text)]">{d.items}</div>
          {items.map((it, idx) => (
            <div key={idx} className="mb-2 grid grid-cols-12 gap-2">
              <input
                placeholder="MI-ONU"
                value={it.materialCode}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, materialCode: e.target.value } : x))}
                className="col-span-4 h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
              />
              <input
                placeholder={d.spec}
                value={it.spec}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, spec: e.target.value } : x))}
                className="col-span-3 h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
              />
              <input
                type="number"
                placeholder={d.qty}
                value={it.quantity}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, quantity: Number(e.target.value) } : x))}
                className="col-span-2 h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border)] focus:border-[var(--color-border-focus)]"
              />
              <input
                type="number"
                placeholder={d.unitAmount}
                value={it.unitAmount}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, unitAmount: Number(e.target.value) } : x))}
                className="col-span-3 h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
              />
            </div>
          ))}
          <button
            type="button"
            className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
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
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{d.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>
            {busy ? d.submitting : d.confirmReceipt}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        {err && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{err}</div>}
        {receipts.length > 0 && (
          <div className="text-[12px] text-[var(--shell-group-title)]">
            {d.historyReceipt}: {receipts.map((r) => `${r.receiptNo}(${t.common.statusTags[`receipt.${r.status}`] || r.status})`).join(', ')}
          </div>
        )}
        <div className="flex flex-col gap-1.5">
          <label>{d.batchCode}</label>
          <input value={batchCode} onChange={(e) => setBatchCode(e.target.value)}
            placeholder="RK-20260828-001"
            className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" />
        </div>
        <div className="flex flex-col gap-1.5">
          <label>{d.batchName}</label>
          <input value={batchName} onChange={(e) => setBatchName(e.target.value)}
            className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" />
        </div>
        <div>
          <div className="mb-2 text-[13px] text-[var(--shell-content-text)]">{d.items}</div>
          {items.map((it, idx) => (
            <div key={idx} className="mb-2 grid grid-cols-12 gap-2">
              <input placeholder="MI-ONU" value={it.materialCode}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, materialCode: e.target.value } : x))}
                className="col-span-4 h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" />
              <input placeholder={d.spec} value={it.spec}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, spec: e.target.value } : x))}
                className="col-span-3 h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" />
              <input type="number" placeholder={d.qty} value={it.quantity}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, quantity: Number(e.target.value) } : x))}
                className="col-span-2 h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" />
              <input type="number" placeholder={d.unitAmount} value={it.unitAmount}
                onChange={(e) => setItems(items.map((x, i) => i === idx ? { ...x, unitAmount: Number(e.target.value) } : x))}
                className="col-span-3 h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" />
            </div>
          ))}
        </div>
      </div>
    </Drawer>
  )
}
