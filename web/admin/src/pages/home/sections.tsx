// 首页子区块组件:顶栏 / Hero / 数据条 / 核心能力 / 客户成功 / CTA 横幅。
// 文案类型由 props 传入,便于主组件聚合 t.pages.home.* 文案对象。
import { Link } from 'react-router-dom'
import { Dropdown } from '../../components/Dropdown'
import { MaskIcon } from '../../layouts/icons'
import logoFull from '../../assets/brand/logo-mark-gradient.png'
import { HeroNetworkDecoration } from './HeroNetworkDecoration'
import { CrownIcon, ThemeIcon } from './icons'

// ----- 样式 token -----
export const NAV_LINK = 'text-sm text-[var(--shell-nav-text)] hover:text-white transition-colors'
const STAT_VALUE = 'text-[32px] font-bold leading-none text-[var(--color-brand-gold-500)]'
const STAT_LABEL = 'mt-2 text-xs text-[var(--shell-nav-text)]'
const CARD = 'rounded-lg border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-[var(--shell-card-shadow)] transition-shadow hover:shadow-md'
const CARD_TITLE = 'text-base font-semibold text-[var(--shell-heading)]'
const CARD_DESC = 'mt-2 text-xs leading-5 text-[var(--shell-content-text)]'
const ICON_BOX = 'flex h-10 w-10 flex-none items-center justify-center rounded-md bg-[#F4F7FB] text-[var(--color-brand-navy-900)]'
const ICON_BOX_CASE = 'flex h-10 w-10 flex-none items-center justify-center rounded-full bg-[#E7F5EE] text-[#2F8F63]'
const BTN_GOLD = 'rounded-md bg-[var(--color-brand-gold-500)] px-5 py-2.5 text-sm font-semibold text-[var(--color-brand-navy-950)] transition-colors hover:bg-[var(--color-brand-gold-600)]'
const BTN_GOLD_LG = 'rounded-md bg-[var(--color-brand-gold-500)] px-7 py-3 text-sm font-semibold text-[var(--color-brand-navy-950)] transition-colors hover:bg-[var(--color-brand-gold-600)]'
const BTN_GHOST = 'rounded-md border border-white/40 px-5 py-2.5 text-sm text-white transition-colors hover:border-white/70 hover:bg-white/5'

export function TopNav({
  locale, setLocale, theme, toggleTheme, ctaLabel, onCtaClick, t, shellT,
}: {
  locale: string
  setLocale: (v: string) => void
  theme: 'light' | 'dark'
  toggleTheme: () => void
  ctaLabel: string
  onCtaClick: () => void
  t: { heroBadge: string; navFeatures: string; navSolutions: string; navContact: string }
  shellT: { themeToDark: string; themeToLight: string }
}) {
  return (
    <header className="sticky top-0 z-20 border-b border-white/10 bg-[var(--color-brand-navy-950)]">
      <div className="mx-auto flex h-14 max-w-6xl items-center gap-6 px-4">
        <Link to="/home" className="flex items-center gap-2">
          <img src={logoFull} alt="" className="h-7 w-7" />
          <div className="flex flex-col leading-tight">
            <span className="text-base font-bold text-white">Sphere Boss</span>
            <span className="text-[10px] text-[var(--shell-nav-text)]">{t.heroBadge}</span>
          </div>
        </Link>
        <nav className="ml-4 hidden items-center gap-5 md:flex">
          <a href="#features" className={NAV_LINK}>{t.navFeatures}</a>
          <a href="#solutions" className={NAV_LINK}>{t.navSolutions}</a>
          <a href="#contact" className={NAV_LINK}>{t.navContact}</a>
        </nav>
        <div className="ml-auto flex items-center gap-2">
          <Dropdown
            value={locale}
            options={localeOptions()}
            onChange={(v) => setLocale(v)}
            ariaLabel="language"
            triggerStyle={{ height: 32 }}
          />
          <button
            type="button"
            onClick={toggleTheme}
            aria-label={theme === 'dark' ? shellT.themeToLight : shellT.themeToDark}
            className="flex h-8 w-8 items-center justify-center rounded-md text-[var(--shell-nav-text)] transition-colors hover:bg-white/10 hover:text-white"
          >
            <ThemeIcon dark={theme === 'dark'} />
          </button>
          <button type="button" className={BTN_GOLD} onClick={onCtaClick}>{ctaLabel}</button>
        </div>
      </div>
    </header>
  )
}

// localeOptions 来自 i18n,DropDown 需要;这里重新导出避免主组件依赖
import { localeOptions } from '../../i18n'

