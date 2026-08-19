// 框架布局:顶栏 + 侧栏(折叠/抽屉)+ 面包屑 + 内容区 Outlet + FAB + 底栏。规格见 design-spec.md §1。
// 样式:tailwind 原子类(原 shell.css 已删除)。
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
      <div className="flex h-screen flex-col overflow-hidden font-base">
        <TopBar
          profile={profile}
          groups={groups}
          activeGroupId={activeGroupId}
          onOpenDrawer={() => setDrawerOpen(true)}
        />
        <div className="flex min-h-0 flex-1">
          <Sidebar
            groups={groups}
            collapsed={collapsed}
            drawerOpen={drawerOpen}
            onToggleCollapse={() => setCollapsed((v) => !v)}
            onCloseDrawer={() => setDrawerOpen(false)}
          />
          {drawerOpen && <div className="fixed inset-x-0 top-14 bottom-8 z-30 hidden bg-[rgba(15,30,59,0.4)] max-[959px]:block" onClick={() => setDrawerOpen(false)} />}
          <div className="relative min-w-0 flex-1 overflow-y-auto bg-[var(--shell-content-bg)]">
            <main className="px-6 pt-4 pb-12 text-[var(--shell-content-text)] [&_h1]:text-[var(--shell-heading)] [&_h2]:text-[var(--shell-heading)] [&_h3]:text-[var(--shell-heading)]">
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
