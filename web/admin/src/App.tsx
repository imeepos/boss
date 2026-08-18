// 路由:登录页独立;受保护区 AuthGuard→AdminLayout(Outlet);45 页全集 + 403/404。
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AdminLayout } from './layouts/AdminLayout'
import { UCenterLayout } from './layouts/UCenterLayout'
import { AuthGuard } from './layouts/AuthGuard'
import { useProfile } from './layouts/profile'
import LoginPage from './pages/login'
import RegisterPage from './pages/register'
import DashboardPage from './pages/dashboard'
import AccountListPage from './pages/base/account'
import AddressPage from './pages/base/address'
import GeoPage from './pages/base/geo'
import ImporterPage from './pages/base/importer'
import ProfilePage from './pages/profile'
import { ForbiddenPage, NotFoundPage } from './pages/error'
import { PlaceholderPage } from './pages/placeholder'
import { MENU_GROUPS } from './router/menu.def'
import { canAccess } from './router/role-menu'
import { useT } from './i18n'

/** 菜单页:越权直访 403;已接入页正式渲染,其余占位(A1 起逐页替换)。 */
function MenuPage({ pageKey }: { pageKey: string }) {
  const t = useT()
  const profile = useProfile()
  const label = t.menu.items[pageKey] ?? pageKey
  if (!canAccess(profile.roleCode, pageKey)) return <ForbiddenPage />
  if (pageKey === 'dashboard') return <DashboardPage profile={profile} />
  if (pageKey === 'account') return <AccountListPage />
  if (pageKey === 'address') return <AddressPage />
  if (pageKey === 'geo') return <GeoPage />
  if (pageKey === 'importer') return <ImporterPage />
  return <PlaceholderPage title={label} />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route
          path="/ucenter"
          element={
            <AuthGuard>
              {(profile) => <UCenterLayout profile={profile} />}
            </AuthGuard>
          }
        >
          <Route index element={<Navigate to="/ucenter/overview" replace />} />
          <Route path="overview" element={<ProfilePage />} />
          <Route path="personal" element={<ProfilePage />} />
          <Route path="security" element={<ProfilePage />} />
          <Route path="api-keys" element={<ProfilePage />} />
          <Route path="data" element={<ProfilePage />} />
        </Route>
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
            <Route key={it.key} path={it.path} element={<MenuPage pageKey={it.key} />} />
          ))}
          <Route path="profile" element={<Navigate to="/ucenter/overview" replace />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}