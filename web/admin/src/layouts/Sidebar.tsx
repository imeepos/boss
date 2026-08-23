// 侧栏:分组标题 + 平铺菜单项,240px/64px 折叠,移动端抽屉;右缘手柄拖拽调宽并持久化。
// 样式:tailwind 原子类(cn 合并),窄屏断点走 max-[1199px]/max-[959px] 变体。
import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import type { MenuGroup } from '../router/menu.def'
import { useT } from '../i18n'
import { cn } from '../lib/cn'
import { ChevronIcon, CollapseIcon, MaskIcon } from './icons'

const SIDE_MIN_W = 220
const SIDE_MAX_W = 420
const SIDE_W_KEY = 'admin.side-width'
const GROUP_FOLD_KEY = 'admin.side-group-folded'

const SIDE_BASE = 'relative flex flex-none flex-col border-r border-[var(--shell-side-border)] bg-[var(--shell-side-bg)] transition-[width] duration-200 group w-[var(--side-w,240px)] max-[1199px]:w-16 max-[959px]:fixed max-[959px]:bottom-8 max-[959px]:left-0 max-[959px]:top-14 max-[959px]:z-40 max-[959px]:w-60 max-[959px]:-translate-x-full max-[959px]:transition-transform max-[959px]:duration-200'
const SIDE_ITEM = 'relative flex h-11 items-center gap-2.5 px-4 text-sm whitespace-nowrap no-underline text-[var(--shell-menu-text)] hover:bg-[var(--shell-menu-hover-bg)] hover:text-[var(--shell-menu-active-text)] [&_.mask-icon]:text-[var(--shell-menu-icon)]'
const SIDE_ITEM_ACTIVE = 'bg-[var(--shell-menu-active-bg)] font-semibold text-[var(--shell-menu-active-text)] before:absolute before:left-0 before:top-2 before:bottom-2 before:w-[3px] before:rounded-r-sm before:bg-[var(--color-brand-gold-500)] [&_.mask-icon]:text-[var(--color-brand-gold-500)]'
const SIDE_COLLAPSE = 'flex h-11 flex-none cursor-pointer items-center gap-2 border-0 border-t border-[var(--shell-side-border)] bg-none px-5 text-[13px] text-[var(--shell-menu-icon)] hover:text-[var(--shell-menu-active-text)]'
const LABEL = 'max-[1199px]:hidden max-[959px]:inline'

function clampWidth(w: number) {
  return Math.min(SIDE_MAX_W, Math.max(SIDE_MIN_W, Math.round(w)))
}

function loadWidth() {
  const saved = Number(localStorage.getItem(SIDE_W_KEY))
  return Number.isFinite(saved) && saved >= SIDE_MIN_W ? clampWidth(saved) : 240
}

// 已折叠分组 id 集合持久化(仅展开态侧栏生效;当前路由所在分组恒展示其子项)
function loadFolded(): Set<string> {
  try {
    const raw = localStorage.getItem(GROUP_FOLD_KEY)
    const arr = raw ? (JSON.parse(raw) as unknown) : []
    return Array.isArray(arr) ? new Set(arr.filter((v): v is string => typeof v === 'string')) : new Set()
  } catch {
    return new Set()
  }
}

interface SidebarProps {
  groups: MenuGroup[]
  collapsed: boolean
  drawerOpen: boolean
  onToggleCollapse: () => void
  onCloseDrawer: () => void
}

