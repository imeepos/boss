// 个人工作台独立壳层：不复用 AdminLayout 的业务侧栏。
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import type { Profile } from '../api/auth'
import { adminLogout } from '../api/auth'
import { useT } from '../i18n'
import { useTheme } from '../theme/context'
import { ProfileContext } from './profile'
import { MoonIcon, SunIcon } from './icons'
import './ucenter.css'

export function UCenterLayout({ profile }: { profile: Profile }) {
  const t = useT()
  const nav = useNavigate()
  const { pathname } = useLocation()
  const { theme, toggleTheme } = useTheme()
  const items = [
    ['overview', t.pages.profile.navigation.overview, t.pages.profile.navigation.overviewDesc],
    ['personal', t.pages.profile.navigation.personal, t.pages.profile.navigation.personalDesc],
    ['security', t.pages.profile.navigation.security, t.pages.profile.navigation.securityDesc],
    ['api-keys', t.pages.profile.navigation.apiKey, t.pages.profile.navigation.apiKeyDesc],
    ['data', t.pages.profile.navigation.myData, t.pages.profile.navigation.myDataDesc],
  ]
  const logout = async () => { await adminLogout(); nav('/login', { replace: true }) }
  return (
    <ProfileContext.Provider value={profile}>
      <div className="ucenter-shell">
      <header className="ucenter-topbar">
        <Link to="/ucenter/overview" className="ucenter-brand"><span className="ucenter-brand-mark">S</span><span>Sphere Boss</span></Link>
        <div className="ucenter-topbar-actions">
          <Link to="/dashboard" className="ucenter-back">{t.pages.profile.navigation.backAdmin}</Link>
          <button onClick={toggleTheme} aria-label={theme === 'dark' ? t.shell.themeToLight : t.shell.themeToDark}>{theme === 'dark' ? <SunIcon /> : <MoonIcon />}</button>
          <div className="ucenter-user"><span className="ucenter-avatar">{profile.realName.slice(0, 1)}</span><span>{profile.realName}</span><button onClick={() => void logout()}>{t.common.logout}</button></div>
        </div>
      </header>
      <div className="ucenter-body">
        <aside className="ucenter-sidebar">
          <div className="ucenter-sidebar-heading"><span className="ucenter-avatar large">{profile.realName.slice(0, 1)}</span><div><strong>{profile.realName}</strong><small>@{profile.username}</small></div></div>
          <nav aria-label={t.pages.profile.navigation.title}>
            <div className="ucenter-nav-title">{t.pages.profile.navigation.title}</div>
            {items.map(([key, label, desc]) => <NavLink key={key} to={`/ucenter/${key}`} className={({ isActive }) => isActive ? 'active' : ''}><strong>{label}</strong><small>{desc}</small></NavLink>)}
          </nav>
          <div className="ucenter-sidebar-foot"><span>{profile.roleName}</span><span>{profile.regionScope || t.pages.profile.personal.allScope}</span></div>
        </aside>
        <main className="ucenter-main"><div className="ucenter-breadcrumb"><Link to="/ucenter/overview">{t.pages.profile.title}</Link><span>/</span><strong>{items.find(([key]) => pathname.endsWith(`/${key}`))?.[1] ?? t.pages.profile.navigation.overview}</strong></div><Outlet /></main>
      </div>
      </div>
    </ProfileContext.Provider>
  )
}
