// 右侧成员面板:选中节点的成员列表(账号/姓名/角色/岗位/状态)+ 添加/编辑/启停操作。
import { useMemo, useState } from 'react'
import { useT } from '../../../i18n'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
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
  const td = 'h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'
  const act = 'cursor-pointer border-none bg-none px-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline'

  return (
    <div className="min-w-0 flex-1 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
      <div className="flex flex-wrap items-center gap-2 p-4">
        <span className="mr-1 text-sm font-semibold text-[var(--shell-heading)]">{title}</span>
        <input
          className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
          placeholder={t.pages.staff.searchPlaceholder}
          value={kw}
          onChange={(e) => { setKw(e.target.value); setPage(1) }}
        />
        <span className="flex-1" />
        <button
          className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
          onClick={onAdd}
        >
          {t.pages.staff.addMember}
        </button>
      </div>
      <div className="overflow-x-auto px-4 pb-4">
        <table className="w-full border-collapse text-[13px]">
          <thead>
            <tr>
              {t.pages.staff.columns.map((c) => (
                <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {slice.map((r) => (
              <tr key={r.id}>
                <td className={td}>{r.username}</td>
                <td className={td}>{r.realName}</td>
                <td className={td}>{r.roleName}</td>
                <td className={td}>{r.postName || '—'}</td>
                <td className={td}><StatusTag domain="accountStatus" value={String(r.status)} /></td>
                <td className={td}>
                  <span className="inline-flex items-center gap-1.5">
                    <button className={act} onClick={() => onEdit(r)}>{t.pages.staff.edit}</button>
                    <span className="text-[var(--shell-side-border)]">|</span>
                    <button className={act} onClick={() => onToggle(r)}>
                      {r.status === 1 ? t.pages.account.disable : t.pages.geo.enable}
                    </button>
                  </span>
                </td>
              </tr>
            ))}
            {!slice.length && <TableStateRow colSpan={t.pages.staff.columns.length} loading={busy} text={t.pages.staff.empty} />}
          </tbody>
        </table>
      </div>
      <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
        <Pagination
          total={filtered.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }}
          rangeText={t.pages.company.rangeText} prevText={t.pages.company.prev}
          nextText={t.pages.company.next} perPageText={t.pages.company.perPage}
          jumpText={t.pages.company.jumpText} pageUnitText={t.pages.company.pageUnit}
        />
      </div>
    </div>
  )
}