function SideGroup({
  g,
  collapsed,
  folded,
  active,
  onToggleFold,
  onNavigate,
}: {
  g: MenuGroup
  collapsed: boolean
  folded: boolean
  active: boolean
  onToggleFold: () => void
  onNavigate: () => void
}) {
  const t = useT()
  const showItems = collapsed || !folded || active
  const label = t.menu.groups[g.id] ?? g.label
  return (
    <section className="">
      {collapsed ? (
        <div className="mx-4 my-3 border-t border-[var(--shell-side-border)]" />
      ) : (
        <button
          type="button"
          className="mx-4 mt-4 mb-1 flex w-[calc(100%-2rem)] cursor-pointer items-center gap-1.5 border-0 bg-transparent px-0 py-1 text-left text-[11px] font-medium tracking-[2px] text-[var(--shell-group-title)] hover:text-[var(--shell-menu-active-text)] max-[1199px]:justify-center max-[959px]:justify-start"
          aria-expanded={!folded}
          aria-label={label}
          onClick={onToggleFold}
        >
          <MaskIcon url={`/icons/${g.id}.svg`} size={14} />
          <span className={cn('overflow-hidden text-ellipsis', LABEL)}>{label}</span>
          <span className={LABEL}><ChevronIcon open={!folded} /></span>
        </button>
      )}
      {showItems && g.items.map((it) => {
        const label = t.menu.items[it.key] ?? it.label
        return (
          <NavLink
            key={it.key}
            to={it.path}
            title={label}
            onClick={onNavigate}
            className={({ isActive }) => cn(SIDE_ITEM, isActive && SIDE_ITEM_ACTIVE, collapsed && 'justify-center px-0')}
          >
            <MaskIcon url={`/icons/items/${it.key}.svg`} />
            {!collapsed && <span className={cn('overflow-hidden text-ellipsis', LABEL)}>{label}</span>}
          </NavLink>
        )
      })}
    </section>
  )
}

// 激活菜单项滚动到视野中央:仅当目标在可视区外时才滚动,避免打断用户浏览
function scrollActiveIntoView(nav: HTMLElement) {
  const active = nav.querySelector<HTMLElement>('[aria-current=page]')
  if (!active) return
  const { scrollTop, clientHeight } = nav
  const top = active.offsetTop - nav.offsetTop
  const itemTop = top - scrollTop
  const itemBottom = itemTop + active.offsetHeight
  if (itemTop >= 0 && itemBottom <= clientHeight) return
  nav.scrollTo({ top: top - (clientHeight - active.offsetHeight) / 2, behavior: 'smooth' })
}

// 手柄按下后捕获指针,随移动更新宽度,松开时持久化
function onResizeStart(e: ReactPointerEvent<HTMLDivElement>, width: number, setWidth: (w: number) => void) {
  e.preventDefault()
  const startX = e.clientX
  const startW = width
  const target = e.currentTarget
  target.setPointerCapture(e.pointerId)
  const move = (ev: PointerEvent) => setWidth(clampWidth(startW + ev.clientX - startX))
  const up = () => {
    target.removeEventListener('pointermove', move)
    target.removeEventListener('pointerup', up)
    target.removeEventListener('pointercancel', up)
  }
  target.addEventListener('pointermove', move)
  target.addEventListener('pointerup', up)
  target.addEventListener('pointercancel', up)
}

