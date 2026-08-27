// 顶栏:品牌区 + 搜索/主题/通知/语言/用户工具区。分组导航只在侧栏呈现(2026-08-27 移除顶部分组主导航)。
// 样式:tailwind 原子类(原 shell.css 已删除)。
import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import type { Profile } from '../api/auth'
import { adminLogout } from '../api/auth'
import logoMark from '../assets/brand/logo-mark-navy.png'
import { useT, useLang, localeOptions } from '../i18n'
import { useTheme } from '../theme/context'
import { CheckIcon, GlobeIcon, LogoutIcon, MenuIcon, MoonIcon, SunIcon, UserIcon } from './icons'
import { NotifBell } from './NotifBell'
import { QuickSearch } from '../components/QuickSearch'
import { POPOVER, POPOVER_ITEM } from './popover'
import { cn } from '../lib/cn'

const TOOL_BTN = 'grid h-[34px] w-[34px] cursor-pointer place-items-center rounded-full border-0 bg-none text-white/80 hover:bg-[var(--shell-search-bg-focus)] hover:text-white'

interface TopBarProps {
  profile: Profile
  onOpenDrawer: () => void
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
        <div className={POPOVER} role="listbox">
          {localeOptions().map((opt) => (
            <button
              key={opt.value}
              role="option"
              aria-selected={opt.value === locale}
              className={cn(POPOVER_ITEM, opt.value === locale && 'font-semibold text-[var(--color-brand-gold-500)]')}
              onClick={() => {
                setLocale(opt.value as typeof locale)
                setOpen(false)
              }}
            >
              <span>{opt.label}</span>
              {opt.value === locale && <span className="ml-auto"><CheckIcon /></span>}
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
        <div className={POPOVER} role="menu">
          <div className="mb-1 border-b border-[var(--shell-popover-border)] px-3 pt-2 pb-2">
            <div className="text-[13px] font-semibold text-[var(--shell-heading)]">{profile.realName}</div>
            <div className="mt-0.5 text-xs text-[var(--shell-group-title)]">{profile.roleName}</div>
          </div>
          <button role="menuitem" className={POPOVER_ITEM} onClick={() => { setOpen(false); nav('/ucenter/overview') }}>
            <UserIcon />
            <span>{t.common.profile}</span>
          </button>
          <button role="menuitem" className={cn(POPOVER_ITEM, 'justify-start text-[var(--color-danger)] hover:bg-[rgba(217,75,75,0.08)]')} onClick={logout}>
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

export function TopBar({ profile, onOpenDrawer }: TopBarProps) {
  return (
    <header className="flex h-14 flex-none items-center gap-4 border-b border-[var(--shell-topbar-border)] bg-[var(--shell-topbar-bg)] px-4 text-white shadow-[var(--shell-topbar-shadow)]">
      <button className="hidden cursor-pointer border-0 bg-none p-1 text-white max-[959px]:grid max-[959px]:place-items-center" onClick={onOpenDrawer} aria-label="menu">
        <MenuIcon />
      </button>
      <Link to="/dashboard" className="flex flex-none items-center gap-2.5 pr-2 font-brand text-[17px] font-bold whitespace-nowrap text-white no-underline">
        <img className="h-7 w-7 rounded-full bg-white p-px" src={logoMark} alt="Sphere Boss" />
        <span>Sphere Boss</span>
      </Link>
      <div className="ml-auto flex items-center gap-3">
        <QuickSearch profile={profile} />
        <RightTools profile={profile} />
      </div>
    </header>
  )
}
