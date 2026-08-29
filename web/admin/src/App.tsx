// 路由:登录页独立;受保护区 AuthGuard→AdminLayout(Outlet);45 页全集 + 403/404。
import { lazy, Suspense, useEffect, useState } from 'react'
import { AttachmentManager } from './components/AttachmentManager'
import { BrowserRouter, Navigate, Route, Routes, useNavigate } from 'react-router-dom'
import { AdminLayout } from './layouts/AdminLayout'
import { UCenterLayout } from './layouts/UCenterLayout'
import { AuthGuard } from './layouts/AuthGuard'
import { useProfile } from './layouts/profile'
import { onLicenseRequired } from './api/client'
const LoginPage = lazy(() => import('./pages/login'))
const HomePage = lazy(() => import('./pages/home'))
const NewsDetailPage = lazy(() => import('./pages/news'))
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
const CrashLogsPage = lazy(() => import('./pages/base/crashlogs'))
const RealIDConfigPage = lazy(() => import('./pages/base/realidconfig'))
const RealnameReviewPage = lazy(() => import('./pages/base/realname-review'))
const StripeConfigPage = lazy(() => import('./pages/base/stripeconfig'))
const StorageConfigPage = lazy(() => import('./pages/base/storageconfig'))
const ServersPage = lazy(() => import('./pages/base/servers'))
const ApiDocsPage = lazy(() => import('./pages/base/apidocs'))
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
const OnboardingPage = lazy(() => import('./pages/bss/onboarding'))
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
const DailyClosePage = lazy(() => import('./pages/billing/daily-close'))
const AssetPage = lazy(() => import('./pages/ams/asset'))
const TagPage = lazy(() => import('./pages/ams/tag'))
const StockPage = lazy(() => import('./pages/ams/stock'))
const ReplacePage = lazy(() => import('./pages/ams/replace'))
const PurchasePage = lazy(() => import('./pages/ams/purchase'))
const InventoryPage = lazy(() => import('./pages/ams/inventory'))
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
const InstallBoardPage = lazy(() => import('./pages/boss/install-board'))
const DismantlePage = lazy(() => import('./pages/boss/dismantle'))
const ComplaintPage = lazy(() => import('./pages/boss/complaint'))
const KnowledgePage = lazy(() => import('./pages/boss/knowledge'))
const SitePostsPage = lazy(() => import('./pages/boss/site'))
const SitePostEditorPage = lazy(() => import('./pages/boss/site/editor'))
const SiteCategoriesPage = lazy(() => import('./pages/boss/site/categories'))
const ClientReleasePage = lazy(() => import('./pages/boss/release'))
const LicensePage = lazy(() => import('./pages/boss/license'))
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
import { canAccess, landingPathFor } from './router/role-menu'
import { useT } from './i18n'
import { getAuthToken } from './api/client'
import { ConfirmProvider } from './components/ConfirmDialog'

/** 菜单页:越权直访 403;已接入页正式渲染,其余占位(A1 起逐页替换)。 */
function MenuPage({ pageKey }: { pageKey: string }) {
  const t = useT()
  const profile = useProfile()
  const label = t.menu.items[pageKey] ?? pageKey
  // 落地页裁定A(2026-08-28):/dashboard 是登录默认落点,未持 menu:dashboard 的角色
  // 分流到权限码序首个有权页,消除"登录即 403";无可落地权限码时维持 403 页。
  if (pageKey === 'dashboard') {
    const held = new Set(profile.permissionCodes)
    if (!held.has('menu:dashboard')) {
      const to = landingPathFor(profile.permissionCodes)
      if (to) return <Navigate to={to} replace />
    }
  }
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
  if (pageKey === 'crashlogs') return <CrashLogsPage />
  if (pageKey === 'realidconfig') return <RealIDConfigPage />
  if (pageKey === 'realname-review') return <RealnameReviewPage />
  if (pageKey === 'stripeconfig') return <StripeConfigPage />
  if (pageKey === 'storageconfig') return <StorageConfigPage />
  if (pageKey === 'servers') return <ServersPage />
  if (pageKey === 'apidocs') return <ApiDocsPage />
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
  if (pageKey === 'onboarding') return <OnboardingPage />
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
  if (pageKey === 'daily-close') return <DailyClosePage />
  if (pageKey === 'asset') return <AssetPage />
  if (pageKey === 'tag') return <TagPage />
  if (pageKey === 'stock') return <StockPage />
  if (pageKey === 'replace') return <ReplacePage />
  if (pageKey === 'purchase') return <PurchasePage />
  if (pageKey === 'inventory') return <InventoryPage />
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
  if (pageKey === 'install-board') return <InstallBoardPage />
  if (pageKey === 'dismantle') return <DismantlePage />
  if (pageKey === 'complaint') return <ComplaintPage />
  if (pageKey === 'knowledge') return <KnowledgePage />
  if (pageKey === 'site') return <SitePostsPage />
  if (pageKey === 'site-cats') return <SiteCategoriesPage />
  if (pageKey === 'release') return <ClientReleasePage />
  if (pageKey === 'license') return <LicensePage />
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

/** 官网内容编辑子路由:new=新建,:postId=编辑;权限与列表页同源(menu:site)。 */
function SiteEditorRoute() {
  const profile = useProfile()
  if (!canAccess(profile.roleCode, 'site', profile.permissionCodes)) return <ForbiddenPage />
  return <SitePostEditorPage />
}

/** 系统授权跳转监听:业务接口被门禁拦截(无证书/证书失效)时,自动跳转系统授权页引导激活。
 *  事件由 apiFetch 在 LICENSE_REQUIRED 时 dispatch(见 api/client.ts)。 */
function LicenseRequiredWatcher() {
  const navigate = useNavigate()
  useEffect(
    () =>
      onLicenseRequired(() => {
        navigate('/base/license', { replace: true })
      }),
    [navigate],
  )
  return null
}

export default function App() {
  return (
    <ConfirmProvider>
    <Suspense fallback={<RouteFallback />}>
    <BrowserRouter>
      <LicenseRequiredWatcher />
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/partner/apply" element={<PartnerApplyPage />} />
        <Route path="/home" element={<HomePage />} />
        <Route path="/news/:slug" element={<NewsDetailPage />} />
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
          <Route path="boss/site/new" element={<SiteEditorRoute />} />
          <Route path="boss/site/:postId" element={<SiteEditorRoute />} />
          {/* 系统授权页直达路由:不经 MenuPage 的 canAccess(后端 /license/status 仅要求登录,
              无菜单权限码;未激活跳转必须对所有登录角色可达,否则引导失效)。 */}
          <Route path="base/license" element={<LicensePage />} />
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