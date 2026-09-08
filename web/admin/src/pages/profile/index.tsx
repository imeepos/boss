// 个人工作台内容页：各分区由 UCenterLayout 的独立路由承载。
// 大分区 ApiKey/Audit 独立成文件,共享样式与小组件在 shared.tsx。
// 样式:tailwind 原子类(原 profile.css 已删除),令牌走 shell-* 体系。
import { useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useProfile } from '../../layouts/profile'
import type { Profile } from '../../api/auth'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { SubmitButton, type SubmitState } from '../../components/business/submit-button'
import { Input } from '../../components/ui/input'
import {
  AVATAR, BOX, FORM_LABEL, LIST_BTN, PAGE, READONLY_INPUT,
  SectionTitle, SecurityRow, Summary, OverviewLink, ReadOnlyField,
} from './shared'
import { ApiKeySection } from './ApiKeySection'
import { AuditSection } from './AuditSection'

export default function ProfilePage() {
  const profile = useProfile()
  const { pathname } = useLocation()
  if (pathname.endsWith('/overview')) return <OverviewSection profile={profile} />
  if (pathname.endsWith('/security')) return <SecuritySection />
  if (pathname.endsWith('/api-keys')) return <ApiKeySection />
  if (pathname.endsWith('/permissions')) return <PermissionsSection profile={profile} />
  if (pathname.endsWith('/work')) return <MyDataSection />
  if (pathname.endsWith('/audit')) return <AuditSection />
  if (pathname.endsWith('/data')) return <MyDataSection />
  return <PersonalSection profile={profile} />
}

function OverviewSection({ profile }: { profile: Profile }) {
  const t = useT()
  return (
    <div className={PAGE}>
      <SectionTitle title={t.pages.profile.overview.title} desc={t.pages.profile.overview.desc} />
      <div className="mt-6 grid grid-cols-[minmax(0,1.5fr)_minmax(220px,1fr)] gap-4 max-[820px]:grid-cols-1">
        <div className={'flex items-center gap-3.5 ' + BOX}>
          <span className={AVATAR}>{profile.realName.slice(0, 1)}</span>
          <div className="grid gap-1"><strong className="text-[15px] font-semibold text-[var(--shell-heading)]">{profile.realName}</strong><span className="text-xs text-[var(--shell-content-text)]">@{profile.username}</span><small className="text-xs text-[var(--shell-content-text)]">{profile.roleName} · {profile.legalEntityName || t.pages.profile.personal.unassigned}</small></div>
        </div>
        <div className={BOX}>
          <span className="text-xs text-[var(--shell-content-text)]">{t.pages.profile.overview.accountStatus}</span>
          <strong className="text-[15px] font-semibold text-[var(--shell-heading)]">{t.pages.profile.activeAccount}</strong>
          <small className="text-xs text-[var(--shell-content-text)]">{t.pages.profile.overview.accountStatusDesc}</small>
        </div>
      </div>
      <div className="mt-7">
        <h2 className="m-0 mb-3 text-[15px] font-semibold text-[var(--shell-heading)]">{t.pages.profile.overview.quickAccess}</h2>
        <div className="grid grid-cols-3 gap-3 max-[820px]:grid-cols-1">
          <OverviewLink title={t.pages.profile.overview.dispatch} desc={t.pages.profile.overview.dispatchDesc} href="/boss/dispatch" />
          <OverviewLink title={t.pages.profile.overview.order} desc={t.pages.profile.overview.orderDesc} href="/boss/order" />
          <OverviewLink title={t.pages.profile.overview.complaint} desc={t.pages.profile.overview.complaintDesc} href="/boss/complaint" />
        </div>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-3 max-[560px]:grid-cols-1">
        <div className={BOX}><span className="text-xs text-[var(--shell-content-text)]">{t.pages.profile.overview.permissionSummary}</span><strong className="text-[15px] font-semibold text-[var(--shell-heading)]">{profile.roleName}</strong></div>
        <div className={BOX}><span className="text-xs text-[var(--shell-content-text)]">{t.pages.profile.overview.dataScopeSummary}</span><strong className="text-[15px] font-semibold text-[var(--shell-heading)]">{profile.regionScope || t.pages.profile.personal.allScope}</strong></div>
      </div>
    </div>
  )
}

// 基本资料:自助可编辑仅 realName/phone(PUT /auth/profile);角色/公司/数据范围只读。
function PersonalSection({ profile }: { profile: Profile }) {
  const t = useT()
  const p = t.pages.profile.personal
  const [realName, setRealName] = useState(profile.realName)
  const [phone, setPhone] = useState(profile.phone ?? '')
  const [submit, setSubmit] = useState<SubmitState>('idle')
  const onSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (submit === 'loading') return
    setSubmit('loading')
    apiFetch('/auth/profile', { method: 'PUT', body: { realName: realName.trim(), phone: phone.trim() } })
      .then(() => { setSubmit('success'); toast.success(p.saved) })
      .catch((e) => { setSubmit('failed'); toast.error(p.saveFail, { description: e instanceof Error ? e.message : undefined }) })
      .finally(() => setTimeout(() => setSubmit('idle'), 1500))
  }
  const labels: Record<SubmitState, string> = {
    idle: t.pages.profile.save, loading: t.common.loading, success: p.saved, failed: p.saveFail,
  }
  return (
    <div className={PAGE}>
      <SectionTitle title={p.title} desc={p.desc} />
      <form className="grid max-w-[760px] grid-cols-2 gap-x-6 gap-y-4.5 pt-6 max-[560px]:grid-cols-1" onSubmit={onSubmit}>
        <label className={FORM_LABEL}>{p.username}<Input value={profile.username} readOnly className={READONLY_INPUT} /></label>
        <label className={FORM_LABEL}>{p.realName}<Input value={realName} required onChange={(e) => setRealName(e.target.value)} /></label>
        <label className={FORM_LABEL}>{p.phone}<Input value={phone} placeholder={p.phonePlaceholder} onChange={(e) => setPhone(e.target.value)} /></label>
        <div className="col-span-full grid grid-cols-3 gap-4 border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] p-4 max-[560px]:grid-cols-1">
          <ReadOnlyField label={p.role} value={profile.roleName} />
          <ReadOnlyField label={p.company} value={profile.legalEntityName || p.unassigned} />
          <ReadOnlyField label={p.dataScope} value={profile.regionScope || p.allScope} />
        </div>
        <div className="col-span-full flex min-h-9 items-center justify-end text-xs">
          <SubmitButton state={submit} labels={labels} />
        </div>
      </form>
    </div>
  )
}

