// 经营区域页:四级树(level 1集团 2大区 3省 4城市),列名以 fields.md 1.3 为准。
// 契约: GET /regions(org.yaml;parentPath 前缀过滤=下钻)。下级数由全量列表派生。
// 覆盖主体列:PUT /regions/:id/coverage 划分子公司经营区域(0=摘除,兜底总公司)。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'
import { ErrorBanner } from '../../../components/business/page-head'
import { PageHead, pagerTexts } from '../shared'
import { buildRegionView, filterRegions, pageSlice, type RegionRow } from './tree'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { Input } from '../../../components/ui/input'

interface EntityOption { id: number; name: string; isPlatform: boolean }

export default function RegionPage() {
  const t = useT()
  const [rows, setRows] = useState<RegionRow[]>([])
  const [entities, setEntities] = useState<EntityOption[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [level, setLevel] = useState('')
  const [drillPath, setDrillPath] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const confirmDialog = useConfirm()

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<RegionRow[]>('/regions')
      .then((d) => setRows(d ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.region.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    apiFetch<EntityOption[]>('/legal-entities').then((d) => setEntities(d ?? [])).catch(() => {})
  }, [])

  // assignCoverage 划分/摘除覆盖主体:先确认(资产/客户归属随动),成功 toast,失败可复制。
  const assignCoverage = async (regionId: number, legalEntityId: number) => {
    if (!(await confirmDialog(t.pages.region.assignConfirm))) return
    setBusy(true)
    try {
      await apiFetch('/regions/' + regionId + '/coverage', {
        method: 'PUT',
        body: { legalEntityId },
      })
      toast.success(t.pages.staff.statusOk)
      load()
    } catch (e) {
      const msg = e instanceof Error ? e.message : t.pages.region.assignFail
      setError(msg)
      toast.error(msg)
    } finally {
      setBusy(false)
    }
  }

  const coverageOptions = useMemo(
    () => [
      { value: '', label: t.pages.region.coverageNone },
      ...entities.map((e) => ({
        value: String(e.id),
        label: e.name + (e.isPlatform ? '（平台）' : ''),
      })),
    ],
    [entities, t],
  )

  const view = useMemo(
    () => filterRegions(buildRegionView(rows, t.pages.region.levelNames), keyword, level, drillPath),
    [rows, keyword, level, drillPath, t],
  )
  const slice = pageSlice(view, page, pageSize)

  return (
    <div>
      <PageHead title={t.pages.region.title} desc={t.pages.region.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-56" placeholder={t.pages.region.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <Dropdown
            value={level}
            options={[{ value: '', label: t.pages.region.allLevel }, ...t.pages.region.levelNames.map((n, i) => ({ value: String(i + 1), label: n }))]}
            onChange={(v) => { setLevel(v); setPage(1) }}
            ariaLabel={t.pages.region.allLevel}
          />
          {drillPath && (
            <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => { setDrillPath(''); setPage(1) }}>
              {t.pages.region.drill}: {drillPath} ×
            </button>
          )}
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-50" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error && <ErrorBanner message={error} />}
        <Table>
          <TableHeader>
            <TableRow>
              {t.pages.region.columns.map((c) => <TableHead key={c}>{c}</TableHead>)}
            </TableRow>
          </TableHeader>
          <TableBody>
            {slice.map((r) => (
              <TableRow key={r.path}>
                <TableCell>{r.path}</TableCell>
                <TableCell>{r.name}</TableCell>
                <TableCell>{r.levelName}</TableCell>
                <TableCell>{r.parent || '—'}</TableCell>
                <TableCell>{r.childCount}</TableCell>
                <TableCell>
                  <Dropdown
                    value={r.legalEntityId ? String(r.legalEntityId) : ''}
                    options={coverageOptions}
                    onChange={(v) => assignCoverage(r.id, Number(v) || 0)}
                    ariaLabel={t.pages.region.coverageNone}
                  />
                </TableCell>
                <TableCell>
                  {r.childCount > 0 && (
                    <span className="inline-flex items-center">
                      <button className="cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => { setDrillPath(r.path); setPage(1) }}>{t.pages.region.drill}</button>
                    </span>
                  )}
                </TableCell>
              </TableRow>
            ))}
            {!slice.length && <TableStateRow colSpan={7} loading={busy} text={t.pages.region.empty} />}
          </TableBody>
        </Table>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={view.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.region)} />
        </div>
      </Card>
    </div>
  )
}
