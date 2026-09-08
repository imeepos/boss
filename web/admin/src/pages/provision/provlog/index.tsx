// 下发日志页:契约 GET /provision-logs?taskId(只读,成功/失败/重试留痕)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type ProvisionLogDetail, type ProvisionLogRow } from '../types'
import { ActionLink, IdRef, TableStateRow, ToolbarButton } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { ProvisionLogDetailDrawer } from './detail'

export default function ProvisionLogPage() {
  const t = useT()
  const p = t.pages.provlogPage
  const [rows, setRows] = useState<ProvisionLogRow[]>([])
  const [error, setError] = useState('')
  const [taskId, setTaskId] = useState('')
  const [result, setResult] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [detail, setDetail] = useState<ProvisionLogDetail | null>(null)
  const [detailBusy, setDetailBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ProvisionLogRow[] }>('/provision-logs', {
      query: { taskId: taskId ? Number(taskId) || undefined : undefined },
    })
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openDetail = (id: number) => {
    if (detailBusy) return
    setError('')
    setDetailBusy(true)
    apiFetch<ProvisionLogDetail>(`/provision-logs/${id}`)
      .then((x) => setDetail(x))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setDetailBusy(false))
  }

  const filtered = result ? rows.filter((x) => x.result === result) : rows
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <ResourcePicker
              value={taskId}
              onChange={(v) => { setTaskId(v); setPage(1) }}
              load={() => apiFetch<{ items: { id: number; taskNo: string; stageEvent: string }[] }>('/provision-tasks').then((x) => x?.items ?? [])}
              toOption={(x) => ({ value: String(x.id), label: `${x.taskNo} · ${x.stageEvent}` })}
              ariaLabel={p.filterTask}
              emptyLabel={t.pages.pickers.common.all}
              searchPlaceholder={t.pages.pickers.common.placeholder}
              errorText={p.loadFail}
            />
            <Dropdown
              value={result}
              options={[{ value: '', label: p.allResult }, { value: 'SUCCESS', label: 'SUCCESS' }, { value: 'FAILED', label: 'FAILED' }]}
              onChange={(v) => { setResult(v); setPage(1) }}
              ariaLabel={p.allResult}
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
                    <TableCell><IdRef value={x.id} /></TableCell>
                    <TableCell><IdRef value={x.taskId} /></TableCell>
                    <TableCell className="font-mono">{x.resourceCode || `#${x.resourceId}`}</TableCell>
                    <TableCell className="font-mono">{x.templateCode || `#${x.templateId}`}</TableCell>
                    <TableCell>{x.result}</TableCell>
                    <TableCell>{x.retries}</TableCell>
                    <TableCell>{fmtTime(x.createdAt)}</TableCell>
                    <TableCell>
                      <ActionLink onClick={() => openDetail(x.id)} label={p.detail} testId={`provlog-detail-${x.id}`} />
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={p.columns.length} loading={busy} text={p.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </CardFooter>
      </Card>
      {detail && <ProvisionLogDetailDrawer detail={detail} onClose={() => setDetail(null)} />}
    </div>
  )
}
