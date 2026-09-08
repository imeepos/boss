// 资产台账页:列名以 fields.md §4.1 为准;契约 GET /assets(P3-T1 服务端分页)。
// 翻页/每页条数/状态筛选/q(防抖)变更均触发服务端请求;不再全量拉取+前端切片。
import { useEffect, useState } from 'react'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { statusTagLabel } from '../../../components/StatusTag'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { ASSET_TYPES, SCAN_STATUSES, type AssetRow } from '../types'
import { AssetTable } from './AssetTable'
import { AssetTrailDrawer } from './TrailDrawer'
import { CreateDrawer } from './CreateDrawer'
import { EditDrawer } from './EditDrawer'
import { ScrapDialog } from './ScrapDialog'
import { useAssetList } from './useAssetList'
import { ModelDictDrawer } from './ModelDictDrawer'
import { BatchDrawer } from './BatchDrawer'

const ALL = ''
const Q_DEBOUNCE_MS = 300

export default function AssetPage() {
  const t = useT()
  const a = t.pages.assetPage
  const [keyword, setKeyword] = useState('')
  const [q, setQ] = useState('')
  const [status, setStatus] = useState(ALL)
  const [typeF, setTypeF] = useState(ALL)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [trail, setTrail] = useState<AssetRow | null>(null)
  const [detail, setDetail] = useState<AssetRow | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [editRow, setEditRow] = useState<AssetRow | null>(null)
  const [scrapRow, setScrapRow] = useState<AssetRow | null>(null)
  const [modelsOpen, setModelsOpen] = useState(false)
  const [batchesOpen, setBatchesOpen] = useState(false)
  const { rows, total, error, busy, load, tagOf, batchOf, delRow } = useAssetList({ page, pageSize, status, typeF, q })

  // q 防抖:输入停顿后下发服务端,并回第一页(挂载时同值回写不触发请求)。
  useEffect(() => {
    const h = setTimeout(() => { setQ(keyword.trim()); setPage(1) }, Q_DEBOUNCE_MS)
    return () => clearTimeout(h)
  }, [keyword])

  // 服务端 total 收缩(如他处删除)导致当前页空:回缩到最后非空页。
  useEffect(() => {
    if (!busy && total > 0 && rows.length === 0 && page > 1) {
      setPage(Math.max(1, Math.ceil(total / pageSize)))
    }
  }, [busy, total, rows.length, page, pageSize])

  const pickStatus = (v: string) => { setStatus(v); setPage(1) }
  const pickType = (v: string) => { setTypeF(v); setPage(1) }
  const pickSize = (n: number) => { setPageSize(n); setPage(1) }
  const statusOptions = [{ value: ALL, label: a.filterAll }].concat(
    SCAN_STATUSES.map((s) => ({ value: s, label: statusTagLabel('asset', s, t.common.statusTags) })))
  // 类型筛选=P4-T2 白名单(权威码直出);服务端 type 精确匹配(ListAssetsPage)。
  const typeOptions = [{ value: ALL, label: a.filterAllType }].concat(
    ASSET_TYPES.map((v) => ({ value: v, label: v })))

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={a.searchPlaceholder}
              value={keyword} onChange={(e) => setKeyword(e.target.value)} />
            <Dropdown value={status} ariaLabel={a.dStatus} onChange={pickStatus} options={statusOptions} />
            <Dropdown value={typeF} ariaLabel={a.fType} onChange={pickType} options={typeOptions} />
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
            <ToolbarButton onClick={() => setModelsOpen(true)}>{a.modelsManage}</ToolbarButton>
            <ToolbarButton onClick={() => setBatchesOpen(true)}>{a.batchesManage}</ToolbarButton>
            <ToolbarButton primary onClick={() => setCreateOpen(true)}>{a.create}</ToolbarButton>
          </div>
        </CardContent>
        {error && <ErrorBanner message={error} />}
        <AssetTable rows={rows} tagOf={tagOf} batchOf={batchOf} busy={busy}
          onTrail={setTrail} onDetail={setDetail} onEdit={setEditRow} onScrap={setScrapRow} onDelete={delRow} />
        <CardFooter>
          <Pagination total={total} page={page} pageSize={pageSize}
            onPage={setPage} onSize={pickSize} {...pagerTexts(a)} />
        </CardFooter>
      </Card>
      {trail && <AssetTrailDrawer asset={trail} tag={tagOf(trail.tagId)} onClose={() => setTrail(null)} />}
      {detail && <AssetTrailDrawer asset={detail} tag={tagOf(detail.tagId)} showMain onClose={() => setDetail(null)} />}
      {createOpen && <CreateDrawer onClose={() => setCreateOpen(false)} onSaved={load} />}
      {editRow && <EditDrawer asset={editRow} onClose={() => setEditRow(null)} onSaved={load} />}
      {scrapRow && <ScrapDialog asset={scrapRow} tag={tagOf(scrapRow.tagId)} onClose={() => setScrapRow(null)} onSaved={load} />}
      {modelsOpen && <ModelDictDrawer onClose={() => setModelsOpen(false)} onSaved={load} />}
      {batchesOpen && <BatchDrawer onClose={() => setBatchesOpen(false)} onSaved={load} />}
    </div>
  )
}