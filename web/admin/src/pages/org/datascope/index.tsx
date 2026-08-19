// 数据权限页:账号级数据范围清单。列名对齐原型(账号/角色(功能)/子公司/部门/岗位/数据范围/操作)。
// 数据源: GET /data-scopes(后端已实现,keyword 服务端过滤账号/姓名/角色);空 keyword=全量。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterDataScopes, pageSlice, type ScopeRow } from './filter'
import { Pagination } from '../../../components/Pagination'

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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={t.pages.datascope.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{t.pages.datascope.columns.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.username}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.roleName}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.legalEntityName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.deptName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.postName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{scopeText(r)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => setDetail(r)}>{t.pages.datascope.detail}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{t.pages.datascope.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
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
