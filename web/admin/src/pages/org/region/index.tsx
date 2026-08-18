// 经营区域页:四级树(level 1集团 2大区 3省 4城市),列名以 fields.md 1.3 为准。
// 契约: GET /regions(org.yaml;parentPath 前缀过滤=下钻)。下级数由全量列表派生。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../shared'
import { buildRegionView, filterRegions, pageSlice, type RegionRow } from './tree'
import { Pagination } from '../../../components/Pagination'
import '../org.css'

export default function RegionPage() {
  const t = useT()
  const [rows, setRows] = useState<RegionRow[]>([])
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

  const view = useMemo(
    () => filterRegions(buildRegionView(rows, t.pages.region.levelNames), keyword, level, drillPath),
    [rows, keyword, level, drillPath, t],
  )
  const slice = pageSlice(view, page, pageSize)

  return (
    <div>
      <PageHead title={t.pages.region.title} desc={t.pages.region.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={t.pages.region.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <select className="org-select" value={level} onChange={(e) => { setLevel(e.target.value); setPage(1) }}>
            <option value="">{t.pages.region.allLevel}</option>
            {t.pages.region.levelNames.map((n, i) => <option key={n} value={String(i + 1)}>{n}</option>)}
          </select>
          {drillPath && (
            <button className="org-btn" onClick={() => { setDrillPath(''); setPage(1) }}>
              {t.pages.region.drill}: {drillPath} ×
            </button>
          )}
          <span className="spacer" />
          <button className="org-btn" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{t.pages.region.columns.map((c) => <th key={c}>{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.path}>
                    <td>{r.path}</td>
                    <td>{r.name}</td>
                    <td>{r.levelName}</td>
                    <td>{r.parent || '—'}</td>
                    <td>{r.childCount}</td>
                    <td>
                      {r.childCount > 0 && (
                        <span className="org-act">
                          <button onClick={() => { setDrillPath(r.path); setPage(1) }}>{t.pages.region.drill}</button>
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{t.pages.region.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={view.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.region)} />
        </div>
      </div>
    </div>
  )
}
