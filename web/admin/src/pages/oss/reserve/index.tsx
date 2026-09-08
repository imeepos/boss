// 预占与释放页:契约 GET /reserves?portId、POST /reserves/:reserveId/release。
import { IdRef } from '../../../components/business'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { pageSlice, type ReserveRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, ErrorBanner, ToolbarButton } from '../../../components/business'

export default function ReservePage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const r = t.pages.reservePage
  const [rows, setRows] = useState<ReserveRow[]>([])
  const [error, setError] = useState('')
  const [portId, setPortId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReserveRow[] }>('/reserves', {
      query: { portId: portId || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => {
        const msg = e instanceof Error ? e.message : r.loadFail
        setError(msg)
        toast.error(r.loadFail, { description: msg })
      })
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const release = async (reserveId: number) => {
    if (busy || !(await confirmDialog(r.releaseConfirm, { danger: true }))) return
    setBusy(true)
    try {
      await apiFetch(`/reserves/${reserveId}/release`, { method: 'POST' })
      toast.success(r.release)
      load()
    } catch (e) {
      const msg = e instanceof Error ? e.message : r.actionFail
      setError(msg)
      toast.error(r.actionFail, { description: msg })
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <Card className="mb-4">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <ResourcePicker
            value={portId}
            onChange={(v) => { setPortId(v); setPage(1) }}
            load={() => apiFetch<{ items: { portId: number; portCode: string }[] }>('/ports').then((x) => x?.items ?? [])}
            toOption={(p) => ({ value: String(p.portId), label: p.portCode })}
            ariaLabel={r.filterPort}
            emptyLabel={t.pages.pickers.common.all}
            searchPlaceholder={t.pages.pickers.common.placeholder}
            errorText={r.loadFail}
          />
          <span className="flex-1" />
          <ToolbarButton onClick={load} disabled={busy}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <div className="px-4 pb-3"><ErrorBanner message={error} /></div> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {r.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell><IdRef value={x.id} /></TableCell>
                    <TableCell><IdRef value={x.portId} /></TableCell>
                    <TableCell>{x.orderId ? <IdRef value={x.orderId} /> : '—'}</TableCell>
                    <TableCell><StatusTag domain="reserve" value={x.status} /></TableCell>
                    <TableCell>
                      {x.status === 'HELD' ? (
                        <button type="button" disabled={busy} onClick={() => release(x.id)}>{r.release}</button>
                      ) : '—'}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={r.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </CardFooter>
      </Card>
    </div>
  )
}
