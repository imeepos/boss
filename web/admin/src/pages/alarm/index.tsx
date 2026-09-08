// 告警列表页:契约 GET /alarms?resourceId + POST /alarms/:id/ack + POST /alarms/batch-retest。
// W3 收尾:裸卡片壳/裸 table 收口为 Card/ui-table,工具钮/错误横幅/行内操作走标准组件。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../api/client'
import { useQueryState } from '../../lib/useQueryState'
import { useT } from '../../i18n'
import { PageHead, pagerTexts } from '../org/shared'
import { StatusTag } from '../../components/StatusTag'
import { Pagination } from '../../components/Pagination'
import { ResourcePicker } from '../../components/ResourcePicker'
import { Drawer } from '../../components/Drawer'
import { fmtTime } from '../../lib/format'
import { pageSlice, type AlarmRow } from '../quad/types'
import { useConfirm } from '../../components/ConfirmDialog'
import { ActionLink, ActionLinks, ErrorBanner, TableStateRow, ToolbarButton } from '../../components/business'
import { Card, CardContent, CardFooter } from '../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../components/ui/table'

export default function AlarmPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const a = t.pages.alarmPage
  const [rows, setRows] = useState<AlarmRow[]>([])
  const [error, setError] = useState('')
  const [resourceId, setResourceId] = useState('')
  const [urlStatus] = useQueryState('status', '')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [retestOpen, setRetestOpen] = useState(false)
  const [scope, setScope] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: AlarmRow[] }>('/alarms', { query: { resourceId: resourceId || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const ack = async (alarmId: number) => {
    if (busy || !(await confirmDialog(a.ackConfirm))) return
    setBusy(true)
    try {
      await apiFetch(`/alarms/${alarmId}/ack`, { method: 'POST' })
      toast.success(a.ackOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : a.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const retest = async () => {
    if (busy || !scope.trim()) return
    setBusy(true)
    try {
      const r = await apiFetch<{ taskNo: string }>('/alarms/batch-retest', {
        method: 'POST', body: { scope: scope.trim() },
      })
      toast.success(a.retestDone.replace('{no}', r?.taskNo ?? ''))
      setRetestOpen(false)
      setScope('')
    } catch (e) {
      setError(e instanceof Error ? e.message : a.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const filtered = urlStatus ? rows.filter((row) => row.status === urlStatus) : rows
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <ResourcePicker
              value={resourceId}
              onChange={(v) => { setResourceId(v); setPage(1) }}
              load={() => apiFetch<{ items: { id: number; name: string; code: string }[] }>('/resources').then((x) => x?.items ?? [])}
              toOption={(x) => ({ value: String(x.id), label: `${x.name} (${x.code})` })}
              ariaLabel={a.filterResource}
              emptyLabel={t.pages.pickers.common.all}
              searchPlaceholder={t.pages.pickers.common.placeholder}
              errorText={a.loadFail}
            />
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
            <ToolbarButton primary onClick={() => setRetestOpen(true)}>{a.batchRetest}</ToolbarButton>
          </div>
        </CardContent>
        {error && <ErrorBanner message={error} />}
        {!error && (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {a.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell>{x.alarmNo || `#${x.id}`}</TableCell>
                    <TableCell><StatusTag domain="alarmLevel" value={x.level} /></TableCell>
                    <TableCell>{x.source}</TableCell>
                    <TableCell>{x.content || '—'}</TableCell>
                    <TableCell><StatusTag domain="alarmStatus" value={x.status} /></TableCell>
                    <TableCell>{fmtTime(x.createdAt)}</TableCell>
                    <TableCell>
                      {x.status === 'OPEN' ? (
                        <ActionLinks>
                          <ActionLink onClick={() => ack(x.id)} label={a.ack} />
                        </ActionLinks>
                      ) : '—'}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={a.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </CardFooter>
      </Card>
      {retestOpen && (
        <Drawer title={a.batchRetest} onClose={() => setRetestOpen(false)}
          footer={
            <>
              <ToolbarButton onClick={() => setRetestOpen(false)}>{t.pages.company.cancel}</ToolbarButton>
              <ToolbarButton primary disabled={busy || !scope.trim()} onClick={retest}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </ToolbarButton>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{a.fScope}</label>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={scope} placeholder={a.pScope}
                onChange={(e) => setScope(e.target.value)} />
            </div>
          </div>
        </Drawer>
      )}
    </div>
  )
}
