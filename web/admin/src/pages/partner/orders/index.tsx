// 企业订单(partner_admin/partner_staff):本企业订单只读列表(orders 按 legal_entity 隔离)。
import { useEffect, useState } from 'react'
import { listPartnerOrders, type PartnerOrder } from '../../../api/partner'
import { useT } from '../../../i18n'
import { fmtTime } from '../../../lib/format'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { PageHead, ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'

/** 订单状态徽标(枚举见 terms.md 第 3 节)。 */
function OrderStatusBadge({ status }: { status: string }) {
  const map: Record<string, 'warning' | 'info' | 'success' | 'danger' | 'default'> = {
    PENDING: 'warning', RESERVED: 'info', INSTALLING: 'info',
    DONE: 'success', CANCELLED: 'danger',
  }
  return <Badge variant={map[status] ?? 'default'}>{status}</Badge>
}

export default function PartnerOrdersPage() {
  const t = useT()
  const [items, setItems] = useState<PartnerOrder[]>([])
  const [error, setError] = useState('')

  const load = () => {
    setError('')
    listPartnerOrders()
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.partnerOrders.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div>
      <PageHead title={t.pages.partnerOrders.title} desc={t.pages.partnerOrders.desc} />
      <div className="mb-3 flex items-center">
        <div className="flex-1" />
        <ToolbarButton onClick={load}>{t.pages.partnerOrders.refresh}</ToolbarButton>
      </div>
      <Card className="p-4">
        {error ? <ErrorBanner message={error} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t.pages.partnerOrders.colOrderNo}</TableHead>
                <TableHead>{t.pages.partnerOrders.colCustomer}</TableHead>
                <TableHead>{t.pages.partnerOrders.colStage}</TableHead>
                <TableHead>{t.pages.partnerOrders.colStatus}</TableHead>
                <TableHead>{t.pages.partnerOrders.colCreatedAt}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-medium">{r.orderNo}</TableCell>
                  <TableCell>{r.customerName || '-'}</TableCell>
                  <TableCell>{t.pages.partnerOrders.stageUnit.replace('{n}', String(r.stage))}</TableCell>
                  <TableCell><OrderStatusBadge status={r.status} /></TableCell>
                  <TableCell className="whitespace-nowrap">{fmtTime(r.createdAt)}</TableCell>
                </TableRow>
              ))}
              {!items.length && (
                <TableRow><TableCell colSpan={5}><EmptyState text={t.pages.partnerOrders.empty} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  )
}
