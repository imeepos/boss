// 用户列表页:GET /users(keyword 过滤)+ GET /users/{customerId} 14 段详情聚合抽屉。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { filterUsers, pageSlice, type UserRow } from './filter'

export default function UserListPage() {
  const t = useT()
  const u = t.pages.userPage
  const [rows, setRows] = useState<UserRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<Record<string, unknown> | null>(null)
  const [detailId, setDetailId] = useState(0)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: UserRow[] }>('/users', { query: { keyword: keyword || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : u.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openDetail = (id: number) => {
    setDetailId(id)
    apiFetch<Record<string, unknown>>(`/users/${id}`)
      .then((d) => setDetail(d))
      .catch((e) => setError(e instanceof Error ? e.message : u.loadFail))
  }

  const filtered = useMemo(() => filterUsers(rows, keyword), [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)
  const sections = detail ? (Object.keys(detail).filter((k) => Array.isArray(detail[k]))) : []

  return (
    <div>
      <PageHead title={u.title} desc={u.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={u.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{u.columns.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.customerId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.customerId}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.phone}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.loginName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.planName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(r.createdAt)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => openDetail(r.customerId)}>{u.detail}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {slice.length === 0 && <tr><td colSpan={u.columns.length} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{u.empty}</td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <Pagination total={filtered.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize}
          rangeText={u.rangeText} prevText={u.prev} nextText={u.next}
          perPageText={u.perPage} jumpText={u.jumpText} pageUnitText={u.pageUnit} />
      </div>
      {detail && (
      <Drawer title={`${u.detailTitle} #${detailId}`} onClose={() => setDetail(null)}>
        {detail && (
          <div>
            {sections.map((k) => {
              const arr = detail[k] as Record<string, unknown>[]
              return (
                <details key={k} open={sections.indexOf(k) < 4}
                  className="border-b border-dashed border-border py-1.5">
                  <summary className="cursor-pointer font-medium">{u.sectionNames[k] ?? k}({arr.length})</summary>
                  {arr.length === 0 ? <p className="pl-3 text-[var(--shell-crumb-text)]">{u.empty}</p> : (
                    arr.slice(0, 20).map((item, i) => (
                      <div key={i} className="flex flex-wrap gap-x-3.5 gap-y-1 py-1 pl-3 text-xs">
                        {Object.entries(item).map(([fk, fv]) => (
                          <span key={fk}><b className="mr-0.5 font-medium text-[var(--shell-crumb-text)]">{fk}</b>: {String(fv ?? '—')}</span>
                        ))}
                      </div>
                    ))
                  )}
                </details>
              )
            })}
          </div>
        )}
      </Drawer>
      )}
    </div>
  )
}
