// 资产台账页:列名以 fields.md §4.1 为准;契约 GET /assets(+ /tags 联表标签编号/EPC)。
// 操作列五动作(生命周期/详情/编辑/报废/删除)与建档抽屉;删除 40900 阻断项落页首 error 条。
import { useMemo, useState } from 'react'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type AssetRow } from '../types'
import { AssetTable } from './AssetTable'
import { AssetTrailDrawer } from './TrailDrawer'
import { CreateDrawer } from './CreateDrawer'
import { EditDrawer } from './EditDrawer'
import { ScrapDialog } from './ScrapDialog'
import { useAssetList } from './useAssetList'

export default function AssetPage() {
  const t = useT()
  const a = t.pages.assetPage
  const { rows, error, busy, load, tagOf, delRow } = useAssetList()
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [trail, setTrail] = useState<AssetRow | null>(null)
  const [detail, setDetail] = useState<AssetRow | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [editRow, setEditRow] = useState<AssetRow | null>(null)
  const [scrapRow, setScrapRow] = useState<AssetRow | null>(null)

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
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setCreateOpen(true)}>{a.create}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <AssetTable rows={slice} tagOf={tagOf} busy={busy}
            onTrail={setTrail} onDetail={setDetail} onEdit={setEditRow} onScrap={setScrapRow} onDelete={delRow} />
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
      {trail && <AssetTrailDrawer asset={trail} tag={tagOf(trail.tagId)} onClose={() => setTrail(null)} />}
      {detail && <AssetTrailDrawer asset={detail} tag={tagOf(detail.tagId)} showMain onClose={() => setDetail(null)} />}
      {createOpen && <CreateDrawer onClose={() => setCreateOpen(false)} onSaved={load} />}
      {editRow && <EditDrawer asset={editRow} onClose={() => setEditRow(null)} onSaved={load} />}
      {scrapRow && <ScrapDialog asset={scrapRow} onClose={() => setScrapRow(null)} onSaved={load} />}
    </div>
  )
}