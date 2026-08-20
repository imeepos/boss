// 路由:登录页独立;受保护区 AuthGuard→AdminLayout(Outlet);45 页全集 + 403/404。
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AdminLayout } from './layouts/AdminLayout'
import { UCenterLayout } from './layouts/UCenterLayout'
import { AuthGuard } from './layouts/AuthGuard'
import { useProfile } from './layouts/profile'
import LoginPage from './pages/login'
import DashboardPage from './pages/dashboard'
import AccountListPage from './pages/base/account'
import AddressPage from './pages/base/address'
import GeoPage from './pages/base/geo'
import ImporterPage from './pages/base/importer'
import ParamsPage from './pages/base/params'
import AuthConfigPage from './pages/base/authconfig'
import SmsConfigPage from './pages/base/smsconfig'
import RealIDConfigPage from './pages/base/realidconfig'
import ServersPage from './pages/base/servers'
import AuditPage from './pages/base/audit'
import CompanyPage from './pages/org/company'
import DepartmentPage from './pages/org/department'
import PostPage from './pages/org/post'
import RegionPage from './pages/org/region'
import MenuPermPage from './pages/org/menuperm'
import DataScopePage from './pages/org/datascope'
import ApiKeyPage from './pages/org/apikey'
import MessageCenterPage from './pages/boss/message'
import CustomerPage from './pages/bss/customer'
import ProductPage from './pages/bss/product'
import UserListPage from './pages/bss/user'
import UserDataPage from './pages/bss/userdata'
import BillPage from './pages/billing/billing'
import PaymentPage from './pages/billing/payment'
import ArrearsPage from './pages/billing/arrears'
import StopSrvPage from './pages/billing/stopsrv'
import PayCheckPage from './pages/billing/paycheck'
import AssetPage from './pages/ams/asset'
import TagPage from './pages/ams/tag'
import StockPage from './pages/ams/stock'
import ReplacePage from './pages/ams/replace'
import ResourcePage from './pages/oss/resource'
import ReservePage from './pages/oss/reserve'
import TransferPage from './pages/oss/transfer'
import DevicePage from './pages/oss/device'
import LoAccountPage from './pages/oss/loaccount'
import ExpandPage from './pages/oss/expand'
import OrderPage from './pages/boss/order'
import WorkerPage from './pages/boss/worker'
import WorkerOpsPage from './pages/boss/worker-ops'
import DispatchPage from './pages/boss/dispatch'
import DismantlePage from './pages/boss/dismantle'
import ComplaintPage from './pages/boss/complaint'
import CallbackPage from './pages/boss/callback'
import QuadLinkPage from './pages/quad/quadlink'
import QuadCheckPage from './pages/quad/check'
import ScanLogPage from './pages/quad/scanlog'
import AlarmPage from './pages/alarm'
import AaaLogPage from './pages/aaalog'
import ProvisionTaskPage from './pages/provision/provision'
import TemplatePage from './pages/provision/template'
import ProvlogPage from './pages/provision/provlog'
import GisPage from './pages/intel/gis'
import AnalyticsPage from './pages/intel/analytics'
import ReportPage from './pages/intel/report'
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
  if (pageKey === 'params') return <ParamsPage />
  if (pageKey === 'authconfig') return <AuthConfigPage />
  if (pageKey === 'smsconfig') return <SmsConfigPage />
  if (pageKey === 'realidconfig') return <RealIDConfigPage />
  if (pageKey === 'servers') return <ServersPage />
  if (pageKey === 'audit') return <AuditPage />
  if (pageKey === 'company') return <CompanyPage />
  if (pageKey === 'department') return <DepartmentPage />
  if (pageKey === 'post') return <PostPage />
  if (pageKey === 'region') return <RegionPage />
  if (pageKey === 'menuperm') return <MenuPermPage />
  if (pageKey === 'datascope') return <DataScopePage />
  if (pageKey === 'apikey') return <ApiKeyPage />
  if (pageKey === 'message') return <MessageCenterPage />
  if (pageKey === 'customer') return <CustomerPage />
  if (pageKey === 'product') return <ProductPage />
  if (pageKey === 'user') return <UserListPage />
  if (pageKey === 'userdata') return <UserDataPage />
  if (pageKey === 'billing') return <BillPage />
  if (pageKey === 'payment') return <PaymentPage />
  if (pageKey === 'arrears') return <ArrearsPage />
  if (pageKey === 'stopsrv') return <StopSrvPage />
  if (pageKey === 'paycheck') return <PayCheckPage />
  if (pageKey === 'asset') return <AssetPage />
  if (pageKey === 'tag') return <TagPage />
  if (pageKey === 'stock') return <StockPage />
  if (pageKey === 'replace') return <ReplacePage />
  if (pageKey === 'resource') return <ResourcePage />
  if (pageKey === 'reserve') return <ReservePage />
  if (pageKey === 'transfer') return <TransferPage />
  if (pageKey === 'device') return <DevicePage />
  if (pageKey === 'loaccount') return <LoAccountPage />
  if (pageKey === 'expand') return <ExpandPage />
  if (pageKey === 'order') return <OrderPage />
  if (pageKey === 'worker') return <WorkerPage />
  if (pageKey === 'worker-ops') return <WorkerOpsPage />
  if (pageKey === 'dispatch') return <DispatchPage />
  if (pageKey === 'dismantle') return <DismantlePage />
  if (pageKey === 'complaint') return <ComplaintPage />
  if (pageKey === 'callback') return <CallbackPage />
  if (pageKey === 'quadlink') return <QuadLinkPage />
  if (pageKey === 'check') return <QuadCheckPage />
  if (pageKey === 'scanlog') return <ScanLogPage />
  if (pageKey === 'alarm') return <AlarmPage />
  if (pageKey === 'aaalog') return <AaaLogPage />
  if (pageKey === 'provision') return <ProvisionTaskPage />
  if (pageKey === 'template') return <TemplatePage />
  if (pageKey === 'provlog') return <ProvlogPage />
  if (pageKey === 'gis') return <GisPage />
  if (pageKey === 'analytics') return <AnalyticsPage />
  if (pageKey === 'report') return <ReportPage />
  return <PlaceholderPage title={label} />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
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
          <Route path="permissions" element={<ProfilePage />} />
          <Route path="work" element={<ProfilePage />} />
          <Route path="api-keys" element={<ProfilePage />} />
          <Route path="audit" element={<ProfilePage />} />
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