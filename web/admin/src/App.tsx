// 路由:登录页独立;受保护区 AuthGuard→AdminLayout(Outlet);45 页全集 + 403/404。
import { lazy, Suspense, useState } from 'react'
import { AttachmentManager } from './components/AttachmentManager'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AdminLayout } from './layouts/AdminLayout'
import { UCenterLayout } from './layouts/UCenterLayout'
import { AuthGuard } from './layouts/AuthGuard'
import { useProfile } from './layouts/profile'
const LoginPage = lazy(() => import('./pages/login'))
const HomePage = lazy(() => import('./pages/home'))
const DashboardPage = lazy(() => import('./pages/dashboard'))
const AccountListPage = lazy(() => import('./pages/base/account'))
const AddressPage = lazy(() => import('./pages/base/address'))
const GeoPage = lazy(() => import('./pages/base/geo'))
const ImporterPage = lazy(() => import('./pages/base/importer'))
const BackupPage = lazy(() => import('./pages/backup'))
const StaffOrgPage = lazy(() => import('./pages/org/staff'))
const ParamsPage = lazy(() => import('./pages/base/params'))
const AuthConfigPage = lazy(() => import('./pages/base/authconfig'))
const SmsConfigPage = lazy(() => import('./pages/base/smsconfig'))
const PushConfigPage = lazy(() => import('./pages/base/pushconfig'))
const RealIDConfigPage = lazy(() => import('./pages/base/realidconfig'))
const StorageConfigPage = lazy(() => import('./pages/base/storageconfig'))
const ServersPage = lazy(() => import('./pages/base/servers'))
const AuditPage = lazy(() => import('./pages/base/audit'))
const CompanyPage = lazy(() => import('./pages/org/company'))
const DepartmentPage = lazy(() => import('./pages/org/department'))
const PostPage = lazy(() => import('./pages/org/post'))
const RegionPage = lazy(() => import('./pages/org/region'))
const MenuPermPage = lazy(() => import('./pages/org/menuperm'))
const DataScopePage = lazy(() => import('./pages/org/datascope'))
const ApiKeyPage = lazy(() => import('./pages/org/apikey'))
const OpenPlatPage = lazy(() => import('./pages/org/openplat'))
const MessageCenterPage = lazy(() => import('./pages/boss/message'))
const CustomerPage = lazy(() => import('./pages/bss/customer'))
const ProductPage = lazy(() => import('./pages/bss/product'))
const UserListPage = lazy(() => import('./pages/bss/user'))
const UserDataPage = lazy(() => import('./pages/bss/userdata'))
const MarketingRulesPage = lazy(() => import('./pages/bss/marketing'))
const MarketingReconPage = lazy(() => import('./pages/bss/marketing/recon'))
const BillPage = lazy(() => import('./pages/billing/billing'))
const PaymentPage = lazy(() => import('./pages/billing/payment'))
const ArrearsPage = lazy(() => import('./pages/billing/arrears'))
const CollectionTasksPage = lazy(() => import('./pages/billing/collection-tasks'))
const StopSrvPage = lazy(() => import('./pages/billing/stopsrv'))
const PayCheckPage = lazy(() => import('./pages/billing/paycheck'))
const AssetPage = lazy(() => import('./pages/ams/asset'))
const TagPage = lazy(() => import('./pages/ams/tag'))
const StockPage = lazy(() => import('./pages/ams/stock'))
const ReplacePage = lazy(() => import('./pages/ams/replace'))
const ResourcePage = lazy(() => import('./pages/oss/resource'))
const ODNPage = lazy(() => import('./pages/oss/odn'))
const ReservePage = lazy(() => import('./pages/oss/reserve'))
const TransferPage = lazy(() => import('./pages/oss/transfer'))
const DevicePage = lazy(() => import('./pages/oss/device'))
const LoAccountPage = lazy(() => import('./pages/oss/loaccount'))
const ExpandPage = lazy(() => import('./pages/oss/expand'))
const OrderPage = lazy(() => import('./pages/boss/order'))
const WorkerPage = lazy(() => import('./pages/boss/worker'))
const WorkerRegPage = lazy(() => import('./pages/boss/worker-registration'))
const WorkerOpsPage = lazy(() => import('./pages/boss/worker-ops'))
const DispatchPage = lazy(() => import('./pages/boss/dispatch'))
const DismantlePage = lazy(() => import('./pages/boss/dismantle'))
const ComplaintPage = lazy(() => import('./pages/boss/complaint'))
const ServiceMetricsPage = lazy(() => import('./pages/boss/service-metrics'))
const FeedbackPage = lazy(() => import('./pages/boss/feedback'))
const CallbackPage = lazy(() => import('./pages/boss/callback'))
const QuadLinkPage = lazy(() => import('./pages/quad/quadlink'))
const QuadCheckPage = lazy(() => import('./pages/quad/check'))
const ScanLogPage = lazy(() => import('./pages/quad/scanlog'))
const AlarmPage = lazy(() => import('./pages/alarm'))
const AaaDashboardPage = lazy(() => import('./pages/aaa-dashboard'))
const AaaLogPage = lazy(() => import('./pages/aaalog'))
const ProvisionTaskPage = lazy(() => import('./pages/provision/provision'))
const TemplatePage = lazy(() => import('./pages/provision/template'))
const ProvlogPage = lazy(() => import('./pages/provision/provlog'))
const GisPage = lazy(() => import('./pages/intel/gis'))
const AnalyticsPage = lazy(() => import('./pages/intel/analytics'))
const ReportPage = lazy(() => import('./pages/intel/report'))
const ProfilePage = lazy(() => import('./pages/profile'))
const PartnerApplyPage = lazy(() => import('./pages/partner/apply'))
const ForbiddenPage = lazy(() => import('./pages/error').then((m) => ({ default: m.ForbiddenPage })))
const NotFoundPage = lazy(() => import('./pages/error').then((m) => ({ default: m.NotFoundPage })))
const PlaceholderPage = lazy(() => import('./pages/placeholder').then((m) => ({ default: m.PlaceholderPage })))
const PartnerReviewPage = lazy(() => import('./pages/org/partner'))
const PartnerHomePage = lazy(() => import('./pages/partner/home'))
const PartnerStaffPage = lazy(() => import('./pages/partner/staff'))
const PartnerOrdersPage = lazy(() => import('./pages/partner/orders'))
import { MENU_GROUPS } from './router/menu.def'
import { canAccess } from './router/role-menu'
import { useT } from './i18n'
import { getAuthToken } from './api/client'
import { ConfirmProvider } from './components/ConfirmDialog'

