// 右侧成员面板:选中节点的成员列表(账号/姓名/角色/岗位/状态)+ 添加/编辑/启停操作。
import { useMemo, useState } from 'react'
import { useT } from '../../../i18n'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { EmptyState } from '../../../components/business/feedback'
import { pagerTexts } from '../shared'
import type { AccountRow } from '../../base/account/list'

interface MemberPanelProps {
  title: string
  members: AccountRow[]
  busy: boolean
  onAdd: () => void
  onEdit: (r: AccountRow) => void
  onToggle: (r: AccountRow) => void
}

export function MemberPanel({ title, members, busy, onAdd, onEdit, onToggle }: MemberPanelProps) {
  const t = useT()
  const [kw, setKw] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const filtered = useMemo(() => {
    const q = kw.trim().toLowerCase()
    if (!q) return members
    return members.filter((r) => r.username.toLowerCase().includes(q) || r.realName.toLowerCase().includes(q))
  }, [members, kw])
  const slice = filtered.slice((page - 1) * pageSize, page * pageSize)
  const act = 'cursor-pointer border-none bg-none px-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline'
  const pager = pagerTexts(t.pages.company)
  const isEmpty = !slice.length

  return (
    <Card className="flex min-w-0 flex-1 flex-col">
      <div className="flex flex-wrap items-center gap-2 p-4">
        <span className="mr-1 text-sm font-semibold text-[var(--shell-heading)]">{title}</span>
        <Input
          className="w-56"
          placeholder={t.pages.staff.searchPlaceholder}
          value={kw}
          onChange={(e) => { setKw(e.target.value); setPage(1) }}
        />
        <span className="flex-1" />
        <button
          className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed disabled:opacity-70"
          onClick={onAdd} disabled={busy}
        >
          {t.pages.staff.addMember}
        </button>
      </div>
      {isEmpty
        ? <div className="flex flex-1 items-center justify-center px-4 pb-10 pt-4"><EmptyState text={t.pages.staff.empty} /></div>
        : (
          <Table>
            <TableHeader>
              <TableRow>
                {t.pages.staff.columns.map((c) => (
                  <TableHead key={c}>{c}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>{r.username}</TableCell>
                  <TableCell>{r.realName}</TableCell>
                  <TableCell>{r.roleName}</TableCell>
                  <TableCell>{r.postName || '—'}</TableCell>
                  <TableCell><StatusTag domain="accountStatus" value={String(r.status)} /></TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-1.5">
                      <button className={act} onClick={() => onEdit(r)}>{t.pages.staff.edit}</button>
                      <span className="text-[var(--shell-side-border)]">|</span>
                      <button className={act} onClick={() => onToggle(r)}>
                        {r.status === 1 ? t.pages.account.disable : t.pages.geo.enable}
                      </button>
                    </span>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
        <Pagination
          total={filtered.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }}
          {...pager}
        />
      </div>
    </Card>
  )
}
