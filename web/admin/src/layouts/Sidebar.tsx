// 侧栏:按 roleCode 过滤 13 分组,两级折叠,当前路由高亮。
import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { MENU_GROUPS } from '../router/menu.def'
import { visibleGroupIds } from '../router/role-menu'

export function Sidebar({ roleCode }: { roleCode: string }) {
  const visible = new Set(visibleGroupIds(roleCode))
  const firstOpen = MENU_GROUPS.find((g) => visible.has(g.id))?.id ?? 'overview'
  const [open, setOpen] = useState<Set<string>>(new Set([firstOpen]))

  const toggle = (id: string) => {
    setOpen((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  return (
    <nav style={{ width: 208, background: '#fff', borderRight: '1px solid #e8e8e8', padding: '12px 0' }}>
      {MENU_GROUPS.filter((g) => visible.has(g.id)).map((g) => (
        <div key={g.id}>
          <div
            onClick={() => toggle(g.id)}
            style={{ padding: '10px 16px', fontWeight: 600, cursor: 'pointer', fontSize: 13 }}
          >
            {g.label}
          </div>
          {open.has(g.id) && (
            <div>
              {g.items.map((it) => (
                <NavLink
                  key={it.key}
                  to={it.path}
                  className={({ isActive }) => (isActive ? 'side-item active' : 'side-item')}
                  style={({ isActive }) => ({
                    display: 'block',
                    padding: '8px 16px 8px 28px',
                    fontSize: 13,
                    color: isActive ? '#1677ff' : '#333',
                    textDecoration: 'none',
                  })}
                >
                  {it.label}
                </NavLink>
              ))}
            </div>
          )}
        </div>
      ))}
    </nav>
  )
}
