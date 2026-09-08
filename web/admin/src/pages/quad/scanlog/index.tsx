// 扫码绑定记录页:契约 GET /scan-logs?orderId。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { pageSlice, type ScanLogRow } from '../types'
import { IdRef, TableStateRow, ToolbarButton } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'

export default function ScanLogPage() {
  const t = useT()
  const s = t.pages.scanlogPage
  const [rows, setRows] = useState<ScanLogRow[]>([])
  const [error, setError] = useState('')
  const [orderId, setOrderId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ScanLogRow[] }>('/scan-logs', { query: { orderId: orderId || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={s.title} desc={s.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <ResourcePicker
              value={orderId}
              onChange={(v) => { setOrderId(v); setPage(1) }}
              load={() => apiFetch<{ items: { id: number; orderNo: string }[] }>('/orders').then((x) => x?.items ?? [])}
              toOption={(o) => ({ value: String(o.id), label: o.orderNo })}
              ariaLabel={s.filterOrder}
              emptyLabel={t.pages.pickers.common.all}
              searchPlaceholder={t.pages.pickers.common.placeholder}
              errorText={s.loadFail}
            />
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
          </div>
        </CardContent>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{s.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell><IdRef value={x.id} /></TableCell>
                    <TableCell><IdRef value={x.orderId} /></TableCell>
                    <TableCell>{x.workerName || (x.workerId ? `#${x.workerId}` : '—')}</TableCell>
                    <TableCell>{x.tagId ? `#${x.tagId}` : '—'}</TableCell>
                    <TableCell><StatusTag domain="scan" value={x.result} /></TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={s.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(s)} />
        </CardFooter>
      </Card>
    </div>
  )
}