// 自定义短滑块:高度取可视占比但封顶 80px,无溢出时不渲染;支持拖拽滚动
function ScrollThumb({ nav }: { nav: HTMLElement | null }) {
  const trackRef = useRef<HTMLDivElement>(null)
  const [drag, setDrag] = useState(false)
  const [m, setM] = useState({ ratio: 1, pos: 0 })
  useEffect(() => {
    if (!nav) return
    const update = () => {
      const range = nav.scrollHeight - nav.clientHeight
      setM(range > 0 ? { ratio: nav.clientHeight / nav.scrollHeight, pos: nav.scrollTop / range } : { ratio: 1, pos: 0 })
    }
    update()
    nav.addEventListener('scroll', update)
    const ro = new ResizeObserver(update)
    ro.observe(nav)
    return () => {
      nav.removeEventListener('scroll', update)
      ro.disconnect()
    }
  }, [nav])
  if (m.ratio >= 1) return null
  const thumbH = Math.min(80, m.ratio * 100)
  const onDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    const track = trackRef.current
    if (!track || !nav) return
    e.preventDefault()
    const startY = e.clientY
    const startTop = nav.scrollTop
    const range = nav.scrollHeight - nav.clientHeight
    const pxPerTrack = track.clientHeight * (1 - thumbH / 100)
    const target = e.currentTarget
    target.setPointerCapture(e.pointerId)
    const move = (ev: PointerEvent) => {
      if (pxPerTrack > 0) nav.scrollTop = startTop + ((ev.clientY - startY) / pxPerTrack) * range
    }
    const up = () => {
      target.removeEventListener('pointermove', move)
      target.removeEventListener('pointerup', up)
      target.removeEventListener('pointercancel', up)
    }
    target.addEventListener('pointermove', move)
    target.addEventListener('pointerup', up)
    target.addEventListener('pointercancel', up)
  }
  return (
    <div ref={trackRef} className="absolute right-0.5 bottom-14 top-3 z-[4] w-1">
      <div
        className={cn('absolute min-h-6 w-1 rounded-sm bg-[var(--shell-menu-icon)] opacity-0 transition-opacity duration-150 group-hover:opacity-60 hover:opacity-100', drag && 'opacity-100')}
        style={{ height: `${thumbH}%`, top: `${m.pos * (100 - thumbH)}%` }}
        onPointerDown={(e) => {
          setDrag(true)
          onDown(e)
        }}
        onPointerUp={() => setDrag(false)}
        onPointerCancel={() => setDrag(false)}
      />
    </div>
  )
}

export function Sidebar({ groups, collapsed, drawerOpen, onToggleCollapse, onCloseDrawer }: SidebarProps) {
  const t = useT()
  const [navEl, setNavEl] = useState<HTMLElement | null>(null)
  const [width, setWidth] = useState(loadWidth)
  const [dragging, setDragging] = useState(false)
  const [folded, setFolded] = useState(loadFolded)
  const { pathname } = useLocation()
  useEffect(() => {
    if (navEl) scrollActiveIntoView(navEl)
  }, [pathname, collapsed, groups, navEl])
  const commitWidth = (w: number) => {
    setWidth(w)
    localStorage.setItem(SIDE_W_KEY, String(w))
  }
  const toggleFold = (id: string) => {
    setFolded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      localStorage.setItem(GROUP_FOLD_KEY, JSON.stringify([...next]))
      return next
    })
  }
  return (
    <aside
      className={cn(SIDE_BASE, collapsed && 'w-16 max-[959px]:w-60', drawerOpen && 'max-[959px]:translate-x-0')}
      style={{ ['--side-w' as string]: `${width}px` }}
    >
      <nav ref={setNavEl} className="flex-1 overflow-x-hidden overflow-y-auto px-0 py-2 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        {groups.map((g) => (
          <SideGroup
            key={g.id}
            g={g}
            collapsed={collapsed}
            folded={folded.has(g.id)}
            active={g.items.some((it) => pathname.startsWith(it.path))}
            onToggleFold={() => toggleFold(g.id)}
            onNavigate={onCloseDrawer}
          />
        ))}
      </nav>
      <ScrollThumb nav={navEl} />
      <div
        className={cn(
          'absolute top-0 right-[-3px] z-[5] h-full w-1.5 cursor-col-resize rounded-sm hover:bg-[var(--color-brand-gold-500)] max-[1199px]:hidden',
          dragging && 'bg-[var(--color-brand-gold-500)]',
          collapsed && 'hidden',
        )}
        onPointerDown={(e) => {
          setDragging(true)
          onResizeStart(e, width, setWidth)
        }}
        onPointerUp={() => {
          setDragging(false)
          commitWidth(width)
        }}
        onPointerCancel={() => setDragging(false)}
      />
      <button
        className={cn(SIDE_COLLAPSE, collapsed && 'justify-center px-0')}
        onClick={onToggleCollapse}
        title={collapsed ? t.shell.expandMenu : t.shell.collapseMenu}
      >
        <CollapseIcon collapsed={collapsed} />
        {!collapsed && <span className={LABEL}>{t.shell.collapseMenu}</span>}
      </button>
    </aside>
  )
}
