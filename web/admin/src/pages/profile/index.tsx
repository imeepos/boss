// 个人工作台内容页：各分区由 UCenterLayout 的独立路由承载。
// 样式:tailwind 原子类(原 profile.css 已删除),令牌走 shell-* 体系。
import { useEffect, useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useProfile } from '../../layouts/profile'
import type { Profile } from '../../api/auth'
import { apiFetch } from '../../api/client'
import { ApiError } from '../../api/envelope'
import { toAuditLog, type AuditEntry, type AuditLog } from '../base/audit/logic'
import { useT } from '../../i18n'
import { ToolbarButton } from '../../components/business/page-head'
import { Badge } from '../../components/ui/badge'
import { Input } from '../../components/ui/input'
import { useConfirm } from '../../components/ConfirmDialog'

const PAGE = 'border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-6 shadow-[var(--shell-card-shadow)] md:p-8'
const BOX = 'grid gap-[7px] border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] p-4.5'
const AVATAR = 'grid h-[52px] w-[52px] flex-none place-items-center rounded-full bg-[var(--color-brand-gold-300)] text-[21px] font-semibold text-[var(--color-brand-navy-950)]'
const CHEVRON = 'absolute right-4 top-10 inline-block h-2 w-2 rotate-45 border-t-[1.5px] border-r-[1.5px] border-current text-[var(--color-brand-gold-600)]'
const FORM_LABEL = 'grid gap-[7px] text-[13px] text-[var(--shell-content-text)]'
const READONLY_INPUT = 'read-only:bg-[var(--shell-input-disabled-bg)] read-only:text-[var(--shell-crumb-text)]'
const MSG = (ok: boolean) => ({ color: ok ? 'var(--color-success)' : 'var(--color-danger)' })
const TIP = 'mt-4 bg-[var(--shell-menu-hover-bg)] px-3.5 py-2.5 text-xs leading-relaxed text-[var(--shell-content-text)]'
const LIST_BTN = 'flex w-full min-h-16 cursor-pointer items-center justify-between border-0 border-b border-[var(--shell-side-border)] bg-transparent px-1 py-3 text-left text-[var(--shell-heading)] hover:text-[var(--color-brand-gold-600)]'

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

function SectionTitle({ title, desc }: { title: string; desc: string }) {
  return (
    <div className="mb-0 border-b border-[var(--shell-side-border)] pb-[18px]">
      <h1 className="m-0 mb-1.5 text-lg font-semibold text-[var(--shell-heading)]">{title}</h1>
      <p className="m-0 text-[13px] text-[var(--shell-content-text)]">{desc}</p>
    </div>
  )
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

function OverviewLink({ title, desc, href }: { title: string; desc: string; href: string }) {
  return (
    <Link className="relative grid min-h-[92px] gap-[7px] border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] p-4 text-[var(--shell-heading)] no-underline hover:border-[var(--color-brand-gold-500)] hover:bg-[var(--shell-menu-hover-bg)]" to={href}>
      <strong className="text-sm font-semibold">{title}</strong><span className="text-xs leading-relaxed text-[var(--shell-content-text)]">{desc}</span><i className={CHEVRON} />
    </Link>
  )
}

// 基本资料:自助可编辑仅 realName/phone(PUT /auth/profile);角色/公司/数据范围只读。
function PersonalSection({ profile }: { profile: Profile }) {
  const t = useT()
  const [realName, setRealName] = useState(profile.realName)
  const [phone, setPhone] = useState(profile.phone ?? '')
  const [msg, setMsg] = useState('')
  const [ok, setOk] = useState(false)
  const [busy, setBusy] = useState(false)
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setMsg(''); setBusy(true)
    apiFetch('/auth/profile', { method: 'PUT', body: { realName: realName.trim(), phone: phone.trim() } })
      .then(() => { setOk(true); setMsg(t.pages.profile.personal.saved) })
      .catch(() => { setOk(false); setMsg(t.pages.profile.personal.saveFail) })
      .finally(() => setBusy(false))
  }
  return (
    <div className={PAGE}>
      <SectionTitle title={t.pages.profile.personal.title} desc={t.pages.profile.personal.desc} />
      <form className="grid max-w-[760px] grid-cols-2 gap-x-6 gap-y-4.5 pt-6 max-[560px]:grid-cols-1" onSubmit={submit}>
        <label className={FORM_LABEL}>{t.pages.profile.personal.username}<Input value={profile.username} readOnly className={READONLY_INPUT} /></label>
        <label className={FORM_LABEL}>{t.pages.profile.personal.realName}<Input value={realName} required onChange={(e) => setRealName(e.target.value)} /></label>
        <label className={FORM_LABEL}>{t.pages.profile.personal.phone}<Input value={phone} placeholder={t.pages.profile.personal.phonePlaceholder} onChange={(e) => setPhone(e.target.value)} /></label>
        <div className="col-span-full grid grid-cols-3 gap-4 border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] p-4 max-[560px]:grid-cols-1">
          <ReadOnlyField label={t.pages.profile.personal.role} value={profile.roleName} />
          <ReadOnlyField label={t.pages.profile.personal.company} value={profile.legalEntityName || t.pages.profile.personal.unassigned} />
          <ReadOnlyField label={t.pages.profile.personal.dataScope} value={profile.regionScope || t.pages.profile.personal.allScope} />
        </div>
        <div className="col-span-full flex min-h-9 items-center justify-between text-xs">
          <span style={MSG(ok)}>{msg}</span>
          <ToolbarButton primary disabled={busy}>{t.pages.profile.save}</ToolbarButton>
        </div>
      </form>
    </div>
  )
}