export function Hero({
  onCtaClick, t,
}: {
  onCtaClick: () => void
  t: { heroBadge: string; heroTitle: string; heroLine2: string; heroSubtitle: string; bookDemo: string; learnMore: string }
}) {
  return (
    <section className="relative overflow-hidden bg-gradient-to-br from-[#E8F0FA] via-[#F0F4FA] to-[#F4F5F7]">
      <div className="mx-auto grid max-w-6xl items-center gap-10 px-4 py-16 md:grid-cols-[1.2fr_1fr] md:py-24">
        <div className="relative z-10">
          <span className="inline-block rounded-full border border-[var(--color-brand-gold-500)]/40 bg-white/40 px-3 py-1 text-xs text-[var(--color-brand-gold-600)]">
            {t.heroBadge}
          </span>
          <h1 className="mt-5 text-[36px] font-bold leading-[1.2] text-[var(--color-brand-navy-950)] md:text-[44px]">
            <span className="block whitespace-nowrap">{t.heroTitle}</span>
            <span className="mt-2 block whitespace-nowrap text-[var(--color-brand-navy-900)]">{t.heroLine2}</span>
          </h1>
          <p className="mt-5 max-w-md text-sm leading-6 text-[#4E5664]">{t.heroSubtitle}</p>
          <div className="mt-8 flex flex-wrap gap-3">
            <button type="button" className={BTN_GOLD} onClick={onCtaClick}>{t.bookDemo}</button>
            <a href="#features" className={`${BTN_GHOST} inline-flex items-center bg-[var(--color-brand-navy-950)]`}>
              {t.learnMore}
            </a>
          </div>
        </div>
        <div className="relative hidden min-h-[320px] md:block">
          <HeroNetworkDecoration />
        </div>
      </div>
    </section>
  )
}

export function Stats({ stats }: { stats: Array<{ value: string; label: string }> }) {
  return (
    <section className="border-y border-white/5 bg-gradient-to-r from-[var(--color-brand-navy-950)] to-[#1F355F]">
      <div className="mx-auto grid max-w-6xl grid-cols-1 gap-6 px-4 py-8 sm:grid-cols-3">
        {stats.map((s, i) => (
          <div
            key={s.label}
            className={'flex items-center justify-center sm:justify-start' + (i > 0 ? ' sm:border-l sm:border-white/10 sm:pl-8' : '')}
          >
            <div className="text-center sm:text-left">
              <div className="flex items-baseline justify-center gap-1 sm:justify-start">
                <span className={STAT_VALUE}>{s.value}</span>
                <span className="text-base text-[var(--color-brand-gold-500)]" aria-hidden>↑</span>
              </div>
              <div className={STAT_LABEL}>{s.label}</div>
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

export function Features({
  features, title, subtitle,
}: {
  features: Array<{ icon: string; title: string; desc: string }>
  title: string
  subtitle: string
}) {
  return (
    <section id="features" className="mx-auto max-w-6xl scroll-mt-20 px-4 py-20">
      <h2 className="text-center text-[28px] font-bold text-[var(--shell-heading)]">{title}</h2>
      <p className="mt-2 text-center text-sm text-[var(--shell-content-text)]">{subtitle}</p>
      <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {features.map((f) => (
          <div key={f.icon} className={CARD}>
            <div className="flex items-start gap-4">
              <span className={ICON_BOX}>
                <MaskIcon url={`/icons/${f.icon}.svg`} size={20} />
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <h3 className={CARD_TITLE}>{f.title}</h3>
                  <span className="text-[var(--color-brand-gold-500)]" aria-hidden>→</span>
                </div>
                <p className={CARD_DESC}>{f.desc}</p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

export function Cases({
  cases, title, subtitle, viewDetail,
}: {
  cases: Array<{ icon: string; title: string; desc: string; metric: string }>
  title: string
  subtitle: string
  viewDetail: string
}) {
  return (
    <section id="solutions" className="bg-gradient-to-b from-transparent to-[#F8FAFC]">
      <div className="mx-auto max-w-6xl scroll-mt-20 px-4 pb-20">
        <h2 className="text-center text-[28px] font-bold text-[var(--shell-heading)]">{title}</h2>
        <p className="mt-2 text-center text-sm text-[var(--shell-content-text)]">{subtitle}</p>
        <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {cases.map((c) => (
            <div key={c.icon} className={CARD}>
              <div className="flex items-start gap-4">
                <span className={ICON_BOX_CASE}>
                  <MaskIcon url={`/icons/${c.icon}.svg`} size={20} />
                </span>
                <div className="min-w-0 flex-1">
                  <h3 className={CARD_TITLE}>{c.title}</h3>
                  <p className={CARD_DESC}>
                    {c.desc}
                    <span className="ml-1 font-semibold text-[#2F8F63]">{c.metric}</span>
                  </p>
                  <a
                    href="#contact"
                    className="mt-4 inline-flex items-center gap-1 text-xs font-medium text-[var(--color-brand-gold-600)] hover:underline"
                  >
                    {viewDetail}
                    <span aria-hidden>→</span>
                  </a>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

export function CtaBanner({
  title, subtitle, buttonLabel, onClick,
}: {
  title: string
  subtitle: string
  buttonLabel: string
  onClick: () => void
}) {
  return (
    <section id="contact" className="mx-auto max-w-6xl scroll-mt-20 px-4 pb-20">
      <div className="flex flex-col items-center gap-6 rounded-lg bg-[var(--color-brand-navy-950)] px-6 py-10 md:flex-row md:justify-between md:gap-8 md:px-10">
        <div className="flex items-center gap-5">
          <span className="flex-none"><CrownIcon size={56} /></span>
          <div>
            <h2 className="text-xl font-bold text-white">{title}</h2>
            <p className="mt-1 text-sm text-[var(--shell-nav-text)]">{subtitle}</p>
          </div>
        </div>
        <button type="button" className={BTN_GOLD_LG} onClick={onClick}>{buttonLabel}</button>
      </div>
    </section>
  )
}