// 停复机执行页:契约 GET /stop-resume-tasks(customerId 过滤);失败任务 POST /stop-resume-tasks/:taskId/retry。
// 客户/LO 账号仅后端 ID(接口无姓名快照,登记汇报);过滤走客户选择器(服务端检索)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { searchCustomers } from '../../../api/pickers'
import { pageSlice, type StopResumeTaskRow } from '../types'
import { useCustomerPin } from '../useCustomerPin'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, ErrorBanner } from '../../../components/business'
import { ToolbarButton } from '../../../components/business/page-head'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'

export default function StopSrvPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const s = t.pages.stopsrv
  const [rows, setRows] = useState<StopResumeTaskRow[]>([])
  const [error, setError] = useState('')
  const [customerId, setCustomerId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: StopResumeTaskRow[] }>('/stop-resume-tasks', {
      query: { customerId: customerId || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const retry = async (taskId: number) => {
    if (busy) return
    if (!(await confirmDialog(s.retryConfirm, { danger: true }))) return
    setBusy(true)
    try {
      await apiFetch(`/stop-resume-tasks/${taskId}/retry`, { method: 'POST' })
      toast.success(s.retryOk)
      load()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : s.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  // 钉选回显(W0 基线交接项):已选客户名经详情接口取,保证触发器不回显裸编号。
  const pinnedCustomer = useCustomerPin(customerId)

  return (
    <div>
      <PageHead title={s.title} desc={s.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <ResourcePicker
            value={customerId}
            onChange={(v) => { setCustomerId(v); setPage(1) }}
            search={searchCustomers}
            toOption={(c) => ({ value: String(c.id), label: `${c.name} · ${c.phone || c.customerCode}` })}
            ariaLabel={s.filterCustomer}
            emptyLabel={t.pages.pickers.common.all}
            searchPlaceholder={t.pages.pickers.common.placeholder}
            errorText={s.loadFail}
            pinnedOptions={pinnedCustomer}
          />
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader><TableRow>{s.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell className="font-mono text-[var(--shell-group-title)]">#{r.id}</TableCell>
                    <TableCell className="font-mono text-[var(--shell-group-title)]">#{r.customerId}</TableCell>
                    <TableCell className="font-mono text-[var(--shell-group-title)]">#{r.loAccountId}</TableCell>
                    <TableCell>{r.action === 'STOP' ? s.actionStop : s.actionResume}</TableCell>
                    <TableCell><StatusTag domain="task" value={r.status} /></TableCell>
                    <TableCell>
                      {r.status === 'FAILED' ? (
                        <button disabled={busy} onClick={() => retry(r.id)}
                          className="px-1 text-xs text-[var(--color-text-link)] bg-none border-none cursor-pointer hover:underline disabled:cursor-not-allowed disabled:opacity-50">{s.retry}</button>
                      ) : '—'}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={s.empty} />}
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