function SecuritySection() {
  const t = useT()
  const p = t.pages.profile.password
  const [oldPw, setOldPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [confirmPw, setConfirmPw] = useState('')
  const [submit, setSubmit] = useState<SubmitState>('idle')
  const onSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (submit === 'loading') return
    if (newPw !== confirmPw) { setSubmit('failed'); toast.error(p.mismatch); return }
    setSubmit('loading')
    apiFetch('/auth/change-password', { method: 'POST', body: { oldPassword: oldPw, newPassword: newPw } })
      .then(() => {
        setSubmit('success')
        toast.success(p.success)
        setOldPw(''); setNewPw(''); setConfirmPw('')
      })
      .catch((e) => { setSubmit('failed'); toast.error(p.fail, { description: e instanceof Error ? e.message : undefined }) })
      .finally(() => setTimeout(() => setSubmit('idle'), 1500))
  }
  const labels: Record<SubmitState, string> = {
    idle: p.submit, loading: t.common.loading, success: p.success, failed: p.fail,
  }
  return (
    <div className={PAGE}>
      <SectionTitle title={p.title} desc={p.desc} />
      <form className="grid max-w-[520px] gap-y-4.5 pt-6" onSubmit={onSubmit}>
        <label className={FORM_LABEL}>{p.old}<Input type="password" required autoComplete="current-password" value={oldPw} onChange={(e) => setOldPw(e.target.value)} /></label>
        <label className={FORM_LABEL}>{p.next}<Input type="password" required minLength={6} autoComplete="new-password" value={newPw} onChange={(e) => setNewPw(e.target.value)} /></label>
        <label className={FORM_LABEL}>{p.confirm}<Input type="password" required minLength={6} autoComplete="new-password" value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)} /></label>
        <div className="flex min-h-9 items-center justify-end text-xs">
          <SubmitButton state={submit} labels={labels} />
        </div>
      </form>
      <div className="mt-8 max-w-[760px] border-t border-[var(--shell-side-border)]">
        <h2 className="mt-5 mb-2 text-[15px] font-semibold text-[var(--shell-heading)]">{t.pages.profile.securityProtection.title}</h2>
        <SecurityRow label={t.pages.profile.securityProtection.loginProtection} value={t.pages.profile.securityProtection.pending} />
        <SecurityRow label={t.pages.profile.securityProtection.loginHistory} value={t.pages.profile.securityProtection.pending} />
      </div>
    </div>
  )
}

// 我的工作:快捷入口跳转对应页面(计数聚合后端无对应端点,不展示假数字)。
function MyDataSection() {
  const t = useT(); const navigate = useNavigate()
  const items: Array<[string, string, string]> = [
    [t.pages.profile.myData.orders, t.pages.profile.myData.ordersDesc, '/boss/order'],
    [t.pages.profile.myData.bills, t.pages.profile.myData.billsDesc, '/billing/billing'],
    [t.pages.profile.myData.service, t.pages.profile.myData.serviceDesc, '/boss/complaint'],
    [t.pages.profile.myData.messages, t.pages.profile.myData.messagesDesc, '/boss/message'],
    [t.pages.profile.myData.audit, t.pages.profile.myData.auditDesc, '/ucenter/audit'],
    [t.pages.profile.myData.permissions, t.pages.profile.myData.permissionsDesc, '/ucenter/permissions'],
  ]
  return (
    <div className={PAGE}>
      <SectionTitle title={t.pages.profile.myData.title} desc={t.pages.profile.myData.desc} />
      <div className="border-t border-[var(--shell-side-border)]">
        {items.map(([label, desc, href]) => (
          <button key={label} className={LIST_BTN} onClick={() => navigate(href)}>
            <span className="grid gap-1"><strong className="text-[13px] font-medium">{label}</strong><small className="text-xs text-[var(--shell-content-text)]">{desc}</small></span>
            <i className={'inline-block h-2 w-2 rotate-45 border-t-[1.5px] border-r-[1.5px] border-current'} />
          </button>
        ))}
      </div>
    </div>
  )
}

function PermissionsSection({ profile }: { profile: Profile }) {
  const t = useT()
  return (
    <div className={PAGE}>
      <SectionTitle title={t.pages.profile.permissions.title} desc={t.pages.profile.permissions.desc} />
      <div className="my-6 grid grid-cols-3 gap-3 max-[560px]:grid-cols-1">
        <Summary value={profile.roleName} label={t.pages.profile.permissions.role} />
        <Summary value={profile.legalEntityName || t.pages.profile.personal.unassigned} label={t.pages.profile.permissions.company} />
        <Summary value={profile.regionScope || t.pages.profile.personal.allScope} label={t.pages.profile.permissions.scope} />
      </div>
    </div>
  )
}
