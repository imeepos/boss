// 对账与告警页:契约 GET /quad-conflicts + POST /quad-conflicts/:id/resolve + POST /quad-links/reconcile。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useQueryState } from '../../../lib/useQueryState'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type QuadLinkRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { ActionLink, IdRef, TableStateRow, ToolbarButton } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'

interface ReconReport { Total: number; Linked: number; Conflict: number; Unlinked: number }

export default function QuadCheckPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const c = t.pages.quadCheckPage
  const [rows, setRows] = useState<QuadLinkRow[]>([])
  const [error, setError] = useState('')
  const [urlStatus] = useQueryState('status', '')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: QuadLinkRow[] }>('/quad-conflicts')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const reconcile = async () => {
    if (busy || !(await confirmDialog(c.reconcileConfirm))) return
    setBusy(true)
    try {
      const rep = await apiFetch<ReconReport>('/quad-links/reconcile', { method: 'POST' })
      toast.success(c.reconcileDone
        .replace('{total}', String(rep?.Total ?? 0))
        .replace('{linked}', String(rep?.Linked ?? 0))
        .replace('{conflict}', String(rep?.Conflict ?? 0))
        .replace('{unlinked}', String(rep?.Unlinked ?? 0)))
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const resolve = async (id: number) => {
    if (busy || !(await confirmDialog(c.resolveConfirm))) return
    setBusy(true)
    try {
      await apiFetch(`/quad-conflicts/${id}/resolve`, { method: 'POST' })
      toast.success(c.resolveOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const filtered = urlStatus ? rows.filter((row) => row.status === urlStatus) : rows
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
            <ToolbarButton primary disabled={busy} onClick={reconcile}>{c.reconcile}</ToolbarButton>
          </div>
        </CardContent>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{c.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell><IdRef value={x.id} /></TableCell>
                    <TableCell><IdRef value={x.assetId} /></TableCell>
                    <TableCell><IdRef value={x.customerId} /></TableCell>
                    <TableCell><IdRef value={x.portId} /></TableCell>
                    <TableCell><IdRef value={x.addressId} /></TableCell>
                    <TableCell><StatusTag domain="quad" value={x.status} /></TableCell>
                    <TableCell>
                      <ActionLink onClick={() => resolve(x.id)} label={c.resolve} testId={`quad-resolve-${x.id}`} />
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={c.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(c)} />
        </CardFooter>
      </Card>
    </div>
  )
}
