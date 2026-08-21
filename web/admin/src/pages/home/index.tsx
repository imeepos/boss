// 官网首页(公开落地页):根路径未登录时入口;导航 + hero + 数据 + 核心能力 + 页脚。
// 文案走 i18n,颜色走 brand/shell 令牌,图标复用 public/icons 组图标(MaskIcon)。
import { Link, useNavigate } from 'react-router-dom'
import { useT, useLang, localeOptions } from '../../i18n'
import { useTheme } from '../../theme/context'
import { getAuthToken } from '../../api/client'
import { Dropdown } from '../../components/Dropdown'
import { AdCarousel } from '../auth-ads'
import { GlobeIcon, MaskIcon, MoonIcon, SunIcon } from '../../layouts/icons'
import logoFull from '../../assets/brand/logo-mark-gradient.png'

const NAV_LINK = 'text-sm text-[var(--shell-nav-text)] hover:text-white transition-colors'
const HERO_TITLE = 'text-4xl md:text-5xl font-bold leading-tight text-white'
const STAT_VALUE = 'text-3xl font-bold text-[var(--color-brand-gold-500)]'
const STAT_LABEL = 'mt-1 text-xs text-[var(--shell-nav-text)]'
const CARD = 'rounded-lg border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-[var(--shell-card-shadow)]'
const CARD_TITLE = 'text-sm font-semibold text-[var(--shell-heading)]'
const CARD_DESC = 'mt-2 text-xs leading-5 text-[var(--shell-content-text)]'
const BTN_GOLD = 'rounded-md bg-[var(--color-brand-gold-500)] px-5 py-2.5 text-sm font-semibold text-[var(--color-brand-navy-950)] transition-colors hover:bg-[var(--color-brand-gold-600)]'
const BTN_GHOST = 'rounded-md border border-white/30 px-5 py-2.5 text-sm text-white transition-colors hover:border-white/60'

export default function HomePage() {
  const t = useT()
  const nav = useNavigate()
  const { locale, setLocale } = useLang()
  const { theme, toggleTheme } = useTheme()
  const signedIn = !!getAuthToken()
  const ctaTarget = signedIn ? '/dashboard' : '/login'
  const ctaLabel = signedIn ? t.pages.home.enterConsole : t.pages.home.login

  return (
    <div className="min-h-screen bg-[var(--shell-content-bg)]">
      {/* 顶栏:品牌 + 锚点导航 + 语言/主题切换 + 登录 CTA */}
      <header className="sticky top-0 z-20 border-b border-white/10 bg-[var(--shell-topbar-bg)]">
        <div className="mx-auto flex h-14 max-w-6xl items-center gap-6 px-4">
          <Link to="/home" className="flex items-center gap-2">
            <img src={logoFull} alt="" className="h-7 w-7" />
            <span className="text-base font-bold text-white">Sphere Boss</span>
          </Link>
          <nav className="ml-4 hidden items-center gap-5 md:flex">
            <a href="#features" className={NAV_LINK}>{t.pages.home.navFeatures}</a>
            <a href="#contact" className={NAV_LINK}>{t.pages.home.navContact}</a>
          </nav>
          <div className="ml-auto flex items-center gap-3">
            <Dropdown
              value={locale}
              options={localeOptions()}
              onChange={(v) => setLocale(v as typeof locale)}
              ariaLabel="language"
              triggerStyle={{ height: 32 }}
            />
            <button
              type="button"
              onClick={toggleTheme}
              aria-label={theme === 'dark' ? t.shell.themeToLight : t.shell.themeToDark}
              className="flex h-8 w-8 items-center justify-center rounded-md text-[var(--shell-nav-text)] transition-colors hover:bg-white/10 hover:text-white"
            >
              {theme === 'dark' ? <SunIcon size={16} /> : <MoonIcon size={16} />}
            </button>
            <button type="button" className={BTN_GOLD} onClick={() => nav(ctaTarget)}>
              {ctaLabel}
            </button>
          </div>
        </div>
      </header>

      {/* hero:藏青底 + 金色主 CTA + 广告轮播 */}
      <section className="bg-[var(--color-brand-navy-950)]">
        <div className="mx-auto grid max-w-6xl items-center gap-10 px-4 py-16 md:grid-cols-2 md:py-24">
          <div>
            <span className="inline-block rounded-full border border-[var(--color-brand-gold-500)]/40 px-3 py-1 text-xs text-[var(--color-brand-gold-300)]">
              {t.pages.home.heroBadge}
            </span>
            <h1 className={`mt-5 ${HERO_TITLE}`}>{t.pages.home.heroTitle}</h1>
            <p className="mt-4 max-w-md text-sm leading-6 text-[var(--shell-nav-text)]">
              {t.pages.home.heroSubtitle}
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <button type="button" className={BTN_GOLD} onClick={() => nav(ctaTarget)}>
                {t.pages.home.heroCta}
              </button>
              <a href="#features" className={`${BTN_GHOST} inline-flex items-center`}>
                {t.pages.home.heroCtaAlt}
              </a>
            </div>
          </div>
          <div className="hidden justify-center md:flex">
            <AdCarousel width={360} />
          </div>
        </div>
        {/* 数据条 */}
        <div className="border-t border-white/10">
          <div className="mx-auto grid max-w-6xl grid-cols-3 gap-4 px-4 py-8">
            {t.pages.home.stats.map((s) => (
              <div key={s.label} className="text-center">
                <div className={STAT_VALUE}>{s.value}</div>
                <div className={STAT_LABEL}>{s.label}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* 核心能力:6 卡片,图标复用菜单组图标 */}
      <section id="features" className="mx-auto max-w-6xl px-4 py-16">
        <h2 className="text-center text-2xl font-bold text-[var(--shell-heading)]">
          {t.pages.home.featuresTitle}
        </h2>
        <p className="mt-2 text-center text-sm text-[var(--shell-content-text)]">
          {t.pages.home.featuresSubtitle}
        </p>
        <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {t.pages.home.features.map((f) => (
            <div key={f.title} className={CARD}>
              <div className="flex items-center gap-3">
                <span className="flex h-9 w-9 items-center justify-center rounded-md bg-[var(--shell-menu-hover-bg)] text-[var(--shell-menu-active-text)]">
                  <MaskIcon url={`/icons/${f.icon}.svg`} size={20} />
                </span>
                <span className={CARD_TITLE}>{f.title}</span>
              </div>
              <p className={CARD_DESC}>{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* 联系 CTA */}
      <section id="contact" className="mx-auto max-w-6xl px-4 pb-16">
        <div className="rounded-lg bg-[var(--color-brand-navy-950)] px-6 py-12 text-center">
          <h2 className="text-xl font-bold text-white">{t.pages.home.contactTitle}</h2>
          <p className="mt-2 text-sm text-[var(--shell-nav-text)]">{t.pages.home.contactDesc}</p>
          <button type="button" className={`${BTN_GOLD} mt-6`} onClick={() => nav(ctaTarget)}>
            {ctaLabel}
          </button>
        </div>
      </section>

      <footer className="border-t border-[var(--shell-side-border)] bg-[var(--shell-side-bg)]">
        <div className="mx-auto flex h-14 max-w-6xl items-center gap-2 px-4 text-xs text-[var(--shell-crumb-text)]">
          <GlobeIcon size={14} />
          {t.pages.home.footer}
        </div>
      </footer>
    </div>
  )
}
