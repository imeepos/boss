// 首页子区块组件:顶栏 / Hero / 数据条 / 核心能力 / 客户成功 / CTA 横幅。
// 文案类型由 props 传入,便于主组件聚合 t.pages.home.* 文案对象。
// 颜色一律走 tokens.css / home.css 令牌,禁止裸色值。
import { Link } from 'react-router-dom'
import { Dropdown } from '../../components/Dropdown'
import { MaskIcon } from '../../layouts/icons'
import logoFull from '../../assets/brand/logo-mark-gradient.png'
import { HeroConsoleMock } from './HeroConsoleMock'
import { CrownIcon, ThemeIcon } from './icons'
import { localeOptions } from '../../i18n'

// ----- 样式 token(字体层级对齐 MiniMax 官网:大标题/紧凑字距/大留白) -----
export const NAV_LINK = 'text-sm text-[var(--shell-nav-text)] transition-colors hover:text-white'
const STAT_VALUE = 'font-brand text-5xl font-bold leading-none tracking-tight text-[var(--color-brand-gold-500)]'
const STAT_LABEL = 'mt-2 text-sm text-[var(--shell-nav-text)]'
const CARD = 'rounded-xl border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-8 shadow-[var(--shell-card-shadow)] transition-all hover:-translate-y-1 hover:border-[var(--home-hero-badge-border)] hover:shadow-lg'
const CARD_TITLE = 'font-brand text-lg font-semibold tracking-tight text-[var(--shell-heading)]'
const CARD_DESC = 'mt-3 text-sm leading-6 text-[var(--shell-content-text)]'
const ICON_BOX = 'flex h-12 w-12 flex-none items-center justify-center rounded-lg bg-[var(--home-icon-bg)] text-[var(--home-icon-fg)]'
const ICON_BOX_CASE = 'flex h-12 w-12 flex-none items-center justify-center rounded-full bg-[var(--home-case-bg)] text-[var(--home-case-fg)]'
const BTN_GOLD = 'rounded-md bg-[var(--color-brand-gold-500)] px-6 py-3 text-sm font-semibold text-[var(--color-brand-navy-950)] transition-colors hover:bg-[var(--color-brand-gold-600)]'
const BTN_GOLD_LG = 'rounded-md bg-[var(--color-brand-gold-500)] px-8 py-3.5 text-base font-semibold text-[var(--color-brand-navy-950)] transition-colors hover:bg-[var(--color-brand-gold-600)]'
const BTN_GHOST = 'inline-flex items-center rounded-md border border-[var(--home-mock-border)] bg-[var(--color-brand-navy-950)] px-6 py-3 text-sm text-white transition-colors hover:border-white/40 hover:bg-white/5'
const SECTION_TITLE = 'font-brand text-center text-4xl font-bold tracking-tight text-[var(--shell-heading)]'
const SECTION_SUB = 'mt-3 text-center text-base text-[var(--shell-content-text)]'

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
    <header className="sticky top-0 z-20 border-b border-white/10 bg-[var(--color-brand-navy-950)]/95 backdrop-blur">
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
          <button type="button" className={BTN_GOLD} onClick={onCtaClick}>{ctaLabel}</button>
        </div>
      </div>
    </header>
  )
}

export function Hero({
  onCtaClick, t,
}: {
  onCtaClick: () => void
  t: { heroBadge: string; heroTitle: string; heroLine2: string; heroSubtitle: string; bookDemo: string; learnMore: string }
}) {
  return (
    <section className="relative overflow-hidden bg-gradient-to-br from-[var(--home-hero-from)] via-[var(--home-hero-via)] to-[var(--home-hero-to)]">
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-[var(--home-hero-badge-border)] to-transparent" />
      <div className="mx-auto grid max-w-6xl items-center gap-12 px-4 py-16 md:grid-cols-[1.15fr_1fr] md:py-28">
        <div className="relative z-10">
          <span className="inline-flex items-center gap-1.5 rounded-full border border-[var(--home-hero-badge-border)] bg-[var(--home-hero-badge-bg)] px-3.5 py-1.5 text-xs font-medium tracking-wide text-[var(--color-brand-gold-600)] dark:text-[var(--color-brand-gold-500)]">
            <span className="h-1.5 w-1.5 rounded-full bg-[var(--color-brand-gold-500)]" />
            {t.heroBadge}
          </span>
          <h1 className="font-brand mt-6 text-[38px] font-bold leading-[1.15] tracking-tight text-[var(--color-brand-navy-950)] dark:text-[var(--shell-heading)] md:text-[56px]">
            <span className="block">{t.heroTitle}</span>
            <span className="mt-3 block bg-gradient-to-r from-[var(--color-brand-gold-500)] to-[var(--color-brand-gold-600)] bg-clip-text text-transparent">{t.heroLine2}</span>
          </h1>
          <p className="mt-6 max-w-lg text-base leading-7 text-[var(--home-hero-subtitle)]">{t.heroSubtitle}</p>
          <div className="mt-10 flex flex-wrap gap-3">
            <button type="button" className={BTN_GOLD} onClick={onCtaClick}>{t.bookDemo}</button>
            <a href="#features" className={BTN_GHOST}>{t.learnMore}</a>
          </div>
        </div>
        <div className="relative hidden min-h-[340px] md:block">
          <HeroConsoleMock />
        </div>
      </div>
    </section>
  )
}

