// 企业订单(partner_admin/partner_staff):本企业订单只读列表(orders 按 legal_entity 隔离)。
import { useEffect, useState } from 'react'
import { listPartnerOrders, type PartnerOrder } from '../../../api/partner'
import { useT } from '../../../i18n'
import { fmtTime } from '../../../lib/format'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { StatusTag } from '../../../components/StatusTag'
import { PageHead, ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'

export default function PartnerOrdersPage() {
  const t = useT()
  const p = t.pages.partnerOrders
  const [items, setItems] = useState<PartnerOrder[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError(''); setBusy(true)
    listPartnerOrders()
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <div className="mb-3 flex items-center">
        <div className="flex-1" />
        <ToolbarButton onClick={load} disabled={busy}>{p.refresh}</ToolbarButton>
      </div>
      <Card className="p-4">
        {error ? <ErrorBanner message={error} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{p.colOrderNo}</TableHead>
                <TableHead>{p.colCustomer}</TableHead>
                <TableHead>{p.colStage}</TableHead>
                <TableHead>{p.colStatus}</TableHead>
                <TableHead>{p.colCreatedAt}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-medium">{r.orderNo}</TableCell>
                  <TableCell>{r.customerName || '-'}</TableCell>
                  <TableCell>{p.stageUnit.replace('{n}', String(r.stage))}</TableCell>
                  <TableCell><StatusTag domain="order" value={r.status} /></TableCell>
                  <TableCell className="whitespace-nowrap">{fmtTime(r.createdAt)}</TableCell>
                </TableRow>
              ))}
              {!items.length && (
                <TableRow><TableCell colSpan={5}><EmptyState text={p.empty} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  )
}
