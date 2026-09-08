// 下发任务页:契约 GET /provision-tasks(裸 items)+ POST /provision-tasks/{taskNo}/retry。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { pageSlice, type ProvisionTaskRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { ActionLink, IdRef, TableStateRow, ToolbarButton } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'

const STATUSES = ['PENDING', 'DOING', 'DONE', 'FAILED'] as const

export default function ProvisionTaskPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const p = t.pages.provisionPage
  const [rows, setRows] = useState<ProvisionTaskRow[]>([])
  const [error, setError] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ProvisionTaskRow[] }>('/provision-tasks')
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const retry = async (taskNo: string) => {
    if (busy || !(await confirmDialog(p.retryConfirm))) return
    setBusy(true)
    try {
      await apiFetch(`/provision-tasks/${encodeURIComponent(taskNo)}/retry`, { method: 'POST' })
      toast.success(p.retryOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.actionFail)
      setBusy(false)
    }
  }

  const filtered = status ? rows.filter((x) => x.status === status) : rows
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <Dropdown
              value={status}
              options={[{ value: '', label: p.allStatus }, ...STATUSES.map((st) => ({ value: st, label: st }))]}
              onChange={(v) => { setStatus(v); setPage(1) }}
              ariaLabel={p.allStatus}
            />
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
          </div>
        </CardContent>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{p.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell className="font-mono">{x.taskNo}</TableCell>
                    <TableCell className="font-mono">{x.orderId ? `#${x.orderId}` : '—'}</TableCell>
                    <TableCell>{x.stageEvent || '—'}</TableCell>
                    <TableCell><IdRef value={x.loAccountId} /></TableCell>
                    <TableCell><IdRef value={x.templateId} /></TableCell>
                    <TableCell><StatusTag domain="task" value={x.status} /></TableCell>
                    <TableCell>
                      {x.status === 'FAILED'
                        ? <ActionLink onClick={() => retry(x.taskNo)} label={p.retry} testId={`prov-retry-${x.id}`} />
                        : '—'}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={p.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </CardFooter>
      </Card>
    </div>
  )
}
