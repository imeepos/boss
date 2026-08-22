// 官网首页顶栏:品牌标识(左) + 居中锚点导航 + 语言/主题/CTA(右)。
// 深藏青品牌底色双主题恒色,文案走 i18n,颜色走 home.css 令牌。
import { Link } from 'react-router-dom'
import { Dropdown } from '../../components/Dropdown'
import logoFull from '../../assets/brand/logo-mark-gradient.png'
import { ThemeIcon } from './icons'
import { localeOptions } from '../../i18n'

export interface TopNavProps {
  locale: string
  setLocale: (v: string) => void
  theme: 'light' | 'dark'
  toggleTheme: () => void
  ctaLabel: string
  onCtaClick: () => void
  t: { heroBadge: string; navFeatures: string; navSolutions: string; navContact: string }
  shellT: { themeToDark: string; themeToLight: string }
}

const NAV_LINK = 'text-sm text-[var(--shell-nav-text)] transition-colors hover:text-white'
const CTA = 'inline-flex items-center justify-center rounded-md bg-[var(--color-brand-gold-500)] px-6 py-2 text-sm font-semibold text-[var(--color-brand-navy-950)] transition-colors hover:bg-[var(--color-brand-gold-600)]'

export function TopNav({ locale, setLocale, theme, toggleTheme, ctaLabel, onCtaClick, t, shellT }: TopNavProps) {
  return (
    <header className="sticky top-0 z-20 border-b border-white/10 backdrop-blur" style={{ backgroundColor: 'var(--home-nav-bg)' }}>
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-6 px-4">
        <Link to="/home" className="flex flex-none items-center gap-2">
          <img src={logoFull} alt="" className="h-8 w-8" />
          <div className="flex flex-col leading-tight">
            <span className="text-base font-bold text-white">Sphere Boss</span>
            <span className="text-[10px] text-[var(--shell-nav-text)]">{t.heroBadge}</span>
          </div>
        </Link>
        <nav className="hidden flex-1 items-center justify-center gap-8 md:flex">
          <a href="#features" className={NAV_LINK}>{t.navFeatures}</a>
          <a href="#solutions" className={NAV_LINK}>{t.navSolutions}</a>
          <a href="#contact" className={NAV_LINK}>{t.navContact}</a>
        </nav>
        <div className="ml-auto flex flex-none items-center gap-2 md:ml-0">
          <Dropdown
            value={locale}
            options={localeOptions()}
            onChange={(v) => setLocale(v)}
            ariaLabel="language"
            triggerStyle={{ height: 32 }}
            onDark
          />
          <button
            type="button"
            onClick={toggleTheme}
            aria-label={theme === 'dark' ? shellT.themeToLight : shellT.themeToDark}
            className="flex h-8 w-8 items-center justify-center rounded-md text-[var(--shell-nav-text)] transition-colors hover:bg-white/10 hover:text-white"
          >
            <ThemeIcon dark={theme === 'dark'} />
          </button>
          <button type="button" className={CTA} onClick={onCtaClick}>{ctaLabel}</button>
        </div>
      </div>
    </header>
  )
}
