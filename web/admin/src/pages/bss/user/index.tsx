// 用户列表页:GET /users(keyword 过滤)+ GET /users/{customerId} 14 段详情聚合抽屉。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { filterUsers, pageSlice, type UserRow } from './filter'
import './user.css'

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
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={u.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{u.columns.map((c) => <th key={c}>{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.customerId}>
                    <td>{r.customerId}</td>
                    <td>{r.name}</td>
                    <td>{r.phone}</td>
                    <td>{r.loginName || '—'}</td>
                    <td>{r.planName || '—'}</td>
                    <td>{fmtTime(r.createdAt)}</td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => openDetail(r.customerId)}>{u.detail}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {slice.length === 0 && <tr><td colSpan={u.columns.length}>{u.empty}</td></tr>}
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
          <div className="user-detail">
            {sections.map((k) => {
              const arr = detail[k] as Record<string, unknown>[]
              return (
                <details key={k} open={sections.indexOf(k) < 4}>
                  <summary>{u.sectionNames[k] ?? k}({arr.length})</summary>
                  {arr.length === 0 ? <p className="user-muted">{u.empty}</p> : (
                    arr.slice(0, 20).map((item, i) => (
                      <div key={i} className="user-kv">
                        {Object.entries(item).map(([fk, fv]) => (
                          <span key={fk}><b>{fk}</b>: {String(fv ?? '—')}</span>
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
