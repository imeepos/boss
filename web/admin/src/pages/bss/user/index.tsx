// 用户列表页:GET /users(keyword 过滤)+ 详情抽屉(GET /users/{customerId} 14 段聚合,见 detail-drawer)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { PageHead } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { ActionLink, ErrorBanner, ToolbarButton } from '../../../components/business'
import { DataTable } from '../../../components/business/data-table'
import { filterUsers, pageSlice, createdAtCell, loginNameCell, type UserRow } from './filter'
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
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-48" placeholder={u.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setUrlKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <DataTable
              emptyText={u.empty}
              rows={slice as unknown as Record<string, unknown>[]}
              columns={[
                { key: 'customerId', label: u.columns[0], render: (r) => String(r.customerId) },
                { key: 'name', label: u.columns[1], render: (r) => String(r.name ?? '') },
                { key: 'phone', label: u.columns[2], render: (r) => String(r.phone ?? '') },
                { key: 'loginName', label: u.columns[3], render: (r) => loginNameCell(r as UserRow) },
                { key: 'planName', label: u.columns[4], render: (r) => String(r.planName || '—') },
                { key: 'createdAt', label: u.columns[5], render: (r) => createdAtCell(r as UserRow) },
                { key: 'op', label: u.columns[6], render: (r) => (
                  <ActionLink onClick={() => setDetailId(Number(r.customerId))} label={u.detail} testId={'user-detail-' + r.customerId} />
                ) },
              ]}
            />
          </div>
        )}
        <Pagination total={filtered.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize}
          rangeText={u.rangeText} prevText={u.prev} nextText={u.next}
          perPageText={u.perPage} jumpText={u.jumpText} pageUnitText={u.pageUnit} />
      </Card>
      {detailId !== null && <UserDetailDrawer id={detailId} summary={activeRow} onClose={() => setDetailId(null)} />}
    </div>
  )
}
