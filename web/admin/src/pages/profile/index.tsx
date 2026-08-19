// 个人工作台内容页：各分区由 UCenterLayout 的独立路由承载。
import { useEffect, useState, type FormEvent } from 'react'
import { useLocation } from 'react-router-dom'
import { useProfile } from '../../layouts/profile'
import type { Profile } from '../../api/auth'
import { apiFetch } from '../../api/client'
import { ApiError } from '../../api/envelope'
import { toAuditLog, type AuditEntry, type AuditLog } from '../base/audit/logic'
import { useT } from '../../i18n'
import './profile.css'

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
  return <div className="profile-section-title"><h1>{title}</h1><p>{desc}</p></div>
}

function OverviewSection({ profile }: { profile: Profile }) {
  const t = useT()
  return <div className="profile-content-page"><SectionTitle title={t.pages.profile.overview.title} desc={t.pages.profile.overview.desc} /><div className="overview-grid"><div className="overview-identity"><span className="overview-avatar">{profile.realName.slice(0, 1)}</span><div><strong>{profile.realName}</strong><span>@{profile.username}</span><small>{profile.roleName} · {profile.legalEntityName || t.pages.profile.personal.unassigned}</small></div></div><div className="overview-status"><span>{t.pages.profile.overview.accountStatus}</span><strong>{t.pages.profile.activeAccount}</strong><small>{t.pages.profile.overview.accountStatusDesc}</small></div></div><div className="overview-section"><h2>{t.pages.profile.overview.quickAccess}</h2><div className="overview-links"><OverviewLink title={t.pages.profile.overview.dispatch} desc={t.pages.profile.overview.dispatchDesc} href="/boss/dispatch" /><OverviewLink title={t.pages.profile.overview.order} desc={t.pages.profile.overview.orderDesc} href="/boss/order" /><OverviewLink title={t.pages.profile.overview.complaint} desc={t.pages.profile.overview.complaintDesc} href="/boss/complaint" /></div></div><div className="overview-summary"><div><span>{t.pages.profile.overview.permissionSummary}</span><strong>{profile.roleName}</strong></div><div><span>{t.pages.profile.overview.dataScopeSummary}</span><strong>{profile.regionScope || t.pages.profile.personal.allScope}</strong></div></div></div>
}

function OverviewLink({ title, desc, href }: { title: string; desc: string; href: string }) {
  return <a className="overview-link" href={href}><strong>{title}</strong><span>{desc}</span><i className="profile-chevron" /></a>
}

function PersonalSection({ profile }: { profile: Profile }) {
  const t = useT(); const [saved, setSaved] = useState('')
  const submit = (event: FormEvent<HTMLFormElement>) => { event.preventDefault(); setSaved(t.pages.profile.personal.saved) }
  return <div className="profile-content-page"><SectionTitle title={t.pages.profile.personal.title} desc={t.pages.profile.personal.desc} /><form className="profile-form profile-personal-form" onSubmit={submit}><label>{t.pages.profile.personal.username}<input value={profile.username} readOnly /></label><label>{t.pages.profile.personal.realName}<input defaultValue={profile.realName} /></label><label>{t.pages.profile.personal.phone}<input placeholder={t.pages.profile.personal.phonePlaceholder} /></label><label>{t.pages.profile.personal.email}<input type="email" placeholder={t.pages.profile.personal.emailPlaceholder} /></label><div className="profile-readonly-grid"><ReadOnlyField label={t.pages.profile.personal.role} value={profile.roleName} /><ReadOnlyField label={t.pages.profile.personal.company} value={profile.legalEntityName || t.pages.profile.personal.unassigned} /><ReadOnlyField label={t.pages.profile.personal.dataScope} value={profile.regionScope || t.pages.profile.personal.allScope} /></div><div className="profile-form-actions"><span>{saved}</span><button type="submit">{t.pages.profile.save}</button></div></form></div>
}

function ReadOnlyField({ label, value }: { label: string; value: string }) { return <div className="profile-readonly-field"><span>{label}</span><strong>{value}</strong></div> }

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
  return <div className="profile-content-page"><SectionTitle title={p.title} desc={p.desc} /><form className="profile-form profile-password-form" onSubmit={submit}><label>{p.old}<input type="password" required value={oldPw} onChange={(e) => setOldPw(e.target.value)} /></label><label>{p.next}<input type="password" required minLength={6} value={newPw} onChange={(e) => setNewPw(e.target.value)} /></label><label>{p.confirm}<input type="password" required minLength={6} value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)} /></label><div className="profile-form-actions"><span style={ok ? { color: '#30a46c' } : { color: '#e5484d' }}>{msg}</span><button type="submit" disabled={busy}>{p.submit}</button></div></form><div className="profile-subsection"><h2>{t.pages.profile.securityProtection.title}</h2><SecurityRow label={t.pages.profile.securityProtection.loginProtection} value={t.pages.profile.securityProtection.pending} /><SecurityRow label={t.pages.profile.securityProtection.loginHistory} value={t.pages.profile.securityProtection.pending} /></div></div>
}

