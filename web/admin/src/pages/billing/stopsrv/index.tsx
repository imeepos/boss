// 停复机执行页:契约 GET /stop-resume-tasks(customerId 过滤);失败任务 POST /stop-resume-tasks/:taskId/retry。
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
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, ErrorBanner, IdRef } from '../../../components/business'

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
      setError(e instanceof Error ? e.message : s.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={s.title} desc={s.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
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
          />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{s.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><IdRef value={r.id} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><IdRef value={r.customerId} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><IdRef value={r.loAccountId} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.action === 'STOP' ? s.actionStop : s.actionResume}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="task" value={r.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {r.status === 'FAILED' ? (
                        <span className="inline-flex items-center">
                          <button disabled={busy} onClick={() => retry(r.id)}>{s.retry}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={s.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(s)} />
        </div>
      </div>
    </div>
  )
}