/** 菜单页:越权直访 403;已接入页正式渲染,其余占位(A1 起逐页替换)。 */
function MenuPage({ pageKey }: { pageKey: string }) {
  const t = useT()
  const profile = useProfile()
  const label = t.menu.items[pageKey] ?? pageKey
  if (!canAccess(profile.roleCode, pageKey, profile.permissionCodes)) {
    // 入驻企业角色误落平台页(如登录后默认 /dashboard):送回企业工作台首页。
    if (profile.roleCode.startsWith('partner_')) return <Navigate to="/partner/home" replace />
    return <ForbiddenPage />
  }
  if (pageKey === 'dashboard') return <DashboardPage profile={profile} />
  if (pageKey === 'account') return <AccountListPage />
  if (pageKey === 'address') return <AddressPage />
  if (pageKey === 'geo') return <GeoPage />
  if (pageKey === 'importer') return <ImporterPage />
  if (pageKey === 'backup') return <BackupPage />
  if (pageKey === 'params') return <ParamsPage />
  if (pageKey === 'authconfig') return <AuthConfigPage />
  if (pageKey === 'smsconfig') return <SmsConfigPage />
  if (pageKey === 'pushconfig') return <PushConfigPage />
  if (pageKey === 'realidconfig') return <RealIDConfigPage />
  if (pageKey === 'storageconfig') return <StorageConfigPage />
  if (pageKey === 'servers') return <ServersPage />
  if (pageKey === 'audit') return <AuditPage />
  if (pageKey === 'company') return <CompanyPage />
  if (pageKey === 'staff') return <StaffOrgPage />
  if (pageKey === 'department') return <DepartmentPage />
  if (pageKey === 'post') return <PostPage />
  if (pageKey === 'region') return <RegionPage />
  if (pageKey === 'menuperm') return <MenuPermPage />
  if (pageKey === 'datascope') return <DataScopePage />
  if (pageKey === 'apikey') return <ApiKeyPage />
  if (pageKey === 'openplat') return <OpenPlatPage />
  if (pageKey === 'partner') return <PartnerReviewPage />
  if (pageKey === 'partner-home') return <PartnerHomePage />
  if (pageKey === 'partner-staff') return <PartnerStaffPage />
  if (pageKey === 'partner-orders') return <PartnerOrdersPage />
  if (pageKey === 'message') return <MessageCenterPage />
  if (pageKey === 'customer') return <CustomerPage />
  if (pageKey === 'product') return <ProductPage />
  if (pageKey === 'user') return <UserListPage />
  if (pageKey === 'userdata') return <UserDataPage />
  if (pageKey === 'marketing') return <MarketingRulesPage />
  if (pageKey === 'marketing-recon') return <MarketingReconPage />
  if (pageKey === 'billing') return <BillPage />
  if (pageKey === 'payment') return <PaymentPage />
  if (pageKey === 'arrears') return <ArrearsPage />
  if (pageKey === 'collection-tasks') return <CollectionTasksPage />
  if (pageKey === 'stopsrv') return <StopSrvPage />
  if (pageKey === 'paycheck') return <PayCheckPage />
  if (pageKey === 'asset') return <AssetPage />
  if (pageKey === 'tag') return <TagPage />
  if (pageKey === 'stock') return <StockPage />
  if (pageKey === 'replace') return <ReplacePage />
  if (pageKey === 'resource') return <ResourcePage />
  if (pageKey === 'odn') return <ODNPage />
  if (pageKey === 'reserve') return <ReservePage />
  if (pageKey === 'transfer') return <TransferPage />
  if (pageKey === 'device') return <DevicePage />
  if (pageKey === 'loaccount') return <LoAccountPage />
  if (pageKey === 'expand') return <ExpandPage />
  if (pageKey === 'order') return <OrderPage />
  if (pageKey === 'worker') return <WorkerPage />
  if (pageKey === 'worker-reg') return <WorkerRegPage />
  if (pageKey === 'worker-ops') return <WorkerOpsPage />
  if (pageKey === 'dispatch') return <DispatchPage />
  if (pageKey === 'dismantle') return <DismantlePage />
  if (pageKey === 'complaint') return <ComplaintPage />
  if (pageKey === 'service-metrics') return <ServiceMetricsPage />
  if (pageKey === 'feedback') return <FeedbackPage />
  if (pageKey === 'callback') return <CallbackPage />
  if (pageKey === 'quadlink') return <QuadLinkPage />
  if (pageKey === 'check') return <QuadCheckPage />
  if (pageKey === 'scanlog') return <ScanLogPage />
  if (pageKey === 'alarm') return <AlarmPage />
  if (pageKey === 'aaadashboard') return <AaaDashboardPage />
  if (pageKey === 'aaalog') return <AaaLogPage />
  if (pageKey === 'provision') return <ProvisionTaskPage />
  if (pageKey === 'template') return <TemplatePage />
  if (pageKey === 'provlog') return <ProvlogPage />
  if (pageKey === 'gis') return <GisPage />
  if (pageKey === 'analytics') return <AnalyticsPage />
  if (pageKey === 'report') return <ReportPage />
  return <PlaceholderPage title={label} />
}

