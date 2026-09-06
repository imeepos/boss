// 电子标签页:契约 GET /tags、POST /tags、POST /:id/disable|enable、/:id/unbind、GET /:id/events。
// 状态枚举 terms.md §4:UNBOUND/BOUND/DISABLED;操作显隐见 ./logic。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { useConfirm } from '../../../components/ConfirmDialog'
import { pageSlice, type TagRow } from '../types'
import { TableStateRow } from '../../../components/business'
import { tagActionsOf } from './logic'
import { CreateTagDrawer } from './CreateTagDrawer'
import { TagEventsDrawer } from './EventsDrawer'

const td = 'h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'
const th = 'h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
const actBtn = 'h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-50'

export default function TagPage() {
  const t = useT()
  const g = t.pages.tagPage
  const confirm = useConfirm()
  const [rows, setRows] = useState<TagRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [eventsTag, setEventsTag] = useState<TagRow | null>(null)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: TagRow[] }>('/tags')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : g.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const runOp = async (row: TagRow, path: string, okMsg: string, body?: unknown) => {
    setBusy(true)
    try {
      await apiFetch('/tags/' + row.tagId + path, { method: 'POST', body })
      toast.success(okMsg)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : g.loadFail)
      setBusy(false)
    }
  }

  const disableTag = async (r: TagRow) => {
    const no = r.tagNo || '#' + r.tagId
    if (!(await confirm(g.disableConfirm.replace('{no}', no), { danger: true, title: g.disable }))) return
    await runOp(r, '/disable', g.disable)
  }
  const enableTag = (r: TagRow) => runOp(r, '/enable', g.enable)
  const unbindTag = async (r: TagRow) => {
    const no = r.tagNo || '#' + r.tagId
    if (!(await confirm(g.unbindConfirm.replace('{no}', no), { danger: true, title: g.unbind }))) return
    await runOp(r, '/unbind', g.unbindOk, { expectedAssetId: r.boundAssetId || undefined })
  }

  const actionLabel = (act: string): string => {
    if (act === 'disable') return g.disable
    if (act === 'enable') return g.enable
    if (act === 'unbind') return g.unbind
    return g.events
  }
  const actionRun = (act: string, r: TagRow) => {
    if (act === 'disable') return disableTag(r)
    if (act === 'enable') return enableTag(r)
    if (act === 'unbind') return unbindTag(r)
    setEventsTag(r)
  }

  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return rows
    return rows.filter((r) => r.tagNo.toLowerCase().includes(k) || r.epcCode.toLowerCase().includes(k))
  }, [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)
  const cols = [...g.columns, g.colActions]

  return (
    <div>
      <PageHead title={g.title} desc={g.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={g.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setCreateOpen(true)}>{g.create}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead><tr>{cols.map((x) => <th key={x} className={th}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.tagId}>
                    <td className={td}>{r.tagNo}</td>
                    <td className={td}>{r.epcCode}</td>
                    <td className={td}>{r.band || '—'}</td>
                    <td className={td}>{r.boundAssetId ? '#' + r.boundAssetId : '—'}</td>
                    <td className={td}>{r.battery || '—'}</td>
                    <td className={td}><StatusTag domain="tag" value={r.status} /></td>
                    <td className={td}>
                      <span className="inline-flex items-center gap-2">
                        {tagActionsOf(r).map((act) => (
                          <button key={act} className={actBtn} disabled={busy} onClick={() => actionRun(act, r)}>{actionLabel(act)}</button>
                        ))}
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={g.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(g)} />
        </div>
      </div>
      {createOpen && <CreateTagDrawer onClose={() => setCreateOpen(false)} onSaved={load} />}
      {eventsTag && <TagEventsDrawer tag={eventsTag} onClose={() => setEventsTag(null)} />}
    </div>
  )
}
