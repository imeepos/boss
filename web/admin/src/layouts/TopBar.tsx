// 顶栏:品牌区 + 分组主导航 + 搜索/主题/通知/语言/用户工具区。规格见 design-spec.md §2.1/§3.1。
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import type { Profile } from '../api/auth'
import { adminLogout } from '../api/auth'
import type { MenuGroup } from '../router/menu.def'
import logoMark from '../assets/brand/logo-mark-navy.png'
import { useT, useLang, localeOptions } from '../i18n'
import { useTheme } from '../theme/context'
import { BellIcon, CheckIcon, GlobeIcon, LogoutIcon, MaskIcon, MenuIcon, MoonIcon, SearchIcon, SunIcon, UserIcon } from './icons'

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
          <MaskIcon url={`/icons/${g.id}.svg`} size={16} />
          <span>{t.menu.groups[g.id] ?? g.label}</span>
        </button>
      ))}
    </nav>
  )
}

/** 顶栏搜索:默认仅一个工具按钮,点击展开输入框;清空失焦或 Esc 收起。 */
function SearchBox({ groups }: { groups: MenuGroup[] }) {
  const t = useT()
  const nav = useNavigate()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const expand = () => {
    setOpen(true)
    requestAnimationFrame(() => inputRef.current?.focus())
  }
  const collapse = () => {
    if (!query) setOpen(false)
  }
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
      inputRef.current?.blur()
    }
  }
  if (!open) {
    return (
      <button className="shell-tool-btn" onClick={expand} title={t.shell.searchPlaceholder} aria-label="search">
        <SearchIcon size={18} />
      </button>
    )
  }
  return (
    <form className="shell-search" onSubmit={onSubmit} role="search">
      <SearchIcon />
      <input
        ref={inputRef}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onBlur={collapse}
        onKeyDown={(e) => e.key === 'Escape' && (setQuery(''), setOpen(false))}
        placeholder={t.shell.searchPlaceholder}
        aria-label={t.shell.searchPlaceholder}
      />
    </form>
  )
}

/** 语言切换:地球图标按钮 + 自定义下拉浮层,风格与其余工具按钮一致。 */
function LangSwitch() {
  const { locale, setLocale } = useLang()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!open) return
    const onDocClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [open])
  return (
    <div className="shell-lang" ref={rootRef}>
      <button
        className={open ? 'shell-tool-btn active' : 'shell-tool-btn'}
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        title="Language"
      >
        <GlobeIcon />
      </button>
      {open && (
        <div className="shell-lang-menu" role="listbox">
          {localeOptions().map((opt) => (
            <button
              key={opt.value}
              role="option"
              aria-selected={opt.value === locale}
              className={opt.value === locale ? 'active' : ''}
              onClick={() => {
                setLocale(opt.value as typeof locale)
                setOpen(false)
              }}
            >
              <span>{opt.label}</span>
              {opt.value === locale && <CheckIcon />}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

/** 用户区:头像+姓名可点击,下拉含个人设置/退出登录(antd Pro 惯例,退出不常驻顶栏)。 */
function UserMenu({ profile }: { profile: Profile }) {
  const t = useT()
  const nav = useNavigate()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!open) return
    const onDocClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [open])
  const logout = async () => {
    setOpen(false)
    await adminLogout()
    nav('/login', { replace: true })
  }
  return (
    <div className="shell-user" ref={rootRef}>
      <button
        className={open ? 'shell-user-btn active' : 'shell-user-btn'}
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        title={`${profile.realName}(${profile.roleName})`}
      >
        <span className="shell-avatar">{profile.realName.slice(0, 1)}</span>
      </button>
      {open && (
        <div className="shell-user-menu" role="menu">
          <div className="shell-user-info">
            <div className="shell-user-info-name">{profile.realName}</div>
            <div className="shell-user-info-role">{profile.roleName}</div>
          </div>
          <button role="menuitem" onClick={() => setOpen(false)}>
            <UserIcon />
            <span>{t.common.profile}</span>
          </button>
          <button role="menuitem" className="danger" onClick={logout}>
            <LogoutIcon />
            <span>{t.common.logout}</span>
          </button>
        </div>
      )}
    </div>
  )
}

function RightTools({ profile }: { profile: Profile }) {
  const t = useT()
  const { theme, toggleTheme } = useTheme()
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
      <LangSwitch />
      <UserMenu profile={profile} />
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
