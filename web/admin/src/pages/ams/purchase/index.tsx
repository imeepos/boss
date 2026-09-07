// 采购单页:契约 GET/POST /procurement/orders + /:id/submit + /:id/cancel +
// /procurement/suppliers + /procurement/receipts + /:id/confirm。
// 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md。
// 样式对齐 stock:大卡片 + TableStateRow + Drawer footer + CSS 变量主题适配。
// 创建/入库确认抽屉见 ./CreateOrderDrawer 与 ./ConfirmReceiptDrawer。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow } from '../../../components/business'
import { pageSlice, type OrderRow, type SupplierRow } from '../types'
import { CreateOrderDrawer } from './CreateOrderDrawer'
import { OrderEditDrawer } from './OrderEditDrawer'
import { OrderDetailDrawer } from './OrderDetailDrawer'
import { ConfirmReceiptDrawer } from './ConfirmReceiptDrawer'
import { SuppliersDrawer } from './SuppliersDrawer'
import { ReceiptsPanel } from './ReceiptsPanel'
import { canEditOrder, fmtAmount } from './purchaseLogic'
import { ErrorBanner } from '../../../components/business/page-head'

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
  const [supOpen, setSupOpen] = useState(false)
  const [confirming, setConfirming] = useState<OrderRow | null>(null)
  const [editing, setEditing] = useState<OrderRow | null>(null)
  const [detailId, setDetailId] = useState<number | null>(null)
  const confirm = useConfirm()

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
      toast.success(d.submitOk)
      load()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : d.opFail)
      setBusy(false)
    }
  }

  // cancelOrder 状态机终止:契约 000163 任意非 RECEIVED 状态可 CANCELLED;走 ConfirmDialog,禁物理删除。
  const cancelOrder = async (o: OrderRow) => {
    const ok = await confirm(d.cancelConfirmText, { danger: true, title: d.cancelOrder })
    if (!ok) return
    setBusy(true)
    try {
      await apiFetch(`/procurement/orders/${o.id}/cancel`, { method: 'POST' })
      toast.success(d.cancelOrder)
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
            onClick={() => setSupOpen(true)}
            className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
          >
            {d.suppliersManage}
          </button>
          <button
            type="button"
            onClick={() => setCreateOpen(true)}
            className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
          >
            {d.newOrder}
          </button>
        </div>
        {error && <ErrorBanner message={error} />}
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
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-right">{fmtAmount(o.totalAmount)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">
                      <span className="inline-flex items-center gap-2">
                        <button type="button" disabled={busy} onClick={() => setDetailId(o.id)} className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]">{d.orderDetail}</button>
                        {canEditOrder(o.status) && (
                          <button type="button" disabled={busy} onClick={() => setEditing(o)} className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]">{d.orderEdit}</button>
                        )}
                        {o.status === 'DRAFT' && (
                          <button type="button" disabled={busy} onClick={() => submitOrder(o.id)} className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]">{d.submit}</button>
                        )}
                        {(o.status === 'SUBMITTED' || o.status === 'PARTIAL') && (
                          <button type="button" disabled={busy} onClick={() => setConfirming(o)} className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]">{d.confirmReceipt}</button>
                        )}
                        {(o.status === 'DRAFT' || o.status === 'SUBMITTED' || o.status === 'PARTIAL') && (
                          <button type="button" disabled={busy} onClick={() => cancelOrder(o)} className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--color-danger)] hover:border-[var(--color-border-hover)]">{d.cancelOrder}</button>
                        )}
                        {(o.status === 'RECEIVED' || o.status === 'CANCELLED') && (
                          <span className="text-[var(--shell-group-title)]">—</span>
                        )}
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={d.empty} />}
              </tbody>
            </table>
          </div>
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

      {supOpen && <SuppliersDrawer onClose={() => setSupOpen(false)} onSaved={load} />}

      {editing && (
        <OrderEditDrawer order={editing} suppliers={suppliers}
          onClose={() => setEditing(null)} onSaved={load} />
      )}
      {detailId !== null && <OrderDetailDrawer orderId={detailId} onClose={() => setDetailId(null)} />}

      <ReceiptsPanel />
    </div>
  )
}