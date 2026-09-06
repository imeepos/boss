// 资产台账页:列名以 fields.md §4.1 为准;契约 GET /assets(+ /tags 联表标签编号/EPC)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type AssetRow, type TagRow } from '../types'
import { Dropdown } from '../../../components/Dropdown'
import { EventDrawer } from '../EventDrawer'
import { ScrapConfirmDialog } from '../DangerOps'
import { AssetTrailDrawer } from './TrailDrawer'
import { TableStateRow } from '../../../components/business'

export default function AssetPage() {
  const t = useT()
  const a = t.pages.assetPage
  const eo = t.pages.eventOps
  const [rows, setRows] = useState<AssetRow[]>([])
  const [tags, setTags] = useState<TagRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [trail, setTrail] = useState<AssetRow | null>(null)
  const [eventsFor, setEventsFor] = useState<AssetRow | null>(null)
  const [scrapFor, setScrapFor] = useState<AssetRow | null>(null)
  const [opBusy, setOpBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    Promise.all([
      apiFetch<{ items: AssetRow[] }>('/assets'),
      apiFetch<{ items: TagRow[] }>('/tags').catch(() => ({ items: [] as TagRow[] })),
    ])
      .then(([d, tg]) => { setRows(d?.items ?? []); setTags(tg?.items ?? []) })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const tagOf = (tagId: number) => tags.find((x) => x.tagId === tagId)
  const scrapAsset = (reason: string) => {
    if (!scrapFor) return
    setOpBusy(true)
    apiFetch('/assets/' + scrapFor.assetId + '/scrap', { method: 'POST', body: { reason } })
      .then(() => { setScrapFor(null); load() })
      .catch(() => setError(eo.opFail))
      .finally(() => setOpBusy(false))
  }
  const actionOpts = (r: AssetRow) => [
    { value: 'trail', label: a.lifecycle },
    { value: 'events', label: eo.actEvents },
    { value: 'scrap', label: eo.actScrap, disabled: r.status === 'SCRAPPED' },
  ]
  const onAction = (r: AssetRow, v: string) => {
    if (v === 'trail') setTrail(r)
    else if (v === 'events') setEventsFor(r)
    else if (v === 'scrap') setScrapFor(r)
  }
  const filtered = useMemo(
    () => rows.filter((r) => r.assetCode.toLowerCase().includes(keyword.trim().toLowerCase())),
    [rows, keyword],
  )
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={a.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.assetId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.assetCode}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{tagOf(r.tagId)?.tagNo ?? '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{tagOf(r.tagId)?.epcCode ?? '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.type || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{r.batchId}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.addressId ? `#${r.addressId}` : '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="asset" value={r.status} /></td>
                    <td className="h-11 px-3 border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <Dropdown value="" options={actionOpts(r)} onChange={(v) => onAction(r, v)} ariaLabel={eo.menu} placeholder={eo.colAction} />
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={8} loading={busy} text={a.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
      {trail && <AssetTrailDrawer asset={trail} tag={tagOf(trail.tagId)} onClose={() => setTrail(null)} />}
      {eventsFor && <EventDrawer kind="asset" id={eventsFor.assetId} code={eventsFor.assetCode} onClose={() => setEventsFor(null)} />}
      {scrapFor && <ScrapConfirmDialog assetCode={scrapFor.assetCode} assetId={scrapFor.assetId} tag={tagOf(scrapFor.tagId)} busy={opBusy} onClose={() => setScrapFor(null)} onConfirm={scrapAsset} />}
    </div>
  )
}
