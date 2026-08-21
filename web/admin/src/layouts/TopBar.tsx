// 顶栏:品牌区 + 分组主导航 + 搜索/主题/通知/语言/用户工具区。规格见 design-spec.md §2.1/§3.1。
// 样式:tailwind 原子类(原 shell.css 已删除)。
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import type { Profile } from '../api/auth'
import { adminLogout } from '../api/auth'
import type { MenuGroup } from '../router/menu.def'
import logoMark from '../assets/brand/logo-mark-navy.png'
import { useT, useLang, localeOptions } from '../i18n'
import { useTheme } from '../theme/context'
import { CheckIcon, GlobeIcon, LogoutIcon, MaskIcon, MenuIcon, MoonIcon, SearchIcon, SunIcon, UserIcon } from './icons'
import { NotifBell } from './NotifBell'
import { cn } from '../lib/cn'

const TOOL_BTN = 'grid h-[34px] w-[34px] cursor-pointer place-items-center rounded-full border-0 bg-none text-white/80 hover:bg-[var(--shell-search-bg-focus)] hover:text-white'
const MENU = 'absolute right-0 top-[calc(100%+8px)] z-50 rounded-[10px] border border-[var(--shell-side-border)] bg-[var(--shell-content-bg)] p-1 shadow-[var(--shell-fab-shadow)]'
const MENU_BTN = 'flex w-full cursor-pointer items-center justify-between gap-3 rounded-md border-0 bg-none px-2.5 py-2 text-left text-[13px] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'

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
    <nav className="flex h-full items-stretch gap-0.5 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
      {groups.map((g) => (
        <button
          key={g.id}
          className={cn(
            'relative inline-flex cursor-pointer items-center gap-1.5 whitespace-nowrap border-0 bg-none px-3.5 text-sm font-medium text-[var(--shell-nav-text)] hover:text-[var(--shell-nav-active)]',
            g.id === activeGroupId && 'text-[var(--shell-nav-active)] after:absolute after:right-3.5 after:bottom-0 after:left-3.5 after:h-[3px] after:rounded-t-sm after:bg-[var(--shell-nav-line)] after:content-[\'\']',
          )}
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
      <button className={TOOL_BTN} onClick={expand} title={t.shell.searchPlaceholder} aria-label="search">
        <SearchIcon size={18} />
      </button>
    )
  }
  return (
    <form className="flex h-[34px] items-center gap-2 rounded-[17px] bg-[var(--shell-search-bg)] px-3 text-white/60 transition-colors focus-within:bg-[var(--shell-search-bg-focus)]" onSubmit={onSubmit} role="search">
      <SearchIcon />
      <input
        ref={inputRef}
        className="w-45 border-0 bg-none text-[13px] text-white outline-none transition-[width] placeholder:text-white/45 max-[959px]:w-27"
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
    <div className="relative" ref={rootRef}>
      <button
        className={cn(TOOL_BTN, open && 'bg-[var(--shell-search-bg-focus)] text-white')}
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        title="Language"
      >
        <GlobeIcon />
      </button>
      {open && (
        <div className={MENU} role="listbox">
          {localeOptions().map((opt) => (
            <button
              key={opt.value}
              role="option"
              aria-selected={opt.value === locale}
              className={cn(MENU_BTN, opt.value === locale && 'font-semibold text-[var(--color-brand-gold-500)]')}
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
    <div className="relative flex items-center gap-2" ref={rootRef}>
      <button
        className={cn('flex cursor-pointer items-center gap-2 rounded-[17px] border-0 bg-none p-1 text-white transition-colors hover:bg-[var(--shell-search-bg-focus)]', open && 'bg-[var(--shell-search-bg-focus)]')}
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        title={`${profile.realName}(${profile.roleName})`}
      >
        <span className="grid h-8 w-8 place-items-center rounded-full bg-[var(--color-brand-gold-500)] text-[13px] font-semibold text-[var(--color-brand-navy-950)]">{profile.realName.slice(0, 1)}</span>
      </button>
      {open && (
        <div className={cn(MENU, 'min-w-45')} role="menu">
          <div className="mb-1 border-b border-[var(--shell-side-border)] px-2.5 pt-2 pb-2.5">
            <div className="text-sm font-semibold text-[var(--shell-heading)]">{profile.realName}</div>
            <div className="mt-0.5 text-xs text-[var(--shell-group-title)]">{profile.roleName}</div>
          </div>
          <button role="menuitem" className={cn(MENU_BTN, 'justify-start')} onClick={() => { setOpen(false); nav('/ucenter/overview') }}>
            <UserIcon />
            <span>{t.common.profile}</span>
          </button>
          <button role="menuitem" className={cn(MENU_BTN, 'justify-start text-[var(--color-danger)] hover:bg-[rgba(217,75,75,0.08)]')} onClick={logout}>
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
    <div className="flex items-center gap-2.5">
      <button
        className={TOOL_BTN}
        onClick={toggleTheme}
        title={theme === 'dark' ? t.shell.themeToLight : t.shell.themeToDark}
      >
        {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
      </button>
      <NotifBell />
      <LangSwitch />
      <UserMenu profile={profile} />
    </div>
  )
}

export function TopBar({ profile, groups, activeGroupId, onOpenDrawer }: TopBarProps) {
  return (
    <header className="flex h-14 flex-none items-center gap-4 border-b border-[var(--shell-topbar-border)] bg-[var(--shell-topbar-bg)] px-4 text-white shadow-[var(--shell-topbar-shadow)]">
      <button className="hidden cursor-pointer border-0 bg-none p-1 text-white max-[959px]:grid max-[959px]:place-items-center" onClick={onOpenDrawer} aria-label="menu">
        <MenuIcon />
      </button>
      <Link to="/dashboard" className="flex flex-none items-center gap-2.5 pr-2 font-brand text-[17px] font-bold whitespace-nowrap text-white no-underline">
        <img className="h-7 w-7 rounded-full bg-white p-px" src={logoMark} alt="Sphere Boss" />
        <span>Sphere Boss</span>
      </Link>
      <TopNav groups={groups} activeGroupId={activeGroupId} />
      <div className="ml-auto flex items-center gap-3">
        <SearchBox groups={groups} />
        <RightTools profile={profile} />
      </div>
    </header>
  )
}
