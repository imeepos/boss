// 告警列表页:契约 GET /alarms?resourceId + POST /alarms/:id/ack + POST /alarms/batch-retest。
import { useEffect, useState } from 'react'
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
import { TableStateRow } from '../../components/business'

export default function AlarmPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const a = t.pages.alarmPage
  const [rows, setRows] = useState<AlarmRow[]>([])
  const [error, setError] = useState('')
  const [hint, setHint] = useState('')
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
    setHint('')
    try {
      const r = await apiFetch<{ taskNo: string }>('/alarms/batch-retest', {
        method: 'POST', body: { scope: scope.trim() },
      })
      setHint(a.retestDone.replace('{no}', r?.taskNo ?? ''))
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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
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
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setRetestOpen(true)}>{a.batchRetest}</button>
        </div>
        {hint && <div style={{ padding: '4px 12px', color: '#1677ff', fontSize: 13 }}>{hint}</div>}
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.alarmNo || `#${x.id}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="alarmLevel" value={x.level} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.source}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.content || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="alarmStatus" value={x.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(x.createdAt)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {x.status === 'OPEN' ? (
                        <span className="inline-flex items-center">
                          <button disabled={busy} onClick={() => ack(x.id)}>{a.ack}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={a.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
      {retestOpen && (
        <Drawer title={a.batchRetest} onClose={() => setRetestOpen(false)}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setRetestOpen(false)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !scope.trim()} onClick={retest}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{a.fScope}</label>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={scope} placeholder={a.pScope}
                onChange={(e) => setScope(e.target.value)} />
              {!scope.trim() && scope !== '' && <span className="text-[11px] text-[var(--color-danger)]">{a.eScope}</span>}
            </div>
          </div>
        </Drawer>
      )}
    </div>
  )
}
