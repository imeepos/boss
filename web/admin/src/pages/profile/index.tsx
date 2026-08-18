// 用户中心：采用 Ant Design Pro Account Settings 的分区工作区模式。
import { useState, type FormEvent } from 'react'
import type { Profile } from '../../api/auth'
import { useT } from '../../i18n'
import './profile.css'

type Section = 'personal' | 'security' | 'apiKey' | 'myData'
interface ProfilePageProps { profile: Profile }

export default function ProfilePage({ profile }: ProfilePageProps) {
  const t = useT()
  const [section, setSection] = useState<Section>('personal')
  const [saved, setSaved] = useState('')
  const [passwordSaved, setPasswordSaved] = useState('')
  const [keyName, setKeyName] = useState('')
  const [modalOpen, setModalOpen] = useState(false)

  const submitPersonal = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSaved(t.pages.profile.personal.saved)
  }

  const submitPassword = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setPasswordSaved(t.pages.profile.password.passwordPending)
  }

  const navItems: Array<[Section, string, string]> = [
    ['personal', t.pages.profile.navigation.personal, t.pages.profile.navigation.personalDesc],
    ['security', t.pages.profile.navigation.security, t.pages.profile.navigation.securityDesc],
    ['apiKey', t.pages.profile.navigation.apiKey, t.pages.profile.navigation.apiKeyDesc],
    ['myData', t.pages.profile.navigation.myData, t.pages.profile.navigation.myDataDesc],
  ]

  return (
    <div className="profile-page">
      <header className="profile-page-header">
        <div className="profile-avatar">{profile.realName.slice(0, 1)}</div>
        <div className="profile-heading">
          <h1>{t.pages.profile.title}</h1>
          <div className="profile-heading-name">{profile.realName}<span>@{profile.username}</span></div>
          <div className="profile-heading-meta">{profile.roleName} · {profile.legalEntityName || t.pages.profile.personal.unassigned} · {profile.regionScope || t.pages.profile.personal.allScope}</div>
        </div>
        <span className="profile-account-status">{t.pages.profile.activeAccount}</span>
      </header>

      <div className="profile-workspace">
        <nav className="profile-settings-nav" aria-label={t.pages.profile.navigation.title}>
          <div className="profile-settings-nav-title">{t.pages.profile.navigation.title}</div>
          {navItems.map(([key, label, desc]) => (
            <button key={key} className={section === key ? 'active' : ''} onClick={() => setSection(key)}>
              <strong>{label}</strong><span>{desc}</span>
            </button>
          ))}
        </nav>

        <main className="profile-content-panel">
          {section === 'personal' && <PersonalSection profile={profile} saved={saved} onSubmit={submitPersonal} />}
          {section === 'security' && <SecuritySection saved={passwordSaved} onSubmit={submitPassword} />}
          {section === 'apiKey' && <ApiKeySection keyName={keyName} setKeyName={setKeyName} modalOpen={modalOpen} setModalOpen={setModalOpen} />}
          {section === 'myData' && <MyDataSection />}
        </main>
      </div>
    </div>
  )
}

function SectionTitle({ title, desc }: { title: string; desc: string }) {
  return <div className="profile-section-title"><h2>{title}</h2><p>{desc}</p></div>
}

function PersonalSection({ profile, saved, onSubmit }: { profile: Profile; saved: string; onSubmit: (event: FormEvent<HTMLFormElement>) => void }) {
  const t = useT()
  return <>
    <SectionTitle title={t.pages.profile.personal.title} desc={t.pages.profile.personal.desc} />
    <form className="profile-form profile-personal-form" onSubmit={onSubmit}>
      <label>{t.pages.profile.personal.username}<input value={profile.username} readOnly /></label>
      <label>{t.pages.profile.personal.realName}<input defaultValue={profile.realName} /></label>
      <label>{t.pages.profile.personal.phone}<input placeholder={t.pages.profile.personal.phonePlaceholder} /></label>
      <label>{t.pages.profile.personal.email}<input type="email" placeholder={t.pages.profile.personal.emailPlaceholder} /></label>
      <div className="profile-readonly-grid">
        <ReadOnlyField label={t.pages.profile.personal.role} value={profile.roleName} />
        <ReadOnlyField label={t.pages.profile.personal.company} value={profile.legalEntityName || t.pages.profile.personal.unassigned} />
        <ReadOnlyField label={t.pages.profile.personal.dataScope} value={profile.regionScope || t.pages.profile.personal.allScope} />
      </div>
      <div className="profile-form-actions"><span>{saved}</span><button type="submit">{t.pages.profile.save}</button></div>
    </form>
  </>
}

function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return <div className="profile-readonly-field"><span>{label}</span><strong>{value}</strong></div>
}

