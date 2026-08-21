// 顶栏通知铃铛:30s 轮询未读数,点开下拉最近 10 条;点条目=已读+跳转,底部全部已读/查看全部。
import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiFetch } from '../api/client'
import { useT } from '../i18n'
import { cn } from '../lib/cn'
import { markNotificationsRead, useUnreadCount } from '../lib/useNotifications'
import { POPOVER, POPOVER_DIVIDER } from './popover'
import { BellIcon } from './icons'

export const NOTIF_ITEM = 'flex w-full cursor-pointer flex-col gap-0.5 rounded border-0 bg-none px-3 py-2 text-left hover:bg-[var(--shell-menu-hover-bg)]'
const TOOL_BTN = 'relative grid h-[34px] w-[34px] cursor-pointer place-items-center rounded-full border-0 bg-none text-white/80 hover:bg-[var(--shell-search-bg-focus)] hover:text-white'

interface RecentItem {
  id: number
  level: string
  title: string
  createdAt: string
  read: boolean
  link: string
}

/** 未读 URGENT → 红点徽标;普通未读 → 数字角标。 */
function Badge({ count }: { count: number }) {
  if (!count) return null
  return (
    <span className="absolute top-0.5 right-0.5 grid h-4 min-w-4 place-items-center rounded-full bg-[var(--color-danger)] px-1 text-[10px] font-semibold leading-none text-white">
      {count > 99 ? '99+' : count}
    </span>
  )
}

export function NotifBell() {
  const t = useT()
  const n = t.pages.message.notif
  const nav = useNavigate()
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<RecentItem[]>([])
  const { count, refresh } = useUnreadCount(true)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    apiFetch<{ items: RecentItem[] }>('/notifications', { query: { hideResolved: 1, limit: 10 } })
      .then((d) => setItems(d?.items ?? []))
      .catch(() => setItems([]))
    const onDocClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [open])

  const openItem = (r: RecentItem) => {
    setOpen(false)
    if (!r.read) void markNotificationsRead([r.id]).then(refresh)
    if (r.link) nav(r.link)
  }

  const markAll = () => {
    void markNotificationsRead([]).then(refresh)
    setItems((prev) => prev.map((it) => ({ ...it, read: true })))
  }

  return (
    <div className="relative" ref={rootRef}>
      <button
        className={cn(TOOL_BTN, open && 'bg-[var(--shell-search-bg-focus)] text-white')}
        onClick={() => setOpen((v) => !v)}
        title={t.shell.notifications}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <BellIcon />
        <Badge count={count} />
      </button>
      {open && (
        <div className={cn(POPOVER, 'w-80')} role="menu">
          <div className="border-b border-[var(--shell-popover-border)] px-3 pt-2 pb-2 text-[13px] font-semibold text-[var(--shell-heading)]">
            {n.cardTitle}
          </div>
          <div className="max-h-80 overflow-y-auto">
            {items.map((r) => (
              <button key={r.id} role="menuitem" className={NOTIF_ITEM} onClick={() => openItem(r)}>
                <span className={cn('text-[13px]', r.read ? 'text-[var(--shell-content-text)]' : 'font-semibold text-[var(--shell-heading)]')}>
                  {r.title}
                </span>
                <span className="flex items-center gap-1.5 text-xs text-[var(--shell-group-title)]">
                  <i className={cn('h-1.5 w-1.5 rounded-full', r.level === 'URGENT' ? 'bg-[var(--color-danger)]' : r.level === 'WARN' ? 'bg-[var(--color-brand-gold-500)]' : 'bg-[var(--shell-menu-icon)]')} />
                  {r.createdAt.replace('T', ' ').slice(5, 16)}
                </span>
              </button>
            ))}
            {!items.length && (
              <div className="px-2.5 py-4 text-center text-xs text-[var(--shell-group-title)]">{n.bellEmpty}</div>
            )}
          </div>
          <div className={cn(POPOVER_DIVIDER, 'flex pt-1')}>
            <button role="menuitem" className="flex-1 cursor-pointer border-0 bg-none py-2 text-xs text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]" onClick={markAll}>
              {n.markAllRead}
            </button>
            <button role="menuitem" className="flex-1 cursor-pointer border-0 bg-none py-2 text-xs text-[var(--color-brand-gold-500)] hover:bg-[var(--shell-menu-hover-bg)]" onClick={() => { setOpen(false); nav('/boss/message?tab=admin') }}>
              {n.viewAll}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
