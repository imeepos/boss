// 电子标签页:契约 GET /tags;绑定资产经 boundAssetId 反查展示。
// P2-T4:操作列(Dropdown 体系,禁用原生 select)= 事件记录/解绑/报废绑定资产;
// 解绑与报废走危险确认三要素(影响面+不可逆说明+原因必填),原因入事件 detail。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { pageSlice, type AssetRow, type TagRow } from '../types'
import { EventDrawer } from '../EventDrawer'
import { ScrapConfirmDialog, UnbindConfirmDialog } from '../DangerOps'
import { TableStateRow } from '../../../components/business'

export default function TagPage() {
  const t = useT()
  const g = t.pages.tagPage
  const e = t.pages.eventOps
  const [rows, setRows] = useState<TagRow[]>([])
  const [assets, setAssets] = useState<AssetRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [opBusy, setOpBusy] = useState(false)
  const [eventsFor, setEventsFor] = useState<TagRow | null>(null)
  const [unbindFor, setUnbindFor] = useState<TagRow | null>(null)
  const [scrapFor, setScrapFor] = useState<TagRow | null>(null)

  const load = () => {
    setError('')
    setBusy(true)
    Promise.all([
      apiFetch<{ items: TagRow[] }>('/tags'),
      apiFetch<{ items: AssetRow[] }>('/assets').catch(() => ({ items: [] as AssetRow[] })),
    ])
      .then(([d, as]) => { setRows(d?.items ?? []); setAssets(as?.items ?? []) })
      .catch((err) => setError(err instanceof Error ? err.message : g.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const assetOf = (assetId: number) => assets.find((x) => x.assetId === assetId)
  const unbindTag = (reason: string) => {
    if (!unbindFor) return
    setOpBusy(true)
    apiFetch('/tags/' + unbindFor.tagId + '/unbind', { method: 'POST', body: { reason } })
      .then(() => { setUnbindFor(null); load() })
      .catch(() => setError(e.opFail))
      .finally(() => setOpBusy(false))
  }
  const scrapAsset = (reason: string) => {
    if (!scrapFor) return
    setOpBusy(true)
    apiFetch('/assets/' + scrapFor.boundAssetId + '/scrap', { method: 'POST', body: { reason } })
      .then(() => { setScrapFor(null); load() })
      .catch(() => setError(e.opFail))
      .finally(() => setOpBusy(false))
  }

  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return rows
    return rows.filter((r) => r.tagNo.toLowerCase().includes(k) || r.epcCode.toLowerCase().includes(k))
  }, [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)

  const actionOpts = (r: TagRow) => [
    { value: 'events', label: e.actEvents },
    { value: 'unbind', label: e.actUnbind, disabled: !r.boundAssetId },
    { value: 'scrap', label: e.actScrapAsset, disabled: !r.boundAssetId },
  ]
  const onAction = (r: TagRow, v: string) => {
    if (v === 'events') setEventsFor(r)
    else if (v === 'unbind') setUnbindFor(r)
    else if (v === 'scrap') setScrapFor(r)
  }

  const scrapTarget = scrapFor ? assetOf(scrapFor.boundAssetId) : null
  return (
    <div>
      <PageHead title={g.title} desc={g.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={g.searchPlaceholder}
            value={keyword} onChange={(ev) => { setKeyword(ev.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{g.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.tagId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.tagNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.epcCode}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.band || '-'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.boundAssetId ? '#' + r.boundAssetId : '-'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.battery || '-'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="tag" value={r.status} /></td>
                    <td className="h-11 px-3 border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <Dropdown value="" options={actionOpts(r)} onChange={(v) => onAction(r, v)} ariaLabel={e.menu} placeholder={e.colAction} />
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
      {eventsFor && <EventDrawer kind="tag" id={eventsFor.tagId} code={eventsFor.tagNo} onClose={() => setEventsFor(null)} />}
      {unbindFor && <UnbindConfirmDialog tag={unbindFor} busy={opBusy} onClose={() => setUnbindFor(null)} onConfirm={unbindTag} />}
      {scrapFor && <ScrapConfirmDialog assetCode={scrapTarget?.assetCode ?? ''} assetId={scrapFor.boundAssetId} tag={scrapFor} busy={opBusy || !scrapTarget} onClose={() => setScrapFor(null)} onConfirm={scrapAsset} />}
    </div>
  )
}
