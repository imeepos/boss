// 激活回调页(订单第 11 环节):契约 GET /activation-callbacks(裸列表)+ POST /activation-callbacks/:id/retry。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton, IdRef } from '../../../components/business'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { pageSlice, type ActivationCallbackRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow } from '../../../components/business'

export default function CallbackPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const c = t.pages.callbackPage
  const [rows, setRows] = useState<ActivationCallbackRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<ActivationCallbackRow[]>('/activation-callbacks')
      .then((x) => setRows(Array.isArray(x) ? x : []))
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const retry = async (id: number) => {
    if (busy || !(await confirmDialog(c.retryConfirm))) return
    setBusy(true)
    try {
      await apiFetch(`/activation-callbacks/${id}/retry`, { method: 'POST' })
      toast.success(c.toastRetryOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error && <ErrorBanner message={error} />}
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>{c.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((x) => (
                <TableRow key={x.id}>
                  <TableCell><IdRef value={x.id} /></TableCell>
                  <TableCell><IdRef value={x.orderId} /></TableCell>
                  <TableCell><StatusTag domain="callbackResult" value={x.result} /></TableCell>
                  <TableCell>{x.retries}</TableCell>
                  <TableCell>
                    {x.result === 'FAILED' ? (
                      <button type="button" disabled={busy} onClick={() => retry(x.id)}
                        className="cursor-pointer border-none bg-none px-0 text-xs text-[var(--color-text-link)] hover:underline disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:no-underline">
                        {c.retry}
                      </button>
                    ) : '—'}
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={5} loading={busy} text={c.empty} />}
            </TableBody>
          </Table>
        </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(c)} />
        </div>
      </Card>
    </div>
  )
}