function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return <div className="grid gap-1.5"><span className="text-xs text-[var(--shell-content-text)]">{label}</span><strong className="text-[13px] font-medium text-[var(--shell-heading)]">{value}</strong></div>
}

function SecuritySection() {
  const t = useT()
  const p = t.pages.profile.password
  const [oldPw, setOldPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [confirmPw, setConfirmPw] = useState('')
  const [msg, setMsg] = useState('')
  const [ok, setOk] = useState(false)
  const [busy, setBusy] = useState(false)
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (newPw !== confirmPw) { setOk(false); setMsg(p.mismatch); return }
    setMsg('')
    setBusy(true)
    apiFetch('/auth/change-password', { method: 'POST', body: { oldPassword: oldPw, newPassword: newPw } })
      .then(() => { setOk(true); setMsg(p.success); setOldPw(''); setNewPw(''); setConfirmPw('') })
      .catch(() => { setOk(false); setMsg(p.fail) })
      .finally(() => setBusy(false))
  }
  return (
    <div className={PAGE}>
      <SectionTitle title={p.title} desc={p.desc} />
      <form className="grid max-w-[520px] gap-y-4.5 pt-6" onSubmit={submit}>
        <label className={FORM_LABEL}>{p.old}<Input type="password" required value={oldPw} onChange={(e) => setOldPw(e.target.value)} /></label>
        <label className={FORM_LABEL}>{p.next}<Input type="password" required minLength={6} value={newPw} onChange={(e) => setNewPw(e.target.value)} /></label>
        <label className={FORM_LABEL}>{p.confirm}<Input type="password" required minLength={6} value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)} /></label>
        <div className="flex min-h-9 items-center justify-between text-xs">
          <span style={MSG(ok)}>{msg}</span>
          <ToolbarButton primary disabled={busy}>{p.submit}</ToolbarButton>
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

function SecurityRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex min-h-12 items-center gap-3.5 border-b border-[var(--shell-side-border)] text-[13px] text-[var(--shell-heading)]">
      <span>{label}</span><em className="ml-auto text-xs font-normal text-[var(--shell-crumb-text)]">{value}</em>
      <button aria-label={label} className="h-7 w-7 cursor-pointer border-0 bg-transparent p-0 text-[var(--shell-crumb-text)]"><i className={CHEVRON + ' static'} /></button>
    </div>
  )
}

