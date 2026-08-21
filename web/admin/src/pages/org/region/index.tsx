// 经营区域页:四级树(level 1集团 2大区 3省 4城市),列名以 fields.md 1.3 为准。
// 契约: GET /regions(org.yaml;parentPath 前缀过滤=下钻)。下级数由全量列表派生。
// 覆盖主体列:PUT /regions/:id/coverage 划分子公司经营区域(0=摘除,兜底总公司)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { PageHead, pagerTexts } from '../shared'
import { buildRegionView, filterRegions, pageSlice, type RegionRow } from './tree'
import { Pagination } from '../../../components/Pagination'

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

  const load = () => {
    setError('')
    apiFetch<RegionRow[]>('/regions')
      .then((d) => setRows(d ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.region.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    apiFetch<EntityOption[]>('/legal-entities').then((d) => setEntities(d ?? [])).catch(() => {})
  }, [])

  // assignCoverage 划分/摘除覆盖主体后整表刷新。
  const assignCoverage = (regionId: number, legalEntityId: number) => {
    apiFetch('/regions/' + regionId + '/coverage', {
      method: 'PUT',
      body: { legalEntityId },
    })
      .then(load)
      .catch(() => setError(t.pages.region.assignFail))
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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={t.pages.region.searchPlaceholder}
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
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{t.pages.region.columns.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.path}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.path}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.levelName}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.parent || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.childCount}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <Dropdown
                        value={r.legalEntityId ? String(r.legalEntityId) : ''}
                        options={coverageOptions}
                        onChange={(v) => assignCoverage(r.id, Number(v) || 0)}
                        ariaLabel={t.pages.region.coverageNone}
                      />
                    </td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {r.childCount > 0 && (
                        <span className="inline-flex items-center">
                          <button onClick={() => { setDrillPath(r.path); setPage(1) }}>{t.pages.region.drill}</button>
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{t.pages.region.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={view.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.region)} />
        </div>
      </div>
    </div>
  )
}
