// 平台用户工作台独立壳层：视觉沿用 Admin Shell，内容聚焦员工工作与权限。
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useState } from 'react'
import type { Profile } from '../api/auth'
import { adminLogout } from '../api/auth'
import { useT } from '../i18n'
import { useTheme } from '../theme/context'
import { ProfileContext } from './profile'
import { MaskIcon, MenuIcon, MoonIcon, SunIcon } from './icons'
import logoMark from '../assets/brand/logo-mark-navy.png'
import './ucenter.css'

export function UCenterLayout({ profile }: { profile: Profile }) {
  const t = useT()
  const nav = useNavigate()
  const { pathname } = useLocation()
  const { theme, toggleTheme } = useTheme()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const items = [
    ['overview', t.pages.profile.navigation.overview, t.pages.profile.navigation.overviewDesc, 'dashboard'],
    ['personal', t.pages.profile.navigation.personal, t.pages.profile.navigation.personalDesc, 'profile'],
    ['security', t.pages.profile.navigation.security, t.pages.profile.navigation.securityDesc, 'security'],
    ['permissions', t.pages.profile.navigation.permissions, t.pages.profile.navigation.permissionsDesc, 'security'],
    ['work', t.pages.profile.navigation.work, t.pages.profile.navigation.workDesc, 'data'],
    ['api-keys', t.pages.profile.navigation.apiKey, t.pages.profile.navigation.apiKeyDesc, 'key'],
    ['audit', t.pages.profile.navigation.audit, t.pages.profile.navigation.auditDesc, 'data'],
  ]
  const logout = async () => { await adminLogout(); nav('/login', { replace: true }) }
  return (
    <ProfileContext.Provider value={profile}>
      <div className="ucenter-shell">
        <header className="ucenter-topbar">
          <button className="ucenter-drawer-button" aria-label={t.pages.profile.navigation.menu} onClick={() => setSidebarOpen((value) => !value)}><MenuIcon size={20} /></button>
          <Link to="/ucenter/overview" className="ucenter-brand"><img src={logoMark} alt="Sphere Boss" /><span>Sphere Boss</span></Link>
          <div className="ucenter-topbar-title">{t.pages.profile.navigation.platformWorkspace}</div>
          <div className="ucenter-topbar-actions">
            <Link to="/dashboard" className="ucenter-back">{t.pages.profile.navigation.backAdmin}</Link>
            <button className="ucenter-tool-button" onClick={toggleTheme} aria-label={theme === 'dark' ? t.shell.themeToLight : t.shell.themeToDark}>{theme === 'dark' ? <SunIcon /> : <MoonIcon />}</button>
            <div className="ucenter-user"><span className="ucenter-avatar">{profile.realName.slice(0, 1)}</span><span className="ucenter-user-name">{profile.realName}</span><button onClick={() => void logout()}>{t.common.logout}</button></div>
          </div>
        </header>
        <div className="ucenter-body">
          <aside className={sidebarOpen ? 'ucenter-sidebar open' : 'ucenter-sidebar'}>
            <div className="ucenter-sidebar-heading"><span className="ucenter-avatar large">{profile.realName.slice(0, 1)}</span><div><strong>{profile.realName}</strong><small>@{profile.username}</small></div></div>
            <nav aria-label={t.pages.profile.navigation.title}>
              <div className="ucenter-nav-title">{t.pages.profile.navigation.title}</div>
              {items.map(([key, label, desc, icon]) => <NavLink key={key} to={`/ucenter/${key}`} onClick={() => setSidebarOpen(false)} className={({ isActive }) => isActive ? 'active' : ''}><MaskIcon url={`/icons/items/${icon}.svg`} size={16} /><span><strong>{label}</strong><small>{desc}</small></span></NavLink>)}
            </nav>
            <div className="ucenter-sidebar-foot"><span>{profile.roleName}</span><span>{t.pages.profile.personal.dataScope}: {profile.regionScope || t.pages.profile.personal.allScope}</span></div>
          </aside>
          <main className="ucenter-main"><div className="ucenter-breadcrumb"><Link to="/ucenter/overview">{t.pages.profile.title}</Link><span>/</span><strong>{items.find(([key]) => pathname.endsWith(`/${key}`))?.[1] ?? t.pages.profile.navigation.overview}</strong></div><Outlet /></main>
        </div>
      </div>
    </ProfileContext.Provider>
  )
}
