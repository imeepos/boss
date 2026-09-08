// 入库单区:GET /procurement/receipts 列表(采购单页内独立卡片);DRAFT 行提供驳回入口。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { StatusTag } from '../../../components/StatusTag'
import { ActionLink, TableStateRow, ToolbarButton } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { type ReceiptRow } from '../types'
import { canRejectReceipt } from './purchaseLogic'
import { RejectReceiptDrawer } from './RejectReceiptDrawer'

export function ReceiptsPanel() {
  const t = useT()
  const d = t.pages.purchasePage
  const [rows, setRows] = useState<ReceiptRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [rejecting, setRejecting] = useState<ReceiptRow | null>(null)

  const load = useCallback(() => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReceiptRow[] }>('/procurement/receipts')
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(load, [load])

  const cols = [d.colReceiptNo, d.colOrderNo, d.colStatus, d.colReceivedAt, d.colActions]
  return (
    <Card>
      <div className="flex flex-wrap items-center gap-2 p-4">
        <div className="text-[13px] font-medium text-[var(--shell-heading)]">{d.receiptsTitle}</div>
        <span className="spacer" />
        <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
      </div>
      {error && <ErrorBanner message={error} />}
      <div className="px-4 pb-4">
        <Table>
          <TableHeader>
            <TableRow>{cols.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((r) => (
              <TableRow key={r.id}>
                <TableCell className="font-mono">{r.receiptNo || '#' + r.id}</TableCell>
                <TableCell className="font-mono">{r.orderNo || '#' + r.orderId}</TableCell>
                <TableCell><StatusTag domain="receipt" value={r.status} /></TableCell>
                <TableCell>{r.receivedAt || '—'}</TableCell>
                <TableCell>
                  {canRejectReceipt(r.status)
                    ? <ActionLink onClick={() => setRejecting(r)} label={d.reject} testId={`receipt-reject-${r.id}`} />
                    : <span className="text-[var(--shell-group-title)]">—</span>}
                </TableCell>
              </TableRow>
            ))}
            {!rows.length && <TableStateRow colSpan={5} loading={busy} text={d.receiptsEmpty} />}
          </TableBody>
        </Table>
      </div>
      {rejecting && (
        <RejectReceiptDrawer receipt={rejecting} onClose={() => setRejecting(null)} onSaved={load} />
      )}
    </Card>
  )
}
