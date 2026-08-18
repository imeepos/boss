// 顶栏:品牌区 + 分组主导航 + 搜索/主题/通知/语言/用户工具区。规格见 design-spec.md §2.1/§3.1。
import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import type { Profile } from '../api/auth'
import { adminLogout } from '../api/auth'
import type { MenuGroup } from '../router/menu.def'
import logoMark from '../assets/brand/logo-mark-navy.png'
import { useT, useLang, localeOptions } from '../i18n'
import { useTheme } from '../theme/context'
import { BellIcon, MenuIcon, MoonIcon, SearchIcon, SunIcon } from './icons'

interface TopBarProps {
  profile: Profile
  groups: MenuGroup[]
  activeGroupId?: string
  onOpenDrawer: () => void
}

function TopNav({ groups, activeGroupId }: { groups: MenuGroup[]; activeGroupId?: string }) {
  const t = useT()
  const nav = useNavigate()
  return (
    <nav className="shell-topnav">
      {groups.map((g) => (
        <button
          key={g.id}
          className={g.id === activeGroupId ? 'shell-topnav-item active' : 'shell-topnav-item'}
          onClick={() => nav(g.items[0].path)}
        >
          {t.menu.groups[g.id] ?? g.label}
        </button>
      ))}
    </nav>
  )
}

function SearchBox({ groups }: { groups: MenuGroup[] }) {
  const t = useT()
  const nav = useNavigate()
  const [query, setQuery] = useState('')
  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    const q = query.trim().toLowerCase()
    if (!q) return
    const hit = groups
      .flatMap((g) => g.items)
      .find((it) => (t.menu.items[it.key] ?? it.label).toLowerCase().includes(q))
    if (hit) {
      nav(hit.path)
      setQuery('')
    }
  }
  return (
    <form className="shell-search" onSubmit={onSubmit} role="search">
      <SearchIcon />
      <input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={t.shell.searchPlaceholder}
        aria-label={t.shell.searchPlaceholder}
      />
    </form>
  )
}

function RightTools({ profile }: { profile: Profile }) {
  const t = useT()
  const nav = useNavigate()
  const { locale, setLocale } = useLang()
  const { theme, toggleTheme } = useTheme()
  const logout = async () => {
    await adminLogout()
    nav('/login', { replace: true })
  }
  return (
    <div className="shell-tools">
      <button
        className="shell-tool-btn"
        onClick={toggleTheme}
        title={theme === 'dark' ? t.shell.themeToLight : t.shell.themeToDark}
      >
        {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
      </button>
      <button className="shell-tool-btn" title={t.shell.notifications}>
        <BellIcon />
      </button>
      <select
        className="shell-lang"
        value={locale}
        onChange={(e) => setLocale(e.target.value as typeof locale)}
      >
        {localeOptions().map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      <span className="shell-user" title={`${profile.realName}(${profile.roleName})`}>
        <span className="shell-avatar">{profile.realName.slice(0, 1)}</span>
        <span className="shell-user-name">{profile.realName}</span>
      </span>
      <button className="shell-logout" onClick={logout}>
        {t.common.logout}
      </button>
    </div>
  )
}

export function TopBar({ profile, groups, activeGroupId, onOpenDrawer }: TopBarProps) {
  return (
    <header className="shell-topbar">
      <button className="shell-drawer-btn" onClick={onOpenDrawer} aria-label="menu">
        <MenuIcon />
      </button>
      <Link to="/dashboard" className="shell-brand">
        <img src={logoMark} alt="Sphere Boss" />
        <span>Sphere Boss</span>
      </Link>
      <TopNav groups={groups} activeGroupId={activeGroupId} />
      <div className="shell-topbar-right">
        <SearchBox groups={groups} />
        <RightTools profile={profile} />
      </div>
    </header>
  )
}
