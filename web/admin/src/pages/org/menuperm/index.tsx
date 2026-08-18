// 菜单权限页:三层权限模型横幅 + 角色×菜单矩阵(勾选=角色持该 menu:* 权限)。
// 契约: GET /menu-perms(org.yaml;data:{model:{layers[string]},matrix:{roleColumns,rows}})。
// 原型的"调整可见性/保存"为演示交互,无后端操作,不实现(契约只读)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterMatrixRows, pageSlice, type MenuPermData, type MenuPermViewRow } from './matrix'
import { Pagination } from '../../../components/Pagination'
import '../org.css'

export default function MenuPermPage() {
  const t = useT()
  const [data, setData] = useState<MenuPermData>({ layers: [], roleColumns: [], rows: [] })
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [role, setRole] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<MenuPermViewRow | null>(null)

  const load = () => {
    setError('')
    apiFetch<{ model: { layers: string[] }; matrix: MenuPermData }>('/menu-perms')
      .then((d) => setData({
        layers: d?.model?.layers ?? [],
        roleColumns: d?.matrix?.roleColumns ?? [],
        rows: d?.matrix?.rows ?? [],
      }))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.menuperm.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(
    () => filterMatrixRows(data.rows, keyword, role),
    [data.rows, keyword, role],
  )
  const slice = pageSlice(filtered, page, pageSize)
  const roleName = (code: string) =>
    data.roleColumns.find((c) => c.roleCode === code)?.roleName ?? code

  return (
    <div>
      <PageHead title={t.pages.menuperm.title} desc={t.pages.menuperm.desc} />
      <div className="org-card" style={{ marginBottom: 16 }}>
        <div className="org-card-title">{t.pages.menuperm.modelTitle}</div>
        <div className="org-table-wrap">
          <table className="org-table">
            <thead><tr><th>{t.pages.menuperm.modelLayerLabel}</th></tr></thead>
            <tbody>
              {(data.layers ?? []).map((l, i) => (
                <tr key={l}><td>{i + 1}. {l}</td></tr>
              ))}
              {!(data.layers ?? []).length && <tr><td><div className="org-empty">{t.pages.menuperm.empty}</div></td></tr>}
              {error && <tr><td className="org-error" style={{ margin: 0 }}>{error}</td></tr>}
            </tbody>
          </table>
        </div>
      </div>
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={t.pages.menuperm.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <select className="org-select" value={role} onChange={(e) => { setRole(e.target.value); setPage(1) }}>
            <option value="">{t.pages.menuperm.allRole}</option>
            {data.roleColumns.map((c) => (
              <option key={c.roleCode} value={c.roleCode}>{c.roleName}</option>
            ))}
          </select>
          <span className="spacer" />
          <button className="org-btn" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {!error && (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead>
                <tr>
                  <th>{t.pages.menuperm.menuColumn}</th>
                  {data.roleColumns.map((c) => <th key={c.roleCode}>{c.roleName}</th>)}
                  <th>{t.pages.menuperm.actionColumn}</th>
                </tr>
              </thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.code}>
                    <td>{r.name}</td>
                    {data.roleColumns.map((c) => (
                      <td key={c.roleCode}>{r.roles.includes(c.roleCode) ? '✓' : ''}</td>
                    ))}
                    <td>
                      <span className="org-act">
                        <button onClick={() => setDetail(r)}>{t.pages.menuperm.detailTitle}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && (
                  <tr><td colSpan={data.roleColumns.length + 2}><div className="org-empty">{t.pages.menuperm.empty}</div></td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.menuperm)} />
        </div>
      </div>
      {detail && (
        <DetailDrawer
          title={t.pages.menuperm.detailTitle}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: t.pages.menuperm.menuColumn, v: detail.name },
            { k: t.pages.menuperm.visibleRoles, v: detail.roles.map(roleName).join('、') || t.pages.menuperm.noneRole },
            { k: 'Code', v: detail.code },
          ]}
        />
      )}
    </div>
  )
}
