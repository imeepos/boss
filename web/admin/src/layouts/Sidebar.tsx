// 侧栏:分组标题 + 平铺菜单项,240px/64px 折叠,移动端抽屉;右缘手柄拖拽调宽并持久化。
// 宽度经 CSS 变量 --side-w 注入,窄屏媒体查询仍可覆盖。规格见 design-spec.md §2.2/§3.2。
import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import type { MenuGroup } from '../router/menu.def'
import { useT } from '../i18n'
import { CollapseIcon, MaskIcon } from './icons'

const SIDE_MIN_W = 220
const SIDE_MAX_W = 420
const SIDE_W_KEY = 'admin.side-width'

function clampWidth(w: number) {
  return Math.min(SIDE_MAX_W, Math.max(SIDE_MIN_W, Math.round(w)))
}

function loadWidth() {
  const saved = Number(localStorage.getItem(SIDE_W_KEY))
  return Number.isFinite(saved) && saved >= SIDE_MIN_W ? clampWidth(saved) : 240
}

interface SidebarProps {
  groups: MenuGroup[]
  collapsed: boolean
  drawerOpen: boolean
  onToggleCollapse: () => void
  onCloseDrawer: () => void
}

function SideGroup({ g, collapsed, onNavigate }: { g: MenuGroup; collapsed: boolean; onNavigate: () => void }) {
  const t = useT()
  return (
    <section className="shell-side-group">
      {collapsed ? (
        <div className="shell-side-divider" />
      ) : (
        <div className="shell-side-group-title">
          <MaskIcon url={`/icons/${g.id}.svg`} size={14} />
          <span>{t.menu.groups[g.id] ?? g.label}</span>
        </div>
      )}
      {g.items.map((it) => {
        const label = t.menu.items[it.key] ?? it.label
        return (
          <NavLink
            key={it.key}
            to={it.path}
            title={label}
            onClick={onNavigate}
            className={({ isActive }) => (isActive ? 'shell-side-item active' : 'shell-side-item')}
          >
            <MaskIcon url={`/icons/items/${it.key}.svg`} />
            {!collapsed && <span className="shell-side-label">{label}</span>}
          </NavLink>
        )
      })}
    </section>
  )
}

// 激活菜单项滚动到视野中央:仅当目标在可视区外时才滚动,避免打断用户浏览
function scrollActiveIntoView(nav: HTMLElement) {
  const active = nav.querySelector<HTMLElement>('.shell-side-item.active')
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
    <div ref={trackRef} className="shell-side-scrollbar">
      <div
        className={`shell-side-scrollbar-thumb${drag ? ' dragging' : ''}`}
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
  const { pathname } = useLocation()
  useEffect(() => {
    if (navEl) scrollActiveIntoView(navEl)
  }, [pathname, collapsed, groups, navEl])
  const cls = [
    'shell-side',
    collapsed ? 'collapsed' : '',
    drawerOpen ? 'drawer-open' : '',
    dragging ? 'dragging' : '',
  ].filter(Boolean).join(' ')
  const commitWidth = (w: number) => {
    setWidth(w)
    localStorage.setItem(SIDE_W_KEY, String(w))
  }
  return (
    <aside className={cls} style={{ ['--side-w' as string]: `${width}px` }}>
      <nav ref={setNavEl} className="shell-side-nav">
        {groups.map((g) => (
          <SideGroup key={g.id} g={g} collapsed={collapsed} onNavigate={onCloseDrawer} />
        ))}
      </nav>
      <ScrollThumb nav={navEl} />
      <div
        className="shell-side-resizer"
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
        className="shell-side-collapse"
        onClick={onToggleCollapse}
        title={collapsed ? t.shell.expandMenu : t.shell.collapseMenu}
      >
        <CollapseIcon collapsed={collapsed} />
        {!collapsed && <span>{t.shell.collapseMenu}</span>}
      </button>
    </aside>
  )
}