function SecurityRow({ label, value }: { label: string; value: string }) { return <div className="profile-security-row"><span>{label}</span><em>{value}</em><button aria-label={label}><i className="profile-chevron" /></button></div> }

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

  const revoke = (id: number) => {
    if (busy) return
    setBusy(true)
    apiFetch(`/api-keys/${id}`, { method: 'DELETE' })
      .then(load)
      .catch((e) => setError(e instanceof Error ? e.message : k.loadFail))
      .finally(() => setBusy(false))
  }

  return <div className="profile-content-page"><div className="profile-section-title profile-section-title-action"><div><h1>{k.title}</h1><p>{k.desc}</p></div><button onClick={() => { setPlainKey(''); setModalOpen(true) }}>{k.create}</button></div>
    {denied ? <div className="profile-security-tip">{k.denied}</div> : error ? <div className="profile-security-tip">{error}</div> : (
      <div className="profile-key-table"><div className="profile-key-table-head"><span>{k.name}</span><span>{k.key}</span><span>{k.lastUsed}</span><span>{k.status}</span><span /></div>
        {rows.length === 0 && <div className="profile-key-table-row"><strong>{k.empty}</strong><span /><span /><span /><span /></div>}
        {rows.map((r) => <div key={r.id} className="profile-key-table-row"><strong>{r.name}</strong><span>{r.keyPrefix ? `${r.keyPrefix}…` : '—'}</span><span>{r.lastUsedAt ? r.lastUsedAt : k.neverUsed}</span><span className={r.status === 1 ? 'profile-key-active' : ''}>{r.status === 1 ? k.active : t.pages.apikey.revoked}</span>{r.status === 1 ? <button className="profile-danger" disabled={busy} onClick={() => revoke(r.id)}>{k.revoke}</button> : <span />}</div>)}
      </div>)}
    <div className="profile-security-tip">{k.securityTip}</div>
    {plainKey && <div className="apikey-plain-banner"><div>{t.pages.apikey.plainOnce}</div><code className="mono">{plainKey}</code><button onClick={() => setPlainKey('')}>{t.pages.company.cancel}</button></div>}
    {modalOpen && <div className="profile-modal-backdrop"><div className="profile-modal" role="dialog" aria-modal="true"><div className="profile-modal-head"><h2>{k.create}</h2><button aria-label={t.pages.profile.cancel} onClick={() => setModalOpen(false)}><i className="profile-close-icon" /></button></div><label>{k.name}<input value={keyName} onChange={(event) => setKeyName(event.target.value)} placeholder={k.namePlaceholder} /></label><div className="profile-modal-actions"><button className="profile-secondary" onClick={() => setModalOpen(false)}>{t.pages.profile.cancel}</button><button disabled={busy || !keyName.trim()} onClick={create}>{k.create}</button></div></div></div>}
  </div>
}

function MyDataSection() {
  const t = useT(); const items = [[t.pages.profile.myData.orders, t.pages.profile.myData.ordersDesc], [t.pages.profile.myData.bills, t.pages.profile.myData.billsDesc], [t.pages.profile.myData.service, t.pages.profile.myData.serviceDesc], [t.pages.profile.myData.messages, t.pages.profile.myData.messagesDesc], [t.pages.profile.myData.audit, t.pages.profile.myData.auditDesc], [t.pages.profile.myData.permissions, t.pages.profile.myData.permissionsDesc]]
  return <div className="profile-content-page"><SectionTitle title={t.pages.profile.myData.title} desc={t.pages.profile.myData.desc} /><div className="profile-summary-grid"><Summary value="—" label={t.pages.profile.myData.orders} /><Summary value="—" label={t.pages.profile.myData.messages} /><Summary value="—" label={t.pages.profile.myData.bills} /></div><div className="profile-data-list">{items.map(([label, desc]) => <button key={label}><span><strong>{label}</strong><small>{desc}</small></span><i className="profile-chevron" /></button>)}</div></div>
}

function PermissionsSection({ profile }: { profile: Profile }) {
  const t = useT()
  return <div className="profile-content-page"><SectionTitle title={t.pages.profile.permissions.title} desc={t.pages.profile.permissions.desc} /><div className="profile-summary-grid"><Summary value={profile.roleName} label={t.pages.profile.permissions.role} /><Summary value={profile.legalEntityName || t.pages.profile.personal.unassigned} label={t.pages.profile.permissions.company} /><Summary value={profile.regionScope || t.pages.profile.personal.allScope} label={t.pages.profile.permissions.scope} /></div></div>
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

  return <div className="profile-content-page"><SectionTitle title={a.title} desc={a.desc} />
    {error && <div className="profile-security-tip">{error}</div>}
    {denied && <div className="profile-security-tip">{a.loadFail}</div>}
    {!error && !denied && <div className="profile-data-list">
      {rows.length === 0 && <button><span><strong>{a.empty}</strong><small>{a.emptyDesc}</small></span></button>}
      {rows.map((r) => <button key={r.logId}><span><strong>{r.time}</strong><small>{r.type} · {r.action} · {r.ip}</small></span></button>)}
    </div>}
  </div>
}

function Summary({ value, label }: { value: string; label: string }) { return <div className="profile-summary"><strong>{value}</strong><span>{label}</span></div> }
