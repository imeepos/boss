// 菜单权限页:角色管理卡片(自定义角色,迁移 000100)+ 三层权限模型横幅 + 角色×菜单矩阵。
// 契约: GET /role-details、GET /permissions、POST/PUT/DELETE /roles(org.yaml)、GET /menu-perms。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterMatrixRows, pageSlice, type MenuPermData, type MenuPermViewRow } from './matrix'
import { Pagination } from '../../../components/Pagination'
import { EmptyState } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { Input } from '../../../components/ui/input'
import { RoleManagerCard } from './RoleManagerCard'

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
      <RoleManagerCard onChanged={load} />
      <Card>
        <div className="px-4 pt-3.5 text-[15px] font-semibold text-[var(--shell-heading)]">{t.pages.menuperm.modelTitle}</div>
        {error && <div className="px-4 pb-2"><ErrorBanner message={error} className="!mx-0 !mb-0" /></div>}
        <Table>
          <TableHeader>
            <TableRow><TableHead>{t.pages.menuperm.modelLayerLabel}</TableHead></TableRow>
          </TableHeader>
          <TableBody>
            {(data.layers ?? []).map((l, i) => (
              <TableRow key={l}><TableCell>{i + 1}. {l}</TableCell></TableRow>
            ))}
            {!(data.layers ?? []).length && (
              <TableRow><TableCell><EmptyState text={t.pages.menuperm.empty} /></TableCell></TableRow>
            )}
          </TableBody>
        </Table>
        <div className="pb-3" />
      </Card>
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-56" placeholder={t.pages.menuperm.searchPlaceholder}
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
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t.pages.menuperm.menuColumn}</TableHead>
                {data.roleColumns.map((c) => <TableHead key={c.roleCode}>{c.roleName}</TableHead>)}
                <TableHead>{t.pages.menuperm.actionColumn}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.code}>
                  <TableCell>{r.name}</TableCell>
                  {data.roleColumns.map((c) => (
                    <TableCell key={c.roleCode}>{r.roles.includes(c.roleCode) ? '✓' : ''}</TableCell>
                  ))}
                  <TableCell>
                    <span className="inline-flex items-center">
                      <button className="cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => setDetail(r)}>{t.pages.menuperm.detailTitle}</button>
                    </span>
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && (
                <TableRow><TableCell colSpan={data.roleColumns.length + 2}><EmptyState text={t.pages.menuperm.empty} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.menuperm)} />
        </div>
      </Card>
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
