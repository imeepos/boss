// 框架布局:顶栏 + 侧栏(折叠/抽屉)+ 面包屑 + 内容区 Outlet + FAB + 底栏。规格见 design-spec.md §1。
import { useState, type ReactNode } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import type { Profile } from '../api/auth'
import { ProfileContext } from './profile'
import { TopBar } from './TopBar'
import { Sidebar } from './Sidebar'
import { Breadcrumb } from './Breadcrumb'
import { BottomBar } from './BottomBar'
import { Fab } from './Fab'
import { MENU_GROUPS, KEY_BY_PATH, PAGE_BY_KEY } from '../router/menu.def'
import { visibleGroupIds } from '../router/role-menu'
import './shell.css'

export function AdminLayout({ profile }: { profile: Profile; children?: ReactNode }) {
  const { pathname } = useLocation()
  const [collapsed, setCollapsed] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)

  const visible = new Set(visibleGroupIds(profile.roleCode))
  const groups = MENU_GROUPS.filter((g) => visible.has(g.id))
  const activeKey = KEY_BY_PATH.get(pathname)
  const activeGroupId = activeKey ? PAGE_BY_KEY.get(activeKey)?.groupId : undefined

  return (
    <ProfileContext.Provider value={profile}>
      <div className="shell">
        <TopBar
          profile={profile}
          groups={groups}
          activeGroupId={activeGroupId}
          onOpenDrawer={() => setDrawerOpen(true)}
        />
        <div className="shell-body">
          <Sidebar
            groups={groups}
            collapsed={collapsed}
            drawerOpen={drawerOpen}
            onToggleCollapse={() => setCollapsed((v) => !v)}
            onCloseDrawer={() => setDrawerOpen(false)}
          />
          {drawerOpen && <div className="shell-backdrop" onClick={() => setDrawerOpen(false)} />}
          <div className="shell-main-wrap">
            <main className="shell-main">
              <Breadcrumb />
              <Outlet />
            </main>
            <Fab />
          </div>
        </div>
        <BottomBar />
      </div>
    </ProfileContext.Provider>
  )
}
