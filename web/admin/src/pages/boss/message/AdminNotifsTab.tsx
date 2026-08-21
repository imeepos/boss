// 消息中心 · 后台提醒页签:GET /notifications(服务端分页)+ 已读/去处理。
// 状态经 URL 持久(useQueryState),刷新后筛选条件不变(与 geo 页同款约定)。
import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiFetch } from '../../../api/client'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { StatusTag } from '../../../components/StatusTag'
import { useT } from '../../../i18n'
import { fmtTime } from '../../../lib/format'
import { useQueryInt, useQueryState } from '../../../lib/useQueryState'
import { markNotificationsRead } from '../../../lib/useNotifications'

export interface NotifItem {
  id: number
  category: string
  level: string
  title: string
  content: string
  link: string
  resolved: boolean
  createdAt: string
  read: boolean
}

const CARD = 'rounded-lg border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4 shadow-[var(--shell-card-shadow)]'
const CTL = 'h-[30px] rounded-md border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] px-2 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]'
const TH = 'border-b border-[var(--shell-side-border)] px-2.5 py-2 text-left font-semibold'
const TD = 'border-b border-[var(--shell-side-border)] px-2.5 py-2'

export function AdminNotifsTab() {
  const t = useT()
  const n = useT().pages.message.notif
  const nav = useNavigate()
  const [catUrl, setCatUrl] = useQueryState('cat', '')
  const [levelUrl, setLevelUrl] = useQueryState('level', '')
  const [unreadUrl, setUnreadUrl] = useQueryState('unread', '')
  const [page, setPage] = useQueryInt('page', 1)
  const [pageSize, setPageSize] = useQueryInt('size', 10)
  const [rows, setRows] = useState<NotifItem[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    setError('')
    apiFetch<{ items: NotifItem[]; total: number }>('/notifications', {
      query: {
        category: catUrl || undefined,
        level: levelUrl || undefined,
        unread: unreadUrl === '1' ? 1 : undefined,
        limit: pageSize,
        offset: (page - 1) * pageSize,
      },
    })
      .then((d) => { setRows(d?.items ?? []); setTotal(d?.total ?? 0) })
      .catch(() => setError(n.loadFail))
  }, [catUrl, levelUrl, unreadUrl, page, pageSize, n.loadFail])

  useEffect(() => { void load() }, [load])

  const markAll = () => {
    void markNotificationsRead([]).then(load)
  }

  const open = (r: NotifItem) => {
    if (!r.read) void markNotificationsRead([r.id]).then(load)
    if (r.link) nav(r.link)
  }

  return (
    <div className={CARD}>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Dropdown
          value={catUrl}
          options={[
            { value: '', label: n.allCategories },
            { value: 'todo', label: n.categoryTodo },
            { value: 'task', label: n.categoryTask },
          ]}
          onChange={(v) => { setCatUrl(v); setPage(1) }}
          ariaLabel={n.allCategories}
        />
        <Dropdown
          value={levelUrl}
          options={[
            { value: '', label: n.allLevels },
            { value: 'INFO', label: n.levelInfo },
            { value: 'WARN', label: n.levelWarn },
            { value: 'URGENT', label: n.levelUrgent },
          ]}
          onChange={(v) => { setLevelUrl(v); setPage(1) }}
          ariaLabel={n.allLevels}
        />
        <Dropdown
          value={unreadUrl}
          options={[{ value: '', label: n.allLevels }, { value: '1', label: n.unreadOnly }]}
          onChange={(v) => { setUnreadUrl(v); setPage(1) }}
          ariaLabel={n.unreadOnly}
        />
        <span className="flex-1" />
        <button className={`${CTL} cursor-pointer px-3`} onClick={() => void load()}>{n.refresh}</button>
        <button className={`${CTL} cursor-pointer px-3`} onClick={markAll}>{n.markAllRead}</button>
      </div>
      <div className="mb-3 font-semibold text-[var(--shell-heading)]">
        {n.cardTitle}
        <span className="ml-2 text-xs font-normal text-[var(--shell-group-title)]">
          {total ? n.rangeText.replace('{from}', String((page - 1) * pageSize + 1)).replace('{to}', String(Math.min(page * pageSize, total))).replace('{count}', String(total)) : ''}
        </span>
      </div>
      {error ? <div className="py-3 text-[13px] text-[var(--color-danger)]">{error}</div> : (
        <>
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead>
              <tr>{n.columns.map((c) => <th key={c} className={TH}>{c}</th>)}</tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} className={r.resolved ? 'opacity-60' : ''}>
                  <td className={TD}><StatusTag domain="message" value={r.level} /></td>
                  <td className={`${TD} ${r.read ? '' : 'font-semibold text-[var(--shell-heading)]'}`}>{r.title}</td>
                  <td className={TD}>{r.category === 'todo' ? n.categoryTodo : n.categoryTask}</td>
                  <td className={TD}>{fmtTime(r.createdAt)}</td>
                  <td className={TD}>{r.resolved ? n.resolved : (r.read ? t.pages.message.read : t.pages.message.unread)}</td>
                  <td className={TD}>
                    {r.link && (
                      <button className="cursor-pointer border-0 bg-none p-0 text-[13px] text-[var(--color-brand-gold-500)] hover:underline" onClick={() => open(r)}>
                        {r.category === 'todo' && !r.resolved ? n.goHandle : n.viewAll}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
              {!rows.length && (
                <tr><td colSpan={n.columns.length} className={`${TD} text-center text-[var(--shell-group-title)]`}>{n.empty}</td></tr>
              )}
            </tbody>
          </table>
          <Pagination total={total} page={page} pageSize={pageSize} onPage={setPage} onSize={setPageSize}
            rangeText={n.rangeText} prevText={n.prev} nextText={n.next} perPageText={n.perPage}
            jumpText={n.jump} pageUnitText={n.pageUnit} />
        </>
      )}
    </div>
  )
}
