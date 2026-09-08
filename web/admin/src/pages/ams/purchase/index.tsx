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
import { ActionLink, ActionLinks, ActionSep, TableStateRow, ToolbarButton } from '../../../components/business'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
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

      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <Dropdown
              value={statusFilter}
              options={statusOpts}
              onChange={setStatusFilter}
              ariaLabel={d.statusFilter}
            />
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
            <ToolbarButton onClick={() => setSupOpen(true)}>{d.suppliersManage}</ToolbarButton>
            <ToolbarButton primary onClick={() => setCreateOpen(true)}>{d.newOrder}</ToolbarButton>
          </div>
        </CardContent>
        {error && <ErrorBanner message={error} />}
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{d.colNo}</TableHead>
                <TableHead>{d.colSupplier}</TableHead>
                <TableHead>{d.colEntity}</TableHead>
                <TableHead>{d.colStatus}</TableHead>
                <TableHead className="text-right">{d.colTotal}</TableHead>
                <TableHead>{d.colActions}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((o) => (
                <TableRow key={o.id}>
                  <TableCell className="font-mono">{o.procurementNo}</TableCell>
                  <TableCell>{o.supplierName}</TableCell>
                  <TableCell>{o.legalEntityName}</TableCell>
                  <TableCell><StatusTag domain="procurement" value={o.status} /></TableCell>
                  <TableCell className="text-right">{fmtAmount(o.totalAmount)}</TableCell>
                  <TableCell>
                    <ActionLinks>
                      <ActionLink onClick={() => setDetailId(o.id)} label={d.orderDetail} testId={`purchase-detail-${o.id}`} />
                      {canEditOrder(o.status) && (
                        <>
                          <ActionSep />
                          <ActionLink onClick={() => setEditing(o)} label={d.orderEdit} testId={`purchase-edit-${o.id}`} />
                        </>
                      )}
                      {o.status === 'DRAFT' && (
                        <>
                          <ActionSep />
                          <ActionLink onClick={() => submitOrder(o.id)} label={d.submit} testId={`purchase-submit-${o.id}`} />
                        </>
                      )}
                      {(o.status === 'SUBMITTED' || o.status === 'PARTIAL') && (
                        <>
                          <ActionSep />
                          <ActionLink onClick={() => setConfirming(o)} label={d.confirmReceipt} testId={`purchase-confirm-${o.id}`} />
                        </>
                      )}
                      {(o.status === 'DRAFT' || o.status === 'SUBMITTED' || o.status === 'PARTIAL') && (
                        <>
                          <ActionSep />
                          <ActionLink onClick={() => cancelOrder(o)} label={d.cancelOrder} testId={`purchase-cancel-${o.id}`} />
                        </>
                      )}
                      {(o.status === 'RECEIVED' || o.status === 'CANCELLED') && (
                        <span className="text-[var(--shell-group-title)]">—</span>
                      )}
                    </ActionLinks>
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={6} loading={busy} text={d.empty} />}
            </TableBody>
          </Table>
        </div>
        <CardFooter>
          <Pagination
            total={filtered.length}
            page={page}
            pageSize={pageSize}
            onPage={setPage}
            onSize={setPageSize}
            {...pagerTexts(t.pages.company)}
          />
        </CardFooter>
      </Card>

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