function RouteFallback() {
  return <div className="flex min-h-32 items-center justify-center text-sm text-[var(--shell-group-title)]">Loading…</div>
}

/** 根路径分流:有 token 进工作台,无 token 看官网首页(未登录直接访问 / 不再弹登录)。 */
function RootRedirect() {
  return <Navigate to={getAuthToken() ? '/dashboard' : '/home'} replace />
}

/** 附件管理预览:自由筛选 + 选择模式全功能展示。 */
function AttachmentManagerPreview() {
  const [selected, setSelected] = useState<number[]>([])
  return <AttachmentManager selectable selectedIds={selected} onSelectionChange={setSelected} />
}

export default function App() {
  return (
    <ConfirmProvider>
    <Suspense fallback={<RouteFallback />}>
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/partner/apply" element={<PartnerApplyPage />} />
        <Route path="/home" element={<HomePage />} />
        {/* 根路径分流(公开):未登录看官网首页,已登录进工作台;守卫区改无路径布局路由。 */}
        <Route path="/" element={<RootRedirect />} />
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
          element={
            <AuthGuard>
              {(profile) => <AdminLayout profile={profile} />}
            </AuthGuard>
          }
        >
          {/* 附件管理组件预览路由(AttachmentManager 通用组件,正式嵌入业务页后移除)。 */}
          <Route path="dev/attachments" element={<AttachmentManagerPreview />} />
          {MENU_GROUPS.flatMap((g) => g.items).map((it) => (
            <Route key={it.key} path={it.path} element={<MenuPage pageKey={it.key} />} />
          ))}
          <Route path="profile" element={<Navigate to="/ucenter/overview" replace />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
    </Suspense>
    </ConfirmProvider>
  )
}