function SecuritySection({ saved, onSubmit }: { saved: string; onSubmit: (event: FormEvent<HTMLFormElement>) => void }) {
  const t = useT()
  return <>
    <SectionTitle title={t.pages.profile.password.title} desc={t.pages.profile.password.desc} />
    <form className="profile-form profile-password-form" onSubmit={onSubmit}>
      <label>{t.pages.profile.password.old}<input type="password" required /></label>
      <label>{t.pages.profile.password.next}<input type="password" required minLength={6} /></label>
      <label>{t.pages.profile.password.confirm}<input type="password" required minLength={6} /></label>
      <div className="profile-form-actions"><span>{saved}</span><button type="submit">{t.pages.profile.password.submit}</button></div>
    </form>
    <div className="profile-subsection"><h3>{t.pages.profile.securityProtection.title}</h3><SecurityRow label={t.pages.profile.securityProtection.loginProtection} value={t.pages.profile.securityProtection.pending} /><SecurityRow label={t.pages.profile.securityProtection.loginHistory} value={t.pages.profile.securityProtection.pending} /></div>
  </>
}

function SecurityRow({ label, value }: { label: string; value: string }) {
  return <div className="profile-security-row"><span>{label}</span><em>{value}</em><button aria-label={label}><span className="profile-chevron" /></button></div>
}

function ApiKeySection({ keyName, setKeyName, modalOpen, setModalOpen }: { keyName: string; setKeyName: (value: string) => void; modalOpen: boolean; setModalOpen: (value: boolean) => void }) {
  const t = useT()
  return <>
    <div className="profile-section-title profile-section-title-action"><div><h2>{t.pages.profile.apiKey.title}</h2><p>{t.pages.profile.apiKey.desc}</p></div><button onClick={() => setModalOpen(true)}>{t.pages.profile.apiKey.create}</button></div>
    <div className="profile-key-table"><div className="profile-key-table-head"><span>{t.pages.profile.apiKey.name}</span><span>{t.pages.profile.apiKey.key}</span><span>{t.pages.profile.apiKey.lastUsed}</span><span>{t.pages.profile.apiKey.status}</span><span /></div><div className="profile-key-table-row"><strong>ci-pipeline-prod</strong><span>boss_••••••••</span><span>{t.pages.profile.apiKey.neverUsed}</span><span className="profile-key-active">{t.pages.profile.apiKey.active}</span><button className="profile-danger">{t.pages.profile.apiKey.revoke}</button></div></div>
    <div className="profile-security-tip">{t.pages.profile.apiKey.securityTip}</div>
    {modalOpen && <div className="profile-modal-backdrop"><div className="profile-modal" role="dialog" aria-modal="true"><div className="profile-modal-head"><h3>{t.pages.profile.apiKey.create}</h3><button aria-label={t.pages.profile.cancel} onClick={() => setModalOpen(false)}><span className="profile-close-icon" /></button></div><label>{t.pages.profile.apiKey.name}<input value={keyName} onChange={(event) => setKeyName(event.target.value)} placeholder={t.pages.profile.apiKey.namePlaceholder} /></label><div className="profile-modal-actions"><button className="profile-secondary" onClick={() => setModalOpen(false)}>{t.pages.profile.cancel}</button><button onClick={() => setModalOpen(false)}>{t.pages.profile.apiKey.create}</button></div></div></div>}
  </>
}

function MyDataSection() {
  const t = useT()
  const items = [
    [t.pages.profile.myData.orders, t.pages.profile.myData.ordersDesc],
    [t.pages.profile.myData.bills, t.pages.profile.myData.billsDesc],
    [t.pages.profile.myData.service, t.pages.profile.myData.serviceDesc],
    [t.pages.profile.myData.messages, t.pages.profile.myData.messagesDesc],
    [t.pages.profile.myData.audit, t.pages.profile.myData.auditDesc],
    [t.pages.profile.myData.permissions, t.pages.profile.myData.permissionsDesc],
  ]
  return <><SectionTitle title={t.pages.profile.myData.title} desc={t.pages.profile.myData.desc} /><div className="profile-summary-grid"><Summary value="—" label={t.pages.profile.myData.orders} /><Summary value="—" label={t.pages.profile.myData.messages} /><Summary value="—" label={t.pages.profile.myData.bills} /></div><div className="profile-data-list">{items.map(([label, desc]) => <button key={label}><span><strong>{label}</strong><small>{desc}</small></span><b className="profile-chevron" /></button>)}</div></>
}

function Summary({ value, label }: { value: string; label: string }) {
  return <div className="profile-summary"><strong>{value}</strong><span>{label}</span></div>
}
