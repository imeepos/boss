// 采购单详情抽屉:契约 GET /procurement/orders/{id}(单头+明细,明细含已收数量只读回显)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { StatusTag } from '../../../components/StatusTag'
import type { OrderItemRow } from '../types'

const errBanner = 'rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

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
    apiFetch<OrderDetail>('/procurement/orders/' + orderId)
      .then((x) => setDetail(x))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }, [orderId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Drawer title={d.orderDetailTitle} onClose={onClose} width={640}
      footer={<button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>}>
      {error ? (
        <div className={errBanner}>{error}</div>
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
            <Field label={d.dTotal} value={detail.totalAmount.toFixed(2)} />
            <Field label={d.dExpected} value={detail.expectedDate ?? ''} />
            <div className="col-span-2 md:col-span-3">
              <Field label={d.dRemark} value={detail.remark ?? ''} />
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-[13px]">
              <thead>
                <tr>{d.detailItemCols.map((x) => <th key={x} className="h-9 px-2 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr>
              </thead>
              <tbody>
                {(detail.items ?? []).map((i, idx) => (
                  <tr key={idx}>
                    <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] font-mono text-[var(--shell-content-text)]">{i.materialCode}</td>
                    <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)]">{i.spec || '—'}</td>
                    <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] text-right text-[var(--shell-content-text)]">{i.quantity}</td>
                    <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] text-right text-[var(--shell-content-text)]">{i.unitAmount.toFixed(2)}</td>
                    <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] text-right text-[var(--shell-content-text)]">{i.receivedQty ?? 0}</td>
                  </tr>
                ))}
                {!(detail.items ?? []).length && (
                  <tr><td className="h-9 px-2 text-center text-[var(--shell-group-title)]" colSpan={d.detailItemCols.length}>{d.empty}</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </Drawer>
  )
}
