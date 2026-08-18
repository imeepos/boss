// 路由:登录页独立;受保护区 AuthGuard→AdminLayout(Outlet);45 页全集 + 403/404。
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AdminLayout } from './layouts/AdminLayout'
import { AuthGuard } from './layouts/AuthGuard'
import { useProfile } from './layouts/profile'
import LoginPage from './pages/login'
import DashboardPage from './pages/dashboard'
import AccountListPage from './pages/base/account'
import { ForbiddenPage, NotFoundPage } from './pages/error'
import { PlaceholderPage } from './pages/placeholder'
import { MENU_GROUPS } from './router/menu.def'
import { canAccess } from './router/role-menu'

/** 菜单页:越权直访 403;已接入页正式渲染,其余占位(A1 起逐页替换)。 */
function MenuPage({ pageKey, label }: { pageKey: string; label: string }) {
  const profile = useProfile()
  if (!canAccess(profile.roleCode, pageKey)) return <ForbiddenPage />
  if (pageKey === 'dashboard') return <DashboardPage profile={profile} />
  if (pageKey === 'account') return <AccountListPage />
  return <PlaceholderPage title={label} />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={
            <AuthGuard>
              {(profile) => <AdminLayout profile={profile} />}
            </AuthGuard>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          {MENU_GROUPS.flatMap((g) => g.items).map((it) => (
            <Route key={it.key} path={it.path} element={<MenuPage pageKey={it.key} label={it.label} />} />
          ))}
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
