// 用户列表页:GET /users(keyword 过滤)+ 详情抽屉(GET /users/{customerId} 14 段聚合,见 detail-drawer)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { PageHead } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { DataTable } from '../../../components/business/data-table'
import { fmtTime } from '../../../lib/format'
import { filterUsers, pageSlice, type UserRow } from './filter'
import { UserDetailDrawer } from './detail-drawer'

export default function UserListPage() {
  const t = useT()
  const u = t.pages.userPage
  const [rows, setRows] = useState<UserRow[]>([])
  const [error, setError] = useState('')
  const [urlKeyword, setUrlKeyword] = useQueryState('kw', '')
  const [keyword, setKeyword] = useState(urlKeyword)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detailId, setDetailId] = useState<number | null>(null)
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

  const filtered = useMemo(() => filterUsers(rows, keyword), [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)
  const activeRow = detailId !== null ? rows.find((r) => r.customerId === detailId) : undefined

  return (
    <div>
      <PageHead title={u.title} desc={u.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={u.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setUrlKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="px-4 pb-4">
            <DataTable
              emptyText={u.empty}
              rows={slice as unknown as Record<string, unknown>[]}
              columns={[
                { key: 'customerId', label: u.columns[0], render: (r) => String(r.customerId) },
                { key: 'name', label: u.columns[1], render: (r) => String(r.name ?? '') },
                { key: 'phone', label: u.columns[2], render: (r) => String(r.phone ?? '') },
                { key: 'loginName', label: u.columns[3], render: (r) => String(r.loginName || '—') },
                { key: 'planName', label: u.columns[4], render: (r) => String(r.planName || '—') },
                { key: 'createdAt', label: u.columns[5], render: (r) => fmtTime(String(r.createdAt)) },
                { key: 'op', label: u.columns[6], render: (r) => (
                  <button onClick={() => setDetailId(Number(r.customerId))}>{u.detail}</button>
                ) },
              ]}
            />
          </div>
        )}
        <Pagination total={filtered.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize}
          rangeText={u.rangeText} prevText={u.prev} nextText={u.next}
          perPageText={u.perPage} jumpText={u.jumpText} pageUnitText={u.pageUnit} />
      </div>
      {detailId !== null && <UserDetailDrawer id={detailId} summary={activeRow} onClose={() => setDetailId(null)} />}
    </div>
  )
}
