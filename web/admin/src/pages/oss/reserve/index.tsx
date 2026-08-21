// 预占与释放页:契约 GET /reserves?portId、POST /reserves/:reserveId/release。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { pageSlice, type ReserveRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow } from '../../../components/business'

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
      .catch((e) => setError(e instanceof Error ? e.message : r.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const release = async (reserveId: number) => {
    if (busy || !(await confirmDialog(r.releaseConfirm, { danger: true }))) return
    setBusy(true)
    try {
      await apiFetch(`/reserves/${reserveId}/release`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : r.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
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
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{r.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{x.id}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{x.portId}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.orderId ? `#${x.orderId}` : '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="reserve" value={x.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {x.status === 'HELD' ? (
                        <span className="inline-flex items-center">
                          <button disabled={busy} onClick={() => release(x.id)}>{r.release}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={r.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </div>
    </div>
  )
}
