// 侧栏:分组标题 + 平铺菜单项,240px/64px 折叠,移动端抽屉。规格见 design-spec.md §2.2/§3.2。
import { NavLink } from 'react-router-dom'
import type { MenuGroup } from '../router/menu.def'
import { useT } from '../i18n'
import { CollapseIcon, MaskIcon } from './icons'

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

export function Sidebar({ groups, collapsed, drawerOpen, onToggleCollapse, onCloseDrawer }: SidebarProps) {
  const t = useT()
  const cls = [
    'shell-side',
    collapsed ? 'collapsed' : '',
    drawerOpen ? 'drawer-open' : '',
  ].filter(Boolean).join(' ')
  return (
    <aside className={cls}>
      <nav className="shell-side-nav">
        {groups.map((g) => (
          <SideGroup key={g.id} g={g} collapsed={collapsed} onNavigate={onCloseDrawer} />
        ))}
      </nav>
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
