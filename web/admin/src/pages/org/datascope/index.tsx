// 数据权限页:账号级数据范围清单。列名对齐原型(账号/角色(功能)/子公司/部门/岗位/数据范围/操作)。
// 数据源: GET /data-scopes(后端已实现,keyword 服务端过滤账号/姓名/角色);空 keyword=全量。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterDataScopes, pageSlice, type ScopeRow } from './filter'
import { Pagination } from '../../../components/Pagination'
import '../org.css'

export default function DataScopePage() {
  const t = useT()
  const [rows, setRows] = useState<ScopeRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<ScopeRow | null>(null)

  const load = () => {
    setError('')
    apiFetch<ScopeRow[]>('/data-scopes', { query: { keyword: keyword || undefined } })
      .then((d) => setRows(d ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.datascope.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => filterDataScopes(rows, keyword), [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)
  const scopeText = (r: ScopeRow) => r.regionScope || t.pages.datascope.allScope

  return (
    <div>
      <PageHead title={t.pages.datascope.title} desc={t.pages.datascope.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={t.pages.datascope.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{t.pages.datascope.columns.map((c) => <th key={c}>{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{r.username}</td>
                    <td>{r.roleName}</td>
                    <td>{r.legalEntityName || '—'}</td>
                    <td>{r.deptName || '—'}</td>
                    <td>{r.postName || '—'}</td>
                    <td>{scopeText(r)}</td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => setDetail(r)}>{t.pages.datascope.detail}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{t.pages.datascope.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.datascope)} />
        </div>
      </div>
      {detail && (
        <DetailDrawer
          title={t.pages.datascope.detail}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: t.pages.datascope.columns[0], v: detail.username },
            { k: t.pages.datascope.columns[1], v: detail.roleName },
            { k: t.pages.datascope.columns[2], v: detail.legalEntityName },
            { k: t.pages.datascope.columns[3], v: detail.deptName },
            { k: t.pages.datascope.columns[4], v: detail.postName },
            { k: t.pages.datascope.columns[5], v: scopeText(detail) },
          ]}
        />
      )}
    </div>
  )
}