/** 关于区块:公司简介(左) + 关键数据(右),MiniMax 官网式"简介+统计"合并布局。 */
export function About({
  title, desc, stats,
}: {
  title: string
  desc: string
  stats: Array<{ value: string; label: string }>
}) {
  return (
    <section className="relative overflow-hidden border-y border-white/5 bg-[var(--color-brand-navy-950)]">
      <div className="pointer-events-none absolute left-1/2 top-0 h-40 w-[36rem] -translate-x-1/2 rounded-full bg-[var(--home-stats-glow)] blur-3xl" />
      <div className="relative mx-auto grid max-w-6xl gap-10 px-4 py-20 lg:grid-cols-[1.3fr_1fr] lg:items-center">
        <div>
          <h2 className="font-brand text-3xl font-bold tracking-tight text-white">{title}</h2>
          <p className="mt-5 max-w-xl text-base leading-8 text-[var(--shell-nav-text)]">{desc}</p>
        </div>
        <div className="grid grid-cols-1 gap-8 sm:grid-cols-3 lg:grid-cols-1 lg:gap-6">
          {stats.map((s) => (
            <div key={s.label} className="flex items-baseline gap-3 lg:flex-none">
              <span className={STAT_VALUE}>{s.value}</span>
              <span className={STAT_LABEL + ' mt-0'}>{s.label}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

export function Features({
  features, title, subtitle, viewDetail,
}: {
  features: Array<{ icon: string; title: string; tag: string; desc: string }>
  title: string
  subtitle: string
  viewDetail: string
}) {
  return (
    <section id="features" className="mx-auto max-w-6xl scroll-mt-20 px-4 py-24">
      <h2 className={SECTION_TITLE}>{title}</h2>
      <p className={SECTION_SUB}>{subtitle}</p>
      <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {features.map((f) => (
          <div key={f.icon} className={CARD}>
            <div className="flex items-start gap-4">
              <span className={ICON_BOX}>
                <MaskIcon url={`/icons/${f.icon}.svg`} size={20} />
              </span>
              <div className="min-w-0 flex-1">
                <h3 className={CARD_TITLE}>{f.title}</h3>
                <span className="mt-1.5 inline-block rounded bg-[var(--home-icon-bg)] px-1.5 py-0.5 text-[10px] font-medium text-[var(--home-icon-fg)]">
                  {f.tag}
                </span>
                <p className={CARD_DESC}>{f.desc}</p>
              </div>
            </div>
            <a href="#contact" className="mt-4 inline-flex items-center gap-1 text-xs font-medium text-[var(--color-brand-gold-600)] hover:underline dark:text-[var(--color-brand-gold-500)]">
              {viewDetail}
            </a>
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
    <section id="solutions" className="bg-gradient-to-b from-transparent to-[var(--home-section-alt)]">
      <div className="mx-auto max-w-6xl scroll-mt-20 px-4 pb-24">
        <h2 className={SECTION_TITLE}>{title}</h2>
        <p className={SECTION_SUB}>{subtitle}</p>
        <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {cases.map((c) => (
            <div key={c.icon} className={CARD}>
              <div className="flex items-start gap-4">
                <span className={ICON_BOX_CASE}>
                  <MaskIcon url={`/icons/${c.icon}.svg`} size={20} />
                </span>
                <div className="min-w-0 flex-1">
                  <h3 className={CARD_TITLE}>{c.title}</h3>
                  <p className={CARD_DESC}>{c.desc}</p>
                  <div className="mt-3 flex items-center justify-between gap-2">
                    <span className="inline-flex items-baseline rounded-full bg-[var(--home-case-bg)] px-2.5 py-1 text-sm font-bold text-[var(--home-case-fg)]">
                      {c.metric}
                    </span>
                    <a
                      href="#contact"
                      className="inline-flex items-center gap-1 text-xs font-medium text-[var(--color-brand-gold-600)] hover:underline dark:text-[var(--color-brand-gold-500)]"
                    >
                      {viewDetail}
                    </a>
                  </div>
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
      <div className="relative overflow-hidden rounded-xl border border-white/10 bg-[var(--color-brand-navy-950)] px-6 py-12 md:px-12">
        <div className="pointer-events-none absolute -right-10 -top-16 h-56 w-56 rounded-full bg-[var(--home-cta-glow)] blur-3xl" />
        <div className="relative flex flex-col items-center gap-6 md:flex-row md:justify-between md:gap-8">
          <div className="flex items-center gap-5">
            <span className="flex-none"><CrownIcon size={56} /></span>
            <div>
              <h2 className="font-brand text-2xl font-bold tracking-tight text-white">{title}</h2>
              <p className="mt-2 text-sm text-[var(--shell-nav-text)]">{subtitle}</p>
            </div>
          </div>
          <button type="button" className={BTN_GOLD_LG} onClick={onClick}>{buttonLabel}</button>
        </div>
      </div>
    </section>
  )
}
