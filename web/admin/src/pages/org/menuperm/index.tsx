// 菜单权限页:三层权限模型横幅 + 角色×菜单矩阵(勾选=角色持该 menu:* 权限)。
// 契约: GET /menu-perms(org.yaml;data:{model:{layers[string]},matrix:{roleColumns,rows}})。
// 原型的"调整可见性/保存"为演示交互,无后端操作,不实现(契约只读)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterMatrixRows, pageSlice, type MenuPermData, type MenuPermViewRow } from './matrix'
import { Pagination } from '../../../components/Pagination'

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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]" style={{ marginBottom: 16 }}>
        <div className="px-4 pt-3.5 text-[15px] font-semibold text-[var(--shell-heading)]">{t.pages.menuperm.modelTitle}</div>
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr><th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.menuperm.modelLayerLabel}</th></tr></thead>
            <tbody>
              {(data.layers ?? []).map((l, i) => (
                <tr key={l}><td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{i + 1}. {l}</td></tr>
              ))}
              {!(data.layers ?? []).length && <tr><td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{t.pages.menuperm.empty}</div></td></tr>}
              {error && <tr><td className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" style={{ margin: 0 }}>{error}</td></tr>}
            </tbody>
          </table>
        </div>
      </div>
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={t.pages.menuperm.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <Dropdown
            value={role}
            options={[{ value: '', label: t.pages.menuperm.allRole }, ...data.roleColumns.map((c) => ({ value: c.roleCode, label: c.roleName }))]}
            onChange={(v) => { setRole(v); setPage(1) }}
            ariaLabel={t.pages.menuperm.allRole}
          />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {!error && (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">
                <tr>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.menuperm.menuColumn}</th>
                  {data.roleColumns.map((c) => <th key={c.roleCode} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c.roleName}</th>)}
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.menuperm.actionColumn}</th>
                </tr>
              </thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.code}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                    {data.roleColumns.map((c) => (
                      <td key={c.roleCode} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.roles.includes(c.roleCode) ? '✓' : ''}</td>
                    ))}
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => setDetail(r)}>{t.pages.menuperm.detailTitle}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && (
                  <tr><td colSpan={data.roleColumns.length + 2} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{t.pages.menuperm.empty}</div></td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
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
