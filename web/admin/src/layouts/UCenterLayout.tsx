// 平台用户工作台独立壳层：视觉沿用 Admin Shell，内容聚焦员工工作与权限。
// 样式:tailwind 原子类(原 ucenter.css 已删除),令牌走 shell-* 体系。
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Suspense, useState } from 'react'
import { Loading } from '../components/Loading'
import type { Profile } from '../api/auth'
import { adminLogout } from '../api/auth'
import { useT } from '../i18n'
import { useTheme } from '../theme/context'
import { ProfileContext } from './profile'
import { MaskIcon, MenuIcon, MoonIcon, SunIcon } from './icons'
import logoMark from '../assets/brand/logo-mark-navy.png'

const TOPBAR_BTN = 'grid h-8 w-8 flex-none cursor-pointer place-items-center border-0 bg-transparent text-[var(--shell-nav-text)] hover:bg-[var(--shell-search-bg)] hover:text-[var(--shell-nav-active)]'
const AVATAR = 'grid place-items-center rounded-full bg-[var(--color-brand-gold-300)] font-semibold text-[var(--color-brand-navy-950)]'
const NAV_LINK = 'flex min-h-12 items-center gap-2.5 border-l-[3px] border-l-transparent px-4 py-2 text-[var(--shell-menu-text)] no-underline hover:bg-[var(--shell-menu-hover-bg)]'
const NAV_ACTIVE = 'border-l-[var(--color-brand-gold-500)] bg-[var(--shell-menu-active-bg)] text-[var(--shell-menu-active-text)]'
const SIDEBAR_MOBILE = 'max-[760px]:absolute max-[760px]:inset-y-0 max-[760px]:left-0 max-[760px]:top-14 max-[760px]:z-20 max-[760px]:transition-transform max-[760px]:duration-200'

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
      <div className="flex h-screen flex-col overflow-hidden bg-[var(--shell-content-bg)] font-base text-[var(--shell-content-text)]">
        <header className="flex h-14 flex-none items-center gap-4 border-b border-[var(--shell-topbar-border)] bg-[var(--shell-topbar-bg)] px-4 text-white shadow-[var(--shell-topbar-shadow)] max-[760px]:px-3">
          <button className={TOPBAR_BTN + ' hidden max-[760px]:grid'} aria-label={t.pages.profile.navigation.menu} onClick={() => setSidebarOpen((value) => !value)}><MenuIcon size={20} /></button>
          <Link to="/ucenter/overview" className="flex flex-none items-center gap-2.5 pr-2 font-brand text-[17px] font-bold whitespace-nowrap text-white no-underline"><img className="h-7 w-7 rounded-full bg-white p-px" src={logoMark} alt="Sphere Boss" /><span>Sphere Boss</span></Link>
          <div className="text-sm text-[var(--shell-nav-text)] max-[760px]:hidden">{t.pages.profile.navigation.platformWorkspace}</div>
          <div className="ml-auto flex items-center gap-3">
            <Link to="/dashboard" className="text-[13px] text-[var(--shell-nav-text)] no-underline hover:text-[var(--shell-nav-active)] max-[760px]:hidden">{t.pages.profile.navigation.backAdmin}</Link>
            <button className={TOPBAR_BTN} onClick={toggleTheme} aria-label={theme === 'dark' ? t.shell.themeToLight : t.shell.themeToDark}>{theme === 'dark' ? <SunIcon /> : <MoonIcon />}</button>
            <div className="flex items-center gap-3 border-l border-white/20 pl-3 text-[13px] text-[var(--shell-nav-active)]"><span className={AVATAR + ' h-7 w-7 text-[13px]'}>{profile.realName.slice(0, 1)}</span><span className="max-[760px]:hidden">{profile.realName}</span><button className="cursor-pointer border-0 bg-transparent px-0 text-xs text-[var(--color-brand-gold-300)]" onClick={() => void logout()}>{t.common.logout}</button></div>
          </div>
        </header>
        <div className="relative flex min-h-0 flex-1">
          <aside className={'flex w-60 flex-none flex-col overflow-y-auto border-r border-[var(--shell-side-border)] bg-[var(--shell-side-bg)] ' + SIDEBAR_MOBILE + (sidebarOpen ? '' : ' max-[760px]:-translate-x-full')}>
            <div className="flex items-center gap-2.5 border-b border-[var(--shell-side-border)] px-4 py-[18px]"><span className={AVATAR + ' h-10 w-10 text-[17px]'}>{profile.realName.slice(0, 1)}</span><div className="grid min-w-0 gap-1"><strong className="overflow-hidden text-sm text-[var(--shell-heading)] truncate">{profile.realName}</strong><small className="overflow-hidden text-xs text-[var(--shell-crumb-text)] truncate">@{profile.username}</small></div></div>
            <nav aria-label={t.pages.profile.navigation.title} className="py-4">
              <div className="px-4 pb-[9px] text-xs font-semibold text-[var(--shell-group-title)]">{t.pages.profile.navigation.title}</div>
              {items.map(([key, label, desc, icon]) => <NavLink key={key} to={`/ucenter/${key}`} onClick={() => setSidebarOpen(false)} className={({ isActive }) => isActive ? NAV_LINK + ' ' + NAV_ACTIVE : NAV_LINK}><MaskIcon url={`/icons/items/${icon}.svg`} size={16} /><span className="grid min-w-0 gap-[3px]"><strong className="text-sm font-medium">{label}</strong><small className="overflow-hidden text-xs text-[var(--shell-crumb-text)] truncate">{desc}</small></span></NavLink>)}
            </nav>
            <div className="mt-auto grid gap-[7px] border-t border-[var(--shell-side-border)] p-4 text-xs text-[var(--shell-crumb-text)]"><span>{profile.roleName}</span><span>{t.pages.profile.personal.dataScope}: {profile.regionScope || t.pages.profile.personal.allScope}</span></div>
          </aside>
          <main className="relative min-w-0 flex-1 overflow-y-auto bg-[var(--shell-content-bg)] px-6 pb-12 pt-4 max-[760px]:px-3 max-[760px]:pb-9"><div className="mb-4 flex gap-2 text-xs text-[var(--shell-crumb-text)]"><Link className="inherit no-underline" to="/ucenter/overview">{t.pages.profile.title}</Link><span>/</span><strong className="font-semibold text-[var(--shell-heading)]">{items.find(([key]) => pathname.endsWith(`/${key}`))?.[1] ?? t.pages.profile.navigation.overview}</strong></div><Suspense fallback={<Loading />}><Outlet /></Suspense></main>
        </div>
      </div>
    </ProfileContext.Provider>
  )
}
