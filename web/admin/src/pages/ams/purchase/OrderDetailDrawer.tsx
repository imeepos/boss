// 采购单详情抽屉:契约 GET /procurement/orders/{id}(单头+明细,明细含已收数量只读回显)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { StatusTag } from '../../../components/StatusTag'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { TableStateRow } from '../../../components/business'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import type { OrderItemRow } from '../types'
import { fmtAmount, unwrapOrderDetail } from './purchaseLogic'

interface OrderDetail {
  id: number
  procurementNo: string
  supplierId: number
  supplierName?: string
  legalEntityId: number
  legalEntityName?: string
  status: string
  totalAmount: number
  expectedDate?: string
  remark?: string
  items: OrderItemRow[]
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[12px] text-[var(--shell-group-title)]">{label}</span>
      <span className="text-[13px] text-[var(--shell-content-text)]">{value || '—'}</span>
    </div>
  )
}

export function OrderDetailDrawer({ orderId, onClose }: { orderId: number; onClose: () => void }) {
  const t = useT()
  const d = t.pages.purchasePage
  const [detail, setDetail] = useState<OrderDetail | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(true)

  useEffect(() => {
    setBusy(true)
    apiFetch<{ item: OrderDetail }>('/procurement/orders/' + orderId)
      .then((x) => setDetail(unwrapOrderDetail(x)))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }, [orderId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Drawer title={d.orderDetailTitle} onClose={onClose} width={640}
      footer={<ToolbarButton onClick={onClose}>{t.pages.company.cancel}</ToolbarButton>}>
      {error ? (
        <div className="p-4"><ErrorBanner message={error} /></div>
      ) : busy || !detail ? (
        <div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{t.common.loading}</div>
      ) : (
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-3 rounded-sm border border-[var(--shell-side-border)] p-3 md:grid-cols-3">
            <Field label={d.colNo} value={detail.procurementNo} />
            <Field label={d.dSupplier} value={detail.supplierName || '#' + detail.supplierId} />
            <Field label={d.dEntity} value={detail.legalEntityName || '#' + detail.legalEntityId} />
            <div className="flex flex-col gap-1">
              <span className="text-[12px] text-[var(--shell-group-title)]">{d.colStatus}</span>
              <StatusTag domain="procurement" value={detail.status} />
            </div>
            <Field label={d.dTotal} value={fmtAmount(detail.totalAmount)} />
            <Field label={d.dExpected} value={detail.expectedDate ?? ''} />
            <div className="col-span-2 md:col-span-3">
              <Field label={d.dRemark} value={detail.remark ?? ''} />
            </div>
          </div>
          <Table>
            <TableHeader>
              <TableRow>{d.detailItemCols.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {(detail.items ?? []).map((i, idx) => (
                <TableRow key={idx}>
                  <TableCell className="font-mono">{i.materialCode}</TableCell>
                  <TableCell>{i.spec || '—'}</TableCell>
                  <TableCell className="text-right">{i.quantity}</TableCell>
                  <TableCell className="text-right">{fmtAmount(i.unitAmount)}</TableCell>
                  <TableCell className="text-right">{i.receivedQty ?? 0}</TableCell>
                </TableRow>
              ))}
              <TableStateRow colSpan={d.detailItemCols.length} loading={false} text={d.empty} />
            </TableBody>
          </Table>
        </div>
      )}
    </Drawer>
  )
}