interface OwnKeyRow {
  id: number
  subjectType: string
  subjectRef: number
  name: string
  keyPrefix: string
  status: number
  lastUsedAt: string
  createdAt: string
}

function ApiKeySection() {
  const t = useT()
  const k = t.pages.profile.apiKey
  const confirmDialog = useConfirm()
  const profile = useProfile()
  const [rows, setRows] = useState<OwnKeyRow[]>([])
  const [error, setError] = useState('')
  const [denied, setDenied] = useState(false)
  const [keyName, setKeyName] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [plainKey, setPlainKey] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError(''); setDenied(false)
    apiFetch<{ items: OwnKeyRow[] }>('/api-keys')
      .then((d) => setRows((d?.items ?? []).filter((r) => r.subjectType === 'account' && r.subjectRef === profile.accountId)))
      .catch((e) => {
        if (e instanceof ApiError && e.code === 403) setDenied(true)
        else setError(e instanceof Error ? e.message : k.loadFail)
      })
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const create = () => {
    if (busy || !keyName.trim()) return
    setBusy(true)
    apiFetch<{ plainKey: string }>('/api-keys', {
      method: 'POST',
      body: { subjectType: 'account', subjectRef: profile.accountId, name: keyName.trim() },
    })
      .then((res) => { setModalOpen(false); setKeyName(''); setPlainKey(res?.plainKey ?? ''); load() })
      .catch((e) => setError(e instanceof Error ? e.message : k.loadFail))
      .finally(() => setBusy(false))
  }

  const revoke = async (id: number) => {
    if (busy) return
    if (!(await confirmDialog(t.pages.profile.apiKey.revokeConfirm, { danger: true }))) return
    setBusy(true)
    apiFetch(`/api-keys/${id}`, { method: 'DELETE' })
      .then(load)
      .catch((e) => setError(e instanceof Error ? e.message : k.loadFail))
      .finally(() => setBusy(false))
  }

  const headRow = 'grid min-w-[620px] grid-cols-[1.2fr_1fr_1fr_.7fr_70px] items-center gap-4 px-4 py-3 text-[13px]'
  return (
    <div className={PAGE}>
      <div className="flex items-start justify-between gap-5 border-b border-[var(--shell-side-border)] pb-[18px]">
        <div><h1 className="m-0 mb-1.5 text-lg font-semibold text-[var(--shell-heading)]">{k.title}</h1><p className="m-0 text-[13px] text-[var(--shell-content-text)]">{k.desc}</p></div>
        <ToolbarButton primary onClick={() => { setPlainKey(''); setModalOpen(true) }}>{k.create}</ToolbarButton>
      </div>
      {denied ? <div className={TIP}>{k.denied}</div> : error ? <div className={TIP}>{error}</div> : (
        <div className="mt-6 overflow-x-auto border border-[var(--shell-side-border)]">
          <div className={headRow + ' bg-[var(--shell-menu-hover-bg)] text-xs text-[var(--shell-crumb-text)]'}>
            <span>{k.name}</span><span>{k.key}</span><span>{k.lastUsed}</span><span>{k.status}</span><span />
          </div>
          {rows.length === 0 && <div className={headRow + ' min-h-14 text-[var(--shell-heading)]'}><strong className="font-medium">{k.empty}</strong><span /><span /><span /><span /></div>}
          {rows.map((r) => (
            <div key={r.id} className={headRow + ' min-h-14 border-t border-[var(--shell-side-border)] text-[var(--shell-heading)]'}>
              <strong className="text-[13px] font-medium">{r.name}</strong>
              <span className="text-[var(--shell-content-text)]">{r.keyPrefix ? `${r.keyPrefix}…` : '—'}</span>
              <span className="text-[var(--shell-content-text)]">{r.lastUsedAt ? r.lastUsedAt : k.neverUsed}</span>
              {r.status === 1 ? <Badge variant="success">{k.active}</Badge> : <span className="text-[var(--shell-content-text)]">{t.pages.apikey.revoked}</span>}
              {r.status === 1 ? <button className="cursor-pointer border-0 bg-transparent p-0 text-xs text-[var(--color-danger)]" disabled={busy} onClick={() => revoke(r.id)}>{k.revoke}</button> : <span />}
            </div>
          ))}
        </div>)}
      <div className={TIP}>{k.securityTip}</div>
      {plainKey && (
        <div className="mt-4 flex items-center gap-3 border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] px-4 py-3">
          <div className="text-xs text-[var(--shell-content-text)]">{t.pages.apikey.plainOnce}</div>
          <code className="min-w-0 flex-1 break-all rounded-sm bg-black/5 px-2 py-1.5 text-xs">{plainKey}</code>
          <ToolbarButton onClick={() => setPlainKey('')}>{t.pages.company.cancel}</ToolbarButton>
        </div>
      )}
      {modalOpen && (
        <div className="fixed inset-0 z-50 grid place-items-center bg-black/45 p-5">
          <div className="w-[min(440px,100%)] rounded-md bg-[var(--shell-card-bg)] p-6 shadow-[var(--shadow-panel)]" role="dialog" aria-modal="true">
            <div className="mb-5 flex items-center justify-between">
              <h2 className="m-0 text-[17px] text-[var(--shell-heading)]">{k.create}</h2>
              <button aria-label={t.pages.profile.cancel} className="cursor-pointer border-0 bg-transparent p-0 text-sm text-[var(--shell-crumb-text)]" onClick={() => setModalOpen(false)}>×</button>
            </div>
            <label className={FORM_LABEL}>{k.name}<Input value={keyName} onChange={(event) => setKeyName(event.target.value)} placeholder={k.namePlaceholder} /></label>
            <div className="mt-6 flex justify-end gap-2.5">
              <ToolbarButton onClick={() => setModalOpen(false)}>{t.pages.profile.cancel}</ToolbarButton>
              <ToolbarButton primary disabled={busy || !keyName.trim()} onClick={create}>{k.create}</ToolbarButton>
            </div>
          </div>
        </div>
      )}
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

function AuditSection() {
  const t = useT()
  const a = t.pages.profile.audit
  const profile = useProfile()
  const [rows, setRows] = useState<AuditLog[]>([])
  const [error, setError] = useState('')
  const [denied, setDenied] = useState(false)

  const load = () => {
    setError(''); setDenied(false)
    apiFetch<{ items: AuditEntry[] }>('/audit-logs', { query: { accountId: profile.accountId, limit: 20 } })
      .then((d) => setRows((d?.items ?? []).map(toAuditLog)))
      .catch((e) => {
        if (e instanceof ApiError && e.code === 403) setDenied(true)
        else setError(e instanceof Error ? e.message : a.loadFail)
      })
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className={PAGE}>
      <SectionTitle title={a.title} desc={a.desc} />
      {error && <div className={TIP}>{error}</div>}
      {denied && <div className={TIP}>{a.loadFail}</div>}
      {!error && !denied && (
        <div className="border-t border-[var(--shell-side-border)]">
          {rows.length === 0 && <button className={LIST_BTN}><span className="grid gap-1"><strong className="text-[13px] font-medium">{a.empty}</strong><small className="text-xs text-[var(--shell-content-text)]">{a.emptyDesc}</small></span></button>}
          {rows.map((r) => (
            <button key={r.logId} className={LIST_BTN}>
              <span className="grid gap-1"><strong className="text-[13px] font-medium">{r.time}</strong><small className="text-xs text-[var(--shell-content-text)]">{r.type} · {r.action} · {r.ip}</small></span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function Summary({ value, label }: { value: string; label: string }) {
  return (
    <div className="border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] p-4">
      <strong className="block text-2xl font-semibold text-[var(--shell-heading)]">{value}</strong>
      <span className="mt-1.5 block text-xs text-[var(--shell-content-text)]">{label}</span>
    </div>
  )
}
